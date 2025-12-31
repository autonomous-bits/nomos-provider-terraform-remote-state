# Data Model: Folder Rename Entities

**Feature**: Rename Provider Command Folder  
**Date**: 2025-12-31  
**Phase**: 1 - Design

## Core Entities

### 1. File Reference

Represents a reference to the command folder path in a file.

**Attributes**:
- **file_path** (string): Absolute path to the file containing the reference
- **line_number** (integer, optional): Line number where reference appears (for targeted updates)
- **reference_type** (enum): Type of reference
  - `PATH` - Direct file system path reference (e.g., `cmd/nomos-provider-terraform-remote-state`)
  - `GO_BUILD` - Go build command reference (e.g., `./cmd/nomos-provider-terraform-remote-state`)
  - `NARRATIVE` - Documentation prose mentioning the path
- **old_pattern** (string): The text pattern to search for
- **new_pattern** (string): The replacement text
- **is_optional** (boolean): Whether updating this reference is optional (e.g., historical task logs)

**Relationships**:
- Belongs to one **File Category**
- Has one **Update Strategy**

### 2. File Category

Represents a logical grouping of files requiring similar update strategies.

**Attributes**:
- **category_name** (string): Identifier for the category
- **priority** (integer): Update order (lower = earlier)
- **validation_command** (string, optional): Command to validate updates in this category

**Categories** (in priority order):
1. **BUILD** - Build system files (Makefile)
2. **CI_CD** - Continuous integration workflows
3. **DOCUMENTATION** - User-facing documentation (README, docs/)
4. **AGENTS** - Agent context files
5. **SKILLS** - Skill documentation
6. **HISTORICAL** - Historical specs/tasks (optional updates)

**Relationships**:
- Contains many **File References**

### 3. Update Strategy

Defines how to update references in a specific file or category.

**Attributes**:
- **strategy_name** (string): Identifier
- **tool** (enum): Tool to use for updates
  - `GIT_MV` - Git move command
  - `TEXT_REPLACE` - Search and replace in file
  - `MANUAL` - Manual review required
- **pattern_matching** (enum): How to find references
  - `EXACT` - Exact string match
  - `REGEX` - Regular expression pattern
- **atomic** (boolean): Whether all updates must succeed or none
- **verification** (string): Command to verify update succeeded

**Relationships**:
- Applied to many **File References**

### 4. Validation Check

Represents a verification step to ensure rename completed successfully.

**Attributes**:
- **check_name** (string): Identifier for the check
- **check_type** (enum): Category of validation
  - `BUILD` - Compilation and build system
  - `TEST` - Test suite execution
  - `LINT` - Code quality checks
  - `PATH_VERIFICATION` - No old path references remain
  - `GIT_HISTORY` - Git history preserved
  - `BINARY` - Binary output correct
- **command** (string): Shell command to run
- **expected_result** (string): Description of success criteria
- **required** (boolean): Whether this check must pass
- **order** (integer): Execution order

**Relationships**:
- Independent entity (no relationships)

## State Transitions

### File Reference States

```
IDENTIFIED → QUEUED → IN_PROGRESS → UPDATED → VERIFIED
                  ↓
                SKIPPED (for optional references)
```

**State Definitions**:
- **IDENTIFIED**: Reference found via search
- **QUEUED**: Awaiting update
- **IN_PROGRESS**: Currently being updated
- **UPDATED**: Text replaced
- **VERIFIED**: Update confirmed correct
- **SKIPPED**: Marked optional, not updated

### Validation Check States

```
PENDING → RUNNING → PASSED
                 → FAILED → REMEDIATED → RUNNING
```

**State Definitions**:
- **PENDING**: Not yet executed
- **RUNNING**: Currently executing
- **PASSED**: Check succeeded
- **FAILED**: Check failed, requires remediation
- **REMEDIATED**: Issue fixed, ready to re-run

## Data Relationships

```
┌─────────────────┐
│ File Category   │
│ (BUILD, CI_CD,  │
│  DOCS, etc.)    │
└────────┬────────┘
         │
         │ contains
         │ (1:N)
         ↓
┌─────────────────┐      ┌──────────────────┐
│ File Reference  │──────│ Update Strategy  │
│ (path, line,    │ uses │ (tool, pattern,  │
│  pattern)       │ (N:1)│  verification)   │
└─────────────────┘      └──────────────────┘


┌──────────────────┐
│ Validation Check │
│ (independent)    │
└──────────────────┘
```

## Validation Rules

### File Reference Validation
- `file_path` MUST exist on filesystem
- `old_pattern` MUST be non-empty
- `new_pattern` MUST be non-empty and different from `old_pattern`
- If `is_optional` is false, update MUST succeed

### File Category Validation
- `priority` values MUST be unique
- Categories MUST be processed in priority order
- All non-optional references in a category MUST be updated before proceeding to next category

### Update Strategy Validation
- `atomic` updates MUST rollback if any reference in the set fails
- `verification` command MUST exit with code 0 for success

### Validation Check Validation
- `required` checks MUST pass before rename is considered complete
- Checks MUST execute in `order` sequence
- FAILED required checks MUST block completion

## Examples

### File Reference Example

```yaml
file_reference:
  file_path: "/Users/user/repo/Makefile"
  line_number: 23
  reference_type: GO_BUILD
  old_pattern: "./cmd/$(BINARY_NAME)"
  new_pattern: "$(CMD_PATH)"
  is_optional: false
```

### File Category Example

```yaml
file_category:
  category_name: "BUILD"
  priority: 1
  validation_command: "make build"
```

### Validation Check Example

```yaml
validation_check:
  check_name: "No Old Path References"
  check_type: PATH_VERIFICATION
  command: "grep -r 'cmd/nomos-provider-terraform-remote-state' . --exclude-dir=.git --exclude='specs/003-*' | grep -v 'Binary file'"
  expected_result: "No matches (exit code 1 from grep)"
  required: true
  order: 3
```
