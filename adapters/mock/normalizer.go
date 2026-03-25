package mock

import (
	"context"
	"encoding/json"

	sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// Normalize transforms raw execution into canonical normalized JSON.
//
// This mock implementation is intentionally deterministic and side-effect free.
// It serves as the reference adapter implementation for SDK v0.
func (Adapter) Normalize(_ context.Context, input sdk.NormalizationInput) sdk.NormalizedResult {
	out := sdk.NewBaseNormalizedResult(input)
	sdk.ApplyExecutionScenarioDefaults(&out, input)

	switch input.RawExecution.RunResult.Status {
	case sdk.ExecutionStatusTimedOut, sdk.ExecutionStatusUnavailable:
		payload, _ := sdk.MarshalNormalizedResult(out)
		return payload
	}

	// Mock rule:
	//   - completed + tool output present => pass
	//   - failed => fail
	//   - anything else unclear => normalization failure fallback
	switch input.RawExecution.RunResult.Status {
	case sdk.ExecutionStatusCompleted:
		if err := sdk.SetSummary(&out, sdk.SummaryStatusPass, 1, 1, 0, 0); err != nil {
			sdk.MarkNormalizationFailure(&out, err.Error())
			payload, _ := sdk.MarshalNormalizedResult(out)
			return payload
		}
		sdk.ComputeEvidenceCompleteness(input, &out, true)
		out.Extensions.AdapterSpecific["mock"] = map[string]any{
			"mode": "success",
		}
	case sdk.ExecutionStatusFailed:
		if err := sdk.SetSummary(&out, sdk.SummaryStatusFail, 1, 0, 1, 0); err != nil {
			sdk.MarkNormalizationFailure(&out, err.Error())
			payload, _ := sdk.MarshalNormalizedResult(out)
			return payload
		}
		out.Evidence.Issues = append(out.Evidence.Issues, sdk.NormalizationIssue{
			Code:    sdk.IssueTestFailure,
			Message: "mock adapter reported a failed result",
		})
		sdk.ComputeEvidenceCompleteness(input, &out, true)
		out.Extensions.AdapterSpecific["mock"] = map[string]any{
			"mode": "failure",
		}
	default:
		sdk.MarkNormalizationFailure(&out, "unsupported execution status for mock normalization")
	}

	payload, err := sdk.MarshalNormalizedResult(out)
	if err != nil {
		// Deterministic fallback JSON; still schema-shaped enough for local debugging.
		fallback, _ := json.Marshal(map[string]any{
			"schema_version": "0.1",
			"adapter": map[string]any{
				"name":    input.AdapterMeta.Name,
				"type":    string(input.AdapterMeta.Type),
				"version": input.RawExecution.AdapterVersion,
			},
			"execution": map[string]any{
				"status":      string(input.RawExecution.RunResult.Status),
				"duration_ms": input.RawExecution.RunResult.DurationMs,
			},
			"summary": map[string]any{
				"status":  "unknown",
				"total":   0,
				"passed":  0,
				"failed":  0,
				"skipped": 0,
			},
			"evidence": map[string]any{
				"complete": false,
				"issues": []map[string]any{
					{"code": "NORMALIZATION_FAILED", "message": "marshal normalized result failed"},
				},
			},
			"artifacts": map[string]any{
				"raw_stdout_path":      input.RawExecution.StdoutPath,
				"raw_stderr_path":      input.RawExecution.StderrPath,
				"raw_tool_output_path": input.RawExecution.ToolOutputPath,
			},
			"extensions": map[string]any{
				"adapter_specific": map[string]any{},
			},
		})
		return fallback
	}

	return payload
}
