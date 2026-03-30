package trivy

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ResolutionMode identifies how the Trivy CLI was resolved.
type ResolutionMode string

const (
	ResolutionGlobal ResolutionMode = "global_binary"
	ResolutionLocal  ResolutionMode = "local_binary"
	ResolutionNone   ResolutionMode = "not_found"
)

// TargetType is the approved Phase 1 Trivy target type.
//
// Phase 1 recommendation:
//   - filesystem target only
//
// This keeps the contract deterministic and avoids mixing image/repo/fs semantics
// in the first execution slice.
type TargetType string

const (
	TargetTypeFilesystem TargetType = "filesystem"
)

// ExecutionStatus defines canonical execution outcomes at the execution-spec layer.
type ExecutionStatus string

const (
	StatusCompleted            ExecutionStatus = "completed"
	StatusFailed               ExecutionStatus = "failed"
	StatusTimedOut             ExecutionStatus = "timed_out"
	StatusNotFound             ExecutionStatus = "not_found"
	StatusInvalidTarget        ExecutionStatus = "invalid_target"
	StatusPermissionError      ExecutionStatus = "permission_error"
	StatusRuntimeError         ExecutionStatus = "runtime_error"
	StatusStructuredOutputMiss ExecutionStatus = "structured_output_missing"
)

// ExitCodePolicy defines how findings are treated relative to process exit status.
//
// Recommended Phase 1 behavior:
//   - findings do not cause execution failure by default
//   - Trivy should emit JSON report with exit-code 0 so findings are handled later
//     in parser/normalizer/policy layers
type ExitCodePolicy string

const (
	ExitCodePolicyFindingsDoNotFail ExitCodePolicy = "findings_do_not_fail"
)

// SeverityFilteringMode defines how severities are handled at execution time.
//
// Recommended Phase 1 behavior:
//   - do not destructively filter at execution time
//   - preserve full finding set in raw JSON
//   - let downstream policy logic reason over severities
type SeverityFilteringMode string

const (
	SeverityFilteringNone SeverityFilteringMode = "none"
)

// Contract describes the canonical Phase 1 Trivy execution specification.
type Contract struct {
	AdapterName               string
	DefaultTimeout            time.Duration
	StructuredReportName      string
	RawStdoutFileName         string
	RawStderrFileName         string
	VersionFileName           string
	WorkingDirectoryMode      string
	StdoutPrimaryForbidden    bool
	SupportedTargetType       TargetType
	ExitCodePolicy            ExitCodePolicy
	SeverityFilteringMode     SeverityFilteringMode
	OfflineBehaviorOutOfScope bool
	DBUpdateBehaviorDeferred  bool
	ResolutionPriority        []ResolutionMode
}

// DefaultContract returns the canonical Trivy execution contract for Phase 1.
func DefaultContract() Contract {
	return Contract{
		AdapterName:               AdapterName,
		DefaultTimeout:            time.Duration(Metadata().DefaultTimeoutMs) * time.Millisecond,
		StructuredReportName:      "trivy-report.json",
		RawStdoutFileName:         "stdout.log",
		RawStderrFileName:         "stderr.log",
		VersionFileName:           "version.txt",
		WorkingDirectoryMode:      "project_root",
		StdoutPrimaryForbidden:    true,
		SupportedTargetType:       TargetTypeFilesystem,
		ExitCodePolicy:            ExitCodePolicyFindingsDoNotFail,
		SeverityFilteringMode:     SeverityFilteringNone,
		OfflineBehaviorOutOfScope: true,
		DBUpdateBehaviorDeferred:  true,
		ResolutionPriority: []ResolutionMode{
			ResolutionGlobal,
		},
	}
}

// ArtifactPaths contains deterministic file-system paths used by the Trivy execution spec.
type ArtifactPaths struct {
	WorkspaceRoot string
	OutputRoot    string

	TmpDir string
	RawDir string

	StructuredReportPath string
	StdoutPath           string
	StderrPath           string
	VersionPath          string
}

// CanonicalCommandSpec is the exact resolved command and artifact contract
// the runtime should use to invoke Trivy.
type CanonicalCommandSpec struct {
	WorkingDirectory string
	BinaryPath       string
	Args             []string
	Mode             ResolutionMode
	Timeout          time.Duration
	Artifacts        ArtifactPaths
	TargetPath       string
	TargetType       TargetType
}

// BuildArtifactPaths returns deterministic artifact locations rooted at outputRoot.
func BuildArtifactPaths(outputRoot, workspaceRoot string) ArtifactPaths {
	cleanOutput := filepath.Clean(outputRoot)
	cleanWorkspace := filepath.Clean(workspaceRoot)

	tmpDir := filepath.Join(cleanOutput, ".bb", "tmp")
	rawDir := filepath.Join(cleanOutput, ".bb", "raw", AdapterName)

	contract := DefaultContract()

	return ArtifactPaths{
		WorkspaceRoot:        cleanWorkspace,
		OutputRoot:           cleanOutput,
		TmpDir:               tmpDir,
		RawDir:               rawDir,
		StructuredReportPath: filepath.Join(tmpDir, contract.StructuredReportName),
		StdoutPath:           filepath.Join(rawDir, contract.RawStdoutFileName),
		StderrPath:           filepath.Join(rawDir, contract.RawStderrFileName),
		VersionPath:          filepath.Join(rawDir, contract.VersionFileName),
	}
}

// BuildCanonicalArgs builds the canonical Trivy command arguments.
//
// Phase 1 rules:
//   - filesystem target only
//   - one canonical JSON report
//   - findings should not force non-zero execution status by default
//   - no destructive severity filtering at execution time
func BuildCanonicalArgs(targetPath, structuredReportPath string) []string {
	return []string{
		"fs",
		"--format", "json",
		"--output", structuredReportPath,
		"--exit-code", "0",
		targetPath,
	}
}

// ValidateContract performs narrow internal validation on the default spec.
func ValidateContract(c Contract) error {
	if c.AdapterName != AdapterName {
		return fmt.Errorf("unexpected adapter name: %s", c.AdapterName)
	}
	if c.StructuredReportName == "" {
		return fmt.Errorf("structured report file name must be defined")
	}
	if !strings.HasSuffix(strings.ToLower(c.StructuredReportName), ".json") {
		return fmt.Errorf("structured report file must be json")
	}
	if c.DefaultTimeout <= 0 {
		return fmt.Errorf("default timeout must be positive")
	}
	if c.SupportedTargetType != TargetTypeFilesystem {
		return fmt.Errorf("supported target type must be filesystem in Phase 1")
	}
	if c.ExitCodePolicy != ExitCodePolicyFindingsDoNotFail {
		return fmt.Errorf("exit code policy must preserve findings without execution failure in Phase 1")
	}
	if c.SeverityFilteringMode != SeverityFilteringNone {
		return fmt.Errorf("severity filtering must be none in Phase 1")
	}
	if len(c.ResolutionPriority) == 0 {
		return fmt.Errorf("resolution priority must not be empty")
	}
	return nil
}

// ValidateTargetPath applies the canonical Phase 1 target-path contract.
//
// Rules:
//   - target path is required
//   - target must be local
//   - remote URLs are forbidden
//   - the execution spec supports filesystem target semantics only in Phase 1
func ValidateTargetPath(targetPath string) error {
	trimmed := strings.TrimSpace(targetPath)
	if trimmed == "" {
		return fmt.Errorf("target path is required")
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return fmt.Errorf("remote targets are forbidden in Phase 1")
	}

	return nil
}

// GlobalBinaryName returns the canonical global Trivy executable name.
func GlobalBinaryName() string {
	if runtime.GOOS == "windows" {
		return "trivy.exe"
	}
	return "trivy"
}
