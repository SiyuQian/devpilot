---
name: content-creator
description: SEO-optimized blog and content writing skill. Use this skill whenever the user wants to write a blog post, create content for their website, improve SEO rankings, do keyword research, or plan content strategy. Triggers on any mention of blog writing, SEO content, keyword research, content marketing, "写博客", "写文章", "内容创作", "SEO优化", or when the user wants to create any form of long-form content for a website or product. Even if the user just says "write something about X for our site", use this skill.
---

# Content Creator — SEO-Optimized Blog & Article Writing

Write high-quality, SEO-optimized blog posts that rank well in search engines while remaining genuinely useful to readers. The core philosophy: **content that serves readers well is content that ranks well** — search engines reward articles that people actually want to read and share.

## Process Overview

### Phase 1: Understand the Project

Before writing anything, understand what the project is about and who the audience is.

1. Read `CLAUDE.md` or `README.md` in the current project to understand the product/service
2. If these files don't exist or don't provide enough context, ask the user:
   - What does your product/service do?
   - Who is the target audience?

Keep questions concise — one or two at most. Extract what you need and move on.

### Phase 2: Topic Exploration

If the user gives a clear topic, move to Phase 3. If the topic is vague or they want suggestions:

1. Based on the project context, brainstorm 3-5 topic angles that would:
   - Address real problems the target audience searches for
   - Naturally connect back to the product/service
   - Have content gaps (topics not well-covered by existing top results)
2. Present the options briefly and let the user pick or refine

### Phase 3: Keyword Research

Use web search to research keywords before writing. This is the foundation of good SEO content.

1. **Primary keyword discovery** — Search for the topic and identify:
   - The main keyword to target (search volume + relevance balance)
   - 3-5 secondary keywords (related terms, long-tail variations)
   - Common questions people ask about this topic ("People Also Ask" style)

2. **Competitor analysis** — Look at the top 3-5 ranking articles for the primary keyword:
   - What subtopics do they cover?
   - What's missing from their coverage?
   - What angle can we take that's different or better?

3. **Search intent** — Determine what the searcher actually wants:
   - Informational (how-to, explanation)
   - Commercial (comparison, review)
   - Transactional (buying guide)
   - Match the article structure to this intent

Present findings to the user as a brief summary:
- Target keyword + secondary keywords
- Recommended article angle (and why it can compete)
- Suggested structure outline

Get user confirmation before writing.

### Phase 4: Ask About Output Format

Every project is different. Before writing, ask the user:

- What format do you need? (Markdown, HTML, MDX, etc.)
- Where will this be published? (blog directory path, CMS, etc.)
- Any front matter or metadata format required? (e.g., YAML front matter for Hugo/Jekyll/Next.js)

If the user has published blog posts before in the project, check for existing posts to match the format automatically — look for common blog directories like `blog/`, `posts/`, `content/`, `src/content/`, `_posts/`.

### Phase 5: Write the Article

Follow these SEO writing principles:

#### Structure
- **Title (H1)**: Include primary keyword naturally, keep under 60 characters, make it compelling
- **Meta description**: 150-160 characters, include primary keyword, write as a clear value proposition
- **Introduction**: Hook the reader in the first 2-3 sentences, state what they'll learn, include primary keyword in the first 100 words
- **Headings (H2/H3)**: Use a logical hierarchy, incorporate secondary keywords where natural, make headings scannable
- **Conclusion**: Summarize key takeaways, include a clear call-to-action relevant to the product/service

#### Keyword Integration
- Primary keyword appears in: title, first paragraph, 1-2 H2 headings, conclusion
- Secondary keywords spread naturally throughout the article
- Keyword density: aim for 1-2% for primary keyword — if it feels forced, reduce it
- Use semantic variations and related terms rather than repeating the exact same phrase
- **Never sacrifice readability for keyword placement** — if a keyword doesn't fit naturally in a sentence, rephrase or skip it

#### Content Quality
- Write in a clear, natural, friendly tone — as if explaining to a knowledgeable friend
- Use short paragraphs (3-4 sentences max)
- Include practical examples, data points, or actionable steps
- Break up text with lists, tables, or code blocks where appropriate
- Article length: follow SEO best practices for the topic (typically 1,500-2,500 words for competitive keywords, but prioritize completeness over word count)

#### Technical SEO Elements
- Suggest internal links to other relevant pages/posts if they exist in the project
- Suggest 2-3 places where external links to authoritative sources would strengthen the content
- Include image alt text suggestions where images would add value
- Use semantic HTML heading hierarchy (never skip from H2 to H4)

### Phase 6: SEO Checklist

After writing, verify the article against this checklist and report results to the user:

- [ ] Primary keyword in title, first paragraph, and conclusion
- [ ] Meta description under 160 characters with keyword
- [ ] Headings use proper H2/H3 hierarchy
- [ ] At least 2 secondary keywords used naturally
- [ ] Short paragraphs, scannable structure
- [ ] Internal link opportunities identified
- [ ] Call-to-action present
- [ ] Title under 60 characters
- [ ] Content matches the identified search intent

## Language

Write in the same language as the user's input. If the user writes in Chinese, write the article in Chinese. If in English, write in English. This applies to the article itself — internal communication and keyword research can be in whatever language is most effective for finding results.

## What This Skill Does NOT Do

- Does not post or publish content automatically
- Does not guarantee rankings (SEO is a long game)
- Does not do link building or off-page SEO
- Does not replace a full content strategy — but it's a strong starting point for each individual piece
