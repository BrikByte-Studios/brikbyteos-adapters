package trivy

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParser_ParseBytes_PassFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/pass.report.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.VulnerabilityTotal != 3 {
		t.Fatalf("expected vulnerability_total=3, got %d", result.Summary.VulnerabilityTotal)
	}
	if result.Summary.SeverityCounts.Critical != 1 {
		t.Fatalf("expected critical=1, got %d", result.Summary.SeverityCounts.Critical)
	}
	if result.Summary.SeverityCounts.High != 1 {
		t.Fatalf("expected high=1, got %d", result.Summary.SeverityCounts.High)
	}
	if result.Target == nil {
		t.Fatal("expected target to be extracted")
	}
	if result.Target.Name == "" || result.Target.Type == "" {
		t.Fatalf("expected bounded target fields, got %+v", result.Target)
	}
}

func TestParser_ParseBytes_NoVulnsFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/no-vulns.report.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.VulnerabilityTotal != 0 {
		t.Fatalf("expected vulnerability_total=0, got %d", result.Summary.VulnerabilityTotal)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no bounded findings summaries, got %d", len(result.Findings))
	}
}

func TestParser_ParseBytes_WithMisconfigFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/with-misconfig.report.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.MisconfigTotal == nil {
		t.Fatal("expected misconfig_total to be present")
	}
	if *result.Summary.MisconfigTotal != 2 {
		t.Fatalf("expected misconfig_total=2, got %d", *result.Summary.MisconfigTotal)
	}
}

func TestParser_ParseBytes_MalformedFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/malformed.report.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusParseFailed {
		t.Fatalf("expected parse_failed, got %q", result.ParseStatus)
	}
	if result.Error == nil {
		t.Fatal("expected structured parser error")
	}
	if result.Error.Type != ParserErrorInvalidJSON {
		t.Fatalf("expected error type=%q, got %q", ParserErrorInvalidJSON, result.Error.Type)
	}
}

func TestParser_ParseBytes_MissingRequiredFieldFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/missing-required-field.report.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusParseFailed {
		t.Fatalf("expected parse_failed, got %q", result.ParseStatus)
	}
	if result.Error == nil {
		t.Fatal("expected structured parser error")
	}
	if result.Error.Type != ParserErrorSchemaMismatch {
		t.Fatalf("expected error type=%q, got %q", ParserErrorSchemaMismatch, result.Error.Type)
	}
}

func TestParser_ParseFile_MissingReport(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.report.json")
	result := (Parser{}).ParseFile(path)

	if result.ParseStatus != ParseStatusParseFailed {
		t.Fatalf("expected parse_failed, got %q", result.ParseStatus)
	}
	if result.Error == nil {
		t.Fatal("expected structured parser error")
	}
	if result.Error.Type != ParserErrorMissingReport {
		t.Fatalf("expected error type=%q, got %q", ParserErrorMissingReport, result.Error.Type)
	}
}

func TestParser_IsDeterministic(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/pass.report.json")

	a := (Parser{}).ParseBytes(raw)
	b := (Parser{}).ParseBytes(raw)

	if !reflect.DeepEqual(a, b) {
		t.Fatal("expected deterministic parser output for identical input")
	}
}

func TestParser_VulnerabilitiesAreNotParserFailures(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/pass.report.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("vulnerability findings must still parse successfully, got %q", result.ParseStatus)
	}
}

func TestParser_FindsOnlyCriticalAndHighInBoundedFindings(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/pass.report.json")
	result := (Parser{}).ParseBytes(raw)

	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 bounded critical/high findings, got %d", len(result.Findings))
	}
	for _, finding := range result.Findings {
		if finding.Severity != "CRITICAL" && finding.Severity != "HIGH" {
			t.Fatalf("unexpected bounded finding severity: %s", finding.Severity)
		}
	}
}
