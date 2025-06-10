package formatter

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"

	"github.com/siyuan-infoblox/go-imports-group/pkg/errors"
	"github.com/siyuan-infoblox/go-imports-group/pkg/std"
	"github.com/siyuan-infoblox/go-imports-group/pkg/utils"
)

type FormatterConfig struct {
	FilePath       string   // path to the Go source file
	Orgs           []string // organization prefixes to group imports by
	CurrentProject string   // optional current project override
	InPlace        bool     // whether to modify the file in place
}

// formatter handles the import grouping logic
type formatter struct {
	config  FormatterConfig
	fileSet *token.FileSet
}

// New creates a new Formatter with the specified organization prefixes and optional current project
func New(config FormatterConfig) *formatter {
	return &formatter{
		config:  config,
		fileSet: token.NewFileSet(),
	}
}

func (g *formatter) getFilePath() string {
	return g.config.FilePath
}

func (g *formatter) getOrgs() []string {
	return g.config.Orgs
}

func (g *formatter) getCurrentProject() string {
	if g.config.CurrentProject == "" {
		// If no current project is specified, try to infer it from the file path
		return utils.GetProjectModule(g.getFilePath())
	}
	return g.config.CurrentProject
}

func (g *formatter) getInPlace() bool {
	return g.config.InPlace
}

// extractImports extracts import information from the AST
func (g *formatter) extractImports(file *ast.File) []Import {
	var imports []Import
	seen := make(map[string]bool) // Track which paths we've seen

	for _, importSpec := range file.Imports {
		path := strings.Trim(importSpec.Path.Value, `"`)

		// Skip if we've already seen this path
		if seen[path] {
			continue
		}
		seen[path] = true

		imp := Import{
			Path: path,
		}

		if importSpec.Name != nil {
			imp.Name = importSpec.Name.Name
		}

		if importSpec.Comment != nil {
			imp.Comment = importSpec.Comment.Text()
		}

		imports = append(imports, imp)
	}

	return imports
}

// groupImports categorizes imports into different groups
func (g *formatter) groupImports(imports []Import, filePath string) map[ImportGroup][]Import {
	grouped := make(map[ImportGroup][]Import)
	projectModule := g.getCurrentProject()
	if projectModule == "" {
		// If no current project is specified, try to infer it from the file path
		projectModule = utils.GetProjectModule(filePath)
	}
	for i := range imports {
		imports[i].Group = g.classifyImport(imports[i].Path, projectModule)

		// Update condition to check for any org group
		if imports[i].Group >= OrgGroupBase {
			imports[i].OrgIndex, imports[i].ProjectName = g.getOrgInfo(imports[i].Path)
		}

		grouped[imports[i].Group] = append(grouped[imports[i].Group], imports[i])
	}
	// Sort imports within each group
	for group := range grouped {
		g.sortImportsInGroup(grouped[group], group)
	}

	return grouped
}

// classifyImport determines which group an import belongs to
func (g *formatter) classifyImport(importPath, projectModule string) ImportGroup {
	// Check if it's a standard library import
	if g.isStdImport(importPath) {
		return StdGroup
	}

	// Check if it's a project import
	if strings.HasPrefix(importPath, projectModule) {
		return ProjectGroup
	}

	// Check if it's an organization import - assign separate group per org
	for i, org := range g.getOrgs() {
		if strings.HasPrefix(importPath, org) {
			return ImportGroup(OrgGroupBase + i)
		}
	}

	// Default to third-party
	return ThirdPartyGroup
}

// isStdImport checks if an import path is from the Go standard library
func (g *formatter) isStdImport(importPath string) bool {
	return std.IsStandardPackage(importPath)
}

// getOrgInfo returns the organization index and project name for an org import
func (g *formatter) getOrgInfo(importPath string) (int, string) {
	for i, org := range g.getOrgs() {
		if strings.HasPrefix(importPath, org) {
			// Extract project name (next path segment after org)
			remaining := strings.TrimPrefix(importPath, org)
			remaining = strings.TrimPrefix(remaining, "/")
			segments := strings.Split(remaining, "/")
			if len(segments) > 0 {
				return i, segments[0]
			}
			return i, ""
		}
	}
	return -1, ""
}

// sortImportsInGroup sorts imports within a group
func (g *formatter) sortImportsInGroup(imports []Import, group ImportGroup) {
	if group >= OrgGroupBase {
		// Sort org imports by project name, then alphabetically
		sort.Slice(imports, func(i, j int) bool {
			if imports[i].ProjectName != imports[j].ProjectName {
				return imports[i].ProjectName < imports[j].ProjectName
			}
			return imports[i].Path < imports[j].Path
		})
	} else {
		// Sort alphabetically
		sort.Slice(imports, func(i, j int) bool {
			return imports[i].Path < imports[j].Path
		})
	}
}

// replaceImports replaces the imports in the AST with the grouped imports
func (g *formatter) replaceImports(file *ast.File, groupedImports map[ImportGroup][]Import) *ast.File {
	// Remove existing imports
	var newDecls []ast.Decl
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			continue // Skip import declarations
		}
		newDecls = append(newDecls, decl)
	}

	// Create new import declaration
	if hasImports := len(groupedImports[StdGroup]) > 0 ||
		len(groupedImports[ThirdPartyGroup]) > 0 ||
		len(groupedImports[ProjectGroup]) > 0 ||
		g.hasOrgImports(groupedImports); hasImports {

		importDecl := &ast.GenDecl{
			Tok:    token.IMPORT,
			Lparen: token.Pos(1), // Enable parentheses
		}

		// Add std imports
		if imports := groupedImports[StdGroup]; len(imports) > 0 {
			g.addGroupImports(importDecl, imports)
		}

		// Add third-party imports
		if imports := groupedImports[ThirdPartyGroup]; len(imports) > 0 {
			g.addGroupImports(importDecl, imports)
		}

		// Add org imports in order
		for i := range g.getOrgs() {
			orgGroup := ImportGroup(OrgGroupBase + i)
			if imports := groupedImports[orgGroup]; len(imports) > 0 {
				g.addOrgImports(importDecl, imports)
			}
		}

		// Add project imports
		if imports := groupedImports[ProjectGroup]; len(imports) > 0 {
			g.addGroupImports(importDecl, imports)
		}

		// Insert import declaration at the beginning
		newDecls = append([]ast.Decl{importDecl}, newDecls...)
	}

	file.Decls = newDecls
	return file
}

// hasOrgImports checks if there are any organization imports
func (g *formatter) hasOrgImports(groupedImports map[ImportGroup][]Import) bool {
	for i := range g.getOrgs() {
		orgGroup := ImportGroup(OrgGroupBase + i)
		if len(groupedImports[orgGroup]) > 0 {
			return true
		}
	}
	return false
}

// addGroupImports adds imports for a regular group
func (g *formatter) addGroupImports(importDecl *ast.GenDecl, imports []Import) {
	for _, imp := range imports {
		spec := &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf(`"%s"`, imp.Path),
			},
		}

		if imp.Name != "" {
			spec.Name = &ast.Ident{Name: imp.Name}
		}

		if imp.Comment != "" {
			spec.Comment = &ast.CommentGroup{
				List: []*ast.Comment{
					{
						Text: imp.Comment,
					},
				},
			}
		}

		importDecl.Specs = append(importDecl.Specs, spec)
	}
}

// addOrgImports adds organization imports with project-level grouping
func (g *formatter) addOrgImports(importDecl *ast.GenDecl, imports []Import) {
	if len(imports) == 0 {
		return
	}

	// Group by org and project
	type OrgProject struct {
		OrgIndex    int
		ProjectName string
	}

	projectGroups := make(map[OrgProject][]Import)
	for _, imp := range imports {
		key := OrgProject{
			OrgIndex:    imp.OrgIndex,
			ProjectName: imp.ProjectName,
		}
		projectGroups[key] = append(projectGroups[key], imp)
	}

	// Get sorted keys
	var keys []OrgProject
	for key := range projectGroups {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].OrgIndex != keys[j].OrgIndex {
			return keys[i].OrgIndex < keys[j].OrgIndex
		}
		return keys[i].ProjectName < keys[j].ProjectName
	})

	// Add imports (spacing will be handled in post-processing)
	for _, key := range keys {
		projectImports := projectGroups[key]
		g.addGroupImports(importDecl, projectImports)
	}
}

// formatFile formats the AST back to Go source code while preserving import grouping
func (g *formatter) formatFile(file *ast.File) ([]byte, error) {
	// Extract imports and remove them from the AST temporarily
	originalImports := file.Imports
	originalDecls := file.Decls
	var importDecl *ast.GenDecl
	var nonImportDecls []ast.Decl

	// Find and separate import declarations from other declarations
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			importDecl = genDecl
		} else {
			nonImportDecls = append(nonImportDecls, decl)
		}
	}

	// Temporarily remove imports from the file for formatting
	file.Imports = nil
	file.Decls = nonImportDecls

	// Format the file without imports
	var buf strings.Builder
	err := format.Node(&buf, g.fileSet, file)
	if err != nil {
		// Restore original state
		file.Imports = originalImports
		file.Decls = originalDecls
		return nil, err
	}

	// Restore original state
	file.Imports = originalImports
	file.Decls = originalDecls

	// Get the formatted content
	lines := strings.Split(buf.String(), "\n")

	// Find where to insert imports (after package declaration)
	var result []string
	packageLineFound := false

	for _, line := range lines {
		result = append(result, line)
		if !packageLineFound && strings.HasPrefix(strings.TrimSpace(line), "package ") {
			packageLineFound = true
			result = append(result, "") // Add blank line after package

			// Add custom formatted imports
			if importDecl != nil && len(importDecl.Specs) > 0 {
				result = append(result, "import (")

				// Format each import spec preserving the order from replaceImports
				for i, spec := range importDecl.Specs {
					if importSpec, ok := spec.(*ast.ImportSpec); ok {
						importLine := g.formatImportSpec(importSpec)

						// Add spacing based on group changes
						if i > 0 && g.shouldAddSpacingBetweenImports(importDecl.Specs, i) {
							result = append(result, "")
						}

						result = append(result, "\t"+importLine)
					}
				}

				result = append(result, ")")
				result = append(result, "") // Add blank line after imports
			}
		}
	}

	return []byte(strings.Join(result, "\n")), nil
}

// formatImportSpec formats a single import spec
func (g *formatter) formatImportSpec(spec *ast.ImportSpec) string {
	var parts []string

	if spec.Name != nil {
		parts = append(parts, spec.Name.Name)
	}

	if spec.Path != nil {
		parts = append(parts, spec.Path.Value)
	}

	if spec.Comment != nil {
		comment := strings.TrimSpace(spec.Comment.Text())
		if comment != "" {
			parts = append(parts, "// "+comment)
		}
	}

	return strings.Join(parts, " ")
}

// shouldAddSpacingBetweenImports determines if spacing should be added between imports
func (g *formatter) shouldAddSpacingBetweenImports(specs []ast.Spec, currentIndex int) bool {
	if currentIndex == 0 {
		return false
	}

	currentSpec, ok := specs[currentIndex].(*ast.ImportSpec)
	if !ok || currentSpec.Path == nil {
		return false
	}

	prevSpec, ok := specs[currentIndex-1].(*ast.ImportSpec)
	if !ok || prevSpec.Path == nil {
		return false
	}

	// Remove quotes from import paths
	currentPath := strings.Trim(currentSpec.Path.Value, "\"")
	prevPath := strings.Trim(prevSpec.Path.Value, "\"")

	// Classify both imports
	currentGroup := g.classifyImport(currentPath, g.getCurrentProject())
	prevGroup := g.classifyImport(prevPath, g.getCurrentProject())

	// Different groups need spacing
	if currentGroup != prevGroup {
		return true
	}

	// Same group - check for organization project differences
	if currentGroup >= OrgGroupBase {
		_, currentOrgProject := g.getOrgInfo(currentPath)
		_, prevOrgProject := g.getOrgInfo(prevPath)
		if currentOrgProject != prevOrgProject && prevOrgProject != "" && currentOrgProject != "" {
			return true
		}
	}

	return false
}

// extractImportsOnly creates a minimal Go file containing only package declaration and imports
func (g *formatter) extractImportsOnly(file *ast.File) ([]byte, error) {
	// Create a new file with only package declaration and imports
	newFile := &ast.File{
		Name:    file.Name,
		Imports: file.Imports,
	}

	// Find and copy only the import declarations
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			newFile.Decls = append(newFile.Decls, genDecl)
		}
	}

	// Use the same custom formatting logic as formatFile but only for imports
	return g.formatImportsOnly(newFile)
}

// formatImportsOnly formats only the package declaration and imports with proper spacing
func (g *formatter) formatImportsOnly(file *ast.File) ([]byte, error) {
	var result []string

	// Add package declaration
	result = append(result, fmt.Sprintf("package %s", file.Name.Name))
	result = append(result, "") // Blank line after package

	// Find import declaration
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			if len(genDecl.Specs) > 0 {
				result = append(result, "import (")

				// Format each import spec with proper spacing
				for i, spec := range genDecl.Specs {
					if importSpec, ok := spec.(*ast.ImportSpec); ok {
						importLine := g.formatImportSpec(importSpec)

						// Add spacing based on group changes
						if i > 0 && g.shouldAddSpacingBetweenImports(genDecl.Specs, i) {
							result = append(result, "")
						}

						result = append(result, "\t"+importLine)
					}
				}

				result = append(result, ")")
			}
			break
		}
	}

	return []byte(strings.Join(result, "\n") + "\n"), nil
}

// ProcessFileWithOutput processes a Go source file with optional output control
func (g *formatter) ProcessFileWithOutput(verbose bool) error {
	if verbose {
		fmt.Print(errors.InfoMsgCurrentProjectOutput, g.getCurrentProject(), "\n")
	}
	src, err := os.ReadFile(g.getFilePath())
	if err != nil {
		return fmt.Errorf("%s: %w", errors.ErrMsgFailedToReadFile, err)
	}

	file, err := parser.ParseFile(g.fileSet, g.getFilePath(), src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("%s: %w", errors.ErrMsgFailedToParseFile, err)
	}

	if len(file.Imports) == 0 {
		// No imports to process
		if g.getInPlace() {
			return nil
		}
		if !g.getInPlace() && verbose {
			fmt.Print(string(src))
		}
		return nil
	}

	imports := g.extractImports(file)
	groupedImports := g.groupImports(imports, g.getFilePath())
	newFile := g.replaceImports(file, groupedImports)

	var output []byte
	output, err = g.formatFile(newFile)
	if err != nil {
		return fmt.Errorf("%s: %w", errors.ErrMsgFailedToFormatFile, err)
	}

	if g.getInPlace() {
		return os.WriteFile(g.getFilePath(), output, 0644)
	}

	if verbose {
		// For stdout output, show only import declarations using AST
		importsOnly, err := g.extractImportsOnly(newFile)
		if err != nil {
			return fmt.Errorf("%s: %w", errors.ErrMsgFailedToExtractImports, err)
		}
		fmt.Print(string(importsOnly))
	}
	return nil
}

// ProcessFile processes a Go source file and groups its imports
func (g *formatter) ProcessFile() error {
	return g.ProcessFileWithOutput(true)
}

// ProcessFiles processes multiple Go source files and groups their imports
func (g *formatter) ProcessFiles(filePaths []string) error {
	processedCount := 0
	errorCount := 0

	for _, filePath := range filePaths {
		g.config.FilePath = filePath
		if err := g.ProcessFileWithOutput(false); err != nil {
			fmt.Printf(errors.InfoMsgErrorProcessing+"\n", filePath, err)
			errorCount++
		} else {
			processedCount++
			if g.getInPlace() {
				fmt.Printf(errors.InfoMsgProcessedFiles+"\n", filePath)
			}
		}
	}

	fmt.Printf(errors.InfoMsgProcessedCount, processedCount)
	if errorCount > 0 {
		fmt.Printf(errors.InfoMsgErrorCount, errorCount)
	}
	fmt.Println()

	if errorCount > 0 {
		return fmt.Errorf(errors.ErrMsgFilesFailedToProcess, errorCount)
	}
	return nil
}

// ProcessPath processes a file or directory path
func (g *formatter) ProcessPath(path string) error {
	isDir, err := utils.IsDirectory(path)
	if err != nil {
		return fmt.Errorf("%s: %w", errors.ErrMsgFailedToCheckPath, err)
	}

	if isDir {
		// When processing directories, in-place mode is recommended
		if !g.getInPlace() {
			fmt.Printf(errors.WarnMsgProcessingDirWithoutInPlace + "\n")
			fmt.Printf(errors.InfoMsgUseInPlaceFlag + "\n\n")
		}

		// Find all Go files in the directory
		goFiles, err := utils.FindGoFiles(path)
		if err != nil {
			return fmt.Errorf("%s: %w", errors.ErrMsgFailedToFindGoFiles, err)
		}

		if len(goFiles) == 0 {
			fmt.Printf(errors.InfoMsgNoGoFilesFound+"\n", path)
			return nil
		}

		fmt.Printf(errors.InfoMsgFoundGoFiles+"\n", len(goFiles), path)
		if g.getCurrentProject() != "" {
			fmt.Printf(errors.InfoMsgCurrentProject+"\n", g.getCurrentProject())
		}
		fmt.Println()

		return g.ProcessFiles(goFiles)
	} else {
		g.config.FilePath = path
		return g.ProcessFile()
	}
}
