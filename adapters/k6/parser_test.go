package k6

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParser_ParseBytes_PassFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/pass.summary.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.RequestTotal != 1000 {
		t.Fatalf("expected request_total=1000, got %d", result.Summary.RequestTotal)
	}
	if result.Summary.RequestFailed != 0 {
		t.Fatalf("expected request_failed=0, got %d", result.Summary.RequestFailed)
	}
	if result.Summary.DurationMs != 30000 {
		t.Fatalf("expected duration_ms=30000, got %d", result.Summary.DurationMs)
	}
}

func TestParser_ParseBytes_ThresholdFailFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/threshold-fail.summary.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if len(result.Thresholds) == 0 {
		t.Fatal("expected threshold summaries")
	}

	foundFail := false
	for _, th := range result.Thresholds {
		if th.Status == ThresholdStatusFail {
			foundFail = true
			break
		}
	}
	if !foundFail {
		t.Fatal("expected at least one failing threshold summary")
	}
}

func TestParser_ParseBytes_WithPercentilesFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/with-percentiles.summary.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.LatencyP95Ms == nil {
		t.Fatal("expected p95 latency")
	}
	if result.Summary.LatencyP99Ms == nil {
		t.Fatal("expected p99 latency")
	}
}

func TestParser_ParseBytes_WithoutPercentilesFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/without-percentiles.summary.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.LatencyP95Ms != nil {
		t.Fatal("expected nil p95 latency when absent")
	}
	if result.Summary.LatencyP99Ms != nil {
		t.Fatal("expected nil p99 latency when absent")
	}
}

func TestParser_ParseBytes_MinimalFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/minimal.summary.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("expected parse_status=%q, got %q", ParseStatusOK, result.ParseStatus)
	}
	if result.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if result.Summary.RequestTotal != 0 {
		t.Fatalf("expected request_total=0, got %d", result.Summary.RequestTotal)
	}
}

func TestParser_ParseBytes_MalformedFixture(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/malformed.summary.json")
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

	raw := loadFixtureBytes(t, "testdata/raw/missing-required-field.summary.json")
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

func TestParser_ParseFile_MissingSummary(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.summary.json")
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

	raw := loadFixtureBytes(t, "testdata/raw/threshold-fail.summary.json")

	a := (Parser{}).ParseBytes(raw)
	b := (Parser{}).ParseBytes(raw)

	if !reflect.DeepEqual(a, b) {
		t.Fatal("expected deterministic parser output for identical input")
	}
}

func TestParser_ThresholdFailuresAreNotParserFailures(t *testing.T) {
	t.Parallel()

	raw := loadFixtureBytes(t, "testdata/raw/threshold-fail.summary.json")
	result := (Parser{}).ParseBytes(raw)

	if result.ParseStatus != ParseStatusOK {
		t.Fatalf("threshold failures must still parse successfully, got %q", result.ParseStatus)
	}
}