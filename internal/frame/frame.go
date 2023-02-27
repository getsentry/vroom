package frame

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"path"
	"strings"

	"github.com/getsentry/vroom/internal/packageutil"
)

type (
	Frame struct {
		Column          uint32 `json:"colno,omitempty"`
		File            string `json:"filename,omitempty"`
		Function        string `json:"function,omitempty"`
		InApp           *bool  `json:"in_app"`
		InstructionAddr string `json:"instruction_addr,omitempty"`
		Lang            string `json:"lang,omitempty"`
		Line            uint32 `json:"lineno,omitempty"`
		Module          string `json:"module,omitempty"`
		Package         string `json:"package,omitempty"`
		Path            string `json:"abs_path,omitempty"`
		Status          string `json:"status,omitempty"`
		SymAddr         string `json:"sym_addr,omitempty"`
		Symbol          string `json:"symbol,omitempty"`
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
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%s:%d:%s", f.File, f.Function, f.Line, f.InstructionAddr)))
	return hex.EncodeToString(hash[:])
}

func (f Frame) PackageBaseName() string {
	if f.Module != "" {
		return f.Module
	} else if f.Package != "" {
		return path.Base(f.Package)
	}
	return ""
}

func (f Frame) WriteToHash(h hash.Hash) {
	var s string
	if f.Package != "" {
		s = f.PackageBaseName()
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
}

func (f Frame) IsInline() bool {
	return f.Status == "symbolicated" && f.SymAddr == ""
}

func (f Frame) IsNodeApplicationFrame() bool {
	return strings.Contains(f.Path, "node_modules")
}

func (f Frame) IsCocoaApplicationFrame() bool {
	return packageutil.IsCocoaApplicationPackage(f.Package)
}

func (f Frame) IsRustApplicationFrame() bool {
	return packageutil.IsRustApplicationPackage(f.Package)
}

func (f Frame) IsPythonApplicationFrame() bool {
	if strings.Contains(f.Path, "/site-packages/") ||
		strings.Contains(f.Path, "/dist-packages/") ||
		strings.Contains(f.Path, "\\site-packages\\") ||
		strings.Contains(f.Path, "\\dist-packages\\") {
		return false
	}

	module := strings.SplitN(f.Module, ".", 2)
	_, ok := pythonStdlib[module[0]]
	return !ok
}
