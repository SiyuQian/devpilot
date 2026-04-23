# Error Handling (Clean Code, Ch. 7)

> **TypeScript vs Go — two mechanisms, same principles.** TS uses thrown exceptions; Go uses
> explicit `(T, error)` returns. The *spirit* of this chapter — keep the happy path flat, attach
> context, never discard errors, never return null/sentinels — applies to both. Every example
> below shows the idiomatic form in each language.

Error handling is important, but if it obscures logic, it's wrong.

## Keep the Happy Path Flat

Nested error checks mix happy path with error handling:

```ts
// ❌ Bad (TypeScript) — nested guards
function sendShutDown() {
  const handle = getHandle(DEV1);
  if (handle !== INVALID) {
    const record = retrieveDeviceRecord(handle);
    if (record.status !== DEVICE_SUSPENDED) {
      pauseDevice(handle);
      clearDeviceWorkQueue(handle);
      closeDevice(handle);
    } else {
      logger.warn("Device suspended. Unable to shut down");
    }
  } else {
    logger.warn(`Invalid handle for: ${DEV1}`);
  }
}
```

**In TypeScript — throw and catch at one layer above:**
```ts
function sendShutDown() {
  try {
    tryToShutDown();
  } catch (err) {
    logger.error(err);
  }
}

function tryToShutDown() {
  const handle = getHandle(DEV1);                 // throws InvalidHandleError
  const record = retrieveDeviceRecord(handle);
  if (record.status === DEVICE_SUSPENDED) {
    throw new DeviceShutDownError("device suspended");
  }
  pauseDevice(handle);
  clearDeviceWorkQueue(handle);
  closeDevice(handle);
}
```

**In Go — early-return guards keep the happy path at the left margin:**
```go
func sendShutDown(ctx context.Context) error {
    handle, err := getHandle(ctx, dev1)
    if err != nil {
        return fmt.Errorf("get handle: %w", err)
    }
    record, err := retrieveDeviceRecord(ctx, handle)
    if err != nil {
        return fmt.Errorf("retrieve record: %w", err)
    }
    if record.Status == deviceSuspended {
        return errors.New("device suspended")
    }
    pauseDevice(handle)
    clearDeviceWorkQueue(handle)
    return closeDevice(handle)
}
```

Same result either way: the reader sees a flat sequence of steps.

## Write Your Error-Handling Scaffold First

When writing code that can fail, start with the error boundary — the `try`/`catch` in TS, or the
`if err != nil { return ... }` pattern in Go. Scoping the failure first forces you to think about
what the caller sees in the error case, then fill in the happy path.

## Don't Silently Swallow Errors

```ts
// ❌ Bad — caller never learns of the failure
try {
  await publishEvent(evt);
} catch {
  // empty
}
```

```go
// ❌ Bad — discarded with _
result, _ := doThing()
```

If you truly must ignore an error, write a comment explaining **why** and log it.

## Provide Context

Every error should let the caller diagnose what failed.

```ts
// TypeScript — subclass Error and attach context
class OrderProcessError extends Error {
  constructor(readonly orderId: string, readonly customerId: string, cause?: unknown) {
    super(`cannot process order ${orderId}: customer ${customerId} has no active payment method`);
    this.cause = cause;
  }
}
throw new OrderProcessError(orderId, customerId, err);
```

```go
// Go — wrap with %w so errors.Is / errors.As work
return fmt.Errorf("process order %s for customer %s: %w", orderID, customerID, err)
```

## Define Error Types by Caller Needs

Group errors by how the caller will handle them, not by their source. Wrap third-party errors in one
meaningful type you own.

```ts
// TypeScript — wrap external library errors at the boundary
class PortDeviceFailure extends Error {
  constructor(cause: unknown) {
    super("port device failure");
    this.cause = cause;
  }
}

class LocalPort {
  constructor(private readonly inner: ACMEPort) {}
  open() {
    try { this.inner.open(); }
    catch (err) { throw new PortDeviceFailure(err); }
  }
}
```

```go
// Go — wrap at the boundary with a sentinel error the caller can match
var ErrPortDevice = errors.New("port device failure")

type LocalPort struct{ inner *acme.Port }

func (p *LocalPort) Open() error {
    if err := p.inner.Open(); err != nil {
        return fmt.Errorf("%w: %v", ErrPortDevice, err)
    }
    return nil
}
```

Callers now write one `catch` / one `errors.Is(err, ErrPortDevice)` instead of matching every
third-party type.

## Define the Normal Flow — Special Case / Null Object

Don't sprinkle `try`/`catch` (or `if err != nil`) for cases the business logic has to handle
normally. Model "nothing interesting" as a real value.

```ts
// TypeScript — Null Object represents "no custom meals, use per diem"
interface MealExpenses { total(): number; }

class PerDiemMealExpenses implements MealExpenses {
  total() { return PER_DIEM_DEFAULT; }
}

function mealsFor(employee: Employee): MealExpenses {
  return expenseReportRepo.findMeals(employee.id) ?? new PerDiemMealExpenses();
}

// Caller has no special case
total += mealsFor(employee).total();
```

```go
// Go — same idea with an interface and a zero-cost concrete type
type MealExpenses interface{ Total() int }

type perDiemMealExpenses struct{}
func (perDiemMealExpenses) Total() int { return perDiemDefault }

func MealsFor(ctx context.Context, id EmployeeID) MealExpenses {
    m, err := repo.FindMeals(ctx, id)
    if err != nil || m == nil {
        return perDiemMealExpenses{}
    }
    return m
}
```

## Don't Return Null (or Naked Zero Values as "Not Found")

**In TypeScript:** `null`/`undefined` returned from functions metastasize — every caller must
remember to check. Prefer:
- Empty collections (`[]`, `new Map()`) over `null`.
- Discriminated unions (`{ ok: true, value } | { ok: false, error }`) for operations that may fail.
- Throw when absence is truly exceptional.
- Null Object for common "absent" cases (see above).

```ts
// ❌ Bad
function getEmployees(): Employee[] | null { ... }
const emps = getEmployees();
if (emps !== null) { for (const e of emps) total += e.pay; }

// ✅ Good — always return at least an empty array
function getEmployees(): Employee[] { ... }
for (const e of getEmployees()) total += e.pay;
```

**In Go:** Don't use a zero value to mean "not found" — use `(T, bool)` or `(T, error)` so the
caller can't accidentally treat the zero as real data.

```go
// ❌ Bad — callers can't distinguish "empty string" from "not found"
func LookupName(id UserID) string

// ✅ Good
func LookupName(id UserID) (name string, ok bool)
```

**Nil slices are fine in Go** — `var xs []T` reads the same as `[]T{}` via `range` and `len`. Don't
design APIs that distinguish the two.

## Don't Pass Null / Unintended Nil

**TypeScript:** Turn on `strictNullChecks`. If a parameter can be absent, make the type explicit
(`T | undefined`) — don't expect callers to read your mind. Validate at API boundaries.

**Go:** A nil pointer or nil interface passed where the callee dereferences it is a panic. Document
which parameters may be nil; at API boundaries, return an error rather than dereferencing blindly.

## Summary

Clean code is **readable** and **robust**. Error handling is part of the job — but if the error
handling buries the logic, the error handling is wrong. Separate concerns: happy path flat; errors
wrapped with context; absence modeled explicitly, not as null/zero.
