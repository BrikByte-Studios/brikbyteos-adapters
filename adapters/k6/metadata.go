package k6

import sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"

// AdapterName is the canonical stable identifier for the k6 adapter.
const AdapterName = "k6"

// Metadata returns the canonical static metadata for the built-in k6 adapter.
//
// This metadata is intentionally static and safe for discovery, ordering,
// UI reporting, and adapter registry integration.
func Metadata() sdk.AdapterMetadata {
	return sdk.AdapterMetadata{
		Name:             AdapterName,
		Type:             sdk.AdapterTypePerformance,
		Description:      "Performance and load testing adapter powered by k6",
		Order:            30,
		SupportedTool:    "k6",
		VersionCommand:   []string{"k6", "version"},
		DefaultTimeoutMs: 120000,
		SupportedFileGlobs: []string{
			"**/*.k6.js",
			"**/*.k6.ts",
			"**/*.perf.js",
			"**/*.perf.ts",
			"**/*.load.js",
			"**/*.load.ts",
		},
		Aliases:      []string{"performance", "load-test"},
		Capabilities: []string{"performance-test", "load-test"},
	}
}