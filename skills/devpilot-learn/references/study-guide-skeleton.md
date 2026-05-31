# Study Guide Skeleton

Use this when the request is learning-oriented or the source is chaptered, doctrinal,
legal, technical, academic, or exam-prep material. The goal is a Chinese-first study
guide that stays faithful to the source while making it easier to review.

Required characteristics:

- Single-column study-guide layout rather than bilingual comparison columns
- Chinese-first prose with important English source terms preserved inline
- Table of contents near the top
- Follow the source's chapter or section order
- For each major source section, start with an explicit `原文摘要 / Source Summary`
  block that faithfully summarizes that section's original content
- Use study-friendly callouts such as `重点`, `术语`, `法条/案例`, and short review questions
- End with a concise `总结复习` section

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>[中文标题] — 学习资料</title>
  <style>
    body {
      font-family: -apple-system, "PingFang SC", "Microsoft YaHei", sans-serif;
      max-width: 920px;
      margin: 2em auto;
      padding: 0 1.2em;
      line-height: 1.75;
      color: #222;
      background: #fff;
    }
    h1 {
      color: #1a3d6e;
      border-bottom: 3px solid #1a3d6e;
      padding-bottom: .3em;
    }
    h2 {
      color: #1a3d6e;
      margin-top: 2em;
      border-left: 6px solid #1a3d6e;
      padding-left: .6em;
    }
    h3 {
      color: #2a5fa5;
      margin-top: 1.5em;
    }
    .meta,
    .toc,
    .source-summary,
    .key,
    .law,
    .case,
    .term,
    .review {
      border-radius: 6px;
      margin: .8em 0;
      padding: .8em 1em;
    }
    .toc {
      background: #f4f7fb;
      border: 1px solid #d8e1ee;
    }
    .source-summary {
      background: #f7f2ff;
      border-left: 4px solid #7b57c2;
    }
    .source-summary::before {
      content: "🧾 原文摘要：";
      font-weight: 700;
      color: #5b3f98;
    }
    .key {
      background: #fff8d6;
      border-left: 4px solid #e8b923;
    }
    .key::before {
      content: "⭐ 重点：";
      font-weight: 700;
      color: #b88500;
    }
    .law {
      background: #eef6ff;
      border-left: 4px solid #2a78d3;
    }
    .law::before {
      content: "📜 法条/案例：";
      font-weight: 700;
      color: #1a4f99;
    }
    .case {
      background: #f1f9f1;
      border-left: 4px solid #4a9d4a;
    }
    .term {
      background: #fdf3f7;
      border-left: 4px solid #c4528b;
    }
    .review {
      background: #fafafa;
      border: 1px solid #ddd;
    }
    .en {
      color: #555;
      font-style: italic;
      font-size: .92em;
    }
    table {
      border-collapse: collapse;
      width: 100%;
      margin: 1em 0;
      font-size: .95em;
    }
    th, td {
      border: 1px solid #c9d3e1;
      padding: .5em .8em;
      text-align: left;
      vertical-align: top;
    }
    th {
      background: #1a3d6e;
      color: #fff;
    }
    tr:nth-child(even) {
      background: #f6f9fc;
    }
  </style>
</head>
<body>
  <h1>[中文标题]</h1>
  <p class="meta">
    <strong>来源：</strong>[source]<br>
    <strong>作者：</strong>[if available]<br>
    <strong>发布时间：</strong>[if available]<br>
    <strong>说明：</strong>本资料基于原文整理，按学习用途重组，并保留各章节原文摘要。
  </p>

  <div class="toc">
    <strong>目录</strong>
    <ol>
      <li><a href="#sec-1">[章节 1]</a></li>
    </ol>
  </div>

  <h2 id="sec-1">第 1 章 [章节名]</h2>
  <div class="source-summary">
    [2-5 sentence faithful summary of the original section]
  </div>
  <div class="key">
    [High-signal study takeaway]
  </div>
  <div class="term">
    <b>[English term]</b>：[中文解释]
  </div>
  <div class="law">
    [Important statute, case, formula, or cited source detail]
  </div>
  <div class="review">
    <strong>复习题：</strong>
    <ol>
      <li>[Question]</li>
    </ol>
  </div>

  <h2 id="summary">总结复习</h2>
  <p>[Cross-section wrap-up for study and recall]</p>
</body>
</html>
```

Guidance:

- Preserve the source structure instead of flattening everything into one digest
- `原文摘要` is mandatory for each major chapter or section in study-guide mode
- Use review questions only when the source is substantial enough to support them
- Do not fabricate cases, statutes, or exam points that are not grounded in the source
