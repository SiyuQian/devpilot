---
name: devpilot-resolving-review-threads
description: Use when responding to inline review comments after pushing fixes to a GitHub PR or GitLab MR — decide per thread whether to auto-resolve (fix verified at HEAD), reply with technical reasoning (reviewer misread or wrong), or leave open for the reviewer. Triggers on "respond to review comments", "resolve the review threads", "I pushed the fixes, now reply to the comments", "close out the review", "/resolve-review". Do NOT use for writing a review (use devpilot-pr-review) or creating a PR (use devpilot-pr-creator).
---

# Resolving Review Threads

## Overview

After pushing fixes for a code review, every inline thread falls into one of three buckets. This skill is the decision matrix and the exact API calls for each bucket on GitHub and GitLab.

**Core principle:** Resolve only what you can prove is fixed at HEAD. Reply with technical reasoning, never with gratitude or hedging. Never resolve a thread you pushed back on — the reviewer does that.

**REQUIRED BACKGROUND:** superpowers:receiving-code-review — the no-gratitude and technical-rigor rules apply here verbatim.

## The Three Buckets

```
FOR each open review thread:

  Did I change code to address it?
  ├── YES → Does HEAD contain the change? (verify with git/grep, not memory)
  │         ├── YES → BUCKET A: resolve (+ optional one-line reply)
  │         └── NO  → STOP. Push the missing change first.
  └── NO  → Is the reviewer technically wrong, misreading, or asking for out-of-scope work?
            ├── YES → BUCKET B: reply with reasoning, LEAVE OPEN
            └── NO/UNSURE → BUCKET C: reply asking for clarification, LEAVE OPEN
```

**Never resolve buckets B or C.** Resolving a thread closes the conversation; doing it on a pushback signals "I've decided for you." The reviewer resolves when they accept your reasoning.

## Verify Before Resolving

Before bucket A:

```bash
# GitHub: confirm the fix is on the PR head, not just local
gh pr view <PR> --json headRefOid -q .headRefOid
git fetch origin && git log -p <headRefOid> -- <file> | grep -A2 <fix-marker>

# GitLab
glab mr view <IID> --output json | jq -r .sha
```

If the change is in your local branch but not pushed, push first. A resolved thread on a missing change is worse than no reply.

## Forbidden Reply Phrases

| ❌ Never write | ✅ Write instead |
|---|---|
| "Thanks!" / "Good catch!" / "Great point!" | (nothing — the resolved state is the acknowledgment) |
| "I think this may be a misread" | "This is a misread:" |
| "Happy to change if you feel strongly" | (state your reason; stop) |
| "Let me know and I'll resolve" | (the reviewer decides — don't prompt them) |
| "Sorry, you're right" | "You're right — [fact]. Pushed [sha]." |

**Why no gratitude:** Resolving the thread IS the acknowledgment. Adding "thanks" is performative noise that survives in the PR history forever.

**Why no hedging on technical pushback:** "I think this may be a misread" reads as uncertainty. If you verified it, state the fact. If you didn't verify, go verify before replying.

## Reply Templates

### Bucket A — fixed (reply optional, often empty)

Only reply when the fix is non-obvious from the diff:

> Pre-allocated with `make([]Refund, 0, len(items))` in <sha>.

For typos and trivial fixes: resolve with no reply.

### Bucket B — pushback

State the fact, cite evidence, stop. No softening tail.

> Misread — the function takes `ctx` from the caller (line 51) and threads it through. No `context.TODO()` in this file: `grep -n TODO refund.go` returns nothing.

> Intentional. Unique index on `refund_id` exists (migration `20240801_...`). Per `RAILS_CONVENTIONS.md` DB constraints are the source of truth; model validators are racy without the index.

> Out of scope. The design doc linked in the PR description chose at-most-once over transactional. Filing a follow-up issue if we want to revisit.

### Bucket C — clarify

> Can you point to the exact line? `grep` for `puts` in this file returns nothing at HEAD `<sha>`.

## GitHub Commands

GitHub review threads can only be resolved via **GraphQL** — REST cannot resolve.

```bash
# Reply in-thread (REST, uses the top-level comment id, NOT the thread id)
gh api -X POST \
  /repos/<owner>/<repo>/pulls/<pr>/comments/<comment_id>/replies \
  -f body="<reply text>"

# Resolve thread (GraphQL, uses thread node id PRRT_...)
gh api graphql -f query='
  mutation($id: ID!) {
    resolveReviewThread(input: {threadId: $id}) { thread { isResolved } }
  }' -F id=<PRRT_...>

# List all threads with ids + resolved state
gh api graphql -f query='
  query($owner:String!,$repo:String!,$pr:Int!){
    repository(owner:$owner,name:$repo){
      pullRequest(number:$pr){
        reviewThreads(first:100){
          nodes{ id isResolved comments(first:1){ nodes{ databaseId path body author{login} } } }
        }
      }
    }
  }' -F owner=<owner> -F repo=<repo> -F pr=<pr>
```

Companion mutation for mistakes: `unresolveReviewThread`.

## GitLab Commands

```bash
# Reply in a discussion
glab api --method POST \
  "/projects/<id>/merge_requests/<iid>/discussions/<discussion_id>/notes" \
  -f "body=<reply text>"

# Resolve discussion (toggles the whole thread)
glab api --method PUT \
  "/projects/<id>/merge_requests/<iid>/discussions/<discussion_id>?resolved=true"

# List unresolved discussions
glab api "/projects/<id>/merge_requests/<iid>/discussions" \
  | jq '[.[] | select(.notes[0].resolvable==true and .notes[-1].resolved==false)
         | {id, file: .notes[0].position.new_path, line: .notes[0].position.new_line, body: .notes[0].body, author: .notes[0].author.username}]'
```

## Workflow

1. **List unresolved threads** with the listing command above. Dump to `/tmp/threads.json`.
2. **Classify each into A/B/C** using the decision tree. Write the classification to a scratch file before executing anything.
3. **Verify bucket A items** are on HEAD. Drop to bucket C ("can't verify, need to repush") if not.
4. **Execute** in order: bucket A resolves first (clears noise), then B/C replies. One API call per thread; if a call fails, stop and investigate — don't continue.
5. **Print a summary** for the user: `N resolved, M replied-and-left-open, K need their input`.

## Rationalization Table

| Excuse | Reality |
|---|---|
| "I'll resolve all of them, the reviewer can reopen" | Reviewers don't reopen — your pushback dies silently. Leave B/C open. |
| "A 'thanks!' makes the reply friendlier" | It also makes you look performative and survives forever in the diff. Drop it. |
| "I should hedge in case I'm wrong" | If you might be wrong, verify before replying. Don't outsource the doubt to the reviewer. |
| "I'll resolve this stale comment myself" | If the reviewer was looking at old code, reply with HEAD evidence and let THEM resolve. |
| "The fix is in my branch, close enough" | The PR runs on the pushed head. Resolve only what's on HEAD. |
| "I'll batch all replies into one PR comment" | Inline thread replies stay anchored to the line. Top-level comments lose context. Reply in-thread. |

## Red Flags — Stop

- About to type "Thanks" or "Good catch" → delete, just resolve
- About to type "I think" or "Happy to" on a technical reply → state the fact instead
- About to resolve a thread you didn't change code for → only if you replied with verified evidence AND it's a stale-diff case AND the reviewer is clearly looking at old code; otherwise leave open
- About to resolve before pushing → push first

## Bottom Line

Resolve = "this is fixed at HEAD, conversation closed."
Reply-open = "here's my reasoning, your move."
Never the other way around.
