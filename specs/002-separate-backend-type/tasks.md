---
description: "Implementation tasks for separating backend type from provider type"
---

# Tasks: Separate Backend Type from Provider Type

**Input**: Design documents from `/specs/002-separate-backend-type/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Following TDD principles per constitution - tests are written FIRST before implementation.

**Organization**: Tasks are organized by agent responsibility for multi-agent orchestration workflow.

## Format: `[Type##] [P?] Description`

**Task Type Prefixes** (maps to specialized agents):
- **[S]** - Setup → `go-provider-architect`
- **[A]** - Architecture/Design → `go-provider-architect`
- **[T]** - Testing → `go-provider-tester`
- **[I]** - Implementation → `go-provider-implementer`
- **[V]** - Validation → Multiple agents (context-dependent)
- **[R]** - Security Review → `go-security-reviewer`
- **[D]** - Documentation → `documentation-specialist`

**Markers**:
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions

Single Go project structure:
- `internal/config/` - Configuration parsing
- `internal/backend/` - Backend implementations
- `internal/provider/` - Provider service
- `docs/` - Documentation
- `README.md` - Main project documentation

---

## Phase 1: Setup & Architecture Planning

**Purpose**: Project verification and implementation planning  
**Delegate to**: `go-provider-architect`

- [X] [S1] Verify Go module dependencies are up to date with `go mod tidy`
- [X] [S2] [P] Verify existing test infrastructure runs with `make test`
- [X] [S3] [P] Review current configuration parsing in internal/config/config.go to understand baseline implementation
- [X] [A1] Design error types needed: ErrAmbiguousBackendConfig, ErrCannotDetectBackend, ErrBackendConfigMismatch, ErrUnsupportedBackendType
- [X] [A2] Design detectBackendType() function logic: rule-based detection (path → local, storage_account_name + container_name → azurerm)
- [X] [A3] Design validateBackendType() function logic: check against allowlist ["local", "azurerm"]
- [X] [A4] Design validateBackendConfigMatch() function logic: detect conflicts between explicit backend_type and config keys
- [X] [A5] Design ParseConfig() update strategy: extract backend_type instead of type, integrate auto-detection

**Validation Criteria**:
- [X] All design decisions documented
- [X] Implementation approach clear and unambiguous
- [X] Security considerations identified (path validation, input sanitization)

---

## Phase 2: Comprehensive Testing (TDD)

**Purpose**: Write all tests FIRST before implementation  
**Delegate to**: `go-provider-tester`

**⚠️ CRITICAL**: Following TDD principles - all tests written before implementation, tests must FAIL initially

### Unit Tests - Explicit Backend Type (User Story 1)

- [X] [T1] [P] Add unit test in internal/config/config_test.go for explicit backend_type: "local" with path field → expects Type() = "local"
- [X] [T2] [P] Add unit test in internal/config/config_test.go for explicit backend_type: "azurerm" with Azure keys → expects Type() = "azurerm"
- [X] [T3] [P] Add unit test in internal/config/config_test.go for unsupported backend_type value → expects ErrUnsupportedBackendType
- [X] [T4] [P] Add unit test in internal/config/config_test.go for backend_type: "local" with conflicting Azure keys → expects ErrBackendConfigMismatch
- [X] [T5] [P] Add unit test in internal/config/config_test.go for backend_type: "azurerm" with conflicting path key → expects ErrBackendConfigMismatch
- [X] [T6] [P] Add unit test in internal/config/config_test.go verifying type field (if present) is silently ignored without error (provider should not validate or process it)

### Unit Tests - Auto-detection (User Story 2)

- [X] [T7] [P] Add unit test in internal/config/config_test.go for config with only path field → expects auto-detect Type() = "local"
- [X] [T8] [P] Add unit test in internal/config/config_test.go for config with only Azure keys (storage_account_name + container_name) → expects auto-detect Type() = "azurerm"
- [X] [T9] [P] Add unit test in internal/config/config_test.go for config with both path and Azure keys → expects ErrAmbiguousBackendConfig
- [X] [T10] [P] Add unit test in internal/config/config_test.go for config with neither backend_type nor recognizable keys → expects ErrCannotDetectBackend
- [X] [T11] [P] Add unit test in internal/config/config_test.go for config with partial Azure keys (only storage_account_name) → expects ErrCannotDetectBackend
- [X] [T11a] [P] Add unit test in internal/config/config_test.go for config with partial Azure keys (only container_name) → expects ErrCannotDetectBackend
- [X] [T12] [P] Add unit test in internal/config/config_test.go for precedence: explicit backend_type overrides auto-detection

### Integration Tests

- [X] [T13] [P] Update test configurations in internal/backend/local_test.go to use backend_type: "local" (replace type field references)
- [X] [T14] [P] Update test configurations in internal/backend/azurerm_test.go to use backend_type: "azurerm" (replace type field references)
- [X] [T15] [P] Update test configurations in internal/provider/provider_test.go to use backend_type field
- [X] [T16] [P] Update test configurations in internal/provider/provider_grpc_test.go to use backend_type field
- [X] [T17] [P] Add integration test in internal/provider/provider_integration_test.go for Init RPC with explicit backend_type: "local"
- [X] [T18] [P] Add integration test in internal/provider/provider_integration_test.go for Init RPC with auto-detected local backend (only path field)
- [X] [T19] [P] Add integration test in internal/provider/provider_integration_test.go for Init RPC with explicit backend_type: "azurerm"
- [X] [T20] [P] Add integration test in internal/provider/provider_integration_test.go for Init RPC with auto-detected azurerm backend (only Azure keys)
- [X] [T21] Update quickstart validation test in internal/provider/quickstart_validation_test.go to use backend_type field in test configurations
- [X] [T22] Run all tests with `go test ./internal/config/` and verify they FAIL appropriately (no implementation yet)

**Validation Criteria**:
- [X] All test cases written following table-driven pattern
- [X] Tests compile successfully (existing tests), config tests fail compilation as expected
- [X] Tests fail appropriately (red phase of TDD)
- [X] Test coverage plan targets 80%+ overall, 100% for critical paths

---

## Phase 3: Core Implementation

**Purpose**: Implement configuration parsing changes  
**Delegate to**: `go-provider-implementer`

### Foundational Implementation

- [X] [I1] Define error types in internal/config/config.go: ErrAmbiguousBackendConfig, ErrCannotDetectBackend, ErrBackendConfigMismatch, ErrUnsupportedBackendType with clear error messages
- [X] [I2] Implement detectBackendType() function in internal/config/config.go with rule-based detection logic
- [X] [I3] Implement validateBackendType() function in internal/config/config.go to check against allowlist ["local", "azurerm"]
- [X] [I4] Implement validateBackendConfigMatch() function in internal/config/config.go to detect conflicts between explicit backend_type and config keys

### ParseConfig() Updates

- [X] [I5] Update ParseConfig() function in internal/config/config.go to extract backend_type field instead of type field
- [X] [I6] Update ParseConfig() in internal/config/config.go to call detectBackendType() when backend_type is not explicitly provided
- [X] [I7] Update ParseConfig() in internal/config/config.go to call validateBackendType() on the determined backend type
- [X] [I8] Update ParseConfig() in internal/config/config.go to call validateBackendConfigMatch() when backend_type is explicit
- [X] [I9] Update ParseConfig() in internal/config/config.go to ensure type field is not extracted or used anywhere (ignore completely)
- [X] [I10] Add comprehensive error messages to all error paths with actionable guidance for users

### Test Validation

- [X] [I11] Run unit tests with `go test ./internal/config/` and verify all tests pass (green phase of TDD)
- [X] [I12] Run full test suite with `make test` and verify all tests pass
- [X] [I13] Run integration tests with `go test -tags=integration ./...` and verify all pass

**Validation Criteria**:
- [X] Code formatted with gofmt/goimports
- [X] golangci-lint passes with no warnings
- [X] All tests pass (TDD green phase achieved)
- [X] No panics in error paths
- [X] Context properly propagated (if applicable)

---

## Phase 4: Security Review

**Purpose**: Security validation and hardening  
**Delegate to**: `go-security-reviewer`

- [X] [R1] Review input validation: Verify backend_type field is properly sanitized and validated against allowlist
- [X] [R2] Review path validation: Verify path and workspace fields prevent path traversal attacks
- [X] [R3] Review auto-detection logic: Verify it cannot be exploited with malicious configuration keys
- [X] [R4] Review error messages: Ensure they don't expose internal system details or sensitive information
- [X] [R5] Run govulncheck: `govulncheck ./...` to scan for dependency vulnerabilities
- [X] [R6] Review configuration parsing: Verify all user inputs are validated at boundaries

**Validation Criteria**:
- [X] All inputs validated at boundaries
- [X] Path traversal protection verified
- [X] govulncheck passes with no critical issues
- [X] Error messages don't expose internals
- [X] Auto-detection logic cannot be exploited

---

## Phase 5: Documentation

**Purpose**: Update all documentation to reflect changes  
**Delegate to**: `documentation-specialist`

- [X] [D1] [P] Update backend configuration examples in docs/backend-configuration.md to use backend_type field
- [X] [D2] [P] Add auto-detection examples to docs/backend-configuration.md showing omitted backend_type with clear explanation
- [X] [D3] [P] Update error handling documentation in docs/error-handling.md to include new error types: ErrAmbiguousBackendConfig, ErrCannotDetectBackend, ErrBackendConfigMismatch
- [X] [D4] [P] Update configuration examples in specs/001-tfstate-provider/quickstart.md to use backend_type field
- [X] [D5] [P] Update README.md quick start examples to use backend_type field
- [X] [D6] [P] Add clarification section to README.md explaining difference between type (CLI provider source) and backend_type (runtime backend selection)
- [X] [D7] Update godoc comments in internal/config/config.go to document backend_type field and auto-detection behavior with clear examples
- [X] [D8] Review all documentation files for consistency and completeness across docs/, README.md, and specs/

**Validation Criteria**:
- [X] All exported symbols have godoc comments
- [X] README.md updated with clear examples
- [X] All configuration examples use backend_type
- [X] Auto-detection feature clearly documented
- [X] Migration guide provided (if applicable)

---

## Phase 6: Quality Validation & Final Checks

**Purpose**: Comprehensive quality gates and final validation  
**Delegate to**: Multiple agents (context-dependent)

### Code Quality

- [X] [V1] Run `gofmt -s -w .` to format all Go files
- [X] [V2] Run `go vet ./...` to check for common mistakes
- [X] [V3] Run `golangci-lint run` to check linting rules (must pass with zero warnings)

### Testing & Coverage

- [X] [V4] Run test coverage with `go test -cover ./...` and verify ≥80% coverage
- [X] [V5] Run race detector with `go test -race ./...` and verify no race conditions
- [X] [V6] Run benchmarks (if applicable) with `go test -bench=. ./...`

### Build & Integration

- [X] [V7] Build provider binary with `make build` and verify successful compilation
- [X] [V8] Manual verification: Test provider with local backend using explicit backend_type
- [X] [V9] Manual verification: Test provider with local backend using auto-detection
- [X] [V10] Manual verification: Test provider with Azure backend using explicit backend_type
- [X] [V11] Manual verification: Test provider with Azure backend using auto-detection

### Final Validation

- [X] [V12] Run quickstart validation from specs/002-separate-backend-type/quickstart.md
- [X] [V13] Verify all existing tests from feature 001-tfstate-provider still pass (backward compatibility check per SC-003)
- [X] [V14] Review constitution checklist from specs/002-separate-backend-type/checklists/requirements.md (if exists)
- [X] [V15] Verify all phase validation criteria passed

**Validation Criteria**:
- [X] Code quality gate passed (fmt, vet, lint)
- [X] Testing gate passed (80%+ coverage, all tests pass)
- [X] Security gate passed (govulncheck, input validation)
- [X] Documentation gate passed (all exports documented)
- [X] Manual testing completed successfully

---

## Agent Delegation Map

This table shows which specialized agent handles each task type:

| Task Type | Agent | Responsibilities |
|-----------|-------|------------------|
| [S] Setup | `go-provider-architect` | Project setup, dependency verification, baseline review |
| [A] Architecture | `go-provider-architect` | Design decisions, function planning, error type design |
| [T] Testing | `go-provider-tester` | Unit tests, integration tests, table-driven tests, TDD |
| [I] Implementation | `go-provider-implementer` | Core code implementation, ParseConfig updates, error handling |
| [V] Validation | Multiple | Quality checks, coverage analysis, manual testing |
| [R] Security | `go-security-reviewer` | Input validation, vulnerability scanning, security hardening |
| [D] Documentation | `documentation-specialist` | godoc, README, examples, consistency review |

---

## Execution Strategy

### Phase-by-Phase Approach

**Sequential Phase Execution** (complete each before moving to next):
1. **Phase 1: Setup & Architecture** → Establishes design foundation
2. **Phase 2: Testing (TDD)** → Write tests FIRST (must fail)
3. **Phase 3: Implementation** → Implement to pass tests (TDD green)
4. **Phase 4: Security Review** → Validate security posture
5. **Phase 5: Documentation** → Update all docs
6. **Phase 6: Quality Validation** → Final quality gates

### Parallel Task Execution

Within each phase, tasks marked **[P]** can run in parallel:

**Example: Phase 2 Testing** - All unit tests can be written simultaneously:
```bash
[T1] → Test explicit backend_type: "local"
[T2] → Test explicit backend_type: "azurerm"  
[T3] → Test unsupported backend_type
[T4] → Test conflicting config
# ... all can run in parallel (different test cases)
```

**Example: Phase 5 Documentation** - All doc updates can run simultaneously:
```bash
[D1] → Update docs/backend-configuration.md
[D2] → Add auto-detection examples
[D3] → Update docs/error-handling.md
[D4] → Update specs/001-tfstate-provider/quickstart.md
[D5] → Update README.md examples
[D6] → Add type vs backend_type clarification
# ... all can run in parallel (different files)
```

### TDD Workflow (Phase 2 → Phase 3)

Following Test-Driven Development principles:

```
Phase 2: Testing
├─ Write all tests (T1-T22)
├─ Verify tests compile
├─ Run tests → EXPECT FAILURE (RED)
└─ Checkpoint: Ready for implementation

Phase 3: Implementation
├─ Implement code (I1-I13)
├─ Run tests → EXPECT SUCCESS (GREEN)
└─ Checkpoint: TDD cycle complete
```

---

## Dependencies & Critical Path

### Phase Dependencies

```
Setup & Architecture (Phase 1)
         ↓
    Testing (Phase 2) ← TDD: Write tests FIRST
         ↓
  Implementation (Phase 3) ← TDD: Make tests pass
         ↓
  Security Review (Phase 4) ← Validate security
         ↓
  Documentation (Phase 5) ← Update docs
         ↓
Quality Validation (Phase 6) ← Final gates
```

### Task Dependencies Within Phases

**Phase 1: Setup & Architecture**
- [S1-S3] can run in parallel
- [A1-A5] sequential (design decisions build on each other)

**Phase 2: Testing**
- All [T1-T12] unit tests can run in parallel (different test cases)
- All [T13-T21] integration test updates can run in parallel (different files)
- [T22] must run last (validation step)

**Phase 3: Implementation**
- [I1-I4] foundational functions first (other code depends on these)
- [I5-I10] ParseConfig updates (depends on I1-I4)
- [I11-I13] test validation (depends on I5-I10)

**Phase 4: Security Review**
- All [R1-R6] can run in parallel (independent security checks)

**Phase 5: Documentation**
- All [D1-D7] can run in parallel (different files)
- [D8] must run last (consistency review)

**Phase 6: Quality Validation**
- [V1-V3] code quality checks run first
- [V4-V6] testing & coverage checks next
- [V7-V11] build & manual testing next
- [V12-V14] final validation last

---

## Success Criteria & Validation Gates

### Phase 1: Setup & Architecture
✅ **Ready to proceed when**:
- [X] Go dependencies verified current
- [X] Baseline code reviewed and understood
- [X] All design decisions documented
- [X] Error types designed
- [X] Implementation strategy clear

### Phase 2: Testing (TDD Red)
✅ **Ready to proceed when**:
- [X] All test cases written following table-driven pattern
- [X] All tests compile successfully (provider/integration tests compile; config tests fail as expected)
- [X] All tests FAIL appropriately (no implementation yet)
- [X] Test coverage plan targets 80%+ overall

### Phase 3: Implementation (TDD Green)
✅ **Ready to proceed when**:
- [X] All foundational functions implemented
- [X] ParseConfig() updated to use backend_type
- [X] All unit tests PASS
- [X] All integration tests PASS
- [X] Code formatted (gofmt/goimports)
- [X] golangci-lint passes

### Phase 4: Security Review
✅ **Ready to proceed when**:
- [X] All inputs validated at boundaries
- [X] Path traversal protection verified
- [X] govulncheck passes with no critical issues
- [X] Error messages don't expose internals
- [X] Auto-detection logic cannot be exploited

### Phase 5: Documentation
✅ **Ready to proceed when**:
- [X] All exported symbols have godoc comments
- [X] README.md updated with examples
- [X] All configuration examples use backend_type
- [X] Auto-detection feature documented
- [X] Type vs backend_type clarification added

### Phase 6: Quality Validation
✅ **Complete when**:
- [X] Code quality gate passed (fmt, vet, lint)
- [X] Testing gate passed (≥80% coverage, all tests pass)
- [X] Security gate passed (govulncheck, validation)
- [X] Documentation gate passed (all exports documented)
- [X] Manual testing completed successfully
- [X] All constitution requirements met

---

## Task Summary

**Total Tasks**: 61 tasks across 6 phases

| Phase | Task Count | Agent | Parallel Tasks |
|-------|-----------|-------|----------------|
| 1. Setup & Architecture | 8 | `go-provider-architect` | 2 |
| 2. Testing (TDD) | 23 | `go-provider-tester` | 19 |
| 3. Implementation | 13 | `go-provider-implementer` | 0 |
| 4. Security Review | 6 | `go-security-reviewer` | 6 |
| 5. Documentation | 8 | `documentation-specialist` | 6 |
| 6. Quality Validation | 15 | Multiple | 9 |

**Parallel Opportunities**: 42 tasks marked [P] can run in parallel within their phases

**Estimated Complexity**: Low-Medium
- Clear requirements with comprehensive design artifacts
- Small scope: ~5 files affected (config, backends, tests, docs)
- No new features: Pure refactoring
- Good test coverage achievable (all scenarios well-defined)
- No backward compatibility concerns (provider not in use)

**Critical Path** (sequential milestones):
```
Setup → Architecture → Testing (TDD Red) → Implementation (TDD Green) → Security → Documentation → Validation
```

**MVP Scope** (minimum viable delivery):
- Phase 1: Setup & Architecture (8 tasks)
- Phase 2: Testing - User Story 1 tests (6 tasks)
- Phase 3: Implementation - Explicit backend_type support (11 tasks)
- **Total MVP**: 25 tasks
- **Delivers**: Explicit backend_type support, eliminates type field conflict

---

## Notes

- **[P] marker**: Tasks can run in parallel (different files, no dependencies)
- **TDD approach**: All tests written before implementation (per constitution)
- **Agent specialization**: Each task type maps to specialized agent
- **Phase completion**: Validate all criteria before proceeding to next phase
- **Failure handling**: If validation fails, remediate through appropriate agent before continuing
- **Outcome format**: All agents report status, issues, validation results, and next steps
- **Constitution gates**: Testing (80%+ coverage), Security (input validation), Documentation (comprehensive)
