# Error Handling (Clean Code, Ch. 7)

> **Language override:** Clean Code's exception-first stance reflects Java. Go, Rust, and Zig use
> explicit error returns idiomatically — when a language-specific style skill is loaded, follow it.
> The *spirit* of this chapter (keep the happy path flat, attach context, don't discard errors, don't
> return null/sentinels) still applies regardless of mechanism.

Error handling is important, but if it obscures logic, it's wrong.

## Use Exceptions Rather Than Return Codes

Return codes clutter the caller with nested checks and mix happy path with error handling:

```java
public class DeviceController {
    public void sendShutDown() {
        DeviceHandle handle = getHandle(DEV1);
        if (handle != DeviceHandle.INVALID) {
            retrieveDeviceRecord(handle);
            if (record.getStatus() != DEVICE_SUSPENDED) {
                pauseDevice(handle);
                clearDeviceWorkQueue(handle);
                closeDevice(handle);
            } else {
                logger.log("Device suspended. Unable to shut down");
            }
        } else {
            logger.log("Invalid handle for: " + DEV1.toString());
        }
    }
}
```

Cleaner with exceptions:
```java
public void sendShutDown() {
    try {
        tryToShutDown();
    } catch (DeviceShutDownError e) {
        logger.log(e);
    }
}

private void tryToShutDown() throws DeviceShutDownError {
    DeviceHandle handle = getHandle(DEV1);
    DeviceRecord record = retrieveDeviceRecord(handle);
    pauseDevice(handle);
    clearDeviceWorkQueue(handle);
    closeDevice(handle);
}
```

**Language note:** In Go and Rust, explicit error returns are idiomatic. Apply the spirit: use
early-return guards, don't nest, keep happy path flat.

## Write Your Try-Catch-Finally Statement First

When writing code that could throw, **start with the `try`/`catch`**. It defines the scope and forces
you to think about what the caller sees in the error case. Treat it like TDD: demarcate the scope,
then fill in the happy path.

## Use Unchecked Exceptions (Where Applicable)

In Java, checked exceptions were an experiment that lost. They break encapsulation (a change in a
low-level method forces signature changes all the way up) and violate Open-Closed. Use unchecked
exceptions for normal application code. Other languages (C#, Python, JS) already use unchecked.

## Provide Context with Exceptions

Each exception should carry enough context to diagnose:
- The operation that failed.
- The inputs that caused the failure.
- What the code was trying to do.

```java
throw new InvalidOrderException(
    "Cannot process order " + orderId + ": customer " + customerId + " has no active payment method"
);
```

Include stack traces; don't swallow them.

## Define Exception Classes by Caller Needs

Group exceptions by how the caller will handle them, not by source:

```java
// Bad — caller must catch many types that all get handled the same way
try {
    port.open();
} catch (DeviceResponseException e) { reportPortError(e); log(e); }
  catch (ATM1212UnlockedException e) { reportPortError(e); log(e); }
  catch (GMXError e) { reportPortError(e); log(e); }

// Good — wrap third-party exceptions in one meaningful type
public class LocalPort {
    public void open() {
        try { innerPort.open(); }
        catch (Exception e) { throw new PortDeviceFailure(e); }
    }
}
```

## Define the Normal Flow

Don't sprinkle `try`/`catch` for special cases the business logic has to handle. Use the **Special
Case pattern** — an object that represents the "nothing interesting happened" case:

```java
// Special case: MealExpensesNotFound returns a per-diem amount
public class PerDiemMealExpenses implements MealExpenses {
    public int getTotal() { return perDiemDefault; }
}

// Caller has no special case to worry about:
MealExpenses expenses = expenseReportDAO.getMeals(employee.getID());
total += expenses.getTotal();
```

## Don't Return Null

`null` is a lie. Callers must remember to check; eventually someone forgets; NullPointerException
ensues.

Alternatives:
- **Empty collections** — always return `Collections.emptyList()` instead of null lists.
- **`Optional<T>`** / `Option` / `Maybe` types — force callers to deal with absence.
- **Throw** when absence is truly exceptional.
- **Null Object** / **Special Case** pattern for common "absent" cases.

```java
// Bad
List<Employee> employees = getEmployees();
if (employees != null) {
    for (Employee e : employees) totalPay += e.getPay();
}

// Good
for (Employee e : getEmployees()) totalPay += e.getPay();
// getEmployees always returns at least empty list
```

## Don't Pass Null

Passing `null` into methods is even worse: every method now needs defensive checks at every
parameter. Use method overloading, sentinel objects, or throw an IllegalArgumentException at API
boundaries — but make it clear in the API that null is not acceptable.

## Summary

Clean code is **readable** and **robust**. Error handling is part of the job — but if the error
handling buries the logic, the error handling is wrong. Separate concerns: happy path in one place,
error response in another.
