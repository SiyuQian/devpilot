# Comments (Clean Code, Ch. 4)

> **Language override:** Go *requires* a doc comment on every exported (capitalized) identifier,
> starting with the name: `// User represents …`, `// Save writes …`. Clean Code's "comments are a
> failure" stance applies to **inline/explanatory** comments that compensate for unclear code, not
> to API documentation. Godoc/TSDoc/JSDoc on exported surface is mandatory, not a smell.

> Don't comment bad code — rewrite it. — Brian Kernighan & P.J. Plauger

Comments are, at best, a necessary evil. They compensate for our failure to express ourselves in
code. Every time you write a comment, feel a small failure — and try to rename or restructure first.

**Comments lie.** Code changes; comments don't always. Over time they drift from reality and mislead
the reader.

## Let the Code Speak

```ts
// ❌ Comment compensates for unclear predicate
if ((employee.flags & HOURLY_FLAG) !== 0 && employee.age > 65) ...

// ✅ Extract a named method; no comment needed
if (employee.isEligibleForFullBenefits()) ...
```

```go
// Same in Go — extract a method that names the intent
if employee.IsEligibleForFullBenefits() {
    ...
}
```

## Good Comments

### Legal Comments
Copyright headers required by policy or license.

### Informative Comments
When the information can't live in the code itself:
```ts
// format matched kk:mm:ss EEE, MMM dd, yyyy
const timestampPattern = /\d*:\d*:\d* \w*, \w* \d*, \d*/;
```

```go
// format matched kk:mm:ss EEE, MMM dd, yyyy
var timestampPattern = regexp.MustCompile(`\d*:\d*:\d* \w*, \w* \d*, \d*`)
```
(Even better: extract to a well-named constant instead of a regex literal.)

### Explanation of Intent
When the *why* isn't obvious:
```go
// We run many goroutines on purpose to make sure the queue handles contention.
for i := 0; i < 2500; i++ {
    go producer.send(ctx, payload)
}
```

### Clarification
Translating obscure return values or parameters you can't rename (e.g. library code):
```go
if got := a.CompareTo(a); got != 0 { // a == a
    t.Errorf("self-compare = %d, want 0", got)
}
```
Risk: the comment might be wrong. Prefer making the code clear.

### Warning of Consequences
```go
// Don't run unless you have time to kill.
func TestWithReallyBigFile(t *testing.T) {
    if testing.Short() { t.Skip("skipping large-file test in -short mode") }
    ...
}
```

### TODO Comments
Acceptable if they have a ticket reference or owner. Review them periodically.
```ts
// TODO(#1234): replace with new API after v2 rollout
```

```go
// TODO(#1234): replace with new API after v2 rollout
```

### Amplification
Amplify the importance of something that might seem inconsequential:
```ts
// The trim is important — it strips leading spaces that would otherwise
// cause this item to be parsed as a nested list.
const item = listItem.trimStart();
```

### Public API Doc Comments (Godoc / TSDoc)
**Go:** every exported identifier gets a doc comment starting with the name. Non-negotiable.
**TypeScript:** TSDoc on public library APIs; internal exported symbols in a private module don't need them if names are clear. Write carefully — they're the contract.

## Bad Comments

### Mumbling
A comment that you wrote just because you felt you should. If it's unclear to the reader, it's noise.

### Redundant Comments
```ts
// Utility method that returns when this.closed is true. Throws otherwise.
async waitForClose(timeoutMillis: number): Promise<void> {
  if (!this.closed) { ... }
}
```
The code already says this. The comment takes longer to read than the method.

### Misleading Comments
Worse than redundant — the comment says something subtly different from the code. Readers trust the
comment and get burned.

### Mandated Comments
Every function has a Javadoc, every variable a comment — clutter, lies, disorganization. Mandate
clear code instead.

### Journal Comments
`// 2008-03-14: Fixed bug in...`. Version control is the journal. Delete.

### Noise Comments
```ts
/** The day of the month. */
private dayOfMonth: number;

/** Default constructor. */
constructor() {}
```
Both say nothing the code doesn't already say. Delete.

### Don't Use a Comment When You Can Use a Function or Variable

```ts
// ❌ Comment explaining a dense expression
// does the module from the global list <mod> depend on the subsystem we are part of?
if (smodule.dependSubsystems().includes(subSysMod.subsystem())) ...

// ✅ Explanatory variables — no comment needed
const moduleDependees = smodule.dependSubsystems();
const ourSubsystem = subSysMod.subsystem();
if (moduleDependees.includes(ourSubsystem)) ...
```

### Position Markers
```ts
// Actions //////////////////////////////////
```
Tolerable very occasionally. Usually, noise.

### Closing Brace Comments
```ts
} // end for
} // end while
} // end main
```
If your function is so long you need these, shorten the function.

### Attributions and Bylines
```ts
/* Added by Rick */
```
VCS knows. Delete.

### Commented-Out Code

**Delete it.** VCS remembers. Commented-out code rots: no one dares remove it ("maybe it's important")
and it accumulates forever.

### HTML Comments

Comments in source meant to be rendered as HTML in tooling output — move formatting concerns out of
source.

### Non-local Information

Describing system-wide policy in a local function comment. It'll drift.

### Too Much Information

Historical discussions or unnecessary details. Keep comments lean.

### Inobvious Connection
If the connection between the comment and the code isn't clear, the comment fails.

### Function Headers
Short functions with good names don't need headers.
