# Configuration Schema Changes

**Feature**: Separate Backend Type from Provider Type  
**Branch**: `002-separate-backend-type`  
**Date**: 2025-12-31

## Overview

This document specifies the changes to the provider configuration schema passed via the Init RPC `config` parameter.

---

## Schema Changes Summary

| Field | Old Behavior | New Behavior | Status |
|-------|-------------|--------------|--------|
| `type` | Used for backend selection ("local", "azurerm") | **IGNORED** - Reserved for CLI provider source | ❌ **REMOVED** |
| `backend_type` | Not present | Used for explicit backend selection ("local", "azurerm") | ✅ **ADDED** |
| Auto-detection | Not supported | Infers backend type from configuration keys when `backend_type` omitted | ✅ **ADDED** |

---

## Complete Configuration Schemas

### Local Backend

#### With Explicit backend_type

```json
{
  "backend_type": "local",
  "path": string,
  "workspace": string (optional, default: "default")
}
```

#### With Auto-detection

```json
{
  "path": string,
  "workspace": string (optional, default: "default")
}
```
*Auto-detects: `backend_type = "local"`*

**Field Descriptions**:
- `backend_type` (string, optional): Explicit backend type identifier. If omitted, auto-detected from `path` presence.
- `path` (string, **required**): Absolute or relative path to Terraform state file
- `workspace` (string, optional): Terraform workspace name. Default: "default"

**Validation Rules**:
- `path` MUST be non-empty string
- `path` MUST NOT contain path traversal patterns (`..`)
- `workspace` MUST match pattern `^[a-zA-Z0-9_-]+$` if provided
- If `backend_type = "local"` explicitly, Azure keys (`storage_account_name`, `container_name`) MUST NOT be present

---

### Azure Blob Storage Backend

#### With Explicit backend_type

```json
{
  "backend_type": "azurerm",
  "storage_account_name": string,
  "container_name": string,
  "key": string,
  "workspace": string (optional, default: "default")
}
```

#### With Auto-detection

```json
{
  "storage_account_name": string,
  "container_name": string,
  "key": string,
  "workspace": string (optional, default: "default")
}
```
*Auto-detects: `backend_type = "azurerm"`*

**Field Descriptions**:
- `backend_type` (string, optional): Explicit backend type identifier. If omitted, auto-detected from Azure key presence.
- `storage_account_name` (string, **required**): Azure storage account name
- `container_name` (string, **required**): Azure blob container name
- `key` (string, **required**): Blob name/path within container
- `workspace` (string, optional): Terraform workspace name. Default: "default"

**Validation Rules**:
- `storage_account_name` MUST match Azure naming rules (3-24 chars, lowercase alphanumeric)
- `container_name` MUST match Azure naming rules (3-63 chars, lowercase alphanumeric + hyphens)
- `key` MUST be non-empty string
- `key` MUST NOT contain path traversal patterns or invalid blob name characters
- If `backend_type = "azurerm"` explicitly, local key (`path`) MUST NOT be present

---

## Auto-detection Rules

### Detection Algorithm

```
IF "backend_type" field is present and non-empty:
    USE explicit value
ELSE:
    IF "path" field is present:
        IF Azure keys (storage_account_name OR container_name) are also present:
            RETURN ERROR: Ambiguous configuration
        ELSE:
            DETECT: backend_type = "local"
    ELSE IF "storage_account_name" AND "container_name" are present:
        DETECT: backend_type = "azurerm"
    ELSE:
        RETURN ERROR: Cannot detect backend type
```

### Detection Examples

| Configuration Keys | Detection Result |
|-------------------|------------------|
| `path` only | ✅ `backend_type = "local"` |
| `storage_account_name` + `container_name` | ✅ `backend_type = "azurerm"` |
| `storage_account_name` + `container_name` + `key` | ✅ `backend_type = "azurerm"` |
| `path` + `storage_account_name` | ❌ ERROR: Ambiguous |
| `path` + `container_name` | ❌ ERROR: Ambiguous |
| `storage_account_name` only (no `container_name`) | ❌ ERROR: Cannot detect |
| `container_name` only (no `storage_account_name`) | ❌ ERROR: Cannot detect |
| No recognizable keys | ❌ ERROR: Cannot detect |

---

## Validation Error Messages

### InvalidArgument Errors

| Error Scenario | gRPC Code | Error Message Template |
|---------------|-----------|------------------------|
| Missing backend_type and cannot auto-detect | `InvalidArgument` | `backend_type not specified and cannot be auto-detected. Supported backends: local (requires 'path'), azurerm (requires 'storage_account_name' and 'container_name')` |
| Ambiguous configuration | `InvalidArgument` | `ambiguous backend configuration: both local 'path' and Azure keys present. Specify 'backend_type' explicitly.` |
| Unsupported backend_type | `InvalidArgument` | `unsupported backend type: "{value}" (allowed: local, azurerm)` |
| backend_type conflicts with config keys | `InvalidArgument` | `backend_type is '{type}' but conflicting keys present: {conflicting_keys}` |
| Missing required field for detected backend | `InvalidArgument` | `missing required field '{field}' for {backend_type} backend` |

---

## Migration Guide (Not Applicable)

**Status**: Provider not yet in production use

No migration required. New configurations should use `backend_type` or rely on auto-detection.

---

## Comparison with Old Schema

### Local Backend Example

**Old (Incorrect)**:
```json
{
  "type": "local",
  "path": "./terraform.tfstate"
}
```
*Problem: `type` field conflicts with CLI provider source identification*

**New (Explicit)**:
```json
{
  "backend_type": "local",
  "path": "./terraform.tfstate"
}
```

**New (Auto-detected)**:
```json
{
  "path": "./terraform.tfstate"
}
```

### Azure Backend Example

**Old (Incorrect)**:
```json
{
  "type": "azurerm",
  "storage_account_name": "myaccount",
  "container_name": "tfstate",
  "key": "terraform.tfstate"
}
```
*Problem: `type` field conflicts with CLI provider source identification*

**New (Explicit)**:
```json
{
  "backend_type": "azurerm",
  "storage_account_name": "myaccount",
  "container_name": "tfstate",
  "key": "terraform.tfstate"
}
```

**New (Auto-detected)**:
```json
{
  "storage_account_name": "myaccount",
  "container_name": "tfstate",
  "key": "terraform.tfstate"
}
```

---

## Configuration in Context (.csl file)

### Full Nomos Configuration Example

```yaml
# Nomos configuration file
source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'  # ← CLI uses this
  version: '0.1.0'
  backend_type: 'local'                                          # ← Provider uses this
  path: './terraform.tfstate'
```

**Field Routing**:
- `type` field: Used by Nomos CLI to locate and download provider binary
- `backend_type`, `path`, etc.: Passed to provider via Init RPC `config` parameter

---

## JSON Schema (Informative)

### Local Backend Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "backend_type": {
      "type": "string",
      "enum": ["local"],
      "description": "Explicit backend type (optional if 'path' is present)"
    },
    "path": {
      "type": "string",
      "minLength": 1,
      "description": "Path to Terraform state file (required)"
    },
    "workspace": {
      "type": "string",
      "pattern": "^[a-zA-Z0-9_-]+$",
      "default": "default",
      "description": "Terraform workspace name (optional)"
    }
  },
  "required": ["path"],
  "additionalProperties": false
}
```

### Azure Backend Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "backend_type": {
      "type": "string",
      "enum": ["azurerm"],
      "description": "Explicit backend type (optional if Azure keys present)"
    },
    "storage_account_name": {
      "type": "string",
      "pattern": "^[a-z0-9]{3,24}$",
      "description": "Azure storage account name (required)"
    },
    "container_name": {
      "type": "string",
      "pattern": "^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$",
      "description": "Azure blob container name (required)"
    },
    "key": {
      "type": "string",
      "minLength": 1,
      "description": "Blob name/path (required)"
    },
    "workspace": {
      "type": "string",
      "pattern": "^[a-zA-Z0-9_-]+$",
      "default": "default",
      "description": "Terraform workspace name (optional)"
    }
  },
  "required": ["storage_account_name", "container_name", "key"],
  "additionalProperties": false
}
```

---

## Summary

- **BREAKING CHANGE**: `type` field no longer used for backend selection
- **NEW FIELD**: `backend_type` explicitly specifies backend ("local" or "azurerm")
- **NEW FEATURE**: Auto-detection infers backend type from configuration keys when `backend_type` omitted
- **VALIDATION**: Clear error messages for ambiguous or incomplete configurations
- **COMPATIBILITY**: No backward compatibility required (provider not in use)
