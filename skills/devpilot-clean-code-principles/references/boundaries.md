# Boundaries (Clean Code, Ch. 8)

Third-party code is written for broad applicability, not your specific needs. Your code is written
for your specific needs. Managing the boundary between them is where leaks and pain occur.

## Using Third-Party Code

Providers maximize applicability (breadth of API). Users want focused, safe, easy-to-use interfaces.
This tension leads to trouble.

Example: `Map<String, Sensor>`. `Map` exposes `clear()`, `putAll()`, etc. — any user of your object
can wipe the sensor collection. And every client must cast values retrieved by key. **Encapsulate
the Map.**

```java
public class Sensors {
    private Map<String, Sensor> sensors = new HashMap<>();

    public Sensor getById(String id) {
        return sensors.get(id);
    }
    // other Sensor-specific methods
}
```

Benefits:
- Casts are hidden inside `Sensors`.
- Client can't misuse the underlying Map.
- If you swap Map for another structure, no ripple.
- You can add sensor-specific invariants.

**Rule:** Don't return or accept third-party types at public API boundaries. Wrap them.

## Exploring and Learning Boundaries

You have to learn third-party code before you can use it. Reading docs isn't enough. Writing tests
that exercise the library is a cheap, focused way to learn:

### Learning Tests

Instead of stressing the library in your production code and debugging integration issues later,
write **isolated tests** that verify the library does what you think:

```java
@Test
public void testLogCreate() {
    Logger logger = Logger.getLogger("MyLogger");
    logger.info("hello");  // does this actually print?
}
```

Learning tests cost nothing — you'd have to learn the API anyway — and they leave you with a
regression suite that will break loudly when the library changes behavior on upgrade. This is worth
more than reading release notes.

## Using Code That Does Not Yet Exist

Sometimes you're waiting on another team's API. Don't block. Define the interface *you wish you had*
at the boundary:

```java
public interface Transmitter {
    void transmit(Frequency freq, Stream stream);
}
```

Implement your code against the interface. Build an adapter later when the real API lands.

## Clean Boundaries

Good software designs accommodate change without huge investments and rework. Boundaries are where
change enters.

- Depend on code **you control**, not code you don't.
- Wrap third-party APIs in adapters you own.
- Use dependency injection to swap implementations.
- Keep third-party types from leaking into the core domain.

## Summary

Put clear separation between your code and third-party code: wrappers, adapters, learning tests. You
protect against change, improve readability, and decouple your design from the provider's
design choices.
