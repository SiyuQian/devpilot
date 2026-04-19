# Formatting (Clean Code, Ch. 5)

Formatting is communication, and communication is the professional developer's first order of
business.

**Team rules trump personal preference.** Once agreed, enforce with an automatic formatter.

## The Newspaper Metaphor

A source file should read like a newspaper article:
- **Headline** (name) tells you if you're in the right place.
- **Top** — high-level summary (public interface, main flow).
- **Details** grow as you descend.

## Vertical Formatting

### File Size

Small is better. 200 lines is easy to reason about. 500 lines starts to hurt. Files over 1000 lines
are a smell.

### Vertical Openness Between Concepts

Blank lines between groups of related lines signal transitions of thought:
```java
package fitnesse.wikitext.widgets;

import java.util.regex.*;

public class BoldWidget extends ParentWidget {
    public static final String REGEXP = "'''.+?'''";
    private static final Pattern pattern = Pattern.compile("'''(.+?)'''", ...);

    public BoldWidget(ParentWidget parent, String text) {
        super(parent);
        Matcher match = pattern.matcher(text);
        match.find();
        addChildWidgets(match.group(1));
    }
}
```

### Vertical Density

Lines of code that are tightly related should appear vertically dense. Don't break them up with
blank lines or useless comments.

### Vertical Distance

Concepts closely related should be kept close. Things that aren't related shouldn't be in the same
file.

- **Variable declarations** should appear as close to their usage as possible. Local variables at the
  top of the function. Loop variables in the loop.
- **Instance variables** at the top of the class (or bottom — pick one, consistently).
- **Dependent functions**: if one function calls another, put the caller above the callee so readers
  can scroll down to drill in.
- **Conceptual affinity**: functions that perform similar operations belong together, even without
  direct calls.

### Vertical Ordering

General → specific. Top → bottom. The reader shouldn't need to jump around.

## Horizontal Formatting

### Line Width

80–120 characters. Longer lines hurt scanability.

### Horizontal Openness and Density

Space around operators for separation:
```java
int lineSize = line.length();
totalChars += lineSize;
lineWidthHistogram.addLine(lineSize, lineCount);
```

No space between function name and its open parenthesis:
```java
measureLine(line);  // function is a unit with its args
```

Tighter binding for higher-precedence operators:
```java
b*b - 4*a*c
```

### Horizontal Alignment

Don't align variable names or right-hand sides — the alignment draws the eye away from meaning and
becomes a maintenance burden. Just use single spaces.

### Indentation

Make the hierarchy visible. Never collapse scopes onto one line:
```java
// Bad
public class CommentWidget extends TextWidget { public static final String REGEXP = "..."; public CommentWidget(...) { super(...); } }

// Good: actually indent.
```

### Dummy Scopes

Occasionally you'll write an empty loop body. Make the semicolon visible on its own line:
```java
while (dis.read(buf, 0, readBufferSize) != -1)
    ;
```

## Team Rules

Every programmer has formatting preferences, but professionals subordinate theirs to the team's.
Consistency across the codebase matters more than any single rule. Configure the formatter once and
let it decide forever.
