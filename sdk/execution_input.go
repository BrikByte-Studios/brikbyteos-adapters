package sdk

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Logger is the narrow structured logging contract exposed to adapters.
//
// It is intentionally small to prevent adapters from taking dependencies on
// runtime-specific logging implementations.
type Logger interface {
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
}

// ExecutionMode is the runtime selection mode propagated into adapter context.
type ExecutionMode string

const (
	ExecutionModeAll          ExecutionMode = "all"
	ExecutionModeExplicitList ExecutionMode = "explicit_list"
)

// Validate ensures the execution mode is one of the supported values.
func (m ExecutionMode) Validate() error {
	switch m {
	case ExecutionModeAll, ExecutionModeExplicitList:
		return nil
	default:
		return fmt.Errorf("invalid execution mode %q", m)
	}
}

// LogicalEnvironment represents the logical execution environment only.
// It must not be confused with OS-level environment variables.
type LogicalEnvironment string

const (
	LogicalEnvironmentDev        LogicalEnvironment = "dev"
	LogicalEnvironmentStaging    LogicalEnvironment = "staging"
	LogicalEnvironmentProduction LogicalEnvironment = "production"
	LogicalEnvironmentUnknown    LogicalEnvironment = "unknown"
)

// Validate ensures the logical environment is supported.
func (e LogicalEnvironment) Validate() error {
	switch e {
	case LogicalEnvironmentDev,
		LogicalEnvironmentStaging,
		LogicalEnvironmentProduction,
		LogicalEnvironmentUnknown:
		return nil
	default:
		return fmt.Errorf("invalid logical environment %q", e)
	}
}

// NormalizeLogicalEnvironment converts arbitrary input into a supported logical environment.
// Unknown values are intentionally coerced to "unknown" rather than guessed.
func NormalizeLogicalEnvironment(value string) LogicalEnvironment {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(LogicalEnvironmentDev):
		return LogicalEnvironmentDev
	case string(LogicalEnvironmentStaging):
		return LogicalEnvironmentStaging
	case string(LogicalEnvironmentProduction):
		return LogicalEnvironmentProduction
	default:
		return LogicalEnvironmentUnknown
	}
}

// RunRequest is the canonical execution input contract for Phase 1.
//
// Runtime constructs this object once per adapter execution attempt.
// Adapters must treat every field as read-only.
//
// Field semantics:
//   - RunID: traceability identifier for the parent run
//   - WorkspaceRoot: absolute path to repository/workspace root
//   - ArtifactsRoot: absolute path to the run artifact root
//   - AdapterArtifactsPath: relative adapter-scoped artifact path under the run
//   - Environment: logical environment only (dev/staging/production/unknown)
//   - ExecutionMode: all or explicit_list
//   - Timeout: execution timeout requested by runtime
//   - Logger: narrow logging interface
//   - AdapterOptions: adapter-specific execution options, validated before use
//   - EnvVars: subprocess environment variables in KEY=VALUE form
type RunRequest struct {
	RunID                string             `json:"run_id"`
	WorkspaceRoot        string             `json:"workspace_root"`
	ArtifactsRoot        string             `json:"artifacts_root"`
	AdapterArtifactsPath string             `json:"adapter_artifacts_path"`
	Environment          LogicalEnvironment `json:"environment"`
	ExecutionMode        ExecutionMode      `json:"execution_mode"`
	Timeout              time.Duration      `json:"timeout"`
	Logger               Logger             `json:"-"`
	AdapterOptions       map[string]any     `json:"adapter_options,omitempty"`
	EnvVars              []string           `json:"env_vars,omitempty"`
}

// Clone returns a defensive copy of the run request.
// Runtime should use this when handing the request to adapters to prevent
// accidental cross-mutation of maps or slices.
func (r RunRequest) Clone() RunRequest {
	options := make(map[string]any, len(r.AdapterOptions))
	for k, v := range r.AdapterOptions {
		options[k] = v
	}

	envVars := append([]string(nil), r.EnvVars...)

	return RunRequest{
		RunID:                r.RunID,
		WorkspaceRoot:        r.WorkspaceRoot,
		ArtifactsRoot:        r.ArtifactsRoot,
		AdapterArtifactsPath: r.AdapterArtifactsPath,
		Environment:          r.Environment,
		ExecutionMode:        r.ExecutionMode,
		Timeout:              r.Timeout,
		Logger:               r.Logger,
		AdapterOptions:       options,
		EnvVars:              envVars,
	}
}

// Validate ensures the request is structurally valid and safe to hand to an adapter.
func (r RunRequest) Validate() error {
	if strings.TrimSpace(r.RunID) == "" {
		return fmt.Errorf("run_id must not be empty")
	}

	if strings.TrimSpace(r.WorkspaceRoot) == "" {
		return fmt.Errorf("workspace_root must not be empty")
	}
	if !filepath.IsAbs(r.WorkspaceRoot) {
		return fmt.Errorf("workspace_root must be an absolute path")
	}

	if strings.TrimSpace(r.ArtifactsRoot) == "" {
		return fmt.Errorf("artifacts_root must not be empty")
	}
	if !filepath.IsAbs(r.ArtifactsRoot) {
		return fmt.Errorf("artifacts_root must be an absolute path")
	}

	if strings.TrimSpace(r.AdapterArtifactsPath) == "" {
		return fmt.Errorf("adapter_artifacts_path must not be empty")
	}
	if filepath.IsAbs(r.AdapterArtifactsPath) {
		return fmt.Errorf("adapter_artifacts_path must be relative")
	}
	cleanRel := filepath.Clean(r.AdapterArtifactsPath)
	if cleanRel == "." || strings.HasPrefix(cleanRel, "..") {
		return fmt.Errorf("adapter_artifacts_path must not escape its adapter scope")
	}

	if err := r.Environment.Validate(); err != nil {
		return err
	}

	if err := r.ExecutionMode.Validate(); err != nil {
		return err
	}

	if r.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}

	for i, envVar := range r.EnvVars {
		envVar = strings.TrimSpace(envVar)
		if envVar == "" {
			return fmt.Errorf("env_vars[%d] must not be empty", i)
		}
		if !strings.Contains(envVar, "=") {
			return fmt.Errorf("env_vars[%d] must be in KEY=VALUE form", i)
		}
		if strings.HasPrefix(envVar, "=") {
			return fmt.Errorf("env_vars[%d] must have a non-empty key", i)
		}
	}

	for key := range r.AdapterOptions {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("adapter_options must not contain an empty key")
		}
	}

	return nil
}