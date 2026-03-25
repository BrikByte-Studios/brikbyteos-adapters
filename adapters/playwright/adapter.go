package playwright

import (
	"context"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// adapter is the canonical Playwright adapter implementation for Phase 1.
type adapter struct{}

// New returns the canonical Playwright adapter as an sdk.Adapter.
func New() sdk.Adapter {
	return adapter{}
}

// Metadata returns the canonical static metadata for the Playwright adapter.
func (adapter) Metadata() sdk.AdapterMetadata {
	return Metadata()
}

// CheckAvailability determines whether Playwright tooling is available locally.
func (adapter) CheckAvailability(context.Context) sdk.Availability {
	return sdk.Availability{
		Available:      true,
		ResolvedBinary: "npx",
	}
}

// Version returns a best-effort Playwright version.
func (adapter) Version(context.Context) (string, error) {
	return "UNKNOWN", nil
}

// Run executes the adapter and returns structured execution truth only.
func (adapter) Run(context.Context, sdk.RunRequest) sdk.RunResult {
	return sdk.RunResult{
		Status:       sdk.ExecutionStatusUnavailable,
		DurationMs:   0,
		ErrorMessage: "playwright execution not implemented yet",
	}
}

// Normalize transforms raw execution into canonical normalized JSON.
func (adapter) Normalize(context.Context, sdk.NormalizationInput) sdk.NormalizedResult {
	return []byte(`{
		"schema_version":"0.1",
		"adapter":{"name":"playwright","type":"ui","version":"UNKNOWN"},
		"execution":{"status":"unavailable","duration_ms":0},
		"summary":{"status":"unknown","total":0,"passed":0,"failed":0,"skipped":0},
		"evidence":{"complete":false,"issues":[{"code":"NORMALIZATION_FAILED","message":"playwright normalization not implemented yet"}]},
		"artifacts":{"raw_stdout_path":"","raw_stderr_path":"","raw_tool_output_path":""},
		"extensions":{"adapter_specific":{}}
	}`)
}