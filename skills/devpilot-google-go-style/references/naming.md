# Naming

## Underscores

Names in Go should not contain underscores, with three exceptions:
1. Package names imported only by generated code
2. Test/Benchmark/Example function names in `*_test.go`
3. Low-level libraries interoperating with the OS or cgo

Filenames are not Go identifiers and may contain underscores.

## Package Names

- Only lowercase letters and numbers (e.g., `k8s`, `oauth2`)
- Multi-word names stay lowercase and unbroken: `tabwriter` not `tabWriter`
- Avoid names likely shadowed by variables: `usercount` not `count`
- No underscores in package names
- Avoid: `util`, `utility`, `common`, `helper`, `model`, `testhelper`

## Receiver Names

Short (1-2 letters), abbreviation of the type, consistent across all methods:

| Long Name | Better |
|-----------|--------|
| `func (tray Tray)` | `func (t Tray)` |
| `func (info *ResearchInfo)` | `func (ri *ResearchInfo)` |
| `func (this *ReportWriter)` | `func (w *ReportWriter)` |
| `func (self *Scanner)` | `func (s *Scanner)` |

## Constant Names

MixedCaps always. Never `MAX_PACKET_SIZE` or `kMaxBufferSize`.

```go
// Good:
const MaxPacketSize = 512

// Bad:
const MAX_PACKET_SIZE = 512
const kMaxBufferSize = 1024
```

Name constants by their role, not their value. If a constant has no role beyond its value,
don't define it.

## Initialisms

Initialisms keep consistent case: all upper or all lower.

| English | Exported | Unexported | Wrong |
|---------|----------|------------|-------|
| XML API | `XMLAPI` | `xmlAPI` | `XmlApi`, `XMLApi` |
| iOS | `IOS` | `iOS` | `Ios` |
| gRPC | `GRPC` | `gRPC` | `Grpc` |
| DDoS | `DDoS` | `ddos` | `DDOS` |
| ID | `ID` | `id` | `Id` |
| DB | `DB` | `db` | `Db` |

## Getters

No `Get`/`get` prefix. Start with the noun: `Counts()` not `GetCounts()`.

For expensive operations (complex computation, remote calls), use `Compute` or `Fetch` to
signal it may take time or fail.

## Variable Names

Length proportional to scope, inversely proportional to usage frequency:
- Small scope (1-7 lines): single letter or short name
- Medium scope (8-15 lines): short descriptive word
- Large scope (15-25 lines): more descriptive
- Very large scope (25+ lines): fully descriptive

Rules:
- Single-word names like `count` or `options` are good starting points
- Add words to disambiguate: `userCount` vs `projectCount`
- Don't drop letters: `Sandbox` not `Sbx`
- Omit types from names: `users` not `userSlice`, `count` not `numUsers`
- Omit words clear from context

### Single-Letter Variables

Acceptable for:
- Method receivers (1-2 letters)
- `r` for `io.Reader`/`*http.Request`, `w` for `io.Writer`/`http.ResponseWriter`
- `i` for loop indices, `x`/`y` for coordinates
- Short-scope loop variables: `for _, n := range nodes`

## Repetition

### Package vs Exported Symbol

Reduce redundancy between package name and exported symbol:

| Repetitive | Better |
|-----------|--------|
| `widget.NewWidget` | `widget.New` |
| `widget.NewWidgetWithName` | `widget.NewWithName` |
| `db.LoadFromDatabase` | `db.Load` |

If a package exports only one type named after the package, use `New` for the constructor.

### Variable vs Type

Don't encode types in variable names:

| Repetitive | Better |
|-----------|--------|
| `var numUsers int` | `var users int` |
| `var nameString string` | `var name string` |
| `var primaryProject *Project` | `var primary *Project` |

Exception: two versions in scope — use `limitRaw`/`limit` or `limitStr`/`limit`.

### External Context

Don't repeat package path, struct name, or type name in the surrounding context:

```go
// Bad: In package "ads/targeting/revenue/reporting"
type AdsTargetingRevenueReport struct{}

// Good:
type Report struct{}

// Bad:
func (p *Project) ProjectName() string
// Good:
func (p *Project) Name() string

// Bad: In package "sqldb"
type DBConnection struct{}
// Good:
type Connection struct{}
```

Evaluate repetition from the caller's perspective. Repetition compounds:

```go
// Bad: repetition everywhere
func (db *DB) UserCount() (userCount int, err error) {
    var userCountInt64 int64
    if dbLoadError := db.LoadFromDatabase("count(distinct users)", &userCountInt64); dbLoadError != nil {
        return 0, fmt.Errorf("failed to load user count: %s", dbLoadError)
    }
    userCount = int(userCountInt64)
    return userCount, nil
}

// Good: clean, no stuttering
func (db *DB) UserCount() (int, error) {
    var count int64
    if err := db.Load("count(distinct users)", &count); err != nil {
        return 0, fmt.Errorf("failed to load user count: %s", err)
    }
    return int(count), nil
}
```

## Function Names (Best Practice)

Avoid redundancy — don't repeat type, package, or receiver info:
- In package `yamlconfig`: use `Parse()` not `ParseYAMLConfig()`
- Constructor for single exported type: use `New()` not `NewWidget()`

## Test Doubles

Helper packages for test doubles: append "test" to the package name.
- Use `StubService` when multiple types need doubles
- Use `Stub` when testing a single type
