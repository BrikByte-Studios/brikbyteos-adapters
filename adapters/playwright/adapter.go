package playwright

import (
	"context"
	"fmt"
	"strings"

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
	return Normalizer{}.Normalize(parseResult)
}

func fallbackUnavailableNormalization(raw sdk.RawExecution) sdk.NormalizedResult {
	return sdk.NormalizedResult([]byte(fmt.Sprintf(`{
		"schema_version":"0.1",
		"adapter":"%s",
		"status":"normalization_failed",
		"result_kind":"test_suite",
		"summary":null,
		"evidence":{"raw_available":false,"normalized_complete":false},
		"error":{"type":"adapter_unavailable","message":%q,"details":{}}
	}`, AdapterName, nonEmpty(raw.RunResult.ErrorMessage, "playwright adapter unavailable"))))
}

func fallbackTimedOutNormalization(raw sdk.RawExecution) sdk.NormalizedResult {
	return sdk.NormalizedResult([]byte(fmt.Sprintf(`{
		"schema_version":"0.1",
		"adapter":"%s",
		"status":"normalization_failed",
		"result_kind":"test_suite",
		"summary":null,
		"evidence":{"raw_available":true,"normalized_complete":false},
		"error":{"type":"execution_timed_out","message":%q,"details":{}}
	}`, AdapterName, nonEmpty(raw.RunResult.ErrorMessage, "playwright execution timed out"))))
}

func fallbackMissingToolOutputNormalization(raw sdk.RawExecution) sdk.NormalizedResult {
	return sdk.NormalizedResult([]byte(fmt.Sprintf(`{
		"schema_version":"0.1",
		"adapter":"%s",
		"status":"normalization_failed",
		"result_kind":"test_suite",
		"summary":null,
		"evidence":{"raw_available":false,"normalized_complete":false},
		"error":{"type":"missing_report","message":"playwright tool output missing","details":{}}
	}`, AdapterName)))
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
