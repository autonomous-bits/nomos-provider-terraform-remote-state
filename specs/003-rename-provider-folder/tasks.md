# Tasks: Rename Provider Command Folder

**Input**: Design documents from `/specs/003-rename-provider-folder/`  
**Prerequisites**: plan.md, spec.md, contracts/file-updates.md, research.md, data-model.md, quickstart.md

**Tests**: Not applicable - this is a pure refactoring with no new functionality. Existing tests validate correctness.

**Organization**: Tasks are grouped by user story (priority) to enable independent verification of each aspect of the rename.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Verification Baseline)

**Purpose**: Ensure starting from a clean, working state before any changes

- [ ] T001 Verify on correct branch (003-rename-provider-folder) and clean working directory
- [ ] T002 Run baseline validation: make clean && make build && make test && make verify
- [ ] T003 Document current state: binary exists at bin/nomos-provider-terraform-remote-state

---

## Phase 2: User Story 1 - Developer Builds Provider (Priority: P1) 🎯 MVP

**Goal**: Rename folder to cmd/provider and update build system so developers can successfully build the provider

**Independent Test**: After this phase, `make build` succeeds and creates binary at bin/nomos-provider-terraform-remote-state

### Core Rename for User Story 1

- [ ] T004 [US1] Rename folder using git mv: cmd/nomos-provider-terraform-remote-state → cmd/provider
- [ ] T005 [US1] Verify git status shows tracked rename (not delete+add)
- [ ] T006 [US1] Add CMD_PATH variable in Makefile after BINARY_NAME line
- [ ] T007 [US1] Update build target in Makefile to use $(CMD_PATH) instead of ./cmd/$(BINARY_NAME)
- [ ] T008 [US1] Update install target in Makefile to use $(CMD_PATH) instead of ./cmd/$(BINARY_NAME)

### Validation for User Story 1

- [ ] T009 [US1] Run make clean
- [ ] T010 [US1] Run make build (MUST succeed, binary at bin/nomos-provider-terraform-remote-state)
- [ ] T011 [US1] Run make install (MUST succeed)
- [ ] T012 [US1] Verify binary name unchanged: ls bin/ | grep nomos-provider-terraform-remote-state
- [ ] T013 [US1] Run binary to verify functionality: ./bin/nomos-provider-terraform-remote-state (should print PROVIDER_PORT)

**Checkpoint**: Build system works with new folder structure. Developer can build provider from cmd/provider.

---

## Phase 3: User Story 2 - Developer Navigates Codebase (Priority: P2)

**Goal**: Update CI/CD and primary documentation so developers can find code easily and workflows succeed

**Independent Test**: After this phase, CI/CD workflows reference correct path and README is accurate

### CI/CD Updates for User Story 2

- [ ] T014 [P] [US2] Update .github/workflows/ci.yml line ~107: change ./cmd/nomos-provider-terraform-remote-state to ./cmd/provider
- [ ] T015 [P] [US2] Update .github/workflows/release.yml line ~56: change ./cmd/nomos-provider-terraform-remote-state to ./cmd/provider

### Documentation Updates for User Story 2

- [ ] T016 [US2] Update README.md line ~270: change `cmd/nomos-provider-terraform-remote-state/` to `cmd/provider/`

### Validation for User Story 2

- [ ] T017 [US2] Verify git diff shows workflow and README updates
- [ ] T018 [US2] Verify README is accurate and paths are correct when read
- [ ] T019 [US2] Confirm folder location: ls cmd/provider/main.go (MUST exist)

**Checkpoint**: Primary documentation and CI/CD updated. Developer can navigate to cmd/provider/main.go easily.

---

## Phase 4: User Story 3 - Documentation References Updated (Priority: P3)

**Goal**: Update all remaining documentation (skills, historical specs) for complete consistency

**Independent Test**: After this phase, grep finds zero old path references (except this spec)

### Skills Documentation Updates for User Story 3

- [ ] T020 [P] [US3] Update .github/skills/run-provider/SKILL.md line ~275: debug build example
- [ ] T021 [P] [US3] Update .github/skills/run-provider/SKILL.md line ~319: CPU profile example
- [ ] T022 [P] [US3] Update .github/skills/run-provider/SKILL.md line ~322: memory profile example
- [ ] T023 [P] [US3] Update .github/skills/run-provider/SKILL.md line ~393: Linux build example
- [ ] T024 [P] [US3] Update .github/skills/run-provider/SKILL.md line ~396: macOS build example
- [ ] T025 [P] [US3] Update .github/skills/run-provider/SKILL.md line ~399: Windows build example
- [ ] T026 [P] [US3] Update .github/skills/run-provider/SKILL.md line ~520: Dockerfile example
- [ ] T027 [P] [US3] Update .github/skills/run-tests/SKILL.md line ~48: test output reference

### Historical Documentation Updates for User Story 3 (Optional)

- [ ] T028 [P] [US3] Update specs/001-tfstate-provider/tasks.md line 36: setup task S2 path reference
- [ ] T029 [P] [US3] Update specs/001-tfstate-provider/tasks.md line 54: gRPC task G1 path reference

### Validation for User Story 3

- [ ] T030 [US3] Run path reference check: grep -r "cmd/nomos-provider-terraform-remote-state" . --exclude-dir=.git --exclude-dir=bin --exclude="specs/003-rename-provider-folder/*" | grep -v "Binary file" (MUST return zero matches)
- [ ] T031 [US3] Verify all skill examples work: test one example command from run-provider skill
- [ ] T032 [US3] Spot check documentation accuracy: grep -n "cmd/provider" README.md (should have references)

**Checkpoint**: All documentation consistent. Zero old path references remain.

---

## Phase 5: Final Validation & Polish

**Purpose**: Comprehensive validation that all success criteria are met

- [ ] T033 Run full test suite: make test (MUST pass)
- [ ] T034 Run code quality checks: make verify (MUST pass fmt, vet, lint)
- [ ] T035 Verify git history preserved: git log --follow --oneline cmd/provider/main.go | head -5 (should show commits from before rename)
- [ ] T036 Run binary functionality test: start provider, verify PROVIDER_PORT printed, kill process
- [ ] T037 Review all changes: git status and git diff
- [ ] T038 Stage all changes: git add -A
- [ ] T039 Create commit with conventional commit message: "refactor: rename cmd folder to provider for brevity"
- [ ] T040 Final verification: make clean && make build && make test && make verify (all MUST pass)

**Checkpoint**: All success criteria met. Ready to push and create PR.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - establishes baseline
- **User Story 1 (Phase 2)**: Depends on Setup - CRITICAL for build system
- **User Story 2 (Phase 3)**: Depends on User Story 1 completion (folder must exist at new location)
- **User Story 3 (Phase 4)**: Depends on User Story 1 completion (folder must exist at new location)
- **Final Validation (Phase 5)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: BLOCKING - must complete first (creates new folder location)
- **User Story 2 (P2)**: Can start after US1 - updates CI/CD and primary docs
- **User Story 3 (P3)**: Can start after US1 - updates skills and historical docs
- **US2 and US3 can proceed in parallel** after US1 completes (different files, no conflicts)

### Within Each User Story

**User Story 1**:
- T004 (git mv) MUST complete first
- T005 (verify rename) immediate after T004
- T006-T008 (Makefile) can proceed together
- T009-T013 (validation) MUST be sequential, after Makefile updates

**User Story 2**:
- T014-T015 (workflows) marked [P], can run in parallel
- T016 (README) can run in parallel with workflows
- T017-T019 (validation) sequential, after updates

**User Story 3**:
- T020-T029 (all doc updates) marked [P], can run in parallel (different files)
- T030-T032 (validation) sequential, after updates

### Parallel Opportunities

**After User Story 1 completes**, these can run in parallel:
- User Story 2 tasks (T014-T016): 3 files, no conflicts
- User Story 3 tasks (T020-T029): 10 files, no conflicts

**Within User Story 3**, all documentation updates can run in parallel (T020-T029).

---

## Parallel Example: User Story 3 Documentation Updates

```bash
# All these tasks update different files and can run simultaneously:
T020: Update .github/skills/run-provider/SKILL.md (line 275)
T021: Update .github/skills/run-provider/SKILL.md (line 319)
T022: Update .github/skills/run-provider/SKILL.md (line 322)
T023: Update .github/skills/run-provider/SKILL.md (line 393)
T024: Update .github/skills/run-provider/SKILL.md (line 396)
T025: Update .github/skills/run-provider/SKILL.md (line 399)
T026: Update .github/skills/run-provider/SKILL.md (line 520)
T027: Update .github/skills/run-tests/SKILL.md (line 48)
T028: Update specs/001-tfstate-provider/tasks.md (line 36)
T029: Update specs/001-tfstate-provider/tasks.md (line 54)
```

---

## Implementation Strategy

### Sequential Execution (Recommended for Single Developer)

1. **Phase 1 (Setup)**: Validate baseline - 5 minutes
2. **Phase 2 (US1)**: Rename + build system - 10 minutes
   - VALIDATE: Build succeeds
3. **Phase 3 (US2)**: CI/CD + README - 5 minutes
   - VALIDATE: Workflows and docs updated
4. **Phase 4 (US3)**: Skills + historical - 10 minutes
   - VALIDATE: Zero old references
5. **Phase 5 (Final)**: Comprehensive validation + commit - 5 minutes

**Total Time**: ~35 minutes

### Parallel Execution (If Using Automation)

1. Complete Phase 1 (Setup) - 5 minutes
2. Complete Phase 2 (US1) - 10 minutes [BLOCKING]
3. Run Phase 3 (US2) and Phase 4 (US3) in parallel - 10 minutes
4. Complete Phase 5 (Final) - 5 minutes

**Total Time**: ~30 minutes

### MVP Scope (Minimal Working Rename)

**Just User Story 1** delivers the core value:
- Folder renamed
- Build system works
- Developers can build from cmd/provider

This alone is a viable checkpoint. US2 and US3 are polish.

---

## Success Criteria Mapping

| Success Criterion | Validated By |
|-------------------|--------------|
| SC-001: Build commands succeed | T010, T011, T040 |
| SC-002: Tests pass | T002, T033, T040 |
| SC-003: Zero old path references | T030 |
| SC-004: Find main.go in <10 sec | T019 (manual validation) |
| SC-005: Git history preserved | T035 |
| SC-006: Binary name unchanged | T012, T040 |

---

## Notes

- **No tests to write**: This is refactoring only. Existing tests validate correctness.
- **[P] marker**: Indicates tasks that can run in parallel (different files)
- **[Story] marker**: Maps task to user story for traceability and independent verification
- **Checkpoints**: After each user story phase, that aspect should be independently verifiable
- **Atomic commit**: All changes committed together in Phase 5 to preserve history
- **Estimated total time**: 30-35 minutes for complete implementation
- **Risk level**: Low (structural change only, no code changes)
- **Rollback**: Simple - `git reset --hard` before commit, or revert commit after

---

## Task Summary

**Total Tasks**: 40 tasks across 5 phases  
**Setup**: 3 tasks  
**User Story 1 (P1)**: 10 tasks (critical path)  
**User Story 2 (P2)**: 6 tasks (can parallel after US1)  
**User Story 3 (P3)**: 13 tasks (can parallel after US1)  
**Final Validation**: 8 tasks  

**Parallelizable**: 12 tasks marked [P] (all in US2 and US3)  
**Critical Path**: User Story 1 (blocks US2 and US3)  
**MVP**: User Story 1 only (folder renamed, build works)
