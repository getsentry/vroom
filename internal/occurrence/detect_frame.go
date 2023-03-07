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
	ThreadWait       Category = "thread_wait"
	ViewInflation    Category = "view_inflation"
	ViewLayout       Category = "view_layout"
	ViewRender       Category = "view_render"
	ViewUpdate       Category = "view_update"
	XPC              Category = "xpc"

	// MinimumSampleCount is the minimum number of samples in which we need to
	// detect the frame in order to create an occurrence.
	minimumSampleCount = 4
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
				},
			},
		},
		platform.Android: {
			{
				ActiveThreadOnly:  true,
				DurationThreshold: 16 * time.Millisecond,
				FunctionsByPackage: map[string]map[string]Category{
					"android.util": {
						"android.util.Base64.decode(String; int): byte[]":           Base64Decode,
						"android.util.Base64.decode(byte[]; int): byte[]":           Base64Decode,
						"android.util.Base64.decode(byte[]; int; int; int): byte[]": Base64Decode,
						"android.util.Base64.encode(byte[]; int): byte[]":           Base64Encode,
						"android.util.Base64.encode(byte[]; int; int; int): byte[]": Base64Encode,
						"android.util.Base64.encodeToString(byte[]; int): String":   Base64Encode,
						"android.util.JsonReader.hasNext(): boolean":                JSONDecode,
						"android.util.JsonReader.nextBoolean(): boolean":            JSONDecode,
						"android.util.JsonReader.nextDouble(): double":              JSONDecode,
						"android.util.JsonReader.nextInt(): int":                    JSONDecode,
						"android.util.JsonReader.nextLong(): long":                  JSONDecode,
						"android.util.JsonReader.nextName(): String":                JSONDecode,
						"android.util.JsonReader.nextString(): String":              JSONDecode,
						"android.util.JsonReader.peek(): JsonToken":                 JSONDecode,
						"android.util.JsonReader.skipValue()":                       JSONDecode,
						"android.util.JsonWriter.beginArray(): JsonWriter":          JSONEncode,
						"android.util.JsonWriter.beginObject(): JsonWriter":         JSONEncode,
						"android.util.JsonWriter.close()":                           JSONEncode,
						"android.util.JsonWriter.endArray(): JsonWriter":            JSONEncode,
						"android.util.JsonWriter.endObject(): JsonWriter":           JSONEncode,
						"android.util.JsonWriter.flush()":                           JSONEncode,
						"android.util.JsonWriter.name(String): JsonWriter":          JSONEncode,
						"android.util.JsonWriter.value(Number): JsonWriter":         JSONEncode,
						"android.util.JsonWriter.value(String): JsonWriter":         JSONEncode,
						"android.util.JsonWriter.value(boolean): JsonWriter":        JSONEncode,
						"android.util.JsonWriter.value(long): JsonWriter":           JSONEncode,
					},
					"com.google.json": {
						"com.google.gson.Gson.fromJson(JsonElement; Class): Object":    JSONDecode,
						"com.google.gson.Gson.fromJson(JsonElement; Type): Object":     JSONDecode,
						"com.google.gson.Gson.fromJson(JsonReader; Type): Object":      JSONDecode,
						"com.google.gson.Gson.fromJson(JsonReader; TypeToken): Object": JSONDecode,
						"com.google.gson.Gson.fromJson(Reader; Class): Object":         JSONDecode,
						"com.google.gson.Gson.fromJson(Reader; Type): Object":          JSONDecode,
						"com.google.gson.Gson.fromJson(Reader; TypeToken): Object":     JSONDecode,
						"com.google.gson.Gson.fromJson(String; Class): Object":         JSONDecode,
						"com.google.gson.Gson.fromJson(String; Type): Object":          JSONDecode,
						"com.google.gson.Gson.toJson(JsonElement): String":             JSONEncode,
						"com.google.gson.Gson.toJson(JsonElement; Appendable)":         JSONEncode,
						"com.google.gson.Gson.toJson(JsonElement; JsonWriter)":         JSONEncode,
						"com.google.gson.Gson.toJson(Object): String":                  JSONEncode,
						"com.google.gson.Gson.toJson(Object; Appendable)":              JSONEncode,
						"com.google.gson.Gson.toJson(Object; Type): String":            JSONEncode,
						"com.google.gson.Gson.toJson(Object; Type; Appendable)":        JSONEncode,
						"com.google.gson.Gson.toJson(Object; Type; JsonWriter)":        JSONEncode,
						"com.google.gson.Gson.toJsonTree(Object): JsonElement":         JSONEncode,
						"com.google.gson.Gson.toJsonTree(Object; Type): JsonElement":   JSONEncode,
					},
					"org.json": {
						"org.json.JSONArray.get(int): Object":                    JSONDecode,
						"org.json.JSONArray.opt(int): Object":                    JSONDecode,
						"org.json.JSONArray.writeTo(JSONStringer)":               JSONEncode,
						"org.json.JSONObject.checkName(String): String":          JSONDecode,
						"org.json.JSONObject.get(String): Object":                JSONDecode,
						"org.json.JSONObject.opt(String): Object":                JSONDecode,
						"org.json.JSONObject.put(String; Object): JSONObject":    JSONEncode,
						"org.json.JSONObject.putOpt(String; Object): JSONObject": JSONEncode,
						"org.json.JSONObject.remove(String): Object":             JSONEncode,
						"org.json.JSONObject.writeTo(JSONStringer)":              JSONEncode,
					},
					"android.content.res": {
						"android.content.res.AssetManager.open(String; int): InputStream":      FileRead,
						"android.content.res.AssetManager.openFd(String): AssetFileDescriptor": FileRead,
					},
					"java.io": {
						"java.io.File.canExecute(): boolean":                        FileRead,
						"java.io.File.canRead(): boolean":                           FileRead,
						"java.io.File.canWrite(): boolean":                          FileRead,
						"java.io.File.createNewFile(): boolean":                     FileWrite,
						"java.io.File.createTempFile(String; String): File":         FileWrite,
						"java.io.File.createTempFile(String; String; File): File":   FileWrite,
						"java.io.File.delete(): boolean":                            FileWrite,
						"java.io.File.exists(): boolean":                            FileRead,
						"java.io.File.length(): long":                               FileRead,
						"java.io.File.mkdir(): boolean":                             FileWrite,
						"java.io.File.mkdirs(): boolean":                            FileWrite,
						"java.io.File.mkdirs(boolean): boolean":                     FileWrite,
						"java.io.File.renameTo(File): boolean":                      FileWrite,
						"java.io.FileInputStream.open(String)":                      FileRead,
						"java.io.FileInputStream.read(byte[]; int; int): int":       FileRead,
						"java.io.FileOutputStream.open(String; boolean)":            FileRead,
						"java.io.FileOutputStream.write(byte[]; int; int)":          FileWrite,
						"java.io.RandomAccessFile.readBytes(byte[]; int; int): int": FileRead,
						"java.io.RandomAccessFile.writeBytes(byte[]; int; int)":     FileWrite,
					},
					"okio": {
						"okio.Buffer.read(Buffer; long): long":              FileRead,
						"okio.Buffer.read(ByteBuffer): int":                 FileRead,
						"okio.Buffer.read(byte[]; int; int): int":           FileRead,
						"okio.Buffer.readByte(): byte":                      FileRead,
						"okio.Buffer.write(Buffer; long)":                   FileWrite,
						"okio.Buffer.write(ByteString): Buffer":             FileWrite,
						"okio.Buffer.write(ByteString): BufferedSink":       FileWrite,
						"okio.Buffer.write(byte[]): BufferedSink":           FileWrite,
						"okio.Buffer.write(byte[]; int; int)":               FileWrite,
						"okio.Buffer.write(byte[]; int; int): Buffer":       FileWrite,
						"okio.Buffer.write(byte[]; int; int): BufferedSink": FileWrite,
						"okio.Buffer.writeAll(Source): long":                FileWrite,
					},
					"android.graphics": {
						"android.graphics.BitmapFactory.decodeByteArray(byte[]; int; int; BitmapFactory$Options): Bitmap":          ImageDecode,
						"android.graphics.BitmapFactory.decodeFile(String; BitmapFactory$Options): Bitmap":                         ImageDecode,
						"android.graphics.BitmapFactory.decodeFileDescriptor(FileDescriptor; Rect; BitmapFactory$Options): Bitmap": ImageDecode,
						"android.graphics.BitmapFactory.decodeStream(InputStream; Rect; BitmapFactory$Options): Bitmap":            ImageDecode,
						"android.graphics.BitmapFactory.decodeStream(InputStream; Rect; BitmapFactory$Options; boolean): Bitmap":   ImageDecode,
					},
					"android.database.sqlite": {
						"android.database.sqlite.SQLiteDatabase.insertWithOnConflict(String; String; ContentValues; int): long":                                          SQL,
						"android.database.sqlite.SQLiteDatabase.open()":                                                                                                  SQL,
						"android.database.sqlite.SQLiteDatabase.open(String; String)":                                                                                    SQL,
						"android.database.sqlite.SQLiteDatabase.query(boolean; String; String[]; String; String[]; String; String; String; String): Cursor":              SQL,
						"android.database.sqlite.SQLiteDatabase.rawQueryWithFactory(SQLiteDatabase$CursorFactory; String; String[]; String; CancellationSignal): Cursor": SQL,
						"android.database.sqlite.SQLiteStatement.execute()":                                                                                              SQL,
						"android.database.sqlite.SQLiteStatement.executeInsert(): long":                                                                                  SQL,
						"android.database.sqlite.SQLiteStatement.executeUpdateDelete(): int":                                                                             SQL,
						"android.database.sqlite.SQLiteStatement.simpleQueryForLong(): long":                                                                             SQL,
					},
					"androidx.room": {
						"androidx.room.RoomDatabase.query(SupportSQLiteQuery): Cursor":                     SQL,
						"androidx.room.RoomDatabase.query(SupportSQLiteQuery; CancellationSignal): Cursor": SQL,
					},
					"java.util.zip": {
						"java.util.zip.Deflater.deflate(byte[]; int; int; int): int":            Compression,
						"java.util.zip.Deflater.deflateBytes(long; byte[]; int; int; int): int": Compression,
						"java.util.zip.DeflaterOutputStream.write(byte[]; int; int)":            Compression,
						"java.util.zip.GZIPInputStream.read(byte[]; int; int): int":             Compression,
						"java.util.zip.GZIPOutputStream.write(byte[]; int; int)":                Compression,
						"java.util.zip.Inflater.inflate(byte[]; int; int): int":                 Compression,
						"java.util.zip.Inflater.inflateBytes(long; byte[]; int; int): int":      Compression,
					},
					"java.util": {
						"java.util.Base64$Decoder.decode(String): byte[]":                 Base64Decode,
						"java.util.Base64$Decoder.decode(byte[]): byte[]":                 Base64Decode,
						"java.util.Base64$Decoder.decode0(byte[]; int; int; byte[]): int": Base64Decode,
					},
					/*
						"kotlinx.coroutines": {
							"kotlinx.coroutines.AwaitAll.await(d): Object":                    ThreadWait,
							"kotlinx.coroutines.AwaitKt.awaitAll(Collection; d): Object":      ThreadWait,
							"kotlinx.coroutines.BlockingCoroutine.joinBlocking(): Object":     ThreadWait,
							"kotlinx.coroutines.JobSupport.join(Continuation): Object":        ThreadWait,
							"kotlinx.coroutines.JobSupport.joinSuspend(Continuation): Object": ThreadWait,
						},
					*/
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
	*stackTrace = append(*stackTrace, n.ToFrame())
	detectNode(n, options, nodes, stackTrace)
	for _, c := range n.Children {
		newStackTrace := *stackTrace
		detectFrameInCallTree(c, options, nodes, &newStackTrace)
	}
}

func detectNode(n *nodetree.Node, options DetectExactFrameOptions, nodes map[nodeKey]nodeInfo, stackTrace *[]frame.Frame) {
	// Check if we have a list of functions associated to the package.
	functions, exists := options.FunctionsByPackage[n.Package]
	if !exists {
		return
	}

	// Check if we need to detect that function.
	category, exists := functions[n.Name]
	if !exists {
		return
	}

	// Check if it's above the duration threshold.
	if n.DurationNS < uint64(options.DurationThreshold) {
		return
	}

	// Check if it's above the sample threshold.
	if n.SampleCount < minimumSampleCount {
		return
	}

	// Check if we've already detected an occurrence on it.
	nk := nodeKey{Package: n.Package, Function: n.Name}
	_, exists = nodes[nk]
	if exists {
		return
	}

	// Add it to the list of nodes detected.
	nodes[nk] = nodeInfo{
		Category:   category,
		Node:       n,
		StackTrace: *stackTrace,
	}
}
