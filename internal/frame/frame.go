package frame

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/getsentry/vroom/internal/packageutil"
	"github.com/getsentry/vroom/internal/platform"
)

var (
	windowsPathRegex                  = regexp.MustCompile(`(?i)^([a-z]:\\|\\\\)`)
	packageExtensionRegex             = regexp.MustCompile(`\.(dylib|so|a|dll|exe)$`)
	javascriptSystemPackagePathRegexp = regexp.MustCompile(`node_modules|^(@moz-extension|chrome-extension)`)
	cocoaSystemPackage                = map[string]struct{}{
		"Sentry": {},
		"hermes": {},
	}

	ErrFrameNotFound = errors.New("Unable to find matching frame")
)

type (
	Frame struct {
		Column          uint32            `json:"colno,omitempty"`
		Data            Data              `json:"data"`
		File            string            `json:"filename,omitempty"`
		Function        string            `json:"function,omitempty"`
		InApp           *bool             `json:"in_app"`
		InstructionAddr string            `json:"instruction_addr,omitempty"`
		Lang            string            `json:"lang,omitempty"`
		Line            uint32            `json:"lineno,omitempty"`
		MethodID        uint64            `json:"-"`
		Module          string            `json:"module,omitempty"`
		Package         string            `json:"package,omitempty"`
		Path            string            `json:"abs_path,omitempty"`
		Status          string            `json:"status,omitempty"`
		SymAddr         string            `json:"sym_addr,omitempty"`
		Symbol          string            `json:"symbol,omitempty"`
		Platform        platform.Platform `json:"platform,omitempty"`
		// IsReactNativeFrame is not exported as json since we only
		// need it at runtime to distinguish browser/node js frame
		// from ReactNative js frame.
		IsReactNative bool `json:"-"`
	}

	Data struct {
		DeobfuscationStatus string `json:"deobfuscation_status,omitempty"`
		SymbolicatorStatus  string `json:"symbolicator_status,omitempty"`
		JsSymbolicated      *bool  `json:"symbolicated,omitempty"`
	}
)

// IsMain returns true if the function is considered the main function.
// It also returns an offset indicate if we need to keep the previous frame or not.
// This only works for cocoa profiles.
func (f Frame) IsMain() (bool, int) {
	if f.Status != "symbolicated" {
		return false, 0
	}
	switch f.Function {
	case "main":
		return true, 0
	case "UIApplicationMain":
		return true, -1
	}
	return false, 0
}

func (f Frame) ID() string {
	// When we have a symbolicated frame we can't rely on symbol_address
	// to uniquely identify a frame since the following might happen:
	//
	// frame 1 has: sym_addr: 1, file: a.rs, line 2
	// frame 2 has: sym_addr: 1, file: a.rs, line: 4
	// because they have the same sym addr the second frame is reusing the first one,
	// and gets the wrong line number
	//
	// Also, when a frame is symbolicated but is missing the symbol_address
	// we know we're dealing with inlines, but we can't rely on instruction_address
	// neither as the inlines are all using the same one. If we were to return this
	// address in speedscope we would only generate a new frame for the parent one
	// and for the inlines we would show the same information of the parents instead
	// of their own
	//
	// As a solution here we use the following hash function that guarantees uniqueness
	// when all the information required is available
	hash := md5.Sum(
		[]byte(fmt.Sprintf("%s:%s:%d:%s", f.File, f.Function, f.Line, f.InstructionAddr)),
	)
	return hex.EncodeToString(hash[:])
}

// Taken from https://github.com/getsentry/sentry/blob/1c9cf8bd92f65e933a407d8ee37fb90997c1c76c/static/app/components/events/interfaces/frame/utils.tsx#L8-L12
// This takes a frame's package and formats it in such a way that is suitable for displaying/aggregation.
func trimPackage(pkg string) string {
	separator := "/"
	if windowsPathRegex.Match([]byte(pkg)) {
		separator = "\\"
	}

	pieces := strings.Split(pkg, separator)

	filename := pkg

	if len(pieces) >= 1 {
		filename = pieces[len(pieces)-1]
	}

	if len(pieces) >= 2 && filename == "" {
		filename = pieces[len(pieces)-2]
	}

	if filename == "" {
		filename = pkg
	}

	filename = packageExtensionRegex.ReplaceAllString(filename, "")

	return filename
}

func (f Frame) ModuleOrPackage() string {
	if f.Module != "" {
		return f.Module
	} else if f.Package != "" {
		return trimPackage(f.Package)
	}
	return ""
}

func (f Frame) WriteToHash(h hash.Hash) {
	var s string
	if f.Module != "" {
		s = f.Module
	} else if f.Package != "" {
		s = trimPackage(f.Package)
	} else if f.File != "" {
		s = f.File
	} else {
		s = "-"
	}
	h.Write([]byte(s))
	if f.Function != "" {
		s = f.Function
	} else {
		s = "-"
	}
	h.Write([]byte(s))
	// Important for native platforms to distinguish unknown frames
	if f.InstructionAddr != "" {
		h.Write([]byte(f.InstructionAddr))
	}
}

func (f Frame) IsInline() bool {
	return f.Status == "symbolicated" && f.SymAddr == ""
}

func (f Frame) IsNodeApplicationFrame() bool {
	return !strings.HasPrefix(f.Path, "node:") && !strings.Contains(f.Path, "node_modules")
}

func (f Frame) IsJavaScriptApplicationFrame() bool {
	if strings.HasPrefix(f.Function, "[") {
		return false
	}

	if len(f.Path) == 0 {
		return true
	}

	return !javascriptSystemPackagePathRegexp.MatchString(f.Path)
}

func (f Frame) IsCocoaApplicationFrame() bool {
	isMain, _ := f.IsMain()
	if isMain {
		// the main frame is found in the user package but should be treated
		// as a system frame as it does not contain any user code
		return false
	}

	// Some packages are known to be system packages.
	// If we detect them, mark them as a system frame immediately.
	if _, exists := cocoaSystemPackage[f.ModuleOrPackage()]; exists {
		return false
	}

	return packageutil.IsCocoaApplicationPackage(f.Package)
}

func (f Frame) IsRustApplicationFrame() bool {
	return packageutil.IsRustApplicationPackage(f.Package)
}

func (f Frame) IsPythonApplicationFrame() bool {
	if strings.Contains(f.Path, "/site-packages/") ||
		strings.Contains(f.Path, "/dist-packages/") ||
		strings.Contains(f.Path, "\\site-packages\\") ||
		strings.Contains(f.Path, "\\dist-packages\\") ||
		strings.HasPrefix(f.Path, "/usr/local/") {
		return false
	}

	module := strings.SplitN(f.Module, ".", 2)

	// It's possible that users do not install packages
	// into one of the default paths. In this case, we
	// should try to classify the sdk as a system frame
	// at least to minimum false classification.
	if module[0] == "sentry_sdk" {
		return false
	}

	_, ok := pythonStdlib[module[0]]
	return !ok
}

func (f Frame) IsPHPApplicationFrame() bool {
	return !strings.Contains(f.Path, "/vendor/")
}

func (f Frame) Fingerprint() uint32 {
	h := fnv.New64()
	h.Write([]byte(f.ModuleOrPackage()))
	h.Write([]byte{':'})
	h.Write([]byte(f.Function))

	// casting to an uint32 here because snuba does not handle uint64 values well
	// as it is converted to a float somewhere not changing to the 32 bit hash
	// function here to preserve backwards compatibility with existing fingerprints
	// that we can cast
	return uint32(h.Sum64())
}

func defaultFormatter(f Frame) string {
	return f.Function
}

func makeJoinedNameFormatter(separator string) func(f Frame) string {
	return func(f Frame) string {
		// These platforms can additionally use the module/package name to fully
		// qualify the function name and uses `.` as the separator. So extract the
		// module/package name, and concatenate it with the function name.
		moduleOrPackage := f.ModuleOrPackage()
		if moduleOrPackage == "" {
			return f.Function
		}
		return fmt.Sprintf("%s%s%s", moduleOrPackage, separator, f.Function)
	}
}

var fullyQualifiedNameFormatters = map[platform.Platform]func(f Frame) string{
	// These platforms have the module name as a prefix of the function already.
	// So no formatting required.
	platform.Android: defaultFormatter,
	platform.Java:    defaultFormatter,
	platform.PHP:     defaultFormatter,

	// The package name for these platforms varies depending on how it's compiled.
	// So we just use the function name.
	platform.Cocoa: defaultFormatter,

	// These platforms can additionally use the module/package name to fully
	// qualify the function name and uses `.` as the separator. So extract the
	// module/package name, and concatenate it with the function name.
	platform.Python: makeJoinedNameFormatter("."),
	platform.Node:   makeJoinedNameFormatter("."),
}

func (f Frame) FullyQualifiedName(p platform.Platform) string {
	formatter, ok := fullyQualifiedNameFormatters[p]
	if !ok {
		formatter = defaultFormatter
	}
	return formatter(f)
}

func (f *Frame) SetInApp(p platform.Platform) {
	// for react-native the in_app field seems to be messed up most of the times,
	// with system libraries and other frames that are clearly system frames
	// labelled as `in_app`.
	// This is likely because RN uses static libraries which are bundled into the app binary.
	// When symbolicated they are marked in_app.
	//
	// For this reason, for react-native app (p.Platform != f.Platform), we skip the f.InApp!=nil
	// check as this field would be highly unreliable, and rely on our rules instead
	if f.InApp != nil && (p == f.Platform) {
		return
	}
	var isApplication bool
	switch f.Platform {
	case platform.Node:
		isApplication = f.IsNodeApplicationFrame()
	case platform.JavaScript:
		isApplication = f.IsJavaScriptApplicationFrame()
	case platform.Cocoa:
		isApplication = f.IsCocoaApplicationFrame()
	case platform.Rust:
		isApplication = f.IsRustApplicationFrame()
	case platform.Python:
		isApplication = f.IsPythonApplicationFrame()
	case platform.PHP:
		isApplication = f.IsPHPApplicationFrame()
	}
	f.InApp = &isApplication
}

func (f *Frame) IsInApp() bool {
	if f.InApp == nil {
		return false
	}
	return *f.InApp
}

func (f *Frame) SetPlatform(p platform.Platform) {
	if f.Platform == "" {
		f.Platform = p
	}
}

func (f *Frame) SetStatus() {
	if f.Data.SymbolicatorStatus != "" {
		f.Status = f.Data.SymbolicatorStatus
	}
}

func (f *Frame) Normalize(p platform.Platform) {
	// Call order is important since SetInApp uses Status and Platform
	f.SetStatus()
	f.SetPlatform(p)
	f.SetInApp(p)
}
