package k6

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestDefaultContract_IsValid(t *testing.T) {
	t.Parallel()

	contract := DefaultContract()
	if err := ValidateContract(contract); err != nil {
		t.Fatalf("expected valid default contract, got error: %v", err)
	}
}

func TestBuildArtifactPaths_IsDeterministic(t *testing.T) {
	t.Parallel()

	root := "/repo"
	if runtime.GOOS == "windows" {
		root = `C:\repo`
	}

	a := BuildArtifactPaths(root, root)
	b := BuildArtifactPaths(root, root)

	if !reflect.DeepEqual(a, b) {
		t.Fatal("expected artifact paths to be deterministic")
	}

	if filepath.Base(a.StructuredSummaryPath) != "k6-summary.json" {
		t.Fatalf("unexpected structured summary name: %q", filepath.Base(a.StructuredSummaryPath))
	}
	if filepath.Base(a.StdoutPath) != "stdout.log" {
		t.Fatalf("unexpected stdout artifact name: %q", filepath.Base(a.StdoutPath))
	}
	if filepath.Base(a.StderrPath) != "stderr.log" {
		t.Fatalf("unexpected stderr artifact name: %q", filepath.Base(a.StderrPath))
	}
}

func TestBuildCanonicalArgs_UsesSummaryExport(t *testing.T) {
	t.Parallel()

	args := BuildCanonicalArgs("tests/load/auth.k6.js", "/repo/.bb/tmp/k6-summary.json")

	want := []string{
		"run",
		"tests/load/auth.k6.js",
		"--summary-export=/repo/.bb/tmp/k6-summary.json",
	}

	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected canonical args\nwant=%v\ngot=%v", want, args)
	}
}

func TestValidateScriptPath_AcceptsLocalScript(t *testing.T) {
	t.Parallel()

	if err := ValidateScriptPath("tests/load/auth.k6.js"); err != nil {
		t.Fatalf("expected valid local script path, got error: %v", err)
	}
}

func TestValidateScriptPath_RejectsEmpty(t *testing.T) {
	t.Parallel()

	if err := ValidateScriptPath(""); err == nil {
		t.Fatal("expected error for empty script path")
	}
}

func TestValidateScriptPath_RejectsRemoteURL(t *testing.T) {
	t.Parallel()

	if err := ValidateScriptPath("https://example.com/script.js"); err == nil {
		t.Fatal("expected remote script URL to be rejected")
	}
}