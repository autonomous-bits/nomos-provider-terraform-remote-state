# nomos-provider-terraform-remote-state Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-12-30

## Active Technologies
- Go 1.25+ + github.com/autonomous-bits/nomos/libs/provider-proto, google.golang.org/grpc, github.com/Azure/azure-sdk-for-go/sdk/storage/azblob (002-separate-backend-type)
- Local filesystem (local backend), Azure Blob Storage (azurerm backend) (002-separate-backend-type)
- Go 1.25+ + None (refactoring only - no new dependencies) (003-rename-provider-folder)
- N/A (file system operations only) (003-rename-provider-folder)

- Go 1.25+ (001-tfstate-provider)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.21+

## Code Style

Go 1.25+: Follow standard conventions

## Recent Changes
- 003-rename-provider-folder: Added Go 1.25+ + None (refactoring only - no new dependencies)
- 002-separate-backend-type: Added Go 1.25+ + github.com/autonomous-bits/nomos/libs/provider-proto, google.golang.org/grpc, github.com/Azure/azure-sdk-for-go/sdk/storage/azblob

- 001-tfstate-provider: Added Go 1.25+
<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
