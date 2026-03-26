package web

import "net/http"

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(uiHTML))
}

const uiHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>smtp-dev-server</title>
<style>
:root {
  --bg: #0d1117;
  --bg2: #161b22;
  --bg3: #21262d;
  --border: #30363d;
  --text: #e6edf3;
  --text2: #8b949e;
  --accent: #58a6ff;
  --accent2: #388bfd;
  --green: #3fb950;
  --red: #f85149;
  --yellow: #d29922;
  --font: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", sans-serif;
  --mono: "SF Mono", "Fira Code", "Cascadia Code", monospace;
}
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: var(--font); background: var(--bg); color: var(--text); height: 100vh; display: flex; flex-direction: column; }
a { color: var(--accent); text-decoration: none; }
a:hover { text-decoration: underline; }

header {
  background: var(--bg2);
  border-bottom: 1px solid var(--border);
  padding: 12px 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  flex-shrink: 0;
}
header h1 { font-size: 16px; font-weight: 600; }
header .badge {
  background: var(--accent2);
  color: #fff;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}
header .spacer { flex: 1; }
header button {
  background: var(--bg3);
  color: var(--text2);
  border: 1px solid var(--border);
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}
header button:hover { color: var(--text); border-color: var(--text2); }
header button.danger:hover { color: var(--red); border-color: var(--red); }

.tabs {
  background: var(--bg2);
  border-bottom: 1px solid var(--border);
  display: flex;
  padding: 0 20px;
  flex-shrink: 0;
}
.tabs button {
  background: none;
  border: none;
  color: var(--text2);
  padding: 10px 16px;
  cursor: pointer;
  font-size: 13px;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.tabs button.active { color: var(--text); border-bottom-color: var(--accent); }
.tabs button:hover { color: var(--text); }

.main { display: flex; flex: 1; overflow: hidden; }

.list {
  width: 400px;
  min-width: 300px;
  border-right: 1px solid var(--border);
  overflow-y: auto;
  flex-shrink: 0;
}
.list-item {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  cursor: pointer;
  transition: background 0.1s;
}
.list-item:hover { background: var(--bg2); }
.list-item.selected { background: var(--bg3); border-left: 3px solid var(--accent); padding-left: 13px; }
.list-item.unread .subject { font-weight: 600; }
.list-item .subject { font-size: 14px; margin-bottom: 4px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.list-item .meta { font-size: 12px; color: var(--text2); display: flex; gap: 8px; }
.list-item .meta .from { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.list-item .dot { display: inline-block; width: 8px; height: 8px; background: var(--accent); border-radius: 50%; margin-right: 6px; }
.list-empty { padding: 40px 20px; text-align: center; color: var(--text2); }
.list-empty .icon { font-size: 48px; margin-bottom: 12px; opacity: 0.5; }
.list-empty p { font-size: 14px; }

.detail { flex: 1; overflow-y: auto; display: flex; flex-direction: column; }
.detail-empty { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--text2); }
.detail-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
  background: var(--bg2);
  flex-shrink: 0;
}
.detail-header h2 { font-size: 18px; font-weight: 600; margin-bottom: 8px; }
.detail-header .field { font-size: 13px; color: var(--text2); margin-bottom: 2px; }
.detail-header .field span { color: var(--text); }
.detail-header .actions { margin-top: 10px; display: flex; gap: 6px; }
.detail-header .actions button {
  background: var(--bg3);
  color: var(--text2);
  border: 1px solid var(--border);
  padding: 4px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}
.detail-header .actions button:hover { color: var(--text); border-color: var(--text2); }
.detail-header .actions button.active { color: var(--accent); border-color: var(--accent); }

.detail-tabs {
  display: flex;
  border-bottom: 1px solid var(--border);
  background: var(--bg2);
  padding: 0 20px;
  flex-shrink: 0;
}
.detail-tabs button {
  background: none;
  border: none;
  color: var(--text2);
  padding: 8px 12px;
  cursor: pointer;
  font-size: 12px;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.detail-tabs button.active { color: var(--text); border-bottom-color: var(--accent); }

.detail-body { flex: 1; overflow: auto; position: relative; }
.detail-body iframe {
  width: 100%;
  height: 100%;
  border: none;
  background: #fff;
  position: absolute;
  top: 0; left: 0;
}
.detail-body pre {
  padding: 16px 20px;
  font-family: var(--mono);
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}
.detail-body .headers-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.detail-body .headers-table td {
  padding: 6px 12px;
  border-bottom: 1px solid var(--border);
  vertical-align: top;
}
.detail-body .headers-table td:first-child {
  color: var(--accent);
  font-family: var(--mono);
  font-size: 12px;
  white-space: nowrap;
  width: 1%;
}
.attachments-list { padding: 16px 20px; }
.attachments-list a {
  display: inline-block;
  padding: 8px 12px;
  background: var(--bg3);
  border: 1px solid var(--border);
  border-radius: 6px;
  margin: 4px;
  font-size: 13px;
}

.session-detail pre {
  padding: 16px 20px;
  font-family: var(--mono);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
}

@media (max-width: 768px) {
  .list { width: 100%; }
  .main { flex-direction: column; }
}
</style>
</head>
<body>
<header>
  <h1>smtp-dev-server</h1>
  <span class="badge" id="badge">0</span>
  <span class="spacer"></span>
  <button class="danger" onclick="deleteAll()">Clear All</button>
</header>
<div class="tabs">
  <button class="active" onclick="switchTab('messages', this)">Messages</button>
  <button onclick="switchTab('sessions', this)">Sessions</button>
</div>
<div class="main">
  <div class="list" id="list"></div>
  <div class="detail" id="detail">
    <div class="detail-empty">Select a message to view</div>
  </div>
</div>

<script>
let messages = [];
let sessions = [];
let currentTab = 'messages';
let selectedId = null;
let detailTab = 'html';

async function fetchMessages() {
  const res = await fetch('/api/messages');
  messages = await res.json();
  if (currentTab === 'messages') renderList();
  updateBadge();
}

async function fetchSessions() {
  const res = await fetch('/api/sessions');
  sessions = await res.json();
  if (currentTab === 'sessions') renderList();
}

function updateBadge() {
  const unread = messages.filter(m => !m.isRead).length;
  document.getElementById('badge').textContent = unread || messages.length;
  document.title = unread > 0 ? '(' + unread + ') smtp-dev-server' : 'smtp-dev-server';
}

function switchTab(tab, btn) {
  currentTab = tab;
  selectedId = null;
  document.querySelectorAll('.tabs button').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  renderList();
  document.getElementById('detail').innerHTML = '<div class="detail-empty">Select ' + (tab === 'messages' ? 'a message' : 'a session') + ' to view</div>';
  if (tab === 'sessions') fetchSessions();
}

function renderList() {
  const el = document.getElementById('list');
  if (currentTab === 'messages') {
    if (messages.length === 0) {
      el.innerHTML = '<div class="list-empty"><div class="icon">&#9993;</div><p>No messages yet.<br>Send an email to the SMTP server to see it here.</p></div>';
      return;
    }
    el.innerHTML = messages.map(m => {
      const date = new Date(m.receivedAt);
      const time = date.toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'});
      return '<div class="list-item' + (m.id === selectedId ? ' selected' : '') + (!m.isRead ? ' unread' : '') + '" onclick="selectMessage(\'' + m.id + '\')">'
        + '<div class="subject">' + (!m.isRead ? '<span class="dot"></span>' : '') + escHtml(m.subject || '(no subject)') + '</div>'
        + '<div class="meta"><span class="from">' + escHtml(m.from) + '</span><span>' + time + '</span></div>'
        + '</div>';
    }).join('');
  } else {
    if (sessions.length === 0) {
      el.innerHTML = '<div class="list-empty"><p>No sessions yet.</p></div>';
      return;
    }
    el.innerHTML = sessions.map(s => {
      const date = new Date(s.startedAt);
      const time = date.toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'});
      return '<div class="list-item' + (s.id === selectedId ? ' selected' : '') + '" onclick="selectSession(\'' + s.id + '\')">'
        + '<div class="subject">' + escHtml(s.clientAddr) + '</div>'
        + '<div class="meta"><span>' + s.messageCount + ' message(s)</span><span>' + time + '</span></div>'
        + '</div>';
    }).join('');
  }
}

async function selectMessage(id) {
  selectedId = id;
  renderList();
  await fetch('/api/messages/' + id + '/read', {method:'POST'});
  const msg = messages.find(m => m.id === id);
  if (msg) msg.isRead = true;
  updateBadge();
  renderList();
  showMessageDetail(id);
}

function showMessageDetail(id) {
  const msg = messages.find(m => m.id === id);
  if (!msg) return;
  const date = new Date(msg.receivedAt).toLocaleString();
  const det = document.getElementById('detail');
  det.innerHTML = '<div class="detail-header">'
    + '<h2>' + escHtml(msg.subject || '(no subject)') + '</h2>'
    + '<div class="field">From: <span>' + escHtml(msg.from) + '</span></div>'
    + '<div class="field">To: <span>' + escHtml((msg.to||[]).join(', ')) + '</span></div>'
    + '<div class="field">Received: <span>' + date + '</span> &middot; ' + formatSize(msg.size) + '</div>'
    + '<div class="actions">'
    + '<button onclick="setDetailTab(\'html\')" id="dt-html" class="active">HTML</button>'
    + '<button onclick="setDetailTab(\'text\')" id="dt-text">Text</button>'
    + '<button onclick="setDetailTab(\'headers\')" id="dt-headers">Headers</button>'
    + '<button onclick="setDetailTab(\'raw\')" id="dt-raw">Source</button>'
    + '<button onclick="setDetailTab(\'attachments\')" id="dt-attachments">Attachments</button>'
    + '</div>'
    + '</div>'
    + '<div class="detail-body" id="detail-body"></div>';
  detailTab = 'html';
  loadDetailTab(id);
}

async function setDetailTab(tab) {
  detailTab = tab;
  document.querySelectorAll('.detail-header .actions button').forEach(b => b.classList.remove('active'));
  const btn = document.getElementById('dt-' + tab);
  if (btn) btn.classList.add('active');
  loadDetailTab(selectedId);
}

async function loadDetailTab(id) {
  const body = document.getElementById('detail-body');
  if (!body) return;
  switch(detailTab) {
    case 'html':
      body.innerHTML = '<iframe sandbox="allow-same-origin" src="/api/messages/' + id + '/html"></iframe>';
      break;
    case 'text':
      const textRes = await fetch('/api/messages/' + id + '/text');
      body.innerHTML = '<pre>' + escHtml(await textRes.text()) + '</pre>';
      break;
    case 'headers':
      const hdrs = await (await fetch('/api/messages/' + id + '/headers')).json();
      body.innerHTML = '<table class="headers-table">' + (hdrs||[]).map(h =>
        '<tr><td>' + escHtml(h.name) + '</td><td>' + escHtml(h.value) + '</td></tr>'
      ).join('') + '</table>';
      break;
    case 'raw':
      const rawRes = await fetch('/api/messages/' + id + '/raw');
      body.innerHTML = '<pre>' + escHtml(await rawRes.text()) + '</pre>';
      break;
    case 'attachments':
      const atts = await (await fetch('/api/messages/' + id + '/attachments')).json();
      if (!atts || atts.length === 0) {
        body.innerHTML = '<div class="list-empty"><p>No attachments</p></div>';
      } else {
        body.innerHTML = '<div class="attachments-list">' + atts.map((a,i) =>
          '<a href="/api/messages/' + id + '/attachments/' + i + '" download>' + escHtml(a.filename || 'attachment-' + i) + ' (' + formatSize(a.size) + ')</a>'
        ).join('') + '</div>';
      }
      break;
  }
}

async function selectSession(id) {
  selectedId = id;
  renderList();
  const res = await fetch('/api/sessions/' + id);
  const sess = await res.json();
  const det = document.getElementById('detail');
  det.innerHTML = '<div class="detail-header">'
    + '<h2>Session: ' + escHtml(sess.clientAddr) + '</h2>'
    + '<div class="field">Started: <span>' + new Date(sess.startedAt).toLocaleString() + '</span></div>'
    + '<div class="field">Messages: <span>' + sess.messageCount + '</span></div>'
    + '</div>'
    + '<div class="detail-body"><pre class="session-detail">' + escHtml(sess.log || '(no log)') + '</pre></div>';
}

async function deleteAll() {
  if (!confirm('Delete all messages?')) return;
  await fetch('/api/messages', {method:'DELETE'});
  await fetchMessages();
  document.getElementById('detail').innerHTML = '<div class="detail-empty">Select a message to view</div>';
  selectedId = null;
}

function escHtml(s) {
  if (!s) return '';
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1048576) return (bytes/1024).toFixed(1) + ' KB';
  return (bytes/1048576).toFixed(1) + ' MB';
}

// SSE for live updates
const evtSource = new EventSource('/api/events');
evtSource.onmessage = function(e) {
  fetchMessages();
};

// Initial load
fetchMessages();
fetchSessions();
</script>
</body>
</html>`
