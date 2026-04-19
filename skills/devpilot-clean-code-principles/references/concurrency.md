# Concurrency (Clean Code, Ch. 13)

> **Language override — Go:** Prefer **synchronous** functions; let callers add concurrency
> (goroutines) if needed. `context.Context` is always the **first parameter**, never stored in a
> struct. Use channels, `sync.WaitGroup`, and `sync.Mutex` from the standard library — don't
> reinvent them. Make goroutine lifetimes obvious; never start one without a clear exit path.
> See `devpilot-google-go-style` for Go-specific patterns.


> Objects are abstractions of processing. Threads are abstractions of schedule.

Concurrency is a **decoupling strategy** — it separates *what* gets done from *when* it gets done.
Done well, it improves throughput and responsiveness. Done poorly, it introduces bugs that are
non-deterministic, hard to reproduce, and easy to dismiss as "flukes".

## Myths and Misconceptions

- **"Concurrency always improves performance."** Not always. Only when there's significant wait
  time that could be shared, or independent work to parallelize.
- **"Design doesn't change with concurrency."** Often it changes dramatically — decoupling what
  from when restructures algorithms and ownership.
- **"You don't need to think about concurrency when using a container (EJB, Servlet)."** You do —
  you need to know what the container guarantees and doesn't.

## Challenges

Concurrency is hard because small sections of code can interleave in surprising ways. Most of the
time code "works"; the rare interleaving that breaks it is hard to find. Unit tests that pass a
thousand times can fail once — and one failure means a bug.

## Concurrency Defense Principles

### Single Responsibility Principle

Concurrency is complex enough to be a reason to change by itself. Keep concurrency code **separate**
from other code — it's its own layer.

### Limit the Scope of Data

Concurrency bugs come from shared mutable state. Minimize sharing:
- Encapsulate critical sections behind synchronized methods.
- Each piece of shared data should have exactly one place (ideally one method) that guards it.

### Use Copies of Data

When possible, pass copies — read, modify locally, merge results afterward. Cost of copies is
often less than the cost of synchronization bugs.

### Threads Should Be As Independent As Possible

Minimize communication between threads. A thread that owns all its data (thread-local, or a
consumer draining its own queue) is far safer than threads sharing state.

## Know Your Library

Use proven primitives. Don't invent your own:
- Thread-safe collections (`ConcurrentHashMap`, etc.)
- `ExecutorService`, `ForkJoinPool`
- Concurrent queues, semaphores, latches
- Reactive streams, channels, actors

Understand their guarantees. A lot of "weird" concurrency bugs come from misusing these (e.g., using
a thread-safe Map inside a non-atomic check-then-act).

## Know Your Execution Models

Classic patterns to recognize:
- **Producer-Consumer** — producers put items on a bounded queue, consumers take them.
- **Readers-Writers** — many readers share access; writers need exclusivity; starvation risks both ways.
- **Dining Philosophers** — resource contention leading to deadlock/livelock.

Most real-world concurrency problems are variations of these. Name the pattern, use the known
solution.

## Beware Dependencies Between Synchronized Methods

Multiple synchronized methods on one shared object can compose into race conditions even though each
method is individually thread-safe (check-then-act over two methods).

**Avoid** multiple synchronized methods that together constitute a protocol. If you must have them:
- Client-based locking — clients lock the object around the call sequence.
- Server-based locking — move the protocol into one synchronized method on the server.
- Adapter that wraps the protocol.

## Keep Synchronized Sections Small

Synchronization is expensive and error-prone. Lock only what must be locked; release as fast as
possible.

```java
// Bad — the entire method is synchronized, blocking way more than needed
public synchronized void process(Data data) {
    expensiveLocalComputation(data);
    appendToSharedList(data);
}

// Better
public void process(Data data) {
    expensiveLocalComputation(data);
    synchronized (this) { appendToSharedList(data); }
}
```

## Writing Correct Shutdown Code Is Hard

Graceful shutdown — letting threads finish work, drain queues, release resources — is notoriously
difficult. Deadlocks often happen at shutdown. Design for it from the start; don't tack it on.

## Testing Threaded Code

- Write tests that can expose concurrency flaws, and run them frequently.
- Treat spurious failures as real bugs, not "flakes".
- Run on different platforms, different loads.
- Instrument the code to increase the probability of failure (sleep injections, Thread.yield).
- Use tools: race detectors, thread sanitizers, model checkers.

**Rule:** A flaky test is a symptom of a real bug. Don't retry it green; find the bug.

## Summary

Concurrency is powerful but adds a whole new dimension to code complexity. Keep concurrency code
separate (SRP), minimize shared mutable state, use proven libraries and patterns, keep synchronized
blocks small, and take spurious test failures seriously.
