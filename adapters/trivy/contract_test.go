package trivy

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

	if filepath.Base(a.StructuredReportPath) != "trivy-report.json" {
		t.Fatalf("unexpected structured report name: %q", filepath.Base(a.StructuredReportPath))
	}
	if filepath.Base(a.StdoutPath) != "stdout.log" {
		t.Fatalf("unexpected stdout artifact name: %q", filepath.Base(a.StdoutPath))
	}
	if filepath.Base(a.StderrPath) != "stderr.log" {
		t.Fatalf("unexpected stderr artifact name: %q", filepath.Base(a.StderrPath))
	}
}

func TestBuildCanonicalArgs_UsesJSONReportAndExitCodeZero(t *testing.T) {
	t.Parallel()

	args := BuildCanonicalArgs("services/api", "/repo/.bb/tmp/trivy-report.json")

	want := []string{
		"fs",
		"--format", "json",
		"--output", "/repo/.bb/tmp/trivy-report.json",
		"--exit-code", "0",
		"services/api",
	}

	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected canonical args\nwant=%v\ngot=%v", want, args)
	}
}

func TestValidateTargetPath_AcceptsLocalTarget(t *testing.T) {
	t.Parallel()

	if err := ValidateTargetPath("services/api"); err != nil {
		t.Fatalf("expected valid local target path, got error: %v", err)
	}
}

func TestValidateTargetPath_RejectsEmpty(t *testing.T) {
	t.Parallel()

	if err := ValidateTargetPath(""); err == nil {
		t.Fatal("expected error for empty target path")
	}
}

func TestValidateTargetPath_RejectsRemoteURL(t *testing.T) {
	t.Parallel()

	if err := ValidateTargetPath("https://example.com/repo"); err == nil {
		t.Fatal("expected remote target URL to be rejected")
	}
}