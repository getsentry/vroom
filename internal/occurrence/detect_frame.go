package occurrence

import (
	"crypto/md5"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
	"github.com/google/uuid"
)

type (
	DetectExactFrameMetadata struct {
		ActiveThreadOnly   bool
		FunctionsByPackage map[string]map[string]struct{}
		IssueTitle         string
	}

	nodeKey struct {
		Package  string
		Function string
	}

	nodeInfo struct {
		Node       *nodetree.Node
		StackTrace []frame.Frame
	}
)

var (
	detectFrameMetadata = map[platform.Platform][]DetectExactFrameMetadata{
		platform.Node: []DetectExactFrameMetadata{
			DetectExactFrameMetadata{
				ActiveThreadOnly: true,
				FunctionsByPackage: map[string]map[string]struct{}{
					"node:fs": map[string]struct{}{
						"accessSync":          struct{}{},
						"appendFileSync":      struct{}{},
						"chmodSync":           struct{}{},
						"chownSync":           struct{}{},
						"closeSync":           struct{}{},
						"copyFileSync":        struct{}{},
						"cpSync":              struct{}{},
						"existsSync":          struct{}{},
						"fchmodSync":          struct{}{},
						"fchownSync":          struct{}{},
						"fdatasyncSync":       struct{}{},
						"fstatSync":           struct{}{},
						"fsyncSync":           struct{}{},
						"ftruncateSync":       struct{}{},
						"futimesSync":         struct{}{},
						"lchmodSync":          struct{}{},
						"lchownSync":          struct{}{},
						"linkSync":            struct{}{},
						"lstatSync":           struct{}{},
						"lutimesSync":         struct{}{},
						"mkdirSync":           struct{}{},
						"mkdtempSync":         struct{}{},
						"openSync":            struct{}{},
						"opendirSync":         struct{}{},
						"readFileSync":        struct{}{},
						"readSync":            struct{}{},
						"readdirSync":         struct{}{},
						"readlinkSync":        struct{}{},
						"readvSync":           struct{}{},
						"realpathSync":        struct{}{},
						"realpathSync.native": struct{}{},
						"renameSync":          struct{}{},
						"rmSync":              struct{}{},
						"rmdirSync":           struct{}{},
						"statSync":            struct{}{},
						"symlinkSync":         struct{}{},
						"truncateSync":        struct{}{},
						"unlinkSync":          struct{}{},
						"utimesSync":          struct{}{},
						"writeFileSync":       struct{}{},
						"writeSync":           struct{}{},
						"writevSync":          struct{}{},
					},
				},
				IssueTitle: "Synchronous function called on main thread",
			},
		},
		platform.Cocoa: []DetectExactFrameMetadata{
			DetectExactFrameMetadata{
				ActiveThreadOnly: true,
				FunctionsByPackage: map[string]map[string]struct{}{
					"AppleJPEG": map[string]struct{}{
						"applejpeg_decode_image_all": struct{}{},
					},
					"AttributeGraph": map[string]struct{}{
						"AG::LayoutDescriptor::make_layout(AG::swift::metadata const*, AGComparisonMode, AG::LayoutDescriptor::HeapMode)": struct{}{},
					},
					"CoreData": map[string]struct{}{
						"-[NSManagedObjectContext countForFetchRequest:error:]":                 struct{}{},
						"-[NSManagedObjectContext executeFetchRequest:error:]":                  struct{}{},
						"-[NSManagedObjectContext executeRequest:error:]":                       struct{}{},
						"-[NSManagedObjectContext mergeChangesFromContextDidSaveNotification:]": struct{}{},
						"-[NSManagedObjectContext obtainPermanentIDsForObjects:error:]":         struct{}{},
						"-[NSManagedObjectContext performBlockAndWait:]":                        struct{}{},
						"-[NSManagedObjectContext save:]":                                       struct{}{},
						"NSManagedObjectContext.fetch<A>(NSFetchRequest<A>)":                    struct{}{},
					},
					"CoreFoundation": map[string]struct{}{
						"CFReadStreamRead":                         struct{}{},
						"CFURLConnectionSendSynchronousRequest":    struct{}{},
						"CFURLCreateData":                          struct{}{},
						"CFURLCreateDataAndPropertiesFromResource": struct{}{},
						"CFURLWriteDataAndPropertiesToResource":    struct{}{},
						"CFWriteStreamWrite":                       struct{}{},
					},
					"CoreML": map[string]struct{}{
						"+[MLModel modelWithContentsOfURL:configuration:error:]":         struct{}{},
						"-[MLNeuralNetworkEngine predictionFromFeatures:options:error:]": struct{}{},
					},
					"CoreAutoLayout": map[string]struct{}{
						"-[NSISEngine withBehaviors:performModifications:]": struct{}{},
					},
					"Foundation": map[string]struct{}{
						"+[NSJSONSerialization JSONObjectWithStream:options:error:]":              struct{}{},
						"+[NSJSONSerialization writeJSONObject:toStream:options:error:]":          struct{}{},
						"+[NSRegularExpression regularExpressionWithPattern:options:error:]":      struct{}{},
						"+[NSURLConnection sendSynchronousRequest:returningResponse:error:]":      struct{}{},
						"-[NSData(NSData) initWithContentsOfMappedFile:]":                         struct{}{},
						"-[NSData(NSData) initWithContentsOfURL:]":                                struct{}{},
						"-[NSData(NSData) initWithContentsOfURL:options:maxLength:error:]":        struct{}{},
						"-[NSData(NSData) writeToFile:atomically:]":                               struct{}{},
						"-[NSData(NSData) writeToFile:atomically:error:]":                         struct{}{},
						"-[NSData(NSData) writeToFile:options:error:]":                            struct{}{},
						"-[NSData(NSData) writeToURL:atomically:]":                                struct{}{},
						"-[NSData(NSData) writeToURL:options:error:]":                             struct{}{},
						"-[NSFileManager contentsAtPath:]":                                        struct{}{},
						"-[NSFileManager createFileAtPath:contents:attributes:]":                  struct{}{},
						"-[NSISEngine performModifications:withUnsatisfiableConstraintsHandler:]": struct{}{},
						"-[NSISEngine withBehaviors:performModifications:]":                       struct{}{},
						"-[NSRegularExpression initWithPattern:options:error:]":                   struct{}{},
						"@nonobjc NSData.init(contentsOf: URL, options: NSDataReadingOptions)":    struct{}{},
						"Data.init(contentsOf: __shared URL, options: NSDataReadingOptions)":      struct{}{},
						"JSONDecoder.decode<A>(_: A.Type, from: Any)":                             struct{}{},
						"JSONDecoder.decode<A>(_: A.Type, from: Data)":                            struct{}{},
						"JSONDecoder.decode<A>(_: A.Type, jsonData: Data, logErrors: Bool)":       struct{}{},
						"JSONEncoder.encode<A>(A)":                                                struct{}{},
						"NSFileManager.contents(atURL: URL)":                                      struct{}{},
					},
					"ImageIO": map[string]struct{}{
						"DecodeImageData":   struct{}{},
						"DecodeImageStream": struct{}{},
						"GIFReadPlugin::DoDecodeImageData(IIOImageReadSession*, GlobalGIFInfo*, ReadPluginData const&, GIFPluginData const&, unsigned char*, unsigned long, std::__1::shared_ptr<GIFBufferInfo>, long*)": struct{}{},
						"IIOImageProviderInfo::CopyImageBlockSetWithOptions(void*, CGImageProvider*, CGRect, CGSize, __CFDictionary const*)":                                                                             struct{}{},
						"LZWDecode":  struct{}{},
						"NeXTDecode": struct{}{},
						"PNGReadPlugin::DecodeFrameStandard(IIOImageReadSession*, ReadPluginData const&, PNGPluginData const&, IIODecodeFrameParams&)": struct{}{},
						"VP8Decode":        struct{}{},
						"VP8DecodeMB":      struct{}{},
						"WebPDecode":       struct{}{},
						"jpeg_huff_decode": struct{}{},
					},
					"libcompression.dylib": map[string]struct{}{
						"BrotliDecoderDecompress": struct{}{},
						"brotli_encode_buffer":    struct{}{},
						"lz4_decode":              struct{}{},
						"lz4_decode_asm":          struct{}{},
						"lzfseDecode":             struct{}{},
						"lzfseEncode":             struct{}{},
						"lzfseStreamDecode":       struct{}{},
						"lzfseStreamEncode":       struct{}{},
						"lzvnDecode":              struct{}{},
						"lzvnEncode":              struct{}{},
						"lzvnStreamDecode":        struct{}{},
						"lzvnStreamEncode":        struct{}{},
						"zlibDecodeBuffer":        struct{}{},
						"zlib_decode_buffer":      struct{}{},
						"zlib_encode_buffer":      struct{}{},
					},
					"libsqlite3.dylib": map[string]struct{}{
						"sqlite3_blob_read":      struct{}{},
						"sqlite3_column_blob":    struct{}{},
						"sqlite3_column_bytes":   struct{}{},
						"sqlite3_column_double":  struct{}{},
						"sqlite3_column_int":     struct{}{},
						"sqlite3_column_int64":   struct{}{},
						"sqlite3_column_text":    struct{}{},
						"sqlite3_column_text16":  struct{}{},
						"sqlite3_column_value":   struct{}{},
						"sqlite3_step":           struct{}{},
						"sqlite3_value_blob":     struct{}{},
						"sqlite3_value_double":   struct{}{},
						"sqlite3_value_int":      struct{}{},
						"sqlite3_value_int64":    struct{}{},
						"sqlite3_value_pointer":  struct{}{},
						"sqlite3_value_text":     struct{}{},
						"sqlite3_value_text16":   struct{}{},
						"sqlite3_value_text16be": struct{}{},
						"sqlite3_value_text16le": struct{}{},
					},
					"libswiftCoreData.dylib": map[string]struct{}{
						"NSManagedObjectContext.count<A>(for: NSFetchRequest<A>)":                                      struct{}{},
						"NSManagedObjectContext.fetch<A>(NSFetchRequest<A>)":                                           struct{}{},
						"NSManagedObjectContext.perform<A>(schedule: NSManagedObjectContext.ScheduledTaskType, _: ())": struct{}{},
					},
					"libswiftFoundation.dylib": map[string]struct{}{
						"__JSONDecoder.decode<A>(A.Type)": struct{}{},
						"__JSONEncoder.encode<A>(A)":      struct{}{},
					},
					"libsystem_c.dylib": map[string]struct{}{
						"__fread": struct{}{},
						"fread":   struct{}{},
					},
					"libxpc.dylib": map[string]struct{}{
						"xpc_connection_send_message_with_reply_sync": struct{}{},
					},
					"QuartzCore": map[string]struct{}{
						"CA::Layer::layout_and_display_if_needed(CA::Transaction*)": struct{}{},
						"CA::Layer::layout_if_needed(CA::Transaction*)":             struct{}{},
					},
					"SwiftUI": map[string]struct{}{
						"UnaryLayoutEngine.sizeThatFits(_ProposedSize)":                      struct{}{},
						"ViewRendererHost.render(interval: Double, updateDisplayList: Bool)": struct{}{},
						"ViewRendererHost.updateViewGraph<A>(body: (ViewGraph))":             struct{}{},
					},
					"UIKit": map[string]struct{}{
						"-[UINib instantiateWithOwner:options:]": struct{}{},
					},
				},
				IssueTitle: "File I/O function called on main thread",
			},
		},
	}
)

// DetectFrames detects occurrence of an issue based by matching frames of the profile on a list of frames
func detectFrame(p profile.Profile, callTreesPerThreadID map[uint64][]*nodetree.Node, metadata DetectExactFrameMetadata, occurrences *[]Occurrence) {
	// List nodes matching criteria
	nodes := make(map[nodeKey]nodeInfo)
	if metadata.ActiveThreadOnly {
		callTrees, exists := callTreesPerThreadID[p.Transaction().ActiveThreadID]
		if !exists {
			return
		}
		for _, root := range callTrees {
			var stackTrace []frame.Frame
			detectFrameInCallTree(root, metadata.FunctionsByPackage, nodes, &stackTrace)
		}
	} else {
		for _, callTrees := range callTreesPerThreadID {
			for _, root := range callTrees {
				var stackTrace []frame.Frame
				detectFrameInCallTree(root, metadata.FunctionsByPackage, nodes, &stackTrace)
			}
		}
	}

	// Create occurrences
	for _, n := range nodes {
		*occurrences = append(*occurrences, NewOccurrence(p, metadata, n))
	}
}

func detectFrameInCallTree(n *nodetree.Node, functionsByPackage map[string]map[string]struct{}, nodes map[nodeKey]nodeInfo, stackTrace *[]frame.Frame) {
	*stackTrace = append(*stackTrace, n.Frame())
	if functions, exists := functionsByPackage[n.Package]; exists {
		if _, exists := functions[n.Name]; exists {
			nk := nodeKey{Package: n.Package, Function: n.Name}
			if _, exists := nodes[nk]; !exists {
				nodes[nk] = nodeInfo{
					Node:       n,
					StackTrace: *stackTrace,
				}
			}
		}
	}
	for _, c := range n.Children {
		newStackTrace := *stackTrace
		detectFrameInCallTree(c, functionsByPackage, nodes, &newStackTrace)
	}
}

func NewOccurrence(p profile.Profile, metadata DetectExactFrameMetadata, ni nodeInfo) Occurrence {
	t := p.Transaction()
	h := md5.New()
	_, _ = io.WriteString(h, strconv.FormatUint(p.ProjectID(), 10))
	_, _ = io.WriteString(h, metadata.IssueTitle)
	_, _ = io.WriteString(h, t.Name)
	_, _ = io.WriteString(h, strconv.Itoa(int(ProfileBlockedThreadType)))
	_, _ = io.WriteString(h, ni.Node.Package)
	_, _ = io.WriteString(h, ni.Node.Name)
	fingerprint := fmt.Sprintf("%x", h.Sum(nil))
	return Occurrence{
		DetectionTime: time.Now().UTC(),
		Event: Event{
			Environment: p.Environment(),
			ID:          p.ID(),
			Platform:    p.Platform(),
			ProjectID:   p.ProjectID(),
			Received:    p.Received(),
			StackTrace:  StackTrace{Frames: ni.StackTrace},
			Tags:        map[string]string{},
			Timestamp:   p.Timestamp(),
			Transaction: t.ID,
		},
		EvidenceData: map[string]interface{}{},
		EvidenceDisplay: []Evidence{
			Evidence{
				Name:      EvidenceNameFunction,
				Value:     ni.Node.Name,
				Important: true,
			},
			Evidence{
				Name:  EvidenceNamePackage,
				Value: ni.Node.Package,
			},
		},
		Fingerprint: fingerprint,
		ID:          uuid.New().String(),
		IssueTitle:  metadata.IssueTitle,
		Subtitle:    t.Name,
		Type:        ProfileBlockedThreadType,
	}
}
