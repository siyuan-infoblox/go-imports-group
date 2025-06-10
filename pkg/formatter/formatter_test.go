package formatter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatter_isStdImport(t *testing.T) {
	req := require.New(t)
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           []string{},
		CurrentProject: "",
		InPlace:        false,
	})

	tests := []struct {
		name       string
		importPath string
		want       bool
	}{
		// Standard library exact matches
		{"fmt", "fmt", true},
		{"os", "os", true},
		{"strings", "strings", true},
		{"context", "context", true},
		{"log", "log", true},

		// Standard library with paths
		{"net/http", "net/http", true},
		{"encoding/json", "encoding/json", true},
		{"go/ast", "go/ast", true},
		{"crypto/rand", "crypto/rand", true},

		// Third-party packages
		{"github.com/pkg/errors", "github.com/pkg/errors", false},
		{"gitlab.com/myorg/myproject", "gitlab.com/myorg/myproject", false},
		{"golang.org/x/tools", "golang.org/x/tools", false},
		{"google.golang.org/grpc", "google.golang.org/grpc", false},
		{"gopkg.in/yaml.v2", "gopkg.in/yaml.v2", false},

		// Edge cases
		{"example.com/package", "example.com/package", false},
		{"internal", "internal", false}, // "internal" is not an actual standard library package
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.isStdImport(tt.importPath)
			req.Equal(tt.want, result, "isStdImport(%q)", tt.importPath)
		})
	}
}

func TestFormatter_classifyImport(t *testing.T) {
	req := require.New(t)
	orgs := []string{"github.com/myorg", "gitlab.com/anotherorg"}
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           orgs,
		CurrentProject: "",
		InPlace:        false,
	})

	tests := []struct {
		name          string
		importPath    string
		projectModule string
		want          ImportGroup
	}{
		{"standard library", "fmt", "github.com/myorg/myproject", StdGroup},
		{"standard library with path", "net/http", "github.com/myorg/myproject", StdGroup},
		{"project import", "github.com/myorg/myproject/internal", "github.com/myorg/myproject", ProjectGroup},
		{"project import gig", "github.com/username/go-imports-group/pkg/formatter", "github.com/username/go-imports-group", ProjectGroup},
		{"org import", "github.com/myorg/otherproject", "github.com/myorg/myproject", ImportGroup(OrgGroupBase + 0)},
		{"another org import", "gitlab.com/anotherorg/project", "github.com/myorg/myproject", ImportGroup(OrgGroupBase + 1)},
		{"third party", "github.com/external/lib", "github.com/myorg/myproject", ThirdPartyGroup},
		{"third party golang.org", "golang.org/x/tools", "github.com/myorg/myproject", ThirdPartyGroup},
		{"third party cobra", "github.com/spf13/cobra", "github.com/myorg/myproject", ThirdPartyGroup},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.classifyImport(tt.importPath, tt.projectModule)
			req.Equal(tt.want, result, "classifyImport(%q, %q)", tt.importPath, tt.projectModule)
		})
	}
}

func TestFormatter_getOrgInfo(t *testing.T) {
	req := require.New(t)
	orgs := []string{"github.com/myorg", "gitlab.com/anotherorg"}
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           orgs,
		CurrentProject: "",
		InPlace:        false,
	})

	tests := []struct {
		name            string
		importPath      string
		wantIndex       int
		wantProjectName string
	}{
		{"first org", "github.com/myorg/project1", 0, "project1"},
		{"first org with subpath", "github.com/myorg/project1/internal/pkg", 0, "project1"},
		{"second org", "gitlab.com/anotherorg/project2", 1, "project2"},
		{"second org with subpath", "gitlab.com/anotherorg/project2/cmd/tool", 1, "project2"},
		{"not an org", "github.com/external/lib", -1, ""},
		{"org with no project", "github.com/myorg", 0, ""},
		{"org with trailing slash", "github.com/myorg/", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIndex, gotProjectName := g.getOrgInfo(tt.importPath)
			req.Equal(tt.wantIndex, gotIndex, "getOrgInfo(%q) index", tt.importPath)
			req.Equal(tt.wantProjectName, gotProjectName, "getOrgInfo(%q) projectName", tt.importPath)
		})
	}
}

func TestFormatter_sortImportsInGroup(t *testing.T) {
	req := require.New(t)
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           []string{"github.com/myorg"},
		CurrentProject: "",
		InPlace:        false,
	})

	t.Run("sort standard imports alphabetically", func(t *testing.T) {
		imports := []Import{
			{Path: "strings"},
			{Path: "fmt"},
			{Path: "os"},
		}
		g.sortImportsInGroup(imports, StdGroup)

		expected := []string{"fmt", "os", "strings"}
		for i, imp := range imports {
			req.Equal(expected[i], imp.Path, "sortImportsInGroup() index %d", i)
		}
	})

	t.Run("sort org imports by org index, project, then path", func(t *testing.T) {
		imports := []Import{
			{Path: "github.com/myorg/project2/pkg", OrgIndex: 0, ProjectName: "project2"},
			{Path: "github.com/myorg/project1/cmd", OrgIndex: 0, ProjectName: "project1"},
			{Path: "github.com/myorg/project1/api", OrgIndex: 0, ProjectName: "project1"},
		}
		g.sortImportsInGroup(imports, ImportGroup(OrgGroupBase+0))

		expected := []string{
			"github.com/myorg/project1/api",
			"github.com/myorg/project1/cmd",
			"github.com/myorg/project2/pkg",
		}
		for i, imp := range imports {
			req.Equal(expected[i], imp.Path, "sortImportsInGroup() index %d", i)
		}
	})
}

func TestFormatter_ProcessFile(t *testing.T) {
	req := require.New(t)

	// Create a temporary Go file for testing
	tempDir, err := os.MkdirTemp("", "formatter_test")
	req.NoError(err)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create go.mod
	goModContent := `module github.com/test/project

go 1.21
`
	goModPath := filepath.Join(tempDir, "go.mod")
	req.NoError(os.WriteFile(goModPath, []byte(goModContent), 0644))

	// Create test Go file with mixed imports
	testGoContent := `package main

import (
	"github.com/external/lib"
	"fmt"
	"github.com/test/project/internal"
	"strings"
	"github.com/myorg/project1"
)

func main() {
	fmt.Println("Hello")
}
`
	testFile := filepath.Join(tempDir, "main.go")
	req.NoError(os.WriteFile(testFile, []byte(testGoContent), 0644))

	orgs := []string{"github.com/myorg"}
	g := New(FormatterConfig{
		FilePath:       testFile,
		Orgs:           orgs,
		CurrentProject: "",
		InPlace:        true,
	})

	t.Run("process file in place", func(t *testing.T) {
		err := g.ProcessFile()
		req.NoError(err)

		// Read the processed file
		processed, err := os.ReadFile(testFile)
		req.NoError(err)

		processedStr := string(processed)

		// Check that the file contains expected import groupings
		req.Contains(processedStr, `"fmt"`)
		req.Contains(processedStr, `"strings"`)
		req.Contains(processedStr, `"github.com/external/lib"`)
	})

	t.Run("process file without imports", func(t *testing.T) {
		noImportsContent := `package main

func main() {
	println("Hello")
}
`
		noImportsFile := filepath.Join(tempDir, "noimports.go")
		req.NoError(os.WriteFile(noImportsFile, []byte(noImportsContent), 0644))

		g2 := New(FormatterConfig{
			FilePath:       noImportsFile,
			Orgs:           []string{},
			CurrentProject: "",
			InPlace:        true,
		})
		err := g2.ProcessFile()
		req.NoError(err)
	})

	t.Run("process non-existent file", func(t *testing.T) {
		g3 := New(FormatterConfig{
			FilePath:       "/non/existent/file.go",
			Orgs:           []string{},
			CurrentProject: "",
			InPlace:        true,
		})
		err := g3.ProcessFile()
		req.Error(err)
	})
}

func TestFormatter_extractImports(t *testing.T) {
	req := require.New(t)
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           []string{},
		CurrentProject: "",
		InPlace:        false,
	})

	// Create a test Go file content
	testContent := `package test

import (
	"fmt" // test comment
	alias "github.com/pkg/errors"
	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
)
`

	astFile, err := parseString(testContent)
	req.NoError(err)

	imports := g.extractImports(astFile)

	req.Len(imports, 4)

	// Check specific imports
	expectedPaths := []string{"fmt", "github.com/pkg/errors", "github.com/lib/pq", "github.com/onsi/ginkgo/v2"}
	for i, imp := range imports {
		req.Equal(expectedPaths[i], imp.Path)
	}
	// Check comment
	req.Equal("test comment\n", imports[0].Comment)

	// Check that the alias is captured
	req.Equal("alias", imports[1].Name)

	// Check blank import
	req.Equal("_", imports[2].Name)

	// Check dot import
	req.Equal(".", imports[3].Name)
}

// Helper function to parse string content
func parseString(content string) (*ast.File, error) {
	return parser.ParseFile(token.NewFileSet(), "test.go", content, parser.ParseComments)
}

func TestFormatter_groupImports(t *testing.T) {
	req := require.New(t)
	orgs := []string{"github.com/myorg", "github.com/acme-corp"}
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           orgs,
		CurrentProject: "github.com/username/go-imports-group",
		InPlace:        false,
	})

	imports := []Import{
		{Path: "github.com/username/go-imports-group/abc", Group: ProjectGroup},
		{Path: "github.com/spf13/cobra", Group: ThirdPartyGroup},
		{Path: "context", Group: StdGroup},
		{Path: "encoding/json", Group: StdGroup},
		{Path: "github.com/spf13/pflag", Group: ThirdPartyGroup},
		{Path: "github.com/username/go-imports-group/ghi", Group: ProjectGroup},
		{Path: "fmt", Group: StdGroup},
		{Path: "github.com/acme-corp/platform.common/db/connector", Group: ImportGroup(OrgGroupBase + 1), OrgIndex: 1, ProjectName: "platform.common"},
		{Path: "github.com/acme-corp/platform.core.cache/pkg/config", Group: ImportGroup(OrgGroupBase + 1), OrgIndex: 1, ProjectName: "platform.core.cache"},
		{Path: "github.com/username/go-imports-group/def", Group: ProjectGroup},
		{Path: "os", Group: StdGroup},
		{Path: "github.com/myorg/toolkit/gateway", Group: ImportGroup(OrgGroupBase + 0), OrgIndex: 0, ProjectName: "toolkit"},
		{Path: "path/filepath", Group: StdGroup},
		{Path: "github.com/grpc-ecosystem/go-grpc-middleware", Group: ThirdPartyGroup},
		{Path: "strings", Group: StdGroup},
		{Path: "github.com/myorg/toolkit/requestid", Group: ImportGroup(OrgGroupBase + 0), OrgIndex: 0, ProjectName: "toolkit"},
		{Path: "time", Group: StdGroup},
	}

	grouped := g.groupImports(imports, "")
	req.Len(grouped, 5, "Expected 5 import groups") // Std, ThirdParty, Org1, Org2, Project
	req.Len(grouped[StdGroup], 7, "Expected 7 standard library imports")
	req.Len(grouped[ThirdPartyGroup], 3, "Expected 3 third-party imports")
	req.Len(grouped[ImportGroup(OrgGroupBase+0)], 2, "Expected 2 imports in first org group")
	req.Len(grouped[ImportGroup(OrgGroupBase+1)], 2, "Expected 2 imports in second org group")
	req.Len(grouped[ProjectGroup], 3, "Expected 3 project imports")
}

func TestFormatter_replaceImports(t *testing.T) {
	req := require.New(t)

	orgs := []string{"github.com/myorg", "gitlab.com/anotherorg"}
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           orgs,
		CurrentProject: "github.com/test/project",
		InPlace:        false,
	})

	// Create a test AST file with minimal imports (the function will replace these)
	testContent := `package test

import (
	"fmt"
	"github.com/external/lib"
)

func main() {}
`

	astFile, err := parseString(testContent)
	req.NoError(err, "Failed to parse test content")

	// Create comprehensive grouped imports covering all scenarios
	groupedImports := map[ImportGroup][]Import{
		// Standard library imports
		StdGroup: {
			{Path: "context"},
			{Path: "encoding/json"},
			{Path: "fmt"},
			{Path: "net/http"},
			{Path: "os"},
			{Path: "strings"},
			{Path: "time"},
		},
		// Third-party imports with various aliases
		ThirdPartyGroup: {
			{Path: "github.com/external/lib"},
			{Name: "errors", Path: "github.com/pkg/errors"},
			{Name: "_", Path: "github.com/lib/pq"},
			{Name: ".", Path: "github.com/onsi/ginkgo/v2"},
			{Path: "golang.org/x/tools/go/ast/astutil"},
			{Path: "google.golang.org/grpc"},
		},
		// First organization imports (github.com/myorg)
		ImportGroup(OrgGroupBase + 0): {
			{Path: "github.com/myorg/project1/api", OrgIndex: 0, ProjectName: "project1"},
			{Path: "github.com/myorg/project1/internal", OrgIndex: 0, ProjectName: "project1"},
			{Path: "github.com/myorg/project2/cmd", OrgIndex: 0, ProjectName: "project2"},
			{Name: "p2pkg", Path: "github.com/myorg/project2/pkg", OrgIndex: 0, ProjectName: "project2"},
		},
		// Second organization imports (gitlab.com/anotherorg)
		ImportGroup(OrgGroupBase + 1): {
			{Path: "gitlab.com/anotherorg/common/db", OrgIndex: 1, ProjectName: "common"},
			{Path: "gitlab.com/anotherorg/common/utils", OrgIndex: 1, ProjectName: "common"},
			{Name: "_", Path: "gitlab.com/anotherorg/migrations/v1", OrgIndex: 1, ProjectName: "migrations"},
			{Path: "gitlab.com/anotherorg/service/api", OrgIndex: 1, ProjectName: "service"},
		},
		// Project imports
		ProjectGroup: {
			{Path: "github.com/test/project/internal/config"},
			{Path: "github.com/test/project/internal/handlers"},
			{Name: "models", Path: "github.com/test/project/pkg/models"},
			{Path: "github.com/test/project/pkg/utils"},
		},
	}

	newFile := g.replaceImports(astFile, groupedImports)

	// Check that new file has the expected structure
	req.GreaterOrEqual(len(newFile.Decls), 2, "New file should have import declaration and other declarations")

	// First declaration should be import
	genDecl, ok := newFile.Decls[0].(*ast.GenDecl)
	req.True(ok, "First declaration should be *ast.GenDecl")
	req.Equal(token.IMPORT, genDecl.Tok, "First declaration should be import declaration")

	// Verify all import groups are present
	totalExpectedImports := len(groupedImports[StdGroup]) +
		len(groupedImports[ThirdPartyGroup]) +
		len(groupedImports[ImportGroup(OrgGroupBase+0)]) +
		len(groupedImports[ImportGroup(OrgGroupBase+1)]) +
		len(groupedImports[ProjectGroup])

	req.Equal(totalExpectedImports, len(genDecl.Specs), "Expected correct number of import specs")

	// Verify specific imports are present with correct aliases
	foundImports := make(map[string]*ast.ImportSpec)
	for _, spec := range genDecl.Specs {
		if importSpec, ok := spec.(*ast.ImportSpec); ok {
			path := strings.Trim(importSpec.Path.Value, `"`)
			foundImports[path] = importSpec
		}
	}

	// Test standard library imports
	spec, exists := foundImports["fmt"]
	req.True(exists, "fmt import should be present")
	req.Nil(spec.Name, "fmt import should not have an alias")

	// Test aliased third-party import
	spec, exists = foundImports["github.com/pkg/errors"]
	req.True(exists, "github.com/pkg/errors import should be present")
	req.NotNil(spec.Name, "github.com/pkg/errors should have an alias")
	req.Equal("errors", spec.Name.Name, "github.com/pkg/errors should have 'errors' alias")

	// Test blank import
	spec, exists = foundImports["github.com/lib/pq"]
	req.True(exists, "github.com/lib/pq import should be present")
	req.NotNil(spec.Name, "github.com/lib/pq should have an alias")
	req.Equal("_", spec.Name.Name, "github.com/lib/pq should have blank import alias")

	// Test dot import
	spec, exists = foundImports["github.com/onsi/ginkgo/v2"]
	req.True(exists, "github.com/onsi/ginkgo/v2 import should be present")
	req.NotNil(spec.Name, "github.com/onsi/ginkgo/v2 should have an alias")
	req.Equal(".", spec.Name.Name, "github.com/onsi/ginkgo/v2 should have dot import alias")

	// Test org imports
	_, exists = foundImports["github.com/myorg/project1/api"]
	req.True(exists, "github.com/myorg/project1/api import should be present")

	_, exists = foundImports["gitlab.com/anotherorg/common/db"]
	req.True(exists, "gitlab.com/anotherorg/common/db import should be present")

	// Test project imports with alias
	spec, exists = foundImports["github.com/test/project/pkg/models"]
	req.True(exists, "github.com/test/project/pkg/models import should be present")
	req.NotNil(spec.Name, "github.com/test/project/pkg/models should have an alias")
	req.Equal("models", spec.Name.Name, "github.com/test/project/pkg/models should have 'models' alias")

	// Test that imports are ordered correctly (this is a basic check - full ordering is tested elsewhere)
	firstImportPath := strings.Trim(genDecl.Specs[0].(*ast.ImportSpec).Path.Value, `"`)
	req.True(g.isStdImport(firstImportPath), "First import should be from standard library")

	// Print the AST structure of newFile
	t.Logf("AST structure of newFile:")
	if err := ast.Print(g.fileSet, newFile); err != nil {
		t.Logf("Failed to print AST: %v", err)
	}

	// Format and print the new file for verification
	formatted, err := g.formatFile(newFile)
	req.NoError(err, "Failed to format new file")
	outputStr := string(formatted)

	t.Logf("Formatted new file after replaceImports:\n%s", outputStr)
}

func TestFormatter_replaceImports_FormattedOutput(t *testing.T) {
	req := require.New(t)

	orgs := []string{"github.com/myorg", "gitlab.com/anotherorg"}
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           orgs,
		CurrentProject: "github.com/test/project",
		InPlace:        false,
	})

	// Create a test AST file
	testContent := `package test

import (
	"fmt"
)

func main() {
	fmt.Println("hello")
}
`

	astFile, err := parseString(testContent)
	req.NoError(err, "Failed to parse test content")

	// Create grouped imports with realistic examples
	groupedImports := map[ImportGroup][]Import{
		StdGroup: {
			{Path: "context"},
			{Path: "fmt"},
			{Path: "net/http"},
			{Path: "strings"},
		},
		ThirdPartyGroup: {
			{Path: "github.com/external/lib"},
			{Name: "errors", Path: "github.com/pkg/errors"},
			{Name: "_", Path: "github.com/lib/pq"},
		},
		ImportGroup(OrgGroupBase + 0): {
			{Path: "github.com/myorg/project1/api", OrgIndex: 0, ProjectName: "project1"},
			{Path: "github.com/myorg/project2/cmd", OrgIndex: 0, ProjectName: "project2"},
		},
		ImportGroup(OrgGroupBase + 1): {
			{Path: "gitlab.com/anotherorg/common/db", OrgIndex: 1, ProjectName: "common"},
		},
		ProjectGroup: {
			{Path: "github.com/test/project/internal/config"},
			{Name: "models", Path: "github.com/test/project/pkg/models"},
		},
	}

	newFile := g.replaceImports(astFile, groupedImports)

	// Format the file to see the actual output
	formatted, err := g.formatFile(newFile)
	req.NoError(err, "Failed to format file")
	outputStr := string(formatted)

	t.Logf("Formatted output:\n%s", outputStr)

	// Verify the output contains all expected imports
	expectedImports := []string{
		`"context"`,
		`"fmt"`,
		`"net/http"`,
		`"strings"`,
		`"github.com/external/lib"`,
		`errors "github.com/pkg/errors"`,
		`_ "github.com/lib/pq"`,
		`"github.com/myorg/project1/api"`,
		`"github.com/myorg/project2/cmd"`,
		`"gitlab.com/anotherorg/common/db"`,
		`"github.com/test/project/internal/config"`,
		`models "github.com/test/project/pkg/models"`,
	}

	for _, expectedImport := range expectedImports {
		req.Contains(outputStr, expectedImport, "Output should contain import: %s", expectedImport)
	}

	// Verify the package declaration and function are preserved
	req.Contains(outputStr, "package test", "Output should contain package declaration")
	req.Contains(outputStr, "func main()", "Output should contain main function")
	req.Contains(outputStr, `fmt.Println("hello")`, "Output should contain function body")

	// Verify import structure
	req.Contains(outputStr, "import (", "Output should contain import block")
}

func TestFormatter_addGroupImports(t *testing.T) {
	req := require.New(t)
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           []string{},
		CurrentProject: "",
		InPlace:        false,
	})

	importDecl := &ast.GenDecl{
		Tok:    token.IMPORT,
		Lparen: token.Pos(1),
	}

	imports := []Import{
		{Path: "fmt", Comment: "test comment"},
		{Name: "alias", Path: "github.com/pkg/errors"},
		{Name: "_", Path: "github.com/lib/pq"},
	}

	g.addGroupImports(importDecl, imports)

	req.Len(importDecl.Specs, 3, "Expected 3 import specs")

	// Check first import
	spec, ok := importDecl.Specs[0].(*ast.ImportSpec)
	req.True(ok, "First spec should be an ImportSpec")
	req.Equal(`"fmt"`, spec.Path.Value, "Expected fmt import")
	req.Nil(spec.Name, "fmt import should not have alias")
	req.Equal("test comment\n", spec.Comment.Text(), "First import should have comment")

	// Check aliased import
	spec, ok = importDecl.Specs[1].(*ast.ImportSpec)
	req.True(ok, "Second spec should be an ImportSpec")
	req.NotNil(spec.Name, "Second import should have alias name")
	req.Equal("alias", spec.Name.Name, "Second import should have correct alias")
}

func TestFormatter_addOrgImports(t *testing.T) {
	req := require.New(t)
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           []string{"github.com/myorg"},
		CurrentProject: "",
		InPlace:        false,
	})

	importDecl := &ast.GenDecl{
		Tok:    token.IMPORT,
		Lparen: token.Pos(1),
	}

	imports := []Import{
		{Path: "github.com/myorg/project1/pkg", OrgIndex: 0, ProjectName: "project1"},
		{Path: "github.com/myorg/project1/cmd", OrgIndex: 0, ProjectName: "project1"},
		{Path: "github.com/myorg/project2/api", OrgIndex: 0, ProjectName: "project2"},
	}

	g.addOrgImports(importDecl, imports)

	req.Len(importDecl.Specs, 3, "Expected 3 import specs")
}

func TestFormatter_formatFile(t *testing.T) {
	req := require.New(t)
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           []string{},
		CurrentProject: "",
		InPlace:        false,
	})

	testContent := `package test

import "fmt"

func main() {
	fmt.Println("hello")
}
`

	astFile, err := parseString(testContent)
	req.NoError(err, "Failed to parse test content")

	result, err := g.formatFile(astFile)
	req.NoError(err, "formatFile() should not error")

	req.NotEmpty(result, "formatFile() should return non-empty result")

	// Should be valid Go code
	resultStr := string(result)
	req.Contains(resultStr, "package test", "Formatted result should contain package declaration")
}

func TestFormatter_formatImportSpec(t *testing.T) {
	req := require.New(t)
	g := New(FormatterConfig{
		FilePath:       "test.go",
		Orgs:           []string{},
		CurrentProject: "",
		InPlace:        false,
	})

	tests := []struct {
		name     string
		spec     *ast.ImportSpec
		expected string
	}{
		{
			name: "simple import without name or comment",
			spec: &ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"fmt"`,
				},
			},
			expected: `"fmt"`,
		},
		{
			name: "import with alias",
			spec: &ast.ImportSpec{
				Name: &ast.Ident{Name: "f"},
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"fmt"`,
				},
			},
			expected: `f "fmt"`,
		},
		{
			name: "import with dot alias",
			spec: &ast.ImportSpec{
				Name: &ast.Ident{Name: "."},
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"fmt"`,
				},
			},
			expected: `. "fmt"`,
		},
		{
			name: "import with underscore (blank import)",
			spec: &ast.ImportSpec{
				Name: &ast.Ident{Name: "_"},
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"github.com/lib/pq"`,
				},
			},
			expected: `_ "github.com/lib/pq"`,
		},
		{
			name: "import with comment",
			spec: &ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"fmt"`,
				},
				Comment: &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: "// for formatting"},
					},
				},
			},
			expected: `"fmt" // for formatting`,
		},
		{
			name: "import with comment that has extra whitespace",
			spec: &ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"fmt"`,
				},
				Comment: &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: "//   for formatting   "},
					},
				},
			},
			expected: `"fmt" // for formatting`,
		},
		{
			name: "import with alias and comment",
			spec: &ast.ImportSpec{
				Name: &ast.Ident{Name: "f"},
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"fmt"`,
				},
				Comment: &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: "// formatting package"},
					},
				},
			},
			expected: `f "fmt" // formatting package`,
		},
		{
			name: "import with empty comment",
			spec: &ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"fmt"`,
				},
				Comment: &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: "//   "},
					},
				},
			},
			expected: `"fmt"`,
		},
		{
			name: "complex import path with alias and comment",
			spec: &ast.ImportSpec{
				Name: &ast.Ident{Name: "formatter"},
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"github.com/myorg/go-imports-group/pkg/formatter"`,
				},
				Comment: &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: "// import formatting functionality"},
					},
				},
			},
			expected: `formatter "github.com/myorg/go-imports-group/pkg/formatter" // import formatting functionality`,
		},
		{
			name: "nil path (edge case)",
			spec: &ast.ImportSpec{
				Name: &ast.Ident{Name: "test"},
			},
			expected: `test`,
		},
		{
			name: "nil name and comment (minimal spec)",
			spec: &ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"os"`,
				},
			},
			expected: `"os"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.formatImportSpec(tt.spec)
			req.Equal(tt.expected, result, "formatImportSpec() result mismatch")
		})
	}
}
