package sdk

import (
	"context"
	"testing"
	"time"
)

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

type testAdapter struct {
	meta AdapterMetadata
}

func (a testAdapter) Metadata() AdapterMetadata {
	return a.meta
}

func (a testAdapter) CheckAvailability(context.Context) Availability {
	return Availability{Available: true, ResolvedBinary: "/usr/bin/test"}
}

func (a testAdapter) Version(context.Context) (string, error) {
	return "1.0.0", nil
}

func (a testAdapter) Run(context.Context, RunRequest) RunResult {
	code := 0
	return RunResult{
		Status:     ExecutionStatusCompleted,
		ExitCode:   &code,
		DurationMs: 10,
	}
}

func (a testAdapter) Normalize(context.Context, NormalizationInput) NormalizedResult {
	return []byte(`{"schema_version":"0.1"}`)
}

func TestValidateAdapter_Valid(t *testing.T) {
	t.Parallel()

	err := ValidateAdapter(testAdapter{
		meta: AdapterMetadata{
			Name:             "jest",
			Type:             AdapterTypeUnit,
			Description:      "JavaScript unit test adapter",
			Order:            10,
			SupportedTool:    "jest",
			VersionCommand:   []string{"npx", "jest", "--version"},
			DefaultTimeoutMs: 30000,
			Aliases:          []string{"js-test"},
		},
	})
	if err != nil {
		t.Fatalf("expected valid adapter, got error: %v", err)
	}
}

func TestValidateAdapter_InvalidMetadataFails(t *testing.T) {
	t.Parallel()

	err := ValidateAdapter(testAdapter{
		meta: AdapterMetadata{
			Name: "",
			Type: AdapterTypeUnit,
		},
	})
	if err == nil {
		t.Fatal("expected invalid metadata to fail validation")
	}
}

func TestRunRequest_Validate(t *testing.T) {
	t.Parallel()

	req := RunRequest{
		RunID:                "20260324T180000Z-a91c2f",
		WorkspaceRoot:        "/workspace/project",
		ArtifactsRoot:        "/workspace/project/.bb/runs/20260324T180000Z-a91c2f",
		AdapterArtifactsPath: "raw/jest",
		Environment:          "dev",
		ExecutionMode:        "all",
		Timeout:              30 * time.Second,
		Logger:               noopLogger{},
		AdapterOptions:       map[string]any{"coverage": true},
		EnvVars:              []string{"CI=true"},
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid run request, got error: %v", err)
	}
}

func TestRunResult_Validate(t *testing.T) {
	t.Parallel()

	code := 1
	res := RunResult{
		Status:       ExecutionStatusFailed,
		ExitCode:     &code,
		DurationMs:   42,
		ErrorMessage: "process exited with non-zero status",
	}

	if err := res.Validate(); err != nil {
		t.Fatalf("expected valid run result, got error: %v", err)
	}
}
