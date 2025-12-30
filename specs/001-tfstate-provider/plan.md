# Implementation Plan: Terraform Remote State Provider

**Branch**: `001-tfstate-provider` | **Date**: 2025-12-30 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `/specs/001-tfstate-provider/spec.md`

## Summary

The Terraform Remote State Provider enables Nomos configurations to consume Terraform/OpenTofu state outputs, allowing integration with existing infrastructure-as-code workflows. The provider implements the Nomos gRPC contract, supporting local filesystem and Azure Blob Storage backends with environment-variable-based authentication. It reads Terraform state format v4+ (Terraform 0.12+/OpenTofu 1.x+), exposes root module outputs via path-based access, and fetches fresh state on every request (no caching) to ensure data freshness.

**Technical Approach**: Go 1.25.5 implementation with domain-driven package structure, gRPC server with subprocess discovery pattern, Azure SDK with DefaultAzureCredential for environment-based auth, and stdlib JSON parsing for Terraform state. TDD approach with 80%+ coverage, following Nomos constitution standards.

## Technical Context

**Language/Version**: Go 1.25.5  
**Primary Dependencies**:
- `github.com/autonomous-bits/nomos/libs/provider-proto` (gRPC contract, generated stubs)
- `google.golang.org/grpc` (gRPC server implementation)
- `google.golang.org/protobuf` (protobuf type conversion)
- `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob` (Azure Blob Storage client)
- `github.com/Azure/azure-sdk-for-go/sdk/azidentity` (Azure authentication)

**Storage**: Read-only access to Terraform state files (local filesystem or Azure Blob Storage)  
**Testing**: Go test framework (stdlib), table-driven tests, integration tests tagged `//go:build integration`  
**Target Platform**: Linux (amd64, arm64), macOS (amd64/Intel, arm64/Apple Silicon), Windows (amd64)  
**Project Type**: Single binary (gRPC server subprocess)  
**Performance Goals**:
- Local backend: < 5 seconds for compilation (SC-001)
- Provider initialization: < 2 seconds (SC-008)
- Health checks: < 100ms response time

**Constraints**:
- No state caching (fetch fresh on every RPC call)
- Credentials ONLY from environment variables (never config)
### Phase 0: Pre-Research Check

All principles reviewed. No violations detected.

**Status**: ✅ PASSED

**Principle Alignment**:
- ✅ **I. gRPC Contract Compliance**: Will implement all 5 RPC methods (Init, Fetch, Info, Health, Shutdown) per nomos.provider.v1.ProviderService
- ✅ **II. Process Model & Discovery**: Will use subprocess with port discovery (print PROVIDER_PORT to stdout)
- ✅ **III. Test-Driven Development**: Plan includes TDD approach with 80%+ coverage requirement
- ✅ **IV. Idiomatic Go & Code Quality**: Using gofmt, goimports, golangci-lint; proper error handling planned
- ✅ **V. Context Propagation**: Context as first parameter for all I/O operations planned
- ✅ **VI. Security First**: Environment variables for credentials, input validation, no secrets in code
- ✅ **VII. Multi-Agent Coordination**: Following provider-orchestrator workflow with specialized agents

### Phase 1: Post-Design Check


```text
cmd/
  nomos-provider-terraform-remote-state/
    main.go                          # Entry point, gRPC server setup

internal/
  provider/
    provider.go                      # gRPC service implementation
    provider_test.go                 # Unit tests for provider
  backend/
    backend.go                       # Backend interface definition
    local.go                         # Local filesystem backend
    local_test.go                    # Local backend tests
    azurerm.go                       # Azure Blob Storage backend
    azurerm_test.go                  # Azure backend tests
  state/
    parser.go                        # State file parsing logic
    parser_test.go                   # Parser tests
    types.go                         # State file type definitions
  config/
    config.go                        # Configuration parsing/validation
    config_test.go                   # Config validation tests

go.mod                               # Go module definition
go.sum                               # Go module checksums
Makefile                             # Build, test, lint targets
.golangci.yml                        # golangci-lint configuration

README.md                            # Project documentation
CHANGELOG.md                         # Version history
CONTRIBUTING.md                      # Contribution guidelines
LICENSE                              # License file

.github/
  workflows/
    ci.yml                           # CI pipeline (test, lint, build)
    release.yml                      # Release workflow
  agents/
    provider-orchestrator.md         # Orchestrator agent (already exists)
    go-provider-architect.md         # Architecture agent (if exists)
    grpc-service-specialist.md       # gRPC agent (if exists)
    go-provider-implementer.md       # Implementation agent (if exists)
    go-provider-tester.md            # Testing agent (if exists)
    go-security-reviewer.md          # Security agent (if exists)
    documentation-specialist.md      # Documentation agent (if exists)
```

**Structure Decision**: Single Go project with domain-driven internal package structure.

**Rationale**:
- **cmd/**: Single binary entry point (provider subprocess)
- **internal/**: Private packages (cannot be imported by external projects)
  - **provider/**: gRPC service implementation (core logic)
  - **backend/**: Backend abstraction and implementations (local, azurerm)
  - **state/**: State file parsing and type definitions (domain logic)
  - **config/**: Configuration parsing and validation (input handling)
- **No separate tests/ directory**: Tests co-located with source files (`*_test.go` pattern)
- **Integration tests**: Tagged with `//go:build integration` in same files

This structure follows Go best practices:
1. Domain-driven organization (not type-driven)
2. Clear separation of concerns
3. Testable packages with clear interfaces
4. Single binary output (no unnecessary splitting) root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
# [REMOVE IF UNUSED] Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# [REMOVE IF UNUSED] Option 2: Web application (when "frontend" + "backend" detected)
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# [REMOVE IF UNUSED] Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure: feature modules, UI flows, platform tests]
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
