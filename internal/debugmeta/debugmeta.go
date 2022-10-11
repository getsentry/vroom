package debugmeta

type (
	Features struct {
		HasDebugInfo  bool `json:"has_debug_info"`
		HasSources    bool `json:"has_sources"`
		HasSymbols    bool `json:"has_symbols"`
		HasUnwindInfo bool `json:"has_unwind_info"`
	}

	Image struct {
		Arch        string   `json:"arch"`
		CodeFile    string   `json:"code_file"`
		DebugID     string   `json:"debug_id"`
		DebugStatus string   `json:"debug_status"`
		Features    Features `json:"features"`
		ImageAddr   string   `json:"image_addr"`
		ImageSize   uint64   `json:"image_size"`
		ImageVMAddr string   `json:"image_vmaddr"`
		Type        string   `json:"type"`
	}

	DebugMeta struct {
		Images []Image `json:"images,omitempty"`
	}
)
