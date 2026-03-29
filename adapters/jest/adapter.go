package jest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// adapter is the canonical Jest adapter implementation for Phase 1.
//
// Scope for WBS 1.8.1:
//   - deterministic binary resolution
//   - canonical command construction
//   - raw artifact capture
//   - explicit failure handling
//
// Out of scope for this file:
//   - parsing raw Jest JSON
//   - canonical normalization of Jest results
//   - policy interpretation
type adapter struct{}

// New returns the canonical Jest adapter as an sdk.Adapter.
func New() sdk.Adapter {
	return adapter{}
}

// Metadata returns the canonical static metadata for the Jest adapter.
func (adapter) Metadata() sdk.AdapterMetadata {
	return Metadata()
}

// CheckAvailability determines whether the Jest toolchain is available locally.
//
// Resolution order is deterministic:
//  1. local node_modules/.bin/jest
//  2. npx jest
//  3. global jest
func (adapter) CheckAvailability(ctx context.Context) sdk.Availability {
	workspaceRoot := resolveWorkspaceRoot(nil)

	resolved, err := resolveBinary(ctx, workspaceRoot)
	if err != nil {
		return sdk.Availability{
			Available:      false,
			ResolvedBinary: "",
		}
	}

	return sdk.Availability{
		Available:      true,
		ResolvedBinary: resolved.BinaryPath,
	}
}

// Version returns the best-effort Jest version using the canonical resolution chain.
//
// If the tool is not available or version lookup fails, UNKNOWN is returned.
// This keeps version probing non-fatal and avoids making availability/version a source
// of runtime crashes.
func (adapter) Version(ctx context.Context) (string, error) {
	workspaceRoot := resolveWorkspaceRoot(nil)

	resolved, err := resolveBinary(ctx, workspaceRoot)
	if err != nil {
		return "UNKNOWN", nil
	}

	args := versionArgs(resolved.Mode)
	cmd := exec.CommandContext(ctx, resolved.BinaryPath, args...)
	cmd.Dir = workspaceRoot

	out, err := cmd.Output()
	if err != nil {
		return "UNKNOWN", nil
	}

	version := strings.TrimSpace(string(out))
	if version == "" {
		return "UNKNOWN", nil
	}

	return version, nil
}

// Run executes Jest using the canonical execution specification and returns
// process-level execution truth only.
//
// Current design notes:
//   - Raw execution is implemented here for WBS 1.8.1.
//   - The return shape is intentionally limited by sdk.RunResult.
//   - If sdk.RunRequest later exposes stable fields such as WorkspaceRoot or OutputRoot,
//     resolveWorkspaceRoot / resolveOutputRoot can be simplified to direct field access.
func (a adapter) Run(ctx context.Context, req sdk.RunRequest) sdk.RunResult {
	workspaceRoot := resolveWorkspaceRoot(req)
	outputRoot := resolveOutputRoot(req, workspaceRoot)

	paths := buildArtifactPaths(outputRoot)
	if err := ensureDirs(paths); err != nil {
		return sdk.RunResult{
			Status:       sdk.ExecutionStatus("failed"),
			DurationMs:   0,
			ErrorMessage: fmt.Sprintf("prepare artifact directories: %v", err),
		}
	}

	resolved, err := resolveBinary(ctx, workspaceRoot)
	if err != nil {
		return sdk.RunResult{
			Status:       sdk.ExecutionStatus("not_found"),
			DurationMs:   0,
			ErrorMessage: err.Error(),
		}
	}

	spec := buildCanonicalCommandSpec(workspaceRoot, paths.ReportPath, resolved, Metadata().DefaultTimeoutMs)

	stdoutFile, err := os.Create(paths.StdoutPath)
	if err != nil {
		return sdk.RunResult{
			Status:       sdk.ExecutionStatus("failed"),
			DurationMs:   0,
			ErrorMessage: fmt.Sprintf("create stdout log: %v", err),
		}
	}
	defer stdoutFile.Close()

	stderrFile, err := os.Create(paths.StderrPath)
	if err != nil {
		return sdk.RunResult{
			Status:       sdk.ExecutionStatus("failed"),
			DurationMs:   0,
			ErrorMessage: fmt.Sprintf("create stderr log: %v", err),
		}
	}
	defer stderrFile.Close()

	started := time.Now().UTC()

	runCtx, cancel := context.WithTimeout(ctx, spec.Timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, spec.BinaryPath, spec.Args...)
	cmd.Dir = spec.WorkingDirectory
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	runErr := cmd.Run()
	completed := time.Now().UTC()
	durationMs := completed.Sub(started).Milliseconds()

	// Best-effort version capture.
	_ = writeVersionFile(ctx, workspaceRoot, resolved, paths.VersionPath)

	switch {
	case errors.Is(runCtx.Err(), context.DeadlineExceeded):
		return sdk.RunResult{
			Status:       sdk.ExecutionStatus("timed_out"),
			DurationMs:   durationMs,
			ErrorMessage: "jest execution timed out",
		}

	case runErr == nil:
		// Note:
		// A successful process execution does not guarantee the report exists.
		// Missing report handling belongs to parser/normalization flow later.
		return sdk.RunResult{
			Status:       sdk.ExecutionStatus("completed"),
			DurationMs:   durationMs,
			ErrorMessage: "",
		}

	default:
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			// Important:
			// failing tests are not runtime crashes; they are valid execution results.
			return sdk.RunResult{
				Status:       sdk.ExecutionStatus("failed"),
				DurationMs:   durationMs,
				ErrorMessage: fmt.Sprintf("jest exited with non-zero status: %d", exitErr.ExitCode()),
			}
		}

		return sdk.RunResult{
			Status:       sdk.ExecutionStatus("failed"),
			DurationMs:   durationMs,
			ErrorMessage: runErr.Error(),
		}
	}
}

// Normalize transforms raw execution into canonical normalized JSON.
//
// For WBS 1.8.1 this remains a deterministic, schema-compatible placeholder.
// Real Jest parsing and canonical normalization belong to WBS 1.8.2 and 1.8.3.
func (adapter) Normalize(context.Context, sdk.NormalizationInput) sdk.NormalizedResult {
	payload := map[string]any{
		"schema_version": "0.1",
		"adapter": map[string]any{
			"name":    "jest",
			"type":    "unit",
			"version": "UNKNOWN",
		},
		"execution": map[string]any{
			"status":      "unavailable",
			"duration_ms": 0,
		},
		"summary": map[string]any{
			"status":  "unknown",
			"total":   0,
			"passed":  0,
			"failed":  0,
			"skipped": 0,
		},
		"evidence": map[string]any{
			"complete": false,
			"issues": []map[string]any{
				{
					"code":    "NORMALIZATION_NOT_IMPLEMENTED",
					"message": "jest normalization will be implemented in subsequent tasks",
				},
			},
		},
		"artifacts": map[string]any{
			"raw_stdout_path":      "",
			"raw_stderr_path":      "",
			"raw_tool_output_path": "",
		},
		"extensions": map[string]any{
			"adapter_specific": map[string]any{},
		},
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		// Marshal failure here is extremely unlikely, but we still return
		// a deterministic fallback instead of panicking.
		return sdk.NormalizedResult(`{"schema_version":"0.1","adapter":{"name":"jest","type":"unit","version":"UNKNOWN"},"execution":{"status":"unavailable","duration_ms":0},"summary":{"status":"unknown","total":0,"passed":0,"failed":0,"skipped":0},"evidence":{"complete":false,"issues":[{"code":"NORMALIZATION_NOT_IMPLEMENTED","message":"jest normalization will be implemented in subsequent tasks"}]},"artifacts":{"raw_stdout_path":"","raw_stderr_path":"","raw_tool_output_path":""},"extensions":{"adapter_specific":{}}}`)
	}

	return sdk.NormalizedResult(encoded)
}

// resolvedBinary represents the chosen Jest execution strategy.
type resolvedBinary struct {
	Mode       ResolutionMode
	BinaryPath string
	ArgsPrefix []string
}

// ResolutionMode identifies how the adapter resolved the Jest executable.
type ResolutionMode string

const (
	resolutionLocal  ResolutionMode = "local_binary"
	resolutionNPX    ResolutionMode = "npx"
	resolutionGlobal ResolutionMode = "global_binary"
)

// canonicalCommandSpec defines the fully resolved deterministic command to run.
type canonicalCommandSpec struct {
	WorkingDirectory string
	BinaryPath       string
	Args             []string
	Timeout          time.Duration
	ReportPath       string
}

// artifactPaths captures deterministic raw artifact locations.
type artifactPaths struct {
	OutputRoot  string
	TmpDir      string
	RawDir      string
	StdoutPath  string
	StderrPath  string
	ReportPath  string
	VersionPath string
}

// buildCanonicalCommandSpec builds the canonical Jest invocation.
//
// Canonical rules:
//   - always use --json
//   - always use --outputFile=<deterministic path>
//   - do not rely on stdout as primary structured output
func buildCanonicalCommandSpec(
	workspaceRoot string,
	reportPath string,
	resolved resolvedBinary,
	timeoutMs int,
) canonicalCommandSpec {
	args := make([]string, 0, 8)
	args = append(args, resolved.ArgsPrefix...)
	args = append(args,
		"--json",
		fmt.Sprintf("--outputFile=%s", reportPath),
	)

	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = time.Duration(Metadata().DefaultTimeoutMs) * time.Millisecond
	}

	return canonicalCommandSpec{
		WorkingDirectory: workspaceRoot,
		BinaryPath:       resolved.BinaryPath,
		Args:             args,
		Timeout:          timeout,
		ReportPath:       reportPath,
	}
}

// buildArtifactPaths returns deterministic artifact locations rooted at outputRoot.
func buildArtifactPaths(outputRoot string) artifactPaths {
	cleanRoot := filepath.Clean(outputRoot)
	tmpDir := filepath.Join(cleanRoot, ".bb", "tmp")
	rawDir := filepath.Join(cleanRoot, ".bb", "raw", "jest")

	return artifactPaths{
		OutputRoot:  cleanRoot,
		TmpDir:      tmpDir,
		RawDir:      rawDir,
		StdoutPath:  filepath.Join(rawDir, "stdout.log"),
		StderrPath:  filepath.Join(rawDir, "stderr.log"),
		ReportPath:  filepath.Join(tmpDir, "jest-report.json"),
		VersionPath: filepath.Join(rawDir, "version.txt"),
	}
}

// ensureDirs creates the deterministic artifact directories.
func ensureDirs(paths artifactPaths) error {
	for _, dir := range []string{paths.TmpDir, paths.RawDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// resolveBinary resolves Jest in the required deterministic order:
//  1. local node_modules/.bin/jest
//  2. npx jest
//  3. global jest
func resolveBinary(_ context.Context, workspaceRoot string) (resolvedBinary, error) {
	localJest := filepath.Join(workspaceRoot, "node_modules", ".bin", localJestBinaryName())
	if fileExists(localJest) {
		return resolvedBinary{
			Mode:       resolutionLocal,
			BinaryPath: localJest,
			ArgsPrefix: nil,
		}, nil
	}

	npxPath, err := exec.LookPath(npxBinaryName())
	if err == nil {
		return resolvedBinary{
			Mode:       resolutionNPX,
			BinaryPath: npxPath,
			ArgsPrefix: []string{"jest"},
		}, nil
	}

	globalJest, err := exec.LookPath(globalJestBinaryName())
	if err == nil {
		return resolvedBinary{
			Mode:       resolutionGlobal,
			BinaryPath: globalJest,
			ArgsPrefix: nil,
		}, nil
	}

	return resolvedBinary{}, fmt.Errorf("jest binary not found: checked local binary, npx, and global binary")
}

// writeVersionFile captures the best-effort version output for traceability.
// Failure here is intentionally non-fatal.
func writeVersionFile(ctx context.Context, workspaceRoot string, resolved resolvedBinary, outputPath string) error {
	args := versionArgs(resolved.Mode)

	cmd := exec.CommandContext(ctx, resolved.BinaryPath, args...)
	cmd.Dir = workspaceRoot

	out, err := cmd.Output()
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, out, 0o644)
}

func versionArgs(mode ResolutionMode) []string {
	if mode == resolutionNPX {
		return []string{"jest", "--version"}
	}
	return []string{"--version"}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func npxBinaryName() string {
	if runtime.GOOS == "windows" {
		return "npx.cmd"
	}
	return "npx"
}

func localJestBinaryName() string {
	if runtime.GOOS == "windows" {
		return "jest.cmd"
	}
	return "jest"
}

func globalJestBinaryName() string {
	if runtime.GOOS == "windows" {
		return "jest.cmd"
	}
	return "jest"
}

// resolveWorkspaceRoot tries to extract a meaningful workspace root from the request.
// If the SDK request type does not yet expose one, it falls back to the current directory.
//
// This reflection-based helper is deliberate:
// it avoids coupling this implementation to a speculative RunRequest field name
// while still being forward-compatible with future SDK evolution.
func resolveWorkspaceRoot(req any) string {
	if root := stringField(req, "WorkspaceRoot", "ProjectRoot", "RootDir", "WorkDir"); root != "" {
		return filepath.Clean(root)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}

	return filepath.Clean(cwd)
}

// resolveOutputRoot prefers an explicit output root if one exists on the request.
// Otherwise it defaults to the workspace root.
func resolveOutputRoot(req any, workspaceRoot string) string {
	if out := stringField(req, "OutputRoot", "ArtifactsRoot", "RunOutputDir"); out != "" {
		return filepath.Clean(out)
	}
	return workspaceRoot
}

// stringField extracts the first non-empty exported string field by name.
func stringField(v any, names ...string) string {
	if v == nil {
		return ""
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return ""
	}

	for _, name := range names {
		field := rv.FieldByName(name)
		if field.IsValid() && field.Kind() == reflect.String && field.CanInterface() {
			value := strings.TrimSpace(field.String())
			if value != "" {
				return value
			}
		}
	}

	return ""
}
