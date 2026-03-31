package trivy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ParseStatus represents the outcome of parsing a canonical Trivy JSON report.
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

// SeverityCounts contains bounded canonical severity buckets.
//
// Buckets are always present in parsed output and default to zero.
type SeverityCounts struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Unknown  int `json:"unknown"`
}

// ParsedSummary contains the minimum canonical security fields required downstream.
type ParsedSummary struct {
	VulnerabilityTotal int            `json:"vulnerability_total"`
	SeverityCounts     SeverityCounts `json:"severity_counts"`
	MisconfigTotal     *int           `json:"misconfig_total,omitempty"`
}

// ParsedTarget contains bounded scan target identity.
type ParsedTarget struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ParsedFindingSummary is a bounded critical/high findings summary.
//
// Important:
//   - This is intentionally not a raw pass-through of Trivy-native structures.
//   - It excludes nested metadata blobs, references, descriptions, CVSS maps,
//     and other unbounded fields that do not belong in the adapter-private core contract.
type ParsedFindingSummary struct {
	Target           string `json:"target"`
	Severity         string `json:"severity"`
	ID               string `json:"id"`
	Package          string `json:"package"`
	InstalledVersion string `json:"installed_version"`
	FixedVersion     string `json:"fixed_version"`
	Title            string `json:"title"`
}

// ParserError is the structured failure object returned when parsing fails.
type ParserError struct {
	Type    ParserErrorType `json:"type"`
	Message string          `json:"message"`
	Details map[string]any  `json:"details,omitempty"`
}

// ParseResult is the adapter-private intermediate parse contract for Trivy.
type ParseResult struct {
	Adapter     string                 `json:"adapter"`
	ParseStatus ParseStatus            `json:"parse_status"`
	Summary     *ParsedSummary         `json:"summary,omitempty"`
	Target      *ParsedTarget          `json:"target,omitempty"`
	Findings    []ParsedFindingSummary `json:"findings,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Error       *ParserError           `json:"error,omitempty"`
}

// Parser reads the canonical Trivy JSON report and converts it
// into the adapter-private parse model.
//
// Design rules:
//   - primary input is the structured JSON artifact, not stdout
//   - malformed or missing reports return structured parse_failed results
//   - presence of vulnerabilities is a valid parse result, not a parser failure
type Parser struct{}

// ParseFile parses a Trivy report from disk.
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
				"Trivy report file missing",
				map[string]any{"path": reportPath},
			)
		}

		return parseFailure(
			ParserErrorMissingReport,
			"failed to read Trivy report file",
			map[string]any{"path": reportPath, "error": err.Error()},
		)
	}

	return (Parser{}).ParseBytes(data)
}

// ParseBytes parses raw Trivy report bytes.
//
// This function is pure and deterministic for identical input.
func (Parser) ParseBytes(data []byte) ParseResult {
	var raw rawTrivyReport
	if err := json.Unmarshal(data, &raw); err != nil {
		return parseFailure(
			ParserErrorInvalidJSON,
			"Malformed Trivy JSON report",
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

	summary, target, findings, warnings := raw.toParsed()

	return ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusOK,
		Summary:     &summary,
		Target:      &target,
		Findings:    findings,
		Warnings:    warnings,
	}
}

func parseFailure(errorType ParserErrorType, message string, details map[string]any) ParseResult {
	return ParseResult{
		Adapter:     AdapterName,
		ParseStatus: ParseStatusParseFailed,
		Findings:    []ParsedFindingSummary{},
		Warnings:    []string{},
		Error: &ParserError{
			Type:    errorType,
			Message: message,
			Details: details,
		},
	}
}

// --- Raw Trivy report model ---
//
// This model is intentionally narrow and built around the canonical JSON report
// assumptions for Phase 1. Extra sections are tolerated.

type rawTrivyReport struct {
	SchemaVersion int              `json:"SchemaVersion"`
	ArtifactName  string           `json:"ArtifactName"`
	ArtifactType  string           `json:"ArtifactType"`
	Results       []rawTrivyResult `json:"Results"`
}

type rawTrivyResult struct {
	Target            string                `json:"Target"`
	Class             string                `json:"Class"`
	Type              string                `json:"Type"`
	Vulnerabilities   []rawVulnerability    `json:"Vulnerabilities"`
	Misconfigurations []rawMisconfiguration `json:"Misconfigurations"`
}

type rawVulnerability struct {
	VulnerabilityID  string `json:"VulnerabilityID"`
	PkgName          string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion"`
	Title            string `json:"Title"`
	Severity         string `json:"Severity"`
}

type rawMisconfiguration struct {
	ID       string `json:"ID"`
	Severity string `json:"Severity"`
}

// validate checks the minimum required fields for Phase 1 parsing.
//
// The parser remains tolerant of extra sections, but rejects payloads that
// do not provide the required top-level structure.
func (r rawTrivyReport) validate() error {
	if r.SchemaVersion == 0 {
		return fmt.Errorf("schema mismatch: SchemaVersion is required")
	}
	if strings.TrimSpace(r.ArtifactName) == "" {
		return fmt.Errorf("schema mismatch: ArtifactName is required")
	}
	if strings.TrimSpace(r.ArtifactType) == "" {
		return fmt.Errorf("schema mismatch: ArtifactType is required")
	}
	if r.Results == nil {
		return fmt.Errorf("schema mismatch: Results is required")
	}
	return nil
}

func (r rawTrivyReport) toParsed() (ParsedSummary, ParsedTarget, []ParsedFindingSummary, []string) {
	var warnings []string
	var findings []ParsedFindingSummary

	summary := ParsedSummary{
		VulnerabilityTotal: 0,
		SeverityCounts: SeverityCounts{
			Critical: 0,
			High:     0,
			Medium:   0,
			Low:      0,
			Unknown:  0,
		},
	}

	target := ParsedTarget{
		Name: r.ArtifactName,
		Type: r.ArtifactType,
	}

	misconfigTotal := 0
	misconfigSeen := false

	for _, result := range r.Results {
		normalizedTarget := strings.TrimSpace(result.Target)
		if normalizedTarget == "" {
			normalizedTarget = r.ArtifactName
		}

		for _, vuln := range result.Vulnerabilities {
			summary.VulnerabilityTotal++
			incrementSeverity(&summary.SeverityCounts, vuln.Severity)

			normalizedSeverity := canonicalSeverity(vuln.Severity)
			if normalizedSeverity == "CRITICAL" || normalizedSeverity == "HIGH" {
				findings = append(findings, ParsedFindingSummary{
					Target:           normalizedTarget,
					Severity:         normalizedSeverity,
					ID:               strings.TrimSpace(vuln.VulnerabilityID),
					Package:          strings.TrimSpace(vuln.PkgName),
					InstalledVersion: strings.TrimSpace(vuln.InstalledVersion),
					FixedVersion:     strings.TrimSpace(vuln.FixedVersion),
					Title:            strings.TrimSpace(vuln.Title),
				})
			}
		}

		if len(result.Misconfigurations) > 0 {
			misconfigSeen = true
			misconfigTotal += len(result.Misconfigurations)
		}
	}

	if misconfigSeen {
		value := misconfigTotal
		summary.MisconfigTotal = &value
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Target != findings[j].Target {
			return findings[i].Target < findings[j].Target
		}
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity < findings[j].Severity
		}
		if findings[i].ID != findings[j].ID {
			return findings[i].ID < findings[j].ID
		}
		if findings[i].Package != findings[j].Package {
			return findings[i].Package < findings[j].Package
		}
		return findings[i].Title < findings[j].Title
	})

	return summary, target, findings, warnings
}

// canonicalSeverity normalizes Trivy severity strings into bounded canonical values.
//
// Unknown or unsupported severities collapse into UNKNOWN.
func canonicalSeverity(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "CRITICAL":
		return "CRITICAL"
	case "HIGH":
		return "HIGH"
	case "MEDIUM":
		return "MEDIUM"
	case "LOW":
		return "LOW"
	default:
		return "UNKNOWN"
	}
}

func incrementSeverity(counts *SeverityCounts, severity string) {
	switch canonicalSeverity(severity) {
	case "CRITICAL":
		counts.Critical++
	case "HIGH":
		counts.High++
	case "MEDIUM":
		counts.Medium++
	case "LOW":
		counts.Low++
	default:
		counts.Unknown++
	}
}
