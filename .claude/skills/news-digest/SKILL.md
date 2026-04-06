---
name: news-digest
description: >
  Fetch and summarize current news into a structured briefing across categories like
  International, AI, Technology, and Economics. Supports English and Chinese.
  ALWAYS use when the user wants current news: summaries, daily/weekly briefings,
  headlines, top stories, "catch me up", "what's happening", or domain-specific news
  (AI news, tech news, market/economics news, international affairs).
  Triggers on: "news summary", "news digest", "daily briefing", "weekly briefing",
  "what happened this week", "news roundup", "top stories", "headlines", "catch me up",
  "what's going on with the economy", "AI news", "今日新闻", "本周新闻", "新闻简报",
  "每日简报", "最近发生了什么". NOT for: building news apps, translating articles,
  historical data analysis, or summarizing user-provided documents.
---

# News Digest

Generate a professional, neutral news briefing using RSS feeds as the primary data source.
RSS feeds return structured headlines and summaries, which is far more token-efficient than
web search. Only fall back to WebSearch when RSS coverage is insufficient.

## Workflow

### 1. Determine scope

Parse the user's request to figure out:

- **Timeframe**: "today" (last 24h) or "this week" (last 7 days). Default to today if unclear.
- **Language**: If the user writes in Chinese or asks for Chinese news, produce the digest in Chinese.
  Otherwise default to English. The user can also explicitly request a language.
- **Categories**: Use the user's requested categories if specified. Otherwise use the defaults below.

**Default categories:**
- International (geopolitics, diplomacy, conflicts, major world events)
- AI & Machine Learning (models, research, industry moves, regulation)
- Technology (products, startups, platforms, cybersecurity)
- Economics & Finance (markets, central banks, trade, macro indicators)

The user can add, remove, or replace categories — be flexible.

### 2. Fetch RSS feeds

Use `WebFetch` to fetch RSS feeds in parallel. Pick 2-3 feeds per category from the list below
based on the user's language preference. RSS feeds return XML with `<item>` elements containing
`<title>`, `<description>`, `<link>`, and `<pubDate>` — extract these to build your story list.

**RSS Feed Directory:**

#### International (English)
- BBC World: `http://feeds.bbci.co.uk/news/world/rss.xml`
- Al Jazeera: `https://www.aljazeera.com/xml/rss/all.xml`
- NPR World: `https://feeds.npr.org/1004/rss.xml`

#### AI & Technology (English)
- TechCrunch AI: `https://techcrunch.com/category/artificial-intelligence/feed/`
- Ars Technica AI: `https://arstechnica.com/ai/feed/`
- The Verge: `https://www.theverge.com/rss/index.xml`
- Wired AI: `https://www.wired.com/feed/tag/ai/latest/rss`
- MIT Tech Review: `https://www.technologyreview.com/feed/`

#### Economics & Finance (English)
- CNBC Top News: `https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=100003114`
- CNBC Economy: `https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=20910258`
- MarketWatch: `https://www.marketwatch.com/rss/topstories`

#### Chinese Sources (中文)
- 36Kr 快讯: `https://36kr.com/feed`
- 澎湃新闻 (via RSSHub): `https://rsshub.app/thepaper/channel/25950`
- 36Kr 快讯 (via RSSHub): `https://rsshub.app/36kr/newsflashes`
- China Daily: `https://www.chinadaily.com.cn/rss/china_rss.xml`
- China Daily Business: `https://www.chinadaily.com.cn/rss/business_rss.xml`

**Feed selection strategy:**
- For English digests: use the English feeds for each requested category
- For Chinese digests: use Chinese feeds + English feeds (translate English headlines to Chinese)
- Fetch 2-3 feeds per category in parallel — this gives enough coverage without waste

### 3. Filter and supplement

After parsing RSS items:

- **Filter by date**: Only keep items whose `<pubDate>` falls within the requested timeframe.
- **Deduplicate**: If the same story appears in multiple feeds, keep only one.
- **Check coverage**: If any category has fewer than 3 stories after filtering, use `WebSearch`
  as a fallback for that category only. This keeps token usage low — most of the time RSS
  provides enough stories.

### 4. Compile the digest

Write the digest in plain text using this structure:

```
NEWS DIGEST — [Date or Date Range]
============================================

[CATEGORY NAME]
--------------------------------------------
- [Headline]: [1-2 sentence summary]. ([Source])
- [Headline]: [1-2 sentence summary]. ([Source])
...

[NEXT CATEGORY]
--------------------------------------------
- ...
```

For Chinese output:

```
新闻简报 — [日期或日期范围]
============================================

[分类名称]
--------------------------------------------
- [标题]：[1-2句摘要]。（[来源]）
...
```

**Guidelines:**
- Aim for 5-8 stories per category. More is better than fewer, but don't pad with trivial items.
- Each story gets 1-2 sentences max. Lead with the most newsworthy fact.
- Write in neutral, factual tone. No editorializing.
- Always attribute the source in parentheses.
- Order stories within each category by significance, most important first.
- If a story spans multiple categories, place it in the most relevant one only.
- Include the date or date range in the header.

### 5. Present to the user

Output the digest directly. No preamble — just start with the digest itself.

If any category returned very few results (fewer than 3 stories), note this briefly at the end:
"Note: Limited results found for [category] in the specified timeframe."
