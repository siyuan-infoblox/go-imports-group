# GIG - Go Imports Group

GIG (Go Imports Group) is a command-line tool that organizes and sorts Go imports into logical groups for better code readability and consistency.

## Features

- **Smart Import Grouping**: Automatically categorizes imports into:
  - Go standard library
  - Third-party packages
  - Organization/company packages
  - Current project packages

- **Organization-aware**: Groups organization imports by project and respects the order specified via `--orgs` flag

- **Alphabetical Sorting**: Sorts imports within each group alphabetically

- **Project Sub-grouping**: Automatically groups organization imports by individual projects

- **Flexible Output**: Supports both in-place editing and stdout output

## Installation

### Using go install

```bash
go install github.com/siyuan-infoblox/go-imports-group@latest
```

### From Source

```bash
git clone https://github.com/siyuan-infoblox/go-imports-group.git
cd go-imports-group
make build
```

### Using Docker

```bash
# Build the Docker image
make docker-build

# Or build manually
docker build -t go-imports-group .
```

### Building for Multiple Platforms

You can build binaries for multiple platforms:

```bash
# Build for all platforms
make build-all
```

This creates binaries for:
- `gig-linux-amd64` (Linux x86_64)
- `gig-linux-arm64` (Linux ARM64)
- `gig-darwin-amd64` (macOS Intel)
- `gig-darwin-arm64` (macOS Apple Silicon)
- `gig-windows-amd64.exe` (Windows x86_64)
- `gig-windows-arm64.exe` (Windows ARM64)

## Usage

### Basic Usage

```bash
# Group imports in a single file and print to stdout
gig path/to/file.go

# Process all Go files in a directory
gig --in-place .

# Process all Go files in a specific directory
gig --in-place path/to/directory

# Modify file in-place
gig --in-place path/to/file.go

# Specify organization order
gig --orgs=github.com/myorg,github.com/acme-corp path/to/file.go

# Show version information
gig --version
```

### Testing Your Installation

After building, you can test the tool with the included examples:

```bash
# Test with example file
make run-example

# Test with in-place modification
make run-example-inplace
```

### Docker Usage

```bash
# Run with Docker (mounts current directory)
docker run --rm -v $(pwd):/workspace go-imports-group /workspace/path/to/file.go

# With organization flags
docker run --rm -v $(pwd):/workspace go-imports-group \
  --orgs=github.com/myorg,github.com/acme-corp \
  --current-project=github.com/username/current-project \
  /workspace/path/to/file.go

# Test with example file
make docker-test
```

### Makefile Targets

The project includes several convenient Makefile targets:

```bash
make build           # Build the binary
make install         # Install the binary
make run-example     # Run example with the tool
make run-example-inplace # Run example with in-place modification
make docker-build    # Build Docker image
make docker-test     # Test Docker image with example
make build-all       # Build for multiple platforms
make clean           # Clean build artifacts
make clean-example   # Clean up example test files
```

### Command Line Options

- `--orgs`: Comma-separated list of organization prefixes to define the order of organization imports
- `--current-project`: Specify the current project module path (auto-detected from go.mod if not provided)
- `--in-place`: Modify the file(s) in place instead of printing to stdout (recommended when processing directories)
- `--version`, `-v`: Show version information including build details

### Directory Processing

When you specify a directory path, `gig` will:

1. Recursively find all `.go` files
2. Skip `vendor/`, `.git/`, and other hidden directories
3. Process each file and group its imports
4. Report progress and any errors encountered

**Note**: When processing directories, it's recommended to use the `--in-place` flag. Without it, the tool will only analyze the files without making changes.

### Example

Given a Go file with unorganized imports:

```go
package main

import (
    "github.com/acme-corp/platform.core/pkg/config"
    "fmt"
    "github.com/gorilla/mux"
    "github.com/acme-corp/shared-libs/log"
    "context"
    "github.com/myorg/toolkit/server"
    "github.com/acme-corp/platform.common/db/connector"
    "github.com/username/go-imports-group/pkg/service"
    "net/http"
    "github.com/stretchr/testify/assert"
    "encoding/json"
    "github.com/gin-gonic/gin"
    "github.com/myorg/auth-service/client"
    "database/sql"
    "github.com/username/go-imports-group/internal/config"
    "github.com/redis/go-redis/v9"
)
```

Running:
```bash
gig --orgs=github.com/myorg,github.com/acme-corp --current-project=github.com/username/go-imports-group example.go
```

Will produce:

```go
package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/gorilla/mux"
    "github.com/redis/go-redis/v9"
    "github.com/stretchr/testify/assert"

    "github.com/myorg/auth-service/client"
    "github.com/myorg/toolkit/server"

    "github.com/acme-corp/platform.common/db/connector"
    "github.com/acme-corp/platform.core/pkg/config"
    "github.com/acme-corp/shared-libs/log"

    "github.com/username/go-imports-group/internal/config"
    "github.com/username/go-imports-group/pkg/service"
)
```

## Import Grouping Logic

1. **Standard Library**: All Go standard library packages (e.g., `fmt`, `context`, `net/http`)

2. **Third-party**: External packages from public repositories (e.g., `github.com/gorilla/mux`)

3. **Organization**: Company/organization packages, further grouped by:
   - Organization priority (as specified in `--orgs`)
   - Project within organization
   - Alphabetical order within project

4. **Project**: Local project imports (determined from `--current-project` flag or auto-detected from `go.mod`)

## Development

### Prerequisites

- Go 1.21 or later (tested with Go 1.24.0)
- Docker (optional, for containerized usage)
- Make (for using Makefile targets)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/siyuan-infoblox/go-imports-group.git
cd go-imports-group

# Install dependencies
go mod download

# Build the binary
make build

# Run tests
make test
```

### Version Management

The project includes built-in version management tools:

```bash
# Show current version information
make version

# List recent tags
make list-tags

# Create version tags
make tag-patch  # Increment patch version (e.g., v1.0.0 -> v1.0.1)
make tag-minor  # Increment minor version (e.g., v1.0.0 -> v1.1.0)
make tag-major  # Increment major version (e.g., v1.0.0 -> v2.0.0)
```

### Maintenance Tasks

```bash
# Create a backup of the project
make backup

# Update the standard library package list (for development)
make update-std-package-list

# Clean up example test files
make clean-example
```

## Organization Sub-grouping

When multiple imports come from the same organization but different projects, GIG automatically creates sub-groups:

```go
import (
    // Standard library
    "context"
    "fmt"
    "net/http"
    "time"

    // Third-party
    "github.com/gin-gonic/gin"
    "github.com/gorilla/mux"
    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/assert"

    // MyOrg - Auth Service
    "github.com/myorg/auth-service/client"
    "github.com/myorg/auth-service/tokens"

    // MyOrg - Toolkit
    "github.com/myorg/toolkit/middleware"
    "github.com/myorg/toolkit/server"

    // Acme Corp - Platform Common
    "github.com/acme-corp/platform.common/db/connector"
    "github.com/acme-corp/platform.common/utils"

    // Acme Corp - Platform Core
    "github.com/acme-corp/platform.core/pkg/config"
    "github.com/acme-corp/platform.core/redis"

    // Acme Corp - Shared Libraries
    "github.com/acme-corp/shared-libs/log"
    "github.com/acme-corp/shared-libs/metrics"

    // Project imports
    "github.com/username/go-imports-group/internal/config"
    "github.com/username/go-imports-group/pkg/service"
)
```
