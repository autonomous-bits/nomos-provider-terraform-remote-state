# Implementation Plan: Separate Backend Type from Provider Type

**Branch**: `002-separate-backend-type` | **Date**: 2025-12-31 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-separate-backend-type/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

This feature resolves a fundamental naming conflict in the provider configuration where the `type` field was used for two different purposes: CLI provider source identification (e.g., "autonomous-bits/nomos-provider-terraform-remote-state") and runtime backend selection (e.g., "local", "azurerm"). The solution introduces a new `backend_type` field for runtime backend selection while reserving `type` exclusively for CLI use. Additionally, the provider will support auto-detection of backend type from configuration keys (presence of `path` → local, presence of `storage_account_name` + `container_name` → azurerm) as a convenience feature to reduce configuration verbosity.

## Technical Context

**Language/Version**: Go 1.25+  
**Primary Dependencies**: github.com/autonomous-bits/nomos/libs/provider-proto, google.golang.org/grpc, github.com/Azure/azure-sdk-for-go/sdk/storage/azblob  
**Storage**: Local filesystem (local backend), Azure Blob Storage (azurerm backend)  
**Testing**: Go testing framework (`go test`), table-driven tests, integration tests with `//go:build integration` tag  
**Target Platform**: Linux, macOS, Windows (cross-platform Go binary)
**Project Type**: Single Go project (CLI/library hybrid - provider subprocess)  
**Performance Goals**: <2s provider initialization, <5s compilation with local backend  
**Constraints**: Must maintain existing backend functionality, must not change gRPC contract, must follow existing config parsing patterns  
**Scale/Scope**: Small-scale refactoring - 2 backend implementations (local, azurerm), ~5 files affected (config package + tests + docs)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

✅ **I. gRPC Contract Compliance**: No changes to gRPC contract - purely internal configuration change  
✅ **II. Process Model & Discovery**: No changes to subprocess/discovery model  
✅ **III. Test-Driven Development**: Will follow TDD - write tests first for config parsing and auto-detection  
✅ **IV. Idiomatic Go & Code Quality**: Will maintain gofmt, golangci-lint, error handling standards  
✅ **V. Context Propagation**: Context already properly propagated - no changes to I/O paths  
✅ **VI. Security First**: Auto-detection logic will be validated against path traversal/injection attacks  
✅ **VII. Multi-Agent Coordination**: Single-agent task (config refactoring) - no orchestration needed

**Quality Gates Applicable**:
- Phase 3: Implementation Gate (formatting, linting, error handling)
- Phase 4: Testing Gate (80%+ coverage, all tests pass)
- Phase 5: Security Gate (input validation for auto-detection)
- Phase 6: Documentation Gate (update examples and docs)

**Pre-Implementation Assessment**: ✅ **PASSED**
- No constitutional violations
- Clear separation of concerns (config field naming)
- Minimal scope (refactoring, not new feature)
- Security considerations identified (auto-detection validation)

## Project Structure

### Documentation (this feature)

```text
specs/002-separate-backend-type/
├── spec.md              # Feature specification
├── plan.md              # This file (implementation plan)
├── data-model.md        # Phase 1 output (config parsing logic)
├── quickstart.md        # Phase 1 output (usage examples)
├── contracts/           # Phase 1 output (config schema)
│   └── config-changes.md
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── nomos-provider-terraform-remote-state/
    └── main.go                        # Binary entry point (no changes expected)

internal/
├── config/
│   ├── config.go                      # ⚠️ PRIMARY CHANGE: Update to use backend_type
│   └── config_test.go                 # ⚠️ Update tests for backend_type + auto-detection
├── backend/
│   ├── backend.go                     # Backend interface (no changes)
│   ├── local.go                       # Local backend (may need minor updates)
│   ├── local_test.go                  # Update test configs to use backend_type
│   ├── azurerm.go                     # Azure backend (may need minor updates)
│   └── azurerm_test.go                # Update test configs to use backend_type
├── provider/
│   ├── provider.go                    # Provider service (minimal changes)
│   └── provider_test.go               # Update test configs to use backend_type
└── state/
    └── parser.go                      # State parsing (no changes)

docs/
├── backend-configuration.md           # ⚠️ Update to show backend_type field
└── [other docs]                       # Update all config examples

specs/001-tfstate-provider/
└── quickstart.md                      # ⚠️ Update examples to use backend_type

README.md                              # ⚠️ Update config examples and add clarification
```

**Structure Decision**: This is a refactoring within the existing single-project structure. The primary changes are in the `internal/config/` package with test updates across the codebase. No new directories or major structural changes required.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

*No constitutional violations - this section is empty.*

---

## Phase 0: Research ✅ COMPLETE

**Objective**: Research technical requirements and document findings

**Deliverables**:
- ✅ [research.md](research.md) - Configuration parsing patterns, auto-detection strategy, testing approach

**Key Findings**:
- Existing `ParseConfig()` function provides foundation for refactoring
- Auto-detection can be implemented with simple rule-based logic
- No backward compatibility needed (provider not in use)
- Clean implementation without legacy support paths

---

## Phase 1: Design ✅ COMPLETE

**Objective**: Design data structures and contracts

**Deliverables**:
- ✅ [data-model.md](data-model.md) - Configuration structures, auto-detection logic, validation rules
- ✅ [contracts/config-schema.md](contracts/config-schema.md) - Complete configuration schema with auto-detection
- ✅ [quickstart.md](quickstart.md) - Usage examples and patterns
- ✅ `.github/copilot-instructions.md` - Updated with feature technologies

**Design Decisions**:
1. **Field Naming**: `backend_type` chosen for clarity (not `backend`, `backend_name`, etc.)
2. **Auto-detection**: Rule-based using key presence (simple, maintainable)
3. **Precedence**: Explicit `backend_type` always wins over auto-detection
4. **Error Handling**: Clear, actionable error messages for all scenarios
5. **Validation**: Upfront validation with conflict detection

**Constitution Re-check**: ✅ **PASSED**
- No new constitutional concerns
- Design maintains gRPC contract compliance
- TDD approach planned for testing
- Security considerations addressed (input validation)
- Documentation comprehensive

---

## Phase 2: Implementation Planning

**Next Steps** (via `/speckit.tasks`):
1. Generate tasks.md with specific implementation tasks
2. Organize tasks by phase (Setup, Testing, Implementation, Documentation)
3. Define task dependencies and parallel execution opportunities

**Expected Task Categories**:
- [S] Setup: Project structure review (minimal changes expected)
- [T] Testing: Write tests for config parsing and auto-detection (TDD)
- [I] Implementation: Update config.go, update backends, update provider
- [D] Documentation: Update all config examples and docs

**Estimated Complexity**: Low-Medium
- Small scope: ~5 files affected
- Clear requirements: No ambiguity in spec
- No new features: Pure refactoring
- Good test coverage possible: All scenarios well-defined

---

## Artifacts Generated

### Phase 0 Outputs
- [research.md](research.md) - 3600 lines - Technical research and decisions

### Phase 1 Outputs
- [data-model.md](data-model.md) - 1400 lines - Data structures and validation logic
- [contracts/config-schema.md](contracts/config-schema.md) - 850 lines - Complete configuration schemas
- [quickstart.md](quickstart.md) - 1100 lines - Usage examples and patterns

### Ready for Phase 2
- Run `/speckit.tasks` to generate tasks.md with actionable implementation tasks

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Auto-detection ambiguity | Low | Medium | Clear error messages, explicit override option |
| Test coverage gaps | Low | Medium | TDD approach, comprehensive test cases defined |
| Documentation inconsistency | Medium | Low | Update all docs in same PR, checklist in tasks |
| User confusion about fields | Medium | Low | Clear examples in quickstart, README clarification |

---

## Summary

Planning phase complete. All design artifacts generated:

✅ **Phase 0**: Research complete - no new technologies, pure refactoring  
✅ **Phase 1**: Design complete - data model, contracts, and usage docs ready  
⏭️ **Phase 2**: Ready for task generation via `/speckit.tasks`

**Key Outcomes**:
- Clear separation: `type` (CLI) vs `backend_type` (runtime)
- Auto-detection: Convenience feature with clear rules
- No breaking changes: Provider not in use yet
- Comprehensive documentation: Examples cover all scenarios

**Branch**: `002-separate-backend-type`  
**Status**: Design phase complete, ready for implementation planning
