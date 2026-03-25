package k6

import sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"

// Metadata returns the canonical static metadata for the k6 adapter.
func Metadata() sdk.AdapterMetadata {
	return sdk.AdapterMetadata{
		Name:             "k6",
		Type:             sdk.AdapterTypePerformance,
		Description:      "Performance and load testing adapter",
		Order:            30,
		SupportedTool:    "k6",
		VersionCommand:   []string{"k6", "version"},
		DefaultTimeoutMs: 120000,
		SupportedFileGlobs: []string{
			"**/*.k6.js",
		},
		Aliases:      nil,
		Capabilities: []string{"performance"},
	}
}
