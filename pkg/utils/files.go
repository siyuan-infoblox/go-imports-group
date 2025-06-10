package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// IsGoFile checks if a file is a Go source file (includes test files)
func IsGoFile(filename string) bool {
	return strings.HasSuffix(filename, ".go")
}

// FindGoFiles recursively finds all Go source files in a directory
func FindGoFiles(root string) ([]string, error) {
	var goFiles []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor directories and hidden directories (but not the root directory)
		if info.IsDir() && path != root {
			name := filepath.Base(path)
			if name == "vendor" || name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if IsGoFile(filepath.Base(path)) {
			goFiles = append(goFiles, path)
		}

		return nil
	})

	return goFiles, err
}

// IsDirectory checks if the given path is a directory
func IsDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
