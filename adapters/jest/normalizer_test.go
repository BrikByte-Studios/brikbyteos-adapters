package jest

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNormalizer_Normalize_Pass(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			SuiteTotal:  2,
			SuitePassed: 2,
			SuiteFailed: 0,
			TestTotal:   10,
			TestPassed:  10,
			TestFailed:  0,
			TestSkipped: 0,
			DurationMs:  1200,
		},
		Failures: []ParsedFailureSummary{},
		Warnings: []string{},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input))

	if out.Status != statusPass {
		t.Fatalf("expected status=%q, got %q", statusPass, out.Status)
	}
	if out.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if out.Summary.IssueCount != 0 {
		t.Fatalf("expected issue_count=0, got %d", out.Summary.IssueCount)
	}
	if out.Extensions != nil && out.Extensions.Jest != nil && len(out.Extensions.Jest.FailureSummaries) > 0 {
		t.Fatal("did not expect failure summaries for passing payload")
	}
	assertValidNormalizedShape(t, out)
}

func TestNormalizer_Normalize_FailedTests(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			SuiteTotal:  2,
			SuitePassed: 1,
			SuiteFailed: 1,
			TestTotal:   10,
			TestPassed:  8,
			TestFailed:  2,
			TestSkipped: 0,
			DurationMs:  1800,
		},
		Failures: []ParsedFailureSummary{
			{Suite: "auth/login.test.ts", Test: "rejects invalid token", Message: "Expected 401"},
			{Suite: "auth/login.test.ts", Test: "rejects expired token", Message: "Expected 401"},
		},
		Warnings: []string{},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input))

	if out.Status != statusFailed {
		t.Fatalf("expected status=%q, got %q", statusFailed, out.Status)
	}
	if out.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if out.Summary.IssueCount != out.Summary.TestFailed {
		t.Fatalf("expected issue_count=%d, got %d", out.Summary.TestFailed, out.Summary.IssueCount)
	}
	if out.Extensions == nil || out.Extensions.Jest == nil {
		t.Fatal("expected extensions.jest to be present")
	}
	if len(out.Extensions.Jest.FailureSummaries) != 2 {
		t.Fatalf("expected 2 failure summaries, got %d", len(out.Extensions.Jest.FailureSummaries))
	}
	assertValidNormalizedShape(t, out)
}

func TestNormalizer_Normalize_ParserFailure(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusParseFailed,
		Error: &ParserError{
			Type:    ParserErrorInvalidJSON,
			Message: "Malformed Jest report JSON",
			Details: map[string]any{},
		},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input))

	if out.Status != statusNormalizationFailed {
		t.Fatalf("expected status=%q, got %q", statusNormalizationFailed, out.Status)
	}
	if out.Summary != nil {
		t.Fatal("expected nil summary on parser-failure normalization path")
	}
	if out.Error == nil {
		t.Fatal("expected structured error on normalization_failed payload")
	}
	if out.Evidence.NormalizedComplete {
		t.Fatal("expected normalized_complete=false on normalization_failed payload")
	}
	assertValidNormalizedShape(t, out)
}

func TestNormalizer_IsDeterministic(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			SuiteTotal:  1,
			SuitePassed: 0,
			SuiteFailed: 1,
			TestTotal:   2,
			TestPassed:  1,
			TestFailed:  1,
			TestSkipped: 0,
			DurationMs:  200,
		},
		Failures: []ParsedFailureSummary{
			{Suite: "b.test.ts", Test: "test b", Message: "msg b"},
			{Suite: "a.test.ts", Test: "test a", Message: "msg a"},
		},
		Warnings: []string{"z-warning", "a-warning"},
	}

	a := decodeNormalized(t, Normalizer{}.Normalize(input))
	b := decodeNormalized(t, Normalizer{}.Normalize(input))

	if !reflect.DeepEqual(a, b) {
		t.Fatalf("expected deterministic normalized output")
	}
	assertValidNormalizedShape(t, a)
}

func TestNormalizer_NoTopLevelJestLeakage(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			SuiteTotal:  1,
			SuitePassed: 1,
			SuiteFailed: 0,
			TestTotal:   1,
			TestPassed:  1,
			TestFailed:  0,
			TestSkipped: 0,
			DurationMs:  50,
		},
		Failures: []ParsedFailureSummary{},
		Warnings: []string{},
	}

	raw := Normalizer{}.Normalize(input)

	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("unmarshal generic payload: %v", err)
	}

	// Assert that known Jest-specific details are not promoted to the top level.
	for _, forbidden := range []string{"failure_summaries", "warnings", "testResults", "assertionResults"} {
		if _, exists := generic[forbidden]; exists {
			t.Fatalf("unexpected top-level Jest-specific field leakage: %s", forbidden)
		}
	}
}

func decodeNormalized(t *testing.T, raw []byte) normalizedPayload {
	t.Helper()

	var out normalizedPayload
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal normalized payload: %v", err)
	}
	return out
}

func assertValidNormalizedShape(t *testing.T, payload normalizedPayload) {
	t.Helper()

	if err := ValidateNormalizedPayloadShape(payload); err != nil {
		t.Fatalf("invalid normalized payload shape: %v", err)
	}
}
