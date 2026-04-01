package playwright

import (
	"encoding/json"
	"strings"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// normalizedIssue is the bounded issue shape surfaced in canonical output.
type normalizedIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type normalizedAdapter struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

type normalizedExecution struct {
	Status     string `json:"status"`
	DurationMs int64  `json:"duration_ms"`
}

type normalizedSummary struct {
	Status  string `json:"status"`
	Total   int    `json:"total"`
	Passed  int    `json:"passed"`
	Failed  int    `json:"failed"`
	Skipped int    `json:"skipped"`
}

type normalizedEvidence struct {
	Complete bool              `json:"complete"`
	Issues   []normalizedIssue `json:"issues"`
}

type normalizedArtifacts struct {
	RawStdoutPath     string `json:"raw_stdout_path"`
	RawStderrPath     string `json:"raw_stderr_path"`
	RawToolOutputPath string `json:"raw_tool_output_path"`
}

type normalizedExtensions struct {
	AdapterSpecific map[string]any `json:"adapter_specific"`
}

type normalizedPayload struct {
	SchemaVersion string               `json:"schema_version"`
	Adapter       normalizedAdapter    `json:"adapter"`
	Execution     normalizedExecution  `json:"execution"`
	Summary       normalizedSummary    `json:"summary"`
	Evidence      normalizedEvidence   `json:"evidence"`
	Artifacts     normalizedArtifacts  `json:"artifacts"`
	Extensions    normalizedExtensions `json:"extensions"`
}

// Normalizer converts adapter-private Playwright parse output into the canonical
// normalized result contract shared across adapters.
type Normalizer struct{}

// Normalize converts a Playwright ParseResult plus raw execution truth into the
// canonical normalized result JSON.
func (Normalizer) Normalize(input ParseResult, raw sdk.RawExecution) sdk.NormalizedResult {
	payload := normalizedPayload{
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
			Issues:   []normalizedIssue{},
		},
		Artifacts: normalizedArtifacts{
			RawStdoutPath:     raw.StdoutPath,
			RawStderrPath:     raw.StderrPath,
			RawToolOutputPath: raw.ToolOutputPath,
		},
		Extensions: normalizedExtensions{
			AdapterSpecific: map[string]any{},
		},
	}

	if input.ParseStatus == ParseStatusParseFailed {
		payload.Evidence.Complete = false
		payload.Evidence.Issues = []normalizedIssue{
			{
				Code:    parserErrorCode(input.Error),
				Message: parserErrorMessage(input.Error),
			},
		}
		return mustMarshalNormalized(payload)
	}

	if input.Summary == nil {
		payload.Evidence.Complete = false
		payload.Evidence.Issues = []normalizedIssue{
			{
				Code:    "SCHEMA_MISMATCH",
				Message: "parser returned ok status without summary",
			},
		}
		return mustMarshalNormalized(payload)
	}

	payload.Summary.Total = input.Summary.TestTotal
	payload.Summary.Passed = input.Summary.TestPassed
	payload.Summary.Failed = input.Summary.TestFailed
	payload.Summary.Skipped = input.Summary.TestSkipped
	payload.Summary.Status = canonicalStatusFromSummary(*input.Summary)

	payload.Execution.DurationMs = input.Summary.DurationMs
	payload.Evidence.Complete = true
	payload.Evidence.Issues = []normalizedIssue{}

	return mustMarshalNormalized(payload)
}

func canonicalStatusFromSummary(summary ParsedSummary) string {
	switch {
	case summary.TestFailed > 0:
		return "failed"
	case summary.TestPassed > 0 && summary.TestFailed == 0:
		return "passed"
	case summary.TestSkipped > 0 && summary.TestPassed == 0 && summary.TestFailed == 0:
		return "skipped"
	default:
		return "unknown"
	}
}

func parserErrorCode(err *ParserError) string {
	if err == nil {
		return "PARSE_FAILED"
	}

	switch err.Type {
	case ParserErrorMissingReport:
		return "MISSING_TOOL_OUTPUT"
	case ParserErrorInvalidJSON:
		return "INVALID_TOOL_OUTPUT"
	case ParserErrorSchemaMismatch:
		return "SCHEMA_MISMATCH"
	default:
		return "PARSE_FAILED"
	}
}

func parserErrorMessage(err *ParserError) string {
	if err == nil || strings.TrimSpace(err.Message) == "" {
		return "failed to parse playwright tool output"
	}
	return err.Message
}

func mapExecutionStatus(status sdk.ExecutionStatus) string {
	switch status {
	case sdk.ExecutionStatusCompleted:
		return "completed"
	case sdk.ExecutionStatusFailed:
		return "failed"
	case sdk.ExecutionStatusTimedOut:
		return "timed_out"
	case sdk.ExecutionStatusUnavailable:
		return "unavailable"
	default:
		return "unknown"
	}
}

func mustMarshalNormalized(payload normalizedPayload) sdk.NormalizedResult {
	encoded, err := json.Marshal(payload)
	if err != nil {
		fallback := `{"schema_version":"0.1","adapter":{"name":"playwright","type":"ui","version":"UNKNOWN"},"execution":{"status":"unknown","duration_ms":0},"summary":{"status":"unknown","total":0,"passed":0,"failed":0,"skipped":0},"evidence":{"complete":false,"issues":[{"code":"NORMALIZATION_FAILURE","message":"failed to marshal normalized playwright payload"}]},"artifacts":{"raw_stdout_path":"","raw_stderr_path":"","raw_tool_output_path":""},"extensions":{"adapter_specific":{}}}`
		return sdk.NormalizedResult(fallback)
	}
	return sdk.NormalizedResult(encoded)
}