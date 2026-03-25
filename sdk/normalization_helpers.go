package sdk

import (
	"encoding/json"
	"fmt"
)

// CanonicalNormalizedResult is the in-memory canonical normalized shape.
//
// This struct mirrors the schema contract while keeping top-level fields stable.
// Adapter-specific details are confined to Extensions.AdapterSpecific.
type CanonicalNormalizedResult struct {
	SchemaVersion string `json:"schema_version"`

	Adapter struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Version string `json:"version"`
	} `json:"adapter"`

	Execution struct {
		Status     string `json:"status"`
		DurationMs int64  `json:"duration_ms"`
	} `json:"execution"`

	Summary struct {
		Status  string `json:"status"`
		Total   int64  `json:"total"`
		Passed  int64  `json:"passed"`
		Failed  int64  `json:"failed"`
		Skipped int64  `json:"skipped"`
	} `json:"summary"`

	Evidence struct {
		Complete bool                 `json:"complete"`
		Issues   []NormalizationIssue `json:"issues"`
	} `json:"evidence"`

	Artifacts struct {
		RawStdoutPath     string `json:"raw_stdout_path,omitempty"`
		RawStderrPath     string `json:"raw_stderr_path,omitempty"`
		RawToolOutputPath string `json:"raw_tool_output_path,omitempty"`
	} `json:"artifacts"`

	Extensions struct {
		AdapterSpecific map[string]any `json:"adapter_specific"`
	} `json:"extensions"`
}

// NewBaseNormalizedResult constructs a canonical base result that carries execution truth
// forward directly from raw execution without reinterpretation.
func NewBaseNormalizedResult(input NormalizationInput) CanonicalNormalizedResult {
	var out CanonicalNormalizedResult

	out.SchemaVersion = "0.1"

	out.Adapter.Name = input.AdapterMeta.Name
	out.Adapter.Type = string(input.AdapterMeta.Type)
	out.Adapter.Version = input.RawExecution.AdapterVersion

	out.Execution.Status = string(input.RawExecution.RunResult.Status)
	out.Execution.DurationMs = input.RawExecution.RunResult.DurationMs

	out.Summary.Status = string(SummaryStatusUnknown)
	out.Summary.Total = 0
	out.Summary.Passed = 0
	out.Summary.Failed = 0
	out.Summary.Skipped = 0

	out.Evidence.Complete = false
	out.Evidence.Issues = []NormalizationIssue{}

	out.Artifacts.RawStdoutPath = input.RawExecution.StdoutPath
	out.Artifacts.RawStderrPath = input.RawExecution.StderrPath
	out.Artifacts.RawToolOutputPath = input.RawExecution.ToolOutputPath

	out.Extensions.AdapterSpecific = map[string]any{}

	return out
}

// ApplyExecutionScenarioDefaults applies the Phase 1 canonical execution-scenario behavior.
//
// Rules:
//   - completed/failed may still be evidence-complete if normalization succeeds
//   - timed_out => summary.unknown, evidence.complete=false
//   - unavailable => summary.unknown, evidence.complete=false
func ApplyExecutionScenarioDefaults(out *CanonicalNormalizedResult, input NormalizationInput) {
	if out == nil {
		return
	}

	switch input.RawExecution.RunResult.Status {
	case ExecutionStatusTimedOut:
		out.Summary.Status = string(SummaryStatusUnknown)
		out.Evidence.Complete = false
		out.Evidence.Issues = append(out.Evidence.Issues, NormalizationIssue{
			Code:    IssueExecutionTimedOut,
			Message: "adapter execution timed out",
		})
	case ExecutionStatusUnavailable:
		out.Summary.Status = string(SummaryStatusUnknown)
		out.Evidence.Complete = false
		out.Evidence.Issues = append(out.Evidence.Issues, NormalizationIssue{
			Code:    IssueToolUnavailable,
			Message: "required tool or binary was unavailable",
		})
	}
}

// MarkNormalizationFailure converts the result into a valid fallback result when
// normalization cannot determine domain meaning safely.
func MarkNormalizationFailure(out *CanonicalNormalizedResult, message string) {
	if out == nil {
		return
	}

	out.Summary.Status = string(SummaryStatusUnknown)
	out.Summary.Total = 0
	out.Summary.Passed = 0
	out.Summary.Failed = 0
	out.Summary.Skipped = 0
	out.Evidence.Complete = false
	out.Evidence.Issues = append(out.Evidence.Issues, NormalizationIssue{
		Code:    IssueNormalizationFailed,
		Message: message,
	})
}

// SetSummary safely assigns summary metrics and validates the summary status.
func SetSummary(out *CanonicalNormalizedResult, status SummaryStatus, total, passed, failed, skipped int64) error {
	if out == nil {
		return fmt.Errorf("normalized result must not be nil")
	}
	if err := status.Validate(); err != nil {
		return err
	}
	if total < 0 || passed < 0 || failed < 0 || skipped < 0 {
		return fmt.Errorf("summary metrics must be >= 0")
	}

	out.Summary.Status = string(status)
	out.Summary.Total = total
	out.Summary.Passed = passed
	out.Summary.Failed = failed
	out.Summary.Skipped = skipped
	return nil
}

// ComputeEvidenceCompleteness applies the canonical Phase 1 completeness rule.
func ComputeEvidenceCompleteness(input NormalizationInput, output *CanonicalNormalizedResult, normalizationSucceeded bool) {
	if output == nil {
		return
	}

	switch input.RawExecution.RunResult.Status {
	case ExecutionStatusUnavailable, ExecutionStatusTimedOut:
		output.Evidence.Complete = false
		return
	}

	if !normalizationSucceeded {
		output.Evidence.Complete = false
		return
	}

	output.Evidence.Complete = true
}

// MarshalNormalizedResult converts the canonical result into stable JSON bytes.
func MarshalNormalizedResult(out CanonicalNormalizedResult) (NormalizedResult, error) {
	payload, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal normalized result: %w", err)
	}
	return payload, nil
}