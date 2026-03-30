package k6

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ResolutionMode identifies how the k6 CLI was resolved.
type ResolutionMode string

const (
	ResolutionGlobal ResolutionMode = "global_binary"
	ResolutionLocal  ResolutionMode = "local_binary"
	ResolutionNone   ResolutionMode = "not_found"
)

// ExecutionStatus defines canonical execution outcomes at the execution-spec layer.
//
// Notes:
//   - threshold_failed is intentionally modeled as a distinct execution-layer
//     classification only if the runtime chooses to surface it that way.
//   - The recommended Phase 1 interpretation is that threshold failures remain
//     valid executions and are handled later in parsing/normalization.
type ExecutionStatus string

const (
	StatusCompleted            ExecutionStatus = "completed"
	StatusFailed               ExecutionStatus = "failed"
	StatusTimedOut             ExecutionStatus = "timed_out"
	StatusNotFound             ExecutionStatus = "not_found"
	StatusInvalidScript        ExecutionStatus = "invalid_script"
	StatusRuntimeError         ExecutionStatus = "runtime_error"
	StatusStructuredOutputMiss ExecutionStatus = "structured_output_missing"
	StatusThresholdFailed      ExecutionStatus = "threshold_failed"
)

// Contract describes the canonical Phase 1 k6 execution specification.
//
// This is the source of truth for:
//   - binary resolution
//   - command shape
//   - raw artifact layout
//   - script-path assumptions
//   - threshold source-of-truth behavior
type Contract struct {
	AdapterName              string
	DefaultTimeout           time.Duration
	StructuredSummaryName    string
	RawStdoutFileName        string
	RawStderrFileName        string
	VersionFileName          string
	WorkingDirectoryMode     string
	CloudExecutionOutOfScope bool
	MultipleScriptsOutOfScope bool
	StdoutPrimaryForbidden   bool
	ThresholdSourceOfTruth   string
	EnvInjectionDeferred     bool
	ResolutionPriority       []ResolutionMode
}

// DefaultContract returns the canonical k6 execution contract for Phase 1.
func DefaultContract() Contract {
	return Contract{
		AdapterName:               AdapterName,
		DefaultTimeout:            time.Duration(Metadata().DefaultTimeoutMs) * time.Millisecond,
		StructuredSummaryName:     "k6-summary.json",
		RawStdoutFileName:         "stdout.log",
		RawStderrFileName:         "stderr.log",
		VersionFileName:           "version.txt",
		WorkingDirectoryMode:      "project_root",
		CloudExecutionOutOfScope:  true,
		MultipleScriptsOutOfScope: true,
		StdoutPrimaryForbidden:    true,
		ThresholdSourceOfTruth:    "summary_export",
		EnvInjectionDeferred:      true,
		ResolutionPriority: []ResolutionMode{
			ResolutionGlobal,
		},
	}
}

// ArtifactPaths contains deterministic file-system paths used by the k6 execution spec.
type ArtifactPaths struct {
	WorkspaceRoot string
	OutputRoot    string

	TmpDir string
	RawDir string

	StructuredSummaryPath string
	StdoutPath            string
	StderrPath            string
	VersionPath           string
}

// CanonicalCommandSpec is the exact resolved command and artifact contract
// the runtime should use to invoke k6.
type CanonicalCommandSpec struct {
	WorkingDirectory string
	BinaryPath       string
	Args             []string
	Mode             ResolutionMode
	Timeout          time.Duration
	Artifacts        ArtifactPaths
	ScriptPath       string
}

// BuildArtifactPaths returns deterministic artifact locations rooted at outputRoot.
func BuildArtifactPaths(outputRoot, workspaceRoot string) ArtifactPaths {
	cleanOutput := filepath.Clean(outputRoot)
	cleanWorkspace := filepath.Clean(workspaceRoot)

	tmpDir := filepath.Join(cleanOutput, ".bb", "tmp")
	rawDir := filepath.Join(cleanOutput, ".bb", "raw", AdapterName)

	contract := DefaultContract()

	return ArtifactPaths{
		WorkspaceRoot:         cleanWorkspace,
		OutputRoot:            cleanOutput,
		TmpDir:                tmpDir,
		RawDir:                rawDir,
		StructuredSummaryPath: filepath.Join(tmpDir, contract.StructuredSummaryName),
		StdoutPath:            filepath.Join(rawDir, contract.RawStdoutFileName),
		StderrPath:            filepath.Join(rawDir, contract.RawStderrFileName),
		VersionPath:           filepath.Join(rawDir, contract.VersionFileName),
	}
}

// BuildCanonicalArgs builds the canonical k6 command arguments.
//
// Phase 1 rules:
//   - one script per adapter run
//   - one canonical summary-export JSON artifact
//   - local/CI process execution only
//   - no cloud/distributed mode
func BuildCanonicalArgs(scriptPath, structuredSummaryPath string) []string {
	return []string{
		"run",
		scriptPath,
		fmt.Sprintf("--summary-export=%s", structuredSummaryPath),
	}
}

// ValidateContract performs narrow internal validation on the default spec.
func ValidateContract(c Contract) error {
	if c.AdapterName != AdapterName {
		return fmt.Errorf("unexpected adapter name: %s", c.AdapterName)
	}
	if c.StructuredSummaryName == "" {
		return fmt.Errorf("structured summary file name must be defined")
	}
	if !strings.HasSuffix(strings.ToLower(c.StructuredSummaryName), ".json") {
		return fmt.Errorf("structured summary file must be json")
	}
	if c.DefaultTimeout <= 0 {
		return fmt.Errorf("default timeout must be positive")
	}
	if c.ThresholdSourceOfTruth != "summary_export" {
		return fmt.Errorf("threshold source of truth must be summary_export in Phase 1")
	}
	if len(c.ResolutionPriority) == 0 {
		return fmt.Errorf("resolution priority must not be empty")
	}
	return nil
}

// ValidateScriptPath applies the canonical Phase 1 script-path contract.
//
// Rules:
//   - script path is required
//   - exactly one script per run
//   - must be local
//   - workspace-relative or absolute local file path is allowed
//   - remote URLs are forbidden
func ValidateScriptPath(scriptPath string) error {
	trimmed := strings.TrimSpace(scriptPath)
	if trimmed == "" {
		return fmt.Errorf("script path is required")
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return fmt.Errorf("remote script URLs are forbidden in Phase 1")
	}

	return nil
}

// GlobalBinaryName returns the canonical global k6 executable name.
func GlobalBinaryName() string {
	if runtime.GOOS == "windows" {
		return "k6.exe"
	}
	return "k6"
}