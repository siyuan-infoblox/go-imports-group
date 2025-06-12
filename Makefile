.PHONY: build build-all release-archives test clean install run-example clean-example vendor docker-build docker-test backup update-std-package-list version tag-major tag-minor tag-patch list-tags

# Binary name
BINARY_NAME=gig
BUILD_DIR=./build
BACKUP_DIR=../go-backup/go-imports-group-backup

# Version variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_TAG ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Test go files directory
TEST_GO_FILES_DIR=examples/test_go_files

# Build the binary
build:
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) main.go

# Build for multiple platforms
build-all: clean
	@mkdir -p $(BUILD_DIR)
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 main.go
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 main.go
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 main.go
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe main.go
	@GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe main.go
	@echo "Cross-platform builds completed"
	@ls -la $(BUILD_DIR)/

# Create release archives
release-archives: build-all
	@mkdir -p $(BUILD_DIR)/releases
	@echo "Creating release archives..."
	@cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	@cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@cd $(BUILD_DIR) && zip -q releases/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@cd $(BUILD_DIR) && zip -q releases/$(BINARY_NAME)-$(VERSION)-windows-arm64.zip $(BINARY_NAME)-windows-arm64.exe
	@echo "Release archives created:"
	@ls -la $(BUILD_DIR)/releases/

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Git Tag: $(GIT_TAG)"
	@echo "Build Date: $(BUILD_DATE)"

# Clean build artifacts
clean:
	go clean
	rm -rf $(BUILD_DIR)

# Test the application
test:
	go test ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Vendor dependencies
vendor:
	go mod tidy && go mod vendor

# Install the binary
install:
	go install $(LDFLAGS)

# Clean example files
clean-example:
	@echo "Cleaning up example files..."
	@rm -rf ${TEST_GO_FILES_DIR} 2>/dev/null || true
	@echo "Example files cleaned up."

# Create example files
create-example: clean-example
	@mkdir -p $(TEST_GO_FILES_DIR)
	@echo "Copying example files to ${TEST_GO_FILES_DIR}..."
	@rsync examples/example.go.txt ${TEST_GO_FILES_DIR}/example.go

# Run example
run-example: clean-example build create-example
	@echo "Running example with specified orgs and current project..."
	@$(BUILD_DIR)/$(BINARY_NAME) --orgs=github.com/myorg,github.com/acme-corp --current-project=github.com/username/go-imports-group --in-place ${TEST_GO_FILES_DIR}/example.go
	

# Run example with in-place modification
run-example-inplace: build create-example
	@echo "Running example with in-place modification..."
	@$(BUILD_DIR)/$(BINARY_NAME) --orgs=github.com/myorg,github.com/acme-corp --in-place ${TEST_GO_FILES_DIR}/
	@echo "Files modified in-place."

# Format and lint
fmt:
	go fmt ./...
	go vet ./...
	@echo "Build artifacts created in $(BUILD_DIR)"

# Build Docker image
docker-build:
	docker build -t go-imports-group .

# Test Docker image with example file
docker-test: docker-build create-example
	docker run --rm -v $(PWD):/workspace go-imports-group --orgs=github.com/myorg,github.com/acme-corp --current-project=github.com/username/go-imports-group /workspace/$(TEST_GO_FILES_DIR)/example.go

# Create backup of the project
backup:
	@echo "Creating backup of project..."
	@mkdir -p $(BACKUP_DIR)
	@rsync -av --exclude='build/' --exclude='.git/' . $(BACKUP_DIR)/
	@echo "Project backed up to $(BACKUP_DIR)"

# Update standard library package list
update-std-package-list:
	@echo "Updating standard library package list..."
	@go run -tags gen ./pkg/std/gen
	@echo "Standard library package list updated in pkg/std/packages.go"

# Version management helpers
list-tags:
	@echo "Current tags:"
	@git tag -l --sort=-version:refname | head -10

tag-patch:
	@echo "Creating patch version tag..."
	@CURRENT_TAG=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo $$CURRENT_TAG | sed 's/v\([0-9]*\)\.\([0-9]*\)\.\([0-9]*\).*/\1/'); \
	MINOR=$$(echo $$CURRENT_TAG | sed 's/v\([0-9]*\)\.\([0-9]*\)\.\([0-9]*\).*/\2/'); \
	PATCH=$$(echo $$CURRENT_TAG | sed 's/v\([0-9]*\)\.\([0-9]*\)\.\([0-9]*\).*/\3/'); \
	NEW_PATCH=$$((PATCH + 1)); \
	NEW_TAG="v$$MAJOR.$$MINOR.$$NEW_PATCH"; \
	echo "Current version: $$CURRENT_TAG"; \
	echo "New version: $$NEW_TAG"; \
	git tag $$NEW_TAG; \
	echo "Tagged as $$NEW_TAG"

tag-minor:
	@echo "Creating minor version tag..."
	@CURRENT_TAG=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo $$CURRENT_TAG | sed 's/v\([0-9]*\)\.\([0-9]*\)\.\([0-9]*\).*/\1/'); \
	MINOR=$$(echo $$CURRENT_TAG | sed 's/v\([0-9]*\)\.\([0-9]*\)\.\([0-9]*\).*/\2/'); \
	NEW_MINOR=$$((MINOR + 1)); \
	NEW_TAG="v$$MAJOR.$$NEW_MINOR.0"; \
	echo "Current version: $$CURRENT_TAG"; \
	echo "New version: $$NEW_TAG"; \
	git tag $$NEW_TAG; \
	echo "Tagged as $$NEW_TAG"

tag-major:
	@echo "Creating major version tag..."
	@CURRENT_TAG=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo $$CURRENT_TAG | sed 's/v\([0-9]*\)\.\([0-9]*\)\.\([0-9]*\).*/\1/'); \
	NEW_MAJOR=$$((MAJOR + 1)); \
	NEW_TAG="v$$NEW_MAJOR.0.0"; \
	echo "Current version: $$CURRENT_TAG"; \
	echo "New version: $$NEW_TAG"; \
	git tag $$NEW_TAG; \
	echo "Tagged as $$NEW_TAG"
