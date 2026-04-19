# Unit Tests (Clean Code, Ch. 9)

> **Language override — Go:** Do **not** use assertion libraries (testify, gomega). Use `cmp.Diff`
> and `t.Errorf`/`t.Fatalf` with descriptive messages. Test helpers call `t.Helper()`; `Test*`
> functions do not. Table-driven tests with named fields are the norm. See
> `devpilot-google-go-style` for details.

The Agile and TDD movements made unit tests routine. But **having tests isn't enough** — the tests
themselves need to be clean.

## The Three Laws of TDD

1. You may not write production code until you have written a failing unit test.
2. You may not write more of a unit test than is sufficient to fail (compile failures count).
3. You may not write more production code than is sufficient to pass the currently failing test.

The cycles are on the order of **seconds**. You end up with tests covering essentially all production
code, and a test suite you trust.

## Keeping Tests Clean

> Dirty tests are worse than no tests.

Test code is as important as production code. Not a second-class citizen. When tests are hard to
read or modify, you stop writing them. Then you stop refactoring. Then the code rots.

**If tests can't keep up with production changes, they lose their value.** Keep tests simple,
readable, and expressive.

## Tests Enable the -ilities

Tests are what make clean code possible:
- Fearless refactoring.
- Confident architectural change.
- Flexible, maintainable, reusable code.

Without a safety net, you can't change anything. With one, you can change everything.

## Clean Tests

The single most important quality: **readability**. A good test reads like a specification.

**Build a testing DSL.** Extract helpers so the test body says *what*, not *how*:

```java
public void testGetPageHieratchyAsXml() throws Exception {
    makePages("PageOne", "PageOne.ChildOne", "PageTwo");
    submitRequest("root", "type:pages");
    assertResponseIsXML();
    assertResponseContains("<name>PageOne</name>", "<name>PageTwo</name>", "<name>ChildOne</name>");
}
```

No API noise. No HTTP framework details. Just the test's intent.

## One Assert per Test

A controversial guideline: aim for **a single concept per test**, ideally a single assert.

```java
public void testGetPageHierarchyAsXml() { ... }
public void testGetPageHierarchyHasRightTags() { ... }
```

When a test fails, you know exactly what's broken. Tests are independent.

**More flexible interpretation:** One *concept* per test. Multiple asserts that together verify one
invariant are fine.

## F.I.R.S.T.

Clean tests follow five rules:

- **Fast.** Slow tests don't get run. Don't get run → rot.
- **Independent.** Tests don't depend on each other. Any test, any order. Failures are localized.
- **Repeatable.** In any environment — dev laptop, CI, plane with no network. Non-repeatable tests
  get disabled.
- **Self-Validating.** Pass or fail. No log-reading, no manual verification. Binary output.
- **Timely.** Write tests *just before* the production code they verify. After-the-fact tests are
  harder to write and rarely as thorough.

## Test Doubles

- **Stub:** canned return values.
- **Fake:** working implementation with shortcuts (in-memory DB).
- **Mock:** verifies interactions.
- **Spy:** records calls for later assertion.

Use sparingly. Heavy mocking often signals tight coupling in the system under test.

## Tests and Coupling

A test that breaks when you refactor without changing behavior is too coupled to implementation.
Tests should be coupled to **behavior**, not structure. When tests break in large numbers during
refactors, the SUT has leaky abstractions.

## Common Test Smells

| Smell | Fix |
|-------|-----|
| **Insufficient tests** — untested edge cases | Write them |
| **Skipped tests** (`@Ignore`, `it.skip`) | Fix or delete. Commented-out rots. |
| **Testing trivial behaviors** | Test business logic, not getters |
| **Hidden assumptions** (depends on test order) | Refactor for independence |
| **Slow suite** | Isolate slow tests in a separate suite; keep unit tests fast |
| **Non-deterministic** (flaky) | Remove timing and external deps |
| **Complex setup** | Extract DSL / factory helpers |

## Summary

Clean tests are an investment with compounding returns. Dirty tests are a liability with compounding
interest. The test suite is the safety net that makes clean production code possible — treat it
with at least as much care as the production code it protects.
