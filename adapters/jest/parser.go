package jest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ParseStatus represents the outcome of parsing the raw Jest JSON report.
type ParseStatus string

const (
	ParseStatusOK          ParseStatus = "ok"
	ParseStatusParseFailed ParseStatus = "parse_failed"
)

// ParserErrorType is the canonical structured error taxonomy for Jest parser failures.
type ParserErrorType string

const (
	ParserErrorMissingReport  ParserErrorType = "missing_report"
	ParserErrorInvalidJSON    ParserErrorType = "invalid_json"
	ParserErrorSchemaMismatch ParserErrorType = "schema_mismatch"
)

// ParsedFailureSummary is a bounded, deterministic failure summary extracted from the raw report.
//
// Important:
//   - This is not a raw pass-through of Jest internals.
//   - It intentionally excludes unstable or overly verbose data such as full stack traces.
type ParsedFailureSummary struct {
	Suite   string `json:"suite"`
	Test    string `json:"test"`
	Message string `json:"message"`
}

// ParsedSummary contains the required adapter-private summary fields needed downstream.
type ParsedSummary struct {
	SuiteTotal  int `json:"suite_total"`
	SuitePassed int `json:"suite_passed"`
	SuiteFailed int `json:"suite_failed"`

	TestTotal   int `json:"test_total"`
	TestPassed  int `json:"test_passed"`
	TestFailed  int `json:"test_failed"`
	TestSkipped int `json:"test_skipped"`

	DurationMs int64 `json:"duration_ms"`
}

// ParserError is the structured parser failure object.
type ParserError struct {
	Type    ParserErrorType `json:"type"`
	Message string          `json:"message"`
	Details map[string]any  `json:"details,omitempty"`
}

// ParseResult is the adapter-private intermediate parse contract for Jest.
//
// It is intentionally shaped for:
//   - deterministic parser output
//   - clean downstream normalization
//   - structured failure handling
type ParseResult struct {
	Adapter     string                 `json:"adapter"`
	ParseStatus ParseStatus            `json:"parse_status"`
	Summary     *ParsedSummary         `json:"summary,omitempty"`
	Failures    []ParsedFailureSummary `json:"failures,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Error       *ParserError           `json:"error,omitempty"`
}

// Parser reads the canonical Jest JSON report file and converts it into the adapter-private parse model.
type Parser struct{}

// ParseFile parses a Jest report from disk.
//
// Rules:
//   - JSON report file is the only primary structured input
//   - missing or malformed reports return structured parse_failed results
//   - test failures inside the report do not count as parser failures
func (Parser) ParseFile(reportPath string) ParseResult {
	if strings.TrimSpace(reportPath) == "" {
		return parseFailure(
			ParserErrorSchemaMismatch,
			"report path is required",
			map[string]any{},
		)
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parseFailure(
				ParserErrorMissingReport,
				"Jest report file missing",
				map[string]any{"path": reportPath},
			)
		}

		return parseFailure(
			ParserErrorMissingReport,
			"failed to read Jest report file",
			map[string]any{"path": reportPath, "error": err.Error()},
		)
	}

	return (Parser{}).ParseBytes(data)
}

// ParseBytes parses raw Jest JSON bytes.
//
// This function is pure and deterministic for identical input.
func (Parser) ParseBytes(data []byte) ParseResult {
	var raw jestRawReport
	if err := json.Unmarshal(data, &raw); err != nil {
		return parseFailure(
			ParserErrorInvalidJSON,
			"Malformed Jest report JSON",
			map[string]any{"error": err.Error()},
		)
	}

	if err := raw.validate(); err != nil {
		return parseFailure(
			ParserErrorSchemaMismatch,
			err.Error(),
			map[string]any{},
		)
	}

	result := ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			SuiteTotal:  raw.NumTotalTestSuites,
			SuitePassed: raw.NumPassedTestSuites,
			SuiteFailed: raw.NumFailedTestSuites,
			TestTotal:   raw.NumTotalTests,
			TestPassed:  raw.NumPassedTests,
			TestFailed:  raw.NumFailedTests,
			TestSkipped: raw.NumPendingTests,
			DurationMs:  raw.durationMS(),
		},
		Failures: extractFailureSummaries(raw.TestResults),
		Warnings: []string{},
	}

	return result
}

func parseFailure(errorType ParserErrorType, message string, details map[string]any) ParseResult {
	return ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusParseFailed,
		Failures:    []ParsedFailureSummary{},
		Warnings:    []string{},
		Error: &ParserError{
			Type:    errorType,
			Message: message,
			Details: details,
		},
	}
}

// jestRawReport captures only the raw Jest JSON fields needed by the parser.
// Extra fields are tolerated and ignored.
type jestRawReport struct {
	Success bool `json:"success"`

	NumTotalTestSuites  int `json:"numTotalTestSuites"`
	NumPassedTestSuites int `json:"numPassedTestSuites"`
	NumFailedTestSuites int `json:"numFailedTestSuites"`

	NumTotalTests   int `json:"numTotalTests"`
	NumPassedTests  int `json:"numPassedTests"`
	NumFailedTests  int `json:"numFailedTests"`
	NumPendingTests int `json:"numPendingTests"`

	StartTime   int64               `json:"startTime"`
	TestResults []jestRawTestResult `json:"testResults"`
}

// validate enforces strictness on required core fields while remaining tolerant of extra fields.
func (r jestRawReport) validate() error {
	if r.NumTotalTestSuites < 0 ||
		r.NumPassedTestSuites < 0 ||
		r.NumFailedTestSuites < 0 ||
		r.NumTotalTests < 0 ||
		r.NumPassedTests < 0 ||
		r.NumFailedTests < 0 ||
		r.NumPendingTests < 0 {
		return fmt.Errorf("schema mismatch: required numeric fields must be non-negative")
	}

	if r.NumPassedTestSuites+r.NumFailedTestSuites > r.NumTotalTestSuites {
		return fmt.Errorf("schema mismatch: suite counts are inconsistent")
	}

	if r.NumPassedTests+r.NumFailedTests+r.NumPendingTests > r.NumTotalTests {
		return fmt.Errorf("schema mismatch: test counts are inconsistent")
	}

	if r.TestResults == nil {
		return fmt.Errorf("schema mismatch: testResults is required")
	}

	return nil
}

// durationMS computes total execution duration as the sum of per-suite durations.
// This avoids depending on unstable wall-clock timing derived from startTime.
func (r jestRawReport) durationMS() int64 {
	var total int64
	for _, tr := range r.TestResults {
		if tr.EndTime > tr.StartTime && tr.StartTime > 0 {
			total += tr.EndTime - tr.StartTime
		}
	}
	return total
}

type jestRawTestResult struct {
	Name             string             `json:"name"`
	StartTime        int64              `json:"startTime"`
	EndTime          int64              `json:"endTime"`
	AssertionResults []jestRawAssertion `json:"assertionResults"`
}

type jestRawAssertion struct {
	FullName        string   `json:"fullName"`
	Status          string   `json:"status"`
	FailureMessages []string `json:"failureMessages"`
	AncestorTitles  []string `json:"ancestorTitles"`
	Title           string   `json:"title"`
}

// extractFailureSummaries returns bounded, deterministic failure summaries.
//
// Determinism rules:
//   - failure summaries are extracted in stable order
//   - final list is sorted by suite, test, message
func extractFailureSummaries(results []jestRawTestResult) []ParsedFailureSummary {
	failures := make([]ParsedFailureSummary, 0)

	for _, suite := range results {
		for _, assertion := range suite.AssertionResults {
			if strings.ToLower(assertion.Status) != "failed" {
				continue
			}

			failures = append(failures, ParsedFailureSummary{
				Suite:   strings.TrimSpace(suite.Name),
				Test:    resolveAssertionName(assertion),
				Message: firstFailureMessage(assertion.FailureMessages),
			})
		}
	}

	sort.SliceStable(failures, func(i, j int) bool {
		if failures[i].Suite != failures[j].Suite {
			return failures[i].Suite < failures[j].Suite
		}
		if failures[i].Test != failures[j].Test {
			return failures[i].Test < failures[j].Test
		}
		return failures[i].Message < failures[j].Message
	})

	return failures
}

// resolveAssertionName chooses the most useful stable test name for failure summaries.
func resolveAssertionName(assertion jestRawAssertion) string {
	if strings.TrimSpace(assertion.FullName) != "" {
		return strings.TrimSpace(assertion.FullName)
	}
	if strings.TrimSpace(assertion.Title) != "" {
		return strings.TrimSpace(assertion.Title)
	}
	return "UNKNOWN_TEST"
}

// firstFailureMessage extracts a bounded single-line message from Jest failure output.
//
// We intentionally avoid preserving long stack traces in the parser output.
func firstFailureMessage(messages []string) string {
	if len(messages) == 0 {
		return "Test failed"
	}

	msg := strings.TrimSpace(messages[0])
	if msg == "" {
		return "Test failed"
	}

	// Keep only the first non-empty line to avoid noisy stack-trace dumps.
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	return "Test failed"
}
