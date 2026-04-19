---
name: devpilot-clean-code-principles
description: >
  Use when writing, reviewing, or refactoring code and judging quality at the level of naming,
  function size, comments, error handling, class design, test quality, or code smells.
  Language-agnostic; defer to language-specific style skills (e.g. devpilot-google-go-style) when they conflict.
license: Complete terms in LICENSE.txt
---

# Clean Code Principles

Distilled from Robert C. Martin's *Clean Code: A Handbook of Agile Software Craftsmanship* (2008),
incorporating contributions from Kent Beck, Ward Cunningham, Michael Feathers, and others.

This skill captures **language-agnostic** principles for writing code that is easy to read, change,
and extend. When a language-specific style guide is loaded (e.g. Google Go Style), it takes
precedence on syntax and idiom; this skill is still authoritative on higher-level judgment (naming
intent, function design, abstractions, code smells). If no language-specific skill is loaded, the
Conflict Resolution table below encodes the defaults — they are not "placeholders," they are the
rulings to apply.

## Core Principles

**You read code 10x more than you write it.** Optimize for the reader, not the author.

**The Boy Scout Rule:** Leave the code cleaner than you found it. Small, continuous improvements
prevent decay. Don't require permission to rename a variable or extract a function.

**Clean code does one thing well.** Functions, classes, modules — each should have a single,
clearly-named responsibility. If you need "and" to describe what it does, split it.

**Meaningful names > comments.** A good name makes a comment unnecessary. If you need a comment to
explain what a variable or function is, rename it.

**"Does it do one thing?" is the real test — not line count.** If a function cannot be further
decomposed into named sub-functions that each carry meaning, leave it. A 35-line state machine
that does one thing is fine; a 12-line function doing three things is not. Line-count heuristics
(below) are tripwires, not verdicts.

## Conflict Resolution

This skill and a language-specific style skill may disagree. **Language idiom always wins.** If
`devpilot-google-go-style` is loaded and conflicts with Clean Code, follow the Go skill.

If **no language-specific skill is loaded** for the code you're reviewing, apply these defaults:

| Topic | Go default | TypeScript default | Source of truth |
|---|---|---|---|
| Error mechanism | `(T, error)` returns | `throw` + `try/catch` | language idiom, not Clean Code |
| Accessor naming | no `Get` prefix (`User.Name()`) | `getName()` or property accessors | language idiom |
| Doc comments on exports | **required** (godoc) | TSDoc for public APIs only | language idiom |
| Null handling | `(T, bool)` or `(T, error)` — never zero-as-sentinel | avoid `null`; use union types | Clean Code (don't return null) + language form |
| Class vs composition | structs + packages (no classes) | classes OK | language shape |
| Test assertion style | no assertion libs; `cmp.Diff` + `t.Errorf` | Jest `expect` | language community norm |

**Spirit of Clean Code survives the mechanism.** Whichever error mechanism you use: keep the happy
path flat, attach context, never swallow errors, never signal absence with a zero value.

### Go: `(T, bool)` vs `(T, error)` — which to return

- **`(T, bool)`** when absence is an expected, non-exceptional outcome that the caller is designed
  to handle: map lookups (`v, ok := m[k]`), cache probes, parsing "is this a recognized variant?"
- **`(T, error)`** when the operation can fail for reasons the caller can't predict or should log:
  I/O, network, parsing untrusted input, DB queries. If the caller might want to distinguish *why*
  it failed, use `error`.
- **Both** (`(T, bool, error)`) is a smell — pick the dominant dimension.
- **Rule of thumb:** if "not found" is a normal branch in the caller's logic → `bool`. If "not found"
  likely indicates a bug or infrastructure problem → `error`.

## Quick Reference — Principles by Category

### Naming (Ch. 2)

- Use **intention-revealing** names: `elapsedTimeInDays`, not `d`.
- Avoid **disinformation**: don't call it `accountList` if it isn't a List.
- Make **meaningful distinctions**: `ProductData` vs `ProductInfo` is noise.
- **Pronounceable** and **searchable** names. Single letters only for tiny scopes.
- **Class/type names** are nouns (`Customer`, `Account`). **Method/function names** are verbs (`save`, `deletePage`).
- **Accessor prefix (`get*`/`set*`) is language-dependent.** TS/Java/C# accept `getName()`; **Go drops the `Get`** — `User.Name()`, not `User.GetName()`. See Conflict Resolution above.
- **One word per concept**: don't mix `fetch`, `retrieve`, `get` for the same operation.
- **Initialism casing (`ID`, `URL`, `HTTP`)** matters in Go — never `Id`, `Url`, `Http` on exported names.
- Don't **encode types** in names (`strName`, `m_user`) — modern IDEs show types.

**Open `references/naming.md` when:** arguing about a specific name (is `accountList` disinforming? is `Processor` a weasel word?), resolving package-name vs type-name stuttering, or deciding between multiple factory-function names.

### Functions (Ch. 3)

- **Do one thing — primary test.** A function does one thing when you can't extract another meaningful
  named function from it. *This is the rule; size is a consequence.*
- **Size is a tripwire, not a verdict.** If a function exceeds ~20 lines, re-read it looking for a
  hidden sub-concept. If none exists after an honest look, leave it — splitting a genuinely atomic
  function invents noise.
- **One level of abstraction per function.** Don't mix high-level policy with low-level details.
- **Few arguments.** 0 > 1 > 2 > 3. Three or more is a smell — introduce an object / options struct.
- **No flag arguments.** `render(true)` means the function does two things. Split it.
- **No side effects.** `checkPassword()` that also initializes a session is a lie.
- **Command-Query Separation.** Functions either *do* something or *answer* something, never both.
- **Flat happy path > nested error handling.** TS: `throw`. Go: early-return `if err != nil`. Either
  way, don't pyramid. See Conflict Resolution for which mechanism to use.
- **DRY.** Duplication is the root of most maintenance pain.

**Open `references/functions.md` when:** a function exceeds ~20 lines and you're deciding whether to split, evaluating an argument list of 3+ params, deciding between throw vs early-return, or looking for the full error-code-vs-exception comparison with both TS and Go examples.

### Comments (Ch. 4)

- **Inline/explanatory comments are a near-failure.** Every one is an admission the code couldn't
  speak. Rename or extract first.
- **Godoc / TSDoc on exported APIs are REQUIRED, not a smell.** Go enforces this for
  every exported symbol; TS does it for public library surfaces. This is the opposite of
  the "comments are failure" rule — doc comments *are* the contract.
- **Don't comment bad code — rewrite it.**
- **Good inline comments:** legal headers, intent that isn't derivable from code, warnings of
  consequences, TODOs with ticket + owner.
- **Bad comments:** redundant (`// increment i`), misleading, mandated-by-policy noise, journal
  entries, commented-out code (delete it — VCS remembers).

**Open `references/comments.md` when:** weighing whether a specific comment is "good" (intent, warning, TODO) or "bad" (redundant, journal, noise), or if someone wants to add a position marker / closing-brace comment.

### Formatting (Ch. 5)

- **Vertical openness separates concepts.** Blank line between functions, between groups of related lines.
- **Vertical density implies association.** Related lines stay together.
- **Declare variables close to their use.**
- **Dependent functions should be near.** Caller above callee when possible.
- **Team style trumps personal preference.** Pick one, enforce with formatter.

**Open `references/formatting.md` when:** debating file size / vertical openness, deciding where instance variables live, or the team doesn't yet have a formatter config.

### Objects and Data Structures (Ch. 6)

- **Objects hide data, expose behavior.** Data structures expose data, have no behavior.
- **Don't mix.** Hybrids (half-object, half-struct) get the worst of both.
- **Law of Demeter:** a method should only call methods of its class, its parameters, objects it
  creates, or its direct fields — not navigate through chains (`a.getB().getC().doStuff()`).

**Open `references/objects-and-data.md` when:** deciding "is this type data or object?", reviewing a deep chain like `a.b().c().d()`, or weighing whether to expose struct fields vs add methods.

### Error Handling (Ch. 7)

- **Keep happy path flat.** TS: `throw` + one `try/catch` at the boundary. Go: early-return
  `if err != nil { return err }` guards. **Match language idiom; see Conflict Resolution.** Never
  nest error checks into a pyramid.
- **Scaffold the error boundary first** — write `try/catch` or the error-return scope before the
  happy path.
- **Provide context.** TS: subclass `Error` with fields. Go: `fmt.Errorf("op: %w", err)`. Never
  `catch { /* empty */ }` or `result, _ := ...` without a comment explaining why.
- **Don't return null / don't return zero-as-sentinel.** TS: union types or throw. Go: `(T, bool)`
  or `(T, error)`.
- **Wrap third-party APIs** in your own error types so callers match on one thing, not five.

**Open `references/error-handling.md` when:** the code returns `null`/zero-as-sentinel, wraps a third-party library's errors, or you're choosing between throw / Result / (T, error) / Null Object. Has full TS + Go side-by-side examples.

### Boundaries (Ch. 8)

- **Isolate third-party code.** Write an adapter/wrapper so your code depends on your interface, not theirs.
- **Learning tests:** write tests against the third-party API to pin down its behavior and catch changes on upgrade.
- **Depend on code you control** at internal boundaries.

**Open `references/boundaries.md` when:** third-party types are leaking across your codebase, you're wrapping an external API, or considering learning tests for a library upgrade.

### Unit Tests (Ch. 9)

- **Three Laws of TDD:** (1) don't write production code until a failing test requires it,
  (2) write only enough test to fail, (3) write only enough production code to pass.
- **F.I.R.S.T.** Tests must be Fast, Independent, Repeatable, Self-validating, Timely.
- **One assert per test** — or at least one concept per test.
- **Test code is first-class.** Apply the same quality bar as production code.
- **Domain-specific testing language.** Build helpers so tests read like specifications.

**Open `references/unit-tests.md` when:** a test asserts many unrelated things, the test suite is flaky or slow, you're designing a test DSL, or weighing mocks vs fakes.

### Classes (Ch. 10)

- **Small.** Measured by responsibilities, not lines. Name should describe the responsibility — if it
  takes "and" or weasel words (`Processor`, `Manager`), split.
- **Single Responsibility Principle:** one reason to change.
- **High cohesion.** Most methods use most fields. Low cohesion → extract classes.
- **Open-Closed Principle:** open to extension (subclass, compose) but closed to modification.
- **Dependency Inversion:** depend on abstractions, not concretions.

**Open `references/classes.md` when:** splitting a god class, debating SRP boundaries, applying OCP (new report type, new strategy), or reviewing a `Manager`/`Processor` suspect.

### Systems (Ch. 11)

- **Separate construction from use.** Main() and factories build the graph; business code just uses it.
- **Dependency Injection** — don't let objects create their collaborators.
- **Cross-cutting concerns** (logging, transactions, security) via aspects or decorators, not scattered code.
- **Grow systems incrementally.** Start with the simplest architecture; refactor as needs emerge.

**Open `references/systems.md` when:** removing lazy singletons, wiring up DI, introducing cross-cutting concerns (transactions, logging, auth), or deciding between aspects/decorators vs middleware.

### Concurrency (Ch. 13)

- **Concurrency is a decoupling strategy** — separates *what* from *when*, but adds complexity.
- **Keep concurrency code separate from other code.**
- **Limit shared data.** Copy data when possible; use thread-local.
- **Understand your library** — `ExecutorService`, `ConcurrentHashMap`, channels. Don't reinvent.
- **Know the models:** Producer-Consumer, Readers-Writers, Dining Philosophers.

**Open `references/concurrency.md` when:** introducing goroutines/async, narrowing a critical section, debugging a flaky test that may be a race, or designing graceful shutdown.

### Code Smells and Heuristics (Ch. 17)

A checklist of specific anti-patterns to scan for during review: comments smells, environment smells,
function smells (too many arguments, dead parameters, flag args), general smells (duplication, magic
numbers, inconsistency, artificial coupling), and test smells (insufficient, disabled, redundant).

**Open `references/smells-and-heuristics.md` when:** doing a PR review scan, you suspect a smell but can't name it, or investigating duplication / dead code / feature envy / inappropriate statics.

## Review Checklist

When reviewing code, walk down this list in order:

1. **Names** — Do they reveal intent? Any disinformation, noise words, or encodings?
2. **Functions** — Any >20 lines doing more than one thing? Flag args? >3 params?
3. **Duplication** — Copy-pasted logic? Parallel `switch`es on the same type?
4. **Comments** — Any inline comments that could be replaced by a rename or extraction?
5. **Doc comments on exports** — Are godoc / TSDoc present on public APIs? (Required in Go; expected on public library surfaces in TS.)
6. **Error handling** — Null/zero-as-sentinel returned or passed? Errors swallowed? Nested error pyramids instead of flat happy path?
7. **Classes/types** — Does each have a single responsibility? Any `Manager`/`Processor`/`Util`?
8. **Tests** — One concept per test? Fast and independent? Cover edge cases?
9. **Boundaries** — Third-party types leaking across the codebase?

## Common Mistakes

| Mistake | Why it happens | Fix |
|---------|----------------|-----|
| "Small refactor" adds a flag arg | Seems cheaper than splitting | Extract a new function; callers are explicit |
| Comment explains what code does | Feels helpful | Rename variable/function; delete comment |
| Utility class grows unbounded | "Just one more helper" | Each helper belongs on a domain object or its own class |
| Deep getter chains (`a.getB().getC()...`) | Treating objects as data | Tell, don't ask: move the behavior to the owner |
| Returning `null` for "not found" | Matches return type easily | Return `Optional`, empty collection, or throw |
| Tests with many asserts | "Related, might as well group" | Split into one-concept-per-test |
| `// TODO` never gets done | No owner, no deadline | Add ticket link + owner, or do it now |

## Red Flags — STOP and Reconsider

- You need "and" to describe what a function/class does.
- You're adding a comment to explain a variable name.
- You're about to copy-paste more than 3 lines.
- A function exceeds ~20 lines **and** you can still extract a named sub-function from it. (Pure
  size alone isn't the flag — see Functions: *"size is a tripwire, not a verdict."*)
- A class has methods that use disjoint sets of fields.
- A test name starts with `test1`, `test2`, …
- A parameter is a `boolean` flag.
- You're returning `null`/`None`/zero-as-sentinel or checking `== null`.
- **Composite smell:** a single function that violates CQS *and* has a flag arg *and* does
  multiple things *and* returns null. Don't patch one dimension — redesign.

## When NOT to Use This Skill

- **Language-specific syntax or idioms** — use the language's style skill (e.g. `devpilot-google-go-style`).
- **Project-specific conventions** — those belong in `CLAUDE.md`.
- **Mechanical formatting** — run the formatter; don't reason about spaces.
- **When Clean Code conflicts with language idioms** — idioms win. See Conflict Resolution above for
  the concrete table.
- **Security, correctness, performance, and concurrency bugs** — out of scope. Clean Code covers
  *readability* and *maintainability*, not SQL injection, XSS, auth, data races, or algorithmic
  correctness. A reviewer should combine this skill with security and correctness checks, not
  substitute it for them.

### Languages with no dedicated style skill

For Rust, Java, C#, or anything else without a loaded `devpilot-*-style` skill:
1. Apply this skill's **principles** (naming, SRP, flag args, null returns, DRY).
2. For **mechanism** questions (error handling, accessor naming, concurrency primitives, test
   framework), follow that language's community idiom — do NOT transplant Java/Go/TS conventions.
3. Consult the Conflict Resolution table above for the common languages; for others, default to
   the language's own style guide (PEP 8, Rustfmt + Clippy, etc.) over Clean Code's mechanism-level
   advice.

