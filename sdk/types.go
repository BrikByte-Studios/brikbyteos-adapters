package sdk

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// AdapterType is the controlled category enum for adapter classification.
type AdapterType string

const (
	AdapterTypeUnit        AdapterType = "unit"
	AdapterTypeUI          AdapterType = "ui"
	AdapterTypePerformance AdapterType = "performance"
	AdapterTypeSecurity    AdapterType = "security"
	AdapterTypeOther       AdapterType = "other"
)

// Validate ensures the adapter type is one of the approved SDK values.
func (t AdapterType) Validate() error {
	switch t {
	case AdapterTypeUnit, AdapterTypeUI, AdapterTypePerformance, AdapterTypeSecurity, AdapterTypeOther:
		return nil
	default:
		return fmt.Errorf("invalid adapter type %q", t)
	}
}

// AdapterMetadata is the canonical static description of one adapter.
//
// This is intentionally separated from runtime behavior so that registry,
// inspection, ordering, and CLI display can reuse one metadata source.
type AdapterMetadata struct {
	Name               string      `json:"name"`
	Type               AdapterType `json:"type"`
	Description        string      `json:"description"`
	Order              int         `json:"order"`
	SupportedTool      string      `json:"supported_tool"`
	VersionCommand     []string    `json:"version_command,omitempty"`
	DefaultTimeoutMs   int         `json:"default_timeout_ms"`
	SupportedFileGlobs []string    `json:"supported_file_globs,omitempty"`
	Aliases            []string    `json:"aliases,omitempty"`
	Capabilities       []string    `json:"capabilities,omitempty"`
}

// Availability reports whether an adapter can run in the current environment.
type Availability struct {
	Available      bool   `json:"available"`
	Reason         string `json:"reason,omitempty"`
	ResolvedBinary string `json:"resolved_binary,omitempty"`
}

// Logger is the narrow structured logging contract adapters may use.
// This avoids direct writes to stdout/stderr for adapter orchestration logs.
type Logger interface {
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
}

// RunRequest is the canonical execution input passed from runtime to adapters.
//
// Runtime constructs this once per adapter execution.
// Adapters must treat all fields as read-only.
type RunRequest struct {
	RunID                string         `json:"run_id"`
	WorkspaceRoot        string         `json:"workspace_root"`
	ArtifactsRoot        string         `json:"artifacts_root"`
	AdapterArtifactsPath string         `json:"adapter_artifacts_path"`
	Environment          string         `json:"environment"`
	ExecutionMode        string         `json:"execution_mode"`
	Timeout              time.Duration  `json:"timeout"`
	Logger               Logger         `json:"-"`
	AdapterOptions       map[string]any `json:"adapter_options,omitempty"`
	EnvVars              []string       `json:"env_vars,omitempty"`
}

// ExecutionStatus is the process-level result of one adapter execution attempt.
type ExecutionStatus string

const (
	ExecutionStatusCompleted   ExecutionStatus = "completed"
	ExecutionStatusFailed      ExecutionStatus = "failed"
	ExecutionStatusTimedOut    ExecutionStatus = "timed_out"
	ExecutionStatusUnavailable ExecutionStatus = "unavailable"
)

// Validate ensures the execution status is one of the approved values.
func (s ExecutionStatus) Validate() error {
	switch s {
	case ExecutionStatusCompleted, ExecutionStatusFailed, ExecutionStatusTimedOut, ExecutionStatusUnavailable:
		return nil
	default:
		return fmt.Errorf("invalid execution status %q", s)
	}
}

// RunResult is the canonical structured execution result returned by Adapter.Run.
//
// It contains process-level execution truth only.
// It must not include normalized/domain-level interpretation.
type RunResult struct {
	Status       ExecutionStatus `json:"status"`
	ExitCode     *int            `json:"exit_code,omitempty"`
	DurationMs   int64           `json:"duration_ms"`
	Stdout       []byte          `json:"-"`
	Stderr       []byte          `json:"-"`
	ToolOutput   []byte          `json:"-"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

// RawExecution is the canonical pre-normalized execution envelope handed to Normalize.
type RawExecution struct {
	SchemaVersion  string      `json:"schema_version"`
	AdapterName    string      `json:"adapter_name"`
	AdapterType    AdapterType `json:"adapter_type"`
	AdapterVersion string      `json:"adapter_version"`
	Command        string      `json:"command"`
	Args           []string    `json:"args"`
	ResolvedBinary string      `json:"resolved_binary,omitempty"`
	RunResult      RunResult   `json:"run_result"`
	StdoutPath     string      `json:"stdout_path,omitempty"`
	StderrPath     string      `json:"stderr_path,omitempty"`
	ToolOutputPath string      `json:"tool_output_path,omitempty"`
}

// NormalizationInput is the only allowed input to adapter normalization.
type NormalizationInput struct {
	RawExecution RawExecution    `json:"raw_execution"`
	AdapterMeta  AdapterMetadata `json:"adapter_meta"`
}

// NormalizedResult is schema-owned by brikbyteos-schema.
//
// Until generated Go types exist, adapters return canonical JSON payload bytes.
// Runtime is responsible for validating them against the schema contract.
type NormalizedResult = json.RawMessage

// Clone returns a defensive copy of the request so runtime can avoid accidental mutation.
func (r RunRequest) Clone() RunRequest {
	clonedOptions := make(map[string]any, len(r.AdapterOptions))
	for k, v := range r.AdapterOptions {
		clonedOptions[k] = v
	}

	clonedEnv := append([]string(nil), r.EnvVars...)
	return RunRequest{
		RunID:                r.RunID,
		WorkspaceRoot:        r.WorkspaceRoot,
		ArtifactsRoot:        r.ArtifactsRoot,
		AdapterArtifactsPath: r.AdapterArtifactsPath,
		Environment:          r.Environment,
		ExecutionMode:        r.ExecutionMode,
		Timeout:              r.Timeout,
		Logger:               r.Logger,
		AdapterOptions:       clonedOptions,
		EnvVars:              clonedEnv,
	}
}

// Validate ensures adapter metadata is structurally correct before runtime use.
func (m AdapterMetadata) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("adapter metadata name must not be empty")
	}
	if err := m.Type.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(m.Description) == "" {
		return fmt.Errorf("adapter %q description must not be empty", m.Name)
	}
	if m.Order <= 0 {
		return fmt.Errorf("adapter %q order must be > 0", m.Name)
	}
	if strings.TrimSpace(m.SupportedTool) == "" {
		return fmt.Errorf("adapter %q supported_tool must not be empty", m.Name)
	}
	if m.DefaultTimeoutMs <= 0 {
		return fmt.Errorf("adapter %q default_timeout_ms must be > 0", m.Name)
	}
	for i, alias := range m.Aliases {
		if strings.TrimSpace(alias) == "" {
			return fmt.Errorf("adapter %q aliases[%d] must not be empty", m.Name, i)
		}
	}
	return nil
}

// Validate ensures the run request is safe and complete enough for adapter execution.
func (r RunRequest) Validate() error {
	if strings.TrimSpace(r.RunID) == "" {
		return fmt.Errorf("run_id must not be empty")
	}
	if strings.TrimSpace(r.WorkspaceRoot) == "" {
		return fmt.Errorf("workspace_root must not be empty")
	}
	if !filepath.IsAbs(r.WorkspaceRoot) {
		return fmt.Errorf("workspace_root must be absolute")
	}
	if strings.TrimSpace(r.ArtifactsRoot) == "" {
		return fmt.Errorf("artifacts_root must not be empty")
	}
	if strings.TrimSpace(r.AdapterArtifactsPath) == "" {
		return fmt.Errorf("adapter_artifacts_path must not be empty")
	}
	if r.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0")
	}
	return nil
}

// Validate ensures the run result is structurally valid.
func (r RunResult) Validate() error {
	if err := r.Status.Validate(); err != nil {
		return err
	}
	if r.DurationMs < 0 {
		return fmt.Errorf("duration_ms must be >= 0")
	}
	return nil
}

// Validate ensures the raw execution envelope is structurally valid.
func (r RawExecution) Validate() error {
	if strings.TrimSpace(r.SchemaVersion) == "" {
		return fmt.Errorf("schema_version must not be empty")
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
	if strings.TrimSpace(r.Command) == "" && r.RunResult.Status != ExecutionStatusUnavailable {
		return fmt.Errorf("command must not be empty unless adapter is unavailable")
	}
	if err := r.RunResult.Validate(); err != nil {
		return err
	}
	return nil
}
