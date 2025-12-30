# AGENTS.md

Context and instructions for AI coding agents working on the Nomos Terraform Remote State Provider.

## Project Overview

A Nomos provider that reads Terraform/OpenTofu remote state files and exposes outputs via the `nomos.provider.v1.ProviderService` gRPC contract.

**Language**: Go 1.25+
**Architecture**: gRPC subprocess with port discovery (prints `PROVIDER_PORT=<port>` to stdout)  
**Backends**: Local filesystem, Azure Blob Storage (MVP)  
**State Format**: Terraform/OpenTofu v4+ (Terraform 0.12+, OpenTofu 1.x+)

## Essential Commands

```bash
# Setup
make deps && make tidy

# Build and test
make build
make test              # 80%+ coverage required
make verify            # fmt + vet + lint + test

# Run tests for specific package
go test -v -race ./internal/provider/

# Integration tests (requires //go:build integration tag)
go test -tags=integration -v ./...
```

## Key Project-Specific Rules

### Architecture Decisions

- **Domain-driven packages**: `internal/{provider,backend,state,config}` (NOT by type)
- **No state caching**: Fetch fresh state on every RPC call
- **Credentials**: Environment variables ONLY (AWS_*, AZURE_*, GOOGLE_*) - never in config
- **MVP scope**: Root module outputs only (path `["vpc_id"]`). Nested modules deferred to Phase 2

### Critical Patterns

**Backend Interface**:
```go
type Backend interface {
    ReadState(ctx context.Context) ([]byte, error)
}
```

**gRPC Error Mapping**:
- `NotFound`: Missing outputs/files/workspaces
- `InvalidArgument`: Bad configuration
- `FailedPrecondition`: Init errors, unsupported state versions
- `PermissionDenied`: Auth failures
- `Unavailable`: Network/timeout errors
- `Internal`: Parsing errors

**State Version Validation**: Reject state format version < 4 with `FailedPrecondition`

### Before Committing

- Run `make verify` (must pass)
- Tests with race detection pass
- Coverage ≥80%
- Integration tests pass (if modified backends)

## Related Documentation

Read these before making changes:
- Feature Spec: `specs/001-tfstate-provider/spec.md`
- Implementation Plan: `specs/001-tfstate-provider/plan.md`
- Tasks: `specs/001-tfstate-provider/tasks.md`
- Contributing: `CONTRIBUTING.md`

## Agent Coordination

Specialized agents handle different concerns:
- **provider-orchestrator**: Coordinates all phases
- **go-provider-architect**: Architecture decisions
- **go-provider-implementer**: Code implementation
- **go-provider-tester**: Test creation (TDD)
- **go-security-reviewer**: Security review
- **grpc-service-specialist**: gRPC/protobuf design
- **documentation-specialist**: Documentation

Coordinate through orchestrator for cross-cutting concerns.
