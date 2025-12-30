# Feature Specification: Terraform Remote State Provider

**Feature Branch**: `001-tfstate-provider`  
**Created**: 2025-12-30  
**Status**: Draft  
**Input**: User description: "Create a provider to pull information from terraform remote state. This provider will be used by the Nomos CLI to read remote state from OpenTofu modules and pull outputs during compilation."

## Clarifications

### Session 2025-12-30

- Q: How should nested module outputs be accessed? → A: Post-MVP feature. MVP implements root module outputs only. Users must use Terraform/OpenTofu output passthroughs (declare module outputs at root level) as workaround. Direct nested module access with dot notation (path ["app", "database_url"] for module.app.output.database_url) deferred to Phase 2
- Q: How should credentials be handled for remote backends? → A: Environment variables only - provider reads credentials from standard env vars (AWS_*, AZURE_*, GOOGLE_*), never from config
- Q: What should the state file caching strategy be? → A: No caching - fetch state on every Fetch RPC call for guaranteed freshness
- Q: How should incompatible Terraform/OpenTofu state versions be handled? → A: Support current stable versions - validate state format version v4+ (Terraform 0.12+/OpenTofu 1.x+), reject older versions with clear error
- Q: Which backend types should be prioritized for initial release? → A: Local + Azure Blob

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Access Remote State Outputs (Priority: P1)

A DevOps engineer writing Nomos configuration needs to reference outputs from existing OpenTofu/Terraform infrastructure. They declare a tfstate source in their `.csl` file with backend configuration (local, remote, S3, etc.) and use the provider to fetch specific output values during compilation.

**Why this priority**: This is the core value proposition - enabling Nomos configurations to consume Terraform/OpenTofu state outputs, which is essential for integrating Nomos into existing infrastructure-as-code workflows.

**Independent Test**: Can be fully tested by configuring a local backend with a sample tfstate file, declaring the source in a `.csl` file, and verifying that output values are correctly retrieved and available during compilation.

**Acceptance Scenarios**:

1. **Given** a local Terraform state file with outputs, **When** a Nomos configuration declares a tfstate source with local backend configuration, **Then** the provider successfully initializes and can fetch the state outputs
2. **Given** a configured tfstate provider, **When** compilation requests a specific output path, **Then** the provider returns the correct value from the state file
3. **Given** an invalid backend configuration, **When** the provider attempts to initialize, **Then** a clear error message indicates the configuration problem

---

### User Story 2 - Multiple Backend Support (Priority: P2)

A platform team needs to access state from different storage backends (local files, Azure Blob Storage) based on where their infrastructure state is stored. They configure the tfstate provider with appropriate backend types and credentials.

**Why this priority**: Real-world infrastructure uses various backend types. Supporting multiple backends makes the provider practical for diverse environments.

**Independent Test**: Can be tested by creating multiple source declarations with different backend types (local, azurerm) and verifying each backend type correctly retrieves state outputs.

**Acceptance Scenarios**:

1. **Given** Terraform state stored in Azure Blob Storage, **When** a tfstate source is configured with Azure backend parameters (storage account, container, key), **Then** the provider fetches outputs from the Azure Blob state
2. **Given** Terraform state in local filesystem, **When** a tfstate source is configured with local backend parameters, **Then** the provider retrieves outputs from the local state file
3. **Given** multiple tfstate sources with different backend types, **When** compilation occurs, **Then** each provider instance correctly connects to its respective backend

---

### User Story 3 - Workspace Selection (Priority: P3)

A team managing multiple environments (dev, staging, prod) needs to access state from specific Terraform workspaces. They specify the workspace name in the provider configuration to target the correct environment's state.

**Why this priority**: Workspace support enables multi-environment configurations, but the provider is still useful without it by defaulting to the "default" workspace.

**Independent Test**: Can be tested by creating Terraform state with multiple workspaces and verifying the provider fetches outputs from the specified workspace.

**Acceptance Scenarios**:

1. **Given** Terraform state with multiple workspaces, **When** a tfstate source specifies a workspace name, **Then** the provider fetches outputs from that specific workspace
2. **Given** no workspace specified in configuration, **When** the provider initializes, **Then** it defaults to the "default" workspace
3. **Given** a non-existent workspace name, **When** the provider attempts to fetch, **Then** a clear error indicates the workspace doesn't exist

---

### Edge Cases

- What happens when the state file doesn't exist or is corrupted? (Provider returns NotFound for missing files, Internal for corrupted/unparseable JSON)
- How does the system handle authentication failures for remote backends? (Provider returns PermissionDenied for Azure auth failures)
- What if required credential environment variables are missing or invalid? (Provider returns FailedPrecondition during Init with clear error message)
- What if the requested output path doesn't exist in the state? (Provider returns NotFound with clear message identifying missing output name)
- How are deeply nested module outputs accessed (e.g., module.app.submodule.db.output.connection_string - more than 2 levels deep)? [Post-MVP: Phase 2 feature, requires root-level output workaround in MVP]
- What happens with state files from Terraform versions older than 0.12 (state format version < 4)? (Provider validates version during parsing, returns FailedPrecondition with clear message)
- How does the provider handle concurrent access to state files? [N/A: Read-only operations, no locking required. Multiple compilations can safely read same state file]
- What if network connectivity is lost during remote state fetch? (Provider returns Unavailable with timeout/network error details)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Provider MUST implement the Nomos Provider gRPC contract (`nomos.provider.v1.ProviderService`) with Init, Fetch, Info, Health, and Shutdown RPCs
- **FR-002**: Provider MUST support backend configuration through the Init RPC (backend type, path/URL, region, workspace) but MUST NOT accept credentials via backend configuration - credentials are read from environment variables only
- **FR-003**: Provider MUST support the "local" backend type for reading state from local filesystem paths and the "azurerm" backend type for reading state from Azure Blob Storage
- **FR-004**: Provider MUST parse Terraform/OpenTofu state file format (JSON) and extract root module output values (MVP scope: root outputs only; nested module outputs deferred to Phase 2 post-MVP)
- **FR-005**: Provider MUST implement path-based output access where path segments correspond to output names (e.g., path `["vpc_id"]` returns root output `vpc_id`). MVP scope: single-segment paths for root outputs only. Nested module output access (e.g., path `["app", "database_url"]` for module.app.output.database_url) is deferred to Phase 2 post-MVP
- **FR-006**: Provider MUST return outputs as structured data compatible with `google.protobuf.Struct` (maps, lists, scalars)
- **FR-007**: Provider MUST return appropriate gRPC error codes (NotFound for missing outputs, InvalidArgument for bad config, FailedPrecondition for initialization errors)
- **FR-008**: Provider MUST validate required backend configuration parameters during Init and fail fast with clear error messages
- **FR-008a**: Provider MUST read authentication credentials exclusively from environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, GOOGLE_APPLICATION_CREDENTIALS, etc.) following Terraform backend conventions
- **FR-009**: Provider MUST respect context cancellation and timeouts for all RPC operations
- **FR-010**: Provider MUST handle both Terraform and OpenTofu state file formats with state format version 4 or higher (Terraform 0.12+ / OpenTofu 1.x+)
- **FR-010a**: Provider MUST validate the state file version field during Init or first Fetch and return FailedPrecondition error with clear message for unsupported versions (< v4)
- **FR-011**: Provider SHOULD support additional backend types in future releases, prioritized as: 1) S3 (AWS market dominance), 2) GCS (Google Cloud), 3) HTTP (generic REST), 4) Terraform Cloud/Enterprise remote backend
- **FR-012**: Provider SHOULD support workspace selection through configuration
- **FR-013**: Provider MUST fetch state file on every Fetch RPC call to ensure data freshness (no caching)
- **FR-014**: Provider SHOULD provide meaningful metadata in the Info RPC (alias, version, type="terraform-remote-state")

### Key Entities *(include if feature involves data)*

- **State File**: JSON document containing Terraform/OpenTofu infrastructure state, including root module outputs
- **Backend Configuration**: Parameters specifying where and how to access the state (type, path, credentials, region, workspace)
- **Output Value**: Named value exported from Terraform root module, with type (string, number, bool, list, map) and value
- **Workspace**: Named environment context in Terraform/OpenTofu (defaults to "default")

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can compile Nomos configurations that reference Terraform state outputs within 5 seconds for local backends, within 10 seconds for remote backends (including network I/O and state file reads on each fetch)
- **SC-002**: Provider successfully retrieves outputs from local and Azure Blob Storage backends
- **SC-003**: Provider handles missing outputs gracefully with NotFound errors that clearly identify the missing output name
- **SC-004**: Provider returns output values that exactly match the types and values in the source Terraform state
- **SC-005**: Integration with Nomos CLI completes compilation without requiring manual provider registration beyond standard `nomos init` workflow
- **SC-006**: Provider passes all gRPC contract validation tests defined in the provider-proto module
- **SC-007**: Common Terraform backend configurations (local path, Azure storage account/container/key, standard workspace names) work without requiring custom provider logic beyond standard backend parameters
- **SC-008**: Provider startup and initialization completes in under 2 seconds for local backends

## Assumptions *(optional)*

- Terraform/OpenTofu state files follow standard JSON format as documented for state format version 4+ (Terraform 0.12+)
- State files contain a `outputs` section with root module outputs
- Users have appropriate credentials/permissions to access remote backends before using the provider, configured via standard environment variables (never in config files)
- The Nomos CLI will manage provider process lifecycle (start, stop, health checks)
- State files are not excessively large (< 100MB) for reasonable performance

## Dependencies *(optional)*

- **Nomos provider-proto module**: Provides gRPC service contract and generated Go stubs
- **Nomos CLI**: Manages provider installation, process lifecycle, and compilation integration
- **Go standard library**: For JSON parsing and file I/O
- **gRPC libraries**: For implementing the provider service (google.golang.org/grpc)
- **Cloud SDK libraries**: For remote backend access (AWS SDK for S3, Azure SDK for Blob, GCP SDK for GCS) - optional dependencies activated per backend
- **Terraform state reading libraries**: Consider leveraging existing Go libraries for state parsing if available

## Out of Scope *(optional)*

- Writing or modifying Terraform state (read-only provider)
- State locking mechanisms (assumes state is already locked by Terraform if needed, concurrent reads are safe)
- State migration between Terraform versions
- Direct resource data access (only root module outputs, not internal resource state)
- Nested module outputs in MVP (deferred to Phase 2 post-MVP - users must use root-level output passthroughs)
- Terraform plan file parsing (only state files)
- Running Terraform operations (init, plan, apply)
- State encryption/decryption beyond what the backend provides
- Custom backend implementations not in Terraform/OpenTofu core
- Backends beyond local and Azure Blob Storage in MVP (S3, GCS, HTTP deferred to future releases)

## Related Work *(optional)*

- OpenTofu `terraform_remote_state` data source: https://opentofu.org/docs/language/state/remote-state-data/
- Terraform backend configuration: Standard patterns for configuring state storage
- Nomos external providers architecture: docs/architecture/nomos-external-providers-feature-breakdown.md
- Nomos provider authoring guide: docs/guides/provider-authoring-guide.md
- Provider-proto module: libs/provider-proto/README.md

## Notes

This provider enables Nomos to integrate with existing Terraform/OpenTofu infrastructure by consuming state outputs. It follows the same pattern as Terraform's built-in `terraform_remote_state` data source but exposes the functionality through the Nomos provider interface.

The provider should be distributed as `nomos-provider-terraform-remote-state` following Nomos naming conventions, with binaries for major platforms (linux-amd64, darwin-amd64, darwin-arm64, windows-amd64).

Backend support can be phased: start with local and Azure Blob Storage backends for MVP, then add S3, GCS, and other remote backends in subsequent iterations. Each backend type can be implemented as a separate internal module with a common interface.

Output support is also phased: MVP implements root module outputs only (single-segment paths). Users requiring nested module outputs must use Terraform/OpenTofu root-level output passthroughs as a workaround. Direct nested module output access will be added in Phase 2 post-MVP.

Output support is also phased: MVP implements root module outputs only (single-segment paths). Users requiring nested module outputs must use Terraform/OpenTofu root-level output passthroughs as a workaround. Direct nested module output access will be added in Phase 2 post-MVP.
