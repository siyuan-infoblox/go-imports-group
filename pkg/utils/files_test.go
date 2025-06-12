package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsGoFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "regular go file",
			filename: "main.go",
			expected: true,
		},
		{
			name:     "go file with path",
			filename: "cmd/root.go",
			expected: true,
		},
		{
			name:     "test file should be included",
			filename: "main_test.go",
			expected: true,
		},
		{
			name:     "test file with path should be included",
			filename: "pkg/utils/files_test.go",
			expected: true,
		},
		{
			name:     "non-go file",
			filename: "README.md",
			expected: false,
		},
		{
			name:     "file with .go in middle",
			filename: "file.go.txt",
			expected: false,
		},
		{
			name:     "empty string",
			filename: "",
			expected: false,
		},
		{
			name:     "just .go",
			filename: ".go",
			expected: true,
		},
		{
			name:     "hidden go file",
			filename: ".hidden.go",
			expected: true,
		},
		{
			name:     "benchmark test file",
			filename: "benchmark_test.go",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			result := IsGoFile(tt.filename)
			req.Equal(tt.expected, result, "IsGoFile(%q) = %v, want %v", tt.filename, result, tt.expected)
		})
	}
}

func TestIsDirectory(t *testing.T) {
	req := require.New(t)
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a temporary file
	tempFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(tempFile, []byte("test"), 0644)
	req.NoError(err, "Failed to create temp file: %v", err)

	tests := []struct {
		name      string
		path      string
		expected  bool
		expectErr bool
	}{
		{
			name:      "existing directory",
			path:      tempDir,
			expected:  true,
			expectErr: false,
		},
		{
			name:      "existing file",
			path:      tempFile,
			expected:  false,
			expectErr: false,
		},
		{
			name:      "non-existent path",
			path:      "/non/existent/path",
			expected:  false,
			expectErr: true,
		},
		{
			name:      "current directory",
			path:      ".",
			expected:  true,
			expectErr: false,
		},
		{
			name:      "parent directory",
			path:      "..",
			expected:  true,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			result, err := IsDirectory(tt.path)

			if tt.expectErr {
				req.Error(err, "IsDirectory(%q) expected error, got nil", tt.path)
			} else {
				req.NoError(err, "IsDirectory(%q) unexpected error: %v", tt.path, err)
				req.Equal(tt.expected, result, "IsDirectory(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFindGoFiles(t *testing.T) {
	req := require.New(t)
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()

	// Create test directory structure
	dirs := []string{
		"pkg/cmd",
		"pkg/utils",
		"examples",
		"vendor/github.com/test",
		".git",
		".hidden",
	}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		req.NoError(err, "Failed to create directory %s: %v", dir, err)
	}

	// Create test files
	files := map[string]string{
		"main.go":                          "package main",
		"pkg/cmd/root.go":                  "package cmd",
		"pkg/utils/files.go":               "package utils",
		"pkg/utils/files_test.go":          "package utils", // Should be included
		"examples/example.go":              "package main",
		"examples/example_test.go":         "package main", // Should be included
		"vendor/github.com/test/vendor.go": "package test", // Should be excluded (vendor dir)
		".git/config":                      "config",       // Should be excluded (hidden dir)
		".hidden/hidden.go":                "package main", // Should be excluded (hidden dir)
		"README.md":                        "# README",     // Should be excluded (not .go)
		"script.sh":                        "#!/bin/bash",  // Should be excluded (not .go)
	}

	for filePath, content := range files {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		req.NoError(err, "Failed to create file %s: %v", filePath, err)
	}

	tests := []struct {
		name          string
		root          string
		expectedLen   int
		expectedFiles []string
		expectErr     bool
	}{
		{
			name:        "find go files in temp directory",
			root:        tempDir,
			expectedLen: 6, // main.go, pkg/cmd/root.go, pkg/utils/files.go, pkg/utils/files_test.go, examples/example.go, examples/example_test.go
			expectedFiles: []string{
				filepath.Join(tempDir, "main.go"),
				filepath.Join(tempDir, "pkg/cmd/root.go"),
				filepath.Join(tempDir, "pkg/utils/files.go"),
				filepath.Join(tempDir, "pkg/utils/files_test.go"),
				filepath.Join(tempDir, "examples/example.go"),
				filepath.Join(tempDir, "examples/example_test.go"),
			},
			expectErr: false,
		},
		{
			name:        "non-existent directory",
			root:        "/non/existent/path",
			expectedLen: 0,
			expectErr:   true,
		},
		{
			name:        "empty directory",
			root:        filepath.Join(tempDir, "empty"),
			expectedLen: 0,
			expectErr:   false,
		},
	}

	// Create empty directory for test
	err := os.Mkdir(filepath.Join(tempDir, "empty"), 0755)
	req.NoError(err, "Failed to create empty directory: %v", err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			result, err := FindGoFiles(tt.root)

			if tt.expectErr {
				req.Error(err, "FindGoFiles(%q) expected error, got nil", tt.root)
				return
			}

			req.NoError(err, "FindGoFiles(%q) unexpected error: %v", tt.root, err)
			req.Len(result, tt.expectedLen, "FindGoFiles(%q) returned %d files, expected %d. Found files: %v", tt.root, len(result), tt.expectedLen, result)

			// For the main test case, verify specific files are found
			if tt.name == "find go files in temp directory" {
				foundFiles := make(map[string]bool)
				for _, file := range result {
					foundFiles[file] = true
				}

				for _, expected := range tt.expectedFiles {
					req.True(foundFiles[expected], "Expected file %q not found in results", expected)
				}
			}
		})
	}
}
