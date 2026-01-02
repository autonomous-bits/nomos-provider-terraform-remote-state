# Release Packaging Guide

This document describes the packaging format for Nomos provider releases.

## Package Structure

Each release package contains a platform-specific binary organized in the following structure:

```text
nomos-provider-terraform-remote-state-v0.1.0-darwin-arm64/
└── provider

nomos-provider-terraform-remote-state-v0.1.0-linux-amd64/
└── provider

nomos-provider-terraform-remote-state-v0.1.0-windows-amd64/
└── provider.exe
```

## Nomos Provider Directory Layout

When the Nomos tooling installs a provider, it expects the following directory structure:

```text
~/.nomos/providers/<owner>/<repo>/<version>/<platform>/provider
```

Example:
```text
~/.nomos/providers/autonomous-bits/nomos-provider-terraform-remote-state/0.1.0/darwin-arm64/provider
```

### Key Points

1. **Binary Name**: The executable must be named `provider` (or `provider.exe` on Windows)
2. **Platform String**: Format is `<os>-<arch>` (e.g., `darwin-arm64`, `linux-amd64`, `windows-amd64`)
3. **Version**: Follows semantic versioning without the `v` prefix in the directory path

## Release Archive Formats

### Unix Platforms (Linux, macOS)

Archives use `.tar.gz` format:

```bash
nomos-provider-terraform-remote-state-v0.1.0-darwin-arm64.tar.gz
```

Extract with:
```bash
tar xzf nomos-provider-terraform-remote-state-v0.1.0-darwin-arm64.tar.gz
```

### Windows

Archives use `.zip` format:

```bash
nomos-provider-terraform-remote-state-v0.1.0-windows-amd64.zip
```

Extract with Windows Explorer or:
```powershell
Expand-Archive nomos-provider-terraform-remote-state-v0.1.0-windows-amd64.zip
```

## Checksums

Each release includes:

1. Individual `.sha256` files for each archive
2. A combined `SHA256SUMS.txt` file containing all checksums

Verify a download:

```bash
# Using the combined file
sha256sum -c SHA256SUMS.txt --ignore-missing

# Or verify individually
sha256sum -c nomos-provider-terraform-remote-state-v0.1.0-darwin-arm64.tar.gz.sha256
```

## Supported Platforms

| Operating System | Architecture | Platform String |
|-----------------|--------------|----------------|
| Linux | AMD64 | `linux-amd64` |
| macOS | AMD64 (Intel) | `darwin-amd64` |
| macOS | ARM64 (Apple Silicon) | `darwin-arm64` |
| Windows | AMD64 | `windows-amd64` |

## Build Configuration

All binaries are built with:

- **CGO_ENABLED=0**: Pure Go, no C dependencies
- **Static linking**: No external runtime dependencies
- **Stripped**: Debug symbols removed (`-s -w` ldflags) for smaller size
- **Version embedded**: Build version, commit, and timestamp included

## Manual Installation

If installing manually (without `nomos init`):

1. Download the appropriate archive for your platform
2. Extract the archive
3. Create the provider directory structure:
   ```bash
   mkdir -p ~/.nomos/providers/autonomous-bits/nomos-provider-terraform-remote-state/0.1.0/darwin-arm64
   ```
4. Copy the `provider` binary to the directory:
   ```bash
   cp nomos-provider-terraform-remote-state-v0.1.0-darwin-arm64/provider \
      ~/.nomos/providers/autonomous-bits/nomos-provider-terraform-remote-state/0.1.0/darwin-arm64/
   ```
5. Make it executable (Unix only):
   ```bash
   chmod +x ~/.nomos/providers/autonomous-bits/nomos-provider-terraform-remote-state/0.1.0/darwin-arm64/provider
   ```

## Troubleshooting

### "exec format error" on macOS/Linux

This error indicates the binary was not built correctly for your platform. Verify:

1. You downloaded the correct platform archive (check `uname -sm`)
2. The binary is executable: `chmod +x provider`
3. The binary matches your architecture:
   ```bash
   file provider
   # Should show: Mach-O 64-bit executable arm64 (for macOS ARM)
   # or: ELF 64-bit LSB executable, x86-64 (for Linux AMD64)
   ```

### Binary doesn't start

1. Check for missing dependencies (shouldn't happen with static builds):
   ```bash
   ldd provider  # Linux
   otool -L provider  # macOS
   ```

2. Verify the binary can execute:
   ```bash
   ./provider
   # Should print: PROVIDER_PORT=<port>
   ```

3. Check logs if running through Nomos tooling:
   ```bash
   nomos build --verbose
   ```

## CI/CD Pipeline

The release workflow automatically:

1. Builds binaries for all supported platforms
2. Packages them in the correct structure
3. Compresses archives (tar.gz for Unix, zip for Windows)
4. Generates checksums
5. Creates a GitHub release with all artifacts

No manual intervention required for releases - just push a version tag:

```bash
git tag v0.2.0
git push origin v0.2.0
```
