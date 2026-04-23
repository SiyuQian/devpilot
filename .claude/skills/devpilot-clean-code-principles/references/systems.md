# Systems (Clean Code, Ch. 11)

At the system level, cleanliness comes from **separation of concerns** and **the ability to evolve
the architecture** as the business does.

## Separate Constructing a System from Using It

Construction is a very different process from use. **Main()** (or a factory / module-wiring layer)
should be the only code that knows how objects are constructed; the rest of the system should assume
objects are already connected and ready.

```ts
// ❌ Lazy init pollutes business logic with construction decisions
class OrderService {
  private service?: Service;
  getService(): Service {
    if (!this.service) this.service = new MyServiceImpl(new DbConnection(url));
    return this.service;
  }
}

// ✅ Construction happens once at startup; runtime code just uses it
class OrderService {
  constructor(private readonly service: Service) {}
}
```

```go
// ❌ Go equivalent of the smell — package-level lazy singleton with hard deps
var svc *MyServiceImpl
func GetService() *MyServiceImpl {
    if svc == nil { svc = NewMyServiceImpl(NewDB(url)) }
    return svc
}

// ✅ Assemble in main; consumers take the dependency they need
type OrderService struct{ svc Service }
func NewOrderService(s Service) *OrderService { return &OrderService{svc: s} }
```

### Lazy Initialization Is a Smell at the Business Layer

- Hard-codes a dependency (`MyServiceImpl`, `DbConnection`).
- Couples runtime behavior to construction logic.
- Complicates testing (you have to intercept or nullify the lazy init).
- Violates SRP — the class now does construction AND business work.

**Solution:** Factories, builders, and Dependency Injection.

## Dependency Injection

Inversion of Control applied to dependencies. Objects don't create their collaborators — they
receive them. A DI container (Inversify, NestJS, or just hand-wired `main()`/root wiring) assembles
the graph.

```ts
// ❌ Concrete dependency baked in — hard to test, hard to swap
class EmailSender {
  private client = new SmtpClient("localhost", 25);
}

// ✅ Inject — stub in tests, configure in main
class EmailSender {
  constructor(private readonly client: MailClient) {}
}
```

```go
// ❌ Same smell in Go
type EmailSender struct { client *smtp.Client }
func NewEmailSender() *EmailSender {
    return &EmailSender{client: smtp.Dial("localhost:25")}
}

// ✅ Accept the collaborator; wire it in main
type MailClient interface {
    Send(ctx context.Context, msg Mail) error
}
type EmailSender struct { client MailClient }
func NewEmailSender(c MailClient) *EmailSender { return &EmailSender{client: c} }
// Note: per devpilot-google-go-style, MailClient is defined in the CONSUMER package (where
// EmailSender lives), not next to the concrete smtp.Client implementation.
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

```ts
// ❌ Every service method repeats transaction boilerplate
async transfer() {
  await tx.begin();
  try { /* ... */ await tx.commit(); }
  catch (err) { await tx.rollback(); throw err; }
}

// ✅ TypeScript decorator (or NestJS interceptor) keeps the method pure business
@Transactional()
async transfer() { /* ... */ }
```

```go
// Go has no decorators. Use higher-order functions / middleware instead.
func Transactional(db *sql.DB, fn func(ctx context.Context, tx *sql.Tx) error) func(context.Context) error {
    return func(ctx context.Context) error {
        tx, err := db.BeginTx(ctx, nil)
        if err != nil { return err }
        if err := fn(ctx, tx); err != nil {
            _ = tx.Rollback()
            return err
        }
        return tx.Commit()
    }
}

// Business method stays focused
transfer := Transactional(db, func(ctx context.Context, tx *sql.Tx) error {
    // just business
    return nil
})
```

Common mechanisms:
- TypeScript decorators + reflection (NestJS, TypeORM).
- Functional composition / middleware chains (Express, Fastify, Go's `http.Handler` middleware).
- Proxies / wrappers around the concrete implementation.

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
