<!--
=============================================================================
SYNC IMPACT REPORT
=============================================================================
Version Change: INITIAL → 1.0.0
Change Type: Initial Constitution Creation
Date: 2025-12-30

Modified Principles:
  - N/A (Initial creation)

Added Sections:
  ✅ Core Principles (7 principles)
  ✅ Quality Gates
  ✅ Multi-Agent Development Workflow
  ✅ Governance

Removed Sections:
  - N/A (Initial creation)

Template Consistency:
  ✅ .specify/templates/plan-template.md - Updated with constitution check references
  ✅ .specify/templates/spec-template.md - Aligned with acceptance criteria requirements
  ✅ .specify/templates/tasks-template.md - Updated with TDD and phase structure
  ✅ .github/agents/provider-orchestrator.md - Already aligned with constitution

Follow-up Actions:
  - None required for initial constitution

Commit Message Suggestion:
  docs: establish constitution v1.0.0 (Nomos provider development standards)
=============================================================================
-->

# Nomos Provider Terraform Remote State Constitution

## Core Principles

### I. gRPC Contract Compliance (NON-NEGOTIABLE)

All provider implementations MUST fully implement the `nomos.provider.v1.ProviderService` gRPC contract without exception. This includes:

- **Init**: Initialize provider with configuration (MUST validate inputs, return proper status codes)
- **Fetch**: Retrieve configuration data by path (MUST handle errors, use context for cancellation)
- **Info**: Return provider metadata (alias, version, type)
- **Health**: Check provider health status (MUST be reliable and fast)
- **Shutdown**: Gracefully shut down the provider (MUST clean up all resources)

**Rationale**: Contract compliance ensures interoperability with the Nomos compiler and consistent behavior across all providers. The gRPC interface is the provider's public API and MUST be implemented correctly.

**Enforcement**: Phase 2 validation MUST verify all methods implemented with proper signatures, error handling, and status codes before proceeding to Phase 3.

### II. Process Model & Discovery (MANDATORY)

Providers MUST operate as independent subprocesses with discoverable endpoints:

- Start as subprocess, listen on random TCP port (use `net.Listen("tcp", ":0")`)
- Print `PROVIDER_PORT=<port>` to stdout immediately after binding (REQUIRED for Nomos discovery)
- Respond to health checks within 100ms
- Implement graceful shutdown with resource cleanup
- Exit cleanly on SIGTERM/SIGINT

**Rationale**: The subprocess model with port discovery enables Nomos to manage provider lifecycles independently, enables parallel execution, and ensures proper isolation between providers.

**Enforcement**: Phase 2 validation MUST verify startup behavior, port printing, and graceful shutdown before implementation phase begins.

### III. Test-Driven Development (NON-NEGOTIABLE)

TDD is MANDATORY for all provider development:

1. **Write Tests First**: All tests MUST be written before implementation
2. **Red-Green-Refactor**: Tests MUST fail initially, then pass after implementation
3. **Coverage Requirements**:
   - Minimum 80% overall code coverage
   - 100% coverage for critical paths (Init, Fetch, Health, Shutdown)
4. **Test Organization**:
   - Unit tests: Table-driven tests in `*_test.go` files
   - Integration tests: Tagged with `//go:build integration`
   - Benchmarks: For performance-critical code paths

**Rationale**: TDD ensures code correctness, enables safe refactoring, and produces testable designs. The coverage requirements ensure critical functionality is thoroughly validated.

**Enforcement**: Phase 4 validation MUST verify tests were written first (commit history), all tests pass, and coverage meets minimums before security review.

### IV. Idiomatic Go & Code Quality (MANDATORY)

All code MUST follow Go best practices and autonomous-bits standards:

- **Formatting**: MUST pass `gofmt` and `goimports` (zero violations)
- **Linting**: MUST pass `golangci-lint` with zero warnings
- **Error Handling**: NO panics in library code; all errors MUST return with context
- **Naming**: Follow Go conventions (PascalCase exports, camelCase unexported, short receivers)
- **Documentation**: ALL exported symbols MUST have godoc comments
- **Package Organization**: Domain-driven structure (not type-driven); avoid util/common/helpers

**Rationale**: Idiomatic Go code is more maintainable, easier to review, and follows community standards. Consistent formatting and linting prevent bikeshedding and catch bugs early.

**Enforcement**: Phase 3 validation MUST verify formatting, linting, and error handling before testing phase. All violations MUST be resolved.

### V. Context Propagation (MANDATORY)

Context MUST be first parameter in all functions that perform I/O or long-running operations:

- Context enables cancellation and timeout control
- MUST propagate context through call chains
- MUST check context cancellation in loops and long operations
- Goroutines MUST have clean exit paths using context
- Use `context.WithTimeout` for operations with deadlines

**Rationale**: Proper context usage enables graceful cancellation, prevents resource leaks, and allows timeouts to be enforced consistently across the provider.

**Enforcement**: Code review MUST verify context as first parameter. Integration tests MUST verify context cancellation behavior.

### VI. Security First (NON-NEGOTIABLE)

Security MUST be built-in, not bolted-on:

- **Input Validation**: Validate ALL inputs at boundaries (MUST prevent injection, path traversal)
- **Secrets Management**: NO hardcoded secrets; use environment variables or secure vaults
- **Error Messages**: MUST NOT expose internal details or sensitive information
- **Resource Limits**: MUST prevent DoS through timeouts and size limits
- **Dependencies**: MUST run `govulncheck` and resolve critical vulnerabilities
- **Cryptography**: Use `crypto/rand` for security-sensitive random (NEVER `math/rand`)

**Rationale**: Security vulnerabilities in providers can compromise entire Nomos deployments. Security must be considered at every phase, not as an afterthought.

**Enforcement**: Phase 5 security review is MANDATORY. All critical vulnerabilities MUST be resolved before documentation phase.

### VII. Multi-Agent Coordination (PROJECT-SPECIFIC)

This provider follows a multi-agent development workflow coordinated by `provider-orchestrator`:

- **Architecture**: `go-provider-architect` designs package structure and interfaces
- **gRPC Implementation**: `grpc-service-specialist` implements ProviderService contract
- **Core Implementation**: `go-provider-implementer` implements business logic
- **Testing**: `go-provider-tester` creates comprehensive test suites
- **Security**: `go-security-reviewer` performs security audits
- **Documentation**: `documentation-specialist` creates comprehensive documentation

**Rationale**: Specialized agents ensure expertise is applied at each phase. The orchestrator ensures phases execute in the correct order with proper validation gates.

**Enforcement**: The `provider-orchestrator` MUST delegate to specialized agents using the `runSubagent` tool. Direct implementation without agent coordination is prohibited.

## Quality Gates

Each development phase MUST pass its quality gate before proceeding to the next phase.

### Phase 1: Architecture Gate
- [ ] Package structure follows domain-driven design
- [ ] All ProviderService methods planned
- [ ] Configuration schema defined
- [ ] Architecture diagram created
- [ ] Security considerations documented
- [ ] Performance requirements identified

### Phase 2: gRPC Service Gate
- [ ] All ProviderService methods implemented
- [ ] Error handling uses correct gRPC status codes
- [ ] Context properly propagated throughout
- [ ] Server prints `PROVIDER_PORT=<port>` on startup
- [ ] Graceful shutdown implemented
- [ ] Interceptors configured (logging, recovery, validation)

### Phase 3: Implementation Gate
- [ ] Code formatted with gofmt/goimports (zero violations)
- [ ] golangci-lint passes (zero warnings)
- [ ] go vet passes (no issues)
- [ ] No panics in library code
- [ ] All errors properly wrapped with context
- [ ] Context as first parameter throughout
- [ ] Goroutines have clean exit paths
- [ ] All exported symbols documented

### Phase 4: Testing Gate
- [ ] Minimum 80% code coverage achieved
- [ ] Critical paths (Init, Fetch, Health, Shutdown) at 100% coverage
- [ ] Table-driven tests used appropriately
- [ ] Integration tests tagged with `//go:build integration`
- [ ] All tests pass: `go test ./...`
- [ ] Benchmarks written for performance-critical code
- [ ] Error cases thoroughly tested
- [ ] Context cancellation tested

### Phase 5: Security Gate
- [ ] All inputs validated at boundaries
- [ ] Path traversal protection implemented
- [ ] No hardcoded secrets in code
- [ ] crypto/rand used for security-sensitive random
- [ ] Error messages don't expose internals
- [ ] Resource limits prevent DoS
- [ ] govulncheck passes (no critical issues)
- [ ] TLS configured for production (if applicable)

### Phase 6: Documentation Gate
- [ ] README.md comprehensive and accurate
- [ ] All exported symbols have godoc comments
- [ ] Usage examples provided and tested
- [ ] CHANGELOG.md follows Keep a Changelog format
- [ ] CONTRIBUTING.md present
- [ ] Architecture documented with diagrams
- [ ] Examples are runnable

## Multi-Agent Development Workflow

### Task-Driven Execution

When implementing features based on specifications (from `specs/` directory):

1. **Load Context**: Read `tasks.md`, `plan.md`, `data-model.md`, `contracts/`, `research.md`, and `spec.md`
2. **Parse Task Structure**: Extract phases, task IDs, dependencies, and parallel markers [P]
3. **Execute by Phase**: Complete each phase sequentially, respecting task dependencies
4. **TDD Enforcement**: Execute test tasks before implementation tasks
5. **Delegate to Specialists**: Route tasks to appropriate specialized agents
6. **Validate Gates**: Run phase validation checkpoints after each phase
7. **Track Progress**: Update task status in `tasks.md` after each completion
8. **Handle Failures**: Remediate issues before proceeding to next phase

### Phase Execution Order

```
Setup Phase (Sequential)
    ↓
Testing Phase (Write tests first - TDD)
    ↓
Implementation Phase (Implement to pass tests)
    ↓
Integration Phase (Connect components)
    ↓
Validation Phase (Quality gates)
    ↓
Documentation Phase (Polish)
```

### Agent Routing

Tasks are delegated based on category:

| Task Category | Delegate To |
|--------------|-------------|
| [S] Setup | `go-provider-architect` |
| [A] Architecture | `go-provider-architect` |
| [G] gRPC Service | `grpc-service-specialist` |
| [T] Testing | `go-provider-tester` |
| [I] Implementation | `go-provider-implementer` |
| [R] Security | `go-security-reviewer` |
| [D] Documentation | `documentation-specialist` |

## Governance

### Authority

This constitution supersedes all other development practices and guidelines. When conflicts arise between this constitution and other documentation, the constitution takes precedence.

### Amendment Process

1. **Proposal**: Document proposed changes with rationale and impact analysis
2. **Version Bump**: Follow semantic versioning for amendments
   - MAJOR: Backward incompatible governance/principle removals or redefinitions
   - MINOR: New principle/section added or materially expanded guidance
   - PATCH: Clarifications, wording improvements, typo fixes
3. **Sync Check**: Update all dependent templates and agent files
4. **Impact Report**: Document changes in HTML comment at top of constitution
5. **Approval**: Changes require explicit review and approval

### Compliance

- All pull requests MUST verify compliance with this constitution
- Quality gates MUST be enforced at each phase
- Violations MUST be documented and justified (complexity tracking in `plan.md`)
- The `provider-orchestrator` agent enforces constitutional compliance during development

### Rationale Documentation

When deviating from constitution principles (rare, exceptional cases only):

1. Document the deviation in `specs/###-feature/plan.md` under "Complexity Tracking"
2. Provide clear rationale for why the deviation is necessary
3. Demonstrate that alternatives were considered
4. Get explicit approval before proceeding

### Runtime Guidance

For detailed runtime development guidance, specialized agents provide implementation patterns:

- `.github/agents/provider-orchestrator.md` - Overall coordination and standards
- `.github/agents/go-provider-architect.md` - Architecture patterns
- `.github/agents/grpc-service-specialist.md` - gRPC implementation details
- `.github/agents/go-provider-implementer.md` - Code quality standards
- `.github/agents/go-provider-tester.md` - Testing patterns
- `.github/agents/go-security-reviewer.md` - Security practices
- `.github/agents/documentation-specialist.md` - Documentation standards

**Version**: 1.0.0 | **Ratified**: 2025-12-30 | **Last Amended**: 2025-12-30
