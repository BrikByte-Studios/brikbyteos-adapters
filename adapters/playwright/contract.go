package playwright

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ResolutionKind is the canonical resolution taxonomy for Playwright executables.
type ResolutionKind string

const (
	ResolutionLocal  ResolutionKind = "local_binary"			// The highest priority resolution mode: directly resolve the Playwright binary in the workspace's node_modules/.bin directory. This is the most common and recommended setup for Playwright users, as it ensures version consistency with project dependencies.
	ResolutionNPX    ResolutionKind = "npx"						// The next priority resolution mode: use npx to execute Playwright. This allows users to leverage npx's resolution logic, which can find Playwright in the workspace or globally. This is a convenient fallback for users who have Playwright installed but not directly in node_modules/.bin.		
	ResolutionGlobal ResolutionKind = "global_binary"			// The lowest priority resolution mode: directly resolve a globally installed Playwright binary. This is a less common setup and is generally not recommended due to potential version mismatches, but it serves as a final fallback for users who have Playwright installed globally.
)

// ExecutionStatus defines canonical execution outcomes at the execution-spec layer.
//
// Notes:
//   - "failed" means Playwright executed and returned non-zero, typically due to test failure.
//   - "dependency_failure" is reserved for missing browser/runtime prerequisites.
//   - "structured_output_missing" indicates execution ended but canonical report output was absent.
type ExecutionStatus string

const (
	StatusCompleted            ExecutionStatus = "completed"
	StatusFailed               ExecutionStatus = "failed"
	StatusTimedOut             ExecutionStatus = "timed_out"
	StatusNotFound             ExecutionStatus = "not_found"
	StatusDependencyFailure    ExecutionStatus = "dependency_failure"
	StatusStructuredOutputMiss ExecutionStatus = "structured_output_missing"
)

// Contract describes the canonical execution specification for Playwright.
//
// This is the source of truth for:
//   - resolution priority
//   - command shape
//   - artifact layout
//   - browser/runtime assumptions
//   - failure classification policy
type Contract struct {
	AdapterName               string
	DefaultTimeout            time.Duration
	HeadlessDefault           bool
	StructuredReporter        string
	StructuredOutputFileName  string
	RawStdoutFileName         string
	RawStderrFileName         string
	VersionFileName           string
	WorkingDirectoryMode      string
	AutoInstallForbidden      bool
	HTMLReportOutOfScope      bool
	TraceCaptureDeferred      bool
	ScreenshotCaptureDeferred bool
	VideoCaptureDeferred      bool
	BrowserInstallRequired    bool
	ResolutionPriority		  []ResolutionKind        
}

// DefaultContract returns the canonical Playwright execution contract for Phase 1.
//
// Design intent:
//   - one structured machine-readable reporter
//   - one deterministic output file
//   - no hidden installs
//   - headless by default
func DefaultContract() Contract {
	return Contract{
		AdapterName:               AdapterName,
		DefaultTimeout:            time.Duration(Metadata().DefaultTimeoutMs) * time.Millisecond,
		HeadlessDefault:           true,
		StructuredReporter:        "json",
		StructuredOutputFileName:  "playwright-report.json",
		RawStdoutFileName:         "stdout.log",
		RawStderrFileName:         "stderr.log",
		VersionFileName:           "version.txt",
		WorkingDirectoryMode:      "project_root",
		AutoInstallForbidden:      true,
		HTMLReportOutOfScope:      true,
		TraceCaptureDeferred:      true,
		ScreenshotCaptureDeferred: true,
		VideoCaptureDeferred:      true,
		BrowserInstallRequired:    true,
		ResolutionPriority: []ResolutionKind{
			ResolutionLocal,
			ResolutionNPX,
			ResolutionGlobal,
		},
	}
}

// ArtifactPaths contains deterministic file-system paths used by the Playwright execution spec.
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
// the runtime should use to invoke Playwright.
type CanonicalCommandSpec struct {
	WorkingDirectory string
	BinaryPath       string
	Args             []string
	Kind  			 ResolutionKind
	Timeout          time.Duration
	Artifacts        ArtifactPaths
}

// BuildArtifactPaths returns deterministic artifact locations rooted at outputRoot.
//
// Path rules:
//   - all paths are workspace/output-root relative
//   - stable file names
//   - no random temp names
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
		StructuredReportPath: filepath.Join(tmpDir, contract.StructuredOutputFileName),
		StdoutPath:           filepath.Join(rawDir, contract.RawStdoutFileName),
		StderrPath:           filepath.Join(rawDir, contract.RawStderrFileName),
		VersionPath:          filepath.Join(rawDir, contract.VersionFileName),
	}
}

// BuildCanonicalArgs builds the canonical Playwright command arguments.
//
// Phase 1 rules:
//   - JSON reporter is mandatory
//   - structured output file is deterministic
//   - headless execution is assumed
//   - HTML reporter is excluded from canonical parsing flow
func BuildCanonicalArgs(structuredReportPath string) []string {
	args := []string{
		"test",
		"--reporter=json",
	}

	// Playwright's JSON reporter writes to stdout by default in many setups.
	// For BrikByteOS we enforce one canonical structured report artifact path
	// via an environment-aware or wrapper-aware contract later if needed.
	//
	// This explicit path token is kept in the spec/builder layer so downstream
	// runtime implementations can standardize how they materialize the report.
	args = append(args, fmt.Sprintf("--output=%s", filepath.Dir(structuredReportPath)))

	return args
}

// ValidateContract performs narrow internal validation on the default spec.
// This guards accidental drift in constants and file naming.
func ValidateContract(c Contract) error {
	if c.AdapterName != AdapterName {
		return fmt.Errorf("unexpected adapter name: %s", c.AdapterName)
	}
	if c.StructuredReporter == "" {
		return fmt.Errorf("structured reporter must be defined")
	}
	if strings.ToLower(c.StructuredReporter) != "json" {
		return fmt.Errorf("structured reporter must be json for Phase 1")
	}
	if c.StructuredOutputFileName == "" {
		return fmt.Errorf("structured output file name must be defined")
	}
	if c.DefaultTimeout <= 0 {
		return fmt.Errorf("default timeout must be positive")
	}
	if len(c.ResolutionPriority) == 0 {
		return fmt.Errorf("resolution priority must not be empty")
	}
	return nil
}

// LocalBinaryPath returns the canonical local Playwright executable path.
func LocalBinaryPath(workspaceRoot string) string {
	bin := "playwright"
	if runtime.GOOS == "windows" {
		bin = "playwright.cmd"
	}
	return filepath.Join(workspaceRoot, "node_modules", ".bin", bin)
}

// GlobalBinaryName returns the canonical global Playwright executable name.
func GlobalBinaryName() string {
	if runtime.GOOS == "windows" {
		return "playwright.cmd"
	}
	return "playwright"
}

// NPXBinaryName returns the canonical npx executable name.
func NPXBinaryName() string {
	if runtime.GOOS == "windows" {
		return "npx.cmd"
	}
	return "npx"
}
