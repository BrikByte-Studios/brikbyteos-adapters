package sdk

import (
	"context"
	"fmt"
)

// SummaryStatus is the domain-level interpretation of normalized output.
//
// This is intentionally separate from execution lifecycle state.
type SummaryStatus string

const (
	SummaryStatusPass    SummaryStatus = "pass"
	SummaryStatusFail    SummaryStatus = "fail"
	SummaryStatusWarn    SummaryStatus = "warn"
	SummaryStatusUnknown SummaryStatus = "unknown"
)

// Validate ensures the summary status is supported.
func (s SummaryStatus) Validate() error {
	switch s {
	case SummaryStatusPass, SummaryStatusFail, SummaryStatusWarn, SummaryStatusUnknown:
		return nil
	default:
		return fmt.Errorf("invalid summary status %q", s)
	}
}

// NormalizationIssueCode is the canonical structured issue code used in normalized output.
type NormalizationIssueCode string

const (
	IssueTestFailure         NormalizationIssueCode = "TEST_FAILURE"
	IssueVulnerabilityFound  NormalizationIssueCode = "VULNERABILITY_FOUND"
	IssueNormalizationFailed NormalizationIssueCode = "NORMALIZATION_FAILED"
	IssueToolUnavailable     NormalizationIssueCode = "TOOL_UNAVAILABLE"
	IssueExecutionTimedOut   NormalizationIssueCode = "EXECUTION_TIMED_OUT"
)

// NormalizationIssue is a structured issue entry in normalized output.
type NormalizationIssue struct {
	Code    NormalizationIssueCode `json:"code"`
	Message string                 `json:"message,omitempty"`
}

// Validate ensures normalization input is structurally valid.
func (n NormalizationInput) Validate() error {
	if err := n.RawExecution.Validate(); err != nil {
		return fmt.Errorf("validate raw_execution: %w", err)
	}
	if err := n.AdapterMeta.Validate(); err != nil {
		return fmt.Errorf("validate adapter_meta: %w", err)
	}
	return nil
}

// Normalizer is the canonical normalization boundary used by adapters.
type Normalizer interface {
	// Normalize must be deterministic and side-effect free.
	Normalize(ctx context.Context, input NormalizationInput) NormalizedResult
}
