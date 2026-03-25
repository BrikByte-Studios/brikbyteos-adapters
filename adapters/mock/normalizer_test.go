package mock

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

func normalizationInput(status sdk.ExecutionStatus) sdk.NormalizationInput {
	exitCode := 0

	return sdk.NormalizationInput{
		RawExecution: sdk.RawExecution{
			SchemaVersion:  sdk.RawExecutionSchemaVersion,
			AdapterName:    "mock",
			AdapterType:    sdk.AdapterTypeOther,
			AdapterVersion: "0.0.1",
			Command:        "mock-tool",
			Args:           []string{"run"},
			ResolvedBinary: "/usr/bin/mock-tool",
			RunResult: sdk.RunResult{
				Status:     status,
				ExitCode:   &exitCode,
				DurationMs: 2000,
			},
			StdoutPath:     "raw/mock/stdout.log",
			StderrPath:     "raw/mock/stderr.log",
			ToolOutputPath: "raw/mock/tool-output.json",
		},
		AdapterMeta: sdk.AdapterMetadata{
			Name:             "jest",
			Type:             sdk.AdapterTypeUnit,
			Description:      "JavaScript unit test adapter",
			Order:            10,
			SupportedTool:    "jest",
			VersionCommand:   []string{"npx", "jest", "--version"},
			DefaultTimeoutMs: 30000,
		},
	}
}

func decode(t *testing.T, payload []byte) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}
	return out
}

func TestNormalize_Success(t *testing.T) {
	t.Parallel()

	payload := Adapter{}.Normalize(context.Background(), normalizationInput(sdk.ExecutionStatusCompleted))
	out := decode(t, payload)

	summary := out["summary"].(map[string]any)
	if summary["status"] != "pass" {
		t.Fatalf("expected pass, got %v", summary["status"])
	}
}

func TestNormalize_Failure(t *testing.T) {
	t.Parallel()

	payload := Adapter{}.Normalize(context.Background(), normalizationInput(sdk.ExecutionStatusFailed))
	out := decode(t, payload)

	summary := out["summary"].(map[string]any)
	if summary["status"] != "fail" {
		t.Fatalf("expected fail, got %v", summary["status"])
	}
}

func TestNormalize_TimedOut(t *testing.T) {
	t.Parallel()

	payload := Adapter{}.Normalize(context.Background(), normalizationInput(sdk.ExecutionStatusTimedOut))
	out := decode(t, payload)

	summary := out["summary"].(map[string]any)
	if summary["status"] != "unknown" {
		t.Fatalf("expected unknown, got %v", summary["status"])
	}
}

func TestNormalize_Unavailable(t *testing.T) {
	t.Parallel()

	payload := Adapter{}.Normalize(context.Background(), normalizationInput(sdk.ExecutionStatusUnavailable))
	out := decode(t, payload)

	summary := out["summary"].(map[string]any)
	if summary["status"] != "unknown" {
		t.Fatalf("expected unknown, got %v", summary["status"])
	}
}

func TestNormalize_Deterministic(t *testing.T) {
	t.Parallel()

	input := normalizationInput(sdk.ExecutionStatusCompleted)
	out1 := Adapter{}.Normalize(context.Background(), input)
	out2 := Adapter{}.Normalize(context.Background(), input)

	if string(out1) != string(out2) {
		t.Fatal("expected normalization output to be deterministic")
	}
}
