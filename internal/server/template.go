package server

const pageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}}</title>
  <style>
    :root {
      --bg: #f4efe6;
      --panel: rgba(255, 252, 247, 0.94);
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
        radial-gradient(circle at top left, rgba(237, 196, 128, 0.35), transparent 28%),
        radial-gradient(circle at bottom right, rgba(11, 110, 79, 0.18), transparent 24%),
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
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <span class="eyebrow">{{if .IsDir}}Folder Share{{else}}File Share{{end}}</span>
      <h1>{{.SharedName}}</h1>
      <div class="meta">Path: {{.SharedPath}}</div>
      <div class="meta">Address: <a href="{{.Address}}">{{.Address}}</a></div>
    </section>
    <section class="content">
      <div class="card">
        <h2>{{if .IsDir}}Contents{{else}}File Download{{end}}</h2>
        <p class="hint">{{if .IsDir}}Folder shares are read-only by default. Upload is enabled only when a password is set.{{else}}File shares are always read-only and can be downloaded directly.{{end}}</p>
        {{if .ErrorMessage}}<div class="status error">{{.ErrorMessage}}</div>{{end}}
        {{if .SuccessMessage}}<div class="status ok">{{.SuccessMessage}}</div>{{end}}
        {{if .IsDir}}
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Size</th>
                <th>Modified</th>
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
              <tr><td colspan="3">Folder is empty</td></tr>
              {{end}}
            </tbody>
          </table>
        {{else}}
          <a class="download" href="/raw">Download File</a>
        {{end}}
      </div>
      <div class="card">
        <h2>{{if .UploadEnabled}}Upload{{else}}Access Mode{{end}}</h2>
        {{if .UploadEnabled}}
          <p class="hint">Enter the upload password to upload files to the shared root folder.</p>
          <form action="/upload" method="post" enctype="multipart/form-data">
            <input type="file" name="file" required>
            <input type="password" name="password" placeholder="Upload Password" required>
            <button type="submit">Upload File</button>
          </form>
        {{else}}
          <p class="readonly">
            {{if .IsDir}}
            This folder share is read-only. Recreate the share with a password to enable upload.
            {{else}}
            File shares do not support upload.
            {{end}}
          </p>
        {{end}}
      </div>
    </section>
  </div>
</body>
</html>
`
