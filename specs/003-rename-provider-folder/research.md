# Research: Folder Rename Best Practices

**Feature**: Rename Provider Command Folder  
**Date**: 2025-12-31  
**Phase**: 0 - Research & Analysis

## Research Questions

1. What is the best practice for renaming folders in Go projects while preserving Git history?
2. What are the standard Go project conventions for command folder naming?
3. What files typically reference command folder paths in Go projects?
4. How should build systems (Makefile) be updated after folder renames?
5. What validation steps ensure a successful folder rename?

## Findings

### 1. Git History Preservation

**Decision**: Use `git mv` for all folder renames  
**Rationale**: Git automatically tracks renames when using `git mv`, preserving file history and enabling `git log --follow` to work correctly. This is critical for maintaining attribution and change history.

**Best Practice**:
```bash
# Correct approach
git mv cmd/old-name cmd/new-name
git commit -m "refactor: rename cmd/old-name to cmd/new-name"

# Incorrect approach (breaks history)
mv cmd/old-name cmd/new-name  # Don't do this
git add cmd/new-name
git rm cmd/old-name
```

**Verification**:
```bash
# After rename, verify history is preserved
git log --follow cmd/provider/main.go
# Should show full history from when it was cmd/nomos-provider-terraform-remote-state/main.go
```

**Alternatives Considered**:
- Manual `mv` + `git add/rm`: Rejected because it breaks `git log --follow` tracking
- Creating new folder and copying files: Rejected because it loses all Git history

### 2. Go Command Folder Naming Conventions

**Decision**: Use concise, descriptive names like `cmd/provider`, `cmd/server`, `cmd/cli`  
**Rationale**: Go community convention favors short, clear command names that identify the binary's purpose without redundant organization/project prefixes.

**Examples from Popular Go Projects**:
- **Kubernetes**: `cmd/kubelet`, `cmd/kube-apiserver`, `cmd/kubectl` (not `cmd/kubernetes-kubelet`)
- **Docker**: `cmd/docker`, `cmd/dockerd` (not `cmd/docker-docker`)
- **Terraform**: `cmd/terraform` (not `cmd/hashicorp-terraform`)
- **Prometheus**: `cmd/prometheus`, `cmd/promtool` (not `cmd/prometheus-server`)

**Pattern Observed**:
- Single-binary projects: `cmd/[project-name]` (e.g., `cmd/terraform`)
- Multi-binary projects: `cmd/[purpose]` (e.g., `cmd/server`, `cmd/cli`, `cmd/worker`)
- Provider projects: `cmd/provider` is idiomatic when the repo is clearly a provider

**Why `cmd/provider` is appropriate**:
- Repository is `nomos-provider-terraform-remote-state` (context clear from repo name)
- Only one binary in this project (main provider executable)
- "provider" clearly identifies the purpose within the cmd/ directory
- Consistent with patterns: `cmd/server`, `cmd/client`, `cmd/worker`

**Alternatives Considered**:
- `cmd/nomos-provider-terraform-remote-state`: Current name, rejected for redundancy
- `cmd/main`: Rejected as too generic, doesn't convey purpose
- `cmd/tfstate`: Rejected as too abbreviated, unclear

### 3. Files That Reference Command Paths

**Decision**: Systematically search and update all references  
**Rationale**: Incomplete updates cause confusion and broken documentation. Must update all references atomically.

**File Categories Requiring Updates**:

**Build Configuration**:
- `Makefile`: Build targets, install targets, run targets
- CI/CD workflows: `.github/workflows/*.yml`

**Documentation**:
- `README.md`: Architecture diagrams, folder structure, build instructions
- `AGENTS.md`: Agent context about project structure
- `docs/*.md`: Any architectural or development guides
- `specs/*/tasks.md`: Historical task references (optional but recommended)

**Skills & Automation**:
- `.github/skills/*/SKILL.md`: Example commands and procedures

**Search Strategy**:
```bash
# Find all references to old path
grep -r "cmd/nomos-provider-terraform-remote-state" . \
  --exclude-dir=.git \
  --exclude-dir=bin \
  --exclude-dir=vendor

# Common patterns to search for:
# - Direct path references: cmd/nomos-provider-terraform-remote-state
# - Go module imports: ./cmd/nomos-provider-terraform-remote-state
# - Make variables: $(BINARY_NAME)/...
```

**Alternatives Considered**:
- Update only Makefile: Rejected as incomplete, breaks documentation
- Use symlinks for backward compatibility: Rejected as confusing, doesn't solve problem

### 4. Makefile Update Strategy

**Decision**: Update `BINARY_NAME` variable usage, keep binary name unchanged  
**Rationale**: The Makefile references `cmd/$(BINARY_NAME)` in build commands. After rename, this path will be wrong. Must update to use new folder name while keeping binary output name the same.

**Current Makefile Pattern**:
```makefile
BINARY_NAME=nomos-provider-terraform-remote-state
$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
```

**Problem**: This assumes folder name == binary name. After rename, `./cmd/$(BINARY_NAME)` won't exist.

**Solution**: Separate folder path from binary name
```makefile
BINARY_NAME=nomos-provider-terraform-remote-state
CMD_PATH=./cmd/provider

$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
$(GO) install $(LDFLAGS) $(CMD_PATH)
```

**Benefits**:
- Binary name stays `nomos-provider-terraform-remote-state` (preserves external references)
- Folder can be renamed independently
- Clear separation of concerns
- Future renames easier

**Alternatives Considered**:
- Hardcode new path: Rejected as less maintainable
- Change binary name too: Rejected as breaks external tooling expectations

### 5. Validation & Testing Strategy

**Decision**: Multi-level verification with automated checks  
**Rationale**: Ensure rename is complete and correct before merging. Catch all broken references.

**Validation Levels**:

**Level 1: Build System**
```bash
make clean
make build    # Must succeed
make test     # All tests must pass
make verify   # fmt + vet + lint must pass
make install  # Install must work
```

**Level 2: Binary Verification**
```bash
# Verify binary name unchanged
ls -la bin/ | grep nomos-provider-terraform-remote-state

# Verify binary runs correctly
./bin/nomos-provider-terraform-remote-state
# Should print: PROVIDER_PORT=<port>
```

**Level 3: Path Reference Check**
```bash
# Ensure NO references to old path remain (except this spec)
grep -r "cmd/nomos-provider-terraform-remote-state" . \
  --exclude-dir=.git \
  --exclude-dir=bin \
  --exclude="specs/003-rename-provider-folder/*" \
  | grep -v "Binary file"

# Expected: Zero matches
```

**Level 4: Git History Verification**
```bash
# Verify history preserved
git log --follow --oneline cmd/provider/main.go
# Should show commits from before the rename
```

**Level 5: Documentation Accuracy**
```bash
# Spot check key documentation
grep -n "cmd/provider" README.md     # Should have references
grep -n "cmd/provider" Makefile      # Should have references
grep -n "cmd/provider" .github/workflows/ci.yml  # Should have references
```

**Alternatives Considered**:
- Manual verification only: Rejected as error-prone
- Only build verification: Rejected as incomplete (doesn't catch doc issues)

## Implementation Approach

Based on research findings, the rename will follow this sequence:

1. **Atomic Rename**: Use `git mv` to rename folder
2. **Build Configuration**: Update Makefile with separate CMD_PATH variable
3. **Documentation Sweep**: Update all markdown files with new path
4. **CI/CD Update**: Update workflow files
5. **Skill Update**: Update skill documentation
6. **Validation**: Run complete test suite + path verification
7. **Commit**: Single atomic commit preserving history

## Risk Mitigation

**Risk**: External tools reference old path  
**Mitigation**: Binary name unchanged, external tools use binary not source path

**Risk**: Developers have stale clones  
**Mitigation**: Document in CHANGELOG, provide migration note

**Risk**: Incomplete reference updates  
**Mitigation**: Automated grep verification as part of validation

**Risk**: CI/CD breaks  
**Mitigation**: Update workflows before merge, verify in PR checks

## References

- Go Project Layout: https://github.com/golang-standards/project-layout
- Git History Preservation: `git mv` documentation
- Kubernetes cmd/ structure: https://github.com/kubernetes/kubernetes/tree/master/cmd
- Terraform cmd/ structure: https://github.com/hashicorp/terraform/tree/main/cmd
