# Changelog

All notable changes to the Nomos Terraform Remote State Provider will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - MVP - TBD

### Added
- Initial provider implementation
- Core gRPC service contract (Init, Fetch, Info, Health, Shutdown)
- Azure Blob Storage backend support
  - Azure identity-based authentication (DefaultAzureCredential)
  - Account key authentication
  - SAS token authentication
- AWS S3 backend support (planned)
  - IAM role-based authentication
  - Access key authentication
- Local filesystem backend support
  - File path resolution
  - Basic security validation
- Configuration validation and error handling
- Graceful shutdown and resource cleanup
- Provider metadata (alias, version, type)
- Health check endpoint
- Subprocess mode with TCP port discovery
- Output filtering by path
- Output type conversion (string, number, boolean, list, map)

### Security
- Path traversal prevention
- Secure credential handling
- TLS support for gRPC (when required)
- Input validation for all configuration parameters

### Documentation
- README with quickstart guide
- Architecture documentation
- Backend configuration reference
- API contract documentation
- Contributing guidelines

[Unreleased]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/autonomous-bits/nomos-provider-terraform-remote-state/releases/tag/v0.1.0
