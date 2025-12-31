# Feature Specification: Separate Backend Type from Provider Type

**Feature Branch**: `002-separate-backend-type`  
**Created**: 2025-12-31  
**Status**: Draft  
**Input**: User description: "Fix the type property issue - the type property should be used by CLI to identify provider source (like autonomous-bits/nomos-provider-file), not to specify the backend type within the provider configuration. The backend type should be inferred from the config or use a different field name like backend_type."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Configure Provider with Explicit Backend Type (Priority: P1)

A DevOps engineer configuring a Nomos provider needs to specify which Terraform backend type to use (local, azurerm, etc.) without conflicting with the provider's source type identifier used by the CLI. They declare a provider source with `type: 'autonomous-bits/nomos-provider-terraform-remote-state'` for CLI provider discovery, and separately specify `backend_type: 'local'` or `backend_type: 'azurerm'` in the configuration to indicate which backend implementation to use.

**Why this priority**: This is the core fix - separating concerns between CLI provider discovery (using `type` for package source) and runtime backend selection (using `backend_type` for storage backend). Without this, the provider configuration conflicts with Nomos CLI conventions.

**Independent Test**: Can be fully tested by configuring a provider with `type` set to the provider source identifier and `backend_type` set to 'local', then verifying the provider initializes correctly and accesses local state files. The CLI uses `type` for provider installation while the provider uses `backend_type` for backend selection.

**Acceptance Scenarios**:

1. **Given** a Nomos configuration file with a provider source using `type: 'autonomous-bits/nomos-provider-terraform-remote-state'` and `backend_type: 'local'`, **When** the provider initializes, **Then** the CLI uses the `type` field to locate/install the provider binary and the provider runtime uses `backend_type` to select the local backend implementation
2. **Given** a provider configured with `backend_type: 'azurerm'`, **When** the provider accesses state, **Then** it uses the Azure Blob Storage backend
3. **Given** a provider configured with `backend_type: 'local'`, **When** the provider accesses state, **Then** it uses the local filesystem backend
4. **Given** a provider configuration missing the `backend_type` field but containing backend-specific fields (like `path` for local or `storage_account_name` for azurerm), **When** the provider initializes, **Then** it auto-detects the backend type based on configuration keys present

---

### User Story 2 - Auto-detect Backend Type from Configuration (Priority: P2)

A DevOps engineer wants simplified configuration without explicitly specifying `backend_type` when the backend type is obvious from other configuration keys. They provide backend-specific configuration (e.g., `path` for local backend, or `storage_account_name`/`container_name` for Azure backend), and the provider automatically infers the backend type.

**Why this priority**: Reduces configuration verbosity and improves developer experience. The backend type is often redundant when backend-specific configuration keys are present. This is a convenience feature that doesn't break existing functionality.

**Independent Test**: Can be tested by creating configurations with backend-specific keys (e.g., `path` only, or `storage_account_name` + `container_name`) without `backend_type`, and verifying the provider correctly infers and initializes the appropriate backend.

**Acceptance Scenarios**:

1. **Given** a configuration with `path` field but no `backend_type`, **When** the provider initializes, **Then** it automatically selects the local backend
2. **Given** a configuration with `storage_account_name`, `container_name`, and `key` fields but no `backend_type`, **When** the provider initializes, **Then** it automatically selects the azurerm backend
3. **Given** a configuration with both `backend_type: 'local'` and `path` field, **When** the provider initializes, **Then** the explicit `backend_type` takes precedence over auto-detection
4. **Given** a configuration with conflicting signals (e.g., `backend_type: 'local'` but `storage_account_name` present), **When** the provider initializes, **Then** it returns a clear error indicating the configuration conflict

---

### Edge Cases

- How does the system handle a configuration with neither `backend_type` nor recognizable backend-specific keys? (Provider returns InvalidArgument error with message listing supported backends and required config keys)
- What if `backend_type` contains an unsupported value? (Provider returns InvalidArgument error listing supported backend types: local, azurerm)
- What if auto-detection finds ambiguous configuration (keys that could match multiple backends)? (Provider returns InvalidArgument error requesting explicit `backend_type` specification)
- How are backend-specific validation errors handled after backend type selection? (Provider returns clear error messages indicating which backend was selected and which config validation failed)
- What happens if the `type` field is present in the configuration? (Provider silently ignores it completely, as `type` is reserved for CLI use only. No validation or error checking is performed on this field)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Provider MUST accept `backend_type` field in configuration to explicitly specify the backend type (values: "local", "azurerm", or future backend types)
- **FR-002**: Provider MUST silently ignore the `type` field if present in configuration, as it is reserved for CLI provider source identification. The provider treats `type` as a CLI-only field and does not validate or process it in any way
- **FR-003**: Provider MUST support auto-detection of backend type when `backend_type` is not provided, by analyzing configuration keys: presence of `path` field indicates local backend; presence of `storage_account_name` + `container_name` indicates azurerm backend
- **FR-004**: Provider MUST prioritize explicit `backend_type` over auto-detection when both are present
- **FR-005**: Provider MUST return InvalidArgument error when: neither `backend_type` nor recognizable backend-specific keys are present; `backend_type` contains an unsupported value; auto-detection finds ambiguous configuration; explicit `backend_type` conflicts with provided configuration keys
- **FR-006**: Provider MUST update internal configuration parsing logic in config package to: extract `backend_type` instead of `type`; implement auto-detection logic
- **FR-007**: Provider MUST update all backend implementations to use the new configuration field name
- **FR-008**: Provider MUST update example configurations and documentation to use `backend_type` field
- **FR-009**: Provider MUST maintain all existing backend functionality (local, azurerm) with the new configuration field name

### Key Entities

- **Backend Type Identifier**: The `backend_type` field specifies which backend implementation to use. Transitions from `type` (conflicted with CLI usage) to `backend_type` (clear separation of concerns)
- **Provider Source Type**: String field (`type`) used by Nomos CLI to identify and download the provider. This field is not used by the provider runtime
- **Backend Configuration**: Map of configuration keys that includes both the backend type identifier and backend-specific parameters
- **Auto-detection Rules**: Logic that infers backend type from presence of specific configuration keys

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Provider configurations use separate fields for CLI provider discovery (`type`) and runtime backend selection (`backend_type`), eliminating configuration conflicts
- **SC-002**: Auto-detection successfully infers backend type from configuration keys in 100% of unambiguous cases
- **SC-003**: All existing test cases pass with the new `backend_type` field without requiring changes beyond configuration updates
- **SC-004**: Documentation and examples clearly demonstrate the separation between `type` (for CLI) and `backend_type` (for runtime)
- **SC-005**: Provider initialization errors clearly indicate whether the issue is with backend type selection or backend-specific configuration validation
- **SC-006**: Users creating new configurations can omit `backend_type` when using backend-specific keys, reducing configuration verbosity

## Assumptions *(mandatory)*

- The Nomos CLI uses the `type` field in source declarations to identify and download provider binaries
- The provider is not yet in production use, so no backward compatibility is required
- Users understand that `type` is for CLI provider source identification and `backend_type` is for runtime backend selection after reading documentation
- Backend-specific configuration keys are unique enough to enable reliable auto-detection (e.g., `path` is only used by local backend, `storage_account_name` is only used by azurerm backend)
- The CLI does not pass the `type` field to the provider's Init RPC, or if it does, the provider can safely ignore it

## Dependencies *(mandatory)*

- **internal/config/config.go**: Must be updated to accept `backend_type` and implement auto-detection logic
- **internal/backend/local.go**: May need updates if it directly references the type field
- **internal/backend/azurerm.go**: May need updates if it directly references the type field
- **All test files**: Must be updated to use `backend_type` in test configurations
- **Documentation files** (docs/*.md): Must be updated to reflect the new configuration field
- **Example configurations** (specs/001-tfstate-provider/quickstart.md): Must be updated to use `backend_type`
- **README.md**: Must include clear explanation of `type` vs `backend_type` usage

## Out of Scope *(mandatory)*

- Changing the gRPC contract or adding new RPC fields (this is purely an internal configuration change)
- Adding new backend types (S3, GCS, etc.) - this feature only addresses the field naming
- Implementing complex backend type detection heuristics beyond simple key presence checks
- Validating that `type` field in source declaration matches the provider name (that's the CLI's responsibility)
- Coordinating with CLI team to filter `type` field from Init RPC (assumption: provider can safely ignore it)

## Related Work *(mandatory)*

- **nomos-provider-file**: Reference implementation showing correct usage of `type` field for CLI provider source identification: https://github.com/autonomous-bits/nomos-provider-file
- **Feature 001-tfstate-provider**: Original implementation that introduced the `type` field for backend selection
- **Nomos CLI provider discovery**: Documentation of how CLI uses `type` field to locate and install providers
- **internal/config/config.go**: Current configuration parsing implementation that needs updating

## Notes

This change addresses a fundamental design issue where the same field name (`type`) was used for two different purposes:
1. **CLI level**: Identifying provider source for download (e.g., "autonomous-bits/nomos-provider-terraform-remote-state")
2. **Provider runtime level**: Selecting backend implementation (e.g., "local", "azurerm")

The fix introduces `backend_type` for provider runtime backend selection while reserving `type` for CLI provider source identification, aligning with the pattern used by nomos-provider-file and other Nomos providers.

**Example of correct usage:**

```yaml
# Nomos configuration file (.csl)
source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'  # CLI uses this for provider installation
  version: '0.1.0'
  backend_type: 'azurerm'  # Provider runtime uses this for backend selection
  storage_account_name: 'mystorageacct'
  container_name: 'tfstate'
  key: 'prod.terraform.tfstate'
```

Or with auto-detection:

```yaml
source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.0'
  # backend_type omitted - auto-detected from Azure-specific keys
  storage_account_name: 'mystorageacct'
  container_name: 'tfstate'
  key: 'prod.terraform.tfstate'
```

**Implementation Strategy:**
1. Update config parsing to use `backend_type` field for backend selection
2. Remove all references to `type` field for backend selection from config parsing
3. Implement auto-detection logic based on configuration keys
4. Update all tests and documentation
5. Ensure provider ignores any `type` field in configuration (reserved for CLI use)
