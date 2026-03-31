package trivy

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

	out := decodeNormalized(t, Normalizer{}.Normalize(input))

	if out.Status != trivyStatusPass {
		t.Fatalf("expected status=%q, got %q", trivyStatusPass, out.Status)
	}
	if out.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if out.Summary.IssueCount != 0 {
		t.Fatalf("expected issue_count=0, got %d", out.Summary.IssueCount)
	}
	assertValidNormalizedShape(t, out)
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

	out := decodeNormalized(t, Normalizer{}.Normalize(input))

	if out.Status != trivyStatusFailed {
		t.Fatalf("expected status=%q, got %q", trivyStatusFailed, out.Status)
	}
	if out.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if out.Summary.IssueCount != out.Summary.VulnerabilityTotal {
		t.Fatalf("expected issue_count=%d, got %d", out.Summary.VulnerabilityTotal, out.Summary.IssueCount)
	}
	if out.Extensions == nil || out.Extensions.Trivy == nil {
		t.Fatal("expected extensions.trivy to be present")
	}
	if len(out.Extensions.Trivy.CriticalHighFindings) != 2 {
		t.Fatalf("expected 2 bounded findings, got %d", len(out.Extensions.Trivy.CriticalHighFindings))
	}
	assertValidNormalizedShape(t, out)
}

func TestNormalizer_Normalize_WithMisconfigTotal(t *testing.T) {
	t.Parallel()

	misconfigTotal := 2

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
			MisconfigTotal: &misconfigTotal,
		},
		Target: &ParsedTarget{
			Name: "infra",
			Type: "filesystem",
		},
		Findings: []ParsedFindingSummary{},
		Warnings: []string{},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input))

	if out.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if out.Summary.MisconfigTotal == nil {
		t.Fatal("expected misconfig_total to be present")
	}
	if *out.Summary.MisconfigTotal != 2 {
		t.Fatalf("expected misconfig_total=2, got %d", *out.Summary.MisconfigTotal)
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
			Message: "Malformed Trivy JSON report",
			Details: map[string]any{},
		},
	}

	out := decodeNormalized(t, Normalizer{}.Normalize(input))

	if out.Status != trivyStatusNormalizationFailed {
		t.Fatalf("expected status=%q, got %q", trivyStatusNormalizationFailed, out.Status)
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

	a := decodeNormalized(t, Normalizer{}.Normalize(input))
	b := decodeNormalized(t, Normalizer{}.Normalize(input))

	if !reflect.DeepEqual(a, b) {
		t.Fatalf("expected deterministic normalized output")
	}
	assertValidNormalizedShape(t, a)
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

	raw := Normalizer{}.Normalize(input)

	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("unmarshal generic payload: %v", err)
	}

	for _, forbidden := range []string{"Results", "Vulnerabilities", "critical_high_findings", "target"} {
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

func assertValidNormalizedShape(t *testing.T, payload normalizedPayload) {
	t.Helper()

	if err := ValidateNormalizedPayloadShape(payload); err != nil {
		t.Fatalf("invalid normalized payload shape: %v", err)
	}
}
