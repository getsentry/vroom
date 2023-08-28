package occurrence

import (
	"strings"
	"time"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/nodetree"
	"github.com/getsentry/vroom/internal/platform"
	"github.com/getsentry/vroom/internal/profile"
)

type (
	DetectFrameOptions interface {
		OnlyCheckActiveThread() bool
		CheckNode(*nodetree.Node) *nodeInfo
	}

	DetectExactFrameOptions struct {
		ActiveThreadOnly   bool
		DurationThreshold  time.Duration
		FunctionsByPackage map[string]map[string]Category

		// SampleThreshold is the minimum number of samples in which we need to
		// detect the frame in order to create an occurrence.
		SampleThreshold int
	}

	DetectAndroidFrameOptions struct {
		ActiveThreadOnly   bool
		DurationThreshold  time.Duration
		FunctionsByPackage map[string]map[string]Category

		// SampleThreshold is the minimum number of samples in which we need to
		// detect the frame in order to create an occurrence.
		SampleThreshold int
	}

	nodeKey struct {
		Package  string
		Function string
	}

	nodeInfo struct {
		Category   Category
		Node       nodetree.Node
		StackTrace []frame.Frame
	}
)

const (
	Base64Decode     Category = "base64_decode"
	Base64Encode     Category = "base64_encode"
	Compression      Category = "compression"
	CoreDataBlock    Category = "core_data_block"
	CoreDataMerge    Category = "core_data_merge"
	CoreDataRead     Category = "core_data_read"
	CoreDataWrite    Category = "core_data_write"
	Decompression    Category = "decompression"
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
	SourceContext    Category = "source_context"
	ThreadWait       Category = "thread_wait"
	ViewInflation    Category = "view_inflation"
	ViewLayout       Category = "view_layout"
	ViewRender       Category = "view_render"
	ViewUpdate       Category = "view_update"
	XPC              Category = "xpc"
)

func (options DetectExactFrameOptions) OnlyCheckActiveThread() bool {
	return options.ActiveThreadOnly
}

func (options DetectExactFrameOptions) CheckNode(n *nodetree.Node) *nodeInfo {
	// Check if we have a list of functions associated to the package.
	functions, exists := options.FunctionsByPackage[n.Package]
	if !exists {
		return nil
	}

	// Check if we need to detect that function.
	category, exists := functions[n.Name]
	if !exists {
		return nil
	}

	// Check if it's above the duration threshold.
	if n.DurationNS < uint64(options.DurationThreshold) {
		return nil
	}

	// Check if it's above the sample threshold.
	if n.SampleCount < options.SampleThreshold {
		return nil
	}

	ni := nodeInfo{
		Category: category,
		Node:     *n,
	}
	ni.Node.Children = nil
	return &ni
}

func (options DetectAndroidFrameOptions) OnlyCheckActiveThread() bool {
	return options.ActiveThreadOnly
}

func (options DetectAndroidFrameOptions) CheckNode(n *nodetree.Node) *nodeInfo {
	// Check if we have a list of functions associated to the package.
	functions, exists := options.FunctionsByPackage[n.Package]
	if !exists {
		return nil
	}

	// Android frame names contain the deobfuscated signature.
	// Here we strip away the argument and return types to only
	// match on the the package + function name.
	name := n.Name
	parts := strings.SplitN(name, "(", 2)
	if len(parts) > 0 {
		name = parts[0]
	}

	// Check if we need to detect that function.
	category, exists := functions[name]
	if !exists {
		return nil
	}

	// Check if it's above the duration threshold.
	if n.DurationNS < uint64(options.DurationThreshold) {
		return nil
	}

	// Check if it's above the sample threshold.
	if n.SampleCount < options.SampleThreshold {
		return nil
	}

	ni := nodeInfo{
		Category: category,
		Node:     *n,
	}
	ni.Node.Children = nil
	return &ni
}

var detectFrameJobs = map[platform.Platform][]DetectFrameOptions{
	platform.Node: {
		DetectExactFrameOptions{
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
		DetectExactFrameOptions{
			DurationThreshold: 100 * time.Millisecond,
			FunctionsByPackage: map[string]map[string]Category{
				"": {
					"addSourceContext":         SourceContext,
					"addSourceContextToFrames": SourceContext,
				},
			},
		},
	},
	platform.Cocoa: {
		DetectExactFrameOptions{
			ActiveThreadOnly:  true,
			DurationThreshold: 16 * time.Millisecond,
			SampleThreshold:   4,
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
					"+[NSJSONSerialization JSONObjectWithStream:options:error:]":                            JSONDecode,
					"+[NSJSONSerialization writeJSONObject:toStream:options:error:]":                        JSONEncode,
					"+[NSRegularExpression regularExpressionWithPattern:options:error:]":                    Regex,
					"-[NSRegularExpression initWithPattern:options:error:]":                                 Regex,
					"-[NSRegularExpression(NSMatching) enumerateMatchesInString:options:range:usingBlock:]": Regex,
					"Regex.firstMatch(in: String)":                                                          Regex,
					"Regex.wholeMatch(in: String)":                                                          Regex,
					"Regex.prefixMatch(in: String)":                                                         Regex,
					"+[NSURLConnection sendSynchronousRequest:returningResponse:error:]":                    HTTP,
					"-[NSData(NSData) initWithContentsOfMappedFile:]":                                       FileRead,
					"-[NSData(NSData) initWithContentsOfURL:]":                                              FileRead,
					"-[NSData(NSData) initWithContentsOfURL:options:maxLength:error:]":                      FileRead,
					"-[NSData(NSData) writeToFile:atomically:]":                                             FileWrite,
					"-[NSData(NSData) writeToFile:atomically:error:]":                                       FileWrite,
					"-[NSData(NSData) writeToFile:options:error:]":                                          FileWrite,
					"-[NSData(NSData) writeToURL:atomically:]":                                              FileWrite,
					"-[NSData(NSData) writeToURL:options:error:]":                                           FileWrite,
					"-[NSFileManager contentsAtPath:]":                                                      FileRead,
					"-[NSFileManager createFileAtPath:contents:attributes:]":                                FileWrite,
					"-[NSISEngine performModifications:withUnsatisfiableConstraintsHandler:]":               ViewLayout,
					"@nonobjc NSData.init(contentsOf: URL, options: NSDataReadingOptions)":                  FileRead,
					"Data.init(contentsOf: __shared URL, options: NSDataReadingOptions)":                    FileRead,
					"JSONDecoder.decode<A>(_: A.Type, from: Any)":                                           JSONDecode,
					"JSONDecoder.decode<A>(_: A.Type, from: Data)":                                          JSONDecode,
					"JSONDecoder.decode<A>(_: A.Type, jsonData: Data, logErrors: Bool)":                     JSONDecode,
					"JSONEncoder.encode<A>(A)":                                                              JSONEncode,
					"NSFileManager.contents(atURL: URL)":                                                    FileRead,
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
			},
		},
	},
	platform.Android: {
		DetectAndroidFrameOptions{
			ActiveThreadOnly:  true,
			DurationThreshold: 40 * time.Millisecond,
			FunctionsByPackage: map[string]map[string]Category{
				"com.google.gson": {
					"com.google.gson.Gson.fromJson":   JSONDecode,
					"com.google.gson.Gson.toJson":     JSONEncode,
					"com.google.gson.Gson.toJsonTree": JSONEncode,
				},
				"org.json": {
					"org.json.JSONArray.get":        JSONDecode,
					"org.json.JSONArray.opt":        JSONDecode,
					"org.json.JSONArray.writeTo":    JSONEncode,
					"org.json.JSONObject.checkName": JSONDecode,
					"org.json.JSONObject.get":       JSONDecode,
					"org.json.JSONObject.opt":       JSONDecode,
					"org.json.JSONObject.put":       JSONEncode,
					"org.json.JSONObject.putOpt":    JSONEncode,
					"org.json.JSONObject.remove":    JSONEncode,
					"org.json.JSONObject.writeTo":   JSONEncode,
				},
				"android.content.res": {
					"android.content.res.AssetManager.open":   FileRead,
					"android.content.res.AssetManager.openFd": FileRead,
				},
				"java.io": {
					"java.io.File.canExecute":             FileRead,
					"java.io.File.canRead":                FileRead,
					"java.io.File.canWrite":               FileRead,
					"java.io.File.createNewFile":          FileWrite,
					"java.io.File.createTempFile":         FileWrite,
					"java.io.File.delete":                 FileWrite,
					"java.io.File.exists":                 FileRead,
					"java.io.File.length":                 FileRead,
					"java.io.File.mkdir":                  FileWrite,
					"java.io.File.mkdirs":                 FileWrite,
					"java.io.File.renameTo":               FileWrite,
					"java.io.FileInputStream.open":        FileRead,
					"java.io.FileInputStream.read":        FileRead,
					"java.io.FileOutputStream.open":       FileRead,
					"java.io.FileOutputStream.write":      FileWrite,
					"java.io.RandomAccessFile.readBytes":  FileRead,
					"java.io.RandomAccessFile.writeBytes": FileWrite,
				},
				"okio": {
					"okio.Buffer.read":     FileRead,
					"okio.Buffer.readByte": FileRead,
					"okio.Buffer.write":    FileWrite,
					"okio.Buffer.writeAll": FileWrite,
				},
				"android.graphics": {
					"android.graphics.BitmapFactory.decodeByteArray":      ImageDecode,
					"android.graphics.BitmapFactory.decodeFile":           ImageDecode,
					"android.graphics.BitmapFactory.decodeFileDescriptor": ImageDecode,
					"android.graphics.BitmapFactory.decodeStream":         ImageDecode,
				},
				"android.database.sqlite": {
					"android.database.sqlite.SQLiteDatabase.insertWithOnConflict": SQL,
					"android.database.sqlite.SQLiteDatabase.open":                 SQL,
					"android.database.sqlite.SQLiteDatabase.query":                SQL,
					"android.database.sqlite.SQLiteDatabase.rawQueryWithFactory":  SQL,
					"android.database.sqlite.SQLiteStatement.execute":             SQL,
					"android.database.sqlite.SQLiteStatement.executeInsert":       SQL,
					"android.database.sqlite.SQLiteStatement.executeUpdateDelete": SQL,
					"android.database.sqlite.SQLiteStatement.simpleQueryForLong":  SQL,
				},
				"androidx.room": {
					"androidx.room.RoomDatabase.query": SQL,
				},
				"java.util.zip": {
					"java.util.zip.Deflater.deflate":           Compression,
					"java.util.zip.Deflater.deflateBytes":      Compression,
					"java.util.zip.DeflaterOutputStream.write": Compression,
					"java.util.zip.GZIPInputStream.read":       Compression,
					"java.util.zip.GZIPOutputStream.write":     Compression,
					"java.util.zip.Inflater.inflate":           Compression,
					"java.util.zip.Inflater.inflateBytes":      Compression,
				},
				"java.util": {
					"java.util.Base64$Decoder.decode":  Base64Decode,
					"java.util.Base64$Decoder.decode0": Base64Decode,
				},
				"java.util.regex": {
					"java.util.regex.Matcher.matches":   Regex,
					"java.util.regex.Matcher.find":      Regex,
					"java.util.regex.Matcher.lookingAt": Regex,
				},
				"kotlinx.coroutines": {
					"kotlinx.coroutines.AwaitAll.await":                 ThreadWait,
					"kotlinx.coroutines.AwaitKt.awaitAll":               ThreadWait,
					"kotlinx.coroutines.BlockingCoroutine.joinBlocking": ThreadWait,
					"kotlinx.coroutines.JobSupport.join":                ThreadWait,
					"kotlinx.coroutines.JobSupport.joinSuspend":         ThreadWait,
				},
			},
		},
	},
}

// DetectFrames detects occurrence of an issue based by matching frames of the profile on a list of frames.
func detectFrame(
	p profile.Profile,
	callTreesPerThreadID map[uint64][]*nodetree.Node,
	options DetectFrameOptions,
	occurrences *[]*Occurrence,
) {
	// List nodes matching criteria
	nodes := make(map[nodeKey]nodeInfo)
	if options.OnlyCheckActiveThread() {
		callTrees, exists := callTreesPerThreadID[p.Transaction().ActiveThreadID]
		if !exists {
			return
		}
		for _, root := range callTrees {
			detectFrameInCallTree(root, options, nodes)
		}
	} else {
		for _, callTrees := range callTreesPerThreadID {
			for _, root := range callTrees {
				detectFrameInCallTree(root, options, nodes)
			}
		}
	}

	// Create occurrences.
	for _, n := range nodes {
		*occurrences = append(*occurrences, NewOccurrence(p, n))
	}
}

func detectFrameInCallTree(
	n *nodetree.Node,
	options DetectFrameOptions,
	nodes map[nodeKey]nodeInfo,
) {
	st := make([]frame.Frame, 0, 128)
	detectFrameInNode(n, options, nodes, &st)
}

func detectFrameInNode(
	n *nodetree.Node,
	options DetectFrameOptions,
	nodes map[nodeKey]nodeInfo,
	st *[]frame.Frame,
) *nodeInfo {
	*st = append(*st, n.ToFrame())
	defer func() {
		*st = (*st)[:len(*st)-1]
	}()
	var issueDetected bool
	for _, c := range n.Children {
		if ni := detectFrameInNode(c, options, nodes, st); ni != nil {
			issueDetected = true
		}
	}
	if issueDetected {
		return nil
	}
	ni := options.CheckNode(n)
	if ni != nil {
		nk := nodeKey{Package: ni.Node.Package, Function: ni.Node.Name}
		if _, exists := nodes[nk]; !exists {
			ni.StackTrace = make([]frame.Frame, len(*st))
			copy(ni.StackTrace, *st)
			nodes[nk] = *ni
		}
	}
	return ni
}
