# Changelog

All notable changes to the Nomos Terraform Remote State Provider will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-12-31

### Changed
- **Project Structure**: Renamed command folder from `cmd/nomos-provider-terraform-remote-state/` to `cmd/provider/` for improved developer experience
  - Follows Go community conventions for concise command naming
  - Updated all documentation, build scripts, and CI/CD workflows
  - Binary name remains unchanged as `nomos-provider-terraform-remote-state`
  - Git history fully preserved

## [0.0.1] - 2025-12-31

Initial MVP release of the Nomos Terraform Remote State Provider.

### Added

#### Core Provider Features
- **gRPC Service Contract**: Full implementation of `nomos.provider.v1.ProviderService`
  - Init RPC: Initialize provider with backend configuration
  - Fetch RPC: Retrieve output values from state by path
  - Info RPC: Return provider metadata (type, version, alias)
  - Health RPC: Health status checking
  - Shutdown RPC: Graceful shutdown and resource cleanup
- **Provider Discovery**: Subprocess mode with TCP port discovery (`PROVIDER_PORT=<port>`)
- **Version Information**: Build-time version injection via ldflags
- **Structured Logging**: JSON-formatted logs to stderr for debugging and monitoring

#### Backend Support
- **Local Filesystem Backend**
  - Read state files from local filesystem paths
  - Workspace support with automatic path resolution (`terraform.tfstate.d/<workspace>/`)
  - Absolute and relative path support
  - Default workspace handling
- **Azure Blob Storage Backend**
  - Azure blob storage integration via Azure SDK
  - DefaultAzureCredential authentication (service principal, managed identity, Azure CLI)
  - Workspace handling via blob key paths
  - Support for `env:/` workspace prefix pattern
  - Custom blob key patterns for workspace organization
- **Backend Registry Pattern**: Extensible architecture for future backend types (S3, GCS, HTTP)

#### State File Parsing
- **Terraform State Format Support**: Version 4+ (Terraform 0.12+, OpenTofu 1.x+)
- **Streaming JSON Parser**: High-performance state file parsing with minimal memory overhead
- **State Validation**: Comprehensive validation of required fields and format version
- **Output Extraction**: Type-safe extraction of root module outputs
  - Primitives: string, number, boolean
  - Collections: list, map, set
  - Structural: object, tuple
  - Sensitivity flag support

#### Configuration Management
- **Type-Safe Configuration Parsing**: Structured configuration validation
- **Backend Type Validation**: Allowlist-based backend type validation
- **Required Field Validation**: Clear error messages for missing configuration
- **Configuration Schema**: Support for backend-specific configuration parameters

#### Workspace Features
- **Local Backend Workspaces**: Automatic `terraform.tfstate.d/<workspace>/` path resolution
- **Azure Backend Workspaces**: Key-based workspace handling (user-controlled)
- **Default Workspace Support**: Seamless handling of default workspace
- **Workspace Name Validation**: Security validation to prevent path traversal

#### Error Handling
- **gRPC Error Mapping**: Standard error codes with descriptive messages
  - `NotFound`: Missing outputs, files, blobs, workspaces
  - `InvalidArgument`: Configuration errors, validation failures
  - `FailedPrecondition`: Initialization errors, unsupported versions
  - `PermissionDenied`: Authentication failures
  - `Unavailable`: Network errors, timeouts
  - `Internal`: Parsing errors, unexpected errors
  - `Canceled`: Context cancellation
  - `DeadlineExceeded`: Operation timeouts
- **Context Propagation**: Full context cancellation and timeout support
- **Detailed Error Messages**: Clear, actionable error descriptions

### Security

#### Input Validation
- **Path Traversal Prevention**: Strict validation to prevent `../` attacks
- **Workspace Name Sanitization**: Alphanumeric, hyphens, underscores only
- **Character Allowlisting**: Regex-based validation for paths, keys, names
- **Length Limits**: Maximum lengths enforced to prevent DoS attacks
  - Paths: 1024 characters
  - Blob keys: 1024 characters
  - Workspace names: 100 characters
  - Storage account names: 3-24 characters (Azure limit)
  - Container names: 3-63 characters (Azure limit)
- **Control Character Filtering**: Null bytes and control characters rejected
- **Backend Type Allowlist**: Only explicitly approved backend types accepted

#### Authentication
- **Environment Variables Only**: No credentials in configuration files
- **Azure DefaultAzureCredential**: Industry-standard authentication chain
  - Service principal (AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET)
  - Managed identity
  - Azure CLI
  - Visual Studio Code
  - Azure PowerShell
- **No Credential Storage**: Credentials never stored in provider state

#### Defense in Depth
- **Input Sanitization**: Removal of control characters from all string inputs
- **Path Normalization**: Validation that cleaned paths match original
- **Backend-Specific Validation**: Type-specific validation (storage account format, container names, blob keys)

### Performance

#### Optimization
- **Streaming JSON Parser**: `json.Decoder` for large state files with minimal memory overhead
- **Benchmark Results**: Consistent ~55-60 MB/s throughput for 1KB-10MB state files
- **Zero State Caching**: Intentional design for guaranteed data freshness
- **Efficient gRPC**: Optimized message sizes (10MB max), keepalive parameters

#### Metrics and Observability
- **In-Memory Metrics**: Atomic counters for call tracking
  - Per-RPC call counts (Init, Fetch, Info, Health, Shutdown)
  - Per-RPC error counts
  - Per-RPC duration tracking (nanosecond precision)
- **Structured Logging**: Contextual logging with slog
  - Request tracing
  - Error details
  - Backend-specific context
- **Future-Ready Metrics**: Architecture prepared for Prometheus/OpenTelemetry migration

### Documentation

#### User Documentation
- **README.md**: Comprehensive project overview
  - Quick start guide
  - Installation instructions
  - Configuration examples
  - Architecture overview
  - Troubleshooting section
- **Backend Configuration Guide** (`docs/backend-configuration.md`)
  - Detailed local backend setup
  - Detailed Azure backend setup
  - Workspace patterns and resolution
  - Security considerations
  - Validation rules
- **Output Access Guide** (`docs/output-access.md`)
  - Output type examples (string, number, boolean, list, map, object)
  - Sensitive output handling
  - Missing output handling
  - Advanced patterns
- **Workspace Usage Guide** (`docs/workspace-usage.md`)
  - Workspace concepts and strategies
  - Local backend workspace resolution
  - Azure backend workspace patterns
  - Multi-workspace configurations
- **Error Handling Guide** (`docs/error-handling.md`)
  - gRPC error code reference
  - Common error scenarios and solutions
  - Debugging tips
  - FAQ

#### Developer Documentation
- **CONTRIBUTING.md**: Development guidelines
  - Setup instructions
  - Code standards
  - Testing requirements
  - Commit message format
  - Pull request process
- **AGENTS.md**: AI coding agent instructions
  - Project context
  - Essential commands
  - Architecture decisions
  - Critical patterns
- **Godoc Comments**: Comprehensive package and symbol documentation
  - Package-level documentation for all packages
  - Exported type documentation with examples
  - Function documentation with parameters, returns, and examples
  - Error documentation
  - Usage examples in comments

#### Specifications
- **Feature Spec** (`specs/001-tfstate-provider/spec.md`)
- **Implementation Plan** (`specs/001-tfstate-provider/plan.md`)
- **Quick Start Guide** (`specs/001-tfstate-provider/quickstart.md`)
- **Data Model** (`specs/001-tfstate-provider/data-model.md`)
- **Architecture Documents** (`docs/architecture/`)

### Testing

#### Test Coverage
- **80%+ Code Coverage**: Comprehensive test suite exceeding coverage requirements
- **Unit Tests**: Package-level unit tests for all core logic
  - Provider service tests
  - Backend tests (local, Azure)
  - State parser tests
  - Configuration validation tests
- **Integration Tests**: Tagged integration tests for backend connectivity
- **Table-Driven Tests**: Extensive scenario coverage with table-driven patterns
- **Race Detection**: All tests pass with `-race` flag
- **Benchmark Tests**: Performance benchmarks for state parsing

#### Test Organization
- **Co-Located Tests**: `*_test.go` files alongside source code
- **Build Tags**: Integration tests tagged with `//go:build integration`
- **Mock Interfaces**: Testable backend implementations

### Build and Tooling

#### Build System
- **Makefile**: Comprehensive build automation
  - `make build`: Compile provider binary
  - `make test`: Run unit tests
  - `make test-coverage`: Generate coverage reports
  - `make verify`: Complete validation (fmt + vet + lint + test)
  - `make deps`: Install dependencies
  - `make tidy`: Clean up go.mod
  - `make fmt`: Format code
  - `make lint`: Run golangci-lint
- **Multi-Platform Support**: Linux, macOS, Windows (amd64, arm64)
- **Version Injection**: ldflags-based version, commit, and build time

#### Code Quality
- **golangci-lint**: Comprehensive linting with multiple linters enabled
- **gofmt/goimports**: Consistent code formatting
- **go vet**: Static analysis
- **Continuous Integration**: Ready for CI/CD integration

### Limitations (MVP Scope)

The following features are intentionally deferred to future releases:

#### Phase 2 Features (Planned)
- **Nested Module Outputs**: Direct access to child module outputs
  - Current workaround: Expose module outputs at root level
- **Additional Backends**: AWS S3, Google Cloud Storage, HTTP, Terraform Cloud
- **State Caching**: Optional caching for performance (trade-off with freshness)

#### Out of Scope
- **State Writing**: Read-only provider by design
- **State Locking**: Not required for read-only operations
- **State Migration**: Terraform's responsibility
- **Resource Data Access**: Only outputs, not internal resource state

## Notes

### Breaking Changes from Previous Releases
- Initial release, no breaking changes

### Migration Guide
- No migration required (initial release)

### Deprecations
- None

### Known Issues
- None

### Compatibility
- **Terraform**: 0.12+ (state format version 4+)
- **OpenTofu**: 1.x+ (state format version 4+)
- **Go**: 1.25+
- **Nomos**: Compatible with provider-proto v1 contract

[Unreleased]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/compare/v0.0.1...v0.1.0
[0.0.1]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/releases/tag/v0.0.1
