---
name: devpilot-clean-code-principles
description: >
  Use when writing, reviewing, or refactoring code in any language and judging quality —
  naming, function size, comments, error handling, class design, test quality, or code smells.
  Triggers on requests like "is this clean?", "review this code", "refactor this", "improve readability",
  "too complex", "code smell", or when reviewing a PR without a language-specific style skill.
  Language-agnostic; defer to language-specific style skills (e.g. devpilot-google-go-style) when they conflict.
---

# Clean Code Principles

Distilled from Robert C. Martin's *Clean Code: A Handbook of Agile Software Craftsmanship* (2008),
incorporating contributions from Kent Beck, Ward Cunningham, Michael Feathers, and others.

This skill captures **language-agnostic** principles for writing code that is easy to read, change,
and extend. When a language-specific style guide is loaded (e.g. Google Go Style), follow that guide
for syntax-level decisions; use this skill for higher-level judgment about naming, function design,
abstractions, and code smells.

## Core Principles

**You read code 10x more than you write it.** Optimize for the reader, not the author.

**The Boy Scout Rule:** Leave the code cleaner than you found it. Small, continuous improvements
prevent decay. Don't require permission to rename a variable or extract a function.

**Clean code does one thing well.** Functions, classes, modules — each should have a single,
clearly-named responsibility. If you need "and" to describe what it does, split it.

**Meaningful names > comments.** A good name makes a comment unnecessary. If you need a comment to
explain what a variable or function is, rename it.

**Violating the letter of the rules is violating the spirit.** "My function is 30 lines but it's
still readable" is a rationalization. Extract.

## Quick Reference — Principles by Category

### Naming (Ch. 2)

- Use **intention-revealing** names: `elapsedTimeInDays`, not `d`.
- Avoid **disinformation**: don't call it `accountList` if it isn't a List.
- Make **meaningful distinctions**: `ProductData` vs `ProductInfo` is noise.
- **Pronounceable** and **searchable** names. Single letters only for tiny scopes.
- **Class names** are nouns (`Customer`, `Account`). **Method names** are verbs (`save`, `deletePage`).
- **One word per concept**: don't mix `fetch`, `retrieve`, `get` for the same operation.
- Don't **encode types** in names (`strName`, `m_user`) — modern IDEs show types.

See `references/naming.md`.

### Functions (Ch. 3)

- **Small. Then smaller.** Aim for <20 lines; most should be <10.
- **Do one thing.** A function does one thing when you can't extract another meaningful function from it.
- **One level of abstraction per function.** Don't mix high-level policy with low-level details.
- **Few arguments.** 0 > 1 > 2 > 3. Three or more is a smell — introduce an object.
- **No flag arguments.** `render(true)` means the function does two things. Split it.
- **No side effects.** `checkPassword()` that also initializes a session is a lie.
- **Command-Query Separation.** Functions either *do* something or *answer* something, never both.
- **Prefer exceptions to error codes.** Error codes pollute callers with nested `if`s.
- **DRY.** Duplication is the root of most evil in software.

See `references/functions.md`.

### Comments (Ch. 4)

- **Comments are a failure.** Every comment is an admission that you couldn't make the code speak.
- **Don't comment bad code — rewrite it.**
- **Good comments:** legal headers, explanation of intent, warnings of consequences, TODOs, public API docs.
- **Bad comments:** redundant (`// increment i`), misleading, mandated by policy, journal entries,
  commented-out code (delete it — version control remembers).

See `references/comments.md`.

### Formatting (Ch. 5)

- **Vertical openness separates concepts.** Blank line between functions, between groups of related lines.
- **Vertical density implies association.** Related lines stay together.
- **Declare variables close to their use.**
- **Dependent functions should be near.** Caller above callee when possible.
- **Team style trumps personal preference.** Pick one, enforce with formatter.

See `references/formatting.md`.

### Objects and Data Structures (Ch. 6)

- **Objects hide data, expose behavior.** Data structures expose data, have no behavior.
- **Don't mix.** Hybrids (half-object, half-struct) get the worst of both.
- **Law of Demeter:** a method should only call methods of its class, its parameters, objects it
  creates, or its direct fields — not navigate through chains (`a.getB().getC().doStuff()`).

See `references/objects-and-data.md`.

### Error Handling (Ch. 7)

- **Use exceptions, not return codes.** Error codes mix happy path with error path.
- **Write the `try` block first** — it defines the scope of what can go wrong.
- **Provide context with exceptions.** The message should let the caller diagnose.
- **Don't return null. Don't pass null.** Null checks metastasize. Use Optional, empty collections,
  or the Null Object pattern.
- **Wrap third-party APIs** to define your own exception hierarchy.

See `references/error-handling.md`.

### Boundaries (Ch. 8)

- **Isolate third-party code.** Write an adapter/wrapper so your code depends on your interface, not theirs.
- **Learning tests:** write tests against the third-party API to pin down its behavior and catch changes on upgrade.
- **Depend on code you control** at internal boundaries.

See `references/boundaries.md`.

### Unit Tests (Ch. 9)

- **Three Laws of TDD:** (1) don't write production code until a failing test requires it,
  (2) write only enough test to fail, (3) write only enough production code to pass.
- **F.I.R.S.T.** Tests must be Fast, Independent, Repeatable, Self-validating, Timely.
- **One assert per test** — or at least one concept per test.
- **Test code is first-class.** Apply the same quality bar as production code.
- **Domain-specific testing language.** Build helpers so tests read like specifications.

See `references/unit-tests.md`.

### Classes (Ch. 10)

- **Small.** Measured by responsibilities, not lines. Name should describe the responsibility — if it
  takes "and" or weasel words (`Processor`, `Manager`), split.
- **Single Responsibility Principle:** one reason to change.
- **High cohesion.** Most methods use most fields. Low cohesion → extract classes.
- **Open-Closed Principle:** open to extension (subclass, compose) but closed to modification.
- **Dependency Inversion:** depend on abstractions, not concretions.

See `references/classes.md`.

### Systems (Ch. 11)

- **Separate construction from use.** Main() and factories build the graph; business code just uses it.
- **Dependency Injection** — don't let objects create their collaborators.
- **Cross-cutting concerns** (logging, transactions, security) via aspects or decorators, not scattered code.
- **Grow systems incrementally.** Start with the simplest architecture; refactor as needs emerge.

See `references/systems.md`.

### Concurrency (Ch. 13)

- **Concurrency is a decoupling strategy** — separates *what* from *when*, but adds complexity.
- **Keep concurrency code separate from other code.**
- **Limit shared data.** Copy data when possible; use thread-local.
- **Understand your library** — `ExecutorService`, `ConcurrentHashMap`, channels. Don't reinvent.
- **Know the models:** Producer-Consumer, Readers-Writers, Dining Philosophers.

See `references/concurrency.md`.

### Code Smells and Heuristics (Ch. 17)

A checklist of specific anti-patterns to scan for during review: comments smells, environment smells,
function smells (too many arguments, dead parameters, flag args), general smells (duplication, magic
numbers, inconsistency, artificial coupling), and test smells (insufficient, disabled, redundant).

See `references/smells-and-heuristics.md`.

## Review Checklist

When reviewing code, walk down this list in order:

1. **Names** — Do they reveal intent? Any disinformation, noise words, or encodings?
2. **Functions** — Any >20 lines? Flag args? >3 params? Doing more than one thing?
3. **Duplication** — Copy-pasted logic? Parallel `switch`es on the same type?
4. **Comments** — Any that could be replaced by a rename or extraction?
5. **Error handling** — Null returned/passed? Error codes? try-blocks without clear scope?
6. **Classes** — Does each have a single responsibility? Any `Manager`/`Processor`/`Util`?
7. **Tests** — One concept per test? Fast and independent? Cover edge cases?
8. **Boundaries** — Third-party types leaking across the codebase?

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
- A function is longer than a screen.
- A class has methods that use disjoint sets of fields.
- A test name starts with `test1`, `test2`, …
- A parameter is a `boolean` flag.
- You're returning `null` or checking `== null`.

## When NOT to Use This Skill

- **Language-specific syntax or idioms** — use the language's style skill (e.g. `devpilot-google-go-style`).
- **Project-specific conventions** — those belong in `CLAUDE.md`.
- **Mechanical formatting** — run the formatter; don't reason about spaces.
- **When Clean Code conflicts with language idioms** — idioms win (e.g. Go prefers error returns over exceptions;
  Go constructor convention is `New`, not banned by this skill even though Clean Code dislikes prefixes).

## Real-World Impact

- Reduced defect density in refactored modules (Martin cites multiple case studies).
- Faster onboarding: new engineers productive in days, not weeks, when naming and structure are consistent.
- Lower cost of change: SRP + DI mean features land in one place, not scattered across the codebase.
