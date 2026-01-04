# Contributing to Nomos Terraform Remote State Provider

Thank you for your interest in contributing to the Nomos Terraform Remote State Provider! This document provides guidelines and standards for contributing to this project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Development Standards](#development-standards)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing Guidelines](#testing-guidelines)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)

## Code of Conduct

This project follows the autonomous-bits development standards. We expect all contributors to:

- Be respectful and professional
- Focus on constructive feedback
- Collaborate openly and transparently
- Prioritize code quality and maintainability

## Development Standards

This provider follows the autonomous-bits development standards and the Go community best practices:

### Architecture Principles

- **Provider Contract**: Implement the full `nomos.provider.v1.ProviderService` gRPC contract
- **Package Organization**: Domain-driven design (NOT type-driven)
- **Interface Design**: Consumer-defined, minimal interfaces
- **Dependency Injection**: Accept dependencies through constructors
- **Error Handling**: Always return errors, never panic in library code

### Go Standards

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `context.Context` as first parameter in functions
- Never store context in structs
- Accept interfaces, return structs
- Package by domain, not by type

## Getting Started

### Prerequisites

- Go 1.25+ or later
- Make
- golangci-lint (for linting)
- Git

### Initial Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/nomos-provider-terraform-remote-state.git
   cd nomos-provider-terraform-remote-state
   ```

3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/autonomous-bits/nomos-provider-terraform-remote-state.git
   ```

4. Install dependencies:
   ```bash
   make deps
   ```

5. Verify setup:
   ```bash
   make verify
   ```

## Development Workflow

### Branch Strategy

- `main`: Stable, production-ready code
- `feature/*`: Feature branches
- `bugfix/*`: Bug fix branches
- `hotfix/*`: Urgent production fixes

### Making Changes

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following code standards

3. Run verification:
   ```bash
   make verify
   ```

4. Commit your changes:
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

5. Keep your branch up to date:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

## Code Standards

### File Organization

```
internal/
├── provider/        # Provider implementation
│   ├── provider.go  # Core provider logic
│   └── handler.go   # gRPC handler
├── backend/         # Backend implementations
│   ├── azure/       # Azure Blob Storage
│   ├── s3/          # AWS S3
│   └── local/       # Local filesystem
├── state/           # State parsing and output handling
│   ├── parser.go    # State file parsing
│   └── outputs.go   # Output extraction
└── config/          # Configuration
    └── config.go    # Configuration structures
```

### Naming Conventions

- **Packages**: Short, lowercase, singular (`provider`, `config`, `state`)
- **Types**: PascalCase for exported (`Provider`, `Config`)
- **Functions**: camelCase for unexported, PascalCase for exported
- **Interfaces**: End with "-er" for single method (`Fetcher`, `Parser`)
- **Receivers**: 1-2 letter abbreviations, consistent (`p *Provider`)
- **Variables**: Short in short scopes (`ctx`, `err`), descriptive in larger scopes

### Documentation

All exported types, functions, and constants MUST have documentation comments:

```go
// Provider implements the nomos.provider.v1.ProviderService gRPC contract
// for fetching Terraform remote state outputs.
type Provider struct {
    // ...
}

// Fetch retrieves configuration data from the specified path in the
// Terraform remote state. The path follows the dot notation format
// (e.g., "outputs.vpc_id").
func (p *Provider) Fetch(ctx context.Context, req *pb.FetchRequest) (*pb.FetchResponse, error) {
    // ...
}
```

### Error Handling

- Always return errors, never panic
- Wrap errors with context: `fmt.Errorf("failed to fetch state: %w", err)`
- Use sentinel errors for specific conditions: `var ErrStateNotFound = errors.New("state file not found")`
- Log errors at appropriate levels

### Context Usage

```go
// CORRECT: Context as first parameter
func DoWork(ctx context.Context, data string) error {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    // ... do work
}

// INCORRECT: Context in struct
type Worker struct {
    ctx context.Context // NEVER DO THIS
}
```

## Testing Guidelines

### Test Organization

- Place tests in `*_test.go` files alongside the code
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test both success and error paths

### Test Structure

```go
func TestProvider_Fetch(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        want    string
        wantErr bool
    }{
        {
            name:    "fetch vpc_id",
            path:    "outputs.vpc_id",
            want:    "vpc-123456",
            wantErr: false,
        },
        {
            name:    "invalid path",
            path:    "invalid",
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Coverage Requirements

- Aim for >80% code coverage
- All exported functions must have tests
- Critical paths must have comprehensive tests

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific test
go test -v -run TestProvider_Fetch ./internal/provider
```

## Submitting Changes

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(backend): add AWS S3 backend support
fix(provider): handle nil state outputs correctly
docs(readme): update quickstart guide
test(state): add parser unit tests
```

### Pull Request Process

1. Ensure all tests pass: `make verify`
2. Update documentation if needed
3. Update CHANGELOG.md with your changes
4. Push your branch to your fork
5. Create a Pull Request to `autonomous-bits/nomos-provider-terraform-remote-state:main`

### PR Description Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project standards
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] All tests pass
- [ ] No linting errors
```

## Release Process

Releases follow [Semantic Versioning](https://semver.org/):

- **Major** (v1.0.0): Breaking changes
- **Minor** (v0.1.0): New features, backwards compatible
- **Patch** (v0.0.1): Bug fixes, backwards compatible

### Creating a Release

1. Update CHANGELOG.md with release notes
2. Update version in code if applicable
3. Create and push tag:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push upstream v0.1.0
   ```
4. GitHub Actions will build and publish the release

## Questions or Issues?

- Open an issue for bug reports or feature requests
- Check existing issues before creating new ones
- Provide detailed information and reproduction steps

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

---

Thank you for contributing to Nomos Terraform Remote State Provider!
