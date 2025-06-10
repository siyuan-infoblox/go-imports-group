package version

import (
	"fmt"
	"runtime"
)

var (
	// These variables are set at build time using ldflags
	Version   = "dev"
	GitCommit = "unknown"
	GitTag    = "unknown"
	BuildDate = "unknown"
)

// Info holds version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	GitTag    string `json:"gitTag"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

// Get returns version information
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		GitTag:    GitTag,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a human-readable version string
func (i Info) String() string {
	return fmt.Sprintf("gig version %s\nGit commit: %s\nGit tag: %s\nBuild date: %s\nGo version: %s\nCompiler: %s\nPlatform: %s",
		i.Version, i.GitCommit, i.GitTag, i.BuildDate, i.GoVersion, i.Compiler, i.Platform)
}
