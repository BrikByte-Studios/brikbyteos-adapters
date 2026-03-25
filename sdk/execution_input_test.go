package sdk

import (
	"testing"
	"time"
)

type testLogger struct{}

func (testLogger) Info(string, ...any)  {}
func (testLogger) Warn(string, ...any)  {}
func (testLogger) Error(string, ...any) {}

func validRunRequest() RunRequest {
	return RunRequest{
		RunID:                "20260325T180000Z-a91c2f",
		WorkspaceRoot:        "/workspace/repo",
		ArtifactsRoot:        "/workspace/repo/.bb/runs/20260325T180000Z-a91c2f",
		AdapterArtifactsPath: "raw/jest",
		Environment:          LogicalEnvironmentDev,
		ExecutionMode:        ExecutionModeAll,
		Timeout:              30 * time.Second,
		Logger:               testLogger{},
		AdapterOptions: map[string]any{
			"coverage": true,
		},
		EnvVars: []string{"CI=true", "NODE_ENV=test"},
	}
}

func TestRunRequest_Validate_Valid(t *testing.T) {
	t.Parallel()

	req := validRunRequest()
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}
}

func TestRunRequest_Validate_InvalidEnvironmentFails(t *testing.T) {
	t.Parallel()

	req := validRunRequest()
	req.Environment = LogicalEnvironment("local")

	if err := req.Validate(); err == nil {
		t.Fatal("expected invalid environment to fail validation")
	}
}

func TestRunRequest_Validate_InvalidExecutionModeFails(t *testing.T) {
	t.Parallel()

	req := validRunRequest()
	req.ExecutionMode = ExecutionMode("single")

	if err := req.Validate(); err == nil {
		t.Fatal("expected invalid execution mode to fail validation")
	}
}

func TestRunRequest_Validate_InvalidTimeoutFails(t *testing.T) {
	t.Parallel()

	req := validRunRequest()
	req.Timeout = 0

	if err := req.Validate(); err == nil {
		t.Fatal("expected invalid timeout to fail validation")
	}
}

func TestRunRequest_Validate_RelativeWorkspaceRootFails(t *testing.T) {
	t.Parallel()

	req := validRunRequest()
	req.WorkspaceRoot = "./repo"

	if err := req.Validate(); err == nil {
		t.Fatal("expected relative workspace_root to fail validation")
	}
}

func TestRunRequest_Validate_AbsoluteAdapterArtifactsPathFails(t *testing.T) {
	t.Parallel()

	req := validRunRequest()
	req.AdapterArtifactsPath = "/tmp/raw/jest"

	if err := req.Validate(); err == nil {
		t.Fatal("expected absolute adapter_artifacts_path to fail validation")
	}
}

func TestRunRequest_Validate_EscapingAdapterArtifactsPathFails(t *testing.T) {
	t.Parallel()

	req := validRunRequest()
	req.AdapterArtifactsPath = "../outside"

	if err := req.Validate(); err == nil {
		t.Fatal("expected escaping adapter_artifacts_path to fail validation")
	}
}

func TestRunRequest_Validate_InvalidEnvVarFails(t *testing.T) {
	t.Parallel()

	req := validRunRequest()
	req.EnvVars = []string{"CI"}

	if err := req.Validate(); err == nil {
		t.Fatal("expected invalid env var to fail validation")
	}
}

func TestRunRequest_Clone_DeepCopiesMutableFields(t *testing.T) {
	t.Parallel()

	req := validRunRequest()
	cloned := req.Clone()

	cloned.AdapterOptions["coverage"] = false
	cloned.EnvVars[0] = "CI=false"

	if req.AdapterOptions["coverage"] != true {
		t.Fatal("expected original adapter options to remain unchanged")
	}
	if req.EnvVars[0] != "CI=true" {
		t.Fatal("expected original env vars to remain unchanged")
	}
}

func TestNormalizeLogicalEnvironment(t *testing.T) {
	t.Parallel()

	if got := NormalizeLogicalEnvironment("DEV"); got != LogicalEnvironmentDev {
		t.Fatalf("expected dev, got %q", got)
	}
	if got := NormalizeLogicalEnvironment("weird"); got != LogicalEnvironmentUnknown {
		t.Fatalf("expected unknown, got %q", got)
	}
}
