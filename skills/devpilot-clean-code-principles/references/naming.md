# Meaningful Names (Clean Code, Ch. 2)

> **Language overrides:**
> - **Go drops the `Get` prefix** on accessors: `User.Name()` not `User.GetName()`. Follow
>   `devpilot-google-go-style` when writing Go.
> - **Go uses MixedCaps for exported, mixedCaps for unexported** — no underscores in identifiers.
>   Filenames and flag names are the only snake_case exceptions.
> - **Go package names** should be short, lowercase, no underscores, and single-word
>   (`bytes`, `http`). `Manager`/`Util`-style names are banned in both guides.
> - **Initialisms keep case consistent in Go**: `URL`, `ID`, `HTTP` — never `Url`, `Id`, `Http`.

Names are everywhere — variables, functions, classes, packages, files. Getting them right is the
highest-leverage thing you can do for readability.

## Use Intention-Revealing Names

The name should answer: *why* does it exist, *what* does it do, *how* is it used?

```java
// Bad
int d; // elapsed time in days

// Good
int elapsedTimeInDays;
int daysSinceCreation;
int fileAgeInDays;
```

If you need a comment to explain the name, the name is wrong.

## Avoid Disinformation

- Don't use names that mean something else in the domain (`hp` for "hypotenuse" collides with Hewlett-Packard).
- Don't use `accountList` unless it's actually a List. Prefer `accounts`.
- Beware of names that differ by only a letter or two: `XYZControllerForEfficientHandlingOfStrings`
  vs `XYZControllerForEfficientStorageOfStrings`.
- Lowercase `l` and uppercase `O` look like `1` and `0`. Avoid.

## Make Meaningful Distinctions

- Don't add noise: `ProductData` vs `ProductInfo`, `NameString` vs `Name`, `theMessage` vs `message`.
- Don't number-series names: `a1`, `a2`, `a3`.
- If two things are genuinely different, the names should reflect *how* they differ.

```java
// Noise — what's the difference?
getActiveAccount();
getActiveAccounts();
getActiveAccountInfo();
```

## Use Pronounceable and Searchable Names

- `genymdhms` → `generationTimestamp`. You'll talk about code out loud.
- Single-letter names only for tiny scopes (loop index in a 5-line loop). `e` in a 500-line method
  cannot be grep'd.
- Constants like `7` or `86400` should be `WORK_DAYS_PER_WEEK`, `SECONDS_PER_DAY` — searchable.

## Avoid Encodings

- No Hungarian notation: `strName`, `iCount`. The type system knows.
- No `m_` member prefixes.
- No `I` prefix on interfaces (`IShapeFactory` → `ShapeFactory`; the implementation can be `ShapeFactoryImpl` or better yet, named after what it is).

## Class Names

Nouns or noun phrases: `Customer`, `WikiPage`, `Account`, `AddressParser`. Avoid verbs.

Avoid weasel words: `Manager`, `Processor`, `Data`, `Info`. They signal unclear responsibility.

## Method Names

Verbs or verb phrases: `postPayment`, `deletePage`, `save`.

Accessors, mutators, predicates: `getName`, `setName`, `isPosted` (Java/TypeScript).
**In Go:** drop the `Get` prefix — `Name()`, `SetName(...)`, `Posted()` / `IsPosted()`.

Overloaded constructors → use static factory methods with descriptive names:
```java
Complex fulcrum = Complex.FromRealNumber(23.0);
// beats
Complex fulcrum = new Complex(23.0);
```

## Pick One Word per Concept

- Don't have `fetch`, `retrieve`, and `get` all mean the same thing across your codebase.
- Don't have `controller`, `manager`, and `driver` used interchangeably.
- Consistent vocabulary is a form of documentation.

## Don't Pun

Opposite: using the same word for two purposes. If `add` means "concatenate" in one class and "insert
into a collection" in another, pick a different word (`insert`, `append`).

## Use Solution Domain Names

Programmers reading your code are programmers. `AccountVisitor` (GoF Visitor pattern), `JobQueue`,
`PriorityQueue` — these communicate precisely.

## Use Problem Domain Names When No Tech Term Applies

When the concept belongs to the business, use the business's word. Ask the domain expert.

## Add Meaningful Context

A variable named `state` is ambiguous. In a method named `addrState` or inside a class `Address`,
it's clear. Prefer the class over the prefix when possible.

## Don't Add Gratuitous Context

Inside `GasStationDeluxe`, don't prefix every class with `GSD`. The package already provides context.
