package playwright

import sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"

// AdapterName is the canonical stable identifier for the Playwright adapter.
const AdapterName = "playwright"

// Metadata returns the canonical static metadata for the Playwright adapter.
func Metadata() sdk.AdapterMetadata {
	return sdk.AdapterMetadata{
		Name:             AdapterName,
		Type:             sdk.AdapterTypeUI,
		Description:      "Browser/UI test adapter powered by Playwright",
		Order:            20,
		SupportedTool:    "playwright",
		VersionCommand:   []string{"npx", "playwright", "--version"},
		DefaultTimeoutMs: 60000,
		SupportedFileGlobs: []string{
			"**/*.spec.ts",
			"**/*.spec.js",
			"**/*.e2e.ts",
			"**/*.e2e.js",
			"**/*.e2e.ts",
		},
		Aliases:      []string{"pw", "browser-test"},
		Capabilities: []string{"ui-test", "browser-test"},
	}
}
