package trivy

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
			VulnerabilityTotal: 0,
			SeverityCounts: SeverityCounts{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
				Unknown:  0,
			},
			MisconfigTotal: nil,
		},
		Target: &ParsedTarget{
			Name: "services/api",
			Type: "filesystem",
		},
		Findings: []ParsedFindingSummary{},
		Warnings: []string{},
	}

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypeSecurity,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusCompleted,
			DurationMs: 0,
		},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input, raw))

	if out.SchemaVersion != "0.1" {
		t.Fatalf("schema_version = %q, want %q", out.SchemaVersion, "0.1")
	}
	if out.Adapter.Name != AdapterName {
		t.Fatalf("adapter.name = %q, want %q", out.Adapter.Name, AdapterName)
	}
	if out.Adapter.Type != string(sdk.AdapterTypeSecurity) {
		t.Fatalf("adapter.type = %q, want %q", out.Adapter.Type, string(sdk.AdapterTypeSecurity))
	}
	if out.Execution.Status != "completed" {
		t.Fatalf("execution.status = %q, want %q", out.Execution.Status, "completed")
	}
	if out.Summary.Status != "passed" {
		t.Fatalf("summary.status = %q, want %q", out.Summary.Status, "passed")
	}
	if out.Summary.Total != 0 {
		t.Fatalf("summary.total = %d, want %d", out.Summary.Total, 0)
	}
	if out.Summary.Passed != 0 {
		t.Fatalf("summary.passed = %d, want %d", out.Summary.Passed, 0)
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

func TestNormalizer_Normalize_FailedFindings(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			VulnerabilityTotal: 3,
			SeverityCounts: SeverityCounts{
				Critical: 1,
				High:     1,
				Medium:   1,
				Low:      0,
				Unknown:  0,
			},
			MisconfigTotal: nil,
		},
		Target: &ParsedTarget{
			Name: "services/api",
			Type: "filesystem",
		},
		Findings: []ParsedFindingSummary{
			{
				Target:           "services/api/package-lock.json",
				Severity:         "CRITICAL",
				ID:               "CVE-2024-0001",
				Package:          "openssl",
				InstalledVersion: "1.1.1",
				FixedVersion:     "1.1.2",
				Title:            "Example critical vulnerability",
			},
			{
				Target:           "services/api/package-lock.json",
				Severity:         "HIGH",
				ID:               "CVE-2024-0002",
				Package:          "lodash",
				InstalledVersion: "4.17.20",
				FixedVersion:     "4.17.21",
				Title:            "Example high vulnerability",
			},
		},
		Warnings: []string{},
	}

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypeSecurity,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusCompleted,
			DurationMs: 0,
		},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input, raw))

	if out.Execution.Status != "completed" {
		t.Fatalf("execution.status = %q, want %q", out.Execution.Status, "completed")
	}
	if out.Summary.Status != "failed" {
		t.Fatalf("summary.status = %q, want %q", out.Summary.Status, "failed")
	}
	if out.Summary.Total != 3 {
		t.Fatalf("summary.total = %d, want %d", out.Summary.Total, 3)
	}
	if out.Summary.Passed != 1 {
		t.Fatalf("summary.passed = %d, want %d", out.Summary.Passed, 1)
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

func TestNormalizer_Normalize_WithOnlyMediumAndLowFindings_IsPassed(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			VulnerabilityTotal: 2,
			SeverityCounts: SeverityCounts{
				Critical: 0,
				High:     0,
				Medium:   1,
				Low:      1,
				Unknown:  0,
			},
			MisconfigTotal: nil,
		},
		Target:   &ParsedTarget{Name: "services/api", Type: "filesystem"},
		Findings: []ParsedFindingSummary{},
		Warnings: []string{},
	}

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypeSecurity,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusCompleted,
			DurationMs: 0,
		},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input, raw))

	if out.Summary.Status != "passed" {
		t.Fatalf("summary.status = %q, want %q", out.Summary.Status, "passed")
	}
	if out.Summary.Total != 2 {
		t.Fatalf("summary.total = %d, want %d", out.Summary.Total, 2)
	}
	if out.Summary.Passed != 2 {
		t.Fatalf("summary.passed = %d, want %d", out.Summary.Passed, 2)
	}
	if out.Summary.Failed != 0 {
		t.Fatalf("summary.failed = %d, want %d", out.Summary.Failed, 0)
	}
}

func TestNormalizer_Normalize_ParserFailure(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusParseFailed,
		Error: &ParserError{
			Type:    ParserErrorInvalidJSON,
			Message: "Malformed Trivy JSON report",
			Details: map[string]any{},
		},
	}

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypeSecurity,
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
			VulnerabilityTotal: 2,
			SeverityCounts: SeverityCounts{
				Critical: 1,
				High:     1,
				Medium:   0,
				Low:      0,
				Unknown:  0,
			},
			MisconfigTotal: nil,
		},
		Target: &ParsedTarget{
			Name: "services/api",
			Type: "filesystem",
		},
		Findings: []ParsedFindingSummary{
			{
				Target:           "z-target",
				Severity:         "HIGH",
				ID:               "CVE-Z",
				Package:          "zpkg",
				InstalledVersion: "1",
				FixedVersion:     "2",
				Title:            "z issue",
			},
			{
				Target:           "a-target",
				Severity:         "CRITICAL",
				ID:               "CVE-A",
				Package:          "apkg",
				InstalledVersion: "1",
				FixedVersion:     "2",
				Title:            "a issue",
			},
		},
		Warnings: []string{"z-warning", "a-warning"},
	}

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypeSecurity,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusCompleted,
			DurationMs: 0,
		},
	}

	a := decodeNormalized(t, Normalizer{}.Normalize(input, raw))
	b := decodeNormalized(t, Normalizer{}.Normalize(input, raw))

	if !reflect.DeepEqual(a, b) {
		t.Fatal("expected deterministic normalized output")
	}
}

func TestNormalizer_NoTopLevelTrivyLeakage(t *testing.T) {
	t.Parallel()

	input := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			VulnerabilityTotal: 1,
			SeverityCounts: SeverityCounts{
				Critical: 1,
				High:     0,
				Medium:   0,
				Low:      0,
				Unknown:  0,
			},
			MisconfigTotal: nil,
		},
		Target: &ParsedTarget{
			Name: "services/api",
			Type: "filesystem",
		},
		Findings: []ParsedFindingSummary{
			{
				Target:           "services/api/package-lock.json",
				Severity:         "CRITICAL",
				ID:               "CVE-2024-0001",
				Package:          "openssl",
				InstalledVersion: "1.1.1",
				FixedVersion:     "1.1.2",
				Title:            "Example critical vulnerability",
			},
		},
		Warnings: []string{},
	}

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    AdapterName,
		AdapterType:    sdk.AdapterTypeSecurity,
		AdapterVersion: "UNKNOWN",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusCompleted,
			DurationMs: 0,
		},
	}

	normalized := Normalizer{}.Normalize(input, raw)

	var generic map[string]any
	if err := json.Unmarshal(normalized, &generic); err != nil {
		t.Fatalf("unmarshal normalized payload: %v", err)
	}

	for _, forbidden := range []string{
		"Results",
		"Vulnerabilities",
		"critical_high_findings",
		"target",
		"result_kind",
		"raw_available",
		"normalized_complete",
		"error",
	} {
		if _, exists := generic[forbidden]; exists {
			t.Fatalf("unexpected top-level Trivy-specific field leakage: %s", forbidden)
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