package mock

import (
	"context"

	"github.com/BrikByte-Studios/brikbyteos-adapters/sdk"
)

// Adapter is a lightweight mock adapter used for runtime integration tests.
// It proves runtime can orchestrate adapters without special-casing.
type Adapter struct{}

// Metadata returns the canonical static description of the mock adapter.
func (Adapter) Metadata() sdk.AdapterMetadata {
	return sdk.AdapterMetadata{
		Name:             "mock",
		Type:             sdk.AdapterTypeOther,
		Description:      "Mock adapter for runtime integration tests",
		Order:            999,
		SupportedTool:    "mock-tool",
		VersionCommand:   []string{"mock-tool", "--version"},
		DefaultTimeoutMs: 1000,
		Aliases:          []string{"fake"},
		Capabilities:     []string{"test-double"},
	}
}

// CheckAvailability reports the mock adapter as always locally available.
func (Adapter) CheckAvailability(context.Context) sdk.Availability {
	return sdk.Availability{
		Available:      true,
		ResolvedBinary: "/usr/bin/mock-tool",
	}
}

// Version returns a deterministic mock version.
func (Adapter) Version(context.Context) (string, error) {
	return "0.0.1", nil
}

// Run returns a structured, successful execution result.
func (Adapter) Run(context.Context, sdk.RunRequest) sdk.RunResult {
	exitCode := 0
	return sdk.RunResult{
		Status:     sdk.ExecutionStatusCompleted,
		ExitCode:   &exitCode,
		DurationMs: 5,
		Stdout:     []byte(`{"ok":true}`),
		Stderr:     []byte{},
		ToolOutput: []byte(`{"checks":1,"passed":1}`),
	}
}

// New returns the canonical mock adapter as an sdk.Adapter.
func New() sdk.Adapter {
	return Adapter{}
}
