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

```java
// Bad — mixes HTTP parsing with business logic with DB access
public String login(HttpRequest req) {
    String user = req.getHeader("user").split("=")[1];
    if (db.query("SELECT * FROM users WHERE name='" + user + "'").isEmpty()) ...
}

// Good — each function one abstraction level
public String login(HttpRequest req) {
    User user = extractUser(req);
    return authenticate(user);
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
- Ask a question: `boolean fileExists("MyFile")`.
- Operate and return a transformed value: `InputStream fileOpen("MyFile")`.
- Event (take input, change state, no return): `passwordAttemptFailedNtimes(int attempts)`.

**Flag arguments are a smell.** `render(true)` screams "this function does two things." Split into
`renderForSuite()` and `renderForSingleTest()`.

**Argument objects.** When a function naturally takes 3+ related args, wrap them:
```java
Circle makeCircle(double x, double y, double radius);
Circle makeCircle(Point center, double radius);  // better
```

**Argument lists (variadic)** are acceptable when the args are truly a homogeneous list (`String.format`).

## Verbs and Keywords

- One-arg functions should form a verb/noun pair: `write(name)`.
- Better: keyword form encoding the argument's role: `writeField(name)`.

## Have No Side Effects

A function named `checkPassword` that *also* initializes a session is a lie. Callers now can't call
it without risking the session state. Either:
- Rename: `checkPasswordAndInitializeSession` (and deal with the fact that it does two things), or
- Split: `checkPassword()` and `initializeSession()` called separately.

**Output arguments** (modifying a parameter) are confusing. Prefer return values or `this`:
```java
appendFooter(s);       // does s get modified? is s the footer?
report.appendFooter(); // clear
```

## Command-Query Separation

Functions should either **do** something or **answer** something, but not both.

```java
// Confusing — does it check? does it set?
if (set("username", "unclebob")) ...

// Clear
if (attributeExists("username")) {
    setAttribute("username", "unclebob");
}
```

## Prefer Exceptions to Returning Error Codes

> **Language override:** Go, Rust, and Zig use explicit error returns. Follow the language's style
> skill. The principle below — don't force nested conditionals on callers — still applies via
> early-return guards.


Error codes force the caller to deal with error handling immediately and nest `if`s:
```java
if (deletePage(page) == E_OK) {
    if (registry.deleteReference(page.name) == E_OK) {
        if (configKeys.deleteKey(...) == E_OK) {
            logger.log("done");
        } else { ... }
    } else { ... }
}
```

With exceptions, the happy path is flat:
```java
try {
    deletePage(page);
    registry.deleteReference(page.name);
    configKeys.deleteKey(...);
} catch (Exception e) {
    logger.log(e.getMessage());
}
```

**Note:** Some languages (Go, Rust) prefer explicit error returns. Follow language idioms. The
principle — "don't force nested conditionals on callers" — still applies; use early-return guards.

## Extract Try/Catch Blocks

Error-handling is one thing. Separate it:
```java
public void delete(Page page) {
    try {
        deletePageAndAllReferences(page);
    } catch (Exception e) {
        logError(e);
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
