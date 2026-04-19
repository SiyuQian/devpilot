# Systems (Clean Code, Ch. 11)

At the system level, cleanliness comes from **separation of concerns** and **the ability to evolve
the architecture** as the business does.

## Separate Constructing a System from Using It

Construction is a very different process from use. **Main()** (or a factory / module-wiring layer)
should be the only code that knows how objects are constructed; the rest of the system should assume
objects are already connected and ready.

```java
// Bad — lazy init pollutes business logic
public Service getService() {
    if (service == null) service = new MyServiceImpl(new DbConnection(url));
    return service;
}

// Good — construction happens once, at startup, in Main/factory
public Service getService() { return service; }
```

### Lazy Initialization Is a Smell at the Business Layer

- Hard-codes a dependency (`MyServiceImpl`, `DbConnection`).
- Couples runtime behavior to construction logic.
- Complicates testing (you have to intercept or nullify the lazy init).
- Violates SRP — the class now does construction AND business work.

**Solution:** Factories, builders, and Dependency Injection.

## Dependency Injection

Inversion of Control applied to dependencies. Objects don't create their collaborators — they
receive them. A DI container (Spring, Guice, or just hand-wired main()) assembles the graph.

```java
// Concrete dependency - hard to test, hard to change
public class EmailSender {
    private SmtpClient client = new SmtpClient("localhost", 25);
}

// Injected - test with stub, swap in main()
public class EmailSender {
    private final MailClient client;
    public EmailSender(MailClient client) { this.client = client; }
}
```

## Scaling Up

Software systems are unique in that their architectures can grow **incrementally**, IF we maintain
proper separation of concerns.

You don't have to get it all right up front. You *do* have to keep concerns separate so that you can
replace/rearrange as new constraints emerge.

**Don't big-bang redesign.** Refactor toward the new architecture while shipping.

## Aspects and Cross-Cutting Concerns

Some concerns (transactions, logging, security, caching) span many classes. Scattering the code for
these concerns across business logic is brittle and obscures intent.

**AOP**, **Proxies**, and **Decorators** let you apply these concerns uniformly without modifying
each business class:

```java
// Without aspects - every service method begins and ends with tx boilerplate
public void transfer() {
    tx.begin();
    try { ... tx.commit(); }
    catch (Exception e) { tx.rollback(); throw e; }
}

// With aspects / decorators - the service method is just business
@Transactional
public void transfer() { ... }
```

Common mechanisms:
- Java Proxies / Dynamic Proxies.
- AOP frameworks (Spring AOP, AspectJ).
- Functional composition (middleware chains).

## Test Drive the System Architecture

You can test-drive architecture the same way you test-drive code. Start with a simple "naive"
architecture and evolve it — new capabilities added as the business demands, not speculated into
existence. The only way to know an architecture decision was correct is to see it under the pressure
of real requirements.

## Optimize Decision Making

- Defer decisions until you have to make them (concrete requirements > speculation).
- Decentralize decisions — let the team closest to the problem decide.
- Postpone rather than commit prematurely.

## Use Standards Wisely

Frameworks, libraries, and standards have massive benefits — but also lock-in. Evaluate honestly:
does this standard solve our problem, or is it the solution in search of a problem? Many teams adopt
heavyweight frameworks for trivial needs and pay ongoing costs.

## Systems Need Domain-Specific Languages

A good DSL (internal or external) expresses the domain's concepts directly. Business rules written
in the domain's vocabulary are verifiable by domain experts and easier for new engineers to grasp.

## Summary

At every level — functions, classes, systems — **keep concerns separate**. Decouple construction
from use, business from cross-cutting. Depend on abstractions. Evolve architecture incrementally.
Never let architecture get in the way of shipping business value, but never let short-term shipping
rot the ability to evolve.
