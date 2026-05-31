# HTML Skeleton

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
