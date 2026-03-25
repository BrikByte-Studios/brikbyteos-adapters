package trivy

import sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"

// Metadata returns the canonical static metadata for the Trivy adapter.
func Metadata() sdk.AdapterMetadata {
	return sdk.AdapterMetadata{
		Name:             "trivy",
		Type:             sdk.AdapterTypeSecurity,
		Description:      "Security vulnerability and configuration scanning adapter",
		Order:            40,
		SupportedTool:    "trivy",
		VersionCommand:   []string{"trivy", "--version"},
		DefaultTimeoutMs: 120000,
		SupportedFileGlobs: []string{
			"Dockerfile",
			"**/*.lock",
		},
		Aliases:      nil,
		Capabilities: []string{"security-scan"},
	}
}
