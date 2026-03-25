package sdk

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// CanonicalName returns the normalized canonical adapter name.
func (m AdapterMetadata) CanonicalName() string {
	return normalizeMetadataToken(m.Name)
}

// NormalizedAliases returns aliases normalized for comparison and resolution.
func (m AdapterMetadata) NormalizedAliases() []string {
	out := make([]string, 0, len(m.Aliases))
	for _, alias := range m.Aliases {
		n := normalizeMetadataToken(alias)
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

// Clone returns a defensive copy suitable for safe reuse by runtime consumers.
func (m AdapterMetadata) Clone() AdapterMetadata {
	return AdapterMetadata{
		Name:               m.Name,	
		Type:               m.Type,
		Description:        m.Description,
		Order:              m.Order,
		SupportedTool:      m.SupportedTool,
		VersionCommand:     slices.Clone(m.VersionCommand),
		DefaultTimeoutMs:   m.DefaultTimeoutMs,
		SupportedFileGlobs: slices.Clone(m.SupportedFileGlobs),
		Aliases:            slices.Clone(m.Aliases),
		Capabilities:       slices.Clone(m.Capabilities),
	}
}

// Validate enforces the canonical metadata contract.
//
// This validation is intentionally strict because metadata is used by:
//   - registry ordering
//   - alias resolution
//   - inspection utilities
//   - CLI output
func (m AdapterMetadata) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("metadata.name must not be empty")
	}
	if err := m.Type.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(m.Description) == "" {
		return fmt.Errorf("metadata.description for %q must not be empty", m.Name)
	}
	if m.Order <= 0 {
		return fmt.Errorf("metadata.order for %q must be > 0", m.Name)
	}
	if strings.TrimSpace(m.SupportedTool) == "" {
		return fmt.Errorf("metadata.supported_tool for %q must not be empty", m.Name)
	}
	if m.DefaultTimeoutMs <= 0 {
		return fmt.Errorf("metadata.default_timeout_ms for %q must be > 0", m.Name)
	}

	canonical := m.CanonicalName()
	if canonical == "" {
		return fmt.Errorf("metadata.name for %q normalizes to empty", m.Name)
	}

	seenAliases := make(map[string]struct{}, len(m.Aliases))
	for i, alias := range m.Aliases {
		normalized := normalizeMetadataToken(alias)
		if normalized == "" {
			return fmt.Errorf("metadata.aliases[%d] for %q must not be empty", i, m.Name)
		}
		if normalized == canonical {
			return fmt.Errorf("metadata.alias %q for %q duplicates canonical name", alias, m.Name)
		}
		if _, exists := seenAliases[normalized]; exists {
			return fmt.Errorf("metadata.alias %q for %q is duplicated", alias, m.Name)
		}
		seenAliases[normalized] = struct{}{}
	}

	for i, part := range m.VersionCommand {
		if strings.TrimSpace(part) == "" {
			return fmt.Errorf("metadata.version_command[%d] for %q must not be empty", i, m.Name)
		}
	}

	for i, glob := range m.SupportedFileGlobs {
		if strings.TrimSpace(glob) == "" {
			return fmt.Errorf("metadata.supported_file_globs[%d] for %q must not be empty", i, m.Name)
		}
	}

	for i, capability := range m.Capabilities {
		if strings.TrimSpace(capability) == "" {
			return fmt.Errorf("metadata.capabilities[%d] for %q must not be empty", i, m.Name)
		}
	}

	return nil
}

// MarshalStableJSON produces stable JSON for tests, examples, and CLI JSON output.
func (m AdapterMetadata) MarshalStableJSON() ([]byte, error) {
	// Struct field order already provides stable JSON key ordering in Go's encoder.
	return json.MarshalIndent(m, "", "  ")
}

// ValidateMetadataSet validates a complete set of adapter metadata values.
//
// It enforces:
//   - unique canonical names
//   - unique aliases across adapters
//   - no alias collisions with canonical names
func ValidateMetadataSet(all []AdapterMetadata) error {
	seenNames := make(map[string]string, len(all))
	seenAliases := make(map[string]string)

	for _, meta := range all {
		if err := meta.Validate(); err != nil {
			return err
		}

		canonical := meta.CanonicalName()
		if existing, exists := seenNames[canonical]; exists {
			return fmt.Errorf("duplicate adapter canonical name %q between %q and %q", canonical, existing, meta.Name)
		}
		seenNames[canonical] = meta.Name

		for _, alias := range meta.NormalizedAliases() {
			if owner, exists := seenNames[alias]; exists {
				return fmt.Errorf("alias %q for %q collides with canonical adapter name %q", alias, meta.Name, owner)
			}
			if owner, exists := seenAliases[alias]; exists {
				return fmt.Errorf("alias %q for %q collides with alias owned by %q", alias, meta.Name, owner)
			}
			seenAliases[alias] = meta.Name
		}
	}

	return nil
}

func normalizeMetadataToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}