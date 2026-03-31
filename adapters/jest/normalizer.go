package jest

import (
	"encoding/json"
	"strings"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// normalizedIssue is the bounded issue shape surfaced in normalized results.
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

type normalizedJestResult struct {
	SchemaVersion string               `json:"schema_version"`
	Adapter       normalizedAdapter    `json:"adapter"`
	Execution     normalizedExecution  `json:"execution"`
	Summary       normalizedSummary    `json:"summary"`
	Evidence      normalizedEvidence   `json:"evidence"`
	Artifacts     normalizedArtifacts  `json:"artifacts"`
	Extensions    normalizedExtensions `json:"extensions"`
}

// Normalizer converts raw execution truth into canonical normalized JSON.
type Normalizer struct{}

// Normalize converts the adapter-private ParseResult into canonical normalized JSON.
//
// This is useful when you want to keep parser and mapping concerns separate.
func (Normalizer) Normalize(input ParseResult, raw sdk.RawExecution) sdk.NormalizedResult {
	result := normalizedJestResult{
		SchemaVersion: "0.1",
		Adapter: normalizedAdapter{
			Name:    raw.AdapterName,
			Type:    string(raw.AdapterType),
			Version: raw.AdapterVersion,
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
		result.Evidence.Complete = false
		result.Evidence.Issues = []normalizedIssue{
			{
				Code:    parserErrorCode(input.Error),
				Message: parserErrorMessage(input.Error),
			},
		}
		encoded, err := marshalNormalizedResult(result)
		if err != nil {
			return fallbackNormalizationFailure(raw, "failed to marshal parser-failure normalized result")
		}
		return encoded
	}

	if input.Summary == nil {
		result.Evidence.Complete = false
		result.Evidence.Issues = []normalizedIssue{
			{
				Code:    "SCHEMA_MISMATCH",
				Message: "parser returned ok status without summary",
			},
		}
		encoded, err := marshalNormalizedResult(result)
		if err != nil {
			return fallbackNormalizationFailure(raw, "failed to marshal schema-mismatch normalized result")
		}
		return encoded
	}

	result.Summary.Total = input.Summary.TestTotal
	result.Summary.Passed = input.Summary.TestPassed
	result.Summary.Failed = input.Summary.TestFailed
	result.Summary.Skipped = input.Summary.TestSkipped
	result.Summary.Status = canonicalStatusFromSummary(*input.Summary)
	result.Execution.DurationMs = input.Summary.DurationMs
	result.Evidence.Complete = true
	result.Evidence.Issues = []normalizedIssue{}

	encoded, err := marshalNormalizedResult(result)
	if err != nil {
		return fallbackNormalizationFailure(raw, "failed to marshal normalized result")
	}
	return encoded
}

// normalizeJestRawExecution converts raw execution truth into canonical normalized JSON.
//
// This is the main entry point for adapter normalization.
func normalizeJestRawExecution(raw sdk.RawExecution) (sdk.NormalizedResult, error) {
	result := normalizedJestResult{
		SchemaVersion: "0.1",
		Adapter: normalizedAdapter{
			Name:    raw.AdapterName,
			Type:    string(raw.AdapterType),
			Version: raw.AdapterVersion,
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

	switch raw.RunResult.Status {
	case sdk.ExecutionStatusUnavailable:
		result.Evidence.Complete = false
		result.Evidence.Issues = append(result.Evidence.Issues, normalizedIssue{
			Code:    "ADAPTER_UNAVAILABLE",
			Message: nonEmptyOr(raw.RunResult.ErrorMessage, "adapter unavailable"),
		})
		return marshalNormalizedResult(result)

	case sdk.ExecutionStatusTimedOut:
		result.Evidence.Complete = false
		result.Evidence.Issues = append(result.Evidence.Issues, normalizedIssue{
			Code:    "EXECUTION_TIMED_OUT",
			Message: nonEmptyOr(raw.RunResult.ErrorMessage, "adapter execution timed out"),
		})
		return marshalNormalizedResult(result)
	}

	toolOutput := strings.TrimSpace(string(raw.RunResult.ToolOutput))
	if toolOutput == "" {
		result.Evidence.Complete = false
		result.Evidence.Issues = append(result.Evidence.Issues, normalizedIssue{
			Code:    "MISSING_TOOL_OUTPUT",
			Message: "jest tool output was empty; unable to derive test summary",
		})
		return marshalNormalizedResult(result)
	}

	parseResult := (Parser{}).ParseBytes([]byte(toolOutput))
	return Normalizer{}.Normalize(parseResult, raw), nil
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
		return "failed to parse jest tool output"
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

func marshalNormalizedResult(v any) (sdk.NormalizedResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return sdk.NormalizedResult(b), nil
}

func nonEmptyOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func fallbackNormalizationFailure(raw sdk.RawExecution, message string) sdk.NormalizedResult {
	payload := normalizedJestResult{
		SchemaVersion: "0.1",
		Adapter: normalizedAdapter{
			Name:    raw.AdapterName,
			Type:    string(raw.AdapterType),
			Version: raw.AdapterVersion,
		},
		Execution: normalizedExecution{
			Status:     "unknown",
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
					Code:    "NORMALIZATION_FAILURE",
					Message: message,
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
	}

	encoded, err := marshalNormalizedResult(payload)
	if err != nil {
		return sdk.NormalizedResult(`{"schema_version":"0.1","adapter":{"name":"jest","type":"unit","version":"UNKNOWN"},"execution":{"status":"unknown","duration_ms":0},"summary":{"status":"unknown","total":0,"passed":0,"failed":0,"skipped":0},"evidence":{"complete":false,"issues":[{"code":"NORMALIZATION_FAILURE","message":"failed to build fallback normalized result"}]},"artifacts":{"raw_stdout_path":"","raw_stderr_path":"","raw_tool_output_path":""},"extensions":{"adapter_specific":{}}}`)
	}
	return encoded
}