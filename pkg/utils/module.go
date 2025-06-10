package utils

import (
	"os"
	"strings"
)

// GetProjectModule extracts the module name from go.mod or infers from file path
func GetProjectModule(filePath string) string {
	// Convert to absolute path if relative
	absPath := filePath
	if !strings.HasPrefix(filePath, "/") {
		if wd, err := os.Getwd(); err == nil {
			absPath = wd + "/" + filePath
		}
	}

	// Try to find go.mod file
	dir := absPath
	iterations := 0
	maxIterations := 20 // Prevent infinite loop

	for iterations < maxIterations {
		iterations++

		// Get parent directory
		lastSlash := strings.LastIndex(dir, "/")
		if lastSlash <= 0 {
			break
		}
		dir = dir[:lastSlash]

		goModPath := dir + "/go.mod"

		if content, err := os.ReadFile(goModPath); err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "module ") {
					module := strings.TrimSpace(strings.TrimPrefix(line, "module"))
					return module
				}
			}
		}
	}

	// Fallback: try to infer from file path
	if strings.Contains(filePath, "/src/") {
		parts := strings.Split(filePath, "/src/")
		if len(parts) > 1 {
			pathParts := strings.Split(parts[1], "/")
			if len(pathParts) >= 3 {
				module := strings.Join(pathParts[:3], "/")
				return module
			}
		}
	}
	return ""
}
