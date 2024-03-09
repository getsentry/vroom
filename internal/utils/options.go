package utils

type Options struct {
	FunctionMetricsIngestionEnabled       bool  `json:"profiling.generic_metrics.functions_ingestion.enabled"`
	FunctionMetricsIngestionAllowedOrgIDs []int `json:"profiling.generic_metrics.functions_ingestion.allowed_org_ids"`
}
