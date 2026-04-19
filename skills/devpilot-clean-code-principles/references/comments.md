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

```java
// Check to see if the employee is eligible for full benefits
if ((employee.flags & HOURLY_FLAG) && (employee.age > 65)) ...

// Better
if (employee.isEligibleForFullBenefits()) ...
```

## Good Comments

### Legal Comments
Copyright headers required by policy or license.

### Informative Comments
When the information can't live in the code itself:
```java
// format matched kk:mm:ss EEE, MMM dd, yyyy
Pattern p = Pattern.compile("\\d*:\\d*:\\d* \\w*, \\w* \\d*, \\d*");
```
(Even better: extract to a well-named constant.)

### Explanation of Intent
When the *why* isn't obvious:
```java
// We're running many threads on purpose to make sure the queue handles contention.
for (int i = 0; i < 2500; i++) new Thread(...).start();
```

### Clarification
Translating obscure return values or parameters you can't rename (e.g. library code):
```java
assertTrue(a.compareTo(a) == 0); // a == a
```
Risk: the comment might be wrong. Prefer making the code clear.

### Warning of Consequences
```java
// Don't run unless you have some time to kill.
public void _testWithReallyBigFile() { ... }
```

### TODO Comments
Acceptable if they have a ticket reference or owner. Review them periodically.
```java
// TODO(#1234): replace with new API after v2 rollout
```

### Amplification
Amplify the importance of something that might seem inconsequential:
```java
// The trim is real important. It removes the starting spaces
// that could cause the item to be recognized as another list.
```

### Public API Javadoc / Godoc / Docstrings
Required for documented public APIs. Write carefully; they're the contract.

## Bad Comments

### Mumbling
A comment that you wrote just because you felt you should. If it's unclear to the reader, it's noise.

### Redundant Comments
```java
// Utility method that returns when this.closed is true. Throws an exception otherwise.
public synchronized void waitForClose(final long timeoutMillis) throws Exception {
    if (!closed) { ... }
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
```java
/** Default constructor. */
protected AnnualDateRule() { }

/** The day of the month. */
private int dayOfMonth;
```

### Don't Use a Comment When You Can Use a Function or Variable

```java
// does the module from the global list <mod> depend on the subsystem we are part of?
if (smodule.getDependSubsystems().contains(subSysMod.getSubSystem())) ...

// Better
ArrayList moduleDependees = smodule.getDependSubsystems();
String ourSubSystem = subSysMod.getSubSystem();
if (moduleDependees.contains(ourSubSystem)) ...
```

### Position Markers
```java
// Actions //////////////////////////////////
```
Tolerable very occasionally. Usually, noise.

### Closing Brace Comments
```java
} // end for
} // end while
} // end main
```
If your function is so long you need these, shorten the function.

### Attributions and Bylines
```java
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
