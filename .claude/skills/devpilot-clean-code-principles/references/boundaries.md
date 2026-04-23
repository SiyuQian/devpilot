# Boundaries (Clean Code, Ch. 8)

Third-party code is written for broad applicability, not your specific needs. Your code is written
for your specific needs. Managing the boundary between them is where leaks and pain occur.

## Using Third-Party Code

Providers maximize applicability (breadth of API). Users want focused, safe, easy-to-use interfaces.
This tension leads to trouble.

Example: a `Map<string, Sensor>` exposes `clear`, `delete`, and iteration — any caller can wipe or
mutate the collection. **Encapsulate it.**

```ts
class Sensors {
  private sensors = new Map<string, Sensor>();

  getById(id: string): Sensor | undefined { return this.sensors.get(id); }
  // other sensor-specific methods; no raw Map leaks
}
```

```go
// Go — same idea; the map is unexported, only chosen methods are public
type Sensors struct {
    sensors map[string]*Sensor
}
func (s *Sensors) ByID(id string) (*Sensor, bool) {
    v, ok := s.sensors[id]
    return v, ok
}
```

Benefits:
- Callers can't misuse the underlying structure.
- If you swap the Map for something else, no ripple.
- You can enforce sensor-specific invariants in one place.

**Rule:** Don't return or accept third-party types at public API boundaries. Wrap them.

## Exploring and Learning Boundaries

You have to learn third-party code before you can use it. Reading docs isn't enough. Writing tests
that exercise the library is a cheap, focused way to learn:

### Learning Tests

Instead of stressing the library in your production code and debugging integration issues later,
write **isolated tests** that verify the library does what you think:

```ts
// TypeScript — Jest learning test
test("pino info includes level and msg", () => {
  const chunks: string[] = [];
  const stream = { write: (s: string) => chunks.push(s) };
  const logger = pino(stream);
  logger.info("hello");
  expect(JSON.parse(chunks[0])).toMatchObject({ level: 30, msg: "hello" });
});
```

```go
// Go — learning test against a third-party client
func TestRedisClient_SetGet(t *testing.T) {
    c := redis.NewClient(&redis.Options{Addr: miniredisAddr(t)})
    t.Cleanup(func() { c.Close() })

    if err := c.Set(ctx, "k", "v", 0).Err(); err != nil { t.Fatal(err) }
    got, err := c.Get(ctx, "k").Result()
    if err != nil || got != "v" {
        t.Fatalf("Get = %q, %v; want v, nil", got, err)
    }
}
```

Learning tests cost nothing — you'd have to learn the API anyway — and they leave you with a
regression suite that will break loudly when the library changes behavior on upgrade. This is worth
more than reading release notes.

## Using Code That Does Not Yet Exist

Sometimes you're waiting on another team's API. Don't block. Define the interface *you wish you had*
at the boundary:

```ts
interface Transmitter {
  transmit(freq: Frequency, stream: ReadableStream): Promise<void>;
}
```

```go
type Transmitter interface {
    Transmit(ctx context.Context, freq Frequency, stream io.Reader) error
}
// In Go, define this interface in the CONSUMER package (per devpilot-google-go-style),
// not alongside an eventual implementation.
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
