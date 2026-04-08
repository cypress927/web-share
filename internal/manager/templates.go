package manager

const homeHTML = `{{define "home"}}<!DOCTYPE html>
<html lang="{{.CurrentLang}}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}}</title>
  <style>
    :root {
      --bg: #efe9dd;
      --panel: rgba(255,255,255,0.88);
      --line: #d7cbb8;
      --text: #241d16;
      --muted: #6f675d;
      --accent: #0f676b;
      --warm: #bf6738;
      --shadow: 0 20px 60px rgba(43, 31, 18, 0.12);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: "Segoe UI", "PingFang SC", sans-serif;
      color: var(--text);
      background:
        radial-gradient(circle at top left, rgba(191,103,56,0.16), transparent 24%),
        radial-gradient(circle at bottom right, rgba(15,103,107,0.14), transparent 30%),
        linear-gradient(135deg, #e6dcc8 0%, #f7f3eb 48%, #ece6dd 100%);
      padding: 18px;
    }
    .shell {
      max-width: 980px;
      margin: 0 auto;
      background: var(--panel);
      border-radius: 28px;
      border: 1px solid rgba(255,255,255,0.7);
      box-shadow: var(--shadow);
      overflow: hidden;
      backdrop-filter: blur(10px);
    }
    .hero {
      padding: 24px;
      border-bottom: 1px solid var(--line);
      background: linear-gradient(135deg, rgba(15,103,107,0.08), rgba(191,103,56,0.08));
    }
    .lang-switch {
      display: flex;
      justify-content: flex-end;
      gap: 8px;
      font-size: 13px;
      margin-bottom: 8px;
    }
    .lang-switch a.active {
      font-weight: 700;
      text-decoration: underline;
    }
    .eyebrow {
      display: inline-block;
      padding: 6px 10px;
      border-radius: 999px;
      background: rgba(15,103,107,0.12);
      color: var(--accent);
      font-size: 12px;
      letter-spacing: 0.1em;
      text-transform: uppercase;
    }
    h1 {
      margin: 14px 0 8px;
      font-size: clamp(30px, 5vw, 46px);
    }
    p { margin: 0; color: var(--muted); line-height: 1.6; }
    .content {
      padding: 20px;
      display: grid;
      gap: 18px;
    }
    .card {
      border: 1px solid var(--line);
      border-radius: 22px;
      padding: 18px;
      background: rgba(255,255,255,0.72);
    }
    .code-form {
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      gap: 10px;
      margin-top: 14px;
    }
    input[type="text"] {
      width: 100%;
      padding: 12px 14px;
      border-radius: 14px;
      border: 1px solid var(--line);
      font: inherit;
    }
    button {
      border: 0;
      border-radius: 14px;
      padding: 12px 16px;
      background: var(--accent);
      color: white;
      font: inherit;
      font-weight: 600;
      cursor: pointer;
    }
    .status {
      margin-top: 12px;
      padding: 12px 14px;
      border-radius: 14px;
      background: rgba(154,43,43,0.1);
      color: #9a2b2b;
      font-size: 14px;
    }
    .share-list {
      display: grid;
      gap: 12px;
      margin-top: 14px;
    }
    .share-item {
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      align-items: start;
      gap: 12px;
      padding: 14px;
      border-radius: 18px;
      background: rgba(255,255,255,0.84);
      border: 1px solid rgba(215,203,184,0.8);
    }
    .share-item.unavailable {
      background: rgba(154,43,43,0.06);
      border-color: rgba(154,43,43,0.18);
    }
    .share-name {
      margin: 0 0 4px;
      font-size: 18px;
    }
    .meta {
      color: var(--muted);
      font-size: 13px;
      word-break: break-all;
    }
    .code-chip {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 8px 12px;
      align-self: start;
      border-radius: 999px;
      background: rgba(191,103,56,0.12);
      color: var(--warm);
      font-weight: 700;
    }
    .status-chip {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 6px 10px;
      border-radius: 999px;
      background: rgba(15,103,107,0.12);
      color: var(--accent);
      font-size: 12px;
      font-weight: 700;
    }
    .status-chip.unavailable {
      background: rgba(154,43,43,0.1);
      color: #9a2b2b;
    }
    .share-meta-row {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-top: 8px;
      align-items: center;
    }
    .share-content {
      margin-top: 10px;
      padding: 10px 12px;
      border-radius: 12px;
      border: 1px solid rgba(215,203,184,0.7);
      background: rgba(255,255,255,0.86);
      color: var(--text);
      line-height: 1.5;
      white-space: pre-wrap;
      word-break: break-word;
      font-size: 14px;
    }
    .thumb {
      width: 100%;
      max-width: 220px;
      max-height: 150px;
      object-fit: contain;
      border-radius: 12px;
      border: 1px solid rgba(215,203,184,0.7);
      background: #fff;
      padding: 6px;
      margin-top: 10px;
    }
    .share-actions {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-top: 10px;
    }
    .action-btn {
      border: 1px solid rgba(15,103,107,0.26);
      border-radius: 10px;
      padding: 8px 12px;
      background: rgba(15,103,107,0.08);
      color: var(--accent);
      font-size: 13px;
      font-weight: 600;
      cursor: pointer;
      text-decoration: none;
    }
    .action-btn.primary {
      border-color: transparent;
      background: var(--accent);
      color: #fff;
    }
    .action-btn:hover {
      text-decoration: none;
      filter: brightness(0.96);
    }
    .share-link {
      color: var(--accent);
      font-weight: 600;
    }
    a {
      color: var(--accent);
      text-decoration: none;
    }
    a:hover { text-decoration: underline; }
    .empty {
      color: var(--muted);
      text-align: center;
      padding: 18px 0 4px;
    }
    @media (max-width: 720px) {
      body { padding: 10px; }
      .hero, .content { padding: 16px; }
      .code-form { grid-template-columns: 1fr; }
      .share-item {
        grid-template-columns: 1fr;
      }
      .code-chip { width: fit-content; }
      .thumb { max-width: 100%; }
    }
  </style>
  <script>
    async function copyShareText(url) {
      if (!url) return;
      try {
        const resp = await fetch(url, { cache: "no-store" });
        if (!resp.ok) {
          throw new Error("copy fetch failed");
        }
        const value = await resp.text();
        await navigator.clipboard.writeText(value);
      } catch (_) {
        alert({{printf "%q" (tr .CurrentLang "home.copy_failed")}});
      }
    }
  </script>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="lang-switch">
        <span>{{tr .CurrentLang "lang.switch"}}:</span>
        <a href="{{.LangZHURL}}" class="{{if eq .CurrentLang "zh-CN"}}active{{end}}">{{tr .CurrentLang "lang.zh"}}</a>
        <a href="{{.LangENURL}}" class="{{if eq .CurrentLang "en-US"}}active{{end}}">{{tr .CurrentLang "lang.en"}}</a>
      </div>
      <span class="eyebrow">{{tr .CurrentLang "site.brand"}}</span>
      <h1>{{tr .CurrentLang "home.title"}}</h1>
      <p>{{tr .CurrentLang "home.subtitle"}}</p>
    </section>
    <section class="content">
      <div class="card">
        <h2>{{tr .CurrentLang "home.input_code"}}</h2>
        <form class="code-form" action="/" method="get">
          <input type="text" name="code" placeholder="{{tr .CurrentLang "home.code_placeholder"}}" autocomplete="off" required>
          <button type="submit">{{tr .CurrentLang "home.open_share"}}</button>
        </form>
        {{if .ErrorMessage}}<div class="status">{{.ErrorMessage}}</div>{{end}}
      </div>
      <div class="card">
        <h2>{{tr .CurrentLang "home.visible"}}</h2>
        {{if .VisibleShares}}
        <div class="share-list">
          {{range .VisibleShares}}
          <div class="share-item {{if .Unavailable}}unavailable{{end}}">
            <div>
              <h3 class="share-name">{{.Name}}</h3>
              <div class="meta">{{.Type}}</div>
              {{if .FileName}}<div class="meta">{{tr $.CurrentLang "home.file_meta_prefix"}}: {{.FileName}}{{if .FileSize}} · {{.FileSize}}{{end}}</div>{{end}}
              {{if .PreviewText}}<div class="share-content">{{.PreviewText}}</div>{{end}}
              {{if .ShowThumbnail}}<img class="thumb" src="{{.ContentURL}}" alt="{{.Name}} thumbnail">{{end}}
              {{if not .Unavailable}}
              <div class="share-actions">
                {{if .ShowCopy}}<button class="action-btn" type="button" onclick='copyShareText({{printf "%q" .CopyURL}})'>{{tr $.CurrentLang "home.copy"}}</button>{{end}}
                {{if .ShowDownload}}<a class="action-btn primary" href="{{.DownloadURL}}">{{tr $.CurrentLang "home.download"}}</a>{{end}}
              </div>
              {{end}}
              <div class="share-meta-row">
                <span class="status-chip {{if .Unavailable}}unavailable{{end}}">{{.Status}}</span>
                <a class="share-link" href="{{.URL}}">{{tr $.CurrentLang "home.open"}}</a>
              </div>
            </div>
            <span class="code-chip">{{tr $.CurrentLang "home.share_code"}} {{.Code}}</span>
          </div>
          {{end}}
        </div>
        {{else}}
        <div class="empty">{{tr .CurrentLang "home.empty"}}</div>
        {{end}}
      </div>
    </section>
  </div>
</body>
</html>{{end}}
`

const manageHTML = `{{define "manage"}}<!DOCTYPE html>
<html lang="{{.CurrentLang}}">
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
      max-width: 1180px;
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
    .lang-switch {
      display: flex;
      justify-content: flex-end;
      gap: 8px;
      font-size: 13px;
      margin-bottom: 8px;
    }
    .lang-switch a.active {
      font-weight: 700;
      text-decoration: underline;
    }
    .lang-default-form {
      margin-top: 12px;
      display: flex;
      gap: 8px;
      align-items: center;
      flex-wrap: wrap;
    }
    .lang-default-form select {
      padding: 8px 10px;
      border-radius: 10px;
      border: 1px solid var(--line);
      font: inherit;
      background: #fff;
    }
    .lang-default-form button {
      padding: 8px 12px;
      border-radius: 10px;
    }
    .cards {
      display: grid;
      gap: 18px;
      padding: 22px;
    }
    .card {
      display: grid;
      grid-template-columns: 1.3fr 0.7fr;
      gap: 18px;
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: 18px;
      background: rgba(255,255,255,0.75);
    }
    .name { font-size: 24px; margin: 0 0 6px; }
    .meta { font-size: 13px; color: var(--muted); word-break: break-all; }
    .tags { display: flex; flex-wrap: wrap; gap: 8px; margin: 12px 0; }
    .tag {
      display: inline-block;
      padding: 5px 10px;
      border-radius: 999px;
      background: rgba(192,90,43,0.12);
      color: var(--warm);
      font-size: 12px;
      font-weight: 700;
    }
    .tag.ok {
      background: rgba(31,122,82,0.12);
      color: var(--ok);
    }
    .section-title { margin: 16px 0 8px; font-size: 14px; color: var(--muted); }
    .clip-preview-text {
      margin-top: 12px;
      padding: 12px;
      border-radius: 12px;
      border: 1px solid rgba(213,208,196,0.9);
      background: rgba(255,255,255,0.88);
      color: var(--text);
      line-height: 1.55;
      font-size: 14px;
      white-space: pre-wrap;
      word-break: break-word;
    }
    .clip-preview-image {
      width: 100%;
      max-height: 180px;
      object-fit: contain;
      margin-top: 12px;
      border-radius: 12px;
      border: 1px solid rgba(213,208,196,0.9);
      background: #fff;
      padding: 6px;
    }
    .code-chip {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 10px 14px;
      border-radius: 999px;
      background: rgba(13,92,99,0.12);
      color: var(--accent);
      font-weight: 700;
    }
    .link-row {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-bottom: 8px;
    }
    .link-chip {
      display: inline-block;
      padding: 10px 14px;
      border-radius: 14px;
      background: rgba(13,92,99,0.12);
      color: var(--accent);
      text-decoration: none;
      word-break: break-all;
    }
    .controls {
      display: grid;
      gap: 12px;
      align-content: start;
    }
    .qr-box {
      border: 1px solid var(--line);
      border-radius: 20px;
      padding: 14px;
      background: rgba(255,255,255,0.84);
      text-align: center;
    }
    .qr-box img {
      width: 180px;
      max-width: 100%;
      border-radius: 14px;
      background: white;
      padding: 10px;
    }
    form {
      display: grid;
      gap: 10px;
    }
    input[type="text"] {
      width: 100%;
      padding: 12px 14px;
      border-radius: 14px;
      border: 1px solid var(--line);
      font: inherit;
    }
    label.toggle {
      display: flex;
      align-items: center;
      gap: 10px;
      color: var(--muted);
      font-size: 14px;
    }
    input[type="checkbox"] {
      width: 18px;
      height: 18px;
    }
    .action-row {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
    }
    button {
      border: 0;
      border-radius: 14px;
      padding: 12px 16px;
      font: inherit;
      font-weight: 600;
      cursor: pointer;
      background: var(--accent);
      color: white;
    }
    button.secondary {
      background: #8f2d2d;
    }
    .empty {
      padding: 36px 22px;
      color: var(--muted);
      text-align: center;
    }
    @media (max-width: 820px) {
      body { padding: 10px; }
      .hero, .cards { padding: 16px; }
      .card { grid-template-columns: 1fr; }
    }
  </style>
  <script>
    async function stopShare(id) {
      const ok = window.confirm({{printf "%q" (tr .CurrentLang "manage.stop_confirm")}});
      if (!ok) return;
      const resp = await fetch('/api/shares/' + id + '/stop', { method: 'POST' });
      if (resp.ok) {
        window.location.reload();
        return;
      }
      alert({{printf "%q" (tr .CurrentLang "manage.stop_failed")}});
    }
  </script>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="lang-switch">
        <span>{{tr .CurrentLang "lang.switch"}}:</span>
        <a href="{{.LangZHURL}}" class="{{if eq .CurrentLang "zh-CN"}}active{{end}}">{{tr .CurrentLang "lang.zh"}}</a>
        <a href="{{.LangENURL}}" class="{{if eq .CurrentLang "en-US"}}active{{end}}">{{tr .CurrentLang "lang.en"}}</a>
      </div>
      <span class="eyebrow">{{tr .CurrentLang "site.brand"}}</span>
      <h1>{{tr .CurrentLang "manage.title"}}</h1>
      <p>{{tr .CurrentLang "manage.subtitle"}}</p>
      <form class="lang-default-form" action="/manage/settings/language" method="post">
        <span>{{tr .CurrentLang "manage.default_lang"}}:</span>
        <select name="default_lang">
          <option value="zh-CN" {{if eq .DefaultLang "zh-CN"}}selected{{end}}>{{tr .CurrentLang "lang.zh"}}</option>
          <option value="en-US" {{if eq .DefaultLang "en-US"}}selected{{end}}>{{tr .CurrentLang "lang.en"}}</option>
        </select>
        <button type="submit">{{tr .CurrentLang "manage.default_lang_apply"}}</button>
      </form>
    </section>
    <section class="cards">
      {{if .Shares}}
        {{range .Shares}}
        <article class="card">
          <div>
            <h2 class="name">{{.Name}}</h2>
            <div class="meta">{{.Path}}</div>
            <div class="tags">
              <span class="tag">{{.Type}}</span>
              <span class="tag ok">{{.Mode}}</span>
              <span class="tag">{{.Visibility}}</span>
            </div>
            <div class="section-title">{{tr $.CurrentLang "manage.section_share_code"}}</div>
            <div class="code-chip">{{.Code}}</div>
            <div class="section-title">{{tr $.CurrentLang "manage.section_public_home"}}</div>
            <div class="link-row">
              <a class="link-chip" href="{{.PublicURL}}" target="_blank">{{.PublicURL}}</a>
            </div>
            <div class="section-title">{{tr $.CurrentLang "manage.section_local_access"}}</div>
            <div class="link-row">
              <a class="link-chip" href="{{.LocalURL}}" target="_blank">{{.LocalURL}}</a>
            </div>
            <div class="section-title">{{tr $.CurrentLang "manage.section_lan_access"}}</div>
            <div class="link-row">
              {{range .NetworkLinks}}
              <a class="link-chip" href="{{.}}" target="_blank">{{.}}</a>
              {{else}}
              <span class="meta">{{tr $.CurrentLang "manage.no_lan_ipv4"}}</span>
              {{end}}
            </div>
            <div class="section-title">{{tr $.CurrentLang "manage.section_time"}}</div>
            <div class="meta">{{tr $.CurrentLang "manage.created_at"}} {{.CreatedAt}}, {{tr $.CurrentLang "manage.updated_at"}} {{.UpdatedAt}}</div>
            {{if .PreviewText}}
            <div class="clip-preview-text">{{.PreviewText}}</div>
            {{end}}
            {{if .PreviewImage}}
            <img class="clip-preview-image" src="{{.PreviewImage}}" alt="{{.Name}} preview">
            {{end}}
          </div>
          <div class="controls">
            <div class="qr-box">
              <img src="{{.QRCodeDataURL}}" alt="Share QR Code">
              <div class="meta" style="margin-top: 10px;">{{tr $.CurrentLang "manage.qr_hint"}}</div>
            </div>
            <form action="/manage/shares/{{.ID}}/update" method="post">
              <input type="text" name="name" value="{{.NameInput}}" required>
              <label class="toggle">
                <input type="checkbox" name="visible" {{if .VisibleChecked}}checked{{end}}>
                {{tr $.CurrentLang "manage.show_on_home"}}
              </label>
              <div class="action-row">
                <button type="submit">{{tr $.CurrentLang "manage.save"}}</button>
                <button class="secondary" type="button" onclick="stopShare('{{.ID}}')">{{tr $.CurrentLang "manage.stop"}}</button>
              </div>
            </form>
          </div>
        </article>
        {{end}}
      {{else}}
        <div class="empty">{{if eq .CurrentLang "zh-CN"}}当前还没有分享内容。使用右键菜单新建分享后，这里会实时显示列表。{{else}}No shares yet. Create one from the context menu and it will appear here in real time.{{end}}</div>
      {{end}}
    </section>
  </div>
</body>
</html>{{end}}
`

const shareHTML = `{{define "share"}}<!DOCTYPE html>
<html lang="{{.CurrentLang}}">
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
    .lang-switch {
      display: flex;
      justify-content: flex-end;
      gap: 8px;
      font-size: 13px;
      margin-bottom: 8px;
    }
    .lang-switch a.active {
      font-weight: 700;
      text-decoration: underline;
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
    .code-chip {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      width: fit-content;
      padding: 8px 12px;
      border-radius: 999px;
      background: rgba(198,107,61,0.14);
      color: #8e4a26;
      font-size: 13px;
      font-weight: 700;
    }
    .crumbs {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      align-items: center;
    }
    .crumb-sep {
      color: var(--muted);
      font-size: 12px;
    }
    .path-row {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      align-items: center;
    }
    .path-chip {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 8px 12px;
      border-radius: 999px;
      background: rgba(11,110,79,0.1);
      color: var(--accent-strong);
      font-size: 13px;
      font-weight: 600;
    }
    .back-link {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 8px 12px;
      border-radius: 999px;
      background: rgba(109,97,84,0.1);
      color: var(--muted);
      font-size: 13px;
      font-weight: 600;
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
    .folder-link {
      font-weight: 700;
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
    .item-actions {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      align-items: center;
    }
    .inline-link {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      color: var(--accent-strong);
      font-weight: 600;
    }
    .clipboard-text {
      margin: 0;
      padding: 14px;
      border-radius: 14px;
      border: 1px solid rgba(215,200,178,0.8);
      background: rgba(255,255,255,0.8);
      color: var(--text);
      line-height: 1.6;
      font-size: 14px;
      white-space: pre-wrap;
      word-break: break-word;
      max-height: 420px;
      overflow: auto;
    }
    .preview-image {
      width: 100%;
      max-height: 520px;
      object-fit: contain;
      border-radius: 14px;
      border: 1px solid rgba(215,200,178,0.8);
      background: rgba(255,255,255,0.85);
      padding: 8px;
    }
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
    button:disabled {
      cursor: wait;
      opacity: 0.75;
    }
    .readonly {
      margin: 0;
      padding: 14px;
      border-radius: 14px;
      background: rgba(109,97,84,0.1);
      color: var(--muted);
      font-size: 14px;
      line-height: 1.5;
    }
    .upload-status {
      display: grid;
      gap: 10px;
    }
    .progress-shell {
      overflow: hidden;
      height: 12px;
      border-radius: 999px;
      background: rgba(11,110,79,0.12);
    }
    .progress-bar {
      width: 0%;
      height: 100%;
      border-radius: inherit;
      background: linear-gradient(90deg, #0b6e4f, #c66b3d);
      transition: width 0.2s ease;
    }
    .progress-meta {
      display: grid;
      gap: 6px;
      color: var(--muted);
      font-size: 13px;
    }
    .upload-status-text {
      color: var(--text);
      font-weight: 600;
    }
    .upload-error {
      color: var(--danger);
    }
    .upload-ok {
      color: var(--ok);
    }
    .upload-actions {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
    }
    .ghost-button {
      border: 1px solid rgba(198,107,61,0.35);
      background: rgba(198,107,61,0.08);
      color: #8e4a26;
    }
    .ghost-button:hover {
      background: rgba(198,107,61,0.18);
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
      td:nth-child(1)::before { content: "{{tr .CurrentLang "share.col_name"}}"; }
      td:nth-child(2)::before { content: "{{tr .CurrentLang "share.col_size"}}"; }
      td:nth-child(3)::before { content: "{{tr .CurrentLang "share.col_mod_time"}}"; }
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
      <div class="lang-switch">
        <span>{{tr .CurrentLang "lang.switch"}}:</span>
        <a href="{{.LangZHURL}}" class="{{if eq .CurrentLang "zh-CN"}}active{{end}}">{{tr .CurrentLang "lang.zh"}}</a>
        <a href="{{.LangENURL}}" class="{{if eq .CurrentLang "en-US"}}active{{end}}">{{tr .CurrentLang "lang.en"}}</a>
      </div>
      <span class="eyebrow">{{.ShareTypeLabel}}</span>
      <h1>{{.SharedName}}</h1>
      <div class="hero-stack">
        <div class="code-chip">{{tr .CurrentLang "home.share_code"}} {{.ShareCode}}</div>
        {{if and .IsDir (not .Unavailable)}}
        <div class="crumbs">
          {{range $index, $crumb := .Breadcrumbs}}
            {{if $index}}<span class="crumb-sep">/</span>{{end}}
            <a href="{{$crumb.URL}}">{{$crumb.Name}}</a>
          {{end}}
        </div>
        <div class="path-row">
          <span class="path-chip">{{tr .CurrentLang "share.current_dir"}} {{.CurrentLabel}}</span>
          {{if .ParentURL}}<a class="back-link" href="{{.ParentURL}}">{{tr .CurrentLang "share.back_parent"}}</a>{{end}}
        </div>
        {{end}}
        {{if .SharedPath}}<div class="meta">{{tr .CurrentLang "share.path"}}: {{.SharedPath}}</div>{{end}}
        <a class="hero-link" href="{{.Address}}">{{.Address}}</a>
      </div>
    </section>
    <section class="content">
      <div class="card">
        <h2>{{if eq .ShareKind "clipboard_text"}}{{tr .CurrentLang "share.h2.clipboard_text"}}{{else if eq .ShareKind "clipboard_image"}}{{tr .CurrentLang "share.h2.clipboard_image"}}{{else if .IsDir}}{{tr .CurrentLang "share.h2.dir"}}{{else}}{{tr .CurrentLang "share.h2.file"}}{{end}}</h2>
        <p class="hint">{{if eq .ShareKind "clipboard_text"}}{{tr .CurrentLang "share.hint.clipboard_text"}}{{else if eq .ShareKind "clipboard_image"}}{{tr .CurrentLang "share.hint.clipboard_image"}}{{else if .IsDir}}{{tr .CurrentLang "share.hint.dir"}}{{else if eq .PreviewKind "text"}}{{tr .CurrentLang "share.hint.text_file"}}{{else if eq .PreviewKind "image"}}{{tr .CurrentLang "share.hint.image_file"}}{{else}}{{tr .CurrentLang "share.hint.file"}}{{end}}</p>
        {{if .ErrorMessage}}<div class="status error">{{.ErrorMessage}}</div>{{end}}
        {{if .SuccessMessage}}<div class="status ok">{{.SuccessMessage}}</div>{{end}}
        {{if .Unavailable}}
          <p class="readonly">{{tr .CurrentLang "share.readonly.missing"}}</p>
        {{else if eq .ShareKind "clipboard_text"}}
          <pre class="clipboard-text">{{.TextContent}}</pre>
          <div class="section-divider"></div>
          <a class="download" href="{{.DownloadURL}}">{{tr .CurrentLang "share.download_text"}}</a>
        {{else if eq .ShareKind "clipboard_image"}}
          <img class="preview-image" src="{{.ContentURL}}" alt="Clipboard Image Preview">
          <div class="section-divider"></div>
          <a class="download" href="{{.DownloadURL}}">{{tr .CurrentLang "share.download_image"}}</a>
        {{else if and (not .IsDir) (eq .PreviewKind "text")}}
          <pre class="clipboard-text">{{.PreviewText}}</pre>
          <div class="section-divider"></div>
          <a class="download" href="{{.DownloadURL}}">{{tr .CurrentLang "share.download"}}</a>
        {{else if and (not .IsDir) (eq .PreviewKind "image")}}
          <img class="preview-image" src="{{.ContentURL}}" alt="File Image Preview">
          <div class="section-divider"></div>
          <a class="download" href="{{.DownloadURL}}">{{tr .CurrentLang "share.download_image"}}</a>
        {{else if .IsDir}}
          <div class="upload-actions">
            <a class="download" href="/s/{{.ShareCode}}/archive">{{tr .CurrentLang "share.download_all"}}</a>
          </div>
          <div class="section-divider"></div>
          <table>
            <thead>
              <tr>
                <th>{{tr .CurrentLang "share.col_name"}}</th>
                <th>{{tr .CurrentLang "share.col_size"}}</th>
                <th>{{tr .CurrentLang "share.col_mod_time"}}</th>
                <th>{{tr .CurrentLang "share.col_actions"}}</th>
              </tr>
            </thead>
            <tbody>
              {{range .Items}}
              <tr>
                <td>{{if .URL}}<a {{if .IsDir}}class="folder-link"{{end}} href="{{.URL}}">{{.Name}}</a>{{else}}{{.Name}}{{end}}</td>
                <td>{{.Size}}</td>
                <td>{{.ModTime}}</td>
                <td>
                  {{if .IsDir}}
                    <div class="item-actions">
                      <a class="inline-link" href="{{.URL}}">{{tr $.CurrentLang "share.enter"}}</a>
                      <a class="inline-link" href="{{.ArchiveURL}}">{{tr $.CurrentLang "share.archive_download"}}</a>
                    </div>
                  {{else}}
                    <a class="inline-link" href="{{.URL}}">{{tr $.CurrentLang "home.download"}}</a>
                  {{end}}
                </td>
              </tr>
              {{else}}
              <tr><td colspan="4">{{tr .CurrentLang "share.folder_empty"}}</td></tr>
              {{end}}
            </tbody>
          </table>
        {{else}}
          <a class="download" href="/s/{{.ShareCode}}/raw">{{tr .CurrentLang "share.download"}}</a>
        {{end}}
      </div>
      <div class="card">
        <h2>{{if .UploadEnabled}}{{tr .CurrentLang "share.h2.upload"}}{{else}}{{tr .CurrentLang "share.h2.access_mode"}}{{end}}</h2>
        {{if .Unavailable}}
          <p class="readonly">{{tr .CurrentLang "share.readonly.unavailable"}}</p>
        {{else if or (eq .ShareKind "clipboard_text") (eq .ShareKind "clipboard_image")}}
          <p class="readonly">{{tr .CurrentLang "share.readonly.clipboard"}}</p>
        {{else if .UploadEnabled}}
          <p class="hint">{{tr .CurrentLang "share.hint.upload_enabled"}}</p>
          <div class="section-divider"></div>
          <form id="upload-form">
            <input type="hidden" name="path" value="{{.CurrentPath}}">
            <input type="file" name="file">
            <input type="file" name="folder" id="folder-input" webkitdirectory directory multiple hidden>
            <input type="password" name="password" placeholder="{{tr .CurrentLang "share.upload_password_placeholder"}}" required>
            <div class="upload-actions">
              <button type="submit" id="upload-button">{{tr .CurrentLang "share.upload_file"}}</button>
              <button type="button" class="ghost-button" id="upload-folder-button">{{tr .CurrentLang "share.upload_folder"}}</button>
            </div>
            <div class="upload-status" id="upload-status" hidden>
              <div class="progress-shell"><div class="progress-bar" id="upload-progress"></div></div>
              <div class="progress-meta">
                <div class="upload-status-text" id="upload-status-text">{{if eq .CurrentLang "zh-CN"}}准备上传{{else}}Preparing upload{{end}}</div>
                <div id="upload-progress-text">0%</div>
                <div id="upload-detail-text"></div>
              </div>
            </div>
          </form>
        {{else}}
          <p class="readonly">
            {{if .IsDir}}
            {{tr .CurrentLang "share.readonly.dir"}}
            {{else}}
            {{tr .CurrentLang "share.readonly.file"}}
            {{end}}
          </p>
        {{end}}
      </div>
    </section>
  </div>
  {{if and .UploadEnabled (not .Unavailable)}}
  <script>
    (() => {
      const form = document.getElementById("upload-form");
      const fileInput = form?.querySelector('input[name="file"]');
      const folderInput = document.getElementById("folder-input");
      const passwordInput = form?.querySelector('input[name="password"]');
      const pathInput = form?.querySelector('input[name="path"]');
      const button = document.getElementById("upload-button");
      const folderButton = document.getElementById("upload-folder-button");
      const statusBox = document.getElementById("upload-status");
      const progressBar = document.getElementById("upload-progress");
      const statusText = document.getElementById("upload-status-text");
      const progressText = document.getElementById("upload-progress-text");
      const detailText = document.getElementById("upload-detail-text");
      const chunkSize = {{.ChunkSize}};
      const shareCode = "{{.ShareCode}}";
      const t = {{if eq .CurrentLang "zh-CN"}}{
        uploaded: "已上传 ",
        chunk: "，分片 ",
        cannotStart: "无法开始上传",
        uploading: "正在上传 ",
        currentFileDone: "当前文件 ",
        doneBytes: "，已完成 ",
        currentFileChunk: "当前文件 ",
        chunkLabel: "，分片 ",
        waitServer: "正在等待服务端写入...",
        selectFiles: "请选择要上传的文件或文件夹",
        needPassword: "请输入上传密码",
        preparing: "正在准备上传...",
        totalFiles: "共 ",
        fileSuffix: " 个文件",
        processing: "正在处理 ",
        indexPrefix: "第 ",
        indexMid: " / ",
        uploadDone: "上传完成",
        uploadedTotal: "共上传 ",
        uploadSuccess: "上传成功",
        uploadFailed: "上传失败",
        chunkUploadFailed: "上传分片失败"
      }{{else}}{
        uploaded: "Uploaded ",
        chunk: ", chunk ",
        cannotStart: "Failed to start upload",
        uploading: "Uploading ",
        currentFileDone: "Current file ",
        doneBytes: ", completed ",
        currentFileChunk: "Current file ",
        chunkLabel: ", chunk ",
        waitServer: "Waiting for server write...",
        selectFiles: "Please select files or a folder to upload",
        needPassword: "Please enter upload password",
        preparing: "Preparing upload...",
        totalFiles: "Total ",
        fileSuffix: " files",
        processing: "Processing ",
        indexPrefix: "File ",
        indexMid: " / ",
        uploadDone: "Upload complete",
        uploadedTotal: "Uploaded ",
        uploadSuccess: "Upload succeeded",
        uploadFailed: "Upload failed",
        chunkUploadFailed: "Chunk upload failed"
      }{{end}};

      if (!form || !fileInput || !folderInput || !passwordInput || !pathInput || !button || !folderButton || !statusBox || !progressBar || !statusText || !progressText || !detailText) {
        return;
      }

      const formatBytes = (value) => {
        if (value < 1024) return value + " B";
        const units = ["KB", "MB", "GB", "TB"];
        let size = value;
        let unit = -1;
        do {
          size /= 1024;
          unit += 1;
        } while (size >= 1024 && unit < units.length - 1);
        return size.toFixed(size >= 100 ? 0 : 1) + " " + units[unit];
      };

      const updateProgress = (uploadedBytes, totalBytes, nextIndex, totalChunks, stateClass, message, detail) => {
        const percent = totalBytes === 0 ? 0 : Math.min(100, (uploadedBytes / totalBytes) * 100);
        statusBox.hidden = false;
        progressBar.style.width = percent.toFixed(2) + "%";
        progressText.textContent = percent.toFixed(percent >= 100 ? 0 : 1) + "%";
        statusText.textContent = message;
        statusText.className = "upload-status-text" + (stateClass ? " " + stateClass : "");
        detailText.textContent = detail || (t.uploaded + formatBytes(uploadedBytes) + " / " + formatBytes(totalBytes) + t.chunk + Math.min(nextIndex, totalChunks) + " / " + totalChunks);
      };

      const setBusy = (busy) => {
        button.disabled = busy;
        folderButton.disabled = busy;
        fileInput.disabled = busy;
        folderInput.disabled = busy;
        passwordInput.disabled = busy;
      };

      const uploadOneFile = async (entry, overall) => {
        const file = entry.file;
        const relativePath = entry.relativePath;
        const totalChunks = Math.max(1, Math.ceil(file.size / chunkSize));
        let uploadedBytes = 0;
        let nextIndex = 0;

        const startResp = await fetch("/s/" + shareCode + "/upload/start", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            path: pathInput.value,
            password: passwordInput.value,
            filePath: relativePath,
            fileSize: file.size,
            chunkSize,
            totalChunks
          })
        });
        if (!startResp.ok) {
          throw new Error(await startResp.text() || t.cannotStart);
        }

        const startData = await startResp.json();
        uploadedBytes = startData.uploadedBytes || 0;
        nextIndex = startData.nextIndex || 0;
        overall.baseBytes += uploadedBytes;

        updateProgress(
          overall.baseBytes,
          overall.totalBytes,
          nextIndex,
          totalChunks,
          "",
          t.uploading + relativePath,
          t.currentFileDone + relativePath + t.doneBytes + formatBytes(overall.baseBytes) + " / " + formatBytes(overall.totalBytes)
        );

        if (startData.done) {
          return;
        }

        for (let index = nextIndex; index < totalChunks; index += 1) {
          const start = index * chunkSize;
          const end = Math.min(file.size, start + chunkSize);
          const chunk = file.slice(start, end);

          updateProgress(
            overall.baseBytes,
            overall.totalBytes,
            index,
            totalChunks,
            "",
            t.uploading + relativePath,
            t.currentFileChunk + relativePath + t.chunkLabel + (index + 1) + " / " + totalChunks
          );

          const chunkResp = await fetch("/s/" + shareCode + "/upload/chunk?upload_id=" + encodeURIComponent(startData.uploadId) + "&index=" + index, {
            method: "POST",
            headers: { "Content-Type": "application/octet-stream" },
            body: chunk
          });
          if (!chunkResp.ok) {
            throw new Error(await chunkResp.text() || t.chunkUploadFailed);
          }

          const chunkData = await chunkResp.json();
          const serverUploadedBytes = chunkData.uploadedBytes || end;
          const delta = Math.max(0, serverUploadedBytes - uploadedBytes);
          uploadedBytes = serverUploadedBytes;
          overall.baseBytes += delta;

          updateProgress(
            overall.baseBytes,
            overall.totalBytes,
            chunkData.nextIndex || (index + 1),
            totalChunks,
            "",
            t.waitServer,
            t.currentFileDone + relativePath + t.doneBytes + formatBytes(overall.baseBytes) + " / " + formatBytes(overall.totalBytes)
          );
        }
      };

      const runUpload = async (entries) => {
        const password = passwordInput.value;
        if (!entries.length) {
          updateProgress(0, 0, 0, 0, "upload-error", t.selectFiles);
          return;
        }
        if (!password) {
          updateProgress(0, 0, 0, 0, "upload-error", t.needPassword);
          return;
        }

        const overall = {
          totalBytes: entries.reduce((sum, entry) => sum + entry.file.size, 0),
          baseBytes: 0
        };
        setBusy(true);

        try {
          updateProgress(0, overall.totalBytes, 0, 0, "", t.preparing, t.totalFiles + entries.length + t.fileSuffix);

          for (let fileIndex = 0; fileIndex < entries.length; fileIndex += 1) {
            const entry = entries[fileIndex];
            updateProgress(
              overall.baseBytes,
              overall.totalBytes,
              0,
              0,
              "",
              t.processing + entry.relativePath,
              t.indexPrefix + (fileIndex + 1) + t.indexMid + entries.length + t.fileSuffix
            );
            await uploadOneFile(entry, overall);
          }

          updateProgress(overall.totalBytes, overall.totalBytes, 0, 0, "upload-ok", t.uploadDone, t.uploadedTotal + entries.length + t.fileSuffix);
          setTimeout(() => {
            const nextURL = new URL(window.location.href);
            nextURL.searchParams.set("success", t.uploadSuccess);
            nextURL.searchParams.delete("error");
            window.location.href = nextURL.toString();
          }, 500);
        } catch (error) {
          updateProgress(overall.baseBytes, overall.totalBytes, 0, 0, "upload-error", error instanceof Error ? error.message : t.uploadFailed);
          setBusy(false);
          return;
        }

        setBusy(false);
      };

      form.addEventListener("submit", async (event) => {
        event.preventDefault();
        const file = fileInput.files?.[0];
        const entries = file ? [{ file, relativePath: file.name }] : [];
        await runUpload(entries);
      });

      folderButton.addEventListener("click", () => folderInput.click());
      folderInput.addEventListener("change", async () => {
        const entries = Array.from(folderInput.files || []).map((file) => ({
          file,
          relativePath: file.webkitRelativePath || file.name
        }));
        await runUpload(entries);
        folderInput.value = "";
      });
    })();
  </script>
  {{end}}
</body>
</html>{{end}}
`
