---
name: devpilot-learn
description: >
  Summarize any single article, document, or web page into a standalone bilingual
  HTML digest with an English | 中文 comparison layout and a terminal terminology
  glossary. MUST use this skill — not raw summarization — whenever the user asks
  to summarize, digest, get key points from, or extract takeaways from a URL, PDF,
  .docx, .md, .txt, paper, report, or blog post. The skill produces a shareable
  HTML file you cannot generate without it. Trigger on: summarize this, key points,
  takeaways, digest, highlights, TL;DR, what does it say, overview, 总结, 摘要,
  要点, 重点, 提取, 整理, 讲了什么. NOT for: writing new content, translating,
  comparing two documents, building apps, multi-source news, code review, or data/CSV
  analysis.
---

# Learn

Generate a standalone bilingual HTML digest from a single source. The output is a
fixed-format comparison document: desktop uses an `English | 中文` two-column layout,
mobile stacks the same content in that order, and the document ends with a four-column
terminology glossary.

This skill does **not** switch to single-language output. Even if the user asks for
English-only or Chinese-only output, still produce the bilingual digest.

## Output contract

The final artifact must be:

- A single self-contained `.html` file with inline CSS only
- A fixed bilingual digest with `English | 中文` content blocks
- Desktop: side-by-side columns
- Mobile: stacked blocks, `English` first and `中文` second
- A document with the sections below, in this exact order:
  1. Bilingual Title
  2. Metadata
  3. `At a Glance / 一眼速览`
  4. `Summary / 摘要`
  5. `Key Points / 要点`
  6. `Evidence & Data / 证据与数据`
  7. `Visuals / 图表与视觉要素`
  8. `Terminology Glossary / 术语对照表`

## Non-negotiable rules

- Always produce bilingual output in the fixed `English | 中文` order
- Use one canonical English content structure first, then translate block-by-block
  into Chinese
- Keep the English and Chinese sides information-equivalent; do not add examples,
  caveats, or claims on only one side
- Keep Chinese close to the English meaning; terminology consistency is more important
  than stylistic freedom
- Prefer common industry Chinese translations when they preserve the original meaning
- For proper nouns such as company names, product names, model names, and organizations,
  keep the original term and add a common Chinese rendering only when useful
- Stay within what the source explicitly states or directly supports; do not add speculative
  interpretation
- If the source quality is poor or incomplete, degrade honestly while preserving the
  full section structure

## Workflow

### 1. Identify the source type

The user will provide one of:

- **URL** — web page or online article
- **PDF** — local `.pdf`
- **Word doc** — local `.docx`
- **Text file** — local `.txt`, `.md`, or similar
- **Pasted text** — content directly in the message

### 2. Fetch the content

**For URLs:**
Use `WebFetch` to retrieve the page. If the page is too short, obviously paywalled,
or mostly navigation/login content, tell the user the source is weak and continue
only if enough content remains to build a constrained digest.

**For PDFs:**
Use the `Read` tool with the file path. For long PDFs, read enough to cover the whole
argument before summarizing.

**For Word docs (.docx):**
Use `Bash` to convert with `textutil -convert txt <file> -stdout` on macOS or use
`pandoc <file> -t plain` as fallback.

**For text files / pasted text:**
Read directly.

### 3. Normalize the source

Before writing any bilingual content:

- Determine the title
- Determine `Content Type / 内容类型` from this fixed set when possible:
  - `Article / 文章`
  - `Report / 报告`
  - `Paper / 论文`
  - `PDF Document / PDF 文档`
  - `Web Page / 网页`
  - `Transcript / 访谈/实录`
  - `Document / 文档` when uncertain
- Extract `Source / 来源`
- Extract `Author / 作者` if available
- Extract `Published / 发布时间` if available
- Identify the source's core thesis, supporting evidence, visuals, and important terms

### 4. Build the canonical English digest first

Write the English side first. Treat it as the canonical source for the Chinese side.
Do not draft English and Chinese independently.

Required structure:

- **Bilingual Title**
  - Produce an English title and a Chinese title as a paired title block
- **Metadata**
  - Include `Source / 来源`
  - Include `Author / 作者` if available
  - Include `Published / 发布时间` if available
  - Include `Content Type / 内容类型`
  - Show a clickable original URL or filename, and include a human-readable site/source
    label when possible
- **At a Glance / 一眼速览**
  - Exactly 2 sentences
  - High-signal overview only
- **Summary / 摘要**
  - Exactly 1 short paragraph
  - Explain the main argument and conclusion without repeating the full key points
- **Key Points / 要点**
  - At least 6 items
  - Usually aim for 6-8 items; use up to 10 only when the source is unusually dense
  - Each item must be a single sentence containing one complete, concrete point
- **Evidence & Data / 证据与数据**
  - Always present
  - At least 3 items
  - Use list cards, not tables
  - Prefer concrete numbers, percentages, dates, findings, and direct factual support
  - Include short source anchors when possible, such as section names, dates, figure
    labels, or other traceable cues
  - If the source has limited quantitative evidence, say so explicitly while still
    giving the strongest available support
- **Visuals / 图表与视觉要素**
  - Always present
  - Include up to 3 essential visuals
  - If an image URL is available and stable, embed it and add bilingual caption/description
  - If the image cannot be embedded, include a bilingual textual description
  - If there are no essential visuals, include one bilingual item stating that and
    briefly explain why
- **Terminology Glossary / 术语对照表**
  - Always the final section
  - Use a four-column table:
    `English Term | 中文对照 | English Explanation | 中文解释`
  - Target 8-15 terms
  - Do not force the count upward if the source truly has fewer valid terms
  - Terms must come from the source or its directly necessary core concepts
  - Sort terms by first appearance in the digest, not alphabetically

### 5. Translate to Chinese block-by-block

Translate each English block into Chinese only after the English block is finalized.

Requirements:

- Preserve the same meaning and scope
- Preserve the same ordering
- Preserve the same number of items in each paired list
- Do not add or remove claims on one side only
- Keep terminology aligned across the entire document, especially in the glossary

### 6. Generate the HTML digest

Create one self-contained HTML file with inline styles only. The visual design should
read as a bilingual comparison document, not a generic article page.

Design requirements:

- Clear `English` and `中文` labels in every bilingual content block
- Single bilingual section headings such as `Key Points / 要点`
- Reader-friendly spacing and typography
- Distinct section cards or blocks that make left/right comparison obvious
- Responsive layout: two columns on desktop, stacked on narrow screens
- Glossary rendered as a compact four-column table at the end
- Footer uses a bilingual product label, such as:
  `Generated by Bilingual Article Digest / 双语文章摘要生成`

### 7. Save and present

Save the HTML file using the `Write` tool.

Naming convention:

- For URLs: `digest-bilingual-[slugified-domain-or-title].html`
- For files: `digest-bilingual-[original-filename].html`

Save to the current working directory unless the user specifies otherwise.

Tell the user the saved file path.

## HTML skeleton

Use this as the structural baseline and adapt content as needed:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>[English Title] / [中文标题] - Bilingual Digest</title>
  <style>
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      line-height: 1.7;
      color: #1f2328;
      background: #f4f1ea;
      padding: 24px;
    }
    .page {
      max-width: 1100px;
      margin: 0 auto;
      background: #fffdf8;
      border: 1px solid #e6dece;
      border-radius: 16px;
      padding: 32px;
      box-shadow: 0 10px 30px rgba(66, 41, 0, 0.08);
    }
    .title-pair {
      margin-bottom: 20px;
      padding-bottom: 20px;
      border-bottom: 1px solid #eadfcb;
    }
    .title-pair h1,
    .title-pair h2 {
      margin: 0;
      line-height: 1.25;
    }
    .title-pair h1 { font-size: 2rem; }
    .title-pair h2 {
      margin-top: 8px;
      font-size: 1.25rem;
      color: #6b5c45;
      font-weight: 600;
    }
    .meta-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 12px 20px;
      margin-bottom: 28px;
    }
    .meta-item {
      padding: 12px 14px;
      background: #faf6ee;
      border-radius: 10px;
      border: 1px solid #ede3d1;
    }
    .section {
      margin-top: 28px;
    }
    .section h3 {
      margin: 0 0 12px;
      font-size: 1.1rem;
    }
    .pair-block {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 16px;
    }
    .lang-card {
      padding: 16px;
      background: #fcfaf5;
      border: 1px solid #eadfcb;
      border-radius: 12px;
    }
    .lang-label {
      display: inline-block;
      margin-bottom: 10px;
      font-size: 0.75rem;
      font-weight: 700;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: #8a6f3b;
    }
    .list {
      margin: 0;
      padding-left: 20px;
    }
    .list li + li {
      margin-top: 10px;
    }
    .visual {
      margin-top: 12px;
      padding: 12px;
      background: #fff;
      border-radius: 10px;
      border: 1px solid #eadfcb;
    }
    .visual img {
      max-width: 100%;
      height: auto;
      border-radius: 8px;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      margin-top: 12px;
    }
    th, td {
      border: 1px solid #eadfcb;
      padding: 10px 12px;
      text-align: left;
      vertical-align: top;
    }
    th {
      background: #faf6ee;
    }
    .footer {
      margin-top: 32px;
      padding-top: 20px;
      border-top: 1px solid #eadfcb;
      color: #6b5c45;
      font-size: 0.9rem;
    }
    @media (max-width: 768px) {
      body { padding: 12px; }
      .page { padding: 20px; }
      .pair-block,
      .meta-grid {
        grid-template-columns: 1fr;
      }
    }
  </style>
</head>
<body>
  <main class="page">
    <section class="title-pair">
      <h1>[English Title]</h1>
      <h2>[中文标题]</h2>
    </section>

    <section class="meta-grid">
      <div class="meta-item"><strong>Source / 来源</strong>: [Clickable source]</div>
      <div class="meta-item"><strong>Author / 作者</strong>: [If available]</div>
      <div class="meta-item"><strong>Published / 发布时间</strong>: [If available]</div>
      <div class="meta-item"><strong>Content Type / 内容类型</strong>: [Fixed enum label]</div>
    </section>

    <section class="section">
      <h3>At a Glance / 一眼速览</h3>
      <div class="pair-block">
        <div class="lang-card">
          <div class="lang-label">English</div>
          <p>[Exactly 2 sentences]</p>
        </div>
        <div class="lang-card">
          <div class="lang-label">中文</div>
          <p>[Exactly 2 corresponding sentences]</p>
        </div>
      </div>
    </section>

    <section class="section">
      <h3>Summary / 摘要</h3>
      <div class="pair-block">
        <div class="lang-card">
          <div class="lang-label">English</div>
          <p>[One short paragraph]</p>
        </div>
        <div class="lang-card">
          <div class="lang-label">中文</div>
          <p>[Corresponding short paragraph]</p>
        </div>
      </div>
    </section>

    <section class="section">
      <h3>Key Points / 要点</h3>
      <div class="pair-block">
        <div class="lang-card">
          <div class="lang-label">English</div>
          <ol class="list">
            <li>[Single-sentence point]</li>
          </ol>
        </div>
        <div class="lang-card">
          <div class="lang-label">中文</div>
          <ol class="list">
            <li>[对应单句要点]</li>
          </ol>
        </div>
      </div>
    </section>

    <section class="section">
      <h3>Evidence & Data / 证据与数据</h3>
      <div class="pair-block">
        <div class="lang-card">
          <div class="lang-label">English</div>
          <ul class="list">
            <li>[Evidence card with anchor]</li>
          </ul>
        </div>
        <div class="lang-card">
          <div class="lang-label">中文</div>
          <ul class="list">
            <li>[对应证据条目]</li>
          </ul>
        </div>
      </div>
    </section>

    <section class="section">
      <h3>Visuals / 图表与视觉要素</h3>
      <div class="pair-block">
        <div class="lang-card">
          <div class="lang-label">English</div>
          <div class="visual">[Image or bilingual-ready description]</div>
        </div>
        <div class="lang-card">
          <div class="lang-label">中文</div>
          <div class="visual">[对应图表说明]</div>
        </div>
      </div>
    </section>

    <section class="section">
      <h3>Terminology Glossary / 术语对照表</h3>
      <table>
        <thead>
          <tr>
            <th>English Term</th>
            <th>中文对照</th>
            <th>English Explanation</th>
            <th>中文解释</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>[Term]</td>
            <td>[中文对照]</td>
            <td>[Short English explanation]</td>
            <td>[对应中文解释]</td>
          </tr>
        </tbody>
      </table>
    </section>

    <footer class="footer">
      Generated by Bilingual Article Digest / 双语文章摘要生成
    </footer>
  </main>
</body>
</html>
```

## Edge cases

- **Paywalled or weak content**: If the fetched content is too thin or mostly blocked,
  say so and produce a constrained digest only from what is actually accessible
- **Very long sources**: Still cover the full argument; do not collapse into a short
  abstract if the source is materially complex
- **Multiple URLs/files in one request**: Produce one bilingual HTML file per source
- **Non-article inputs**: Still use the same fixed format, but label the content type
  honestly and note when the source is not argument-driven
- **Sparse terminology**: Do not invent terms just to hit the target range; include
  only valid source-grounded terms and note that terminology density is limited
