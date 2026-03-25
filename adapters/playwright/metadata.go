package playwright

import sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"

// Metadata returns the canonical static metadata for the Playwright adapter.
func Metadata() sdk.AdapterMetadata {
	return sdk.AdapterMetadata{
		Name:             "playwright",
		Type:             sdk.AdapterTypeUI,
		Description:      "Browser UI and end-to-end testing adapter",
		Order:            20,
		SupportedTool:    "playwright",
		VersionCommand:   []string{"npx", "playwright", "--version"},
		DefaultTimeoutMs: 60000,
		SupportedFileGlobs: []string{
			"**/*.spec.ts",
			"**/*.e2e.ts",
		},
		Aliases:      []string{"pw"},
		Capabilities: []string{"ui-test", "e2e"},
	}
}
