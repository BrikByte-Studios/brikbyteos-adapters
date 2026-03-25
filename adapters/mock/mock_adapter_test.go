package mock

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

/*
Test suite for Mock Adapter.

Goals:
  - Validate Adapter SDK contract compliance
  - Ensure deterministic behavior
  - Verify execution → normalization pipeline correctness
  - Ensure no hidden side-effects or runtime dependencies

These tests act as:
  - reference implementation validation
  - guardrails for future adapter implementations
*/

// --- Helpers ---

type testLogger struct{}

func (testLogger) Info(string, ...any)  {}
func (testLogger) Warn(string, ...any)  {}
func (testLogger) Error(string, ...any) {}

// --- Tests ---

func TestMockAdapter_Metadata(t *testing.T) {
	t.Parallel()

	adapter := Adapter{}
	meta := adapter.Metadata()

	if err := meta.Validate(); err != nil {
		t.Fatalf("metadata validation failed: %v", err)
	}

	if meta.Name != "mock" {
		t.Errorf("expected name 'mock', got %q", meta.Name)
	}

	if meta.Order <= 0 {
		t.Errorf("expected positive order, got %d", meta.Order)
	}
}

func TestMockAdapter_Availability(t *testing.T) {
	t.Parallel()

	adapter := Adapter{}
	availability := adapter.CheckAvailability(context.Background())

	if !availability.Available {
		t.Fatal("expected mock adapter to always be available")
	}

	if availability.ResolvedBinary == "" {
		t.Error("expected resolved binary to be populated")
	}
}

func TestMockAdapter_Version(t *testing.T) {
	t.Parallel()

	adapter := Adapter{}

	version, err := adapter.Version(context.Background())
	if err != nil {
		t.Fatalf("version call failed: %v", err)
	}

	if version == "" {
		t.Error("expected non-empty version")
	}
}

func TestMockAdapter_Run_Deterministic(t *testing.T) {
	t.Parallel()

	adapter := Adapter{}

	req := sdk.RunRequest{
		RunID:                "20260325T180000Z-test",
		WorkspaceRoot:        "/workspace",
		ArtifactsRoot:        "/workspace/.bb/runs/test",
		AdapterArtifactsPath: "raw/mock",
		Environment:          "dev",
		ExecutionMode:        "all",
		Timeout:              5 * time.Second,
		Logger:               testLogger{},
	}

	result1 := adapter.Run(context.Background(), req)
	result2 := adapter.Run(context.Background(), req)

	// Determinism check
	if result1.Status != result2.Status {
		t.Fatalf("non-deterministic status: %v vs %v", result1.Status, result2.Status)
	}

	if result1.DurationMs != result2.DurationMs {
		t.Fatalf("non-deterministic duration: %d vs %d", result1.DurationMs, result2.DurationMs)
	}

	if string(result1.Stdout) != string(result2.Stdout) {
		t.Fatal("stdout is not deterministic")
	}
}

func TestMockAdapter_RunResult_Valid(t *testing.T) {
	t.Parallel()

	adapter := Adapter{}

	req := sdk.RunRequest{
		RunID:                "test",
		WorkspaceRoot:        "/workspace",
		ArtifactsRoot:        "/workspace/.bb",
		AdapterArtifactsPath: "raw/mock",
		Environment:          "dev",
		ExecutionMode:        "all",
		Timeout:              time.Second,
		Logger:               testLogger{},
	}

	result := adapter.Run(context.Background(), req)

	if err := result.Validate(); err != nil {
		t.Fatalf("invalid run result: %v", err)
	}
}

func TestMockAdapter_Normalize_SchemaShape(t *testing.T) {
	t.Parallel()

	adapter := Adapter{}

	meta := adapter.Metadata()

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    meta.Name,
		AdapterType:    meta.Type,
		AdapterVersion: "0.0.1",
		Command:        "mock-tool",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusCompleted,
			DurationMs: 5,
		},
		StdoutPath:     "raw/stdout.log",
		StderrPath:     "raw/stderr.log",
		ToolOutputPath: "raw/tool-output.json",
	}

	input := sdk.NormalizationInput{
		RawExecution: raw,
		AdapterMeta:  meta,
	}

	output := adapter.Normalize(context.Background(), input)

	if len(output) == 0 {
		t.Fatal("expected normalized output")
	}

	// Ensure valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// Validate required top-level fields
	requiredFields := []string{
		"schema_version",
		"adapter",
		"execution",
		"summary",
		"evidence",
		"artifacts",
		"extensions",
	}

	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("missing field: %s", field)
		}
	}
}

func TestMockAdapter_Normalize_Deterministic(t *testing.T) {
	t.Parallel()

	adapter := Adapter{}
	meta := adapter.Metadata()

	raw := sdk.RawExecution{
		SchemaVersion:  "0.1",
		AdapterName:    meta.Name,
		AdapterType:    meta.Type,
		AdapterVersion: "0.0.1",
		RunResult: sdk.RunResult{
			Status:     sdk.ExecutionStatusCompleted,
			DurationMs: 5,
		},
	}

	input := sdk.NormalizationInput{
		RawExecution: raw,
		AdapterMeta:  meta,
	}

	out1 := adapter.Normalize(context.Background(), input)
	out2 := adapter.Normalize(context.Background(), input)

	if string(out1) != string(out2) {
		t.Fatal("normalization is not deterministic")
	}
}

func TestMockAdapter_NoSideEffects(t *testing.T) {
	t.Parallel()

	adapter := Adapter{}

	req := sdk.RunRequest{
		RunID:                "test",
		WorkspaceRoot:        "/workspace",
		ArtifactsRoot:        "/workspace/.bb",
		AdapterArtifactsPath: "raw/mock",
		Environment:          "dev",
		ExecutionMode:        "all",
		Timeout:              time.Second,
		Logger:               testLogger{},
	}

	// Run multiple times to ensure no mutation / hidden state
	for i := 0; i < 5; i++ {
		_ = adapter.Run(context.Background(), req)
	}
}

func TestMockAdapter_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ sdk.Adapter = Adapter{} // compile-time assertion
}
