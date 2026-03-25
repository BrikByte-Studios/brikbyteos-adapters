package mock

import (
	"context"

	"github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// Adapter is a lightweight mock adapter used for runtime integration tests.
// It proves runtime can orchestrate adapters without special-casing.
type Adapter struct{}

// Metadata returns the canonical static description of the mock adapter.
func (Adapter) Metadata() sdk.AdapterMetadata {
	return sdk.AdapterMetadata{
		Name:             "mock",
		Type:             sdk.AdapterTypeOther,
		Description:      "Mock adapter for runtime integration tests",
		Order:            999,
		SupportedTool:    "mock-tool",
		VersionCommand:   []string{"mock-tool", "--version"},
		DefaultTimeoutMs: 1000,
		Aliases:          []string{"fake"},
		Capabilities:     []string{"test-double"},
	}
}

// CheckAvailability reports the mock adapter as always locally available.
func (Adapter) CheckAvailability(context.Context) sdk.Availability {
	return sdk.Availability{
		Available:      true,
		ResolvedBinary: "/usr/bin/mock-tool",
	}
}

// Version returns a deterministic mock version.
func (Adapter) Version(context.Context) (string, error) {
	return "0.0.1", nil
}

// Run returns a structured, successful execution result.
func (Adapter) Run(context.Context, sdk.RunRequest) sdk.RunResult {
	exitCode := 0
	return sdk.RunResult{
		Status:     sdk.ExecutionStatusCompleted,
		ExitCode:   &exitCode,
		DurationMs: 5,
		Stdout:     []byte(`{"ok":true}`),
		Stderr:     []byte{},
		ToolOutput: []byte(`{"checks":1,"passed":1}`),
	}
}

// // Normalize converts the raw execution into a canonical schema-shaped JSON payload.
// //
// // In production adapters, this payload must match the schema owned by brikbyteos-schema.
// // This mock keeps the shape intentionally small but representative.
// func (Adapter) Normalize(_ context.Context, input sdk.NormalizationInput) sdk.NormalizedResult {
// 	payload := map[string]any{
// 		"schema_version": "0.1",
// 		"adapter": map[string]any{
// 			"name":    input.AdapterMeta.Name,
// 			"type":    string(input.AdapterMeta.Type),
// 			"version": input.RawExecution.AdapterVersion,
// 		},
// 		"execution": map[string]any{
// 			"status":      string(input.RawExecution.RunResult.Status),
// 			"duration_ms": input.RawExecution.RunResult.DurationMs,
// 		},
// 		"summary": map[string]any{
// 			"status":  "pass",
// 			"total":   1,
// 			"passed":  1,
// 			"failed":  0,
// 			"skipped": 0,
// 		},
// 		"evidence": map[string]any{
// 			"complete": true,
// 			"issues":   []any{},
// 		},
// 		"artifacts": map[string]any{
// 			"raw_stdout_path":      input.RawExecution.StdoutPath,
// 			"raw_stderr_path":      input.RawExecution.StderrPath,
// 			"raw_tool_output_path": input.RawExecution.ToolOutputPath,
// 		},
// 		"extensions": map[string]any{
// 			"adapter_specific": map[string]any{
// 				"mock": true,
// 			},
// 		},
// 	}

// 	b, err := json.Marshal(payload)
// 	if err != nil {
// 		// This should never happen with the fixed mock payload.
// 		// Return a deterministic fallback that still satisfies the contract shape.
// 		return []byte(`{"schema_version":"0.1","execution":{"status":"completed","duration_ms":0},"summary":{"status":"unknown","total":0,"passed":0,"failed":0,"skipped":0},"evidence":{"complete":false,"issues":["NORMALIZATION_FAILED"]},"artifacts":{"raw_stdout_path":"","raw_stderr_path":"","raw_tool_output_path":""},"extensions":{"adapter_specific":{"mock_fallback":true}}}`)
// 	}

// 	_ = time.Second // keeps this file future-proof for later expansion without lint churn
// 	return b
// }
