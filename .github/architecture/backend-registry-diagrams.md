# Backend Registry Architecture Diagram

## Component Relationships

```
┌─────────────────────────────────────────────────────────────────────┐
│                     internal/backend Package                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────────────────────────────────────────┐     │
│  │              Registry Infrastructure                      │     │
│  │                                                           │     │
│  │  registry map[string]BackendConstructor                  │     │
│  │  registryMu sync.RWMutex                                 │     │
│  │                                                           │     │
│  │  Register(type, constructor) ─┐                          │     │
│  │  Get(type) BackendConstructor │                          │     │
│  │  List() []string              │                          │     │
│  └───────────────────────────────┼───────────────────────────┘     │
│                                  │                                 │
│  ┌───────────────────────────────▼──────────────────────────┐     │
│  │           BackendConstructor Type                        │     │
│  │  func(ctx, config) (Backend, error)                      │     │
│  └──────────────────────────────────────────────────────────┘     │
│                                  │                                 │
│                                  │ returns                         │
│                                  ▼                                 │
│  ┌──────────────────────────────────────────────────────────┐     │
│  │              Backend Interface                           │     │
│  │  FetchState(ctx) (*StateFile, error)                     │     │
│  └──────────────────────────────────────────────────────────┘     │
│                                  │                                 │
│                                  │ implemented by                  │
│           ┌──────────────────────┴────────────────────┐           │
│           │                                            │           │
│  ┌────────▼─────────┐                     ┌───────────▼────────┐  │
│  │  LocalBackend    │                     │  AzureBackend      │  │
│  │                  │                     │                    │  │
│  │  init() {        │                     │  init() {          │  │
│  │    Register(     │                     │    Register(       │  │
│  │      "local",    │                     │      "azurerm",    │  │
│  │      constructor)│                     │      constructor)  │  │
│  │  }               │                     │  }                 │  │
│  └──────────────────┘                     └────────────────────┘  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              │ used by
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    internal/provider Package                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────────────────────────────────────────┐     │
│  │           Provider Service (Init RPC)                    │     │
│  │                                                           │     │
│  │  1. constructor := backend.Get(cfg.Type())               │     │
│  │     if constructor == nil {                              │     │
│  │       return "unsupported type: available [...]"         │     │
│  │     }                                                     │     │
│  │                                                           │     │
│  │  2. b, err := constructor(ctx, cfg.Raw())                │     │
│  │     if err != nil {                                      │     │
│  │       return "failed to create backend"                  │     │
│  │     }                                                     │     │
│  │                                                           │     │
│  │  3. instances[alias] = &instance{backend: b}             │     │
│  └──────────────────────────────────────────────────────────┘     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Sequence Diagram: Backend Registration and Usage

### Phase 1: Package Initialization (Before main())

```
┌──────────┐         ┌─────────────┐         ┌──────────────┐
│  Go      │         │  local.go   │         │  azurerm.go  │
│ Runtime  │         │   init()    │         │    init()    │
└────┬─────┘         └──────┬──────┘         └──────┬───────┘
     │                      │                       │
     │ ┌──────────────────────────────────────────────────┐
     │ │ Package initialization phase (sequential)        │
     │ └──────────────────────────────────────────────────┘
     │                      │                       │
     ├─ Run init() ────────►│                       │
     │                      │                       │
     │              Register("local", constructor) ─┼──────────┐
     │                      │                       │          │
     │                      │◄──────────────────────┼──────────┘
     │                      │ "local" registered    │
     │                      │                       │
     ├─ Run init() ────────────────────────────────►│
     │                      │                       │
     │              Register("azurerm", constructor)┼──────────┐
     │                      │                       │          │
     │                      │                       │◄─────────┘
     │                      │           "azurerm" registered
     │                      │                       │
     │ Registry now contains:                      │
     │   "local" → localConstructor                │
     │   "azurerm" → azureConstructor              │
     │                      │                       │
     ├─ Start main() ──────┤                       │
     │                      │                       │
     ▼                      ▼                       ▼
```

### Phase 2: Runtime (Provider Init RPC)

```
┌─────────┐    ┌──────────┐    ┌──────────┐    ┌────────────┐
│  Nomos  │    │ Provider │    │ Registry │    │  Backend   │
│ Client  │    │   Init   │    │          │    │ (local)    │
└────┬────┘    └────┬─────┘    └────┬─────┘    └─────┬──────┘
     │              │               │                 │
     │ InitRequest  │               │                 │
     │ (type=local) │               │                 │
     ├─────────────►│               │                 │
     │              │               │                 │
     │              │ Get("local")  │                 │
     │              ├──────────────►│                 │
     │              │               │                 │
     │              │◄──────────────┤                 │
     │              │ constructor   │                 │
     │              │               │                 │
     │              │ constructor(ctx, config)        │
     │              ├─────────────────────────────────►│
     │              │               │                 │
     │              │               │   NewLocalBackend
     │              │               │                 │
     │              │◄─────────────────────────────────┤
     │              │ Backend instance                │
     │              │               │                 │
     │◄─────────────┤               │                 │
     │ InitResponse │               │                 │
     │              │               │                 │
     ▼              ▼               ▼                 ▼
```

## Data Flow: Config Map → Backend Instance

```
┌─────────────────────────────────────────────────────────────────┐
│                   gRPC InitRequest                              │
│   alias: "prod"                                                 │
│   config: {                                                     │
│     type: "local"                                               │
│     path: "/path/to/terraform.tfstate"                          │
│     workspace: "production"                                     │
│   }                                                             │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ cfg.Type() = "local"
                         │ cfg.Raw() = { path: "...", workspace: "..." }
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                  backend.Get("local")                           │
│  Returns: BackendConstructor function                           │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ constructor(ctx, cfg.Raw())
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│              Local Backend Constructor                          │
│                                                                 │
│  1. Extract path from config["path"]                            │
│     ✓ Type assert to string                                     │
│     ✓ Validate not empty                                        │
│                                                                 │
│  2. Extract workspace from config["workspace"]                  │
│     ✓ Type assert to string (optional)                          │
│     ✓ Default to "default" if missing                           │
│                                                                 │
│  3. Call NewLocalBackend(LocalBackendConfig{                    │
│       Path: path,                                               │
│       Workspace: workspace,                                     │
│     })                                                          │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ Returns Backend instance or error
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    LocalBackend Instance                        │
│  config: {                                                      │
│    Path: "/path/to/terraform.tfstate"                           │
│    Workspace: "production"                                      │
│  }                                                              │
│  FetchState(ctx) method ready to call                           │
└─────────────────────────────────────────────────────────────────┘
```

## Thread Safety Model

```
┌──────────────────────────────────────────────────────────────────┐
│                    Registry Access Patterns                      │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  INIT TIME (before main)                                         │
│  ─────────────────────────                                       │
│                                                                  │
│  Thread: Single (Go runtime initializer)                         │
│  Operation: Register()                                           │
│  Lock: registryMu.Lock() / Unlock()                              │
│  Frequency: Once per backend (2 times for MVP)                   │
│                                                                  │
│  ┌─────────────┐                                                 │
│  │ local.init()│──┐                                              │
│  └─────────────┘  │                                              │
│                   │ Lock                                         │
│  ┌─────────────┐  ├──► registry["local"] = constructor          │
│  │azurerm.init()  │                                              │
│  └─────────────┘  │ Unlock                                       │
│                   │                                              │
│                   │ Lock                                         │
│                   └──► registry["azurerm"] = constructor         │
│                                                                  │
│                      Unlock                                      │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  RUNTIME (after server starts)                                   │
│  ─────────────────────────────                                   │
│                                                                  │
│  Threads: Multiple (concurrent gRPC requests)                    │
│  Operation: Get(), List()                                        │
│  Lock: registryMu.RLock() / RUnlock()                            │
│  Frequency: Many times per second                                │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  Request 1  │  │  Request 2  │  │  Request 3  │             │
│  │ (goroutine) │  │ (goroutine) │  │ (goroutine) │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
│         │ RLock          │ RLock          │ RLock              │
│         ├────────────────┼────────────────┼─────► registry     │
│         │                │                │                     │
│         │                │                │    (all can read    │
│         │                │                │     concurrently)   │
│         │                │                │                     │
│         │ RUnlock        │ RUnlock        │ RUnlock            │
│         └────────────────┴────────────────┴──────              │
│                                                                  │
│  Benefits:                                                       │
│  • Multiple readers don't block each other                       │
│  • No lock contention on hot path                               │
│  • Registry never modified after init                           │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

## Error Handling Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                  Error Handling Boundaries                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  INIT TIME ERRORS (programming errors)                          │
│  ──────────────────────────────────                             │
│                                                                 │
│  Register("", constructor)                                      │
│    └─► PANIC: "backendType cannot be empty"                    │
│                                                                 │
│  Register("local", nil)                                         │
│    └─► PANIC: "constructor cannot be nil"                      │
│                                                                 │
│  Register("local", c1)                                          │
│  Register("local", c2)  // duplicate                            │
│    └─► PANIC: "backend type local already registered"          │
│                                                                 │
│  Result: Server fails to start (desired!)                       │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  RUNTIME ERRORS (user errors)                                   │
│  ─────────────────────────                                      │
│                                                                 │
│  backend.Get("s3")  // not registered                           │
│    └─► return nil                                               │
│        Provider checks: status.InvalidArgument                  │
│        Message: "unsupported type: s3 (available: [local...])"  │
│                                                                 │
│  constructor(ctx, {"path": 123})  // wrong type                 │
│    └─► return error                                             │
│        Provider maps: status.InvalidArgument                    │
│        Message: "failed to create backend: path must be string" │
│                                                                 │
│  constructor(ctx, {})  // missing required field                │
│    └─► return error                                             │
│        Provider maps: status.InvalidArgument                    │
│        Message: "failed to create backend: missing field: path" │
│                                                                 │
│  Result: User sees helpful error, can fix config                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Key Architectural Benefits

```
┌───────────────────────────────────────────────────────────────┐
│                BEFORE: Switch Statement                       │
├───────────────────────────────────────────────────────────────┤
│  Provider (internal/provider/provider.go)                     │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ switch cfg.Type() {                                     │  │
│  │ case "local":                                           │  │
│  │   b, err = createLocalBackend(cfg)    ◄──────────┐     │  │
│  │ case "azurerm":                                  │     │  │
│  │   b, err = createAzureBackend(ctx, cfg) ◄────────┤     │  │
│  │ default:                                         │     │  │
│  │   return "unsupported"                           │     │  │
│  │ }                                                │     │  │
│  │                                                  │     │  │
│  │ createLocalBackend(cfg) { ... }  ────────────────┘     │  │
│  │ createAzureBackend(ctx, cfg) { ... } ─────────────────┘│  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                               │
│  Problems:                                                    │
│  ✗ Provider must know about all backends                     │
│  ✗ Adding backend requires modifying provider                │
│  ✗ Backend logic leaks into provider package                 │
│  ✗ Tight coupling, hard to test                              │
└───────────────────────────────────────────────────────────────┘

                            │
                            │ Refactor to Registry Pattern
                            ▼

┌───────────────────────────────────────────────────────────────┐
│                 AFTER: Registry Pattern                       │
├───────────────────────────────────────────────────────────────┤
│  Provider (internal/provider/provider.go)                     │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ constructor := backend.Get(cfg.Type())              │  │
│  │ if constructor == nil {                             │  │
│  │   return "unsupported (available: [...])"           │  │
│  │ }                                                   │  │
│  │ b, err := constructor(ctx, cfg.Raw())               │  │
│  └─────────────────────────────────────────────────────────┘  │
│                          │                                    │
│                          │ No backend-specific code!          │
│                          ▼                                    │
│  Backend (internal/backend/backend.go)                        │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ registry = map[string]BackendConstructor{}          │  │
│  │                                                     │  │
│  │ Register(type, constructor)                         │  │
│  │ Get(type) BackendConstructor                        │  │
│  │ List() []string                                     │  │
│  └─────────────────────────────────────────────────────────┘  │
│           ▲                                  ▲                │
│           │                                  │                │
│  ┌────────┴─────────┐              ┌────────┴────────┐       │
│  │ local.go         │              │ azurerm.go      │       │
│  │ init() {         │              │ init() {        │       │
│  │   Register(...)  │              │   Register(...) │       │
│  │ }                │              │ }               │       │
│  └──────────────────┘              └─────────────────┘       │
│                                                               │
│  Benefits:                                                    │
│  ✓ Backends self-register (no provider changes)              │
│  ✓ Provider independent of backend implementations           │
│  ✓ Backend logic stays in backend package                    │
│  ✓ Loose coupling via interface                              │
│  ✓ Easy to add new backends (just create file with init)     │
│  ✓ Testable in isolation                                     │
└───────────────────────────────────────────────────────────────┘
```

## Files Modified/Created

```
Project Structure:
├── internal/backend/
│   └── backend.go ───────────────► MODIFIED (architecture added)
│       • BackendConstructor type
│       • registry, registryMu variables
│       • Register(), Get(), List() signatures
│       • Package documentation with examples
│
└── .github/architecture/ ─────────► NEW DIRECTORY
    ├── backend-registry-pattern.md
    │   • Complete architecture specification
    │   • Component design
    │   • Thread safety model
    │   • Error handling strategy
    │   • Migration path
    │
    ├── backend-registry-implementation-guide.md
    │   • Task breakdown (A5-A9)
    │   • Code snippets for implementer
    │   • Common pitfalls
    │   • Testing checklist
    │
    ├── backend-registry-diagrams.md ──► THIS FILE
    │   • Visual component relationships
    │   • Sequence diagrams
    │   • Data flow diagrams
    │   • Thread safety model
    │   • Before/after comparison
    │
    └── phase4-us2-architecture-complete.md
        • Status summary
        • Deliverables
        • Design decisions
        • Validation results
        • Next steps
```
