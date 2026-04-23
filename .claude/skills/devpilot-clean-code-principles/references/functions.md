# Functions (Clean Code, Ch. 3)

## Small

The first rule of functions is that they should be small. The second rule is that they should be
smaller than that.

- Aim for **<20 lines**; most functions should be **<10 lines**.
- Blocks inside `if`, `else`, `while` statements should be one line — probably a function call. That
  line can have a descriptive name.
- Indent level should not exceed **1 or 2**.

## Do One Thing

> Functions should do one thing. They should do it well. They should do it only.

A function does one thing when:
- You cannot extract another function from it with a name that isn't just a restatement of its implementation.
- All statements inside it are at the same level of abstraction.

If your function has *sections* (comments labeling groups of statements), each section is a function
waiting to be born.

## One Level of Abstraction per Function

Mixing high-level policy and low-level detail in the same function confuses readers. The **Stepdown
Rule**: code should read top-down, each function followed by those at the next level of abstraction.

```ts
// ❌ Bad — mixes HTTP parsing, SQL construction, and business logic
function login(req: Request) {
  const user = req.headers["user"]?.split("=")[1];
  if ((await db.query(`SELECT * FROM users WHERE name='${user}'`)).length === 0) ...
}

// ✅ Good — each function at one abstraction level
function login(req: Request) {
  const user = extractUser(req);
  return authenticate(user);
}
```

```go
// ✅ Same idea in Go — top-level function reads like policy
func Login(ctx context.Context, req *http.Request) (*Session, error) {
    user, err := extractUser(req)
    if err != nil {
        return nil, err
    }
    return authenticate(ctx, user)
}
```

## Use Descriptive Names

Long descriptive names are better than short cryptic ones. A name like
`includeSetupAndTeardownPagesIntoPageData` is fine if it tells the truth. The function body should
do exactly what the name says — no more, no less.

## Function Arguments

**Fewer is better.** Ideal: 0. Next: 1. Then 2. **3 is a smell**, 4+ requires justification.

**Why:** Each argument is another thing the reader must hold in their head. Testing multiplies with
argument combinations.

**Types of one-argument functions:**
- Ask a question: `fileExists("MyFile"): boolean`.
- Operate and return a transformed value: `openFile("MyFile"): ReadStream`.
- Event (take input, change state, no return): `passwordAttemptFailedNTimes(n: number): void`.

**Flag arguments are a smell.** `render(true)` screams "this function does two things." Split into
`renderForSuite()` and `renderForSingleTest()`. This applies to both TS booleans and Go `bool` params.

**Argument objects.** When a function naturally takes 3+ related args, wrap them:

```ts
// TypeScript — options object
makeCircle(x: number, y: number, radius: number);
makeCircle(center: Point, radius: number);  // better
```

```go
// Go — named struct for options
func MakeCircle(x, y, radius float64) Circle          // crowded
func MakeCircle(center Point, radius float64) Circle  // better
// For many optional args, use a Config/Options struct or functional options.
```

**Variadic args** (`...T` in both Go and TS) are fine for truly homogeneous lists (`fmt.Sprintf`,
`console.log`). They are not a way to hide a flag argument.

## Verbs and Keywords

- One-arg functions should form a verb/noun pair: `write(name)`.
- Better: keyword form encoding the argument's role: `writeField(name)`.

## Have No Side Effects

A function named `checkPassword` that *also* initializes a session is a lie. Callers now can't call
it without risking the session state. Either:
- Rename: `checkPasswordAndInitializeSession` (and deal with the fact that it does two things), or
- Split: `checkPassword()` and `initializeSession()` called separately.

**Output arguments** (mutating a parameter) are confusing.

```ts
// ❌ Unclear — does it modify s, or is s the footer?
appendFooter(s);

// ✅ The receiver is the thing being modified
report.appendFooter();
```

```go
// Go convention: if the function mutates, use a pointer receiver (method)
// or return the new value. Don't take a *T parameter whose mutation is
// surprising to the caller.
func (r *Report) AppendFooter()        // clear
func appendFooter(r *Report)            // only OK if r is obviously the target
```

## Command-Query Separation

Functions should either **do** something or **answer** something, but not both.

```ts
// ❌ Confusing — does it check existence, or does it set?
if (set("username", "unclebob")) ...

// ✅ Query then command
if (attributes.has("username")) {
  attributes.set("username", "unclebob");
}
```

(Go applies the same principle: `func Set(...) bool` that *both* assigns and reports prior presence
is a CQS violation — prefer `Has(...) bool` followed by `Set(...)`.)

## Prefer Exceptions to Returning Error Codes

> **Language override:** Go, Rust, and Zig use explicit error returns. Follow the language's style
> skill. The principle below — don't force nested conditionals on callers — still applies via
> early-return guards.


Error codes force the caller to deal with error handling immediately and nest `if`s:

```ts
// ❌ Nested error-code style (bad in any language)
if (deletePage(page) === E_OK) {
  if (registry.deleteReference(page.name) === E_OK) {
    if (configKeys.deleteKey(...) === E_OK) {
      logger.info("done");
    } else { ... }
  } else { ... }
}

// ✅ In TypeScript — throw; happy path is flat
try {
  deletePage(page);
  registry.deleteReference(page.name);
  configKeys.deleteKey(...);
} catch (err) {
  logger.error(err);
}
```

```go
// ✅ In Go — early-return guards keep happy path at the left margin
if err := deletePage(page); err != nil {
    return fmt.Errorf("delete page: %w", err)
}
if err := registry.DeleteReference(page.Name); err != nil {
    return fmt.Errorf("delete reference: %w", err)
}
if err := configKeys.DeleteKey(...); err != nil {
    return fmt.Errorf("delete key: %w", err)
}
```

Same principle: don't nest; let errors terminate the current function so the reader sees a
sequence, not a pyramid.

## Extract Try/Catch Blocks

Error-handling is one thing. Separate it:

```ts
function deletePage(page: Page) {
  try {
    deletePageAndAllReferences(page);
  } catch (err) {
    logError(err);
  }
}
// deletePageAndAllReferences contains only the happy path
```

```go
func DeletePage(ctx context.Context, page *Page) {
    if err := deletePageAndAllReferences(ctx, page); err != nil {
        logError(err)
    }
}
```

## Don't Repeat Yourself

Duplication is the root of most maintenance pain. Subroutines, inheritance, AOP, template methods —
all exist to eliminate duplication.

## Structured Programming

- One entry, one exit — **but**: for small functions, multiple `return`s / `break`s can be clearer
  than contorting the code. Use judgment.
- Never `goto` (in languages where it exists).

## How Do You Write Functions Like This?

Nobody writes clean functions on the first try. Write it to work, then massage:
- Extract sub-functions.
- Rename.
- Eliminate duplication.
- Shorten methods.
- Run tests after every change.
