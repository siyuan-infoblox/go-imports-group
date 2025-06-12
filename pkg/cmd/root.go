package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/siyuan-infoblox/go-imports-group/pkg/formatter"
)

const (
	UseDescription   = "gig [flags] PATH"
	ShortDescription = "Go imports grouper - A tool to group and sort Go imports"
	LongDescription  = `gig is a command-line tool that groups and sorts Go imports.

It organizes imports into groups:
1. Go standard library
2. Third-party packages
3. Organization/company packages (configurable)
4. Current project packages

Organization packages can be further subdivided by project.

PATH can be either a single Go file or a directory. When a directory is specified,
all Go source files (excluding test files) in the directory and subdirectories
will be processed recursively.`
)

var (
	orgs           []string
	currentProject string
	inPlace        bool
	showVersion    bool
	versionStr     string
)

var rootCmd = &cobra.Command{
	Use:          UseDescription,
	Short:        ShortDescription,
	Long:         LongDescription,
	Args:         validateArgs,
	RunE:         run,
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringSliceVar(&orgs, "orgs", []string{}, "Comma-separated list of organization prefixes (e.g., github.com/myorg,github.com/acme-corp)")
	rootCmd.PersistentFlags().StringVar(&currentProject, "current-project", "", "Name of the current project (e.g., github.com/username/go-imports-group)")
	rootCmd.PersistentFlags().BoolVar(&inPlace, "in-place", false, "Modify the file in place instead of printing to stdout")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
}

func validateArgs(cmd *cobra.Command, args []string) error {
	// If version flag is set, we don't need file arguments
	if showVersion {
		return nil
	}
	return cobra.ExactArgs(1)(cmd, args)
}

func run(cmd *cobra.Command, args []string) error {
	// Handle version flag
	if showVersion {
		fmt.Printf("Go Imports Group (GIG) version %s\n", versionStr)
		return nil
	}

	path := args[0]

	g := formatter.New(formatter.FormatterConfig{
		FilePath:       path, // This will be updated for each file when processing directories
		Orgs:           orgs,
		CurrentProject: currentProject,
		InPlace:        inPlace,
	})
	return g.ProcessPath(path)
}

func Execute(version string) error {
	versionStr = version
	return rootCmd.Execute()
}
