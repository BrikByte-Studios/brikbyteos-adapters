package jest

import (
	"context"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// adapter is the canonical Jest adapter implementation for Phase 1.
//
// This implementation is intentionally minimal.
// It satisfies the SDK contract while execution and normalization mature.
type adapter struct{}

// New returns the canonical Jest adapter as an sdk.Adapter.
func New() sdk.Adapter {
	return adapter{}
}

// Metadata returns the canonical static metadata for the Jest adapter.
func (adapter) Metadata() sdk.AdapterMetadata {
	return Metadata()
}

// CheckAvailability determines whether the Jest toolchain is available locally.
//
// Phase 1 note:
// This lightweight implementation assumes the command is potentially available.
// Later work can replace this with real local binary/toolchain detection.
func (adapter) CheckAvailability(context.Context) sdk.Availability {
	return sdk.Availability{
		Available:      true,
		ResolvedBinary: "npx",
	}
}

// Version returns the best-effort Jest version.
//
// Phase 1 note:
// Returning UNKNOWN is acceptable until command execution is wired fully.
func (adapter) Version(context.Context) (string, error) {
	return "UNKNOWN", nil
}

// Run executes the adapter and returns process-level execution truth only.
//
// Phase 1 note:
// This placeholder avoids crashes while the real command execution flow is still evolving.
func (adapter) Run(context.Context, sdk.RunRequest) sdk.RunResult {
	return sdk.RunResult{
		Status:       sdk.ExecutionStatusUnavailable,
		DurationMs:   0,
		ErrorMessage: "jest execution not implemented yet",
	}
}

// Normalize transforms raw execution into canonical normalized JSON.
//
// Phase 1 note:
// This placeholder keeps the adapter schema-compatible until real normalization is added.
func (adapter) Normalize(context.Context, sdk.NormalizationInput) sdk.NormalizedResult {
	return []byte(`{
		"schema_version":"0.1",
		"adapter":{"name":"jest","type":"unit","version":"UNKNOWN"},
		"execution":{"status":"unavailable","duration_ms":0},
		"summary":{"status":"unknown","total":0,"passed":0,"failed":0,"skipped":0},
		"evidence":{"complete":false,"issues":[{"code":"NORMALIZATION_FAILED","message":"jest normalization not implemented yet"}]},
		"artifacts":{"raw_stdout_path":"","raw_stderr_path":"","raw_tool_output_path":""},
		"extensions":{"adapter_specific":{}}
	}`)
}
