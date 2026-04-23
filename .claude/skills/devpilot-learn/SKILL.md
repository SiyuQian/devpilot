---
name: devpilot-learn
description: >
  Summarize any single article, document, or web page into a standalone HTML digest
  file with reader-mode styling. MUST use this skill — not raw summarization — whenever
  the user asks to summarize, digest, get key points from, or extract takeaways from
  a URL, PDF, .docx, .md, .txt, paper, report, or blog post. The skill produces a
  shareable HTML file you cannot generate without it. Trigger on: summarize this,
  key points, takeaways, digest, highlights, TL;DR, what does it say, overview,
  总结, 摘要, 要点, 重点, 提取, 整理, 讲了什么. NOT for: writing new content,
  translating, comparing two documents, building apps, multi-source news, code review,
  or data/CSV analysis.
---

# Learn

Generate a clean, reader-mode HTML digest from a single content source. The digest
captures the article's core argument, key points, and important visuals in a format
that's easy to scan and share.

## Workflow

### 1. Identify the input source

The user will provide one of:

- **URL** — a web page or online article
- **PDF** — a local file path to a .pdf
- **Word doc** — a local file path to a .docx
- **Text file** — a local file path to .txt, .md, or similar
- **Pasted text** — content directly in the message

Detect which type it is and proceed accordingly.

### 2. Fetch the content

**For URLs:**
Use `WebFetch` to retrieve the page. If the page content is too short or looks like
a paywall/login wall, tell the user and ask if they have another way to access it.

**For PDFs:**
Use the `Read` tool with the file path. For large PDFs (>10 pages), start with the
first 20 pages, summarize, then continue if needed.

**For Word docs (.docx):**
Use `Bash` to convert with `textutil -convert txt <file> -stdout` (macOS) or
`pandoc <file> -t plain` as a fallback. If neither is available, try reading the
file directly and extracting what you can.

**For text files / pasted text:**
Read directly.

### 3. Determine output language

- If the user explicitly specifies a language (e.g., "summarize in English", "用中文总结"),
  use that language.
- If no language is specified, default to the language of the original content.
- If the content is mixed-language, default to the dominant language.

### 4. Analyze and extract

Read through the full content and identify:

- **Core thesis / main argument** — what is this article fundamentally about?
- **Key points** (5-10) — the most important facts, findings, or arguments. Each
  should be a self-contained statement, not a vague reference.
- **Important data** — statistics, numbers, percentages, dates that support key points.
- **Images and charts** — identify images/charts that are essential to understanding
  the content (skip decorative images, ads, author photos, logos).

For charts and diagrams:
- If you have the original image URL, preserve it for embedding.
- If the image is not accessible via URL (e.g., from a PDF or local file), write a
  clear textual description of what the chart shows, including axis labels, trends,
  and key data points.

### 5. Generate the HTML digest

Create a single, self-contained HTML file. No external dependencies — all styles are
inline. The design should feel like a browser's reader mode: clean, focused, easy to read.

**HTML structure:**

```html
<!DOCTYPE html>
<html lang="[language-code]">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>[Article Title] - Digest</title>
  <style>
    /* Reader-mode styling */
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
                   "Helvetica Neue", Arial, sans-serif;
      line-height: 1.8;
      color: #1d1d1f;
      background: #fafafa;
      padding: 2rem 1rem;
    }
    .container {
      max-width: 720px;
      margin: 0 auto;
      background: #fff;
      padding: 2.5rem 2rem;
      border-radius: 8px;
      box-shadow: 0 1px 3px rgba(0,0,0,0.08);
    }
    h1 {
      font-size: 1.75rem;
      font-weight: 700;
      margin-bottom: 0.5rem;
      line-height: 1.3;
    }
    .meta {
      color: #6e6e73;
      font-size: 0.9rem;
      margin-bottom: 1.5rem;
      padding-bottom: 1rem;
      border-bottom: 1px solid #e5e5e7;
    }
    .meta a { color: #6e6e73; text-decoration: underline; }
    .summary {
      font-size: 1.05rem;
      color: #424245;
      margin-bottom: 2rem;
      padding: 1rem 1.25rem;
      background: #f5f5f7;
      border-radius: 6px;
      border-left: 3px solid #0071e3;
    }
    h2 {
      font-size: 1.2rem;
      font-weight: 600;
      margin: 2rem 0 1rem;
      color: #1d1d1f;
    }
    .key-points { list-style: none; padding: 0; }
    .key-points li {
      padding: 0.75rem 0;
      border-bottom: 1px solid #f0f0f2;
      padding-left: 1.25rem;
      position: relative;
    }
    .key-points li::before {
      content: "";
      position: absolute;
      left: 0;
      top: 1.05rem;
      width: 6px;
      height: 6px;
      background: #0071e3;
      border-radius: 50%;
    }
    .key-points li:last-child { border-bottom: none; }
    .chart-block {
      margin: 1.5rem 0;
      padding: 1rem;
      background: #f5f5f7;
      border-radius: 6px;
      text-align: center;
    }
    .chart-block img {
      max-width: 100%;
      height: auto;
      border-radius: 4px;
    }
    .chart-block .caption {
      font-size: 0.85rem;
      color: #6e6e73;
      margin-top: 0.5rem;
    }
    .chart-description {
      font-size: 0.95rem;
      color: #424245;
      text-align: left;
      padding: 0.75rem 1rem;
      background: #f0f0f2;
      border-radius: 4px;
      margin: 1rem 0;
      border-left: 3px solid #86868b;
    }
    .footer {
      margin-top: 2rem;
      padding-top: 1rem;
      border-top: 1px solid #e5e5e7;
      font-size: 0.8rem;
      color: #86868b;
    }
  </style>
</head>
<body>
  <div class="container">
    <h1>[Article Title]</h1>
    <div class="meta">
      Source: <a href="[url]">[domain or filename]</a> | [date if available]
    </div>

    <div class="summary">
      [2-3 sentence executive summary — the article's core message]
    </div>

    <h2>[Key Points section header, in output language]</h2>
    <ul class="key-points">
      <li>[Key point 1 — a concrete, self-contained statement]</li>
      <li>[Key point 2]</li>
      <!-- 5-10 key points -->
    </ul>

    <!-- For each important chart/image: -->
    <h2>[Charts & Data section header, in output language]</h2>

    <!-- If image URL is available: -->
    <div class="chart-block">
      <img src="[image-url]" alt="[description]">
      <div class="caption">[What this chart shows and why it matters]</div>
    </div>

    <!-- If image is not accessible, use text description: -->
    <div class="chart-description">
      [Detailed description of the chart: type, axes, trends, key numbers]
    </div>

    <div class="footer">
      Generated by Article Digest
    </div>
  </div>
</body>
</html>
```

**Adapt the section headers to the output language.** For example:
- English: "Key Points", "Charts & Data"
- Chinese: "要点", "图表与数据"

**If there are no charts or images worth including, omit the Charts & Data section entirely.**

### 6. Save and present

Save the HTML file using the `Write` tool. Naming convention:
- For URLs: `digest-[slugified-domain-or-title].html`
- For files: `digest-[original-filename].html`

Save to the current working directory unless the user specifies otherwise.

After saving, tell the user the file path and offer to open it:
```bash
open [file-path]  # macOS
```

## Edge cases

- **Paywalled content**: If WebFetch returns very little content or a login page,
  inform the user. Suggest they paste the text directly or provide a PDF.
- **Very long articles (>5000 words)**: Still summarize the whole thing, but aim for
  the higher end of key points (8-10) to ensure adequate coverage.
- **Multiple URLs/files in one request**: Process each one separately, generating
  one HTML per source. Name them distinctly.
- **Non-article content** (e.g., a product page, a code repo): Do your best to
  extract the most informative content, but note to the user that the source isn't
  a typical article.
