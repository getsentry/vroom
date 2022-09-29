package metadata

type Metadata struct {
	AndroidAPILevel      uint32  `json:"android_api_level"`
	DeviceClassification string  `json:"device_classification"`
	DeviceLocale         string  `json:"device_locale"`
	DeviceManufacturer   string  `json:"device_manufacturer"`
	DeviceModel          string  `json:"device_model"`
	DeviceOsBuildNumber  string  `json:"device_os_build_number"`
	DeviceOsName         string  `json:"device_os_name"`
	DeviceOsVersion      string  `json:"device_os_version"`
	ID                   string  `json:"id"`
	ProjectID            string  `json:"project_id"`
	Timestamp            int64   `json:"timestamp"`
	TraceDurationMs      float64 `json:"trace_duration_ms"`
	TransactionID        string  `json:"transaction_id"`
	TransactionName      string  `json:"transaction_name"`
	VersionCode          string  `json:"version_code"`
	VersionName          string  `json:"version_name"`
}
