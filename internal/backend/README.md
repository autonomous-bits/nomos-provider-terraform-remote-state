# Backend System Documentation

Comprehensive guide for adding new backend types to the Nomos Terraform Remote State Provider.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Adding a New Backend](#adding-a-new-backend)
- [Complete Example: HTTP Backend](#complete-example-http-backend)
- [Configuration Handling](#configuration-handling)
- [Best Practices](#best-practices)
- [Error Codes Guide](#error-codes-guide)
- [Testing Your Backend](#testing-your-backend)
- [Reference](#reference)

## Overview

The backend system provides a pluggable architecture for accessing Terraform remote state files from various storage systems. The provider currently supports:

- **local**: Local filesystem
- **azurerm**: Azure Blob Storage

New backends can be added by implementing the `Backend` interface and registering a constructor function during package initialization.

### Why Add a Backend?

Add a backend when you need to:
- Support a new storage system (S3, GCS, HTTP, Consul, etcd, etc.)
- Implement custom authentication or access patterns
- Support organization-specific state storage solutions

### Architecture Principles

- **Single Responsibility**: Each backend handles one storage system
- **No Caching**: Fetch fresh state on every request
- **Environment Credentials**: Authentication via environment variables only
- **Thread Safety**: Backends must handle concurrent requests safely
- **Context Awareness**: Respect context cancellation and timeouts

## Architecture

### Backend Interface

All backends must implement the `Backend` interface:

```go
type Backend interface {
    // FetchState retrieves the Terraform state file from the backend.
    //
    // The context parameter enables cancellation and timeout control for
    // long-running fetch operations.
    //
    // Returns the parsed StateFile on success, or an error with appropriate
    // gRPC status code (NotFound, PermissionDenied, Unavailable, etc.).
    FetchState(ctx context.Context) (*state.StateFile, error)
}
```

**Key Points**:
- Context must be checked for cancellation before expensive operations
- Return fully parsed `*state.StateFile` objects
- Use appropriate error types (see [Error Codes Guide](#error-codes-guide))
- Must be thread-safe for concurrent calls

### Constructor Type

Backends register a constructor function with this signature:

```go
type Constructor func(ctx context.Context, config map[string]interface{}) (Backend, error)
```

**Responsibilities**:
- Extract and validate configuration fields
- Return descriptive errors for invalid configuration
- Create and initialize the backend instance
- Use context for initialization operations (if needed)

### Registration Pattern

Backends register themselves in an `init()` function:

```go
func init() {
    Register("mybackend", func(ctx context.Context, config map[string]interface{}) (Backend, error) {
        // Extract configuration
        // Validate configuration
        // Create backend
        return NewMyBackend(cfg)
    })
}
```

**How It Works**:
1. `init()` runs automatically when the package is imported
2. `Register()` adds the constructor to the global registry
3. The provider's Init RPC calls `GetBackend()` to create instances
4. Thread-safe registry allows concurrent access

### Factory Functions

The registry provides these functions:

- **`Register(backendType, constructor)`**: Register a new backend type (panics on duplicate)
- **`Get(backendType)`**: Retrieve a constructor by type (returns nil if not found)
- **`GetBackend(ctx, backendType, config)`**: Convenience function that retrieves and calls constructor
- **`List()`**: Get all registered backend type names

## Adding a New Backend

Follow these steps to add a new backend type:

### Step 1: Create a New File

Create a file named after your backend in `internal/backend/`:

```bash
touch internal/backend/s3.go
touch internal/backend/s3_test.go
```

**Naming Convention**: Use lowercase, match the Terraform backend name (e.g., `s3.go`, `gcs.go`, `http.go`).

### Step 2: Define Your Backend Struct

```go
package backend

import (
    "context"
    "errors"
    
    "github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/state"
)

// Sentinel errors for your backend
var (
    ErrS3BucketNotFound = errors.New("S3 bucket not found")
    ErrS3ObjectNotFound = errors.New("S3 object not found")
)

// S3BackendConfig holds configuration for the S3 backend.
type S3BackendConfig struct {
    // Bucket is the S3 bucket name.
    Bucket string
    
    // Key is the path to the state file within the bucket.
    Key string
    
    // Region is the AWS region (optional, defaults to us-east-1).
    Region string
}

// S3Backend implements the Backend interface for AWS S3.
type S3Backend struct {
    config S3BackendConfig
    // Add any clients or state needed
}
```

**Best Practices**:
- Define sentinel errors for common failure cases
- Create a config struct for type safety
- Document all fields with comments
- Keep structs minimal (no caching, no state)

### Step 3: Implement the Backend Interface

```go
// NewS3Backend creates a new S3 backend.
func NewS3Backend(cfg S3BackendConfig) (*S3Backend, error) {
    // Validate configuration
    if cfg.Bucket == "" {
        return nil, errors.New("bucket cannot be empty")
    }
    if cfg.Key == "" {
        return nil, errors.New("key cannot be empty")
    }
    
    // Set defaults
    if cfg.Region == "" {
        cfg.Region = "us-east-1"
    }
    
    return &S3Backend{
        config: cfg,
    }, nil
}

// FetchState retrieves the Terraform state file from S3.
func (b *S3Backend) FetchState(ctx context.Context) (*state.StateFile, error) {
    // Check context cancellation before starting
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Fetch data from S3 (pseudo-code)
    data, err := b.downloadFromS3(ctx)
    if err != nil {
        // Map S3 errors to appropriate error types
        if isNotFound(err) {
            return nil, fmt.Errorf("%w: %s/%s", ErrS3ObjectNotFound, b.config.Bucket, b.config.Key)
        }
        return nil, fmt.Errorf("failed to download from S3: %w", err)
    }
    
    // Check context again before parsing
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Parse the state file
    stateFile, err := state.ParseStateFile(data)
    if err != nil {
        return nil, fmt.Errorf("failed to parse state file: %w", err)
    }
    
    return stateFile, nil
}
```

**Critical Points**:
- Check `ctx.Done()` before expensive operations
- Use sentinel errors for common cases
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Parse state using `state.ParseStateFile()`

### Step 4: Create a Constructor Function

```go
func init() {
    Register("s3", func(ctx context.Context, config map[string]interface{}) (Backend, error) {
        // Extract bucket (required)
        bucketValue, ok := config["bucket"]
        if !ok {
            return nil, fmt.Errorf("missing required field: bucket")
        }
        bucket, ok := bucketValue.(string)
        if !ok {
            return nil, fmt.Errorf("bucket must be a string")
        }
        
        // Extract key (required)
        keyValue, ok := config["key"]
        if !ok {
            return nil, fmt.Errorf("missing required field: key")
        }
        key, ok := keyValue.(string)
        if !ok {
            return nil, fmt.Errorf("key must be a string")
        }
        
        // Extract region (optional)
        region := "us-east-1"
        if regionValue, ok := config["region"]; ok {
            if r, ok := regionValue.(string); ok && r != "" {
                region = r
            }
        }
        
        // Create backend
        return NewS3Backend(S3BackendConfig{
            Bucket: bucket,
            Key:    key,
            Region: region,
        })
    })
}
```

**Configuration Extraction Pattern**:
1. Check if field exists: `value, ok := config["field"]`
2. Type assert: `str, ok := value.(string)`
3. Validate: Check for empty strings, invalid values
4. Set defaults for optional fields
5. Return descriptive errors

### Step 5: Register Your Backend

The `init()` function from Step 4 already registers the backend. When your package is imported, the backend is automatically registered and ready to use.

**Important**: The registration happens during package initialization, so your backend is available immediately when the provider starts.

### Step 6: Add Tests

See [Testing Your Backend](#testing-your-backend) for comprehensive testing guidance.

## Complete Example: HTTP Backend

Here's a complete, working example of a simple HTTP backend that fetches state files from HTTP/HTTPS URLs.

### http.go

```go
package backend

import (
    "context"
    "errors"
    "fmt"
    "io"
    "net/http"
    "time"
    
    "github.com/autonomous-bits/nomos-provider-terraform-remote-state/internal/state"
)

func init() {
    Register("http", func(ctx context.Context, config map[string]interface{}) (Backend, error) {
        // Extract address (required)
        addressValue, ok := config["address"]
        if !ok {
            return nil, fmt.Errorf("missing required field: address")
        }
        address, ok := addressValue.(string)
        if !ok {
            return nil, fmt.Errorf("address must be a string")
        }
        
        // Extract timeout (optional, defaults to 30s)
        timeout := 30 * time.Second
        if timeoutValue, ok := config["timeout"]; ok {
            if t, ok := timeoutValue.(float64); ok && t > 0 {
                timeout = time.Duration(t) * time.Second
            }
        }
        
        return NewHTTPBackend(HTTPBackendConfig{
            Address: address,
            Timeout: timeout,
        })
    })
}

// Sentinel errors for HTTP backend
var (
    ErrHTTPNotFound     = errors.New("HTTP 404: state file not found")
    ErrHTTPUnauthorized = errors.New("HTTP 401: unauthorized")
    ErrHTTPForbidden    = errors.New("HTTP 403: forbidden")
    ErrInvalidURL       = errors.New("invalid URL")
)

// HTTPBackendConfig holds configuration for the HTTP backend.
type HTTPBackendConfig struct {
    // Address is the full URL to the state file.
    Address string
    
    // Timeout is the HTTP request timeout.
    Timeout time.Duration
}

// HTTPBackend implements the Backend interface for HTTP(S) endpoints.
type HTTPBackend struct {
    config HTTPBackendConfig
    client *http.Client
}

// NewHTTPBackend creates a new HTTP backend.
func NewHTTPBackend(cfg HTTPBackendConfig) (*HTTPBackend, error) {
    // Validate configuration
    if cfg.Address == "" {
        return nil, ErrInvalidURL
    }
    
    // Create HTTP client with timeout
    client := &http.Client{
        Timeout: cfg.Timeout,
    }
    
    return &HTTPBackend{
        config: cfg,
        client: client,
    }, nil
}

// FetchState retrieves the Terraform state file from an HTTP(S) endpoint.
func (b *HTTPBackend) FetchState(ctx context.Context) (*state.StateFile, error) {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Create HTTP request with context
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.config.Address, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create HTTP request: %w", err)
    }
    
    // Execute request
    resp, err := b.client.Do(req)
    if err != nil {
        // Check if context was cancelled
        if ctx.Err() != nil {
            return nil, ctx.Err()
        }
        return nil, fmt.Errorf("HTTP request failed: %w", err)
    }
    defer resp.Body.Close()
    
    // Check status code
    switch resp.StatusCode {
    case http.StatusOK:
        // Success, continue
    case http.StatusNotFound:
        return nil, fmt.Errorf("%w: %s", ErrHTTPNotFound, b.config.Address)
    case http.StatusUnauthorized:
        return nil, ErrHTTPUnauthorized
    case http.StatusForbidden:
        return nil, ErrHTTPForbidden
    default:
        return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
    }
    
    // Read response body
    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read HTTP response: %w", err)
    }
    
    // Check context again before parsing
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Parse state file
    stateFile, err := state.ParseStateFile(data)
    if err != nil {
        return nil, fmt.Errorf("failed to parse state file: %w", err)
    }
    
    return stateFile, nil
}
```

### http_test.go

```go
package backend

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)

func TestHTTPBackend_FetchState(t *testing.T) {
    validState := `{
        "version": 4,
        "terraform_version": "1.5.0",
        "serial": 1,
        "lineage": "test-lineage",
        "outputs": {
            "vpc_id": {
                "value": "vpc-12345",
                "type": "string",
                "sensitive": false
            }
        }
    }`
    
    tests := []struct {
        name       string
        setupFunc  func() *httptest.Server
        wantErr    bool
        errContains string
    }{
        {
            name: "successful fetch",
            setupFunc: func() *httptest.Server {
                return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    w.WriteHeader(http.StatusOK)
                    w.Write([]byte(validState))
                }))
            },
            wantErr: false,
        },
        {
            name: "404 not found",
            setupFunc: func() *httptest.Server {
                return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    w.WriteHeader(http.StatusNotFound)
                }))
            },
            wantErr:     true,
            errContains: "404",
        },
        {
            name: "401 unauthorized",
            setupFunc: func() *httptest.Server {
                return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    w.WriteHeader(http.StatusUnauthorized)
                }))
            },
            wantErr:     true,
            errContains: "401",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            server := tt.setupFunc()
            defer server.Close()
            
            backend, err := NewHTTPBackend(HTTPBackendConfig{
                Address: server.URL,
                Timeout: 5 * time.Second,
            })
            if err != nil {
                t.Fatalf("NewHTTPBackend() error = %v", err)
            }
            
            ctx := context.Background()
            got, err := backend.FetchState(ctx)
            
            if tt.wantErr {
                if err == nil {
                    t.Error("FetchState() error = nil, wantErr = true")
                }
                return
            }
            
            if err != nil {
                t.Errorf("FetchState() unexpected error = %v", err)
                return
            }
            
            if got == nil {
                t.Error("FetchState() returned nil state")
            }
        })
    }
}

func TestHTTPBackend_Registration(t *testing.T) {
    constructor := Get("http")
    if constructor == nil {
        t.Error("HTTP backend is not registered")
    }
}
```

## Configuration Handling

### Extracting Configuration Values

The configuration map contains `interface{}` values that must be type-asserted.

#### String Fields

```go
// Required string field
value, ok := config["field_name"]
if !ok {
    return nil, fmt.Errorf("missing required field: field_name")
}
str, ok := value.(string)
if !ok {
    return nil, fmt.Errorf("field_name must be a string")
}
if str == "" {
    return nil, fmt.Errorf("field_name cannot be empty")
}

// Optional string field with default
str := "default_value"
if value, ok := config["field_name"]; ok {
    if s, ok := value.(string); ok && s != "" {
        str = s
    }
}
```

#### Boolean Fields

```go
// Optional boolean field with default
flag := false
if value, ok := config["enable_feature"]; ok {
    if b, ok := value.(bool); ok {
        flag = b
    }
}
```

#### Numeric Fields

```go
// Optional int field with default
count := 10
if value, ok := config["max_retries"]; ok {
    // JSON unmarshaling produces float64 for numbers
    if f, ok := value.(float64); ok {
        count = int(f)
    }
}

// Optional duration field
timeout := 30 * time.Second
if value, ok := config["timeout_seconds"]; ok {
    if f, ok := value.(float64); ok && f > 0 {
        timeout = time.Duration(f) * time.Second
    }
}
```

### Validation Patterns

#### Range Validation

```go
if count < 0 || count > 100 {
    return nil, fmt.Errorf("max_retries must be between 0 and 100, got %d", count)
}
```

#### Format Validation

```go
import "regexp"

var bucketNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

if !bucketNameRegex.MatchString(bucketName) {
    return nil, fmt.Errorf("invalid bucket name format: %s", bucketName)
}
```

#### URL Validation

```go
import "net/url"

parsedURL, err := url.Parse(address)
if err != nil {
    return nil, fmt.Errorf("invalid URL: %w", err)
}
if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
    return nil, fmt.Errorf("URL must use http or https scheme")
}
```

### Error Handling

Always return descriptive errors that help users fix configuration issues:

```go
// Good: Specific error with field name
return nil, fmt.Errorf("missing required field: storage_account_name")

// Good: Error with validation details
return nil, fmt.Errorf("bucket name must be 3-63 characters, got %d", len(bucket))

// Bad: Generic error
return nil, errors.New("invalid configuration")

// Bad: Error without context
return nil, errors.New("validation failed")
```

## Best Practices

### Thread Safety

Backends must be safe for concurrent use:

```go
// ✅ Good: Stateless backend (no shared mutable state)
type S3Backend struct {
    config S3BackendConfig  // Immutable after construction
    client *s3.Client       // S3 client is thread-safe
}

// ❌ Bad: Mutable state without synchronization
type BadBackend struct {
    config      Config
    cachedState *state.StateFile  // Shared mutable state
}

// ✅ Good: If you must have mutable state, use sync.RWMutex
type CachingBackend struct {
    config Config
    mu     sync.RWMutex
    cache  map[string]*state.StateFile
}

func (b *CachingBackend) FetchState(ctx context.Context) (*state.StateFile, error) {
    b.mu.RLock()
    cached := b.cache[b.config.Key]
    b.mu.RUnlock()
    
    if cached != nil {
        return cached, nil
    }
    // ... fetch and cache
}
```

**Best Practice**: Design stateless backends. Let the provider handle any necessary caching.

### Context Propagation

Always check context cancellation:

```go
func (b *Backend) FetchState(ctx context.Context) (*state.StateFile, error) {
    // Check at the start
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Check before expensive operations
    data, err := b.downloadData(ctx)  // Use context in I/O operations
    if err != nil {
        return nil, err
    }
    
    // Check again before CPU-intensive operations
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    return state.ParseStateFile(data)
}
```

**When to check**:
- Beginning of function
- Before I/O operations
- After I/O operations (before parsing)
- In long-running loops

### Security Considerations

#### Credentials from Environment Only

**Never** accept credentials in configuration:

```go
// ❌ Bad: Credentials in config
type BadConfig struct {
    AccessKey string
    SecretKey string
}

// ✅ Good: Credentials from environment
func NewS3Backend(cfg S3BackendConfig) (*S3Backend, error) {
    // SDK reads AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY from environment
    awsConfig, err := config.LoadDefaultConfig(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }
    
    client := s3.NewFromConfig(awsConfig)
    return &S3Backend{config: cfg, client: client}, nil
}
```

#### Input Validation

Validate all configuration inputs:

```go
// ✅ Good: Validate and sanitize inputs
if cfg.Path == "" {
    return nil, ErrInvalidPath
}

// Resolve to absolute path to prevent directory traversal
absPath, err := filepath.Abs(cfg.Path)
if err != nil {
    return nil, fmt.Errorf("failed to resolve path: %w", err)
}
cfg.Path = absPath
```

#### Sensitive Data

Don't log sensitive information:

```go
// ❌ Bad: Logs sensitive data
log.Printf("Fetching state from bucket: %s, key: %s, credentials: %s", bucket, key, creds)

// ✅ Good: Logs non-sensitive metadata only
log.Printf("Fetching state from bucket: %s, key: %s", bucket, key)
```

### Error Handling

Use sentinel errors for common cases:

```go
var (
    ErrNotFound = errors.New("state file not found")
    ErrInvalidConfig = errors.New("invalid configuration")
)

// Wrap sentinel errors with context
if fileNotExists {
    return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
}
```

Check error types in tests:

```go
if !errors.Is(err, ErrNotFound) {
    t.Errorf("expected ErrNotFound, got: %v", err)
}
```

### Testing Strategies

1. **Unit Tests**: Test configuration parsing, validation, error cases
2. **Integration Tests**: Test against real backend (with build tag)
3. **Mock Tests**: Test context cancellation, timeouts, edge cases
4. **Table-Driven Tests**: Cover multiple scenarios efficiently

See [Testing Your Backend](#testing-your-backend) for details.

## Error Codes Guide

Map backend errors to appropriate gRPC status codes. The provider handles the conversion, but your errors should indicate the appropriate type.

### NotFound

**When**: Resource doesn't exist (file, blob, bucket, etc.)

```go
var ErrStateFileNotFound = errors.New("state file not found")

// Return with context
return nil, fmt.Errorf("%w: %s", ErrStateFileNotFound, path)
```

### PermissionDenied

**When**: Authentication or authorization fails

```go
var ErrAuthenticationFailed = errors.New("authentication failed")

if resp.StatusCode == 401 || resp.StatusCode == 403 {
    return nil, ErrAuthenticationFailed
}
```

### InvalidArgument

**When**: Configuration is invalid or malformed

```go
if cfg.Bucket == "" {
    return nil, fmt.Errorf("bucket cannot be empty")
}

if !bucketNameRegex.MatchString(cfg.Bucket) {
    return nil, fmt.Errorf("invalid bucket name format")
}
```

### FailedPrecondition

**When**: State version is unsupported (< 4)

```go
// This is handled by state.ParseStateFile(), but you can check explicitly:
if stateFile.Version < 4 {
    return nil, state.ErrUnsupportedVersion
}
```

### Unavailable

**When**: Network errors, timeouts, service unavailable

```go
if err := b.client.Connect(); err != nil {
    return nil, fmt.Errorf("backend unavailable: %w", err)
}

if ctx.Err() == context.DeadlineExceeded {
    return nil, fmt.Errorf("request timeout: %w", ctx.Err())
}
```

### Internal

**When**: Unexpected errors, parsing failures, bugs

```go
data, err := io.ReadAll(resp.Body)
if err != nil {
    return nil, fmt.Errorf("failed to read response: %w", err)
}

stateFile, err := state.ParseStateFile(data)
if err != nil {
    return nil, fmt.Errorf("failed to parse state: %w", err)
}
```

### Error Mapping Example

```go
func (b *S3Backend) FetchState(ctx context.Context) (*state.StateFile, error) {
    obj, err := b.client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(b.config.Bucket),
        Key:    aws.String(b.config.Key),
    })
    if err != nil {
        // Map S3 errors to appropriate types
        var nsk *types.NoSuchKey
        if errors.As(err, &nsk) {
            // Maps to NotFound
            return nil, fmt.Errorf("%w: %s/%s", ErrS3ObjectNotFound, b.config.Bucket, b.config.Key)
        }
        
        var nsb *types.NoSuchBucket
        if errors.As(err, &nsb) {
            // Maps to NotFound
            return nil, fmt.Errorf("%w: %s", ErrS3BucketNotFound, b.config.Bucket)
        }
        
        // Network/timeout errors map to Unavailable
        if ctx.Err() != nil {
            return nil, ctx.Err()
        }
        
        // Default to Internal
        return nil, fmt.Errorf("S3 error: %w", err)
    }
    
    // ... rest of implementation
}
```

## Testing Your Backend

### Unit Test Structure

```go
//go:build !integration

package backend

import (
    "context"
    "testing"
)

func TestMyBackend_FetchState(t *testing.T) {
    tests := []struct {
        name       string
        config     MyBackendConfig
        setupFunc  func() // Setup test fixtures
        wantErr    bool
        errType    error  // Sentinel error to check
    }{
        {
            name: "successful fetch",
            config: MyBackendConfig{
                // Valid config
            },
            setupFunc: func() {
                // Create test files, mock servers, etc.
            },
            wantErr: false,
        },
        {
            name: "not found error",
            config: MyBackendConfig{
                // Config pointing to non-existent resource
            },
            wantErr: true,
            errType: ErrNotFound,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.setupFunc != nil {
                tt.setupFunc()
            }
            
            backend, err := NewMyBackend(tt.config)
            if err != nil {
                t.Fatalf("NewMyBackend() error = %v", err)
            }
            
            ctx := context.Background()
            got, err := backend.FetchState(ctx)
            
            if tt.wantErr {
                if err == nil {
                    t.Error("FetchState() error = nil, wantErr = true")
                    return
                }
                if tt.errType != nil && !errors.Is(err, tt.errType) {
                    t.Errorf("FetchState() error type = %v, want %v", err, tt.errType)
                }
                return
            }
            
            if err != nil {
                t.Errorf("FetchState() unexpected error = %v", err)
                return
            }
            
            if got == nil {
                t.Error("FetchState() returned nil state")
            }
        })
    }
}
```

### Test Cases to Cover

1. **Happy Path**: Valid configuration, successful fetch
2. **Not Found**: Non-existent file/blob/object
3. **Invalid Configuration**: Missing required fields, invalid formats
4. **Authentication Failure**: Invalid credentials
5. **Network Errors**: Timeout, connection refused
6. **Context Cancellation**: Cancelled context
7. **Invalid State Format**: Corrupted data, unsupported version
8. **Edge Cases**: Empty outputs, large files, special characters

### Testing Context Cancellation

```go
func TestMyBackend_ContextCancellation(t *testing.T) {
    backend, err := NewMyBackend(validConfig)
    if err != nil {
        t.Fatalf("NewMyBackend() error = %v", err)
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    cancel()  // Cancel immediately
    
    _, err = backend.FetchState(ctx)
    if err == nil {
        t.Error("FetchState() should return error with cancelled context")
    }
    
    if !errors.Is(err, context.Canceled) {
        t.Errorf("FetchState() error = %v, want context.Canceled", err)
    }
}
```

### Testing Registration

```go
func TestMyBackend_Registration(t *testing.T) {
    constructor := Get("mybackend")
    if constructor == nil {
        t.Error("MyBackend is not registered")
    }
    
    // Test creating through factory
    ctx := context.Background()
    backend, err := GetBackend(ctx, "mybackend", map[string]interface{}{
        "required_field": "value",
    })
    if err != nil {
        t.Errorf("GetBackend() error = %v", err)
    }
    if backend == nil {
        t.Error("GetBackend() returned nil backend")
    }
}
```

### Integration Tests

Use build tags to separate integration tests:

```go
//go:build integration

package backend

import (
    "context"
    "os"
    "testing"
)

func TestS3Backend_Integration(t *testing.T) {
    // Skip if AWS credentials not available
    if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
        t.Skip("AWS credentials not configured")
    }
    
    backend, err := NewS3Backend(S3BackendConfig{
        Bucket: os.Getenv("TEST_S3_BUCKET"),
        Key:    "test-state/terraform.tfstate",
        Region: "us-east-1",
    })
    if err != nil {
        t.Fatalf("NewS3Backend() error = %v", err)
    }
    
    ctx := context.Background()
    state, err := backend.FetchState(ctx)
    if err != nil {
        t.Errorf("FetchState() error = %v", err)
    }
    
    if state == nil {
        t.Error("FetchState() returned nil state")
    }
}
```

Run integration tests:

```bash
go test -tags=integration -v ./internal/backend/
```

## Reference

### Backend Interface

- **File**: [backend.go](backend.go)
- **Interface**: `Backend`
- **Method**: `FetchState(ctx context.Context) (*state.StateFile, error)`

### Existing Implementations

- **Local Backend**: [local.go](local.go) - Simple filesystem implementation
- **Azure Backend**: [azurerm.go](azurerm.go) - Cloud storage with authentication
- **Local Tests**: [local_test.go](local_test.go) - Example test patterns
- **Azure Tests**: [azurerm_test.go](azurerm_test.go) - Cloud backend testing
- **Registry Tests**: [backend_test.go](backend_test.go) - Registration and factory testing

### State Package

- **File**: [../state/types.go](../state/types.go)
- **File**: [../state/parser.go](../state/parser.go)
- **Function**: `ParseStateFile(data []byte) (*StateFile, error)`
- **Error**: `state.ErrUnsupportedVersion` - State version < 4

### Registry Functions

```go
// Register a new backend type (call in init())
func Register(backendType string, constructor Constructor)

// Get a constructor by type
func Get(backendType string) Constructor

// Create a backend instance (convenience function)
func GetBackend(ctx context.Context, backendType string, config map[string]interface{}) (Backend, error)

// List all registered backend types
func List() []string
```

### Constructor Signature

```go
type Constructor func(ctx context.Context, config map[string]interface{}) (Backend, error)
```

### Common Error Types

```go
// From state package
state.ErrUnsupportedVersion  // State version < 4

// Define sentinel errors in your backend
var (
    ErrNotFound     = errors.New("resource not found")
    ErrUnauthorized = errors.New("authentication failed")
    ErrInvalidConfig = errors.New("invalid configuration")
)
```

### Build Tags

```go
//go:build !integration    // Unit tests (default)
//go:build integration     // Integration tests (requires -tags=integration)
```

---

## Quick Start Checklist

Adding a new backend? Follow this checklist:

- [ ] Create `<backend>.go` in `internal/backend/`
- [ ] Define config struct with validation
- [ ] Implement `Backend` interface (`FetchState` method)
- [ ] Create constructor function
- [ ] Register in `init()` function
- [ ] Define sentinel errors
- [ ] Check context cancellation (start and before expensive ops)
- [ ] Map errors to appropriate types (NotFound, PermissionDenied, etc.)
- [ ] Use environment variables for credentials
- [ ] Create `<backend>_test.go`
- [ ] Test: happy path, not found, invalid config, context cancellation
- [ ] Test: registration via `Get()` and `GetBackend()`
- [ ] Add integration tests with `//go:build integration` tag
- [ ] Run `go test ./internal/backend/` (80%+ coverage required)
- [ ] Run `make verify` (all checks must pass)
- [ ] Document configuration fields in comments
- [ ] Update this README with backend-specific notes (if needed)

---

**Questions?** Open an issue or check the [CONTRIBUTING.md](../../CONTRIBUTING.md) guide.
