package sdk

import (
	"encoding/json"
	"testing"
)

func validNormalizationInput(status ExecutionStatus) NormalizationInput {
	exitCode := 0

	return NormalizationInput{
		RawExecution: RawExecution{
			SchemaVersion:  RawExecutionSchemaVersion,
			AdapterName:    "jest",
			AdapterType:    AdapterTypeUnit,
			AdapterVersion: "29.7.0",
			Command:        "npx",
			Args:           []string{"jest", "--json"},
			ResolvedBinary: "/usr/bin/npx",
			RunResult: RunResult{
				Status:     status,
				ExitCode:   &exitCode,
				DurationMs: 2000,
			},
			StdoutPath:     "raw/jest/stdout.log",
			StderrPath:     "raw/jest/stderr.log",
			ToolOutputPath: "raw/jest/tool-output.json",
		},
		AdapterMeta: AdapterMetadata{
			Name:             "jest",
			Type:             AdapterTypeUnit,
			Description:      "JavaScript unit test adapter",
			Order:            10,
			SupportedTool:    "jest",
			VersionCommand:   []string{"npx", "jest", "--version"},
			DefaultTimeoutMs: 30000,
		},
	}
}

func TestNormalizationInput_Validate(t *testing.T) {
	t.Parallel()

	input := validNormalizationInput(ExecutionStatusCompleted)
	if err := input.Validate(); err != nil {
		t.Fatalf("expected valid normalization input, got error: %v", err)
	}
}

func TestApplyExecutionScenarioDefaults_TimedOut(t *testing.T) {
	t.Parallel()

	input := validNormalizationInput(ExecutionStatusTimedOut)
	out := NewBaseNormalizedResult(input)

	ApplyExecutionScenarioDefaults(&out, input)

	if out.Summary.Status != string(SummaryStatusUnknown) {
		t.Fatalf("expected unknown summary status, got %q", out.Summary.Status)
	}
	if out.Evidence.Complete {
		t.Fatal("expected evidence.complete=false for timeout")
	}
}

func TestApplyExecutionScenarioDefaults_Unavailable(t *testing.T) {
	t.Parallel()

	input := validNormalizationInput(ExecutionStatusUnavailable)
	out := NewBaseNormalizedResult(input)

	ApplyExecutionScenarioDefaults(&out, input)

	if out.Summary.Status != string(SummaryStatusUnknown) {
		t.Fatalf("expected unknown summary status, got %q", out.Summary.Status)
	}
	if out.Evidence.Complete {
		t.Fatal("expected evidence.complete=false for unavailable")
	}
}

func TestMarkNormalizationFailure(t *testing.T) {
	t.Parallel()

	input := validNormalizationInput(ExecutionStatusCompleted)
	out := NewBaseNormalizedResult(input)

	MarkNormalizationFailure(&out, "invalid tool output")

	if out.Summary.Status != string(SummaryStatusUnknown) {
		t.Fatalf("expected unknown summary status, got %q", out.Summary.Status)
	}
	if out.Evidence.Complete {
		t.Fatal("expected evidence.complete=false after normalization failure")
	}
	if len(out.Evidence.Issues) == 0 {
		t.Fatal("expected normalization failure issue to be added")
	}
}

func TestMarshalNormalizedResult_ValidJSON(t *testing.T) {
	t.Parallel()

	input := validNormalizationInput(ExecutionStatusCompleted)
	out := NewBaseNormalizedResult(input)

	if err := SetSummary(&out, SummaryStatusPass, 10, 10, 0, 0); err != nil {
		t.Fatalf("unexpected summary error: %v", err)
	}
	ComputeEvidenceCompleteness(input, &out, true)

	payload, err := MarshalNormalizedResult(out)
	if err != nil {
		t.Fatalf("expected marshal success, got error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}

	if decoded["schema_version"] != "0.1" {
		t.Fatalf("unexpected schema_version: %v", decoded["schema_version"])
	}
}
