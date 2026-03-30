package playwright

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParser_ParseBytes_PassingFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/pass.report.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.TestFailed != 0 {
		t.Fatalf("expected no failed tests, got %d", result.Summary.TestFailed)
	}
	if len(result.Failures) != 0 {
		t.Fatalf("expected no failure summaries, got %d", len(result.Failures))
	}
}

func TestParser_ParseBytes_FailingFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/fail.report.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.TestFailed == 0 {
		t.Fatal("expected failed tests to be extracted")
	}
	if len(result.Failures) == 0 {
		t.Fatal("expected bounded failure summaries")
	}
}

func TestParser_ParseBytes_SkippedFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/skipped.report.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.TestSkipped == 0 {
		t.Fatal("expected skipped tests to be extracted")
	}
}

func TestParser_ParseBytes_FlakyFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/flaky.report.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.TestFlaky == 0 {
		t.Fatal("expected flaky tests to be extracted")
	}
}

func TestParser_ParseBytes_MinimalFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/minimal.report.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.TestTotal != 0 {
		t.Fatalf("expected zero tests, got %d", result.Summary.TestTotal)
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

	raw := loadFixtureBytes(t, "testdata/raw/fail.report.json")

	a := (Parser{}).ParseBytes(raw)
	b := (Parser{}).ParseBytes(raw)

	if !reflect.DeepEqual(a, b) {
		t.Fatal("expected deterministic parser output for identical input")
	}
}
