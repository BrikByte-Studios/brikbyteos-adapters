package trivy

import (
	"encoding/json"
	"fmt"
	"sort"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// Canonical normalized result constants.
//
// These constants are centralized to avoid status drift between the
// Trivy parser, normalizer, tests, and downstream consumers.
const (
	trivyNormalizedSchemaVersion = "0.1"
	trivyResultKindSecurityScan  = "security_scan"

	trivyStatusPass                = "pass"
	trivyStatusFailed              = "failed"
	trivyStatusNormalizationFailed = "normalization_failed"
)

// Normalizer converts the adapter-private Trivy parse model into the
// canonical BrikByte normalized result.
//
// Design goals:
//   - pure function of parser output
//   - deterministic field mapping
//   - explicit parser-failure handling
//   - strict anti-leak boundary for Trivy-specific details
type Normalizer struct{}

// Normalize converts a Trivy ParseResult into canonical normalized JSON.
//
// Mapping rules:
//   - parse ok + zero vulnerability_total -> pass
//   - parse ok + one or more vulnerability_total -> failed
//   - parse failed -> normalization_failed
//
// issue_count is always mapped from vulnerability_total.
func (Normalizer) Normalize(input ParseResult) sdk.NormalizedResult {
	payload := buildNormalizedPayload(input)

	encoded, err := json.Marshal(payload)
	if err != nil {
		fallback := normalizedFailurePayload(&ParserError{
			Type:    ParserErrorType("marshal_failed"),
			Message: "failed to marshal normalized Trivy payload",
			Details: map[string]any{},
		})

		encodedFallback, fallbackErr := json.Marshal(fallback)
		if fallbackErr != nil {
			return sdk.NormalizedResult(`{"schema_version":"0.1","adapter":"trivy","status":"normalization_failed","result_kind":"security_scan","summary":null,"evidence":{"raw_available":true,"normalized_complete":false},"error":{"type":"marshal_failed","message":"failed to marshal normalized Trivy payload","details":{}}}`)
		}

		return sdk.NormalizedResult(encodedFallback)
	}

	return sdk.NormalizedResult(encoded)
}

// buildNormalizedPayload creates the in-memory canonical payload.
// It is intentionally separated from Normalize to improve testability.
func buildNormalizedPayload(input ParseResult) normalizedPayload {
	if input.ParseStatus == ParseStatusParseFailed {
		return normalizedFailurePayload(input.Error)
	}

	if input.Summary == nil {
		return normalizedFailurePayload(&ParserError{
			Type:    ParserErrorType("schema_mismatch"),
			Message: "parser returned ok status without summary",
			Details: map[string]any{},
		})
	}

	status := canonicalStatusFromSummary(*input.Summary)

	payload := normalizedPayload{
		SchemaVersion: trivyNormalizedSchemaVersion,
		Adapter:       AdapterName,
		Status:        status,
		ResultKind:    trivyResultKindSecurityScan,
		Summary: &normalizedSummary{
			VulnerabilityTotal: input.Summary.VulnerabilityTotal,
			SeverityCounts: normalizedSeverityCounts{
				Critical: input.Summary.SeverityCounts.Critical,
				High:     input.Summary.SeverityCounts.High,
				Medium:   input.Summary.SeverityCounts.Medium,
				Low:      input.Summary.SeverityCounts.Low,
				Unknown:  input.Summary.SeverityCounts.Unknown,
			},
			MisconfigTotal: input.Summary.MisconfigTotal,
			IssueCount:     input.Summary.VulnerabilityTotal,
		},
		Evidence: normalizedEvidence{
			RawAvailable:       true,
			NormalizedComplete: true,
		},
	}

	// All scanner-specific details must remain inside extensions.trivy.
	extension := buildTrivyExtension(input)
	if extension != nil {
		payload.Extensions = &normalizedExtensions{
			Trivy: extension,
		}
	}

	return payload
}

// canonicalStatusFromSummary computes the canonical normalized status.
func canonicalStatusFromSummary(summary ParsedSummary) string {
	if summary.VulnerabilityTotal > 0 {
		return trivyStatusFailed
	}
	return trivyStatusPass
}

// normalizedFailurePayload converts parser failure input into the canonical
// normalization_failed shape with no fabricated summary metrics.
func normalizedFailurePayload(err *ParserError) normalizedPayload {
	if err == nil {
		err = &ParserError{
			Type:    ParserErrorType("parse_failed"),
			Message: "parser failed without structured error",
			Details: map[string]any{},
		}
	}

	return normalizedPayload{
		SchemaVersion: trivyNormalizedSchemaVersion,
		Adapter:       AdapterName,
		Status:        trivyStatusNormalizationFailed,
		ResultKind:    trivyResultKindSecurityScan,
		Summary:       nil,
		Evidence: normalizedEvidence{
			RawAvailable:       true,
			NormalizedComplete: false,
		},
		Error: &normalizedError{
			Type:    string(err.Type),
			Message: err.Message,
			Details: err.Details,
		},
	}
}

// buildTrivyExtension creates a bounded extension payload or nil when unnecessary.
func buildTrivyExtension(input ParseResult) *normalizedTrivyExtension {
	var ext normalizedTrivyExtension

	if input.Target != nil {
		ext.Target = &normalizedTarget{
			Name: input.Target.Name,
			Type: input.Target.Type,
		}
	}

	if len(input.Findings) > 0 {
		ext.CriticalHighFindings = stableFindings(input.Findings)
	}

	if len(input.Warnings) > 0 {
		ext.Warnings = stableWarnings(input.Warnings)
	}

	if ext.Target == nil && len(ext.CriticalHighFindings) == 0 && len(ext.Warnings) == 0 {
		return nil
	}

	return &ext
}

// stableFindings sorts and copies bounded findings for determinism.
func stableFindings(in []ParsedFindingSummary) []normalizedFindingSummary {
	out := make([]normalizedFindingSummary, 0, len(in))
	for _, f := range in {
		out = append(out, normalizedFindingSummary{
			Target:           f.Target,
			Severity:         f.Severity,
			ID:               f.ID,
			Package:          f.Package,
			InstalledVersion: f.InstalledVersion,
			FixedVersion:     f.FixedVersion,
			Title:            f.Title,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Target != out[j].Target {
			return out[i].Target < out[j].Target
		}
		if out[i].Severity != out[j].Severity {
			return out[i].Severity < out[j].Severity
		}
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		if out[i].Package != out[j].Package {
			return out[i].Package < out[j].Package
		}
		return out[i].Title < out[j].Title
	})

	return out
}

// stableWarnings sorts warnings deterministically.
func stableWarnings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// --- Canonical output model ---

type normalizedPayload struct {
	SchemaVersion string                `json:"schema_version"`
	Adapter       string                `json:"adapter"`
	Status        string                `json:"status"`
	ResultKind    string                `json:"result_kind"`
	Summary       *normalizedSummary    `json:"summary"`
	Evidence      normalizedEvidence    `json:"evidence"`
	Error         *normalizedError      `json:"error,omitempty"`
	Extensions    *normalizedExtensions `json:"extensions,omitempty"`
}

type normalizedSummary struct {
	VulnerabilityTotal int                      `json:"vulnerability_total"`
	SeverityCounts     normalizedSeverityCounts `json:"severity_counts"`
	MisconfigTotal     *int                     `json:"misconfig_total,omitempty"`
	IssueCount         int                      `json:"issue_count"`
}

type normalizedSeverityCounts struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Unknown  int `json:"unknown"`
}

type normalizedEvidence struct {
	RawAvailable       bool `json:"raw_available"`
	NormalizedComplete bool `json:"normalized_complete"`
}

type normalizedError struct {
	Type    string         `json:"type"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type normalizedExtensions struct {
	Trivy *normalizedTrivyExtension `json:"trivy,omitempty"`
}

type normalizedTrivyExtension struct {
	Target               *normalizedTarget          `json:"target,omitempty"`
	CriticalHighFindings []normalizedFindingSummary `json:"critical_high_findings,omitempty"`
	Warnings             []string                   `json:"warnings,omitempty"`
}

type normalizedTarget struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type normalizedFindingSummary struct {
	Target           string `json:"target"`
	Severity         string `json:"severity"`
	ID               string `json:"id"`
	Package          string `json:"package"`
	InstalledVersion string `json:"installed_version"`
	FixedVersion     string `json:"fixed_version"`
	Title            string `json:"title"`
}

// ValidateNormalizedPayloadShape performs lightweight internal invariant checks.
//
// This is intentionally narrow. It is not a replacement for future stricter
// schema validation from brikbyteos-schema.
func ValidateNormalizedPayloadShape(payload normalizedPayload) error {
	if payload.SchemaVersion != trivyNormalizedSchemaVersion {
		return fmt.Errorf("unexpected schema_version: %s", payload.SchemaVersion)
	}

	if payload.Adapter != AdapterName {
		return fmt.Errorf("unexpected adapter: %s", payload.Adapter)
	}

	if payload.ResultKind != trivyResultKindSecurityScan {
		return fmt.Errorf("unexpected result_kind: %s", payload.ResultKind)
	}

	switch payload.Status {
	case trivyStatusPass, trivyStatusFailed:
		if payload.Summary == nil {
			return fmt.Errorf("summary is required for non-failure normalized payload")
		}
		if !payload.Evidence.NormalizedComplete {
			return fmt.Errorf("normalized_complete must be true for successful mapping")
		}
		if payload.Summary.IssueCount != payload.Summary.VulnerabilityTotal {
			return fmt.Errorf("issue_count must equal vulnerability_total")
		}
		if payload.Summary.SeverityCounts.Critical < 0 ||
			payload.Summary.SeverityCounts.High < 0 ||
			payload.Summary.SeverityCounts.Medium < 0 ||
			payload.Summary.SeverityCounts.Low < 0 ||
			payload.Summary.SeverityCounts.Unknown < 0 {
			return fmt.Errorf("severity counts must be non-negative")
		}

	case trivyStatusNormalizationFailed:
		if payload.Summary != nil {
			return fmt.Errorf("summary must be nil for normalization_failed")
		}
		if payload.Evidence.NormalizedComplete {
			return fmt.Errorf("normalized_complete must be false for normalization_failed")
		}
		if payload.Error == nil {
			return fmt.Errorf("error is required for normalization_failed")
		}

	default:
		return fmt.Errorf("unexpected status: %s", payload.Status)
	}

	return nil
}