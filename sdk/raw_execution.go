package sdk

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// RawExecutionSchemaVersion is the canonical schema version for Phase 1 raw execution output.
const RawExecutionSchemaVersion = "0.1"

// RawExecutionResult is the canonical structured, pre-normalized execution envelope
// produced by runtime for every adapter attempt.
//
// This contract is consumed by:
//   - normalization
//   - persistence
//   - reporting
//   - future policy and audit flows
//
// It must remain process-level only.
// No domain-level interpretation belongs here.
type RawExecutionResult struct {
	SchemaVersion   string          `json:"schema_version"`
	AdapterName     string          `json:"adapter_name"`
	AdapterType     AdapterType     `json:"adapter_type"`
	AdapterVersion  string          `json:"adapter_version"`
	Command         string          `json:"command"`
	Args            []string        `json:"args,omitempty"`
	ResolvedBinary  string          `json:"resolved_binary,omitempty"`
	ExecutionStatus ExecutionStatus `json:"execution_status"`
	ExitCode        *int            `json:"exit_code,omitempty"`
	StartedAt       time.Time       `json:"started_at"`
	FinishedAt      time.Time       `json:"finished_at"`
	DurationMs      int64           `json:"duration_ms"`
	StdoutPath      string          `json:"stdout_path,omitempty"`
	StderrPath      string          `json:"stderr_path,omitempty"`
	ToolOutputPath  string          `json:"tool_output_path,omitempty"`
	ErrorMessage    string          `json:"error_message,omitempty"`
}

// Validate ensures the raw execution envelope is structurally valid.
func (r RawExecutionResult) Validate() error {
	if strings.TrimSpace(r.SchemaVersion) == "" {
		return fmt.Errorf("schema_version must not be empty")
	}
	if r.SchemaVersion != RawExecutionSchemaVersion {
		return fmt.Errorf("unsupported schema_version %q", r.SchemaVersion)
	}

	if strings.TrimSpace(r.AdapterName) == "" {
		return fmt.Errorf("adapter_name must not be empty")
	}

	if err := r.AdapterType.Validate(); err != nil {
		return err
	}

	if strings.TrimSpace(r.AdapterVersion) == "" {
		return fmt.Errorf("adapter_version must not be empty")
	}

	if err := r.ExecutionStatus.Validate(); err != nil {
		return err
	}

	if r.DurationMs < 0 {
		return fmt.Errorf("duration_ms must be >= 0")
	}

	if r.StartedAt.IsZero() {
		return fmt.Errorf("started_at must not be zero")
	}
	if r.FinishedAt.IsZero() {
		return fmt.Errorf("finished_at must not be zero")
	}
	if r.FinishedAt.Before(r.StartedAt) {
		return fmt.Errorf("finished_at must not be before started_at")
	}

	if r.ExecutionStatus != ExecutionStatusUnavailable && strings.TrimSpace(r.Command) == "" {
		return fmt.Errorf("command must not be empty unless execution_status=unavailable")
	}

	if err := validateRelativeArtifactPath("stdout_path", r.StdoutPath); err != nil {
		return err
	}
	if err := validateRelativeArtifactPath("stderr_path", r.StderrPath); err != nil {
		return err
	}
	if err := validateRelativeArtifactPath("tool_output_path", r.ToolOutputPath); err != nil {
		return err
	}

	return nil
}

// MarshalJSON ensures stable UTC RFC3339 timestamps in serialized output.
func (r RawExecutionResult) MarshalJSON() ([]byte, error) {
	type alias struct {
		SchemaVersion   string          `json:"schema_version"`
		AdapterName     string          `json:"adapter_name"`
		AdapterType     AdapterType     `json:"adapter_type"`
		AdapterVersion  string          `json:"adapter_version"`
		Command         string          `json:"command"`
		Args            []string        `json:"args,omitempty"`
		ResolvedBinary  string          `json:"resolved_binary,omitempty"`
		ExecutionStatus ExecutionStatus `json:"execution_status"`
		ExitCode        *int            `json:"exit_code,omitempty"`
		StartedAt       string          `json:"started_at"`
		FinishedAt      string          `json:"finished_at"`
		DurationMs      int64           `json:"duration_ms"`
		StdoutPath      string          `json:"stdout_path,omitempty"`
		StderrPath      string          `json:"stderr_path,omitempty"`
		ToolOutputPath  string          `json:"tool_output_path,omitempty"`
		ErrorMessage    string          `json:"error_message,omitempty"`
	}

	return json.Marshal(alias{
		SchemaVersion:   r.SchemaVersion,
		AdapterName:     r.AdapterName,
		AdapterType:     r.AdapterType,
		AdapterVersion:  r.AdapterVersion,
		Command:         r.Command,
		Args:            r.Args,
		ResolvedBinary:  r.ResolvedBinary,
		ExecutionStatus: r.ExecutionStatus,
		ExitCode:        r.ExitCode,
		StartedAt:       r.StartedAt.UTC().Format(time.RFC3339),
		FinishedAt:      r.FinishedAt.UTC().Format(time.RFC3339),
		DurationMs:      r.DurationMs,
		StdoutPath:      r.StdoutPath,
		StderrPath:      r.StderrPath,
		ToolOutputPath:  r.ToolOutputPath,
		ErrorMessage:    r.ErrorMessage,
	})
}

func validateRelativeArtifactPath(fieldName, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	if filepath.IsAbs(value) {
		return fmt.Errorf("%s must be relative", fieldName)
	}

	clean := filepath.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s must not escape its artifact scope", fieldName)
	}

	return nil
}
