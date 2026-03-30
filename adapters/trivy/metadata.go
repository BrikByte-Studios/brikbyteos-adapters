package trivy

import sdk "github.com/BrikByte-Studios/brikbyteos-adapters/sdk"

// AdapterName is the canonical stable identifier for the Trivy adapter.
const AdapterName = "trivy"

// Metadata returns the canonical static metadata for the built-in Trivy adapter.
//
// This metadata is intentionally static and safe for discovery, ordering,
// registry integration, and UI/reporting.
func Metadata() sdk.AdapterMetadata {
	return sdk.AdapterMetadata{
		Name:             AdapterName,
		Type:             sdk.AdapterTypeSecurity,
		Description:      "Security scanning adapter powered by Trivy",
		Order:            40,
		SupportedTool:    "trivy",
		VersionCommand:   []string{"trivy", "--version"},
		DefaultTimeoutMs: 120000,
		SupportedFileGlobs: []string{
			"Dockerfile",
			"**/Dockerfile",
			"package-lock.json",
			"go.mod",
			"pom.xml",
			"requirements.txt",
		},
		Aliases:      []string{"security", "vuln-scan"},
		Capabilities: []string{"security-scan", "vulnerability-scan"},
	}
}
