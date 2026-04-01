package k6

import (
	"encoding/json"
	"reflect"
	"testing"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
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

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypePerformance,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusCompleted,
			DurationMs: 30000,
		},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input, raw))

	if out.SchemaVersion != "0.1" {
		t.Fatalf("schema_version = %q, want %q", out.SchemaVersion, "0.1")
	}
	if out.Adapter.Name != AdapterName {
		t.Fatalf("adapter.name = %q, want %q", out.Adapter.Name, AdapterName)
	}
	if out.Adapter.Type != string(sdk.AdapterTypePerformance) {
		t.Fatalf("adapter.type = %q, want %q", out.Adapter.Type, string(sdk.AdapterTypePerformance))
	}
	if out.Execution.Status != "completed" {
		t.Fatalf("execution.status = %q, want %q", out.Execution.Status, "completed")
	}
	if out.Execution.DurationMs != 30000 {
		t.Fatalf("execution.duration_ms = %d, want %d", out.Execution.DurationMs, 30000)
	}
	if out.Summary.Status != "passed" {
		t.Fatalf("summary.status = %q, want %q", out.Summary.Status, "passed")
	}
	if out.Summary.Total != 1 {
		t.Fatalf("summary.total = %d, want %d", out.Summary.Total, 1)
	}
	if out.Summary.Passed != 1 {
		t.Fatalf("summary.passed = %d, want %d", out.Summary.Passed, 1)
	}
	if out.Summary.Failed != 0 {
		t.Fatalf("summary.failed = %d, want %d", out.Summary.Failed, 0)
	}
	if out.Summary.Skipped != 0 {
		t.Fatalf("summary.skipped = %d, want %d", out.Summary.Skipped, 0)
	}
	if !out.Evidence.Complete {
		t.Fatal("evidence.complete = false, want true")
	}
	if len(out.Evidence.Issues) != 0 {
		t.Fatalf("evidence.issues = %v, want empty", out.Evidence.Issues)
	}
	if out.Extensions.AdapterSpecific == nil {
		t.Fatal("extensions.adapter_specific = nil, want initialized map")
	}
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

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypePerformance,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusFailed,
			DurationMs: 30000,
		},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input, raw))

	if out.Execution.Status != "failed" {
		t.Fatalf("execution.status = %q, want %q", out.Execution.Status, "failed")
	}
	if out.Summary.Status != "failed" {
		t.Fatalf("summary.status = %q, want %q", out.Summary.Status, "failed")
	}
	if out.Summary.Total != 2 {
		t.Fatalf("summary.total = %d, want %d", out.Summary.Total, 2)
	}
	if out.Summary.Passed != 0 {
		t.Fatalf("summary.passed = %d, want %d", out.Summary.Passed, 0)
	}
	if out.Summary.Failed != 2 {
		t.Fatalf("summary.failed = %d, want %d", out.Summary.Failed, 2)
	}
	if out.Summary.Skipped != 0 {
		t.Fatalf("summary.skipped = %d, want %d", out.Summary.Skipped, 0)
	}
	if !out.Evidence.Complete {
		t.Fatal("evidence.complete = false, want true")
	}
	if len(out.Evidence.Issues) != 0 {
		t.Fatalf("evidence.issues = %v, want empty", out.Evidence.Issues)
	}
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

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypePerformance,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusFailed,
			DurationMs: 0,
		},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input, raw))

	if out.Execution.Status != "failed" {
		t.Fatalf("execution.status = %q, want %q", out.Execution.Status, "failed")
	}
	if out.Summary.Status != "unknown" {
		t.Fatalf("summary.status = %q, want %q", out.Summary.Status, "unknown")
	}
	if out.Evidence.Complete {
		t.Fatal("evidence.complete = true, want false")
	}
	if len(out.Evidence.Issues) != 1 {
		t.Fatalf("len(evidence.issues) = %d, want %d", len(out.Evidence.Issues), 1)
	}
	if out.Evidence.Issues[0].Code != "INVALID_TOOL_OUTPUT" {
		t.Fatalf("issue.code = %q, want %q", out.Evidence.Issues[0].Code, "INVALID_TOOL_OUTPUT")
	}
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

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypePerformance,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusFailed,
			DurationMs: 30000,
		},
	}

	a := decodeNormalized(t, Normalizer{}.Normalize(input, raw))
	b := decodeNormalized(t, Normalizer{}.Normalize(input, raw))

	if !reflect.DeepEqual(a, b) {
		t.Fatal("expected deterministic normalized output")
	}
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

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypePerformance,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusCompleted,
			DurationMs: 30000,
		},
	}

	normalized := Normalizer{}.Normalize(input, raw)

	var generic map[string]any
	if err := json.Unmarshal(normalized, &generic); err != nil {
		t.Fatalf("unmarshal normalized payload: %v", err)
	}

	for _, forbidden := range []string{
		"thresholds",
		"metrics",
		"values",
		"thresholds_map",
		"result_kind",
		"raw_available",
		"normalized_complete",
		"error",
	} {
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