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

```java
// Bad — concrete, exposes Cartesian implementation
public class Point {
    public double x;
    public double y;
}

// Good — abstract, could be Cartesian or polar underneath
public interface Point {
    double getX();
    double getY();
    void setCartesian(double x, double y);
    double getR();
    double getTheta();
    void setPolar(double r, double theta);
}
```

Don't expose the data. Express it in abstract terms. Serious thought is required for the best way to
represent the data an object contains. Mindless getters/setters are the worst option.

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

```java
final String outputDir = ctxt.getOptions().getScratchDir().getAbsolutePath();
```
This is a train wreck because it looks like a sequence of coupled train cars. Split it:
```java
Options opts = ctxt.getOptions();
File scratchDir = opts.getScratchDir();
final String outputDir = scratchDir.getAbsolutePath();
```

Both forms violate Demeter if `ctxt`, `Options`, and `ScratchDir` are **objects**. If they're **data
structures** (simple public fields), Demeter doesn't apply — data structures expose their data by
design.

### Hybrids Violate Demeter

```java
final String outputDir = ctxt.options.scratchDir.absolutePath;
```
If these are fields (data structures), fine. If they're objects with methods hidden behind getters,
it's a wreck.

### Hiding Structure

Don't navigate the structure. Ask the object to **do** something for you.

```java
// Navigating structure (violation)
String outFile = outputDir + "/" + className.replace('.', '/') + ".class";
FileOutputStream fout = new FileOutputStream(outFile);

// Telling the object what to do
BufferedOutputStream bos = ctxt.createScratchFileStream(classFileName);
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
