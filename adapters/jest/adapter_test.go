package jest

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuildArtifactPaths_IsDeterministic(t *testing.T) {
	t.Parallel()

	root := "/repo"
	if runtime.GOOS == "windows" {
		root = `C:\repo`
	}

	a := buildArtifactPaths(root)
	b := buildArtifactPaths(root)

	if a.ReportPath != b.ReportPath {
		t.Fatalf("expected deterministic report path, got %q and %q", a.ReportPath, b.ReportPath)
	}
	if filepath.Base(a.StdoutPath) != "stdout.log" {
		t.Fatalf("unexpected stdout artifact name: %q", filepath.Base(a.StdoutPath))
	}
	if filepath.Base(a.StderrPath) != "stderr.log" {
		t.Fatalf("unexpected stderr artifact name: %q", filepath.Base(a.StderrPath))
	}
	if filepath.Base(a.ReportPath) != "jest-report.json" {
		t.Fatalf("unexpected report artifact name: %q", filepath.Base(a.ReportPath))
	}
}

func TestBuildCanonicalCommandSpec_UsesJSONAndOutputFile(t *testing.T) {
	t.Parallel()

	spec := buildCanonicalCommandSpec(
		"/repo",
		"/repo/.bb/tmp/jest-report.json",
		JestResolution{
			Kind:       ResolutionLocal,
			BinaryPath: "/repo/node_modules/.bin/jest",
			ArgsPrefix: nil,
		},
		30000,
	)

	foundJSON := false
	foundOutputFile := false

	for _, arg := range spec.Args {
		if arg == "--json" {
			foundJSON = true
		}
		if arg == "--outputFile=/repo/.bb/tmp/jest-report.json" {
			foundOutputFile = true
		}
	}

	if !foundJSON {
		t.Fatal("expected canonical command to include --json")
	}
	if !foundOutputFile {
		t.Fatal("expected canonical command to include deterministic --outputFile")
	}
}

func TestResolveWorkspaceRoot_FallsBackSafely(t *testing.T) {
	t.Parallel()

	root := resolveWorkspaceRoot(nil)
	if root == "" {
		t.Fatal("expected non-empty workspace root fallback")
	}
}

func TestResolveWorkspaceRoot_UsesKnownFieldNames(t *testing.T) {
	t.Parallel()

	type fakeRequest struct {
		WorkspaceRoot string
	}

	root := resolveWorkspaceRoot(fakeRequest{WorkspaceRoot: "/tmp/project"})
	if root != filepath.Clean("/tmp/project") {
		t.Fatalf("expected explicit workspace root to be used, got %q", root)
	}
}

func TestResolveOutputRoot_UsesExplicitFieldWhenPresent(t *testing.T) {
	t.Parallel()

	type fakeRequest struct {
		OutputRoot string
	}

	out := resolveOutputRoot(fakeRequest{OutputRoot: "/tmp/out"}, "/tmp/project")
	if out != filepath.Clean("/tmp/out") {
		t.Fatalf("expected explicit output root, got %q", out)
	}
}
