package jest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParser_ParseBytes_PassingReport(t *testing.T) {
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
		t.Fatalf("expected test_failed=0, got %d", result.Summary.TestFailed)
	}
	if len(result.Failures) != 0 {
		t.Fatalf("expected no failures, got %d", len(result.Failures))
	}
}

func TestParser_ParseBytes_FailingReport(t *testing.T) {
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

func TestParser_ParseBytes_SkippedReport(t *testing.T) {
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

func TestParser_ParseFile_MissingReport(t *testing.T) {
	t.Parallel()

	result := (Parser{}).ParseFile(filepath.Join(t.TempDir(), "missing.report.json"))

	if result.ParseStatus != ParseStatusParseFailed {
		t.Fatalf("expected parse_failed, got %q", result.ParseStatus)
	}
	if result.Error == nil {
		t.Fatal("expected structured parser error")
	}
	if result.Error.Type != ParserErrorMissingReport {
		t.Fatalf("expected error type %q, got %q", ParserErrorMissingReport, result.Error.Type)
	}
}

func TestParser_ParseBytes_MalformedJSON(t *testing.T) {
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
		t.Fatalf("expected error type %q, got %q", ParserErrorInvalidJSON, result.Error.Type)
	}
}

func TestParser_ParseBytes_MissingRequiredField(t *testing.T) {
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
		t.Fatalf("expected error type %q, got %q", ParserErrorSchemaMismatch, result.Error.Type)
	}
}

func TestParser_IsDeterministic(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/fail.report.json")

	a := (Parser{}).ParseBytes(raw)
	b := (Parser{}).ParseBytes(raw)

	if !reflect.DeepEqual(a, b) {
		encodedA, _ := json.MarshalIndent(a, "", "  ")
		encodedB, _ := json.MarshalIndent(b, "", "  ")
		t.Fatalf("expected deterministic output\nA=%s\nB=%s", string(encodedA), string(encodedB))
	}
}

func loadFixtureBytes(t *testing.T, rel string) []byte {
	t.Helper()

	data, err := os.ReadFile(rel)
	if err != nil {
		t.Fatalf("read fixture %q: %v", rel, err)
	}
	return data
}