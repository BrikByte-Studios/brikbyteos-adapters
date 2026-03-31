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

// ParserErrorType is the structured parser failure taxonomy.
type ParserErrorType string

const (
	ParserErrorMissingReport  ParserErrorType = "missing_report"
	ParserErrorInvalidJSON    ParserErrorType = "invalid_json"
	ParserErrorSchemaMismatch ParserErrorType = "schema_mismatch"
)

// ParsedFailureSummary is a bounded, deterministic failure summary extracted
// from the raw Jest report.
type ParsedFailureSummary struct {
	Suite   string `json:"suite"`
	Test    string `json:"test"`
	Message string `json:"message"`
}

// ParsedSummary contains the core normalized metrics extracted from Jest.
type ParsedSummary struct {
	TestTotal   int   `json:"test_total"`
	TestPassed  int   `json:"test_passed"`
	TestFailed  int   `json:"test_failed"`
	TestSkipped int   `json:"test_skipped"`
	DurationMs  int64 `json:"duration_ms"`
}

// ParserError represents a structured parser failure.
type ParserError struct {
	Type    ParserErrorType `json:"type"`
	Message string          `json:"message"`
	Details map[string]any  `json:"details,omitempty"`
}

// ParseResult is the adapter-private intermediate parse model.
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

// Parser parses Jest JSON output into a deterministic intermediate model.
type Parser struct{}

// ParseFile parses a Jest JSON report from disk.
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

	return ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary: &ParsedSummary{
			TestTotal:   raw.NumTotalTests,
			TestPassed:  raw.NumPassedTests,
			TestFailed:  raw.NumFailedTests,
			TestSkipped: raw.NumPendingTests,
			DurationMs:  raw.durationMS(),
		},
		Failures: extractFailureSummaries(raw.TestResults),
		Warnings: []string{},
	}
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
type jestRawReport struct {
	Success bool `json:"success"`

	NumTotalTests   int `json:"numTotalTests"`
	NumPassedTests  int `json:"numPassedTests"`
	NumFailedTests  int `json:"numFailedTests"`
	NumPendingTests int `json:"numPendingTests"`

	StartTime   int64               `json:"startTime"`
	TestResults []jestRawTestResult `json:"testResults"`
}

func (r jestRawReport) validate() error {
	if r.NumTotalTests < 0 ||
		r.NumPassedTests < 0 ||
		r.NumFailedTests < 0 ||
		r.NumPendingTests < 0 {
		return fmt.Errorf("schema mismatch: required numeric fields must be non-negative")
	}

	if r.NumPassedTests+r.NumFailedTests+r.NumPendingTests > r.NumTotalTests {
		return fmt.Errorf("schema mismatch: test counts are inconsistent")
	}

	if r.TestResults == nil {
		return fmt.Errorf("schema mismatch: testResults is required")
	}

	return nil
}

// durationMS computes a stable duration from per-suite times.
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

func resolveAssertionName(assertion jestRawAssertion) string {
	if strings.TrimSpace(assertion.FullName) != "" {
		return strings.TrimSpace(assertion.FullName)
	}
	if strings.TrimSpace(assertion.Title) != "" {
		return strings.TrimSpace(assertion.Title)
	}
	return "UNKNOWN_TEST"
}

func firstFailureMessage(messages []string) string {
	if len(messages) == 0 {
		return "Test failed"
	}

	msg := strings.TrimSpace(messages[0])
	if msg == "" {
		return "Test failed"
	}

	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	return "Test failed"
}