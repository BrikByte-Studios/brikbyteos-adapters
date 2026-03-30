package playwright

import (
	"encoding/json"
	"fmt"
	"sort"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// Canonical normalized result constants.
//
// These constants are centralized to avoid status drift between the
// Playwright parser, normalizer, tests, and downstream consumers.
const (
	playwrightNormalizedSchemaVersion = "0.1"
	playwrightResultKindTestSuite     = "test_suite"

	playwrightStatusPass                = "pass"
	playwrightStatusFailed              = "failed"
	playwrightStatusNormalizationFailed = "normalization_failed"
)

// Normalizer converts the adapter-private Playwright parse model into the
// canonical BrikByte normalized result.
//
// Design goals:
//   - pure function of parser output
//   - deterministic field mapping
//   - explicit parser-failure handling
//   - strict anti-leak boundary for Playwright-specific details
type Normalizer struct{}

// Normalize converts a Playwright ParseResult into canonical normalized JSON.
//
// Mapping rules:
//   - parse ok + zero failed tests -> pass
//   - parse ok + one or more failed tests -> failed
//   - parse failed -> normalization_failed
//
// issue_count is always mapped from test_failed.
func (Normalizer) Normalize(input ParseResult) sdk.NormalizedResult {
	payload := buildNormalizedPayload(input)

	encoded, err := json.Marshal(payload)
	if err != nil {
		fallback := normalizedFailurePayload(&ParserError{
			Type:    ParserErrorType("marshal_failed"),
			Message: "failed to marshal normalized Playwright payload",
			Details: map[string]any{},
		})

		encodedFallback, fallbackErr := json.Marshal(fallback)
		if fallbackErr != nil {
			return sdk.NormalizedResult(`{"schema_version":"0.1","adapter":"playwright","status":"normalization_failed","result_kind":"test_suite","summary":null,"evidence":{"raw_available":true,"normalized_complete":false},"error":{"type":"marshal_failed","message":"failed to marshal normalized Playwright payload","details":{}}}`)
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

	status := canonicalStatusFromSummary(*input.Summary)

	payload := normalizedPayload{
		SchemaVersion: playwrightNormalizedSchemaVersion,
		Adapter:       AdapterName,
		Status:        status,
		ResultKind:    playwrightResultKindTestSuite,
		Summary: &normalizedSummary{
			TestTotal:   input.Summary.TestTotal,
			TestPassed:  input.Summary.TestPassed,
			TestFailed:  input.Summary.TestFailed,
			TestSkipped: input.Summary.TestSkipped,
			TestFlaky:   input.Summary.TestFlaky,
			DurationMs:  input.Summary.DurationMs,
			IssueCount:  input.Summary.TestFailed,
		},
		Evidence: normalizedEvidence{
			RawAvailable:       true,
			NormalizedComplete: true,
		},
	}

	// All Playwright-specific detail must remain inside extensions.playwright.
	if len(input.Failures) > 0 {
		payload.Extensions = &normalizedExtensions{
			Playwright: &normalizedPlaywrightExtension{
				FailureSummaries: stableFailureSummaries(input.Failures),
			},
		}
	}

	if len(input.Warnings) > 0 {
		if payload.Extensions == nil {
			payload.Extensions = &normalizedExtensions{
				Playwright: &normalizedPlaywrightExtension{},
			}
		}
		payload.Extensions.Playwright.Warnings = stableWarnings(input.Warnings)
	}

	return payload
}

// canonicalStatusFromSummary computes the canonical normalized status.
func canonicalStatusFromSummary(summary ParsedSummary) string {
	if summary.TestFailed > 0 {
		return playwrightStatusFailed
	}
	return playwrightStatusPass
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
		SchemaVersion: playwrightNormalizedSchemaVersion,
		Adapter:       AdapterName,
		Status:        playwrightStatusNormalizationFailed,
		ResultKind:    playwrightResultKindTestSuite,
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

// stableFailureSummaries sorts and copies Playwright failure summaries for determinism.
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
// These structs intentionally represent only the bounded canonical result
// required for this task. They are not a raw pass-through of Playwright.

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
	TestTotal   int   `json:"test_total"`
	TestPassed  int   `json:"test_passed"`
	TestFailed  int   `json:"test_failed"`
	TestSkipped int   `json:"test_skipped"`
	TestFlaky   int   `json:"test_flaky"`
	DurationMs  int64 `json:"duration_ms"`
	IssueCount  int   `json:"issue_count"`
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
	Playwright *normalizedPlaywrightExtension `json:"playwright,omitempty"`
}

type normalizedPlaywrightExtension struct {
	FailureSummaries []normalizedFailureSummary `json:"failure_summaries,omitempty"`
	Warnings         []string                   `json:"warnings,omitempty"`
}

type normalizedFailureSummary struct {
	Suite   string `json:"suite"`
	Test    string `json:"test"`
	Message string `json:"message"`
}

// ValidateNormalizedPayloadShape performs lightweight internal invariant checks.
//
// This is intentionally narrow. It is not a replacement for future stricter
// schema validation from brikbyteos-schema.
func ValidateNormalizedPayloadShape(payload normalizedPayload) error {
	if payload.SchemaVersion != playwrightNormalizedSchemaVersion {
		return fmt.Errorf("unexpected schema_version: %s", payload.SchemaVersion)
	}

	if payload.Adapter != AdapterName {
		return fmt.Errorf("unexpected adapter: %s", payload.Adapter)
	}

	if payload.ResultKind != playwrightResultKindTestSuite {
		return fmt.Errorf("unexpected result_kind: %s", payload.ResultKind)
	}

	switch payload.Status {
	case playwrightStatusPass, playwrightStatusFailed:
		if payload.Summary == nil {
			return fmt.Errorf("summary is required for non-failure normalized payload")
		}
		if !payload.Evidence.NormalizedComplete {
			return fmt.Errorf("normalized_complete must be true for successful mapping")
		}
		if payload.Summary.IssueCount != payload.Summary.TestFailed {
			return fmt.Errorf("issue_count must equal test_failed")
		}

	case playwrightStatusNormalizationFailed:
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