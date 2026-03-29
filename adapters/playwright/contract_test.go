package playwright

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

	if filepath.Base(a.StructuredReportPath) != "playwright-report.json" {
		t.Fatalf("unexpected structured report name: %q", filepath.Base(a.StructuredReportPath))
	}
	if filepath.Base(a.StdoutPath) != "stdout.log" {
		t.Fatalf("unexpected stdout artifact name: %q", filepath.Base(a.StdoutPath))
	}
	if filepath.Base(a.StderrPath) != "stderr.log" {
		t.Fatalf("unexpected stderr artifact name: %q", filepath.Base(a.StderrPath))
	}
}

func TestBuildCanonicalArgs_UsesJSONReporter(t *testing.T) {
	t.Parallel()

	args := BuildCanonicalArgs("/repo/.bb/tmp/playwright-report.json")

	foundReporter := false
	for _, arg := range args {
		if arg == "--reporter=json" {
			foundReporter = true
			break
		}
	}

	if !foundReporter {
		t.Fatal("expected canonical args to include --reporter=json")
	}
}

func TestLocalBinaryPath_IsStable(t *testing.T) {
	t.Parallel()

	root := "/repo"
	if runtime.GOOS == "windows" {
		root = `C:\repo`
	}

	got := LocalBinaryPath(root)
	if filepath.Base(got) == "" {
		t.Fatal("expected stable local binary path")
	}
}