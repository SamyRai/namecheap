package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTestConfigFile creates a test config file with the given content
func CreateTestConfigFile(t *testing.T, content []byte) string {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	if err := os.WriteFile(configPath, content, 0600); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	return configPath
}

// CleanupTestFile removes a test file (useful for explicit cleanup)
func CleanupTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Logf("Failed to cleanup test file %s: %v", path, err)
	}
}

// CleanupTestDir removes a test directory (useful for explicit cleanup)
func CleanupTestDir(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Logf("Failed to cleanup test directory %s: %v", path, err)
	}
}
