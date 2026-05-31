# Close-Reading Study Skeleton

Use this as the structural baseline for the artifact. It is a single-column,
reading-oriented layout:

- The **original text** is the primary prose.
- Each **Chinese translation** sits directly below its passage, marked with a left
  border and a muted tone so it reads as a translation — never as a second column.
- **Quizzes** sit in callout blocks; each answer lives inside a `<details>` element
  so it stays hidden until the reader opens it (no JavaScript needed).

Adapt the content freely, but keep the verbatim-original → translation → quiz
rhythm and the single-column flow.

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>[标题] — 精读学习资料</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI",
        "PingFang SC", "Microsoft YaHei", sans-serif;
      max-width: 860px;
      margin: 2em auto;
      padding: 0 1.2em 4em;
      line-height: 1.8;
      color: #1f2328;
      background: #fbfaf7;
    }
    h1 {
      color: #1a3d6e;
      border-bottom: 3px solid #1a3d6e;
      padding-bottom: .3em;
      line-height: 1.3;
    }
    h2 {
      color: #1a3d6e;
      margin-top: 2.2em;
      border-left: 6px solid #1a3d6e;
      padding-left: .6em;
    }
    .meta {
      background: #f2f1ec;
      border: 1px solid #e3e0d6;
      border-radius: 8px;
      padding: .9em 1.1em;
      margin: 1em 0 1.6em;
      font-size: .95em;
    }
    .meta a { color: #1a6e8a; }
    .toc {
      background: #f4f7fb;
      border: 1px solid #d8e1ee;
      border-radius: 8px;
      padding: .9em 1.2em;
      margin: 1.4em 0;
    }
    .toc strong { color: #1a3d6e; }

    /* original passage + its translation */
    .passage { margin: 1.4em 0; }
    .passage .orig {
      margin: 0 0 .55em;
    }
    .passage .zh {
      margin: 0;
      padding: .5em .2em .5em .95em;
      border-left: 3px solid #c9b27a;
      background: #faf6ea;
      color: #4d4127;
    }
    .passage .zh::before {
      content: "译 ";
      font-weight: 700;
      color: #9a7f3c;
    }

    /* inline term / hard-phrase note */
    .note {
      margin: .8em 0;
      padding: .5em .9em;
      background: #fdf3f7;
      border-left: 3px solid #c4528b;
      border-radius: 4px;
      font-size: .94em;
    }
    .note b { color: #9c2e6a; }

    /* quizzes */
    .quiz {
      background: #f4f7fb;
      border: 1px solid #d8e1ee;
      border-radius: 8px;
      padding: 1em 1.3em;
      margin: 1.8em 0;
    }
    .quiz h3 { margin: 0 0 .6em; color: #1a3d6e; }
    .quiz ol { margin: 0; padding-left: 1.4em; }
    .quiz li { margin: 1em 0; }
    .final-test { background: #fff7e9; border-color: #e8d6ab; }
    .final-test h2 { border-left-color: #b8860b; color: #8a5a00; }

    details.answer { margin-top: .5em; }
    details.answer > summary {
      cursor: pointer;
      color: #1a6e3d;
      font-weight: 600;
      list-style: none;
    }
    details.answer > summary::before { content: "▸ "; }
    details.answer[open] > summary::before { content: "▾ "; }
    .answer-body {
      margin-top: .55em;
      padding: .55em .9em;
      background: #eef7ef;
      border-left: 3px solid #4a9d4a;
      border-radius: 4px;
    }
    .answer-body .label { font-weight: 700; color: #2e7d46; }
    .footer {
      margin-top: 3em;
      padding-top: 1.2em;
      border-top: 1px solid #e3e0d6;
      color: #6b5c45;
      font-size: .9em;
    }
  </style>
</head>
<body>
  <h1>[标题（原文标题；可加中文）]</h1>

  <div class="meta">
    <strong>来源 / Source：</strong><a href="[url-or-filename]">[clickable source]</a><br>
    <strong>作者 / Author：</strong>[if available]<br>
    <strong>发布时间 / Published：</strong>[if available]<br>
    <strong>类型 / Content Type：</strong>[fixed enum label]<br>
    <strong>说明：</strong>本资料保留原文，逐段附中文翻译，并配节后小测与文末总测，供精读复习。
  </div>

  <!-- include only when the source has several sections -->
  <div class="toc">
    <strong>目录</strong>
    <ol>
      <li><a href="#sec-1">[Section 1 title]</a></li>
    </ol>
  </div>

  <h2 id="sec-1">[Section 1 title / 章节标题]</h2>

  <div class="passage">
    <p class="orig">[Verbatim original paragraph from the source]</p>
    <p class="zh">[忠实的中文翻译]</p>
  </div>

  <div class="passage">
    <p class="orig">[Next verbatim original paragraph]</p>
    <p class="zh">[对应中文翻译]</p>
  </div>

  <!-- optional, only where it aids reading -->
  <div class="note"><b>[term / 术语]</b>：[简短中文注解]</div>

  <div class="quiz">
    <h3>节后小测</h3>
    <ol>
      <li>
        <p>[Question grounded only in this section]</p>
        <details class="answer">
          <summary>查看答案</summary>
          <div class="answer-body">
            <p><span class="label">答案：</span>[answer]</p>
            <p><span class="label">解析：</span>[short explanation pointing back to the passage]</p>
          </div>
        </details>
      </li>
    </ol>
  </div>

  <!-- ...repeat <h2> + passages + quiz for each section... -->

  <div class="quiz final-test">
    <h2 id="final-test">总测 / Final Test</h2>
    <ol>
      <li>
        <p>[Comprehensive question spanning the whole source]</p>
        <details class="answer">
          <summary>查看答案</summary>
          <div class="answer-body">
            <p><span class="label">答案：</span>[answer]</p>
            <p><span class="label">解析：</span>[short explanation grounded in the source]</p>
          </div>
        </details>
      </li>
    </ol>
  </div>

  <div class="footer">精读学习资料 / Close-Reading Study Guide</div>
</body>
</html>
```

Guidance:

- The `.orig` text must be the source's actual wording. The `.zh` block is where
  interpretation happens — keep it faithful.
- Keep large portions of the original; don't reduce a section to one quoted line.
- Use `.note` sparingly — only for terms or phrases that genuinely need a gloss.
- Every quiz answer goes inside `<details>` so it stays collapsed by default.
- Drop the `.toc` block for short, single-section sources.
