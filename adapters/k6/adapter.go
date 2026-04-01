package k6

import (
	"context"
	"strings"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// adapter is the canonical k6 adapter implementation for Phase 1.
type adapter struct{}

// New returns the canonical k6 adapter as an sdk.Adapter.
func New() sdk.Adapter {
	return adapter{}
}

// Metadata returns the canonical static metadata for the k6 adapter.
func (adapter) Metadata() sdk.AdapterMetadata {
	return Metadata()
}

// CheckAvailability determines whether k6 is available locally.
//
// Current Phase 1 behavior:
//   - returns unavailable until deterministic binary resolution + execution are implemented
//   - keeps availability truthful rather than optimistic
func (adapter) CheckAvailability(context.Context) sdk.Availability {
	return sdk.Availability{
		Available:      false,
		ResolvedBinary: "",
		Reason:         "binary not found in PATH",
	}
}

// Version returns a best-effort k6 version.
func (adapter) Version(context.Context) (string, error) {
	return "UNKNOWN", nil
}

// Run executes the adapter and returns structured execution truth only.
//
// Current Phase 1 behavior:
//   - execution is intentionally not implemented yet
//   - returns a structured unavailable result rather than crashing
func (adapter) Run(context.Context, sdk.RunRequest) sdk.RunResult {
	return sdk.RunResult{
		Status:       sdk.ExecutionStatusUnavailable,
		DurationMs:   0,
		ErrorMessage: "binary not found in PATH",
	}
}

// Normalize transforms raw execution into canonical normalized JSON.
//
// Rules:
//   - unavailable/timed-out executions are represented in execution.status and evidence.issues
//   - missing tool output is represented as incomplete evidence, not as a schema shape change
//   - parseable tool output is normalized via Parser + Normalizer
func (adapter) Normalize(_ context.Context, in sdk.NormalizationInput) sdk.NormalizedResult {
	switch in.RawExecution.RunResult.Status {
	case sdk.ExecutionStatusUnavailable:
		return fallbackUnavailableNormalization(in.RawExecution)
	case sdk.ExecutionStatusTimedOut:
		return fallbackTimedOutNormalization(in.RawExecution)
	}

	toolOutput := in.RawExecution.RunResult.ToolOutput
	if len(toolOutput) == 0 {
		return fallbackMissingToolOutputNormalization(in.RawExecution)
	}

	parseResult := (Parser{}).ParseBytes(toolOutput)
	return Normalizer{}.Normalize(parseResult, in.RawExecution)
}

func fallbackUnavailableNormalization(raw sdk.RawExecution) sdk.NormalizedResult {
	return mustMarshalNormalized(normalizedPayload{
		SchemaVersion: "0.1",
		Adapter: normalizedAdapter{
			Name:    AdapterName,
			Type:    string(raw.AdapterType),
			Version: nonEmpty(raw.AdapterVersion, "UNKNOWN"),
		},
		Execution: normalizedExecution{
			Status:     "unavailable",
			DurationMs: raw.RunResult.DurationMs,
		},
		Summary: normalizedSummary{
			Status:  "unknown",
			Total:   0,
			Passed:  0,
			Failed:  0,
			Skipped: 0,
		},
		Evidence: normalizedEvidence{
			Complete: false,
			Issues: []normalizedIssue{
				{
					Code:    "ADAPTER_UNAVAILABLE",
					Message: nonEmpty(raw.RunResult.ErrorMessage, "k6 adapter unavailable"),
				},
			},
		},
		Artifacts: normalizedArtifacts{
			RawStdoutPath:     raw.StdoutPath,
			RawStderrPath:     raw.StderrPath,
			RawToolOutputPath: raw.ToolOutputPath,
		},
		Extensions: normalizedExtensions{
			AdapterSpecific: map[string]any{},
		},
	})
}

func fallbackTimedOutNormalization(raw sdk.RawExecution) sdk.NormalizedResult {
	return mustMarshalNormalized(normalizedPayload{
		SchemaVersion: "0.1",
		Adapter: normalizedAdapter{
			Name:    AdapterName,
			Type:    string(raw.AdapterType),
			Version: nonEmpty(raw.AdapterVersion, "UNKNOWN"),
		},
		Execution: normalizedExecution{
			Status:     "timed_out",
			DurationMs: raw.RunResult.DurationMs,
		},
		Summary: normalizedSummary{
			Status:  "unknown",
			Total:   0,
			Passed:  0,
			Failed:  0,
			Skipped: 0,
		},
		Evidence: normalizedEvidence{
			Complete: false,
			Issues: []normalizedIssue{
				{
					Code:    "EXECUTION_TIMED_OUT",
					Message: nonEmpty(raw.RunResult.ErrorMessage, "k6 execution timed out"),
				},
			},
		},
		Artifacts: normalizedArtifacts{
			RawStdoutPath:     raw.StdoutPath,
			RawStderrPath:     raw.StderrPath,
			RawToolOutputPath: raw.ToolOutputPath,
		},
		Extensions: normalizedExtensions{
			AdapterSpecific: map[string]any{},
		},
	})
}

func fallbackMissingToolOutputNormalization(raw sdk.RawExecution) sdk.NormalizedResult {
	return mustMarshalNormalized(normalizedPayload{
		SchemaVersion: "0.1",
		Adapter: normalizedAdapter{
			Name:    AdapterName,
			Type:    string(raw.AdapterType),
			Version: nonEmpty(raw.AdapterVersion, "UNKNOWN"),
		},
		Execution: normalizedExecution{
			Status:     mapExecutionStatus(raw.RunResult.Status),
			DurationMs: raw.RunResult.DurationMs,
		},
		Summary: normalizedSummary{
			Status:  "unknown",
			Total:   0,
			Passed:  0,
			Failed:  0,
			Skipped: 0,
		},
		Evidence: normalizedEvidence{
			Complete: false,
			Issues: []normalizedIssue{
				{
					Code:    "MISSING_TOOL_OUTPUT",
					Message: "k6 tool output missing",
				},
			},
		},
		Artifacts: normalizedArtifacts{
			RawStdoutPath:     raw.StdoutPath,
			RawStderrPath:     raw.StderrPath,
			RawToolOutputPath: raw.ToolOutputPath,
		},
		Extensions: normalizedExtensions{
			AdapterSpecific: map[string]any{},
		},
	})
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}