package k6

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ParseStatus represents the outcome of parsing a canonical k6 summary export.
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

// ThresholdStatus is the bounded threshold outcome stored in the intermediate model.
type ThresholdStatus string

const (
	ThresholdStatusPass ThresholdStatus = "pass"
	ThresholdStatusFail ThresholdStatus = "fail"
)

// ParsedSummary contains the minimum canonical performance fields required downstream.
//
// Notes:
//   - latency percentiles are optional and nil when absent
//   - request_total is HTTP request count, not checks or iterations
//   - request_failed is failed HTTP request count, not threshold-failed count
type ParsedSummary struct {
	RequestTotal  int      `json:"request_total"`
	RequestFailed int      `json:"request_failed"`
	LatencyP95Ms  *float64 `json:"latency_p95_ms,omitempty"`
	LatencyP99Ms  *float64 `json:"latency_p99_ms,omitempty"`
	DurationMs    int64    `json:"duration_ms"`
}

// ParsedThresholdSummary is a bounded threshold summary.
//
// Important:
//   - this intentionally excludes raw threshold internals
//   - ordering is made deterministic before returning parser output
type ParsedThresholdSummary struct {
	Name   string          `json:"name"`
	Status ThresholdStatus `json:"status"`
}

// ParserError is the structured failure object returned when parsing fails.
type ParserError struct {
	Type    ParserErrorType `json:"type"`
	Message string          `json:"message"`
	Details map[string]any  `json:"details,omitempty"`
}

// ParseResult is the adapter-private intermediate parse contract for k6.
type ParseResult struct {
	Adapter     string                   `json:"adapter"`
	ParseStatus ParseStatus              `json:"parse_status"`
	Summary     *ParsedSummary           `json:"summary,omitempty"`
	Thresholds  []ParsedThresholdSummary `json:"thresholds,omitempty"`
	Warnings    []string                 `json:"warnings,omitempty"`
	Error       *ParserError             `json:"error,omitempty"`
}

// Parser reads the canonical k6 structured summary export and converts it
// into the adapter-private parse model.
//
// Design rules:
//   - primary input is the structured summary artifact, not stdout
//   - malformed or missing summaries return structured parse_failed results
//   - threshold failures are valid parse results, not parser failures
type Parser struct{}

// ParseFile parses a k6 summary export from disk.
func (Parser) ParseFile(summaryPath string) ParseResult {
	if strings.TrimSpace(summaryPath) == "" {
		return parseFailure(
			ParserErrorSchemaMismatch,
			"summary path is required",
			map[string]any{},
		)
	}

	data, err := os.ReadFile(summaryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parseFailure(
				ParserErrorMissingReport,
				"k6 summary export file missing",
				map[string]any{"path": summaryPath},
			)
		}

		return parseFailure(
			ParserErrorMissingReport,
			"failed to read k6 summary export file",
			map[string]any{"path": summaryPath, "error": err.Error()},
		)
	}

	return (Parser{}).ParseBytes(data)
}

// ParseBytes parses raw k6 summary bytes.
//
// This function is pure and deterministic for identical input.
func (Parser) ParseBytes(data []byte) ParseResult {
	var raw rawK6Summary
	if err := json.Unmarshal(data, &raw); err != nil {
		return parseFailure(
			ParserErrorInvalidJSON,
			"Malformed k6 summary JSON",
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

	summary, thresholds, warnings := raw.toParsed()

	return ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary:     &summary,
		Thresholds:  thresholds,
		Warnings:    warnings,
	}
}

func parseFailure(errorType ParserErrorType, message string, details map[string]any) ParseResult {
	return ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusParseFailed,
		Thresholds:  []ParsedThresholdSummary{},
		Warnings:    []string{},
		Error: &ParserError{
			Type:    errorType,
			Message: message,
			Details: details,
		},
	}
}

// --- Raw k6 summary model ---
//
// This model is intentionally narrow and built around the canonical summary-export
// assumptions for Phase 1. Extra sections are tolerated.

type rawK6Summary struct {
	State     rawState             `json:"state"`
	Metrics   map[string]rawMetric `json:"metrics"`
	RootGroup *json.RawMessage     `json:"root_group,omitempty"` // tolerated, ignored
	Options   *json.RawMessage     `json:"options,omitempty"`    // tolerated, ignored
	SetupData *json.RawMessage     `json:"setup_data,omitempty"` // tolerated, ignored
	Tainted   *json.RawMessage     `json:"tainted,omitempty"`    // tolerated, ignored
}

type rawState struct {
	TestRunDurationMs *int64 `json:"testRunDurationMs"`
}

type rawMetric struct {
	Type       string                  `json:"type"`
	Contains   string                  `json:"contains"`
	Values     map[string]float64      `json:"values"`
	Thresholds map[string]rawThreshold `json:"thresholds"`
}

type rawThreshold struct {
	Ok *bool `json:"ok"`
}

func (r rawK6Summary) validate() error {
	if r.State.TestRunDurationMs == nil {
		return fmt.Errorf("schema mismatch: state.testRunDurationMs is required")
	}
	if *r.State.TestRunDurationMs < 0 {
		return fmt.Errorf("schema mismatch: state.testRunDurationMs must be non-negative")
	}
	if r.Metrics == nil {
		return fmt.Errorf("schema mismatch: metrics is required")
	}

	reqCountMetric, ok := r.Metrics["http_reqs"]
	if !ok {
		return fmt.Errorf("schema mismatch: metrics.http_reqs is required")
	}
	if reqCountMetric.Values == nil {
		return fmt.Errorf("schema mismatch: metrics.http_reqs.values is required")
	}
	if _, ok := reqCountMetric.Values["count"]; !ok {
		return fmt.Errorf("schema mismatch: metrics.http_reqs.values.count is required")
	}

	reqFailedMetric, ok := r.Metrics["http_req_failed"]
	if !ok {
		return fmt.Errorf("schema mismatch: metrics.http_req_failed is required")
	}
	if reqFailedMetric.Values == nil {
		return fmt.Errorf("schema mismatch: metrics.http_req_failed.values is required")
	}
	if _, ok := reqFailedMetric.Values["fails"]; !ok {
		return fmt.Errorf("schema mismatch: metrics.http_req_failed.values.fails is required")
	}

	return nil
}

func (r rawK6Summary) toParsed() (ParsedSummary, []ParsedThresholdSummary, []string) {
	var warnings []string

	requestTotal := int(r.Metrics["http_reqs"].Values["count"])
	requestFailed := int(r.Metrics["http_req_failed"].Values["fails"])
	durationMs := *r.State.TestRunDurationMs

	var p95 *float64
	var p99 *float64

	if metric, ok := r.Metrics["http_req_duration"]; ok && metric.Values != nil {
		if value, ok := lookupPercentile(metric.Values, "p(95)"); ok {
			v := value
			p95 = &v
		}
		if value, ok := lookupPercentile(metric.Values, "p(99)"); ok {
			v := value
			p99 = &v
		}
	}

	thresholds := extractThresholdSummaries(r.Metrics)

	summary := ParsedSummary{
		RequestTotal:  requestTotal,
		RequestFailed: requestFailed,
		LatencyP95Ms:  p95,
		LatencyP99Ms:  p99,
		DurationMs:    durationMs,
	}

	return summary, thresholds, warnings
}

// lookupPercentile returns a percentile metric if present.
func lookupPercentile(values map[string]float64, key string) (float64, bool) {
	if values == nil {
		return 0, false
	}
	v, ok := values[key]
	return v, ok
}

// extractThresholdSummaries emits bounded threshold summaries in stable order.
//
// Strategy:
//   - gather thresholds from all metrics
//   - emit one summary entry per threshold expression under the metric namespace
//   - sort deterministically by name
func extractThresholdSummaries(metrics map[string]rawMetric) []ParsedThresholdSummary {
	out := make([]ParsedThresholdSummary, 0)

	for metricName, metric := range metrics {
		for thresholdExpr, threshold := range metric.Thresholds {
			if threshold.Ok == nil {
				// Unsupported or incomplete threshold entry is ignored safely.
				continue
			}

			name := metricName
			if strings.TrimSpace(thresholdExpr) != "" {
				name = fmt.Sprintf("%s:%s", metricName, thresholdExpr)
			}

			status := ThresholdStatusFail
			if *threshold.Ok {
				status = ThresholdStatusPass
			}

			out = append(out, ParsedThresholdSummary{
				Name:   name,
				Status: status,
			})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Status < out[j].Status
	})

	return out
}
