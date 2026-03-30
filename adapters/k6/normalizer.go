package k6

import (
	"encoding/json"
	"fmt"
	"sort"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// Canonical normalized result constants.
//
// These constants are centralized to avoid status drift between the
// k6 parser, normalizer, tests, and downstream consumers.
const (
	k6NormalizedSchemaVersion = "0.1"
	k6ResultKindPerformance   = "performance_suite"

	k6StatusPass                = "pass"
	k6StatusFailed              = "failed"
	k6StatusNormalizationFailed = "normalization_failed"
)

// Normalizer converts the adapter-private k6 parse model into the
// canonical BrikByte normalized result.
//
// Design goals:
//   - pure function of parser output
//   - deterministic field mapping
//   - explicit parser-failure handling
//   - strict anti-leak boundary for k6-specific details
type Normalizer struct{}

// Normalize converts a k6 ParseResult into canonical normalized JSON.
//
// Mapping rules:
//   - parse ok + zero threshold failures -> pass
//   - parse ok + one or more threshold failures -> failed
//   - parse failed -> normalization_failed
//
// issue_count is always mapped from threshold_failed_count.
func (Normalizer) Normalize(input ParseResult) sdk.NormalizedResult {
	payload := buildNormalizedPayload(input)

	encoded, err := json.Marshal(payload)
	if err != nil {
		fallback := normalizedFailurePayload(&ParserError{
			Type:    ParserErrorType("marshal_failed"),
			Message: "failed to marshal normalized k6 payload",
			Details: map[string]any{},
		})

		encodedFallback, fallbackErr := json.Marshal(fallback)
		if fallbackErr != nil {
			return sdk.NormalizedResult(`{"schema_version":"0.1","adapter":"k6","status":"normalization_failed","result_kind":"performance_suite","summary":null,"evidence":{"raw_available":true,"normalized_complete":false},"error":{"type":"marshal_failed","message":"failed to marshal normalized k6 payload","details":{}}}`)
		}

		return sdk.NormalizedResult(encodedFallback)
	}

	return sdk.NormalizedResult(encoded)
}

// buildNormalizedPayload creates the in-memory canonical payload.
// It is intentionally separated from Normalize to improve testability.
func buildNormalizedPayload(input ParseResult) normalizedPayload {
	if input.ParseStatus == ParseStatusParseFailed {
		return normalizedFailurePayload(input.Error)
	}

	if input.Summary == nil {
		return normalizedFailurePayload(&ParserError{
			Type:    ParserErrorType("schema_mismatch"),
			Message: "parser returned ok status without summary",
			Details: map[string]any{},
		})
	}

	thresholds := stableThresholdSummaries(input.Thresholds)
	thresholdFailedCount := countFailedThresholds(thresholds)
	status := canonicalStatusFromThresholds(thresholdFailedCount)

	payload := normalizedPayload{
		SchemaVersion: k6NormalizedSchemaVersion,
		Adapter:       AdapterName,
		Status:        status,
		ResultKind:    k6ResultKindPerformance,
		Summary: &normalizedSummary{
			RequestTotal:         input.Summary.RequestTotal,
			RequestFailed:        input.Summary.RequestFailed,
			LatencyP95Ms:         input.Summary.LatencyP95Ms,
			LatencyP99Ms:         input.Summary.LatencyP99Ms,
			DurationMs:           input.Summary.DurationMs,
			IssueCount:           thresholdFailedCount,
			ThresholdFailedCount: thresholdFailedCount,
		},
		Evidence: normalizedEvidence{
			RawAvailable:       true,
			NormalizedComplete: true,
		},
	}

	// Keep k6-specific details inside extensions.k6 only.
	if len(thresholds) > 0 {
		payload.Extensions = &normalizedExtensions{
			K6: &normalizedK6Extension{
				Thresholds: thresholds,
			},
		}
	}

	if len(input.Warnings) > 0 {
		if payload.Extensions == nil {
			payload.Extensions = &normalizedExtensions{
				K6: &normalizedK6Extension{},
			}
		}
		payload.Extensions.K6.Warnings = stableWarnings(input.Warnings)
	}

	return payload
}

// canonicalStatusFromThresholds computes the canonical normalized status.
func canonicalStatusFromThresholds(thresholdFailedCount int) string {
	if thresholdFailedCount > 0 {
		return k6StatusFailed
	}
	return k6StatusPass
}

// countFailedThresholds returns the number of threshold summaries marked fail.
func countFailedThresholds(thresholds []normalizedThresholdSummary) int {
	count := 0
	for _, t := range thresholds {
		if t.Status == string(ThresholdStatusFail) {
			count++
		}
	}
	return count
}

// normalizedFailurePayload converts parser failure input into the canonical
// normalization_failed shape with no fabricated summary metrics.
func normalizedFailurePayload(err *ParserError) normalizedPayload {
	if err == nil {
		err = &ParserError{
			Type:    ParserErrorType("parse_failed"),
			Message: "parser failed without structured error",
			Details: map[string]any{},
		}
	}

	return normalizedPayload{
		SchemaVersion: k6NormalizedSchemaVersion,
		Adapter:       AdapterName,
		Status:        k6StatusNormalizationFailed,
		ResultKind:    k6ResultKindPerformance,
		Summary:       nil,
		Evidence: normalizedEvidence{
			RawAvailable:       true,
			NormalizedComplete: false,
		},
		Error: &normalizedError{
			Type:    string(err.Type),
			Message: err.Message,
			Details: err.Details,
		},
	}
}

// stableThresholdSummaries sorts and copies k6 threshold summaries for determinism.
func stableThresholdSummaries(in []ParsedThresholdSummary) []normalizedThresholdSummary {
	out := make([]normalizedThresholdSummary, 0, len(in))
	for _, th := range in {
		out = append(out, normalizedThresholdSummary{
			Name:   th.Name,
			Status: string(th.Status),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Status < out[j].Status
	})

	return out
}

// stableWarnings sorts warnings deterministically.
func stableWarnings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// --- Canonical output model ---
//
// These structs intentionally represent only the bounded canonical result
// required for this task. They are not a raw pass-through of k6.

type normalizedPayload struct {
	SchemaVersion string                `json:"schema_version"`
	Adapter       string                `json:"adapter"`
	Status        string                `json:"status"`
	ResultKind    string                `json:"result_kind"`
	Summary       *normalizedSummary    `json:"summary"`
	Evidence      normalizedEvidence    `json:"evidence"`
	Error         *normalizedError      `json:"error,omitempty"`
	Extensions    *normalizedExtensions `json:"extensions,omitempty"`
}

type normalizedSummary struct {
	RequestTotal         int      `json:"request_total"`
	RequestFailed        int      `json:"request_failed"`
	LatencyP95Ms         *float64 `json:"latency_p95_ms,omitempty"`
	LatencyP99Ms         *float64 `json:"latency_p99_ms,omitempty"`
	DurationMs           int64    `json:"duration_ms"`
	IssueCount           int      `json:"issue_count"`
	ThresholdFailedCount int      `json:"threshold_failed_count"`
}

type normalizedEvidence struct {
	RawAvailable       bool `json:"raw_available"`
	NormalizedComplete bool `json:"normalized_complete"`
}

type normalizedError struct {
	Type    string         `json:"type"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type normalizedExtensions struct {
	K6 *normalizedK6Extension `json:"k6,omitempty"`
}

type normalizedK6Extension struct {
	Thresholds []normalizedThresholdSummary `json:"thresholds,omitempty"`
	Warnings   []string                     `json:"warnings,omitempty"`
}

type normalizedThresholdSummary struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ValidateNormalizedPayloadShape performs lightweight internal invariant checks.
//
// This is intentionally narrow. It is not a replacement for future stricter
// schema validation from brikbyteos-schema.
func ValidateNormalizedPayloadShape(payload normalizedPayload) error {
	if payload.SchemaVersion != k6NormalizedSchemaVersion {
		return fmt.Errorf("unexpected schema_version: %s", payload.SchemaVersion)
	}

	if payload.Adapter != AdapterName {
		return fmt.Errorf("unexpected adapter: %s", payload.Adapter)
	}

	if payload.ResultKind != k6ResultKindPerformance {
		return fmt.Errorf("unexpected result_kind: %s", payload.ResultKind)
	}

	switch payload.Status {
	case k6StatusPass, k6StatusFailed:
		if payload.Summary == nil {
			return fmt.Errorf("summary is required for non-failure normalized payload")
		}
		if !payload.Evidence.NormalizedComplete {
			return fmt.Errorf("normalized_complete must be true for successful mapping")
		}
		if payload.Summary.IssueCount != payload.Summary.ThresholdFailedCount {
			return fmt.Errorf("issue_count must equal threshold_failed_count")
		}

	case k6StatusNormalizationFailed:
		if payload.Summary != nil {
			return fmt.Errorf("summary must be nil for normalization_failed")
		}
		if payload.Evidence.NormalizedComplete {
			return fmt.Errorf("normalized_complete must be false for normalization_failed")
		}
		if payload.Error == nil {
			return fmt.Errorf("error is required for normalization_failed")
		}

	default:
		return fmt.Errorf("unexpected status: %s", payload.Status)
	}

	return nil
}
