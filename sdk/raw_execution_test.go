package sdk

import (
	"encoding/json"
	"testing"
	"time"
)

func validRawExecutionResult() RawExecutionResult {
	started := time.Date(2026, 3, 25, 18, 0, 0, 0, time.UTC)
	finished := started.Add(2 * time.Second)

	exitCode := 0

	return RawExecutionResult{
		SchemaVersion:   RawExecutionSchemaVersion,
		AdapterName:     "jest",
		AdapterType:     AdapterTypeUnit,
		AdapterVersion:  "29.7.0",
		Command:         "npx",
		Args:            []string{"jest", "--json"},
		ResolvedBinary:  "/usr/bin/npx",
		ExecutionStatus: ExecutionStatusCompleted,
		ExitCode:        &exitCode,
		StartedAt:       started,
		FinishedAt:      finished,
		DurationMs:      2000,
		StdoutPath:      "raw/jest/stdout.log",
		StderrPath:      "raw/jest/stderr.log",
		ToolOutputPath:  "raw/jest/tool-output.json",
	}
}

func TestRawExecutionResult_Validate_Valid(t *testing.T) {
	t.Parallel()

	raw := validRawExecutionResult()
	if err := raw.Validate(); err != nil {
		t.Fatalf("expected valid raw execution result, got error: %v", err)
	}
}

func TestRawExecutionResult_Validate_UnavailableAllowsEmptyCommand(t *testing.T) {
	t.Parallel()

	raw := validRawExecutionResult()
	raw.ExecutionStatus = ExecutionStatusUnavailable
	raw.Command = ""

	if err := raw.Validate(); err != nil {
		t.Fatalf("expected unavailable raw execution result to validate, got error: %v", err)
	}
}

func TestRawExecutionResult_Validate_RejectsAbsoluteArtifactPath(t *testing.T) {
	t.Parallel()

	raw := validRawExecutionResult()
	raw.StdoutPath = "/tmp/stdout.log"

	if err := raw.Validate(); err == nil {
		t.Fatal("expected absolute artifact path to fail validation")
	}
}

func TestRawExecutionResult_Validate_RejectsEscapingArtifactPath(t *testing.T) {
	t.Parallel()

	raw := validRawExecutionResult()
	raw.StdoutPath = "../escape.log"

	if err := raw.Validate(); err == nil {
		t.Fatal("expected escaping artifact path to fail validation")
	}
}

func TestRawExecutionResult_Validate_RejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	raw := validRawExecutionResult()
	raw.ExecutionStatus = ExecutionStatus("weird")

	if err := raw.Validate(); err == nil {
		t.Fatal("expected invalid execution status to fail validation")
	}
}

func TestRawExecutionResult_MarshalJSON_UsesUTCTimestamps(t *testing.T) {
	t.Parallel()

	raw := validRawExecutionResult()

	payload, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("expected JSON marshal success, got error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v", err)
	}

	if decoded["started_at"] != "2026-03-25T18:00:00Z" {
		t.Fatalf("unexpected started_at: %v", decoded["started_at"])
	}
	if decoded["finished_at"] != "2026-03-25T18:00:02Z" {
		t.Fatalf("unexpected finished_at: %v", decoded["finished_at"])
	}
}
