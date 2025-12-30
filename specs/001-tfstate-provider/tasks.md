---
description: "Task list for Terraform Remote State Provider implementation"
---

# Tasks: Terraform Remote State Provider

**Feature Branch**: `001-tfstate-provider`  
**Repository**: nomos-provider-terraform-remote-state  
**Input**: Design documents from `/specs/001-tfstate-provider/`  
**Prerequisites**: plan.md, spec.md, data-model.md, contracts/, research.md, quickstart.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [CategoryID] [P?] [Story?] Description with file path`

- **[CategoryID]**: Category prefix + sequence number (e.g., [S1], [T1], [I1], [G1])
  - **[S]** = Setup → delegates to `go-provider-architect`
  - **[A]** = Architecture → delegates to `go-provider-architect`
  - **[G]** = gRPC Service → delegates to `grpc-service-specialist`
  - **[T]** = Testing → delegates to `go-provider-tester`
  - **[I]** = Implementation → delegates to `go-provider-implementer`
  - **[V]** = Validation → delegates to multiple agents
  - **[R]** = Security Review → delegates to `go-security-reviewer`
  - **[D]** = Documentation → delegates to `documentation-specialist`
- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure required before any implementation

- [X] [S1] Initialize Go module at root with `go mod init github.com/autonomous-bits/nomos-provider-terraform-remote-state`
- [X] [S2] [P] Create directory structure: cmd/nomos-provider-terraform-remote-state/, internal/provider/, internal/backend/, internal/state/, internal/config/
- [X] [S3] [P] Add gRPC dependencies in go.mod: github.com/autonomous-bits/nomos/libs/provider-proto, google.golang.org/grpc, google.golang.org/protobuf
- [X] [S4] [P] Add Azure SDK dependencies in go.mod: github.com/Azure/azure-sdk-for-go/sdk/storage/azblob, github.com/Azure/azure-sdk-for-go/sdk/azidentity
- [X] [S5] [P] Create Makefile with targets: build, test, lint, clean
- [X] [S6] [P] Create .golangci.yml with linting configuration (gofmt, goimports, govet, golint)
- [X] [S7] [P] Create CHANGELOG.md with version 0.1.0 (MVP) section
- [X] [S8] [P] Create CONTRIBUTING.md with contribution guidelines

**Checkpoint**: Project structure and dependencies ready for implementation

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] [G1] Create main entry point in cmd/nomos-provider-terraform-remote-state/main.go with gRPC server setup, port discovery (print PROVIDER_PORT), and signal handling
- [X] [A1] [P] Define Backend interface in internal/backend/backend.go with FetchState(ctx context.Context) (*state.StateFile, error) method
- [X] [A2] [P] Define StateFile struct in internal/state/types.go with Version, TerraformVersion, Serial, Lineage, Outputs fields
- [X] [A3] [P] Define OutputValue struct in internal/state/types.go with Value, Type, Sensitive fields
- [X] [I1] [P] Create state file parser in internal/state/parser.go with ParseStateFile(data []byte) (*StateFile, error) function
- [X] [T1] [P] Create state file parser tests in internal/state/parser_test.go with test cases for valid state v4, invalid JSON, unsupported version < 4
- [X] [I2] Create config parser in internal/config/config.go with ParseConfig(configMap map[string]interface{}) (BackendConfig, error) function
- [X] [G2] Create gRPC provider service skeleton in internal/provider/provider.go with UnimplementedProviderServiceServer embedding, mutex, initialized flag, alias, backend fields
- [X] [G3] [P] Implement Info RPC in internal/provider/provider.go returning type="terraform-remote-state", version from build metadata
- [X] [G4] [P] Implement Health RPC in internal/provider/provider.go returning STATUS_OK when initialized, STATUS_UNKNOWN when not initialized
- [X] [G5] [P] Implement Shutdown RPC in internal/provider/provider.go to gracefully close backend connections

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Access Remote State Outputs (Priority: P1) 🎯 MVP

**Goal**: Enable Nomos configurations to fetch Terraform state outputs from local and Azure backends

**Independent Test**: Create local state file with outputs, configure provider with local backend, call Init and Fetch RPCs to retrieve specific output values

### Implementation for User Story 1

#### Local Backend (Root Outputs) - TDD Pattern

- [X] [I3] [P] [US1] Define LocalBackendConfig struct in internal/backend/local.go with Type, Path, Workspace fields
- [X] [I4] [P] [US1] Define AzureBackendConfig struct in internal/backend/azurerm.go with Type, StorageAccountName, ContainerName, Key, ResourceGroupName fields
- [X] [T2] [P] [US1] Create LocalBackend unit tests in internal/backend/local_test.go with test cases for valid state file, file not found, permission denied, workspace path resolution
- [X] [I5] [US1] Implement LocalBackend struct in internal/backend/local.go with FetchState method reading file from filesystem with context propagation
- [X] [I6] [US1] Add workspace path resolution logic in internal/backend/local.go for default workspace (use path as-is) and named workspaces (construct terraform.tfstate.d/<workspace>/<filename>)
- [X] [I7] [US1] Add file validation in LocalBackend constructor checking file exists and is readable, return FailedPrecondition if not found, PermissionDenied if not readable

#### Azure Backend (Root Outputs) - TDD Pattern

- [X] [T3] [P] [US1] Create AzureBackend unit tests in internal/backend/azurerm_test.go with test cases for valid config, invalid storage account name, invalid container name, empty key (use mocks for Azure SDK)
- [X] [I8] [US1] Implement AzureBackend struct in internal/backend/azurerm.go with azblob.Client, config fields, and FetchState method
- [X] [I9] [US1] Add Azure authentication in AzureBackend constructor using azidentity.NewDefaultAzureCredential reading AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET from environment
- [X] [I10] [US1] Implement blob download in AzureBackend.FetchState using DownloadStream, return NotFound for 404, PermissionDenied for 403, Unavailable for network errors
- [X] [I11] [US1] Add config validation in AzureBackend constructor for storage_account_name (3-24 chars, lowercase alphanumeric), container_name (3-63 chars, lowercase alphanumeric + hyphens), key (non-empty)
- [X] [I11a] [US1] Add Azure authentication error handling in AzureBackend.FetchState returning PermissionDenied gRPC code for auth failures (missing credentials, invalid credentials, expired tokens) with clear error messages

#### gRPC Contract Implementation

- [X] [G6] [US1] Implement Init RPC in internal/provider/provider.go parsing config.type, creating LocalBackend or AzureBackend, validating already initialized, storing alias and backend
- [X] [G7] [US1] Add backend factory logic in Init RPC switching on type field, return InvalidArgument for unsupported types, return backend construction errors with appropriate gRPC codes
- [X] [G8] [US1] Implement Fetch RPC in internal/provider/provider.go validating initialized state, validating path non-empty, calling backend.FetchState, parsing state with state.ParseStateFile
- [X] [G9] [US1] Add output resolution in Fetch RPC looking up path[0] in state.Outputs, return NotFound if output doesn't exist with clear message including output name
- [X] [G10] [US1] Add value conversion in Fetch RPC using structpb.NewValue to convert output.Value to protobuf Struct, handle conversion errors as Internal
- [X] [I12] [US1] Add state version validation in ParseStateFile checking version >= 4, return FailedPrecondition with clear message for versions < 4

#### Error Handling

- [X] [I13] [P] [US1] Add comprehensive error handling in internal/provider/provider.go for all gRPC codes: FailedPrecondition, InvalidArgument, NotFound, PermissionDenied, Unavailable, DeadlineExceeded, Canceled, Internal
- [X] [I14] [P] [US1] Add context cancellation checks in backend FetchState methods before I/O operations, return Canceled error if context is done
- [X] [T4] [P] [US1] Create error handling tests in internal/provider/provider_test.go for Init called twice, Fetch before Init, empty path, missing output, invalid JSON state

#### Integration Tests

- [ ] [T5] [US1] Create integration test in internal/provider/provider_integration_test.go with //go:build integration tag for local backend end-to-end flow
- [ ] [T6] [US1] Add test case in provider_integration_test.go: create temp state file, call Init with local config, call Fetch for root output, verify correct value returned
- [ ] [T7] [US1] Add test case in provider_integration_test.go: local backend with workspace, verify workspace path resolution works correctly
- [ ] [T8] [US1] Add test case in provider_integration_test.go: Azure backend (requires env vars or mocks), call Init with azurerm config, call Fetch for root output
- [ ] [T9] [US1] Add test case in provider_integration_test.go: test all supported output types (string, number, bool, list, map, object, null)
- [ ] [T10] [US1] Add test case in provider_integration_test.go: state file not found returns NotFound error
- [ ] [T11] [US1] Add test case in provider_integration_test.go: output not found returns NotFound error with output name in message
- [X] [T15] [P] [US1] Add test cases for corrupted state files in internal/state/parser_test.go testing truncated JSON, invalid structure, partial file content - all should return clear error messages

**Checkpoint**: At this point, User Story 1 should be fully functional - local and Azure backends work, root outputs accessible, all gRPC methods implemented ✅ **COMPLETE** (Integration tests deferred)

---

## Phase 4: User Story 2 - Multiple Backend Support (Priority: P2)

**Goal**: Establish extensibility pattern for adding new backend types (S3, GCS, etc.) in the future

**Independent Test**: Verify backend interface is properly abstracted, new backend can be added by implementing Backend interface without modifying provider core logic

### Implementation for User Story 2

- [X] [A4] [US2] Refactor backend factory in internal/provider/provider.go to use registry pattern with map[string]BackendConstructor
- [X] [A5] [US2] Create backend constructor type in internal/backend/backend.go: type BackendConstructor func(ctx context.Context, config map[string]interface{}) (Backend, error)
- [X] [I15] [US2] Create backend registration function in internal/backend/backend.go: RegisterBackend(name string, constructor BackendConstructor)
- [X] [I16] [US2] Update LocalBackend to self-register in init() function in internal/backend/local.go using RegisterBackend("local", NewLocalBackend)
- [X] [I17] [US2] Update AzureBackend to self-register in init() function in internal/backend/azurerm.go using RegisterBackend("azurerm", NewAzureBackend)
- [X] [I18] [US2] Add backend factory GetBackend function in internal/backend/backend.go looking up constructor in registry, return InvalidArgument if type not found
- [X] [I19] [US2] Update Init RPC in internal/provider/provider.go to use GetBackend factory instead of switch statement
- [X] [T12] [P] [US2] Create backend factory tests in internal/backend/backend_test.go verifying registration, lookup, error for unknown type
- [X] [D1] [P] [US2] Add documentation in internal/backend/README.md explaining how to add new backend types: implement Backend interface, add constructor, call RegisterBackend

**Checkpoint**: At this point, backend architecture is extensible - new backends can be added without modifying provider core

---

## Phase 5: User Story 3 - Workspace Selection (Priority: P3)

**Goal**: Support Terraform workspace selection to access state from specific environments (dev, staging, prod)

**Independent Test**: Create state files for multiple workspaces, configure provider with workspace parameter, verify correct workspace state is fetched

### Implementation for User Story 3

- [X] [I20] [US3] Add workspace parameter support in internal/config/config.go parsing workspace field from config, defaulting to "default" if omitted
- [X] [I21] [US3] Update LocalBackend in internal/backend/local.go to accept workspace parameter in constructor, store workspace field
- [X] [I22] [US3] Enhance workspace path resolution in LocalBackend.FetchState to handle workspace parameter: default workspace uses path as-is, named workspace constructs terraform.tfstate.d/<workspace>/terraform.tfstate
- [X] [I23] [US3] Add workspace validation in LocalBackend ensuring workspace path exists before attempting to read, return NotFound with workspace name in message if missing
- [X] [D2] [US3] Document Azure workspace handling in internal/backend/azurerm.go: workspace is embedded in key path (e.g., "env:/dev/terraform.tfstate"), provider treats key as opaque
- [X] [T13] [P] [US3] Add workspace tests in internal/backend/local_test.go for default workspace, named workspace path resolution, non-existent workspace
- [X] [T14] [US3] Create integration test in internal/provider/provider_integration_test.go for workspace selection: create multiple workspace state files, configure provider with specific workspace, verify correct state fetched

**Checkpoint**: All user stories should now be independently functional - workspaces supported for multi-environment scenarios

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories and production readiness

- [X] [I24] [P] Add structured logging throughout codebase using log/slog with context-aware loggers, log Init/Fetch/Shutdown operations at INFO level, errors at ERROR level
- [X] [I25] [P] Add metrics instrumentation in internal/provider/provider.go for RPC call counts, durations, error rates (prepare for future observability)
- [X] [I26] [P] Optimize state parsing performance in internal/state/parser.go using json.Decoder for streaming, benchmark for 10MB+ state files
- [X] [R1] [P] Add input sanitization for all config fields in internal/config/config.go to prevent injection attacks, validate path characters, blob key characters
- [X] [D3] [P] Create comprehensive README.md at repository root with installation instructions, quick start, configuration examples, troubleshooting guide
- [X] [D4] [P] Create user documentation in docs/ directory: backend-configuration.md, output-access.md, workspace-usage.md, error-handling.md
- [X] [D5] [P] Add code comments and godoc documentation for all exported types, functions, and interfaces following Go documentation conventions
- [X] [I27] [P] Create GitHub Actions CI workflow in .github/workflows/ci.yml: run tests, linting, build binaries for linux-amd64, darwin-amd64, darwin-arm64, windows-amd64
- [X] [I28] [P] Create GitHub Actions release workflow in .github/workflows/release.yml: build multi-platform binaries, create GitHub release with artifacts
- [X] [V1] [P] Run quickstart.md validation: follow all examples in quickstart.md, verify they work end-to-end with compiled provider binary
- [X] [R2] [P] Security audit: verify no credentials in logs, no secrets in error messages, validate all environment variable reads, check for path traversal vulnerabilities
- [X] [V2] [P] Performance testing: benchmark Init time < 2 seconds, Fetch time < 5 seconds for local backend, Health check < 100ms
- [X] [D6] [P] Update CHANGELOG.md with all features, bug fixes, breaking changes for version 0.1.0

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational phase completion
- **User Story 2 (Phase 4)**: Depends on User Story 1 completion (refactors backend architecture)
- **User Story 3 (Phase 5)**: Depends on User Story 1 completion (enhances existing backends)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1) - MVP**: Can start after Foundational (Phase 2)
  - [I3]-[I4]: Backend config structs (parallel)
  - [T2]: LocalBackend tests FIRST (TDD)
  - [I5]-[I7]: LocalBackend implementation
  - [T3]: AzureBackend tests FIRST (TDD)
  - [I8]-[I11]: AzureBackend implementation
  - [G6]-[G10], [I12]: gRPC implementation (sequential, builds on backends)
  - [I13]-[I14], [T4]: Error handling + tests (parallel)
  - [T5]-[T15]: Integration tests (parallel)

- **User Story 2 (P2) - Extensibility**: Depends on US1 backend implementations
  - [A4]-[A5]: Architecture definitions
  - [I15]-[I19]: Backend factory refactoring (sequential)
  - [T12], [D1]: Tests and documentation (parallel)

- **User Story 3 (P3) - Workspaces**: Depends on US1 backend implementations
  - [I20]-[I23]: Workspace support (sequential)
  - [D2]: Documentation
  - [T13]-[T14]: Workspace tests (parallel)

### Within Each User Story

**User Story 1 Flow (TDD Pattern)**:
1. Define config structs ([I3]-[I4]) - parallel
2. **Tests FIRST**: [T2] LocalBackend tests
3. LocalBackend implementation ([I5]-[I7])
4. **Tests FIRST**: [T3] AzureBackend tests
5. AzureBackend implementation ([I8]-[I11])
6. Implement gRPC contract ([G6]-[G10], [I12]) - sequential (depends on backends)
7. Error handling + tests ([I13]-[I14], [T4]) - parallel
8. Integration tests ([T5]-[T15]) - parallel (requires all above complete)

**User Story 2 Flow**:
1. Architecture ([A4]-[A5]) - sequential
2. Refactor to factory pattern ([I15]-[I19]) - mostly sequential
3. Tests and docs ([T12], [D1]) - parallel

**User Story 3 Flow**:
1. Add workspace support ([I20]-[I23]) - sequential
2. Documentation ([D2])
3. Tests ([T13]-[T14]) - parallel

### Parallel Opportunities

**Phase 1 (Setup)**: All tasks except [S1] can run in parallel after [S1] completes
- [S2]-[S8]: 7 parallel tasks (directory structure, dependencies, config files)

**Phase 2 (Foundational)**: Several parallel groups
- After [G1] completes:
  - [A1]-[A3], [I1], [T1]: Backend interface, state types, parser (5 parallel tasks)
  - [G3]-[G5]: Info/Health/Shutdown RPCs (3 parallel tasks)

**Phase 3 (User Story 1)**: Multiple parallel opportunities
- [I3]-[I4]: Config structs (2 parallel)
- [T2] and [T3]: Backend tests written in parallel (TDD)
- [I13]-[I14], [T4]: Error handling (3 parallel)
- [T5]-[T15]: Integration tests (8 parallel)

**Phase 4 (User Story 2)**:
- [T12], [D1]: Tests and docs (2 parallel)

**Phase 5 (User Story 3)**:
- [T13]-[T14]: Tests (2 parallel)

**Phase 6 (Polish)**: Most tasks can run in parallel
- [I24]-[I28], [R1]-[R2], [D3]-[D6], [V1]-[V2]: 13 tasks, nearly all parallel (only dependencies: CI must be set up before release workflow)

---

## Parallel Execution Examples

### Setup Phase (After [S1])
```bash
# All directory structure, dependencies, and config files
Parallel Group 1 → go-provider-architect:
- [S2]: Create directory structure
- [S3]: Add gRPC dependencies
- [S4]: Add Azure SDK dependencies
- [S5]: Create Makefile
- [S6]: Create .golangci.yml
- [S7]: Create CHANGELOG.md
- [S8]: Create CONTRIBUTING.md
```

### User Story 1: Backend Implementation (TDD Pattern)
```bash
# Config structs → go-provider-implementer
Parallel Group 1:
- [I3]: Define LocalBackendConfig struct
- [I4]: Define AzureBackendConfig struct

# Tests FIRST → go-provider-tester
Parallel Group 2:
- [T2]: LocalBackend unit tests
- [T3]: AzureBackend unit tests

# Then implementations → go-provider-implementer
Sequential:
- [I5]-[I7]: LocalBackend implementation
- [I8]-[I11]: AzureBackend implementation

# gRPC implementation → grpc-service-specialist + go-provider-implementer
Sequential:
- [G6]-[G10]: gRPC RPCs (grpc-service-specialist)
- [I12]: State version validation (go-provider-implementer)

# Error handling → go-provider-implementer + go-provider-tester
Parallel Group 3:
- [I13]: Comprehensive error handling
- [I14]: Context cancellation checks
- [T4]: Error handling tests

# Integration tests → go-provider-tester
Parallel Group 4 (after all implementations):
- [T5]: Integration test structure
- [T6]: Local backend E2E test
- [T7]: Workspace test
- [T8]: Azure backend test
- [T9]: Output types test
- [T10]: State not found test
- [T11]: Output not found test
```

### Polish Phase
```bash
# Nearly everything in parallel
Parallel Group 1:
- [I24]: Structured logging → go-provider-implementer
- [I25]: Metrics instrumentation → go-provider-implementer
- [I26]: Performance optimization → go-provider-implementer
- [R1]: Input sanitization → go-security-reviewer
- [D3]: README.md → documentation-specialist
- [D4]: User documentation → documentation-specialist
- [D5]: Code comments → documentation-specialist
- [I27]: CI workflow → go-provider-implementer
- [V1]: Quickstart validation → go-provider-tester
- [R2]: Security audit → go-security-reviewer
- [V2]: Performance testing → go-provider-tester
- [D6]: CHANGELOG update → documentation-specialist

# Then after [I27] completes:
- [I28]: Release workflow (depends on CI workflow) → go-provider-implementer
```

---

## Implementation Strategy

### MVP First (User Story 1 Only) - Recommended

1. **Complete Phase 1: Setup** ([S1]-[S8])
   - Initialize Go project, dependencies, directory structure
   - Delegated to: go-provider-architect
   - ~1-2 hours

2. **Complete Phase 2: Foundational** ([G1]-[G5], [A1]-[A3], [I1]-[I2], [T1])
   - gRPC server skeleton, interfaces, basic state parsing
   - Delegated to: grpc-service-specialist, go-provider-architect, go-provider-implementer, go-provider-tester
   - ~4-6 hours

3. **Complete Phase 3: User Story 1** ([I3]-[I11a], [T2]-[T15], [G6]-[G10])
   - Local backend ([I3], [T2], [I5]-[I7]): ~3-4 hours → go-provider-implementer + go-provider-tester
   - Azure backend ([I4], [T3], [I8]-[I11]): ~4-5 hours → go-provider-implementer + go-provider-tester
   - gRPC contract ([G6]-[G10], [I12]): ~3-4 hours → grpc-service-specialist + go-provider-implementer
   - Error handling ([I13]-[I14], [T4]): ~2 hours → go-provider-implementer + go-provider-tester
   - Integration tests ([T5]-[T15]): ~3-4 hours → go-provider-tester
   - Total: ~15-20 hours

4. **STOP and VALIDATE**
   - Run integration tests
   - Test with real Terraform state files (local + Azure)
   - Verify quickstart examples work
   - Deploy/demo if ready

**MVP Deliverable**: Fully functional provider supporting local and Azure backends with root output access

### Incremental Delivery

1. **Foundation** (Phases 1-2): Project ready, can compile
2. **MVP** (Phase 3): User Story 1 → Full local + Azure support
   - Test independently → Deploy/Demo
3. **Extensible** (Phase 4): User Story 2 → Easy to add new backends
   - Test backend factory → Deploy/Demo
4. **Multi-Environment** (Phase 5): User Story 3 → Workspace support
   - Test workspaces → Deploy/Demo
5. **Production-Ready** (Phase 6): Polish → Documentation, CI/CD, security
   - Full validation → Production release

### Agent-Based Parallel Execution Strategy

With provider-orchestrator managing specialized agents:

**Phase 1 (Setup)** - Delegated to `go-provider-architect`:
- Completes [S1], then parallelizes [S2]-[S8]

**Phase 2 (Foundational)** - Multi-agent coordination:
- `grpc-service-specialist`: [G1]-[G5] (gRPC server + RPC implementations)
- `go-provider-architect`: [A1]-[A3] (interfaces and types)
- `go-provider-implementer`: [I1]-[I2] (parsers)
- `go-provider-tester`: [T1] (parser tests)

**Phase 3 (User Story 1)** - TDD with parallel tracks:

*Track 1: Local Backend*
1. `go-provider-implementer`: [I3] (LocalBackendConfig struct)
2. `go-provider-tester`: [T2] (LocalBackend tests FIRST)
3. `go-provider-implementer`: [I5]-[I7] (LocalBackend implementation)

*Track 2: Azure Backend (parallel with Track 1)*
1. `go-provider-implementer`: [I4] (AzureBackendConfig struct)
2. `go-provider-tester`: [T3] (AzureBackend tests FIRST)
3. `go-provider-implementer`: [I8]-[I11] (AzureBackend implementation)

*Track 3: gRPC Contract (after backends)*
1. `grpc-service-specialist`: [G6]-[G10] (Init + Fetch RPCs)
2. `go-provider-implementer`: [I12] (State validation)

*Track 4: Error Handling (parallel)*
- `go-provider-implementer`: [I13]-[I14]
- `go-provider-tester`: [T4]

*Track 5: Integration Tests (after all)*
- `go-provider-tester`: [T5]-[T11] (parallel execution)

**Phase 4 (User Story 2)** - Sequential then parallel:
- `go-provider-architect`: [A4]-[A5] (architecture)
- `go-provider-implementer`: [I15]-[I19] (factory pattern)
- Parallel: `go-provider-tester` [T12] + `documentation-specialist` [D1]

**Phase 5 (User Story 3)** - Sequential then parallel:
- `go-provider-implementer`: [I20]-[I23] (workspace support)
- `documentation-specialist`: [D2]
- `go-provider-tester`: [T13]-[T14] (parallel tests)

**Phase 6 (Polish)** - Massive parallelization:
- `go-provider-implementer`: [I24]-[I28]
- `go-security-reviewer`: [R1]-[R2]
- `documentation-specialist`: [D3]-[D6]
- `go-provider-tester`: [V1]-[V2]

---

## Task Count Summary

### By Phase
- **Phase 1 (Setup)**: 8 tasks → [S1]-[S8]
- **Phase 2 (Foundational)**: 11 tasks → [G1]-[G5], [A1]-[A3], [I1]-[I2], [T1]
- **Phase 3 (User Story 1 - MVP)**: 29 tasks → [I3]-[I11a], [T2]-[T15], [G6]-[G10]
- **Phase 4 (User Story 2)**: 9 tasks → [A4]-[A5], [I15]-[I19], [T12], [D1]
- **Phase 5 (User Story 3)**: 7 tasks → [I20]-[I23], [D2], [T13]-[T14]
- **Phase 6 (Polish)**: 13 tasks → [I24]-[I28], [R1]-[R2], [D3]-[D6], [V1]-[V2]

**Total**: 77 tasks

### By Category (Agent Delegation)

- **[S] Setup** (→ go-provider-architect): 8 tasks
  - [S1]-[S8]
  
- **[A] Architecture** (→ go-provider-architect): 5 tasks
  - [A1]-[A5]
  
- **[G] gRPC Service** (→ grpc-service-specialist): 10 tasks
  - [G1]-[G10]
  
- **[T] Testing** (→ go-provider-tester): 15 tasks
  - [T1]-[T15]
  
- **[I] Implementation** (→ go-provider-implementer): 29 tasks
  - [I1]-[I29]
  
- **[V] Validation** (→ context-dependent): 2 tasks
  - [V1] → go-provider-tester
  - [V2] → go-provider-tester
  
- **[R] Security Review** (→ go-security-reviewer): 2 tasks
  - [R1]-[R2]
  
- **[D] Documentation** (→ documentation-specialist): 6 tasks
  - [D1]-[D6]

### By User Story

- **Setup + Foundational**: 19 tasks (no story label)
- **User Story 1 (P1) [US1]**: 29 tasks
- **User Story 2 (P2) [US2]**: 9 tasks
- **User Story 3 (P3) [US3]**: 7 tasks
- **Polish**: 13 tasks (no story label)

### Parallel Opportunities

- **Phase 1**: 7 parallel tasks (after [S1])
- **Phase 2**: ~8 parallel tasks in groups
- **Phase 3**: ~17 parallel tasks in groups
- **Phase 4**: 2 parallel tasks
- **Phase 5**: 2 parallel tasks
- **Phase 6**: 12 parallel tasks

**Total Parallel Opportunities**: ~46 tasks can run in parallel with proper coordination

---

## Suggested MVP Scope

**Minimum Viable Product = User Story 1 Only**

Complete:
- Phase 1: Setup (8 tasks: [S1]-[S8])
- Phase 2: Foundational (11 tasks: [G1]-[G5], [A1]-[A3], [I1]-[I2], [T1])
- Phase 3: User Story 1 (29 tasks: [I3]-[I11a], [T2]-[T15], [G6]-[G10])
- Selected polish tasks: [D3] (README), [I27] (CI), [V1] (quickstart validation)

**Total MVP**: 49 tasks

**Deliverable**: Provider binary that:
- ✅ Implements full gRPC contract (Init, Fetch, Info, Health, Shutdown)
- ✅ Supports local filesystem backend
- ✅ Supports Azure Blob Storage backend
- ✅ Reads root module outputs
- ✅ Handles all error cases gracefully
- ✅ Has comprehensive tests (unit + integration)
- ✅ Passes quickstart validation
- ✅ Has CI pipeline for automated testing

**Not included in MVP** (add incrementally):
- ❌ Backend extensibility pattern (US2) - nice to have
- ❌ Workspace selection (US3) - can be added later
- ❌ Full documentation suite
- ❌ Performance optimizations
- ❌ Metrics/observability

**Post-MVP Additions**:
- Add User Story 2 (9 tasks) for easier backend expansion
- Add User Story 3 (7 tasks) for workspace support
- Add remaining polish tasks (10 tasks) for production hardening

---

## TDD Verification

**Test-Before-Implementation Order Maintained**: ✅

### Phase 2 (Foundational)
- [T1] parser tests → [I1] parser implementation ✅

### Phase 3 (User Story 1)
- [T2] LocalBackend tests → [I5]-[I7] LocalBackend implementation ✅
- [T3] AzureBackend tests → [I8]-[I11] AzureBackend implementation ✅
- [T4] error handling tests run parallel with [I13]-[I14] implementation ✅
- [T5]-[T15] integration tests run after all implementations complete ✓

### Phase 4 (User Story 2)
- [T12] backend factory tests run after [I15]-[I19] implementation (acceptable for refactoring) ⚠️

### Phase 5 (User Story 3)
- [T13]-[T14] workspace tests run after [I20]-[I23] implementation (acceptable for enhancement) ⚠️

**Overall**: TDD pattern strictly enforced for new components (Phase 2-3), relaxed for refactoring/enhancements (Phase 4-5)

---

## Example Task Format

### Proper Format Examples

```markdown
✅ CORRECT:
- [ ] [S1] Initialize Go module at root...
- [ ] [S2] [P] Create directory structure...
- [ ] [T2] [P] [US1] Create LocalBackend unit tests...
- [ ] [I5] [US1] Implement LocalBackend struct...
- [ ] [G6] [US1] Implement Init RPC...
- [ ] [V1] [P] Run quickstart.md validation...
- [ ] [R1] [P] Add input sanitization...
- [ ] [D3] [P] Create comprehensive README.md...
```

### Agent Routing Examples

```markdown
[S5] [P] Create Makefile with targets...
→ Routed to: go-provider-architect

[G6] [US1] Implement Init RPC in internal/provider/provider.go...
→ Routed to: grpc-service-specialist

[T2] [P] [US1] Create LocalBackend unit tests...
→ Routed to: go-provider-tester

[I5] [US1] Implement LocalBackend struct...
→ Routed to: go-provider-implementer

[R1] [P] Add input sanitization for all config fields...
→ Routed to: go-security-reviewer

[D3] [P] Create comprehensive README.md...
→ Routed to: documentation-specialist

[V1] [P] Run quickstart.md validation...
→ Routed to: go-provider-tester (validation context)
```

---

## Notes

- **Category Prefixes**: Enable automatic agent delegation by provider-orchestrator
  - [S] → go-provider-architect
  - [A] → go-provider-architect
  - [G] → grpc-service-specialist
  - [T] → go-provider-tester
  - [I] → go-provider-implementer
  - [V] → context-dependent (usually go-provider-tester)
  - [R] → go-security-reviewer
  - [D] → documentation-specialist
- **[P] tasks**: Different files, no dependencies, can be executed in parallel
- **[Story] labels**: Map task to specific user story for traceability and independent testing
- **TDD Pattern**: Test tasks ([T]) before implementation tasks ([I]) for the same component
- **gRPC Separation**: [G] prefix for gRPC-specific work (not [I])
- **Sequential IDs**: Within each category: [S1], [S2]... [T1], [T2]... [I1], [I2]...
- **File paths**: All paths are relative to repository root, included in task descriptions
- **Go conventions**: Follow stdlib patterns, idiomatic Go, proper error handling with context propagation
- **Testing**: Unit tests co-located with source (*_test.go), integration tests with build tags
- **Security**: All credentials from environment variables, never from config or logs
- **Performance**: Local backend < 5s, Health checks < 100ms, Init < 2s
- **State format**: Support Terraform 0.12+ / OpenTofu 1.x+ (state format v4+)
- **No caching**: Fetch fresh state on every RPC call per requirements

**Commit Strategy**:
- Commit after each task or logical group of parallel tasks
- Tag MVP completion for v0.1.0 release
- Tag subsequent user stories for minor version increments

**Validation Checkpoints**:
- After Phase 2: Can compile, gRPC server starts and prints PROVIDER_PORT
- After Phase 3: Full MVP works end-to-end with real state files
- After Phase 4: New backend can be added easily
- After Phase 5: Workspace selection works correctly
- After Phase 6: Production-ready with docs and CI/CD
