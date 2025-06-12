package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUtils_GetProjectModule(t *testing.T) {
	req := require.New(t)
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grouper_test")
	req.NoError(err)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a go.mod file
	goModContent := `module github.com/test/project

go 1.21
`
	goModPath := filepath.Join(tempDir, "go.mod")
	req.NoError(os.WriteFile(goModPath, []byte(goModContent), 0644))

	// Create a subdirectory with a test file
	subDir := filepath.Join(tempDir, "internal", "pkg")
	req.NoError(os.MkdirAll(subDir, 0755))

	testFile := filepath.Join(subDir, "test.go")
	req.NoError(os.WriteFile(testFile, []byte("package pkg"), 0644))

	// Test: finds go.mod in parent directory
	result := GetProjectModule(testFile)
	req.Equal("github.com/test/project", result, "getProjectModule(%q)", testFile)
}

func TestUtils_GetProjectModule_fallbacks(t *testing.T) {
	req := require.New(t)
	// Test with non-existent file
	result := GetProjectModule("/non/existent/path/file.go")
	req.Empty(result, "Expected empty string for non-existent path")

	// Test with src path pattern
	srcPath := "/some/path/src/github.com/user/project/internal/file.go"
	result = GetProjectModule(srcPath)
	req.Equal("github.com/user/project", result, "Expected correct project module from src path")
}
