# Classes (Clean Code, Ch. 10)

> **Language override — Go:** Go has no classes. Apply the principles in this chapter at the
> **package** and **struct** level:
> - *Small, single responsibility* → one package per concept; one struct per concept. `Manager`,
>   `Util`, `Helper` packages and types are banned.
> - *High cohesion* → struct methods should use most of the struct's fields; otherwise split the
>   struct or move the method.
> - *Open-Closed* → accept interfaces defined in the **consumer** package; let new implementations
>   plug in without modifying existing code. (Per `devpilot-google-go-style`, **do not** define
>   interfaces alongside their implementations.)
> - *Dependency Inversion* → constructors return concrete types; consumers define the minimal
>   interfaces they need.

## Class Organization

Standard ordering (Java convention):
1. Public static constants
2. Private static variables
3. Private instance variables
4. Public functions
5. Private utilities called by the public function, placed right after the caller (stepdown rule)

Rarely any public variables.

## Encapsulation

Keep variables and utility methods private. Relax only for testing — and prefer package-private /
protected over public, and extracting a helper class over loosening visibility.

## Classes Should Be Small

First rule: small. Second rule: **smaller than that**.

Measured not in lines but in **responsibilities**.

```java
// Too many responsibilities
public class SuperDashboard extends JFrame implements MetaDataUser {
    // 70 public methods, does everything
}
```

**The name is your first clue.** If you can't name a class in 25 words without "if", "and", "or",
"but" — it has too many responsibilities. Classes named `Manager`, `Processor`, `Super`, `Data`,
`Info` almost always hide too much.

A complete, terse description should fit in ~25 words.

## Single Responsibility Principle (SRP)

A class should have **one reason to change**.

SRP is among the most important OO principles, and among the most violated. We get code working,
then move on — rarely returning to split overloaded classes. Yet a codebase of small, focused
classes is both easier to understand and easier to change.

**System-level argument:** many small classes, each with a single responsibility, collaborate to
achieve complex behavior. Compare to a few monolithic classes that try to do everything. The
monolithic design feels like "less code" but is vastly harder to reason about piece-by-piece.

Worried about too many little classes? Don't be. Organization via names, packages, and logical
grouping makes them discoverable. Chaos comes from *big* classes with lots of responsibilities, not
from many small ones.

## Cohesion

A class is **cohesive** when its methods and fields are highly interdependent — each method uses
most of the fields. As cohesion decreases, the class is pulling apart into smaller classes that
want to escape.

When you see a class where some methods use only some fields, consider extracting a new class from
those fields and their methods.

### Maintaining Cohesion → Many Small Classes

Breaking a long function into smaller pieces often produces many small pieces that can be extracted
into their own class, clarifying the structure. This is normal, desirable refactoring.

## Organizing for Change

In most systems, change is continual. Clean systems organize classes so that **the risk of change is
minimized**.

### Open-Closed Principle (OCP)

Classes should be **open to extension but closed to modification**. Achieve this by extending
(subclass, compose, plug in) rather than modifying existing code.

```java
// Bad — adding a new report type means editing this class
public class Reporter {
    public String report(ReportType type) {
        switch (type) {
            case PDF: return renderPdf();
            case HTML: return renderHtml();
            // add another case every time
        }
    }
}

// Good — new report type = new class, no changes to existing code
public interface Report { String render(); }
public class PdfReport implements Report { ... }
public class HtmlReport implements Report { ... }
```

### Isolating from Change

Depend on **abstractions**, not concrete classes. Tests illustrate the point: if your class pulls
from an external API, wrap the API behind an interface and test against a stub. The class now
doesn't care about the concrete API — and neither does any future replacement.

## Dependency Inversion Principle (DIP)

- High-level modules should not depend on low-level modules. Both should depend on abstractions.
- Abstractions should not depend on details. Details should depend on abstractions.

In practice: programmatic interfaces, dependency injection, plugin architecture. Your business
rules don't care whether persistence is Postgres or SQLite or memory.

## Summary

Small classes with single responsibilities, organized to minimize the blast radius of change, are
the foundation of a maintainable system. Any class you can describe in 25 words with no weasel
words is on the right track.
