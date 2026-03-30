package playwright

import (
	"os"
	"path/filepath"
	"testing"
)

// loadFixtureBytes loads a fixture file from disk for parser tests.
func loadFixtureBytes(t *testing.T, rel string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Clean(rel))
	if err != nil {
		t.Fatalf("read fixture %q: %v", rel, err)
	}
	return data
}