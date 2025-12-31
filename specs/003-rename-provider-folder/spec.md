# Feature Specification: Rename Provider Command Folder

**Feature Branch**: `003-rename-provider-folder`  
**Created**: 2025-12-31  
**Status**: Draft  
**Input**: User description: "the folder should be called provider"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Developer Builds Provider (Priority: P1)

A developer building the provider from source expects to find the main command code in a folder with a conventional, concise name that clearly identifies it as the provider's entry point.

**Why this priority**: This is the most common developer interaction with the codebase. A shorter, clearer path improves developer experience and aligns with Go community conventions (e.g., `cmd/server`, `cmd/cli`).

**Independent Test**: Can be fully tested by running `make build` after the rename and verifying the binary is created successfully. Delivers immediate value by simplifying the build process without requiring other changes.

**Acceptance Scenarios**:

1. **Given** a developer clones the repository, **When** they navigate to the `cmd/` directory, **Then** they see a folder named `provider` (not `nomos-provider-terraform-remote-state`)
2. **Given** the renamed folder structure, **When** the developer runs `make build`, **Then** the binary is created successfully in `bin/nomos-provider-terraform-remote-state`
3. **Given** the build completes, **When** the developer inspects the output binary name, **Then** it remains `nomos-provider-terraform-remote-state` (only the source folder changed, not the binary name)

---

### User Story 2 - Developer Navigates Codebase (Priority: P2)

A developer navigating the codebase expects consistent, intuitive folder naming that follows Go project conventions, making it easier to locate the provider's main entry point.

**Why this priority**: Improved codebase clarity and maintainability. While important for long-term developer productivity, it doesn't block functionality.

**Independent Test**: Can be tested by having developers (or AI agents) navigate the folder structure and report ease of locating the main command code. Success = reduced time to find `main.go`.

**Acceptance Scenarios**:

1. **Given** a developer unfamiliar with the codebase, **When** they look for the provider's main entry point, **Then** they find it in `cmd/provider/main.go` within 10 seconds
2. **Given** a developer views the project structure, **When** they see `cmd/provider/`, **Then** they immediately understand it contains the provider's main command (no ambiguity)

---

### User Story 3 - Documentation References Updated (Priority: P3)

Documentation, scripts, and configuration files that reference the old folder path are updated to use the new path, ensuring consistency across the project.

**Why this priority**: Prevents confusion and broken references, but doesn't affect runtime functionality. Lower priority because it can be addressed after the core rename.

**Independent Test**: Can be tested by searching the entire codebase for references to the old path and verifying all are updated. Success = zero occurrences of old path in documentation/configs.

**Acceptance Scenarios**:

1. **Given** the folder has been renamed, **When** a developer searches for `cmd/nomos-provider-terraform-remote-state` in the repository, **Then** no references are found in documentation files (README, docs/, specs/)
2. **Given** updated documentation, **When** a developer follows instructions in README.md, **Then** all file paths and commands work correctly
3. **Given** updated Makefile, **When** a developer runs `make build`, **Then** the build process references the correct folder path

---

### Edge Cases

- What happens when external tools or scripts reference the old folder path?
  - **Mitigation**: Binary name remains unchanged (`nomos-provider-terraform-remote-state`), so external tools that reference the binary continue to work. Only source code path changes.
- How does the rename affect existing clones of the repository on developer machines?
  - **Impact**: Developers will need to pull the rename. Git automatically handles the rename when pulling. Stale local builds will fail until `make clean && make build` is run.
- What if documentation exists in multiple places (README, docs/, GitHub wiki, external sites)?
  - **Coverage**: This feature updates in-repo documentation. External documentation (wikis, blogs) is out of scope but should be noted in CHANGELOG.
- How do we handle any symbolic links or references in CI/CD pipelines?
  - **Coverage**: CI/CD workflows (.github/workflows/*.yml) are updated as part of this feature (FR-010, US2).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The folder `cmd/nomos-provider-terraform-remote-state` MUST be renamed to `cmd/provider`
- **FR-002**: The Makefile MUST be updated to reference `cmd/provider` instead of `cmd/nomos-provider-terraform-remote-state`
- **FR-003**: The binary output name MUST remain `nomos-provider-terraform-remote-state` (no change to `BINARY_NAME` variable)
- **FR-004**: All user-facing documentation files (README.md, docs/) MUST be updated to reference the new folder path. Historical internal specs (specs/001-*) SHOULD be updated for consistency but are optional.
- **FR-005**: The go.mod module path MUST remain unchanged (`github.com/autonomous-bits/nomos-provider-terraform-remote-state`)
- **FR-006**: All build, test, and verification commands MUST work correctly after the rename
- **FR-007**: The main.go file MUST remain functionally identical (only its location changes)
- **FR-008**: Git history MUST preserve the rename using `git mv` for proper tracking
- **FR-009**: Any scripts in `.specify/` or `.github/` that reference the old path MUST be updated
- **FR-010**: CI/CD workflows (if present) MUST be updated to reference the new path

### Key Entities

- **Provider Command Folder**: The directory containing main.go, currently named `cmd/nomos-provider-terraform-remote-state`, to be renamed to `cmd/provider`
- **Build Configuration**: Makefile and any build scripts that reference the command folder path
- **Documentation**: All markdown files (README, docs/, specs/) that reference file paths

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All build commands (`make build`, `make install`, `make run`) execute successfully after the rename
- **SC-002**: All tests (`make test`, `make verify`) pass after the rename
- **SC-003**: Zero occurrences of the old path `cmd/nomos-provider-terraform-remote-state` remain in documentation files
- **SC-004**: Developers can locate the main entry point in under 10 seconds (average)
- **SC-005**: Git properly tracks the rename (git log --follow shows history continuity)
- **SC-006**: The binary name remains `nomos-provider-terraform-remote-state` (no change to output artifact name)
