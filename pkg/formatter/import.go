package formatter

// Import represents a single import statement
type Import struct {
	Name        string // alias name, empty if no alias
	Path        string // import path
	Comment     string // inline comment
	Group       ImportGroup
	OrgIndex    int    // index in the org list for ordering
	ProjectName string // project name within org for sub-grouping
}

// ImportGroup represents different types of import groups
type ImportGroup int

const (
	StdGroup ImportGroup = iota
	ThirdPartyGroup
	ProjectGroup
	OrgGroupBase = 100 // Org groups will be dynamically assigned starting from this base
)
