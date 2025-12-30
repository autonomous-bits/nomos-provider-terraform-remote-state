---
name: update-changelog
description: Update the CHANGELOG.md file following Keep a Changelog format and semantic versioning. Use this when adding features, fixing bugs, making changes, or preparing releases.
license: MIT
---

# Update Changelog

This skill provides a consistent way to update the CHANGELOG.md file for the Nomos Terraform Remote State Provider project, following Keep a Changelog format and Semantic Versioning principles.

## When to Use This Skill

Use this skill when you need to:
- Document a new feature or enhancement
- Record a bug fix
- Note a breaking change
- Document security fixes
- Prepare for a release
- Update unreleased changes
- Add deprecation notices
- Document removed features

## Prerequisites

- Write access to the repository
- Understanding of the change being documented
- Knowledge of whether the change is breaking or non-breaking

## Changelog Format

The project follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format:

### Change Categories

Changes are organized into these categories:

1. **Added** - New features or capabilities
2. **Changed** - Changes in existing functionality
3. **Deprecated** - Soon-to-be removed features
4. **Removed** - Removed features
5. **Fixed** - Bug fixes
6. **Security** - Security fixes or improvements

### Version Format

Follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html):
- **MAJOR.MINOR.PATCH** (e.g., 1.2.3)
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

## Update Process

### Step 1: Determine the Change Category

Match your change to the appropriate category:

**Added** - Use for:
- New backend support (e.g., "GCS backend support")
- New features (e.g., "Output filtering by path")
- New configuration options
- New API endpoints

**Changed** - Use for:
- Modifications to existing features
- Performance improvements
- Behavior changes (non-breaking)
- Updated dependencies

**Fixed** - Use for:
- Bug fixes
- Error handling improvements
- Crash fixes
- Incorrect behavior corrections

**Security** - Use for:
- Security vulnerability fixes
- Authentication improvements
- Authorization enhancements
- Credential handling updates

**Deprecated** - Use for:
- Features marked for future removal
- Old API versions
- Deprecated configuration options

**Removed** - Use for:
- Removed features
- Deleted configuration options
- Dropped support for versions

### Step 2: Determine Version Impact

**Patch Version** (0.1.X) - For:
- Bug fixes
- Security fixes
- Documentation updates
- Internal refactoring

**Minor Version** (0.X.0) - For:
- New features (backward compatible)
- New backend support
- Performance improvements
- Deprecations

**Major Version** (X.0.0) - For:
- Breaking API changes
- Removed features
- Incompatible configuration changes
- Major architectural changes

### Step 3: Update the Unreleased Section

For ongoing development, add changes to the `[Unreleased]` section:

```markdown
## [Unreleased]

### Added
- New feature description

### Fixed
- Bug fix description
```

### Step 4: Create a Release Entry

When preparing a release, create a new version section:

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- Feature descriptions...

### Fixed
- Bug fix descriptions...
```

### Step 5: Update Links

Add comparison links at the bottom:

```markdown
[Unreleased]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/compare/vX.Y.Z...HEAD
[X.Y.Z]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/releases/tag/vX.Y.Z
```

## Examples

### Example 1: Adding a New Feature

**Scenario:** Implemented Google Cloud Storage backend support

**Update:**
```markdown
## [Unreleased]

### Added
- Google Cloud Storage (GCS) backend support
  - Service account authentication
  - Workload identity authentication
  - Custom endpoint configuration
```

### Example 2: Fixing a Bug

**Scenario:** Fixed a crash when parsing invalid JSON in state files

**Update:**
```markdown
## [Unreleased]

### Fixed
- State parser now handles invalid JSON gracefully without crashing
- Improved error messages for malformed state files
```

### Example 3: Security Fix

**Scenario:** Fixed path traversal vulnerability in local backend

**Update:**
```markdown
## [Unreleased]

### Security
- Fixed path traversal vulnerability in local filesystem backend
- Added path validation to prevent directory traversal attacks
```

### Example 4: Breaking Change

**Scenario:** Changed configuration format for backend credentials

**Update:**
```markdown
## [Unreleased]

### Changed
- **BREAKING**: Backend configuration now uses nested `credentials` object
  - Old: `backend.azure_account_key`
  - New: `backend.credentials.account_key`
- Updated documentation with migration guide
```

### Example 5: Preparing a Release

**Scenario:** Ready to release version 0.2.0

**Before:**
```markdown
## [Unreleased]

### Added
- GCS backend support
- Retry logic for backend connections

### Fixed
- State parser crash on invalid JSON
```

**After:**
```markdown
## [Unreleased]

## [0.2.0] - 2025-12-30

### Added
- Google Cloud Storage (GCS) backend support
  - Service account authentication
  - Workload identity authentication
- Retry logic for backend connections with exponential backoff

### Fixed
- State parser now handles invalid JSON gracefully without crashing

[Unreleased]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/releases/tag/v0.1.0
```

## Writing Guidelines

### Be Specific
❌ **Bad:** "Fixed bug"
✅ **Good:** "Fixed state parser crash when parsing JSON with missing required fields"

### Use Action-Oriented Language
❌ **Bad:** "There is now support for AWS S3"
✅ **Good:** "AWS S3 backend support"

### Include Context for Breaking Changes
Always explain breaking changes:
```markdown
### Changed
- **BREAKING**: Configuration format changed
  - Old format: `backend.key = "value"`
  - New format: `backend.config.key = "value"`
  - See migration guide in docs/migration-v2.md
```

### Group Related Changes
```markdown
### Added
- AWS S3 backend support
  - IAM role-based authentication
  - Access key authentication
  - Session token support
  - Custom endpoint configuration
```

### Reference Issues/PRs (Optional)
```markdown
### Fixed
- Fixed connection timeout in Azure backend (#123)
- Resolved memory leak in state parser (#125)
```

## Best Practices

1. **Update as you work**: Add changelog entries as you implement features
2. **Be thorough**: Include all user-facing changes
3. **Be concise**: One line per change, details in docs
4. **Use consistent formatting**: Follow the established pattern
5. **Review before release**: Ensure all changes are documented
6. **Keep ordering**: Most recent changes at the top
7. **Date releases**: Always include release date in YYYY-MM-DD format
8. **Update links**: Maintain comparison links at the bottom

## Changelog Structure Reference

Complete structure:
```markdown
# Changelog

All notable changes to the Nomos Terraform Remote State Provider will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New features for next release

### Changed
- Changes for next release

### Fixed
- Fixes for next release

## [1.0.0] - 2025-12-30

### Added
- Initial stable release
- Feature descriptions

### Changed
- Change descriptions

### Fixed
- Bug fix descriptions

### Security
- Security fix descriptions

[Unreleased]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/releases/tag/v1.0.0
```

## Semantic Versioning Quick Reference

Given a version number MAJOR.MINOR.PATCH:

- **MAJOR** (1.0.0 → 2.0.0): Breaking changes
  - API changes that break backward compatibility
  - Configuration format changes
  - Removed features
  - Behavior changes that could break existing usage

- **MINOR** (1.0.0 → 1.1.0): New features (backward compatible)
  - New features
  - New backend support
  - New configuration options
  - Deprecations (feature still works)
  - Performance improvements

- **PATCH** (1.0.0 → 1.0.1): Bug fixes (backward compatible)
  - Bug fixes
  - Security fixes
  - Documentation fixes
  - Internal refactoring
  - Dependency updates (non-breaking)

## Troubleshooting

### Uncertain About Version Bump
**Issue:** Not sure if change is breaking

**Solution:**
1. Ask: "Will existing configurations still work?"
2. If NO → Major version bump
3. If YES, new features → Minor version bump
4. If YES, only fixes → Patch version bump

### Multiple Unrelated Changes
**Issue:** Several changes across different categories

**Solution:** Group by category, list all changes:
```markdown
### Added
- Feature 1
- Feature 2

### Fixed
- Bug fix 1
- Bug fix 2
```

### Forgot to Update Before Release
**Issue:** Already released but changelog not updated

**Solution:**
1. Add a new patch release with changelog updates
2. Or update changelog and add note:
```markdown
## [1.0.1] - 2025-12-30

### Fixed
- Updated changelog for v1.0.0 (no code changes)
```

## Related Skills

- **run-tests**: Run tests before updating changelog
- **run-provider**: Verify provider works after changes
