# Code Smells and Heuristics (Clean Code, Ch. 17)

A condensed checklist drawn from the book's final chapter. Use it as a review aid, not a rulebook.
Each entry names the smell; apply judgment for the fix.

## Comments

- **C1 Inappropriate information** — changelogs, authors, metadata that belong in VCS.
- **C2 Obsolete comment** — drifted from the code it describes. Delete or update.
- **C3 Redundant comment** — restates what the code obviously says.
- **C4 Poorly written comment** — worth writing, worth writing well.
- **C5 Commented-out code** — delete. VCS remembers.

## Environment

- **E1 Build requires more than one step** — should be one command.
- **E2 Tests require more than one step** — should be one command.

## Functions

- **F1 Too many arguments** — aim for 0, 1, 2. 3+ is suspect.
- **F2 Output arguments** — modifying an argument is counterintuitive. Return a value.
- **F3 Flag arguments** — boolean params mean the function does two things.
- **F4 Dead function** — never called. Delete.

## General

- **G1 Multiple languages in one source file** — HTML + JS + CSS + SQL in one file is a modular failure.
- **G2 Obvious behavior is unimplemented** — "Principle of Least Surprise": if `Day.fromString("Monday")` doesn't work, users lose faith.
- **G3 Incorrect behavior at the boundaries** — special cases (empty, null, max, min) are where bugs hide. Write tests for every edge case you can imagine.
- **G4 Overridden safeties** — commented-out tests, `@Ignore`, disabled warnings. Don't disable safety checks; fix the underlying issue.
- **G5 Duplication** — the most important smell. Every duplication is a missed abstraction.
- **G6 Code at wrong level of abstraction** — high-level policy mixed with low-level detail in the same function or class.
- **G7 Base classes depending on their derivatives** — base should know nothing of derivatives.
- **G8 Too much information** — a class or module with a bloated public API. Hide more.
- **G9 Dead code** — unreachable conditions, unused methods. Delete.
- **G10 Vertical separation** — variables and functions should be close to where they're used.
- **G11 Inconsistency** — if you name something a certain way, name similar things the same way. Pick a convention and stick with it.
- **G12 Clutter** — empty constructors with no purpose, unused variables, pointless comments. Delete.
- **G13 Artificial coupling** — things that don't depend on each other shouldn't be coupled. Common constants, utilities, functions put in inconvenient places "for now" tend to calcify.
- **G14 Feature envy** — a method that uses another class's accessors more than its own fields belongs on that other class.
- **G15 Selector arguments** — see F3. Flags, enums, or strings that select behavior mean the function does multiple things.
- **G16 Obscured intent** — one-letter names, math-as-code, dense expressions. Rename; extract.
- **G17 Misplaced responsibility** — code belongs where the reader would naturally look for it, not where it was convenient to write.
- **G18 Inappropriate static** — statics that could be polymorphic hurt testability. Use instance methods unless the function truly has no instance state.
- **G19 Use explanatory variables** — break up complex expressions into named intermediates.
- **G20 Function names should say what they do** — `date.add(5)` — five what? Rename `addDaysTo`, `increaseByDays`.
- **G21 Understand the algorithm** — many functions "appear" to work by coincidence. Work the algorithm out until it's obviously correct.
- **G22 Make logical dependencies physical** — if module A depends on module B knowing a fact (like "there are 100 slots"), B should expose that fact; A shouldn't hard-code it.
- **G23 Prefer polymorphism to if/else or switch/case** — switches over type are often missing polymorphism. Exception: one switch at the factory, concrete types below.
- **G24 Follow standard conventions** — team style, language idioms. Rebels cost the team.
- **G25 Replace magic numbers with named constants** — including magic strings. `3.14` is okay; `86400` is not.
- **G26 Be precise** — off-by-one, rounding, thread scheduling, date/timezone, currency — pay attention where precision matters.
- **G27 Structure over convention** — enforce design decisions with structure (types, method signatures) over convention (naming, comments).
- **G28 Encapsulate conditionals** — extract boolean expressions into well-named predicates.
- **G29 Avoid negative conditionals** — `if (!buffer.shouldNotCompact())` — replace with the positive form.
- **G30 Functions should do one thing** — see F1-F4 and Functions chapter.
- **G31 Hidden temporal couplings** — when the order of calls matters, the API should enforce it (each method returns what the next needs).
- **G32 Don't be arbitrary** — structure should reflect reason; capricious choices confuse maintainers.
- **G33 Encapsulate boundary conditions** — `level + 1` repeated everywhere → extract `nextLevel()`.
- **G34 Functions should descend only one level of abstraction** — see Functions chapter.
- **G35 Keep configurable data at high levels** — don't bury defaults deep in the call stack.
- **G36 Avoid transitive navigation** — Law of Demeter. See objects-and-data reference.

## Java-Specific (Chapter 15 — apply as relevant)

- **J1 Avoid long import lists by using wildcards** — taste call; modern IDEs help.
- **J2 Don't inherit constants** — put them in an enum or static import.
- **J3 Constants vs. Enums** — prefer enums.

## Names

- **N1 Choose descriptive names** — see naming reference.
- **N2 Choose names at the appropriate level of abstraction** — don't name for implementation.
- **N3 Use standard nomenclature where possible** — patterns names (Factory, Decorator, Visitor) carry meaning.
- **N4 Unambiguous names** — `doRename` better than `rename` if there are five different renames.
- **N5 Use long names for long scopes** — proportional to how far the name must travel.
- **N6 Avoid encodings** — no Hungarian, no `m_`, no `I`-prefix on interfaces.
- **N7 Names should describe side effects** — `createOrReturnOos` not `getOos`.

## Tests

- **T1 Insufficient tests** — cover every case that can break.
- **T2 Use a coverage tool** — find the gaps.
- **T3 Don't skip trivial tests** — they document behavior and expectations.
- **T4 An ignored test is a question about an ambiguity** — resolve the ambiguity.
- **T5 Test boundary conditions** — empty, null, max, min, off-by-one.
- **T6 Exhaustively test near bugs** — bugs cluster. Where one hides, more do.
- **T7 Patterns of failure are revealing** — clusters of failing tests point to shared root cause.
- **T8 Test coverage patterns can be revealing** — consistent gaps point to untested paths.
- **T9 Tests should be fast** — slow tests don't get run.

## How to Use This Checklist

During a self-review or PR review, scan these smells against the diff. You don't need to invoke the
full list every time — with practice, you'll spot the common ones (G5 duplication, F1 too many args,
G6 wrong abstraction level, C5 commented-out code) at a glance.

For a new contributor, reading the chapter references (plus Martin Fowler's *Refactoring*) is the
fastest way to internalize the vocabulary.
