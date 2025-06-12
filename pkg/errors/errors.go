package errors

// Error message constants for the go-imports-group application
const (
	// File processing errors
	ErrMsgFailedToReadFile       = "failed to read file"
	ErrMsgFailedToParseFile      = "failed to parse file"
	ErrMsgFailedToFormatFile     = "failed to format file"
	ErrMsgFailedToExtractImports = "failed to extract imports"

	// Directory processing errors
	ErrMsgFailedToCheckPath    = "failed to check path"
	ErrMsgFailedToFindGoFiles  = "failed to find Go files in directory"
	ErrMsgFilesFailedToProcess = "%d files failed to process"

	// Standard library generation errors
	ErrMsgGORootNotFound        = "GOROOT not found"
	ErrMsgFailedToGetWorkingDir = "failed to get current working directory"

	// Info/warning messages
	WarnMsgProcessingDirWithoutInPlace = "Warning: Processing directory without --in-place flag. No files will be modified."
	InfoMsgUseInPlaceFlag              = "Use --in-place flag to modify files or specify a single file for stdout output."
	InfoMsgNoGoFilesFound              = "No Go files found in directory: %s"
	InfoMsgFoundGoFiles                = "Found %d Go files in directory: %s"
	InfoMsgCurrentProject              = "Current project: %s"
	InfoMsgProcessedFiles              = "Processed: %s"
	InfoMsgErrorProcessing             = "Error processing %s: %v"
	InfoMsgProcessedCount              = "\nProcessed %d files successfully"
	InfoMsgErrorCount                  = ", %d files had errors"
	InfoMsgCurrentProjectOutput        = "current project: "
)
