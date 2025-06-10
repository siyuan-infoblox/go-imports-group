package std

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsStandardPackage(t *testing.T) {
	req := require.New(t)
	tests := []struct {
		name       string
		importPath string
		expected   bool
	}{
		{"standard package - fmt", "fmt", true},
		{"standard package - net/http", "net/http", true},
		{"standard package - context", "context", true},
		{"standard package - crypto/tls", "crypto/tls", true},
		{"non-standard package - github.com/something", "github.com/something", false},
		{"non-standard package - golang.org/x/tools", "golang.org/x/tools", false},
		{"empty string", "", false},
		{"internal Go package", "internal/something", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStandardPackage(tt.importPath)
			req.Equal(tt.expected, result, "IsStandardPackage(%q)", tt.importPath)
		})
	}
}

func TestStandardPackagesMapNotEmpty(t *testing.T) {
	req := require.New(t)
	req.NotEmpty(StandardPackages, "StandardPackages map should not be empty")

	// Check that some well-known packages are present
	expectedPackages := []string{"fmt", "os", "io", "net/http", "context", "strings"}
	for _, pkg := range expectedPackages {
		req.True(StandardPackages[pkg], "Expected standard package %q not found in StandardPackages map", pkg)
	}
}
