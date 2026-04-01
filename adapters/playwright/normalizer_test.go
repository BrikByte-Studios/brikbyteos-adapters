package playwright

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
			TestTotal:   10,
			TestPassed:  10,
			TestFailed:  0,
			TestSkipped: 0,
			TestFlaky:   0,
			DurationMs:  1200,
		},
		Failures: []ParsedFailureSummary{},
		Warnings: []string{},
	}

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypeUI,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusCompleted,
			DurationMs: 1200,
		},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input, raw))

	if out.SchemaVersion != "0.1" {
		t.Fatalf("schema_version = %q, want %q", out.SchemaVersion, "0.1")
	}
	if out.Adapter.Name != AdapterName {
		t.Fatalf("adapter.name = %q, want %q", out.Adapter.Name, AdapterName)
	}
	if out.Adapter.Type != string(sdk.AdapterTypeUI) {
		t.Fatalf("adapter.type = %q, want %q", out.Adapter.Type, string(sdk.AdapterTypeUI))
	}
	if out.Execution.Status != "completed" {
		t.Fatalf("execution.status = %q, want %q", out.Execution.Status, "completed")
	}
	if out.Execution.DurationMs != 1200 {
		t.Fatalf("execution.duration_ms = %d, want %d", out.Execution.DurationMs, 1200)
	}
	if out.Summary.Status != "passed" {
		t.Fatalf("summary.status = %q, want %q", out.Summary.Status, "passed")
	}
	if out.Summary.Total != 10 {
		t.Fatalf("summary.total = %d, want %d", out.Summary.Total, 10)
	}
	if out.Summary.Passed != 10 {
		t.Fatalf("summary.passed = %d, want %d", out.Summary.Passed, 10)
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

func TestNormalizer_Normalize_FailedTests(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			TestTotal:   10,
			TestPassed:  8,
			TestFailed:  2,
			TestSkipped: 0,
			TestFlaky:   0,
			DurationMs:  1800,
		},
		Failures: []ParsedFailureSummary{
			{Suite: "tests/auth/login.spec.ts", Test: "rejects invalid token", Message: "Expected 401"},
			{Suite: "tests/auth/login.spec.ts", Test: "rejects expired token", Message: "Expected 401"},
		},
		Warnings: []string{},
	}

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypeUI,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusFailed,
			DurationMs: 1800,
		},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input, raw))

	if out.Execution.Status != "failed" {
		t.Fatalf("execution.status = %q, want %q", out.Execution.Status, "failed")
	}
	if out.Summary.Status != "failed" {
		t.Fatalf("summary.status = %q, want %q", out.Summary.Status, "failed")
	}
	if out.Summary.Total != 10 {
		t.Fatalf("summary.total = %d, want %d", out.Summary.Total, 10)
	}
	if out.Summary.Passed != 8 {
		t.Fatalf("summary.passed = %d, want %d", out.Summary.Passed, 8)
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
			Message: "Malformed Playwright report JSON",
			Details: map[string]any{},
		},
	}

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypeUI,
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
			TestTotal:   2,
			TestPassed:  1,
			TestFailed:  1,
			TestSkipped: 0,
			TestFlaky:   0,
			DurationMs:  200,
		},
		Failures: []ParsedFailureSummary{
			{Suite: "b.spec.ts", Test: "test b", Message: "msg b"},
			{Suite: "a.spec.ts", Test: "test a", Message: "msg a"},
		},
		Warnings: []string{"z-warning", "a-warning"},
	}

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypeUI,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusFailed,
			DurationMs: 200,
		},
	}

	a := decodeNormalized(t, Normalizer{}.Normalize(input, raw))
	b := decodeNormalized(t, Normalizer{}.Normalize(input, raw))

	if !reflect.DeepEqual(a, b) {
		t.Fatal("expected deterministic normalized output")
	}
}

func TestNormalizer_NoTopLevelPlaywrightLeakage(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			TestTotal:   1,
			TestPassed:  1,
			TestFailed:  0,
			TestSkipped: 0,
			TestFlaky:   0,
			DurationMs:  50,
		},
		Failures: []ParsedFailureSummary{},
		Warnings: []string{},
	}

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypeUI,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusCompleted,
			DurationMs: 50,
		},
	}

	normalized := Normalizer{}.Normalize(input, raw)

	var generic map[string]any
	if err := json.Unmarshal(normalized, &generic); err != nil {
		t.Fatalf("unmarshal normalized payload: %v", err)
	}

	for _, forbidden := range []string{
		"failure_summaries",
		"warnings",
		"suites",
		"specs",
		"results",
		"result_kind",
		"raw_available",
		"normalized_complete",
		"error",
	} {
		if _, exists := generic[forbidden]; exists {
			t.Fatalf("unexpected top-level Playwright-specific field leakage: %s", forbidden)
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