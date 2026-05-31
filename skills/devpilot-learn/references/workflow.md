# Workflow

## 1. Identify the source type

The user will provide one of:

- **URL** — web page or online article
- **PDF** — local `.pdf`
- **Word doc** — local `.docx`
- **Text file** — local `.txt`, `.md`, or similar
- **Pasted text** — content directly in the message

## 2. Fetch the content

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

## 3. Choose the output mode

Select the mode before outlining the artifact.

Choose **study-guide mode** when one or more of these are true:

- The user asks for 学习资料, notes, revision material, exam prep, or chapter-by-chapter learning help
- The source is long and structurally organized, such as a textbook chapter, legal reading,
  policy manual, course packet, academic paper, or technical explainer
- The user wants help studying the original source rather than only skimming it

Choose **bilingual digest mode** when:

- The user wants a concise digest, high-level summary, or quick bilingual comparison
- The source is article-shaped and not naturally chaptered

If in doubt, prefer **study-guide mode** for educational sources and **digest mode**
for general articles.

## 4. Normalize the source

Before writing the artifact:

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

For study-guide mode, also identify:

- The source's chapter or section boundaries
- Important statutes, cases, formulas, frameworks, or examples
- Terms that should remain in English inline with Chinese explanation
- Which sections are rich enough to support short review questions

## 5. Build the artifact

### If using bilingual digest mode

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

Then translate each English block into Chinese only after the English block is finalized.

Requirements:

- Preserve the same meaning and scope
- Preserve the same ordering
- Preserve the same number of items in each paired list
- Do not add or remove claims on one side only
- Keep terminology aligned across the entire document, especially in the glossary

### If using study-guide mode

Build a Chinese-first learning handout that follows the source structure.

Required structure:

- **Title + metadata**
  - Show source, author, date if available
  - Add a short note that the material is organized for study from the original source
- **Table of contents**
  - Link to each major chapter or section
- **Per major section**
  - Start with `原文摘要 / Source Summary`
    - 2-5 sentences
    - Faithful summary of that source section's original content
    - Do not replace this with only your own explanation
  - Add `重点` callout for the highest-signal takeaway
  - Add `术语` blocks for terms that matter in this section
  - Add `法条/案例` or equivalent source-grounded authority/examples when applicable
  - Add short study notes or comparisons when they help retention and remain source-grounded
  - Add short review questions only if the section is substantial enough
- **总结复习**
  - End with a concise cross-section recap for revision

Study-guide mode rules:

- Preserve the source's chapter order instead of flattening everything into one digest
- Use Chinese as the main explanatory language unless the user asks otherwise
- Keep important English source terms inline where they aid recall
- Review questions must be grounded in the source, not invented from external knowledge

## 6. Generate the HTML artifact

Create one self-contained HTML file with inline styles only. The visual design should
match the chosen mode instead of falling back to a generic article page.

Design requirements for bilingual digest mode:

- Clear `English` and `中文` labels in every bilingual content block
- Single bilingual section headings such as `Key Points / 要点`
- Reader-friendly spacing and typography
- Distinct section cards or blocks that make left/right comparison obvious
- Responsive layout: two columns on desktop, stacked on narrow screens
- Glossary rendered as a compact four-column table at the end
- Footer uses a bilingual product label, such as:
  `Generated by Bilingual Article Digest / 双语文章摘要生成`

Design requirements for study-guide mode:

- Single-column, Chinese-first learning layout
- Strong section hierarchy and visible table of contents
- Distinct visual treatments for `原文摘要`, `重点`, `术语`, and `法条/案例`
- Better suited for reading, revising, and chapter-by-chapter recall than for bilingual comparison
- Do not remove the original-source fidelity cues

## 7. Save and present

Save the HTML file using the `Write` tool.

Naming convention:

- For URLs: `digest-bilingual-[slugified-domain-or-title].html`
- For files: `digest-bilingual-[original-filename].html`

For study-guide mode, prefer:

- For URLs: `study-guide-[slugified-domain-or-title].html`
- For files: `study-guide-[original-filename].html`

Save to the current working directory unless the user specifies otherwise.

Tell the user the saved file path.

## Edge cases

- **Paywalled or weak content**: If the fetched content is too thin or mostly blocked,
  say so and produce a constrained digest only from what is actually accessible
- **Very long sources**: Still cover the full argument; do not collapse into a short
  abstract if the source is materially complex
- **Multiple URLs/files in one request**: Produce one HTML learning artifact per source
- **Non-article inputs**: Still use the chosen mode honestly, but label the content type
  honestly and note when the source is not argument-driven
- **Sparse terminology**: Do not invent terms just to hit the target range; include
  only valid source-grounded terms and note that terminology density is limited
- **Study-guide mode with weak structure**: If the source is too short to support chapters,
  collapse to a lighter study-note layout but still include at least one `原文摘要` block
