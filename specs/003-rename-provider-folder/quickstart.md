# Quick Start: Rename Provider Command Folder

**Feature**: 003-rename-provider-folder  
**Estimated Time**: 20-30 minutes  
**Difficulty**: Low  
**Prerequisites**: Git, Make, Go 1.25+

## Overview

This guide walks through renaming `cmd/nomos-provider-terraform-remote-state/` to `cmd/provider/` while preserving Git history and updating all references.

## Step-by-Step Procedure

### Step 1: Verify Current State

Ensure you're starting from a clean working directory on the feature branch.

```bash
# Verify on correct branch
git branch --show-current
# Expected: 003-rename-provider-folder

# Verify clean working directory
git status
# Expected: nothing to commit, working tree clean

# Verify current build works
make clean && make build && make test
# Expected: All pass
```

**Expected Result**: Starting from a known-good state.

---

### Step 2: Rename the Folder

Use `git mv` to preserve history.

```bash
# Rename the folder using Git
git mv cmd/nomos-provider-terraform-remote-state cmd/provider

# Verify rename tracked by Git
git status
# Expected: renamed: cmd/nomos-provider-terraform-remote-state/main.go -> cmd/provider/main.go

# Verify folder exists at new location
ls -la cmd/provider/
# Expected: main.go present
```

**Expected Result**: Folder renamed, Git tracking rename.

---

### Step 3: Update Makefile

Modify Makefile to use new folder path while keeping binary name unchanged.

```bash
# Open Makefile in editor
# OR use automated replacement (example with sed)

# Add CMD_PATH variable after BINARY_NAME
# Find the line with BINARY_NAME and add CMD_PATH below it
```

**Manual Edit**: Add this line after `BINARY_NAME=nomos-provider-terraform-remote-state`:
```makefile
CMD_PATH=./cmd/provider
```

**Replace build command**:
```makefile
# Change this line in the build target:
# FROM: $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
# TO:   $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
```

**Replace install command**:
```makefile
# Change this line in the install target:
# FROM: $(GO) install $(LDFLAGS) ./cmd/$(BINARY_NAME)
# TO:   $(GO) install $(LDFLAGS) $(CMD_PATH)
```

**Verify Makefile changes**:
```bash
# Check the changes
git diff Makefile

# Test build with new Makefile
make clean && make build
# Expected: Binary created at bin/nomos-provider-terraform-remote-state
```

**Expected Result**: Makefile updated, build works.

---

### Step 4: Update CI/CD Workflows

Update GitHub Actions workflow files.

#### ci.yml

```bash
# Edit .github/workflows/ci.yml
# Find line ~107 with: go build -v -o "$output" ./cmd/nomos-provider-terraform-remote-state
# Replace with:        go build -v -o "$output" ./cmd/provider
```

#### release.yml

```bash
# Edit .github/workflows/release.yml
# Find line ~56 with: -o "$output" ./cmd/nomos-provider-terraform-remote-state
# Replace with:       -o "$output" ./cmd/provider
```

**Verify workflow changes**:
```bash
git diff .github/workflows/
# Expected: Both ci.yml and release.yml show path updates
```

**Expected Result**: Workflows updated to use new path.

---

### Step 5: Update Documentation

Update README.md and other documentation files.

#### README.md

```bash
# Edit README.md
# Find line ~270 with: - `cmd/nomos-provider-terraform-remote-state/`: Main executable entry point
# Replace with:        - `cmd/provider/`: Main executable entry point
```

**Verify documentation change**:
```bash
git diff README.md
# Expected: Path reference updated
```

**Expected Result**: README accurate.

---

### Step 6: Update Skills

Update skill documentation with corrected examples.

#### .github/skills/run-provider/SKILL.md

This file has multiple references. Use search-replace:

```bash
# Option 1: Using sed (macOS)
sed -i '' 's|cmd/nomos-provider-terraform-remote-state|cmd/provider|g' .github/skills/run-provider/SKILL.md

# Option 2: Manual find-replace in editor
# Find: cmd/nomos-provider-terraform-remote-state
# Replace with: cmd/provider
# Replace All
```

**Verify all replacements** (7 expected):
```bash
git diff .github/skills/run-provider/SKILL.md
# Expected: 7 changes across examples
```

#### .github/skills/run-tests/SKILL.md

```bash
# Edit .github/skills/run-tests/SKILL.md
# Find line ~48 with test output example
# Update path in test output
```

**Expected Result**: All skill examples updated.

---

### Step 7: Update Historical Specs (Optional)

Update task history for consistency.

```bash
# Edit specs/001-tfstate-provider/tasks.md
# Update line 36: Change cmd/nomos-provider-terraform-remote-state/ to cmd/provider/
# Update line 54: Change cmd/nomos-provider-terraform-remote-state/main.go to cmd/provider/main.go
```

**Expected Result**: Historical documentation consistent.

---

### Step 8: Validate Changes

Run comprehensive validation suite.

```bash
# Build validation
make clean
make build
# MUST PASS: Binary created at bin/nomos-provider-terraform-remote-state

# Test validation
make test
# MUST PASS: All tests pass

# Code quality validation
make verify
# MUST PASS: fmt, vet, lint all pass

# Path reference validation (expect NO matches except this spec)
grep -r "cmd/nomos-provider-terraform-remote-state" . \
  --exclude-dir=.git \
  --exclude-dir=bin \
  --exclude="specs/003-rename-provider-folder/*" \
  | grep -v "Binary file"
# MUST PASS: Zero matches (or only spec.md mentioning the old path in requirements)

# Git history validation
git log --follow --oneline cmd/provider/main.go | head -5
# MUST PASS: Shows commits from before rename

# Binary functionality validation
./bin/nomos-provider-terraform-remote-state &
PID=$!
sleep 2
ps -p $PID && echo "✓ Provider running"
kill $PID
# MUST PASS: Provider starts successfully and prints PROVIDER_PORT
```

**Expected Result**: All validations pass.

---

### Step 9: Review Changes

Review all changes before committing.

```bash
# See all changed files
git status

# Review all diffs
git diff --staged  # If you staged changes
git diff           # If not yet staged

# Expected changed files:
# - renamed: cmd/nomos-provider-terraform-remote-state/ -> cmd/provider/
# - modified: Makefile
# - modified: .github/workflows/ci.yml
# - modified: .github/workflows/release.yml
# - modified: README.md
# - modified: .github/skills/run-provider/SKILL.md
# - modified: .github/skills/run-tests/SKILL.md
# - modified: specs/001-tfstate-provider/tasks.md (optional)
```

**Expected Result**: All and only expected files changed.

---

### Step 10: Commit Changes

Create a single atomic commit with conventional commit message.

```bash
# Stage all changes
git add -A

# Commit with conventional commit format
git commit -m "refactor: rename cmd folder to provider for brevity

- Rename cmd/nomos-provider-terraform-remote-state to cmd/provider
- Update Makefile to use CMD_PATH variable (binary name unchanged)
- Update CI/CD workflows (.github/workflows/*.yml)
- Update documentation (README.md, skills, historical specs)
- Preserve Git history with git mv

BREAKING CHANGE: Source folder path changed from cmd/nomos-provider-terraform-remote-state to cmd/provider. Binary name remains nomos-provider-terraform-remote-state.

Closes #003"

# Verify commit
git show --stat
# Expected: Shows all changes with rename tracked
```

**Expected Result**: Clean commit with proper message.

---

### Step 11: Final Verification

One last check before pushing.

```bash
# Ensure everything still works
make clean && make build && make test && make verify
# Expected: All pass

# Run binary
./bin/nomos-provider-terraform-remote-state &
PID=$!
sleep 1
ps -p $PID && echo "✓ Final check passed"
kill $PID
# Expected: Provider runs correctly
```

**Expected Result**: Everything works as before, just with shorter path.

---

### Step 12: Push Changes

Push to remote for review.

```bash
# Push feature branch
git push origin 003-rename-provider-folder

# Expected: Branch pushed successfully
# Next: Create pull request for review
```

**Expected Result**: Changes ready for PR review.

---

## Troubleshooting

### Build fails after rename

**Problem**: `make build` fails with "no such file or directory"

**Solution**: Verify Makefile has correct `CMD_PATH=./cmd/provider` and uses it in build commands.

```bash
grep CMD_PATH Makefile
grep "$(CMD_PATH)" Makefile
```

---

### Old path references still found

**Problem**: Validation finds old path references

**Solution**: Check each file reported and update manually.

```bash
# Find remaining references
grep -rn "cmd/nomos-provider-terraform-remote-state" . \
  --exclude-dir=.git \
  --exclude-dir=bin \
  --exclude="specs/003-*"

# Update each file listed
```

---

### Git history not preserved

**Problem**: `git log --follow cmd/provider/main.go` shows no history

**Solution**: You likely used `mv` instead of `git mv`. Redo with git mv:

```bash
# Reset
git reset --hard HEAD~1

# Redo with git mv
git mv cmd/nomos-provider-terraform-remote-state cmd/provider
```

---

### Binary name changed

**Problem**: Binary is now named `provider` instead of `nomos-provider-terraform-remote-state`

**Solution**: Fix Makefile - `BINARY_NAME` should still be `nomos-provider-terraform-remote-state`

```bash
# Check Makefile
grep BINARY_NAME= Makefile
# Should be: BINARY_NAME=nomos-provider-terraform-remote-state (unchanged)
```

---

## Summary

**Total Time**: ~20-30 minutes  
**Files Changed**: 8 files (1 rename + 7 updates)  
**Risk**: Low (structural change only, no code changes)  
**Validation**: 7 automated checks ensure correctness

After completion:
- ✅ Folder renamed to `cmd/provider`
- ✅ Binary name unchanged (`nomos-provider-terraform-remote-state`)
- ✅ All builds, tests, and verification pass
- ✅ Git history preserved
- ✅ All documentation updated
- ✅ CI/CD workflows updated
- ✅ Zero old path references remain

**Next Steps**: Create PR, request review, merge to main.
