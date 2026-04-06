---
name: devpilot-confluence-reviewer
description: Review Atlassian Confluence pages and leave comments (both inline and page-level). Use this skill when the user asks to review, give feedback on, or comment on a Confluence page. Triggers on Confluence page URLs, "review this confluence page", "comment on the doc", "give feedback on the wiki page", or any mention of reviewing Confluence content.
license: Complete terms in LICENSE.txt
---

# Confluence Page Reviewer

Review Confluence pages using parallel specialized agents that each focus on a different review dimension, then post structured feedback as inline comments and a page-level summary.

## Credentials

This skill uses Confluence REST API with Basic Auth (email + API token). Credentials are stored in `~/.config/devpilot/credentials.json` under the `"confluence"` key.

Check for credentials first:

```bash
python3 -c "import json; c=json.load(open('$HOME/.config/devpilot/credentials.json'))['confluence']; print('OK')" 2>/dev/null
```

If credentials are missing, ask the user to provide three values:

1. **Domain** — the subdomain from their Confluence URL (e.g., `mycompany` from `mycompany.atlassian.net`)
2. **Email** — their Atlassian account email
3. **API Token** — generated at https://id.atlassian.com/manage/api-tokens (use "Create API token", the classic non-scoped variant; ensure the token has Confluence access)

Then store them:

```bash
python3 -c "
import json, os, pathlib
path = pathlib.Path(os.path.expanduser('~/.config/devpilot/credentials.json'))
path.parent.mkdir(parents=True, exist_ok=True)
data = json.loads(path.read_text()) if path.exists() else {}
data['confluence'] = {'domain': 'DOMAIN', 'email': 'EMAIL', 'api_token': 'TOKEN'}
path.write_text(json.dumps(data, indent=2))
os.chmod(path, 0o600)
"
```

Replace `DOMAIN`, `EMAIL`, and `TOKEN` with the user's values.

### Reading Credentials

```bash
CONFLUENCE_DOMAIN=$(python3 -c "import sys,json; print(json.load(open('$HOME/.config/devpilot/credentials.json'))['confluence']['domain'])")
CONFLUENCE_EMAIL=$(python3 -c "import sys,json; print(json.load(open('$HOME/.config/devpilot/credentials.json'))['confluence']['email'])")
CONFLUENCE_TOKEN=$(python3 -c "import sys,json; print(json.load(open('$HOME/.config/devpilot/credentials.json'))['confluence']['api_token'])")
CONFLUENCE_BASE="https://${CONFLUENCE_DOMAIN}.atlassian.net"
```

Use `-u "$CONFLUENCE_EMAIL:$CONFLUENCE_TOKEN"` for Basic Auth in all curl calls.

## Review Workflow

Follow these steps precisely:

### Step 1: Fetch Page Content

Identify the page from the user's input. Extract the page ID from the URL:

**Standard URL:** `https://{domain}.atlassian.net/wiki/spaces/{SPACE}/pages/{PAGE_ID}/{Title}` — the page ID is the numeric segment after `/pages/`.

**Tiny link:** `https://{domain}.atlassian.net/wiki/x/{encoded}` — ask the user for the full page URL, or search by title instead.

Fetch the page using the **v1 API** with both storage and view formats:

```bash
curl -s -u "$CONFLUENCE_EMAIL:$CONFLUENCE_TOKEN" \
  -H "Accept: application/json" \
  "$CONFLUENCE_BASE/wiki/rest/api/content/{PAGE_ID}?expand=body.storage,body.view,version,space"
```

Extract readable text from `body.view.value` by stripping HTML tags — this is the rendered text needed for inline comment anchoring. Also keep `body.storage.value` for understanding document structure.

**Search by title** (when no page ID is available):

```bash
curl -s -u "$CONFLUENCE_EMAIL:$CONFLUENCE_TOKEN" \
  -H "Accept: application/json" \
  "$CONFLUENCE_BASE/wiki/rest/api/content/search?cql=title%3D%22Exact+Page+Title%22&expand=body.storage,body.view,version"
```

### Step 2: Use a Haiku agent to produce a page summary

Launch a Haiku agent with the full page text. Ask it to return:
- A brief summary of what the page is about (2-3 sentences)
- The document type (e.g., architecture doc, RFC, runbook, product spec, meeting notes)
- A list of the major sections and their topics

This summary will be provided to each review agent for context.

### Step 3: Launch 5 parallel review agents

Launch 5 parallel **Sonnet** agents, each focusing on a different review dimension. Provide each agent with the full page text, the page summary from Step 2, and any context the user provided about what kind of review they want.

Each agent should return a list of issues found, where each issue includes:
- The **exact rendered text** to anchor the inline comment to (a sentence or distinctive phrase from the page — this must be an exact substring of the rendered page text, not the XHTML)
- The **comment text** (the feedback)
- A **category** label (matching the agent's focus area)
- A **confidence score** from 0-100

The 5 agents and their focus areas:

**Agent 1 — Clarity & Structure:**
Review the document for logical organization, clear headings, readable flow, and whether sections are in a sensible order. Flag sections that are hard to follow, walls of text that should be broken up, missing transitions, or confusing structure. Check that the document can be understood by its intended audience without excessive context.

**Agent 2 — Technical Accuracy & Consistency:**
Review technical claims, architecture descriptions, and system details for correctness and internal consistency. Flag contradictions between sections, outdated technical references, incorrect terminology, and claims that don't match standard practice. Check that tables, diagrams descriptions, and lists are consistent with the prose.

**Agent 3 — Completeness & Gaps:**
Review whether the document covers what it claims to cover. Flag missing sections that a reader would expect, undefined terms or acronyms, unanswered questions that the document raises but doesn't resolve, and areas where the reader is left guessing. Check that decision points have clear options and criteria.

**Agent 4 — Actionability & Decisions:**
Review whether the document leads to clear outcomes. Flag vague action items, decisions without owners or timelines, risks without mitigations, and next steps that are unclear. Check that any "key decisions" or "open questions" sections have enough context for the reader to make those decisions.

**Agent 5 — Language & Polish:**
Review for typos, grammatical errors, inconsistent terminology, and unclear phrasing. Flag sentences that are ambiguous, jargon that isn't explained, and formatting inconsistencies. Focus only on issues that affect comprehension — avoid pedantic nitpicks.

### Step 4: Score and filter issues

For each issue returned by the review agents, use the following confidence scale (provide this rubric to the agents in Step 3):

- **0**: False positive — not a real issue, or purely subjective preference.
- **25**: Minor — might be an issue but could also be intentional. Not worth commenting on.
- **50**: Moderate — real issue but a nitpick. Wouldn't block the document.
- **75**: Important — verified issue that affects document quality. Readers would benefit from a fix.
- **100**: Critical — clear error, contradiction, or missing information that undermines the document's purpose.

Filter out any issues with a score below **50**. If the user asked for a thorough review, lower the threshold to 25.

Deduplicate issues — if multiple agents flagged the same text or the same concern, merge them into a single comment combining the feedback.

### Step 5: Post inline comments

For each surviving issue, post an inline comment. The `textSelection` must be an exact substring of the **rendered page text** (from `body.view`), not the raw XHTML storage format.

Tips for reliable text selection:
- Use a full sentence or distinctive phrase that is unique on the page
- Avoid text that spans across formatting boundaries (e.g., across a bold marker)
- If unsure about uniqueness, use a longer snippet

```bash
curl -s -u "$CONFLUENCE_EMAIL:$CONFLUENCE_TOKEN" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -X POST "$CONFLUENCE_BASE/wiki/api/v2/inline-comments" \
  -d '{
    "pageId": "PAGE_ID",
    "body": {
      "representation": "storage",
      "value": "<p>Your comment here.</p>"
    },
    "inlineCommentProperties": {
      "textSelection": "exact rendered text from the page",
      "textSelectionMatchCount": 1,
      "textSelectionMatchIndex": 0
    }
  }'
```

Post inline comments one at a time. If one fails (400 error usually means the text selection didn't match), log it and continue with the rest. Do not retry failed comments with guessed text.

### Step 6: Post summary comment

Post a page-level footer comment summarizing the review. Structure it as:

```html
<h3>Review Summary</h3>
<p><strong>Overall:</strong> [1-2 sentence overall assessment]</p>
<p><strong>Strengths:</strong></p>
<ul>
  <li>[What the document does well]</li>
</ul>
<p><strong>Key Issues:</strong></p>
<ul>
  <li>[Top issues grouped by theme, not repeating every inline comment]</li>
</ul>
<p><strong>Recommendations:</strong></p>
<ul>
  <li>[Concrete next steps]</li>
</ul>
```

```bash
curl -s -u "$CONFLUENCE_EMAIL:$CONFLUENCE_TOKEN" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -X POST "$CONFLUENCE_BASE/wiki/api/v2/footer-comments" \
  -d '{
    "pageId": "PAGE_ID",
    "body": {
      "representation": "storage",
      "value": "<h3>Review Summary</h3>..."
    }
  }'
```

### Step 7: Report to user

After posting, summarize what you did:
- Number of inline comments posted (and any that failed, with reason)
- Brief recap of the summary comment
- Link to the page so the user can see the comments in context

## Comment Style Guidelines

- Be constructive — frame suggestions positively ("Consider adding..." not "This is missing...")
- Be specific — reference the exact content rather than making vague observations
- Keep inline comments focused on one point each
- Use questions to surface ambiguity ("Does this apply to all environments, or just production?")
- The summary comment should be useful on its own — someone reading only the summary should understand the key takeaways
- Prefix each inline comment with a category tag like `[Clarity]`, `[Technical]`, `[Gap]`, `[Action]`, or `[Polish]` so the author can prioritize

## API Quick Reference

| Operation | Method | Endpoint |
|-----------|--------|----------|
| Get page by ID | GET | `/wiki/rest/api/content/{id}?expand=body.storage,body.view,version,space` |
| Search by title (CQL) | GET | `/wiki/rest/api/content/search?cql=title%3D%22{title}%22` |
| Inline comment | POST | `/wiki/api/v2/inline-comments` |
| Footer comment | POST | `/wiki/api/v2/footer-comments` |

Note: Use v1 API (`/wiki/rest/api/content/...`) for fetching pages — it has broader compatibility across Confluence instances. The v2 comment endpoints (`/wiki/api/v2/inline-comments` and `/wiki/api/v2/footer-comments`) are used for posting comments.
