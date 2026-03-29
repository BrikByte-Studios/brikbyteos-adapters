package jest

import (
	"encoding/json"
	"fmt"
	"sort"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// Canonical normalized result constants.
//
// These are intentionally centralized to avoid status drift between
// parser tests, normalizer tests, and downstream consumers.
const (
	normalizedSchemaVersion = "0.1"
	resultKindTestSuite     = "test_suite"

	statusPass                = "pass"
	statusFailed              = "failed"
	statusNormalizationFailed = "normalization_failed"
)

// Normalizer converts the adapter-private Jest parse model into the
// canonical BrikByte normalized result.
//
// Design goals:
//   - pure function of parser output
//   - deterministic status mapping
//   - zero fabricated metrics on parser-failure paths
//   - anti-leak boundary for Jest-specific details
type Normalizer struct{}

// Normalize converts the Jest ParseResult into canonical normalized JSON.
//
// Mapping rules:
//   - parse ok + zero failed tests -> pass
//   - parse ok + one or more failed tests -> failed
//   - parse failed -> normalization_failed
//
// issue_count is always mapped from test_failed, not suite_failed.
func (Normalizer) Normalize(input ParseResult) sdk.NormalizedResult {
	payload := buildNormalizedPayload(input)

	encoded, err := json.Marshal(payload)
	if err != nil {
		// This should be exceptionally rare because payload is fully controlled.
		// Even then, return a deterministic fallback failure result instead of panicking.
		fallback := normalizedFailurePayload(
			&ParserError{
				Type:    ParserErrorType("marshal_failed"),
				Message: "failed to marshal normalized Jest payload",
				Details: map[string]any{},
			},
		)

		encodedFallback, fallbackErr := json.Marshal(fallback)
		if fallbackErr != nil {
			return sdk.NormalizedResult(`{"schema_version":"0.1","adapter":"jest","status":"normalization_failed","result_kind":"test_suite","summary":null,"evidence":{"raw_available":true,"normalized_complete":false},"error":{"type":"marshal_failed","message":"failed to marshal normalized Jest payload","details":{}}}`)
		}

		return sdk.NormalizedResult(encodedFallback)
	}

	return sdk.NormalizedResult(encoded)
}

// buildNormalizedPayload produces the in-memory canonical normalized object.
// Keeping this separate from Normalize makes tests more direct and easier to debug.
func buildNormalizedPayload(input ParseResult) normalizedPayload {
	if input.ParseStatus == ParseStatusParseFailed {
		return normalizedFailurePayload(input.Error)
	}

	// Defensive guard:
	// successful parse status without summary is treated as normalization failure.
	if input.Summary == nil {
		return normalizedFailurePayload(&ParserError{
			Type:    ParserErrorType("schema_mismatch"),
			Message: "parser returned ok status without summary",
			Details: map[string]any{},
		})
	}

	status := canonicalStatusFromSummary(*input.Summary)

	payload := normalizedPayload{
		SchemaVersion: normalizedSchemaVersion,
		Adapter:       AdapterName,
		Status:        status,
		ResultKind:    resultKindTestSuite,
		Summary: &normalizedSummary{
			SuiteTotal:  input.Summary.SuiteTotal,
			SuitePassed: input.Summary.SuitePassed,
			SuiteFailed: input.Summary.SuiteFailed,
			TestTotal:   input.Summary.TestTotal,
			TestPassed:  input.Summary.TestPassed,
			TestFailed:  input.Summary.TestFailed,
			TestSkipped: input.Summary.TestSkipped,
			DurationMs:  input.Summary.DurationMs,
			IssueCount:  input.Summary.TestFailed,
		},
		Evidence: normalizedEvidence{
			RawAvailable:       true,
			NormalizedComplete: true,
		},
	}

	// Adapter-specific details must remain under extensions.jest only.
	if len(input.Failures) > 0 {
		payload.Extensions = &normalizedExtensions{
			Jest: &normalizedJestExtension{
				FailureSummaries: stableFailureSummaries(input.Failures),
			},
		}
	}

	// Preserve warnings under extensions.jest only if present and bounded.
	if len(input.Warnings) > 0 {
		if payload.Extensions == nil {
			payload.Extensions = &normalizedExtensions{
				Jest: &normalizedJestExtension{},
			}
		}
		payload.Extensions.Jest.Warnings = stableWarnings(input.Warnings)
	}

	return payload
}

// canonicalStatusFromSummary computes the normalized status from parsed Jest metrics.
func canonicalStatusFromSummary(summary ParsedSummary) string {
	if summary.TestFailed > 0 {
		return statusFailed
	}
	return statusPass
}

// normalizedFailurePayload converts parser-failure input into the canonical
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
		SchemaVersion: normalizedSchemaVersion,
		Adapter:       AdapterName,
		Status:        statusNormalizationFailed,
		ResultKind:    resultKindTestSuite,
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

// stableFailureSummaries sorts and copies failure summaries to preserve determinism.
func stableFailureSummaries(in []ParsedFailureSummary) []normalizedFailureSummary {
	out := make([]normalizedFailureSummary, 0, len(in))
	for _, f := range in {
		out = append(out, normalizedFailureSummary{
			Suite:   f.Suite,
			Test:    f.Test,
			Message: f.Message,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Suite != out[j].Suite {
			return out[i].Suite < out[j].Suite
		}
		if out[i].Test != out[j].Test {
			return out[i].Test < out[j].Test
		}
		return out[i].Message < out[j].Message
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
// These structs deliberately represent only the bounded canonical result
// required by this task. They avoid carrying raw Jest-native structures.

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
	SuiteTotal  int   `json:"suite_total"`
	SuitePassed int   `json:"suite_passed"`
	SuiteFailed int   `json:"suite_failed"`

	TestTotal   int   `json:"test_total"`
	TestPassed  int   `json:"test_passed"`
	TestFailed  int   `json:"test_failed"`
	TestSkipped int   `json:"test_skipped"`

	DurationMs int64 `json:"duration_ms"`
	IssueCount int   `json:"issue_count"`
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
	Jest *normalizedJestExtension `json:"jest,omitempty"`
}

type normalizedJestExtension struct {
	FailureSummaries []normalizedFailureSummary `json:"failure_summaries,omitempty"`
	Warnings         []string                   `json:"warnings,omitempty"`
}

type normalizedFailureSummary struct {
	Suite   string `json:"suite"`
	Test    string `json:"test"`
	Message string `json:"message"`
}

// ValidateNormalizedPayloadShape performs lightweight internal invariants checks.
//
// This is not a replacement for canonical schema validation from brikbyteos-schema.
// It is a narrow guard that helps keep mapper behavior honest and easier to test.
func ValidateNormalizedPayloadShape(payload normalizedPayload) error {
	if payload.SchemaVersion != normalizedSchemaVersion {
		return fmt.Errorf("unexpected schema_version: %s", payload.SchemaVersion)
	}

	if payload.Adapter != AdapterName {
		return fmt.Errorf("unexpected adapter: %s", payload.Adapter)
	}

	if payload.ResultKind != resultKindTestSuite {
		return fmt.Errorf("unexpected result_kind: %s", payload.ResultKind)
	}

	switch payload.Status {
	case statusPass, statusFailed:
		if payload.Summary == nil {
			return fmt.Errorf("summary is required for non-failure normalized payload")
		}
		if !payload.Evidence.NormalizedComplete {
			return fmt.Errorf("normalized_complete must be true for successful mapping")
		}

	case statusNormalizationFailed:
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