package sdk

import (
	"fmt"
	"strings"
)

// ValidateAdapter performs interface-level contract validation without executing the adapter.
func ValidateAdapter(adapter Adapter) error {
	if adapter == nil {
		return fmt.Errorf("adapter must not be nil")
	}

	meta := adapter.Metadata()
	if err := meta.Validate(); err != nil {
		return err
	}

	canonicalName := strings.TrimSpace(meta.Name)
	for _, alias := range meta.Aliases {
		if strings.EqualFold(strings.TrimSpace(alias), canonicalName) {
			return fmt.Errorf("adapter %q alias %q duplicates canonical name", meta.Name, alias)
		}
	}

	return nil
}
