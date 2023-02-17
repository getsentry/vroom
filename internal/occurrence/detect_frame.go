package occurrence

import (
	"time"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
)

type (
	DetectExactFrameOptions struct {
		ActiveThreadOnly   bool
		DurationThreshold  time.Duration
		FunctionsByPackage map[string]map[string]Category
	}

	nodeKey struct {
		Package  string
		Function string
	}

	nodeInfo struct {
		Category   Category
		Node       *nodetree.Node
		StackTrace []frame.Frame
	}

	Category string
)

const (
	Compression      Category = "compression"
	CoreDataBlock    Category = "core_data_block"
	CoreDataMerge    Category = "core_data_merge"
	CoreDataRead     Category = "core_data_read"
	CoreDataWrite    Category = "core_data_write"
	FileRead         Category = "file_read"
	FileWrite        Category = "file_write"
	HTTP             Category = "http"
	ImageDecode      Category = "image_decode"
	ImageEncode      Category = "image_encode"
	JSONDecode       Category = "json_decode"
	JSONEncode       Category = "json_encode"
	MLModelInference Category = "ml_model_inference"
	MLModelLoad      Category = "ml_model_load"
	Regex            Category = "regex"
	SQL              Category = "sql"
	ViewInflation    Category = "view_inflation"
	ViewLayout       Category = "view_layout"
	ViewRender       Category = "view_render"
	ViewUpdate       Category = "view_update"
	XPC              Category = "xpc"
)

var (
	detectFrameJobs = map[platform.Platform][]DetectExactFrameOptions{
		platform.Node: {
			{
				ActiveThreadOnly: true,
				FunctionsByPackage: map[string]map[string]Category{
					"node:fs": {
						"accessSync":          FileRead,
						"appendFileSync":      FileRead,
						"chmodSync":           FileRead,
						"chownSync":           FileRead,
						"closeSync":           FileRead,
						"copyFileSync":        FileRead,
						"cpSync":              FileRead,
						"existsSync":          FileRead,
						"fchmodSync":          FileRead,
						"fchownSync":          FileRead,
						"fdatasyncSync":       FileRead,
						"fstatSync":           FileRead,
						"fsyncSync":           FileRead,
						"ftruncateSync":       FileRead,
						"futimesSync":         FileRead,
						"lchmodSync":          FileRead,
						"lchownSync":          FileRead,
						"linkSync":            FileRead,
						"lstatSync":           FileRead,
						"lutimesSync":         FileRead,
						"mkdirSync":           FileRead,
						"mkdtempSync":         FileRead,
						"openSync":            FileRead,
						"opendirSync":         FileRead,
						"readFileSync":        FileRead,
						"readSync":            FileRead,
						"readdirSync":         FileRead,
						"readlinkSync":        FileRead,
						"readvSync":           FileRead,
						"realpathSync":        FileRead,
						"realpathSync.native": FileRead,
						"renameSync":          FileRead,
						"rmSync":              FileRead,
						"rmdirSync":           FileRead,
						"statSync":            FileRead,
						"symlinkSync":         FileRead,
						"truncateSync":        FileRead,
						"unlinkSync":          FileRead,
						"utimesSync":          FileRead,
						"writeFileSync":       FileRead,
						"writeSync":           FileRead,
						"writevSync":          FileRead,
					},
				},
			},
		},
		platform.Cocoa: {
			{
				ActiveThreadOnly:  true,
				DurationThreshold: 16 * time.Millisecond,
				FunctionsByPackage: map[string]map[string]Category{
					"AppleJPEG": {
						"applejpeg_decode_image_all": ImageDecode,
					},
					"AttributeGraph": {
						"AG::LayoutDescriptor::make_layout(AG::swift::metadata const*, AGComparisonMode, AG::LayoutDescriptor::HeapMode)": ViewLayout,
					},
					"CoreData": {
						"-[NSManagedObjectContext countForFetchRequest:error:]":                 CoreDataRead,
						"-[NSManagedObjectContext executeFetchRequest:error:]":                  CoreDataRead,
						"-[NSManagedObjectContext executeRequest:error:]":                       CoreDataRead,
						"-[NSManagedObjectContext mergeChangesFromContextDidSaveNotification:]": CoreDataMerge,
						"-[NSManagedObjectContext obtainPermanentIDsForObjects:error:]":         CoreDataWrite,
						"-[NSManagedObjectContext performBlockAndWait:]":                        CoreDataBlock,
						"-[NSManagedObjectContext save:]":                                       CoreDataWrite,
						"NSManagedObjectContext.fetch<A>(NSFetchRequest<A>)":                    CoreDataRead,
					},
					"CoreFoundation": {
						"CFReadStreamRead":                         FileRead,
						"CFURLConnectionSendSynchronousRequest":    HTTP,
						"CFURLCreateData":                          FileRead,
						"CFURLCreateDataAndPropertiesFromResource": FileRead,
						"CFURLWriteDataAndPropertiesToResource":    FileWrite,
						"CFWriteStreamWrite":                       FileWrite,
					},
					"CoreML": {
						"+[MLModel modelWithContentsOfURL:configuration:error:]":         MLModelLoad,
						"-[MLNeuralNetworkEngine predictionFromFeatures:options:error:]": MLModelInference,
					},
					"Foundation": {
						"+[NSJSONSerialization JSONObjectWithStream:options:error:]":              JSONDecode,
						"+[NSJSONSerialization writeJSONObject:toStream:options:error:]":          JSONEncode,
						"+[NSRegularExpression regularExpressionWithPattern:options:error:]":      Regex,
						"+[NSURLConnection sendSynchronousRequest:returningResponse:error:]":      HTTP,
						"-[NSData(NSData) initWithContentsOfMappedFile:]":                         FileRead,
						"-[NSData(NSData) initWithContentsOfURL:]":                                FileRead,
						"-[NSData(NSData) initWithContentsOfURL:options:maxLength:error:]":        FileRead,
						"-[NSData(NSData) writeToFile:atomically:]":                               FileWrite,
						"-[NSData(NSData) writeToFile:atomically:error:]":                         FileWrite,
						"-[NSData(NSData) writeToFile:options:error:]":                            FileWrite,
						"-[NSData(NSData) writeToURL:atomically:]":                                FileWrite,
						"-[NSData(NSData) writeToURL:options:error:]":                             FileWrite,
						"-[NSFileManager contentsAtPath:]":                                        FileRead,
						"-[NSFileManager createFileAtPath:contents:attributes:]":                  FileWrite,
						"-[NSISEngine performModifications:withUnsatisfiableConstraintsHandler:]": ViewLayout,
						"-[NSRegularExpression initWithPattern:options:error:]":                   Regex,
						"@nonobjc NSData.init(contentsOf: URL, options: NSDataReadingOptions)":    FileRead,
						"Data.init(contentsOf: __shared URL, options: NSDataReadingOptions)":      FileRead,
						"JSONDecoder.decode<A>(_: A.Type, from: Any)":                             JSONDecode,
						"JSONDecoder.decode<A>(_: A.Type, from: Data)":                            JSONDecode,
						"JSONDecoder.decode<A>(_: A.Type, jsonData: Data, logErrors: Bool)":       JSONDecode,
						"JSONEncoder.encode<A>(A)":                                                JSONEncode,
						"NSFileManager.contents(atURL: URL)":                                      FileRead,
					},
					"ImageIO": {
						"DecodeImageData":   ImageDecode,
						"DecodeImageStream": ImageDecode,
						"GIFReadPlugin::DoDecodeImageData(IIOImageReadSession*, GlobalGIFInfo*, ReadPluginData const&, GIFPluginData const&, unsigned char*, unsigned long, std::__1::shared_ptr<GIFBufferInfo>, long*)": ImageDecode,
						"IIOImageProviderInfo::CopyImageBlockSetWithOptions(void*, CGImageProvider*, CGRect, CGSize, __CFDictionary const*)":                                                                             ImageDecode,
						"LZWDecode":  ImageDecode,
						"NeXTDecode": ImageDecode,
						"PNGReadPlugin::DecodeFrameStandard(IIOImageReadSession*, ReadPluginData const&, PNGPluginData const&, IIODecodeFrameParams&)": ImageDecode,
						"VP8Decode":        ImageDecode,
						"VP8DecodeMB":      ImageDecode,
						"WebPDecode":       ImageDecode,
						"jpeg_huff_decode": ImageDecode,
					},
					"libcompression.dylib": {
						"BrotliDecoderDecompress": Compression,
						"brotli_encode_buffer":    Compression,
						"lz4_decode":              Compression,
						"lz4_decode_asm":          Compression,
						"lzfseDecode":             Compression,
						"lzfseEncode":             Compression,
						"lzfseStreamDecode":       Compression,
						"lzfseStreamEncode":       Compression,
						"lzvnDecode":              Compression,
						"lzvnEncode":              Compression,
						"lzvnStreamDecode":        Compression,
						"lzvnStreamEncode":        Compression,
						"zlibDecodeBuffer":        Compression,
						"zlib_decode_buffer":      Compression,
						"zlib_encode_buffer":      Compression,
					},
					"libsqlite3.dylib": {
						"sqlite3_blob_read":      SQL,
						"sqlite3_column_blob":    SQL,
						"sqlite3_column_bytes":   SQL,
						"sqlite3_column_double":  SQL,
						"sqlite3_column_int":     SQL,
						"sqlite3_column_int64":   SQL,
						"sqlite3_column_text":    SQL,
						"sqlite3_column_text16":  SQL,
						"sqlite3_column_value":   SQL,
						"sqlite3_step":           SQL,
						"sqlite3_value_blob":     SQL,
						"sqlite3_value_double":   SQL,
						"sqlite3_value_int":      SQL,
						"sqlite3_value_int64":    SQL,
						"sqlite3_value_pointer":  SQL,
						"sqlite3_value_text":     SQL,
						"sqlite3_value_text16":   SQL,
						"sqlite3_value_text16be": SQL,
						"sqlite3_value_text16le": SQL,
					},
					"libswiftCoreData.dylib": {
						"NSManagedObjectContext.count<A>(for: NSFetchRequest<A>)":                                      CoreDataRead,
						"NSManagedObjectContext.fetch<A>(NSFetchRequest<A>)":                                           CoreDataRead,
						"NSManagedObjectContext.perform<A>(schedule: NSManagedObjectContext.ScheduledTaskType, _: ())": CoreDataBlock,
					},
					"libswiftFoundation.dylib": {
						"__JSONDecoder.decode<A>(A.Type)": JSONDecode,
						"__JSONEncoder.encode<A>(A)":      JSONEncode,
					},
					"libsystem_c.dylib": {
						"__fread": FileRead,
						"fread":   FileRead,
					},
					"libxpc.dylib": {
						"xpc_connection_send_message_with_reply_sync": XPC,
					},
					"SwiftUI": {
						"UnaryLayoutEngine.sizeThatFits(_ProposedSize)":                      ViewLayout,
						"ViewRendererHost.render(interval: Double, updateDisplayList: Bool)": ViewRender,
						"ViewRendererHost.updateViewGraph<A>(body: (ViewGraph))":             ViewUpdate,
					},
					"UIKit": {
						"-[UINib instantiateWithOwner:options:]": ViewInflation,
					},
					"libsystem_kernel.dylib": {
						"mach_msg_trap": FileRead,
					},
				},
			},
		},
	}
)

// DetectFrames detects occurrence of an issue based by matching frames of the profile on a list of frames.
func detectFrame(p profile.Profile, callTreesPerThreadID map[uint64][]*nodetree.Node, options DetectExactFrameOptions, occurrences *[]*Occurrence) {
	// List nodes matching criteria
	nodes := make(map[nodeKey]nodeInfo)
	if options.ActiveThreadOnly {
		callTrees, exists := callTreesPerThreadID[p.Transaction().ActiveThreadID]
		if !exists {
			return
		}
		for _, root := range callTrees {
			var stackTrace []frame.Frame
			detectFrameInCallTree(root, options, nodes, &stackTrace)
		}
	} else {
		for _, callTrees := range callTreesPerThreadID {
			for _, root := range callTrees {
				var stackTrace []frame.Frame
				detectFrameInCallTree(root, options, nodes, &stackTrace)
			}
		}
	}

	// Create occurrences.
	for _, n := range nodes {
		*occurrences = append(*occurrences, NewOccurrence(p, n))
	}
}

func detectFrameInCallTree(n *nodetree.Node, options DetectExactFrameOptions, nodes map[nodeKey]nodeInfo, stackTrace *[]frame.Frame) {
	*stackTrace = append(*stackTrace, n.Frame())
	if functions, exists := options.FunctionsByPackage[n.Package]; exists {
		// Only use time threshold when the sample count is more than one to avoid sampling issues showing up as blocking issues.
		if category, exists := functions[n.Name]; exists && n.DurationNS > uint64(options.DurationThreshold) && n.SampleCount != 1 {
			nk := nodeKey{Package: n.Package, Function: n.Name}
			if _, exists := nodes[nk]; !exists {
				nodes[nk] = nodeInfo{
					Category:   category,
					Node:       n,
					StackTrace: *stackTrace,
				}
			}
		}
	}
	for _, c := range n.Children {
		newStackTrace := *stackTrace
		detectFrameInCallTree(c, options, nodes, &newStackTrace)
	}
}
