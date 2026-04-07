package manager

const manageHTML = `{{define "manage"}}<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}}</title>
  <style>
    :root {
      --bg: #f2f0ea;
      --panel: rgba(255,255,255,0.88);
      --line: #d5d0c4;
      --text: #1e1c18;
      --muted: #6f6a61;
      --accent: #0d5c63;
      --warm: #c05a2b;
      --ok: #1f7a52;
      --shadow: 0 22px 60px rgba(31, 27, 19, 0.12);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Segoe UI", "PingFang SC", sans-serif;
      background:
        radial-gradient(circle at top left, rgba(192,90,43,0.15), transparent 26%),
        radial-gradient(circle at bottom right, rgba(13,92,99,0.16), transparent 28%),
        linear-gradient(135deg, #e9e3d7 0%, #f7f4ee 45%, #ece8de 100%);
      color: var(--text);
      min-height: 100vh;
      padding: 20px;
    }
    .shell {
      max-width: 1120px;
      margin: 0 auto;
      background: var(--panel);
      border: 1px solid rgba(255,255,255,0.7);
      border-radius: 28px;
      box-shadow: var(--shadow);
      overflow: hidden;
      backdrop-filter: blur(12px);
    }
    .hero {
      padding: 28px;
      border-bottom: 1px solid var(--line);
      background: linear-gradient(135deg, rgba(13,92,99,0.08), rgba(192,90,43,0.08));
    }
    .eyebrow {
      display: inline-block;
      font-size: 12px;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      color: var(--accent);
      background: rgba(13,92,99,0.12);
      padding: 6px 10px;
      border-radius: 999px;
    }
    h1 { margin: 14px 0 8px; font-size: clamp(30px, 5vw, 46px); }
    p { margin: 0; color: var(--muted); line-height: 1.6; }
    .cards {
      display: grid;
      gap: 18px;
      padding: 22px;
    }
    .card {
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: 18px;
      background: rgba(255,255,255,0.75);
    }
    .card-top {
      display: flex;
      align-items: start;
      justify-content: space-between;
      gap: 14px;
      margin-bottom: 14px;
    }
    .tag {
      display: inline-block;
      padding: 5px 10px;
      border-radius: 999px;
      background: rgba(192,90,43,0.12);
      color: var(--warm);
      font-size: 12px;
      font-weight: 700;
      margin-right: 8px;
    }
    .mode {
      background: rgba(31,122,82,0.12);
      color: var(--ok);
    }
    .name { font-size: 24px; margin: 0 0 4px; }
    .meta { font-size: 13px; color: var(--muted); word-break: break-all; }
    .section-title { margin: 16px 0 8px; font-size: 14px; color: var(--muted); }
    .link-row {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-bottom: 8px;
    }
    .link-chip, button {
      border: 0;
      border-radius: 14px;
      padding: 10px 14px;
      font: inherit;
    }
    .link-chip {
      display: inline-block;
      background: rgba(13,92,99,0.12);
      color: var(--accent);
      text-decoration: none;
    }
    button {
      background: #8f2d2d;
      color: #fff;
      cursor: pointer;
    }
    button:hover { background: #742020; }
    .empty {
      padding: 36px 22px;
      color: var(--muted);
      text-align: center;
    }
    @media (max-width: 720px) {
      body { padding: 10px; }
      .hero, .cards { padding: 16px; }
      .card-top { flex-direction: column; }
    }
  </style>
  <script>
    async function stopShare(id) {
      const ok = window.confirm('停止这个分享？');
      if (!ok) return;
      const resp = await fetch('/api/shares/' + id + '/stop', { method: 'POST' });
      if (resp.ok) {
        window.location.reload();
        return;
      }
      alert('停止分享失败');
    }
  </script>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <span class="eyebrow">Local Manager</span>
      <h1>正在共享的内容</h1>
      <p>右键新建的分享会进入这里统一管理。这个页面只允许本机访问，外部访问者只能打开具体分享页。</p>
    </section>
    <section class="cards">
      {{if .Shares}}
        {{range .Shares}}
        <article class="card">
          <div class="card-top">
            <div>
              <h2 class="name">{{.Name}}</h2>
              <div class="meta">{{.Path}}</div>
            </div>
            <div>
              <span class="tag">{{.Type}}</span>
              <span class="tag mode">{{.Mode}}</span>
            </div>
          </div>
          <div class="meta">创建于 {{.CreatedAt}}，最近活动 {{.UpdatedAt}}</div>
          <div class="section-title">本机访问</div>
          <div class="link-row">
            <a class="link-chip" href="{{.LocalURL}}" target="_blank">{{.LocalURL}}</a>
          </div>
          <div class="section-title">局域网访问</div>
          <div class="link-row">
            {{range .NetworkLinks}}
            <a class="link-chip" href="{{.}}" target="_blank">{{.}}</a>
            {{else}}
            <span class="meta">未检测到可用局域网 IPv4 地址</span>
            {{end}}
          </div>
          <div class="section-title">操作</div>
          <button type="button" onclick="stopShare('{{.ID}}')">停止分享</button>
        </article>
        {{end}}
      {{else}}
        <div class="empty">当前还没有分享内容。使用右键菜单新建分享后，这里会实时显示列表。</div>
      {{end}}
    </section>
  </div>
</body>
</html>{{end}}
`

const shareHTML = `{{define "share"}}<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}}</title>
  <style>
    :root {
      --bg: #f4efe6;
      --panel: rgba(255,252,247,0.95);
      --line: #d7c8b2;
      --text: #2d241c;
      --muted: #6d6154;
      --accent: #0b6e4f;
      --accent-strong: #084c39;
      --danger: #9d2b2b;
      --ok: #1b6f4f;
      --shadow: 0 18px 50px rgba(56, 41, 24, 0.12);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: "Segoe UI", "PingFang SC", sans-serif;
      color: var(--text);
      background:
        radial-gradient(circle at top left, rgba(237,196,128,0.35), transparent 28%),
        radial-gradient(circle at bottom right, rgba(11,110,79,0.18), transparent 24%),
        linear-gradient(135deg, #efe4d2 0%, #f8f3ea 45%, #ece5db 100%);
      padding: 24px;
    }
    .shell {
      max-width: 980px;
      margin: 0 auto;
      background: var(--panel);
      border: 1px solid rgba(255,255,255,0.6);
      border-radius: 28px;
      box-shadow: var(--shadow);
      overflow: hidden;
      backdrop-filter: blur(10px);
    }
    .hero {
      padding: 28px 28px 18px;
      border-bottom: 1px solid var(--line);
      background: linear-gradient(135deg, rgba(11,110,79,0.08), rgba(237,196,128,0.16));
    }
    .eyebrow {
      display: inline-block;
      padding: 6px 10px;
      border-radius: 999px;
      background: rgba(11,110,79,0.12);
      color: var(--accent-strong);
      font-size: 12px;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    h1 {
      margin: 14px 0 10px;
      font-size: clamp(28px, 4vw, 42px);
      line-height: 1.05;
    }
    .meta {
      color: var(--muted);
      font-size: 14px;
      word-break: break-all;
    }
    .hero-stack {
      display: grid;
      gap: 8px;
      margin-top: 10px;
    }
    .hero-link {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      max-width: 100%;
      padding: 10px 14px;
      border-radius: 14px;
      background: rgba(11,110,79,0.08);
      word-break: break-all;
    }
    .content {
      padding: 24px 28px 32px;
      display: grid;
      grid-template-columns: 1.1fr 0.9fr;
      gap: 22px;
    }
    .card {
      border: 1px solid var(--line);
      border-radius: 22px;
      padding: 18px;
      background: rgba(255,255,255,0.62);
    }
    .card h2 {
      margin: 0 0 12px;
      font-size: 18px;
    }
    .hint {
      margin: 0 0 14px;
      color: var(--muted);
      font-size: 14px;
      line-height: 1.5;
    }
    .status {
      margin-bottom: 12px;
      padding: 12px 14px;
      border-radius: 14px;
      font-size: 14px;
    }
    .status.error { background: rgba(157,43,43,0.1); color: var(--danger); }
    .status.ok { background: rgba(27,111,79,0.1); color: var(--ok); }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 14px;
    }
    th, td {
      padding: 12px 8px;
      text-align: left;
      border-bottom: 1px solid rgba(215,200,178,0.7);
    }
    th { color: var(--muted); font-weight: 600; }
    td:first-child {
      word-break: break-word;
    }
    .section-divider {
      height: 1px;
      margin: 16px 0;
      background: rgba(215,200,178,0.7);
    }
    a {
      color: var(--accent-strong);
      text-decoration: none;
    }
    a:hover { text-decoration: underline; }
    .download {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 12px 16px;
      border-radius: 14px;
      background: var(--accent);
      color: white;
      font-weight: 600;
    }
    .download:hover { text-decoration: none; background: var(--accent-strong); }
    form {
      display: grid;
      gap: 12px;
    }
    input[type="file"], input[type="password"] {
      width: 100%;
      padding: 12px 14px;
      border-radius: 14px;
      border: 1px solid var(--line);
      background: white;
      font: inherit;
    }
    button {
      border: 0;
      border-radius: 14px;
      padding: 12px 16px;
      background: #c66b3d;
      color: white;
      font: inherit;
      font-weight: 600;
      cursor: pointer;
    }
    button:hover { background: #a7552d; }
    .readonly {
      margin: 0;
      padding: 14px;
      border-radius: 14px;
      background: rgba(109,97,84,0.1);
      color: var(--muted);
      font-size: 14px;
      line-height: 1.5;
    }
    @media (max-width: 760px) {
      body { padding: 12px; }
      .content { grid-template-columns: 1fr; padding: 16px; }
      .hero { padding: 20px 18px 14px; }
      .card { padding: 16px; }
      table, thead, tbody, tr, th, td {
        display: block;
        width: 100%;
      }
      thead {
        display: none;
      }
      tbody {
        display: grid;
        gap: 12px;
      }
      tr {
        padding: 14px;
        border: 1px solid rgba(215,200,178,0.7);
        border-radius: 16px;
        background: rgba(255,255,255,0.82);
      }
      td {
        padding: 0;
        border: 0;
      }
      td + td {
        margin-top: 8px;
      }
      td:nth-child(1)::before,
      td:nth-child(2)::before,
      td:nth-child(3)::before {
        display: block;
        margin-bottom: 4px;
        color: var(--muted);
        font-size: 12px;
        letter-spacing: 0.02em;
      }
      td:nth-child(1)::before { content: "名称"; }
      td:nth-child(2)::before { content: "大小"; }
      td:nth-child(3)::before { content: "修改时间"; }
      .download,
      button {
        width: 100%;
        justify-content: center;
      }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <span class="eyebrow">{{if .IsDir}}Folder Share{{else}}File Share{{end}}</span>
      <h1>{{.SharedName}}</h1>
      <div class="hero-stack">
        <div class="meta">路径: {{.SharedPath}}</div>
        <a class="hero-link" href="{{.Address}}">{{.Address}}</a>
      </div>
    </section>
    <section class="content">
      <div class="card">
        <h2>{{if .IsDir}}内容列表{{else}}文件下载{{end}}</h2>
        <p class="hint">{{if .IsDir}}目录默认只读。只有设置上传密码时，页面才允许上传文件到当前目录。{{else}}文件分享始终只读，可直接下载。{{end}}</p>
        {{if .ErrorMessage}}<div class="status error">{{.ErrorMessage}}</div>{{end}}
        {{if .SuccessMessage}}<div class="status ok">{{.SuccessMessage}}</div>{{end}}
        {{if .IsDir}}
          <table>
            <thead>
              <tr>
                <th>名称</th>
                <th>大小</th>
                <th>修改时间</th>
              </tr>
            </thead>
            <tbody>
              {{range .Items}}
              <tr>
                <td>{{if .URL}}<a href="{{.URL}}">{{.Name}}</a>{{else}}{{.Name}}{{end}}</td>
                <td>{{.Size}}</td>
                <td>{{.ModTime}}</td>
              </tr>
              {{else}}
              <tr><td colspan="3">目录为空</td></tr>
              {{end}}
            </tbody>
          </table>
        {{else}}
          <a class="download" href="/s/{{.ShareID}}/raw">下载文件</a>
        {{end}}
      </div>
      <div class="card">
        <h2>{{if .UploadEnabled}}上传入口{{else}}访问模式{{end}}</h2>
        {{if .UploadEnabled}}
          <p class="hint">输入分享者设置的上传密码后，可把文件上传到当前共享目录根目录。</p>
          <div class="section-divider"></div>
          <form action="/s/{{.ShareID}}/upload" method="post" enctype="multipart/form-data">
            <input type="file" name="file" required>
            <input type="password" name="password" placeholder="上传密码" required>
            <button type="submit">上传文件</button>
          </form>
        {{else}}
          <p class="readonly">
            {{if .IsDir}}
            当前目录分享为只读模式。若需要上传文件，请由分享者重新设置带密码的共享。
            {{else}}
            文件分享不提供上传能力。
            {{end}}
          </p>
        {{end}}
      </div>
    </section>
  </div>
</body>
</html>{{end}}
`
