# Contract: File Update Specification

**Feature**: Rename Provider Command Folder  
**Date**: 2025-12-31  
**Phase**: 1 - Design

## File Updates Required

This document specifies the exact updates required for each file affected by the rename.

### Priority 1: Build System

#### Makefile

**File**: `Makefile`

**Changes Required**:
1. Add new variable `CMD_PATH`
2. Update all references to use `CMD_PATH` instead of constructing path from `BINARY_NAME`

**Specific Updates**:

**Update 1**: Add CMD_PATH variable after BINARY_NAME
```makefile
# OLD
BINARY_NAME=nomos-provider-terraform-remote-state
BUILD_DIR=bin

# NEW
BINARY_NAME=nomos-provider-terraform-remote-state
CMD_PATH=./cmd/provider
BUILD_DIR=bin
```

**Update 2**: build target
```makefile
# OLD
$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

# NEW
$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
```

**Update 3**: install target
```makefile
# OLD
$(GO) install $(LDFLAGS) ./cmd/$(BINARY_NAME)

# NEW
$(GO) install $(LDFLAGS) $(CMD_PATH)
```

**Verification**: `make build && make install`

---

### Priority 2: CI/CD

#### GitHub Actions - CI Workflow

**File**: `.github/workflows/ci.yml`

**Changes Required**: Update build command path

**Specific Updates**:

**Update 1**: Build step (approximately line 107)
```yaml
# OLD
go build -v -o "$output" ./cmd/nomos-provider-terraform-remote-state

# NEW
go build -v -o "$output" ./cmd/provider
```

**Verification**: Workflow runs successfully on PR

---

#### GitHub Actions - Release Workflow

**File**: `.github/workflows/release.yml`

**Changes Required**: Update build command path

**Specific Updates**:

**Update 1**: Build step (approximately line 56)
```yaml
# OLD
-o "$output" ./cmd/nomos-provider-terraform-remote-state

# NEW
-o "$output" ./cmd/provider
```

**Verification**: Tag-based release workflow succeeds

---

### Priority 3: Documentation

#### README.md

**File**: `README.md`

**Changes Required**: Update all path references

**Specific Updates**:

**Update 1**: Architecture section (approximately line 270)
```markdown
# OLD
- `cmd/nomos-provider-terraform-remote-state/`: Main executable entry point

# NEW
- `cmd/provider/`: Main executable entry point
```

**Verification**: Documentation is accurate when read by users

---

#### AGENTS.md

**File**: `AGENTS.md`

**Changes Required**: No updates required - file doesn't reference the cmd path

**Verification**: N/A

---

### Priority 4: Skills

#### Run Provider Skill

**File**: `.github/skills/run-provider/SKILL.md`

**Changes Required**: Update all example commands

**Specific Updates**:

**Update 1**: Debug build example (approximately line 275)
```markdown
# OLD
go build -gcflags="all=-N -l" -o bin/provider-debug ./cmd/nomos-provider-terraform-remote-state

# NEW
go build -gcflags="all=-N -l" -o bin/provider-debug ./cmd/provider
```

**Update 2**: CPU profile example (approximately line 319)
```markdown
# OLD
go run -cpuprofile=cpu.prof ./cmd/nomos-provider-terraform-remote-state

# NEW
go run -cpuprofile=cpu.prof ./cmd/provider
```

**Update 3**: Memory profile example (approximately line 322)
```markdown
# OLD
go run -memprofile=mem.prof ./cmd/nomos-provider-terraform-remote-state

# NEW
go run -memprofile=mem.prof ./cmd/provider
```

**Update 4**: Linux build example (approximately line 393)
```markdown
# OLD
GOOS=linux GOARCH=amd64 go build -o bin/provider-linux-amd64 ./cmd/nomos-provider-terraform-remote-state

# NEW
GOOS=linux GOARCH=amd64 go build -o bin/provider-linux-amd64 ./cmd/provider
```

**Update 5**: macOS build example (approximately line 396)
```markdown
# OLD
GOOS=darwin GOARCH=arm64 go build -o bin/provider-darwin-arm64 ./cmd/nomos-provider-terraform-remote-state

# NEW
GOOS=darwin GOARCH=arm64 go build -o bin/provider-darwin-arm64 ./cmd/provider
```

**Update 6**: Windows build example (approximately line 399)
```markdown
# OLD
GOOS=windows GOARCH=amd64 go build -o bin/provider-windows-amd64.exe ./cmd/nomos-provider-terraform-remote-state

# NEW
GOOS=windows GOARCH=amd64 go build -o bin/provider-windows-amd64.exe ./cmd/provider
```

**Update 7**: Dockerfile example (approximately line 520)
```dockerfile
# OLD
RUN go build -o provider ./cmd/nomos-provider-terraform-remote-state

# NEW
RUN go build -o provider ./cmd/provider
```

**Verification**: All example commands execute successfully

---

#### Run Tests Skill

**File**: `.github/skills/run-tests/SKILL.md`

**Changes Required**: Update test output reference

**Specific Updates**:

**Update 1**: Test output example (approximately line 48)
```markdown
# OLD
?   	github.com/autonomous-bits/nomos-provider-terraform-remote-state/cmd/nomos-provider-terraform-remote-state	[no test files]

# NEW
?   	github.com/autonomous-bits/nomos-provider-terraform-remote-state/cmd/provider	[no test files]
```

**Verification**: Skill documentation matches actual test output

---

### Priority 5: Historical (Optional)

#### Feature 001 Tasks

**File**: `specs/001-tfstate-provider/tasks.md`

**Changes Required**: Update historical references for consistency

**Specific Updates**:

**Update 1**: Setup task S2 (line 36)
```markdown
# OLD
- [X] [S2] [P] Create directory structure: cmd/nomos-provider-terraform-remote-state/, internal/provider/, internal/backend/, internal/state/, internal/config/

# NEW
- [X] [S2] [P] Create directory structure: cmd/provider/, internal/provider/, internal/backend/, internal/state/, internal/config/
```

**Update 2**: gRPC task G1 (line 54)
```markdown
# OLD
- [X] [G1] Create main entry point in cmd/nomos-provider-terraform-remote-state/main.go with gRPC server setup, port discovery (print PROVIDER_PORT), and signal handling

# NEW
- [X] [G1] Create main entry point in cmd/provider/main.go with gRPC server setup, port discovery (print PROVIDER_PORT), and signal handling
```

**Verification**: Historical documentation is consistent with current structure

---

## Validation Contract

After all updates are completed, the following validations MUST pass:

### Build Validation
```bash
make clean
make build
# MUST: Exit code 0, binary created at bin/nomos-provider-terraform-remote-state
```

### Test Validation
```bash
make test
# MUST: All tests pass, exit code 0
```

### Verification Validation
```bash
make verify
# MUST: fmt, vet, lint all pass, exit code 0
```

### Path Reference Validation
```bash
grep -r "cmd/nomos-provider-terraform-remote-state" . \
  --exclude-dir=.git \
  --exclude-dir=bin \
  --exclude="specs/003-rename-provider-folder/*" \
  | grep -v "Binary file"
# MUST: No matches (or only spec.md mentioning it in acceptance criteria)
```

### Git History Validation
```bash
git log --follow --oneline cmd/provider/main.go | wc -l
# MUST: > 0 (shows history preserved)
```

### Binary Name Validation
```bash
ls -1 bin/ | grep nomos-provider-terraform-remote-state
# MUST: Binary exists with unchanged name
```

### Binary Functionality Validation
```bash
./bin/nomos-provider-terraform-remote-state &
PID=$!
sleep 1
ps -p $PID > /dev/null && echo "PASS: Provider running"
kill $PID
# MUST: Provider starts and prints PROVIDER_PORT
```

## Acceptance Criteria Mapping

This contract maps directly to spec.md acceptance scenarios:

| Spec Scenario | Contract Validation |
|---------------|---------------------|
| AS1.1: Folder named `provider` | Rename operation section |
| AS1.2: `make build` succeeds | Build Validation |
| AS1.3: Binary name unchanged | Binary Name Validation |
| AS2.1: Find in `cmd/provider/main.go` | Rename operation creates this path |
| AS2.2: Clear purpose | N/A (subjective) |
| AS3.1: Zero old path references | Path Reference Validation |
| AS3.2: Documentation works | Verification Validation |
| AS3.3: Makefile references correct path | Build Validation |

## Summary

**Total Files to Update**: 7 files + 1 folder rename
- 1 folder (git mv)
- 1 Makefile
- 2 GitHub workflows
- 1 README
- 2 skill files
- 1 historical tasks file (optional)

**Total Text Replacements**: ~18 distinct replacements

**Estimated Implementation Time**: 15-30 minutes

**Risk Level**: Low (no code changes, only paths)
