package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/getsentry/vroom/internal/aggregate"
	"github.com/getsentry/vroom/internal/android"
	"github.com/getsentry/vroom/internal/calltree"
	"github.com/google/uuid"
)

type Profile struct {
	OrganizationID       uint64    `ch:"organization_id"`
	ProjectID            uint64    `ch:"project_id"`
	TransactionID        uuid.UUID `ch:"transaction_id"`
	ProfileID            uuid.UUID `ch:"profile_id"`
	Received             time.Time `ch:"received"`
	Profile              string    `ch:"profile"`
	AndroidApiLevel      uint32    `ch:"android_api_level"`
	DeviceClassification string    `ch:"device_classification"`
	DeviceLocale         string    `ch:"device_locale"`
	DeviceManufacturer   string    `ch:"device_manufacturer"`
	DeviceModel          string    `ch:"device_model"`
	DeviceOSBuildNumber  string    `ch:"device_os_build_number"`
	DeviceOSName         string    `ch:"device_os_name"`
	DeviceOSVersion      string    `ch:"device_os_version"`
	DurationNS           uint64    `ch:"duration_ns"`
	Environment          string    `ch:"environment"`
	Platform             string    `ch:"platform"`
	TraceID              uuid.UUID `ch:"trace_id"`
	TransactionName      string    `ch:"transaction_name"`
	VersionName          string    `ch:"version_name"`
	VersionCode          string    `ch:"version_code"`
	RetentionDays        uint16    `ch:"retention_days"`
	Partition            uint16    `ch:"partition"`
	Offset               uint64    `ch:"offset"`
}

type CallTree struct {
	File              string     `json:"file"`
	Image             string     `json:"image"`
	IsApplication     bool       `json:"is_application"`
	Line              uint32     `json:"line"`
	Name              string     `json:"name"`
	StartTimestamp    uint64     `json:"start_timestamp"`
	StopTimestamp     uint64     `json:"stop_timestamp"`
	Children          []CallTree `json:"children"`
	Duration          uint64     `json:"duration"`
	SelfTime          uint64     `json:"self_time"`
	Fingerprint       uint64     `json:"fingerprint"`
	ParentFingerprint uint64     `json:"parent_fingerprint"`
}

func (t *CallTree) PrettyPrint(depth int) {
	for i := 0; i < depth; i++ {
		fmt.Printf("~")
	}

	fmt.Printf("%v %v %v %v %v\n", t.StartTimestamp, t.StopTimestamp, t.File, t.Image, t.Name)
	for _, child := range t.Children {
		child.PrettyPrint(depth + 1)
	}
}

func (t *CallTree) Finalize(parents []*CallTree) {
	t.Duration = t.StopTimestamp - t.StartTimestamp
	t.SelfTime = t.Duration

	self := append(parents, t)

	hasher := md5.New()
	for _, callTree := range self {
		hasher.Write([]byte(" "))
		// hasher.Write([]byte(callTree.File))
		hasher.Write([]byte(callTree.Image))
		hasher.Write([]byte(callTree.Name))
	}
	fingerprint, _ := strconv.ParseUint(hex.EncodeToString(hasher.Sum(nil))[:16], 16, 64)

	t.Fingerprint = fingerprint

	for i := 0; i < len(t.Children); i++ {
		t.Children[i].Finalize(self)
		t.SelfTime -= t.Children[i].Duration
		t.ParentFingerprint = fingerprint
	}
}

func walkCallTree(callTree CallTree, fn func(CallTree) error) error {
	if err := fn(callTree); err != nil {
		return err
	}
	for _, child := range callTree.Children {
		if err := walkCallTree(child, fn); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	addr, limit, err := parseArgs(os.Args)

	if err != nil {
		log.Fatal(err)
	}

	if err = run(addr, limit); err != nil {
		log.Fatal(err)
	}
}

func parseArgs(args []string) (string, int, error) {
	addr := args[1]

	limit, err := strconv.Atoi(args[2])
	if err != nil {
		return "", 0, fmt.Errorf("Unknown limit: %s", os.Args[2])
	}

	return addr, limit, nil
}

func run(addr string, limit int) error {
	ctx, conn, err := connect(addr)

	if err != nil {
		return err
	}

	err = setupFunctionsTable(ctx, conn)
	if err != nil {
		return err
	}

	count := 0
	maxTs := time.Now()
	minTs := maxTs.Add(-31 * 24 * time.Hour)
	step := 1 * time.Hour

	for ts := minTs; ts.Before(maxTs); ts = ts.Add(step) {
		startTs := ts
		stopTs := ts.Add(step)

		lastProfileID := uuid.Nil
		for {
			var rows []Profile

			now := time.Now()

			if lastProfileID == uuid.Nil {
				err = conn.Select(ctx, &rows, `
					SELECT *
					FROM profiles_local
					WHERE
						platform IN ('cocoa', 'android')
						AND received >= $1
						AND received < $2
					ORDER BY received ASC, profile_id ASC
					LIMIT $3
				`, startTs, stopTs, limit)
			} else {
				err = conn.Select(ctx, &rows, `
					SELECT *
					FROM profiles_local
					WHERE 
						platform IN ('cocoa', 'android')
						AND received >= $1
						AND received < $2
						AND (
							received > $1
							OR (received = $1 AND profile_id > $3)
						)
					ORDER BY received ASC, profile_id ASC
					LIMIT $4
				`, startTs, stopTs, lastProfileID, limit)
			}

			elapsed := time.Now().Sub(now)

			if err != nil {
				return err
			}

			count += len(rows)

			log.Printf("rows in result: %d/%d (%d)\n", len(rows), count, elapsed)

			if len(rows) == 0 {
				break
			}

			batch, err := conn.PrepareBatch(ctx, "INSERT INTO functions_local")
			if err != nil {
				return err
			}

			now = time.Now()

			lastProfile := rows[len(rows)-1]
			lastProfileID = lastProfile.ProfileID
			startTs = lastProfile.Received

			for _, row := range rows {
				var callTreesByThreadId map[uint64][]CallTree
				if row.Platform == "cocoa" {
					callTreesByThreadId, err = extractIosCallTreesFromRow(row)
					if err != nil {
						log.Printf("failed to extra ios profile: %v %v\n", row.ProfileID, err)
						continue
					}
				} else if row.Platform == "android" {
					callTreesByThreadId, err = extractAndroidCallTreesFromRow(row)
					if err != nil {
						log.Printf("failed to extra android profile: %v %v\n", row.ProfileID, err)
						continue
					}
				} else {
					// this should be impossible
				}

				var functionID uint64 = 0
				for _, callTrees := range callTreesByThreadId {
					for _, callTree := range callTrees {
						callTree.Finalize([]*CallTree{})

						err = walkCallTree(callTree, func(t CallTree) error {
							err = batch.Append(
								row.OrganizationID,
								row.ProjectID,
								row.ProfileID,
								row.TransactionID,
								row.TraceID,
								row.Received,
								row.AndroidApiLevel,
								row.DeviceClassification,
								row.DeviceLocale,
								row.DeviceManufacturer,
								row.DeviceModel,
								row.DeviceOSBuildNumber,
								row.DeviceOSName,
								row.DeviceOSVersion,
								row.Environment,
								row.Platform,
								row.TransactionName,
								row.VersionName,
								row.VersionCode,
								functionID,
								t.Name,
								t.File,
								t.Line,
								t.SelfTime,
								t.Duration,
								t.StartTimestamp,
								t.Fingerprint,
								t.ParentFingerprint,
								row.RetentionDays,
								row.Partition,
								row.Offset,
								uint8(0),
							)

							functionID++

							if err != nil {
								return err
							}

							return nil
						})

						if err != nil {
							return err
						}
					}
				}
			}

			if err = batch.Send(); err != nil {
				log.Printf("%v\n", err)
				continue
			}

			elapsed = time.Now().Sub(now)
			log.Printf("inserted (%v)\n", elapsed)
		}
	}

	log.Printf("count: %d\n", count)
	return nil
}

func extractIosCallTreesFromRow(row Profile) (map[uint64][]CallTree, error) {
	callTreesByThreadId := make(map[uint64][]CallTree)

	var iosProfile aggregate.IosProfile
	err := json.Unmarshal([]byte(row.Profile), &iosProfile)
	if err != nil {
		return callTreesByThreadId, err
	}

	for _, sample := range iosProfile.Samples {
		threadId, err := parseUInt64(sample.ThreadID)
		if err != nil {
			return callTreesByThreadId, err
		}

		relativeTimestampNS, err := parseUInt64(sample.RelativeTimestampNS)
		if err != nil {
			return callTreesByThreadId, err
		}

		isMainThread, mainFunctionAddress := checkMainThread(sample.Frames)

		frames := make([]aggregate.IosFrame, 0, len(sample.Frames))
		for _, frame := range sample.Frames {
			frames = append(frames, frame)
			// stop now that we reached the main function
			if isMainThread && mainFunctionAddress != "" && frame.InstructionAddr == mainFunctionAddress {
				break
			}
		}

		callTree := convertFramesToCallTree(frames, relativeTimestampNS)
		callTrees, ok := callTreesByThreadId[threadId]
		if !ok {
			callTrees = make([]CallTree, 0, 1)
		}

		if len(callTrees) == 0 {
			callTrees = append(callTrees, callTree)
		} else {
			prev := callTrees[len(callTrees)-1]
			left, right, remaining := mergeCallTrees(prev, callTree)
			callTrees[len(callTrees)-1] = left
			if remaining {
				callTrees = append(callTrees, right)
			}
		}

		callTreesByThreadId[threadId] = callTrees
	}

	return callTreesByThreadId, nil
}

func mergeCallTrees(left CallTree, right CallTree) (CallTree, CallTree, bool) {
	updateCallTreeStopTimestamp(&left, right.StopTimestamp)

	if !canMergeCallTrees(left, right) {
		return left, right, true
	}

	if !isFlatCallTree(right) {
		return left, right, true
	}

	if len(left.Children) == 0 {
		left.Children = right.Children
	} else if len(right.Children) > 0 {
		leftChild := left.Children[len(left.Children)-1]
		rightChild := right.Children[0]
		leftChild, rightChild, remaining := mergeCallTrees(leftChild, rightChild)
		left.Children[len(left.Children)-1] = leftChild
		if remaining {
			left.Children = append(left.Children, rightChild)
		}
	}

	return left, CallTree{}, false
}

func updateCallTreeStopTimestamp(tree *CallTree, timestamp uint64) {
	tree.StopTimestamp = timestamp
	// also make sure to update the last child's timestamp, here we
	// assume that the children are in chronological order
	if len(tree.Children) > 0 {
		updateCallTreeStopTimestamp(&tree.Children[len(tree.Children)-1], timestamp)
	}
}

func canMergeCallTrees(left, right CallTree) bool {
	if left.File != right.File {
		return false
	}
	if left.Image != right.Image {
		return false
	}
	if left.Name != right.Name {
		return false
	}
	if left.StopTimestamp > right.StartTimestamp {
		return false
	}
	return true
}

func isFlatCallTree(tree CallTree) bool {
	if len(tree.Children) == 0 {
		return true
	} else if len(tree.Children) == 1 {
		return isFlatCallTree(tree.Children[0])
	} else {
		return false
	}
}

func convertFramesToCallTree(frames []aggregate.IosFrame, timestamp uint64) CallTree {
	var tree CallTree
	for i, frame := range frames {
		// address, _ := strconv.ParseUint(frame.InstructionAddr, 16, 64)
		children := make([]CallTree, 0)
		if i > 0 {
			children = append(children, tree)
		}
		symbolName := frame.Function
		if symbolName == "" {
			symbolName = fmt.Sprintf("unknown (%s)", frame.InstructionAddr)
		}
		tree = CallTree{
			File:           frame.Filename,
			Image:          calltree.ImageBaseName(frame.Package),
			IsApplication:  aggregate.IsIOSApplicationImage(frame.Package),
			Line:           uint32(frame.LineNo),
			Name:           symbolName,
			StartTimestamp: timestamp,
			StopTimestamp:  timestamp,
			Children:       children,
		}
	}
	return tree
}

func checkMainThread(frames []aggregate.IosFrame) (bool, string) {
	isMainThread := false
	mainFunctionAddress := ""

	for _, frame := range frames {
		if isMainThread {
			mainFunctionAddress = frame.InstructionAddr
			break
		}
		if frame.Function == "main" || frame.Function == "UIApplicationMain" {
			isMainThread = true
		}
	}

	return isMainThread, mainFunctionAddress
}

func parseUInt64(value interface{}) (uint64, error) {
	switch v := value.(type) {
	case string:
		return strconv.ParseUint(v, 10, 64)
	case float64:
		return uint64(v), nil
	case uint64:
		return v, nil
	}
	return 0, errors.New("Unknown type for value")
}

func extractAndroidCallTreesFromRow(row Profile) (map[uint64][]CallTree, error) {
	callTreesByThreadId := make(map[uint64][]CallTree)

	var androidProfile android.AndroidProfile
	err := json.Unmarshal([]byte(row.Profile), &androidProfile)
	if err != nil {
		return callTreesByThreadId, err
	}

	methodIdToFrames := make(map[uint64][]Frame)

	for _, method := range androidProfile.Methods {
		if len(method.InlineFrames) > 0 {
			for _, m := range method.InlineFrames {
				methodIdToFrames[method.ID] = append(methodIdToFrames[method.ID], Frame{
					Name:          m.Name,
					File:          m.SourceFile,
					Line:          uint32(method.SourceLine),
					IsApplication: !aggregate.IsAndroidSystemPackage(m.ClassName),
					Image:         m.ClassName,
				})
			}
		} else {
			packageName, _, err := android.ExtractPackageNameAndSimpleMethodNameFromAndroidMethod(&method)
			if err != nil {
				return callTreesByThreadId, err
			}
			fullMethodName, err := android.FullMethodNameFromAndroidMethod(&method)
			if err != nil {
				return callTreesByThreadId, err
			}
			methodIdToFrames[method.ID] = append(methodIdToFrames[method.ID], Frame{
				Name:          fullMethodName,
				File:          method.SourceFile,
				Line:          uint32(method.SourceLine),
				IsApplication: !aggregate.IsAndroidSystemPackage(fullMethodName),
				Image:         packageName,
			})
		}
	}

	formatTimestamp := func(t android.EventTime) uint64 {
		switch androidProfile.Clock {
		case android.GlobalClock:
			return t.Global.Secs*uint64(time.Second) + t.Global.Nanos - androidProfile.StartTime
		case android.CPUClock:
			return t.Monotonic.Cpu.Secs*uint64(time.Second) + t.Monotonic.Cpu.Nanos
		default:
			return t.Monotonic.Wall.Secs*uint64(time.Second) + t.Monotonic.Wall.Nanos
		}
	}

	// call stack per thread
	stacks := make(map[uint64][]CallTree)
	var ts uint64

	for _, event := range androidProfile.Events {
		stack, ok := stacks[event.ThreadID]
		if !ok {
			stack = make([]CallTree, 0)
		}
		frames, ok := methodIdToFrames[event.MethodID]
		if !ok || len(frames) == 0 {
			frame := Frame{
				Name:          fmt.Sprintf("unknown (id %d)", event.MethodID),
				File:          "unknown",
				Line:          0,
				IsApplication: false,
				Image:         "unknown",
			}
			methodIdToFrames[event.MethodID] = append(methodIdToFrames[event.MethodID], frame)
		}

		ts = formatTimestamp(event.Time)

		switch event.Action {
		case "Enter":
			for _, frame := range frames {
				stack = append(stack, CallTree{
					File:           frame.File,
					Image:          frame.Image,
					IsApplication:  frame.IsApplication,
					Line:           frame.Line,
					Name:           frame.Name,
					StartTimestamp: ts,
				})
			}
		case "Exit":
			frame := frames[0]

			i := len(stack) - 1
			for ; i >= 0; i-- {
				tree := stack[i]
				tree.StopTimestamp = ts

				if i > 0 {
					parent := stack[i-1]
					parent.Children = append(parent.Children, tree)
					stack[i-1] = parent
				} else {
					callTreesByThreadId[event.ThreadID] = append(callTreesByThreadId[event.ThreadID], tree)
				}

				// just need to compare to the first frame since
				// it's guaranteed to be at the bottom of the stack
				if tree.File == frame.File && tree.Image == frame.Image && tree.Name == frame.Name {
					break
				}
			}

			if i >= 0 {
				stack = stack[:i]
			} else {
				stack = make([]CallTree, 0)
			}

		default:
			return callTreesByThreadId, fmt.Errorf("Unexpected action: %s", event.Action)
		}
		stacks[event.ThreadID] = stack
	}

	for threadID, stack := range stacks {
		for i := len(stack) - 1; i >= 0; i-- {
			tree := stack[i]
			tree.StopTimestamp = ts

			if i > 0 {
				parent := stack[i-1]
				parent.Children = append(parent.Children, tree)
				stack[i-1] = parent
			} else {
				callTreesByThreadId[threadID] = append(callTreesByThreadId[threadID], tree)
			}
		}
	}

	return callTreesByThreadId, nil
}

func formatUUID(x uuid.UUID) string {
	return strings.Replace(x.String(), "-", "", -1)
}

func connect(addr string) (context.Context, clickhouse.Conn, error) {
	ctx := clickhouse.Context(context.Background())

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		Debug:           false,
		DialTimeout:     10 * time.Second,
		MaxOpenConns:    10,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})

	if err != nil {
		return ctx, conn, err
	}

	if err = conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			log.Printf("Catch exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
	}

	return ctx, conn, err
}

func setupFunctionsTable(ctx context.Context, conn clickhouse.Conn) error {
	functions_table_name := "functions_local"

	drop_statement := fmt.Sprintf(`DROP TABLE IF EXISTS %s`, functions_table_name)

	if err := conn.Exec(ctx, drop_statement); err != nil {
		return err
	}

	create_statement := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		(
				org_id UInt64,
				project_id UInt64,
				profile_id UUID,
				transaction_id UUID,
				trace_id UUID,
				received DateTime,
				android_api_level UInt32,
				device_classification LowCardinality(String),
				device_locale LowCardinality(String),
				device_manufacturer LowCardinality(String),
				device_model LowCardinality(String),
				device_os_build_number LowCardinality(String),
				device_os_name LowCardinality(String),
				device_os_version LowCardinality(String),
				environment LowCardinality(Nullable(String)),
				platform LowCardinality(String),
				transaction_name LowCardinality(String),
				version_name String,
				version_code String,
				function_id UInt64,
				symbol String,
				filename String,
				line UInt32,
				self_time UInt64,
				duration UInt64,
				timestamp UInt64,
				fingerprint UInt64,
				parent_fingerprint UInt64,
				retention_days UInt16,
				partition UInt16,
				offset UInt64,
				deleted UInt8
		)
		ENGINE = ReplacingMergeTree()
		PARTITION BY (retention_days, toMonday(received))
		ORDER BY (org_id, project_id, toStartOfDay(received), transaction_name, cityHash64(profile_id), cityHash64(function_id))
		SAMPLE BY cityHash64(profile_id)
		TTL received + toIntervalDay(retention_days)
		SETTINGS index_granularity = 8192
	`, functions_table_name)

	if err := conn.Exec(ctx, create_statement); err != nil {
		return err
	}

	return nil
}

type (
	Frame struct {
		Col           int    `json:"col,omitempty"`
		File          string `json:"file,omitempty"`
		Image         string `json:"image,omitempty"`
		IsApplication bool   `json:"is_application"`
		Line          uint32 `json:"line,omitempty"`
		Name          string `json:"name"`
	}
)
