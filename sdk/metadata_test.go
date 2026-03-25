package sdk

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAdapterMetadata_Validate_Succeeds(t *testing.T) {
	t.Parallel()

	meta := AdapterMetadata{
		Name:             "jest",
		Type:             AdapterTypeUnit,
		Description:      "JavaScript unit test adapter",
		Order:            10,
		SupportedTool:    "jest",
		VersionCommand:   []string{"npx", "jest", "--version"},
		DefaultTimeoutMs: 30000,
		SupportedFileGlobs: []string{
			"**/*.test.js",
			"**/*.spec.js",
		},
		Aliases:      []string{"js-test"},
		Capabilities: []string{"unit-test"},
	}

	if err := meta.Validate(); err != nil {
		t.Fatalf("expected valid metadata, got error: %v", err)
	}
}

func TestAdapterMetadata_Validate_FailsOnInvalidType(t *testing.T) {
	t.Parallel()

	meta := AdapterMetadata{
		Name:             "broken",
		Type:             AdapterType("invalid"),
		Description:      "Broken adapter",
		Order:            10,
		SupportedTool:    "broken",
		DefaultTimeoutMs: 1000,
	}

	if err := meta.Validate(); err == nil {
		t.Fatal("expected invalid adapter type to fail validation")
	}
}

func TestAdapterMetadata_Validate_FailsOnInvalidTimeout(t *testing.T) {
	t.Parallel()

	meta := AdapterMetadata{
		Name:             "jest",
		Type:             AdapterTypeUnit,
		Description:      "JavaScript unit test adapter",
		Order:            10,
		SupportedTool:    "jest",
		DefaultTimeoutMs: 0,
	}

	if err := meta.Validate(); err == nil {
		t.Fatal("expected invalid timeout to fail validation")
	}
}

func TestAdapterMetadata_Validate_FailsOnAliasMatchingCanonicalName(t *testing.T) {
	t.Parallel()

	meta := AdapterMetadata{
		Name:             "jest",
		Type:             AdapterTypeUnit,
		Description:      "JavaScript unit test adapter",
		Order:            10,
		SupportedTool:    "jest",
		DefaultTimeoutMs: 30000,
		Aliases:          []string{"JEST"},
	}

	if err := meta.Validate(); err == nil {
		t.Fatal("expected alias matching canonical name to fail validation")
	}
}

func TestValidateMetadataSet_FailsOnDuplicateCanonicalName(t *testing.T) {
	t.Parallel()

	all := []AdapterMetadata{
		{
			Name:             "jest",
			Type:             AdapterTypeUnit,
			Description:      "Jest adapter",
			Order:            10,
			SupportedTool:    "jest",
			DefaultTimeoutMs: 30000,
		},
		{
			Name:             "JEST",
			Type:             AdapterTypeUnit,
			Description:      "Another jest adapter",
			Order:            20,
			SupportedTool:    "jest",
			DefaultTimeoutMs: 30000,
		},
	}

	if err := ValidateMetadataSet(all); err == nil {
		t.Fatal("expected duplicate canonical name to fail validation")
	}
}

func TestValidateMetadataSet_FailsOnAliasCollision(t *testing.T) {
	t.Parallel()

	all := []AdapterMetadata{
		{
			Name:             "jest",
			Type:             AdapterTypeUnit,
			Description:      "Jest adapter",
			Order:            10,
			SupportedTool:    "jest",
			DefaultTimeoutMs: 30000,
			Aliases:          []string{"js-test"},
		},
		{
			Name:             "playwright",
			Type:             AdapterTypeUI,
			Description:      "Playwright adapter",
			Order:            20,
			SupportedTool:    "playwright",
			DefaultTimeoutMs: 60000,
			Aliases:          []string{"js-test"},
		},
	}

	if err := ValidateMetadataSet(all); err == nil {
		t.Fatal("expected alias collision to fail validation")
	}
}

func TestAdapterMetadata_NormalizedAliases(t *testing.T) {
	t.Parallel()

	meta := AdapterMetadata{
		Name:             "jest",
		Type:             AdapterTypeUnit,
		Description:      "Jest adapter",
		Order:            10,
		SupportedTool:    "jest",
		DefaultTimeoutMs: 30000,
		Aliases:          []string{" JS-Test ", "jest-js"},
	}

	got := meta.NormalizedAliases()
	want := []string{"js-test", "jest-js"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized aliases mismatch: got %v want %v", got, want)
	}
}

func TestAdapterMetadata_MarshalStableJSON_IsValidJSON(t *testing.T) {
	t.Parallel()

	meta := AdapterMetadata{
		Name:             "k6",
		Type:             AdapterTypePerformance,
		Description:      "Load testing adapter",
		Order:            30,
		SupportedTool:    "k6",
		VersionCommand:   []string{"k6", "version"},
		DefaultTimeoutMs: 120000,
	}

	payload, err := meta.MarshalStableJSON()
	if err != nil {
		t.Fatalf("marshal stable json failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}

	if decoded["name"] != "k6" {
		t.Fatalf("unexpected serialized name: %v", decoded["name"])
	}
}
