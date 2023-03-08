package debugmeta

type (
	Features struct {
		HasDebugInfo  bool `json:"has_debug_info"`
		HasSources    bool `json:"has_sources"`
		HasSymbols    bool `json:"has_symbols"`
		HasUnwindInfo bool `json:"has_unwind_info"`
	}

	Image struct {
		Arch        string    `json:"arch,omitempty"`
		CodeFile    string    `json:"code_file,omitempty"`
		DebugID     string    `json:"debug_id,omitempty"`
		DebugStatus string    `json:"debug_status,omitempty"`
		Features    *Features `json:"features,omitempty"`
		ImageAddr   string    `json:"image_addr,omitempty"`
		ImageSize   uint64    `json:"image_size,omitempty"`
		ImageVMAddr string    `json:"image_vmaddr,omitempty"`
		Type        string    `json:"type,omitempty"`
	}

	DebugMeta struct {
		Images []Image `json:"images,omitempty"`
	}
)
