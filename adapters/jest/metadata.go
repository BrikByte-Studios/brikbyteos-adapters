package jest

import sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"

// AdapterName is the canonical stable identifier for the Jest adapter.
const AdapterName = "jest"

// Metadata returns the canonical static metadata for the Jest adapter.
func Metadata() sdk.AdapterMetadata {
	return sdk.AdapterMetadata{
		Name:             "jest",
		Type:             sdk.AdapterTypeUnit,
		Description:      "JavaScript and TypeScript unit test adapter",
		Order:            10,
		SupportedTool:    "jest",
		VersionCommand:   []string{"npx", "jest", "--version"},
		DefaultTimeoutMs: 30000,
		SupportedFileGlobs: []string{
			"**/*.test.js",
			"**/*.spec.js",
			"**/*.test.ts",
			"**/*.spec.ts",
		},
		Aliases:      []string{"js-test"},
		Capabilities: []string{"unit-test"},
	}
}
