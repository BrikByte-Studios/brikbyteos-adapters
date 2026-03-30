package playwright

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ParseStatus represents the outcome of parsing a canonical Playwright report.
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

// ParsedFailureSummary is a bounded, deterministic failure summary.
//
// Important:
//   - This is intentionally not a raw pass-through of Playwright-native data.
//   - It excludes traces, attachments, screenshots, and stack-trace blobs.
type ParsedFailureSummary struct {
	Suite   string `json:"suite"`
	Test    string `json:"test"`
	Message string `json:"message"`
}

// ParsedSummary contains the minimum canonical summary fields required downstream.
type ParsedSummary struct {
	TestTotal   int   `json:"test_total"`
	TestPassed  int   `json:"test_passed"`
	TestFailed  int   `json:"test_failed"`
	TestFlaky   int   `json:"test_flaky"`
	TestSkipped int   `json:"test_skipped"`
	DurationMs  int64 `json:"duration_ms"`
}

// ParserError is the structured failure object returned when parsing fails.
type ParserError struct {
	Type    ParserErrorType `json:"type"`
	Message string          `json:"message"`
	Details map[string]any  `json:"details,omitempty"`
}

// ParseResult is the adapter-private intermediate parse contract for Playwright.
type ParseResult struct {
	Adapter     string                 `json:"adapter"`
	ParseStatus ParseStatus            `json:"parse_status"`
	Summary     *ParsedSummary         `json:"summary,omitempty"`
	Failures    []ParsedFailureSummary `json:"failures,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Error       *ParserError           `json:"error,omitempty"`
}

// Parser reads the canonical Playwright structured report and converts it
// into the adapter-private parse model.
//
// Design rules:
//   - primary input is the structured report artifact, not stdout
//   - malformed or missing reports return structured parse_failed results
//   - failed tests are valid parse results, not parser failures
type Parser struct{}

// ParseFile parses a Playwright report from disk.
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
				"Playwright report file missing",
				map[string]any{"path": reportPath},
			)
		}

		return parseFailure(
			ParserErrorMissingReport,
			"failed to read Playwright report file",
			map[string]any{"path": reportPath, "error": err.Error()},
		)
	}

	return (Parser{}).ParseBytes(data)
}

// ParseBytes parses raw Playwright report bytes.
//
// This function is pure and deterministic for identical input.
func (Parser) ParseBytes(data []byte) ParseResult {
	var raw playwrightRawReport
	if err := json.Unmarshal(data, &raw); err != nil {
		return parseFailure(
			ParserErrorInvalidJSON,
			"Malformed Playwright report JSON",
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

	summary, failures := raw.toParsed()

	return ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary:     &summary,
		Failures:    failures,
		Warnings:    []string{},
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

// playwrightRawReport is a deliberately minimal subset of a canonical Playwright JSON report.
//
// The parser is strict on required core fields and tolerant of extra fields.
type playwrightRawReport struct {
	Stats  playwrightStats  `json:"stats"`
	Suites []playwrightSuite `json:"suites"`
}

type playwrightStats struct {
	Expected   *int   `json:"expected"`
	Skipped    *int   `json:"skipped"`
	Unexpected *int   `json:"unexpected"`
	Flaky      *int   `json:"flaky"`
	Duration   *int64 `json:"duration"`
}

type playwrightSuite struct {
	Title  string             `json:"title"`
	File   string             `json:"file"`
	Suites []playwrightSuite  `json:"suites"`
	Specs  []playwrightSpec   `json:"specs"`
}

type playwrightSpec struct {
	Title string             `json:"title"`
	Tests []playwrightTest   `json:"tests"`
}

type playwrightTest struct {
	Results []playwrightResult `json:"results"`
}

type playwrightResult struct {
	Status string `json:"status"`
	Error  *playwrightError `json:"error,omitempty"`
}

type playwrightError struct {
	Message string `json:"message"`
}

func (r playwrightRawReport) validate() error {
	if r.Stats.Expected == nil {
		return fmt.Errorf("schema mismatch: stats.expected is required")
	}
	if r.Stats.Skipped == nil {
		return fmt.Errorf("schema mismatch: stats.skipped is required")
	}
	if r.Stats.Unexpected == nil {
		return fmt.Errorf("schema mismatch: stats.unexpected is required")
	}
	if r.Stats.Flaky == nil {
		return fmt.Errorf("schema mismatch: stats.flaky is required")
	}
	if r.Stats.Duration == nil {
		return fmt.Errorf("schema mismatch: stats.duration is required")
	}

	if *r.Stats.Expected < 0 || *r.Stats.Skipped < 0 || *r.Stats.Unexpected < 0 || *r.Stats.Flaky < 0 {
		return fmt.Errorf("schema mismatch: stats fields must be non-negative")
	}
	if *r.Stats.Duration < 0 {
		return fmt.Errorf("schema mismatch: duration must be non-negative")
	}
	if r.Suites == nil {
		return fmt.Errorf("schema mismatch: suites is required")
	}

	return nil
}

// toParsed converts the raw report into the adapter-private summary and failure list.
func (r playwrightRawReport) toParsed() (ParsedSummary, []ParsedFailureSummary) {
	failures := extractFailureSummaries(r.Suites)

	summary := ParsedSummary{
		TestPassed:  *r.Stats.Expected,
		TestSkipped: *r.Stats.Skipped,
		TestFailed:  *r.Stats.Unexpected,
		TestFlaky:   *r.Stats.Flaky,
		DurationMs:  *r.Stats.Duration,
	}
	summary.TestTotal = summary.TestPassed + summary.TestSkipped + summary.TestFailed + summary.TestFlaky

	return summary, failures
}

// extractFailureSummaries walks suites recursively and returns bounded deterministic summaries.
func extractFailureSummaries(suites []playwrightSuite) []ParsedFailureSummary {
	out := make([]ParsedFailureSummary, 0)

	var walk func(items []playwrightSuite, parents []string)
	walk = func(items []playwrightSuite, parents []string) {
		for _, suite := range items {
			currentParents := parents
			if title := strings.TrimSpace(suite.Title); title != "" {
				currentParents = append(currentParents, title)
			}

			for _, spec := range suite.Specs {
				for _, test := range spec.Tests {
					for _, result := range test.Results {
						if strings.ToLower(strings.TrimSpace(result.Status)) != "failed" {
							continue
						}

						suiteName := suiteIdentity(suite, currentParents)
						testName := strings.TrimSpace(spec.Title)
						if testName == "" {
							testName = "UNKNOWN_TEST"
						}

						message := "Test failed"
						if result.Error != nil && strings.TrimSpace(result.Error.Message) != "" {
							message = firstMeaningfulLine(result.Error.Message)
						}

						out = append(out, ParsedFailureSummary{
							Suite:   suiteName,
							Test:    testName,
							Message: message,
						})
					}
				}
			}

			if len(suite.Suites) > 0 {
				walk(suite.Suites, currentParents)
			}
		}
	}

	walk(suites, nil)

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Suite != out[j].Suite {
			return out[i].Suite < out[j].Suite
		}
		if out[i].Test != out[j].Test {
			return out[i].Test < out[j].Test
		}
		return out[i].Message < out[j].Message
	})

	return out
}

func suiteIdentity(suite playwrightSuite, parents []string) string {
	if file := strings.TrimSpace(suite.File); file != "" {
		return file
	}
	if len(parents) > 0 {
		return strings.Join(parents, " > ")
	}
	if title := strings.TrimSpace(suite.Title); title != "" {
		return title
	}
	return "UNKNOWN_SUITE"
}

func firstMeaningfulLine(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "Test failed"
	}

	for _, line := range strings.Split(msg, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	return "Test failed"
}