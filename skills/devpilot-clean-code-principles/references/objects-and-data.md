# Objects and Data Structures (Clean Code, Ch. 6)

> **Language override — Go:** Go structs are closer to *data structures* than *objects* by design.
> Accessing public fields directly is idiomatic; not every field needs a getter/setter. The
> **data vs. object** distinction still matters — decide per type which role it plays and don't
> mix. Law of Demeter still applies to types that own behavior (methods), not to plain structs
> used as data carriers.

## Data Abstraction

Hiding implementation is about abstraction, not just adding getters/setters. A class should expose
abstract interfaces that let users manipulate the essence of the data without knowing its
representation.

```ts
// ❌ Concrete — exposes Cartesian implementation
class Point {
  constructor(public x: number, public y: number) {}
}

// ✅ Abstract — could be Cartesian or polar underneath
interface Point {
  x(): number;
  y(): number;
  setCartesian(x: number, y: number): void;
  r(): number;
  theta(): number;
  setPolar(r: number, theta: number): void;
}
```

```go
// In Go you'd typically choose one shape and commit.
// Data-structure Point — exposed fields, no methods:
type Point struct{ X, Y float64 }

// Object Point — behavior-owning, no exposed coordinates:
type Point interface {
    X() float64
    Y() float64
    SetCartesian(x, y float64)
    R() float64
    Theta() float64
    SetPolar(r, theta float64)
}
```

Don't expose the data *and* expose methods that compute on it (hybrid). Pick one role.
Mindless getters/setters over public fields is the worst of both worlds.

## Data/Object Anti-Symmetry

**Objects** hide their data behind abstractions and expose functions that operate on that data.
**Data structures** expose their data and have no meaningful functions.

They are virtual opposites:

| Procedural (data structures + functions) | OO (objects) |
|------------------------------------------|--------------|
| Easy to add new functions without changing existing structures | Easy to add new classes without changing existing functions |
| Hard to add new data structures (must change all functions) | Hard to add new functions (must change all classes) |

Neither is universally better. Choose based on what will vary.

**Hybrids** — classes with half-exposed data and half-real-methods — get the worst of both: hard to
add new data structures AND hard to add new functions. Avoid.

## The Law of Demeter

A method `f` of a class `C` should only call methods of:
- `C` itself,
- an object created by `f`,
- an object passed as an argument to `f`,
- an object held in an instance variable of `C`.

**Not** methods of objects returned by any of the above. "Talk to friends, not to strangers."

### Train Wrecks

```ts
// ❌ Chain reaches through three objects' internals
const outputDir = ctx.getOptions().getScratchDir().getAbsolutePath();

// Splitting it doesn't fix the violation, only the formatting
const opts = ctx.getOptions();
const scratchDir = opts.getScratchDir();
const outputDir = scratchDir.getAbsolutePath();
```

```go
// Same pattern in Go with methods:
outputDir := ctx.Options().ScratchDir().AbsolutePath()
```

If `ctx`, `Options`, and `ScratchDir` are **objects** (they own behavior), this violates Demeter.
If they're **data structures** (plain fields), Demeter doesn't apply — data structures exist to be
navigated. The language override at the top of this file matters: Go's struct field access like
`ctx.options.scratchDir.absolutePath` is fine for pure data carriers.

### Hiding Structure

Don't navigate the structure. Ask the object to **do** something for you.

```ts
// ❌ Caller knows how scratch files are built
const outFile = `${outputDir}/${className.replace(/\./g, "/")}.class`;
const fout = fs.createWriteStream(outFile);

// ✅ Tell, don't ask
const out = ctx.createScratchFileStream(classFileName);
```

```go
// Same in Go — move the "how" onto the owner
out, err := ctx.CreateScratchFileStream(className)
```

The caller doesn't need to know *how* the scratch file is created.

## Data Transfer Objects (DTOs)

The quintessential form of a data structure: a class with public variables and no functions. Useful
at system boundaries (JSON parsing, DB rows, RPC payloads). Don't treat them as objects.

## Active Record

A DTO with navigational methods (`save`, `find`). Tempting to add business rules to them — resist.
An Active Record is still a data structure. Put business rules in separate objects that *use* the
Active Record.

## Summary

- Objects expose behavior and hide data. Extend with new object types without changing behaviors.
- Data structures expose data and have no behavior. Extend with new behaviors without changing data.
- Pick one per class. Hybrids are worst-of-both.
- Law of Demeter applies to objects. It doesn't apply to pure data structures.
