---
name: devpilot-daily-toolkit
description: >
  Personal daily-life toolkit for someone living in Auckland, NZ. Bundles four
  tools behind one skill: RSS news (世界科技 / AI / 世界政治 / 经济金融 / 新西兰),
  real-time exchange rates (any currency pair via cdn.moneyconvert.net), GitHub
  daily activity reports (commits / PRs / issues across the user's active repos),
  and NZ public holidays (Auckland-highlighted via Nager.Date). USE WHEN the
  user asks for any of: a personal daily / morning briefing, "my daily digest",
  exchange rates ("汇率", "USD to NZD", "100 AUD 换多少人民币"), NZ public
  holidays ("下个假期", "next NZ holiday", "Auckland anniversary day"), a daily
  GitHub activity report ("今日 github", "github daily report", "what did I
  push today"), or a blended "give me everything" briefing that combines all
  four. Triggers on: "daily toolkit", "my daily", "morning briefing", "今日简报",
  "晨报", "汇率", "exchange rate", "currency convert", "convert X to Y", "NZ
  holiday", "新西兰假期", "Auckland holiday", "github daily report", "今日 github",
  "what did I ship today". For a pure long-form general news digest with no
  other topics, prefer devpilot-news-digest; for everyday Auckland-local
  utilities and combined briefings, use this skill.
---

# Daily Toolkit (Auckland personal helper)

Router for four daily-life utilities — news, exchange rates, GitHub activity,
and NZ public holidays — plus a combined morning-briefing mode that runs all
four and assembles one digest.

The four tools are self-contained Python scripts under `scripts/`. The skill's
job is to (1) pick the right tool for the user's request, (2) call it with the
right arguments, and (3) format the raw output into something readable —
especially for the news tool, whose output is raw English RSS that needs to
be translated and summarized into Chinese.

## Tools at a glance

| Intent | Script | What it returns |
|--------|--------|-----------------|
| News by category | `scripts/fetch_news.py` | Raw English RSS items grouped into 5 categories |
| Exchange rates | `scripts/get_rate.py` | Single pair, multi-target, or cross-rate matrix |
| GitHub daily activity | `scripts/github_daily_report.py` | Today's commits / PRs / issues across active repos |
| NZ public holidays | `scripts/get_holidays.py` | National + Auckland holidays + next-holiday countdown |

Dependencies: `feedparser` (news), `requests` (exchange rate), `gh` CLI logged
in (github report). Holidays uses stdlib only. If `feedparser` or `requests`
is missing, install with `pip install feedparser requests` before running.

## Routing the user's intent

Before running anything, decide which tool the user actually wants. Use these
heuristics:

- **News** — "今日新闻", "daily news", "today's headlines", "what's happening",
  "新西兰新闻". Output is raw RSS — you must translate/summarize into Chinese
  (see *News post-processing* below).
- **Exchange rate** — any currency code (NZD, AUD, USD, CNY, EUR, GBP, JPY,
  KRW, HKD, …), "汇率", "1000 AUD 换多少 CNY", "convert X to Y", "USD to NZD".
- **GitHub daily activity** — "今日 github", "github daily report", "what did
  I ship/push today", "commits today", "daily report".
- **NZ holidays** — "下个假期", "next holiday", "is X a public holiday",
  "Auckland anniversary day", "labour day", "新西兰公共假期".
- **Combined briefing** — "morning briefing", "my daily", "今日简报", "晨报",
  "give me everything", "daily digest". Run all four and assemble (see
  *Combined briefing mode*).

If the user's intent is genuinely ambiguous (e.g., just "daily" with no other
context), ask **one** short clarifying question before running. Don't run all
four when the user only wanted one — that wastes time and tokens.

## Tool 1 — News

```bash
python3 scripts/fetch_news.py
```

The script reads `feeds.json` (sibling of `scripts/`) and fetches RSS feeds in
five categories: 世界科技动态, AI 专项, 世界政治, 经济/金融, 新西兰新闻.
Output is raw English entries grouped by category, 6 items per category max,
deduplicated by title prefix.

### News post-processing

The raw output is in English. The user almost always wants Chinese, so after
running the script:

1. Preserve the five categories in the original order.
2. For each item, translate the title into Chinese and write a 2–3 sentence
   Chinese summary based on the English `summary` text. Don't hallucinate
   details that aren't in the summary — if the summary is empty, just give
   the translated headline.
3. Keep the `[source | YYYY-MM-DD]` attribution.
4. If something is clearly major breaking news (war, market crash, disaster,
   major political event), pin it at the top with a 📌 marker. Use judgment;
   don't over-pin.

If the script exits with `ERROR: All feeds failed to fetch.`, treat that as a
network problem — tell the user, don't pretend you got news.

## Tool 2 — Exchange rate

`get_rate.py` is a small CLI. Pick the form that fits the request — don't run
the default if the user named specific currencies.

```bash
# Default: NZD → CNY/AUD/USD (only use if user said nothing specific)
python3 scripts/get_rate.py

# Specific pair: base target
python3 scripts/get_rate.py AUD CNY

# With amount: amount base target
python3 scripts/get_rate.py 1000 AUD CNY

# Custom base + target list
python3 scripts/get_rate.py --base USD CNY EUR JPY

# Cross-rate matrix
python3 scripts/get_rate.py --cross NZD AUD USD CNY
```

Currency codes are uppercase ISO codes. The default and `--base` forms accept
lowercase and uppercase the args internally; `--cross` is the one form that
does NOT auto-uppercase, so always pass uppercase to it. The script already
knows the Chinese names of common ones (CNY 人民币, NZD 纽币, AUD 澳币, USD
美元, …) and the API supports the full standard ISO set.

**Picking the right form:**
- Single pair, no amount → default form (`AUD CNY`)
- Single pair, with amount → amount form (`1000 AUD CNY`)
- One base, multiple targets, no amount → `--base USD CNY EUR JPY`
- All-pairs comparison table → `--cross NZD AUD USD CNY`
- **Compound: one amount converted to multiple targets** → the CLI has no
  single form for this. Loop the amount form once per target (e.g., for
  "5000 NZD to JPY/AUD/USD" run three calls). Don't try to combine `--base`
  with an amount — `--base` ignores amounts.

For "USD 今天什么价" / "查一下汇率" with no specific target, use the plain
default (or `--base USD` if they named USD as the base).

## Tool 3 — GitHub daily report

```bash
# Today across all recently-active repos
python3 scripts/github_daily_report.py

# Specific date (YYYY-MM-DD)
python3 scripts/github_daily_report.py --date 2026-04-30

# Specific repo
python3 scripts/github_daily_report.py --repo owner/repo
```

Requires `gh` CLI installed and authenticated. The script auto-discovers
recently active repos from the authenticated user's repo list. If `gh` is not
configured, tell the user to run `gh auth login` first rather than retrying.

The script's output is already grouped, classified (features / fixes /
refactors / docs / others), and Chinese-friendly — present it as-is unless
the user asks for a tighter summary. If the user asks "what did I ship
today?", the per-repo "今日改动总结" section at the bottom is usually the
best answer.

## Tool 4 — NZ public holidays

```bash
# Current year, Auckland-highlighted (default)
python3 scripts/get_holidays.py

# Specific year
python3 scripts/get_holidays.py 2026

# Include all regions, not just Auckland
python3 scripts/get_holidays.py --all
```

Uses the free Nager.Date API (no key needed). Auckland is the default focus
region — the script automatically promotes Auckland-related holidays and
shows the next upcoming one with a day countdown.

For "下个假期是什么" → just the default. For "明年的假期" → pass next year.
For a non-Auckland question (e.g., "Wellington anniversary") → pass `--all`.

## Combined briefing mode

When the user asks for a combined briefing ("morning briefing", "my daily",
"今日简报", "晨报", "give me everything"):

1. Run all four scripts. Run them sequentially — that's simple, network calls
   are short, and it avoids hammering rate-limited APIs.
2. Assemble one Chinese-language briefing in this fixed order:

```
📅 今日简报 — YYYY-MM-DD (Auckland)
============================================

🇳🇿 假期
  • <next holiday + countdown, or "今天是 <holiday>" if it's today>

💱 汇率速览 (1 NZD = ...)
  • CNY: <rate>   AUD: <rate>   USD: <rate>

🐙 GitHub 动态
  <one line per active repo with non-zero activity, top 3 by activity count;
   say "今日无活动" if none>

📰 今日要闻
  ## 世界科技动态
    • <top 3, translated, 1-2 sentence Chinese summary each>
  ## AI 专项
    • <top 3>
  ## 世界政治
    • <top 3>
  ## 经济/金融
    • <top 3>
  ## 新西兰新闻
    • <top 3>
```

Keep each section tight — the briefing should be glanceable, not a wall of
text. If a section has no data, say so in one line rather than silently
dropping it.

If a tool fails (e.g., `gh` not configured, network error on RSS), include
the other sections and note the failure inline. Don't refuse to produce a
partial briefing.

## Gotchas (surfaced from real runs)

- `fetch_news.py` prints `[WARN] Failed to parse <feed>: …` lines to **stderr**
  whenever an upstream RSS source is malformed (RSSHub-backed feeds break
  occasionally). These are diagnostic noise — do **not** surface them in the
  user-facing output. The script handles them and degrades gracefully.
- `get_rate.py --cross` does not auto-uppercase its currency args. The other
  two forms do. Always pass uppercase to `--cross`.
- `github_daily_report.py` discovers active repos from `gh repo list` for the
  authed user. If the user wants activity on a repo they don't own (e.g., an
  org repo they only contribute to), pass `--repo owner/repo` explicitly.
- `feedparser` and `requests` are required (`pip install -r requirements.txt`
  from the skill root). Holidays uses stdlib only.

## When NOT to use this skill

- Pure long-form general news digest with no other topics → `devpilot-news-digest`
- Summarizing an article the user pasted → `devpilot-learn`
- Real-time stock prices, crypto, weather → not covered here, decline politely
- GitHub PR review or issue triage → `devpilot-pr-review` / `devpilot-resolve-issues`
