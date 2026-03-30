package k6

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
			RequestTotal:  1000,
			RequestFailed: 0,
			DurationMs:    30000,
		},
		Thresholds: []ParsedThresholdSummary{
			{Name: "http_req_failed:rate<0.01", Status: ThresholdStatusPass},
		},
		Warnings: []string{},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input))

	if out.Status != k6StatusPass {
		t.Fatalf("expected status=%q, got %q", k6StatusPass, out.Status)
	}
	if out.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if out.Summary.IssueCount != 0 {
		t.Fatalf("expected issue_count=0, got %d", out.Summary.IssueCount)
	}
	assertValidNormalizedShape(t, out)
}

func TestNormalizer_Normalize_FailedThresholds(t *testing.T) {
	t.Parallel()

	p95 := 240.0
	p99 := 410.0

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			RequestTotal:  1000,
			RequestFailed: 12,
			LatencyP95Ms:  &p95,
			LatencyP99Ms:  &p99,
			DurationMs:    30000,
		},
		Thresholds: []ParsedThresholdSummary{
			{Name: "http_req_failed:rate<0.01", Status: ThresholdStatusFail},
			{Name: "http_req_duration:p(95)<200", Status: ThresholdStatusFail},
		},
		Warnings: []string{},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input))

	if out.Status != k6StatusFailed {
		t.Fatalf("expected status=%q, got %q", k6StatusFailed, out.Status)
	}
	if out.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if out.Summary.IssueCount != 2 {
		t.Fatalf("expected issue_count=2, got %d", out.Summary.IssueCount)
	}
	if out.Extensions == nil || out.Extensions.K6 == nil {
		t.Fatal("expected extensions.k6 to be present")
	}
	if len(out.Extensions.K6.Thresholds) != 2 {
		t.Fatalf("expected 2 threshold summaries, got %d", len(out.Extensions.K6.Thresholds))
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
			Message: "Malformed k6 summary JSON",
			Details: map[string]any{},
		},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input))

	if out.Status != k6StatusNormalizationFailed {
		t.Fatalf("expected status=%q, got %q", k6StatusNormalizationFailed, out.Status)
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
			RequestTotal:  1000,
			RequestFailed: 12,
			DurationMs:    30000,
		},
		Thresholds: []ParsedThresholdSummary{
			{Name: "z-threshold", Status: ThresholdStatusFail},
			{Name: "a-threshold", Status: ThresholdStatusPass},
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

func TestNormalizer_NoTopLevelK6Leakage(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			RequestTotal:  1000,
			RequestFailed: 0,
			DurationMs:    30000,
		},
		Thresholds: []ParsedThresholdSummary{
			{Name: "http_req_failed:rate<0.01", Status: ThresholdStatusPass},
		},
		Warnings: []string{},
	}

	raw := Normalizer{}.Normalize(input)

	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("unmarshal generic payload: %v", err)
	}

	for _, forbidden := range []string{"thresholds", "metrics", "values", "thresholds_map"} {
		if _, exists := generic[forbidden]; exists {
			t.Fatalf("unexpected top-level k6-specific field leakage: %s", forbidden)
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
