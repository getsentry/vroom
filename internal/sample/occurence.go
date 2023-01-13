package sample

import (
	"github.com/getsentry/vroom/internal/occurrence"
	"github.com/getsentry/vroom/internal/platform"
)

type (
	DetectExactFrameMetadata struct {
		ActiveThreadOnly bool
		Frames           map[string]struct{}
		IssueTitle       string
	}
)

var (
	detectExactFrames = map[platform.Platform][]DetectExactFrameMetadata{
		platform.Node: []DetectExactFrameMetadata{
			DetectExactFrameMetadata{
				ActiveThreadOnly: true,
				Frames: map[string]struct{}{
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
				IssueTitle: "Synchronous function called on main thread",
			},
		},
		platform.Cocoa: []DetectExactFrameMetadata{
			DetectExactFrameMetadata{
				ActiveThreadOnly: true,
				Frames: map[string]struct{}{
					"+[MLModel modelWithContentsOfURL:configuration:error:]":                  struct{}{},
					"+[NSJSONSerialization JSONObjectWithStream:options:error:]":              struct{}{},
					"+[NSJSONSerialization writeJSONObject:toStream:options:error:]":          struct{}{},
					"+[NSURLConnection sendSynchronousRequest:returningResponse:error:]":      struct{}{},
					"-[MLNeuralNetworkEngine predictionFromFeatures:options:error:]":          struct{}{},
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
					"-[NSManagedObjectContext countForFetchRequest:error:]":                   struct{}{},
					"-[NSManagedObjectContext executeFetchRequest:error:]":                    struct{}{},
					"-[NSManagedObjectContext executeRequest:error:]":                         struct{}{},
					"-[NSManagedObjectContext mergeChangesFromContextDidSaveNotification:]":   struct{}{},
					"-[NSManagedObjectContext obtainPermanentIDsForObjects:error:]":           struct{}{},
					"-[NSManagedObjectContext performBlockAndWait:]":                          struct{}{},
					"-[NSManagedObjectContext save:]":                                         struct{}{},
					"-[UINib instantiateWithOwner:options:]":                                  struct{}{},
					"@nonobjc NSData.init(contentsOf: URL, options: NSDataReadingOptions)":    struct{}{},
					"CFReadStreamRead":                         struct{}{},
					"CFURLConnectionSendSynchronousRequest":    struct{}{},
					"CFURLCreateData":                          struct{}{},
					"CFURLCreateDataAndPropertiesFromResource": struct{}{},
					"CFURLWriteDataAndPropertiesToResource":    struct{}{},
					"CFWriteStreamWrite":                       struct{}{},
					"Data.init(contentsOf: __shared URL, options: NSDataReadingOptions)": struct{}{},
					"DecodeImageData":   struct{}{},
					"DecodeImageStream": struct{}{},
					"GIFReadPlugin::DoDecodeImageData(IIOImageReadSession*, GlobalGIFInfo*, ReadPluginData const&, GIFPluginData const&, unsigned char*, unsigned long, std::__1::shared_ptr<GIFBufferInfo>, long*)": struct{}{},
					"IIOImageProviderInfo::CopyImageBlockSetWithOptions(void*, CGImageProvider*, CGRect, CGSize, __CFDictionary const*)":                                                                             struct{}{},
					"JSONDecoder.decode<A>(_: A.Type, from: Any)":                       struct{}{},
					"JSONDecoder.decode<A>(_: A.Type, jsonData: Data, logErrors: Bool)": struct{}{},
					"JSONEncoder.encode<A>(A)":                                          struct{}{},
					"LZWDecode":                                                         struct{}{},
					"NSFileManager.contents(atURL: URL)":                                struct{}{},
					"NSManagedObjectContext.count<A>(for: NSFetchRequest<A>)":           struct{}{},
					"NSManagedObjectContext.fetch<A>(NSFetchRequest<A>)":                struct{}{},
					"NSManagedObjectContext.perform<A>(schedule: NSManagedObjectContext.ScheduledTaskType, _: ())": struct{}{},
					"NeXTDecode": struct{}{},
					"PNGReadPlugin::DecodeFrameStandard(IIOImageReadSession*, ReadPluginData const&, PNGPluginData const&, IIODecodeFrameParams&)": struct{}{},
					"VP8Decode":                       struct{}{},
					"VP8DecodeMB":                     struct{}{},
					"WebPDecode":                      struct{}{},
					"__JSONDecoder.decode<A>(A.Type)": struct{}{},
					"__JSONEncoder.encode<A>(A)":      struct{}{},
					"__fread":                         struct{}{},
					"applejpeg_decode_image_all":      struct{}{},
					"fread":                           struct{}{},
					"jpeg_huff_decode":                struct{}{},
					"specialized JSONDecoder.decode<A>(_: A.Type, jsonData: Data, logErrors: Bool)":                   struct{}{},
					"specialized static JSONDecoder.decode<A>(_: A.Type, from: Data, dateFormatter: NSDateFormatter)": struct{}{},
					"sqlite3_blob_read":                           struct{}{},
					"sqlite3_column_blob":                         struct{}{},
					"sqlite3_column_bytes":                        struct{}{},
					"sqlite3_column_double":                       struct{}{},
					"sqlite3_column_int":                          struct{}{},
					"sqlite3_column_int64":                        struct{}{},
					"sqlite3_column_text":                         struct{}{},
					"sqlite3_column_text16":                       struct{}{},
					"sqlite3_column_value":                        struct{}{},
					"sqlite3_step":                                struct{}{},
					"xpc_connection_send_message_with_reply_sync": struct{}{},
				},
				IssueTitle: "File I/O function called on main thread",
			},
		},
	}
)

func (p *SampleProfile) Occurrences() []occurrence.Occurrence {
	var occurrences []occurrence.Occurrence
	jobs, exists := detectExactFrames[p.Platform]
	if exists {
		for _, metadata := range jobs {
			p.DetectExactFrames(metadata, occurrences)
		}
	}
	return occurrences
}

// DetectFrames detects occurrence of an issue based by matching frames of the profile on a list of frames
func (p *SampleProfile) DetectExactFrames(metadata DetectExactFrameMetadata, occurrences []occurrence.Occurrence) {
	activeThreadID := p.Transaction.ActiveThreadID
	for _, sample := range p.Trace.Samples {
		if metadata.ActiveThreadOnly && sample.ThreadID != activeThreadID {
			continue
		}
		stack := p.Trace.Stacks[sample.StackID]
		for _, frameID := range stack {
			f := p.Trace.Frames[frameID]
			_, exists := metadata.Frames[f.Function]
			if !exists {
				continue
			}
			occurrences = append(occurrences, occurrence.Occurrence{
				Event: occurrence.Event{
					ID:        p.EventID,
					Platform:  p.Platform,
					ProjectID: p.ProjectID,
					Received:  p.Received,
					Timestamp: p.Timestamp,
					Tags:      make(map[string]string),
				},
				EvidenceData: map[string]interface{}{
					"transaction_id": p.Transaction.ID,
				},
				EvidenceDisplay: []occurrence.Evidence{
					occurrence.Evidence{
						Name:      "Suspect function",
						Value:     f.Function,
						Important: true,
					},
					occurrence.Evidence{
						Name:  "Package",
						Value: f.PackageBaseName(),
					},
				},
				Stacktrace: occurrence.Stacktrace{
					Frames: p.Trace.CollectFrames(sample.StackID),
				},
				IssueTitle: metadata.IssueTitle,
				Subtitle:   p.Transaction.Name,
				Type:       occurrence.ProfileBlockedThreadType,
			})
		}
	}
}
