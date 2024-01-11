package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/debugmeta"
	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/measurements"
	"github.com/getsentry/vroom/internal/metadata"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/sample"
	"github.com/getsentry/vroom/internal/speedscope"
	"github.com/getsentry/vroom/internal/timeutil"
	"github.com/getsentry/vroom/internal/transaction"
)

const maxProfileDurationForCallTrees = 15 * time.Second

var ErrProfileHasNoTrace = errors.New("profile has no trace")
var member void

type (
	void struct{}

	LegacyProfile struct {
		RawProfile

		Trace Trace `json:"profile"`
	}

	RawProfile struct {
		AndroidAPILevel      uint32                              `json:"android_api_level,omitempty"`
		Architecture         string                              `json:"architecture,omitempty"`
		BuildID              string                              `json:"build_id,omitempty"`
		DebugMeta            debugmeta.DebugMeta                 `json:"debug_meta,omitempty"`
		DeviceClassification string                              `json:"device_classification"`
		DeviceLocale         string                              `json:"device_locale"`
		DeviceManufacturer   string                              `json:"device_manufacturer"`
		DeviceModel          string                              `json:"device_model"`
		DeviceOSBuildNumber  string                              `json:"device_os_build_number,omitempty"`
		DeviceOSName         string                              `json:"device_os_name"`
		DeviceOSVersion      string                              `json:"device_os_version"`
		DurationNS           uint64                              `json:"duration_ns"`
		Environment          string                              `json:"environment,omitempty"`
		Measurements         map[string]measurements.Measurement `json:"measurements,omitempty"`
		OrganizationID       uint64                              `json:"organization_id"`
		Platform             platform.Platform                   `json:"platform"`
		Profile              json.RawMessage                     `json:"profile,omitempty"`
		JsProfile            json.RawMessage                     `json:"js_profile,omitempty"`
		ProfileID            string                              `json:"profile_id"`
		ProjectID            uint64                              `json:"project_id"`
		Received             timeutil.Time                       `json:"received"`
		RetentionDays        int                                 `json:"retention_days"`
		Timestamp            time.Time                           `json:"timestamp"`
		TraceID              string                              `json:"trace_id"`
		TransactionID        string                              `json:"transaction_id"`
		TransactionMetadata  transaction.Metadata                `json:"transaction_metadata"`
		TransactionName      string                              `json:"transaction_name"`
		TransactionTags      map[string]string                   `json:"transaction_tags,omitempty"`
		VersionCode          string                              `json:"version_code"`
		VersionName          string                              `json:"version_name"`
	}
)

func (p LegacyProfile) GetOrganizationID() uint64 {
	return p.OrganizationID
}

func (p LegacyProfile) GetProjectID() uint64 {
	return p.ProjectID
}

func (p LegacyProfile) GetID() string {
	return p.ProfileID
}

func (p LegacyProfile) Version() string {
	return FormatVersion(p.VersionName, p.VersionCode)
}

func StoragePath(organizationID, projectID uint64, profileID string) string {
	return fmt.Sprintf(
		"%d/%d/%s",
		organizationID,
		projectID,
		strings.ReplaceAll(profileID, "-", ""),
	)
}

func (p LegacyProfile) StoragePath() string {
	return StoragePath(p.OrganizationID, p.ProjectID, p.ProfileID)
}

func (p *LegacyProfile) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &p.RawProfile)
	if err != nil {
		return err
	}
	// when reading a profile from Snuba, there's no profile attached
	if len(p.Profile) == 0 {
		return nil
	}
	var raw []byte
	if p.Profile[0] == '"' {
		var s string
		err := json.Unmarshal(p.Profile, &s)
		if err != nil {
			return err
		}
		raw = []byte(s)
	} else {
		raw = p.Profile
	}
	switch p.Platform {
	case platform.Android:
		var t Android
		err := json.Unmarshal(raw, &t)
		if err != nil {
			return err
		}
		p.Trace = &t
		p.Profile = nil
	default:
		return errors.New("unknown platform")
	}
	return nil
}

func (p LegacyProfile) CallTrees() (map[uint64][]*nodetree.Node, error) {
	// Profiles longer than 5s contain a lot of call trees and it produces a lot of noise for the aggregation.
	// The majority of them might also be timing out and we want to ignore them for the aggregation.
	if time.Duration(p.DurationNS) > maxProfileDurationForCallTrees {
		slog.Debug(
			"profile is too long for call trees",
			slog.Duration("duration", time.Duration(p.DurationNS)),
		)
		return make(map[uint64][]*nodetree.Node), nil
	}
	if p.Trace == nil {
		return nil, ErrProfileHasNoTrace
	}
	return p.Trace.CallTrees(), nil
}

func (p LegacyProfile) IsSampleFormat() bool {
	return false
}

func (p *LegacyProfile) Speedscope() (speedscope.Output, error) {
	t, ok := p.Trace.(Android)
	// this is to handle only the Reactnative (android + js)
	// use case. If it's an Android profile but there is no
	// js profile, we'll skip this entirely
	if ok && p.JsProfile != nil && len(p.JsProfile) > 0 {
		st, err := unmarshalSampleProfile(p.JsProfile)
		if err == nil {
			// collect set of TIDs used and change main thread name
			tidSet := make(map[uint64]void)
			for i := range t.Threads {
				tidSet[t.Threads[i].ID] = member
				if t.Threads[i].Name == "main" {
					t.Threads[i].Name = "android_main"
				}
			}

			ap := sampleToAndroidFormat(st, uint64(len(t.Methods)), tidSet)
			t.Events = append(t.Events, ap.Events...)
			t.Methods = append(t.Methods, ap.Methods...)
			t.Threads = append(t.Threads, ap.Threads...)
			p.Trace = t
		}
	}
	o, err := p.Trace.Speedscope()
	if err != nil {
		return speedscope.Output{}, err
	}

	version := FormatVersion(p.VersionName, p.VersionCode)

	o.DurationNS = p.DurationNS
	o.Metadata = speedscope.ProfileMetadata{
		ProfileView: speedscope.ProfileView(p.RawProfile),
		Version:     version,
	}
	o.Platform = p.Platform
	o.ProfileID = p.ProfileID
	o.ProjectID = p.ProjectID
	o.TransactionName = p.TransactionName
	o.Version = version
	o.Measurements = p.Measurements

	return o, nil
}

func (p *LegacyProfile) Metadata() metadata.Metadata {
	return metadata.Metadata{
		AndroidAPILevel:      p.AndroidAPILevel,
		Architecture:         "unknown",
		DeviceClassification: p.DeviceClassification,
		DeviceLocale:         p.DeviceLocale,
		DeviceManufacturer:   p.DeviceManufacturer,
		DeviceModel:          p.DeviceModel,
		DeviceOSBuildNumber:  p.DeviceOSBuildNumber,
		DeviceOSName:         p.DeviceOSName,
		DeviceOSVersion:      p.DeviceOSVersion,
		ID:                   p.ProfileID,
		ProjectID:            strconv.FormatUint(p.GetProjectID(), 10),
		Timestamp:            p.Timestamp.Unix(),
		TraceDurationMs:      float64(p.DurationNS) / 1_000_000,
		TransactionID:        p.TransactionID,
		TransactionName:      p.TransactionName,
		VersionCode:          p.VersionCode,
		VersionName:          p.VersionName,
	}
}

func (p LegacyProfile) GetPlatform() platform.Platform {
	return p.Platform
}

func (p LegacyProfile) GetEnvironment() string {
	return p.Environment
}

func (p LegacyProfile) GetTransaction() transaction.Transaction {
	return transaction.Transaction{
		ActiveThreadID: p.Trace.ActiveThreadID(),
		DurationNS:     p.DurationNS,
		ID:             p.TransactionID,
		Name:           p.TransactionName,
		TraceID:        p.TraceID,
	}
}

func (p LegacyProfile) GetDebugMeta() debugmeta.DebugMeta {
	return p.DebugMeta
}

func (p LegacyProfile) GetTimestamp() time.Time {
	if p.Timestamp.IsZero() {
		return time.Time(p.Received)
	}
	return p.Timestamp
}

func (p LegacyProfile) GetReceived() time.Time {
	return p.Received.Time()
}

func (p *LegacyProfile) Normalize() {
	switch t := p.Trace.(type) {
	case *Android:
		t.NormalizeMethods(p)
	}

	if p.BuildID != "" {
		p.DebugMeta.Images = append(p.DebugMeta.Images, debugmeta.Image{
			Type: "proguard",
			UUID: p.BuildID,
		})
		p.BuildID = ""
	}
}

func (p LegacyProfile) GetRelease() string {
	return FormatVersion(p.VersionName, p.VersionCode)
}

func (p LegacyProfile) GetRetentionDays() int {
	return p.RetentionDays
}

func (p LegacyProfile) GetDurationNS() uint64 {
	return p.Trace.DurationNS()
}

func (p LegacyProfile) GetTransactionMetadata() transaction.Metadata {
	return p.TransactionMetadata
}

func (p LegacyProfile) GetTransactionTags() map[string]string {
	return p.TransactionTags
}

func (p LegacyProfile) GetMeasurements() map[string]measurements.Measurement {
	return p.Measurements
}

func sampleToAndroidFormat(p sample.Trace, offset uint64, usedTids map[uint64]void) Android {
	//var Clock Clock
	var events []AndroidEvent
	var methods []AndroidMethod
	//var StartTime uint64
	var threads []AndroidThread

	tidLastTimeNs := make(map[uint64]uint64)
	tidLastStack := make(map[uint64][]int)

	methodSet := make(map[uint64]void)
	threadSet := make(map[uint64]void)

	mainTID, newMainTID := getMainThreadIDs(p.ThreadMetadata, usedTids)

	for si, sample := range p.Samples {
		sampleTID := sample.ThreadID
		if sampleTID == mainTID {
			sampleTID = newMainTID
		}
		tidLastTimeNs[sampleTID] = sample.ElapsedSinceStartNS
		eventTime := getEventTimeFromElapsedNanoseconds(sample.ElapsedSinceStartNS)
		i := 0
		lastStack := tidLastStack[sampleTID]
		currentStack := p.Stacks[sample.StackID]
		for i < len(lastStack) && i < len(currentStack) {
			if lastStack[i] != currentStack[i] {
				break
			}
			i++
		}
		// at this point we've scanned through all the common frames at the bottom
		// of the stack. For any frames left in the older stack we need to generate
		// an "exit" event.
		// This logic applies to all samples except the 1st
		if si > 0 {
			for j := len(lastStack) - 1; j >= i; j-- {
				frameID := lastStack[j]
				offsetID := uint64(frameID) + offset

				ev := AndroidEvent{
					Action:   ExitAction,
					ThreadID: sampleTID,
					MethodID: offsetID,
					Time:     eventTime,
				}

				events = append(events, ev)
			}
		}

		// For any frames left in the current stack we need to generate
		// an "enter" event.
		for _, frameID := range currentStack[i:] {
			offsetID := uint64(frameID) + offset

			if _, exists := methodSet[offsetID]; !exists {
				updateMethods(methodSet, &methods, p.Frames[frameID], offsetID)
			}
			if _, exists := threadSet[sampleTID]; !exists {
				metadata := p.ThreadMetadata[strconv.FormatUint(sample.ThreadID, 10)]
				updateThreads(threadSet, &threads, sampleTID, &metadata)
			}
			ev := AndroidEvent{
				Action:   EnterAction,
				ThreadID: sampleTID,
				MethodID: offsetID,
				Time:     eventTime,
			}
			events = append(events, ev)
		}
		tidLastStack[sampleTID] = currentStack
	} // end sample loop

	// once we looped all the samples, for each thread
	// we close all the events that are left open
	for tid, lastStack := range tidLastStack {
		// for the last exit events we use as elpased time
		// whatever the latest time was plus 10ms
		closingTimeNs := tidLastTimeNs[tid] + 1e7
		eventTime := getEventTimeFromElapsedNanoseconds(closingTimeNs)

		for i := len(lastStack) - 1; i >= 0; i-- {
			frameID := lastStack[i]
			offsetID := uint64(frameID) + offset

			ev := AndroidEvent{
				Action:   ExitAction,
				ThreadID: tid,
				MethodID: offsetID,
				Time:     eventTime,
			}

			events = append(events, ev)
		}
	}

	return Android{
		Clock:   DualClock,
		Events:  events,
		Methods: methods,
		Threads: threads,
	}

	// TODO: write unit tests
}

func updateMethods(methodSet map[uint64]void, methods *[]AndroidMethod, fr frame.Frame, offsetID uint64) {
	method := AndroidMethod{
		ID:         offsetID,
		Name:       fr.Function,
		SourceFile: fr.Path,
		SourceLine: fr.Line,
		InApp:      fr.InApp,
	}
	*methods = append(*methods, method)
	methodSet[offsetID] = member
}

func updateThreads(threadSet map[uint64]void, threads *[]AndroidThread, threadID uint64, metadata *sample.ThreadMetadata) {
	const mainThreadName = "JavaScriptThread"
	thread := AndroidThread{
		ID:   threadID,
		Name: metadata.Name,
	}
	// In a few other places (CallTree), we rely on the thread
	// name "main" to figure out which thread data to use.
	// For reactNative we want to ignore the Android "main"
	// thread and instead use the js as the main one.
	if thread.Name == mainThreadName {
		thread.Name = "main"
	}

	*threads = append(*threads, thread)
	threadSet[threadID] = member
}

// Native Android profile and JS profile have a thread ID in common
// As of now, we want to show them separately instead of merged
// To do so, the thread in the JS profile, will get a new ID
// that is not yet used by any of the threads in the native Android
// profile.
func getMainThreadIDs(threads map[string]sample.ThreadMetadata, usedTids map[uint64]void) (uint64, uint64) {
	const mainThreadName = "JavaScriptThread"
	var tid uint64
	var newTid uint64
	for id, threadMetadata := range threads {
		if threadMetadata.Name == mainThreadName {
			intNum, _ := strconv.ParseInt(id, 10, 64)
			tid = uint64(intNum)
			newTid = getUniqueTid(tid, usedTids)
			break
		}
	}
	return tid, newTid
}

func getUniqueTid(tid uint64, usedTids map[uint64]void) uint64 {
	for i := tid + 1; ; i++ {
		if _, exists := usedTids[i]; !exists {
			return i
		}
	}
}

func getEventTimeFromElapsedNanoseconds(ns uint64) EventTime {
	return EventTime{
		Monotonic: EventMonotonic{
			Wall: Duration{
				Secs:  (ns / 1e9),
				Nanos: (ns % 1e9),
			},
		},
	}
}

func unmarshalSampleProfile(p json.RawMessage) (sample.Trace, error) {
	var st sample.Trace
	err := json.Unmarshal(p, &st)
	if err != nil {
		return sample.Trace{}, err
	}

	return st, nil
}
