package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/skip2/go-qrcode"

	serverapp "github.com/temic/go-music/internal/app"
	"github.com/temic/go-music/internal/config"
	"github.com/temic/go-music/pkg/addresses"
	"github.com/temic/go-music/pkg/connectqr"
)

const (
	configFileName = "config.yaml"
	panelAddr      = "127.0.0.1:8099"
)

type panel struct {
	mu         sync.Mutex
	configPath string
	cfg        config.Config
	server     *serverapp.Server
	cancel     context.CancelFunc
	logs       []string
	logger     zerolog.Logger
}

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	configPath := resolveConfigPath()
	cfg := loadOrDefault(configPath)

	p := &panel{
		configPath: configPath,
		cfg:        cfg,
		logs:       make([]string, 0, 200),
	}
	p.logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleIndex)
	mux.HandleFunc("/api/status", p.handleStatus)
	mux.HandleFunc("/api/start", p.handleStart)
	mux.HandleFunc("/api/stop", p.handleStop)
	mux.HandleFunc("/api/save", p.handleSave)
	mux.HandleFunc("/api/logs", p.handleLogs)
	mux.HandleFunc("/api/qrcode", p.handleQRCode)

	listener, err := net.Listen("tcp", panelAddr)
	if err != nil {
		if isAddrInUse(err) {
			url := "http://" + panelAddr
			fmt.Printf("Панель уже запущена: %s\n", url)
			fmt.Println("Открываю в браузере. Закройте предыдущее окно GoMusic, если хотите перезапустить.")
			_ = openBrowser(url)
			return
		}
		log.Fatal().Err(err).Msg("failed to start control panel")
	}

	url := "http://" + panelAddr
	p.appendLog("Панель управления: " + url)
	_ = openBrowser(url)

	fmt.Printf("Go Music control panel: %s\n", url)
	fmt.Println("Закройте это окно, чтобы выйти (сервер музыки тоже остановится).")

	if err := http.Serve(listener, mux); err != nil {
		log.Fatal().Err(err).Msg("control panel failed")
	}
}

func (p *panel) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageTemplate.Execute(w, nil)
}

func (p *panel) handleStatus(w http.ResponseWriter, _ *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()

	running := p.server != nil && p.server.IsRunning()
	tracks := 0
	if p.server != nil {
		tracks = p.server.TrackCount()
	}

	writeJSON(w, map[string]any{
		"running":     running,
		"music_path":  p.cfg.MusicPath,
		"host":        p.cfg.Host,
		"port":        p.cfg.Port,
		"token":       p.cfg.Token,
		"listen_addr": p.cfg.Addr(),
		"endpoints":   enrichEndpoints(p.cfg),
		"tracks":      tracks,
	})
}

func (p *panel) handleLogs(w http.ResponseWriter, _ *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	writeJSON(w, map[string]any{"logs": p.logs})
}

func (p *panel) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		MusicPath string `json:"music_path"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		Token     string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	cfg := p.mergePanelConfig(body.MusicPath, body.Host, body.Port, body.Token)

	if err := config.Save(p.configPath, cfg); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	p.mu.Lock()
	p.cfg = cfg
	p.mu.Unlock()
	p.appendLog("Настройки сохранены")

	writeJSON(w, map[string]any{"ok": true})
}

func (p *panel) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		MusicPath string `json:"music_path"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		Token     string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	cfg := p.mergePanelConfig(body.MusicPath, body.Host, body.Port, body.Token)

	if err := config.Save(p.configPath, cfg); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	p.mu.Lock()
	if p.server != nil && p.server.IsRunning() {
		p.mu.Unlock()
		writeJSONError(w, http.StatusConflict, "server is already running")
		return
	}
	p.cfg = cfg
	p.mu.Unlock()

	p.appendLog("Сканирование библиотеки…")

	logger := p.logger
	srv := serverapp.New(cfg, logger)
	ctx, cancel := context.WithCancel(context.Background())

	if err := srv.Start(ctx); err != nil {
		cancel()
		p.appendLog("Ошибка: " + err.Error())
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	p.mu.Lock()
	p.server = srv
	p.cancel = cancel
	p.mu.Unlock()

	eps := enrichEndpoints(cfg)
	p.appendLog(fmt.Sprintf("Сервер запущен (%d треков), слушает %s", srv.TrackCount(), cfg.Addr()))
	for _, ep := range eps {
		p.appendLog(fmt.Sprintf("  → http://%s (%s)", ep["addr"], ep["label"]))
	}
	writeJSON(w, map[string]any{
		"ok":          true,
		"tracks":      srv.TrackCount(),
		"listen_addr": cfg.Addr(),
		"endpoints":   eps,
	})
}

func (p *panel) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p.mu.Lock()
	srv := p.server
	cancel := p.cancel
	p.server = nil
	p.cancel = nil
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if srv != nil {
		if err := srv.Stop(); err != nil {
			p.appendLog("Ошибка остановки: " + err.Error())
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	p.appendLog("Сервер остановлен")
	writeJSON(w, map[string]any{"ok": true})
}

func (p *panel) handleQRCode(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("text")
	if text == "" || len(text) > 2048 {
		http.Error(w, "invalid text", http.StatusBadRequest)
		return
	}

	png, err := qrcode.Encode(text, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "failed to generate qr", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func enrichEndpoints(cfg config.Config) []map[string]string {
	endpoints := addresses.ForPort(cfg.Port)
	out := make([]map[string]string, 0, len(endpoints))
	for _, ep := range endpoints {
		serverURL := "http://" + ep.Addr
		out = append(out, map[string]string{
			"label":   ep.Label,
			"addr":    ep.Addr,
			"hint":    ep.Hint,
			"qr_text": connectqr.Encode(serverURL, cfg.Token),
		})
	}
	return out
}

func (p *panel) appendLog(msg string) {
	line := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	p.mu.Lock()
	p.logs = append(p.logs, line)
	if len(p.logs) > 200 {
		p.logs = p.logs[len(p.logs)-200:]
	}
	p.mu.Unlock()
	p.logger.Info().Msg(msg)
}

func resolveConfigPath() string {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), configFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		cwdCandidate := configFileName
		if _, err := os.Stat(cwdCandidate); err == nil {
			return cwdCandidate
		}
		return candidate
	}
	return configFileName
}

func loadOrDefault(path string) config.Config {
	cfg, err := config.Load(path)
	if err != nil {
		return config.Normalize(config.Default())
	}
	return cfg
}

func (p *panel) mergePanelConfig(musicPath, host string, port int, token string) config.Config {
	cfg := p.cfg
	if loaded, err := config.Load(p.configPath); err == nil {
		cfg = loaded
	}

	cfg.MusicPath = musicPath
	cfg.Host = host
	cfg.Port = port
	cfg.Token = token

	return config.Normalize(cfg)
}

func openBrowser(url string) error {
	return exec.Command("cmd", "/c", "start", "", url).Start()
}

func isAddrInUse(err error) bool {
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return false
	}
	return strings.Contains(strings.ToLower(opErr.Err.Error()), "bind")
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":   true,
		"message": message,
	})
}

var pageTemplate = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>Go Music Server</title>
  <style>
    :root {
      --bg: #0f1115;
      --card: #171a21;
      --border: #2a3040;
      --text: #e8ecf4;
      --muted: #9aa3b5;
      --accent: #5b8cff;
      --ok: #3ecf8e;
      --danger: #ff6b6b;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: Segoe UI, system-ui, sans-serif;
      background: radial-gradient(circle at top, #1a2235, var(--bg));
      color: var(--text);
      min-height: 100vh;
      padding: 32px 16px;
    }
    .wrap { max-width: 720px; margin: 0 auto; }
    h1 { margin: 0 0 8px; font-size: 28px; }
    .sub { color: var(--muted); margin-bottom: 24px; }
    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 20px;
      margin-bottom: 16px;
    }
    label { display:block; color: var(--muted); font-size: 13px; margin-bottom: 6px; }
    input:not([type="radio"]) {
      width: 100%;
      background: #0d1017;
      border: 1px solid var(--border);
      color: var(--text);
      border-radius: 10px;
      padding: 12px 14px;
      font-size: 15px;
      margin-bottom: 14px;
    }
    input:not([type="radio"]):focus { outline: none; border-color: var(--accent); }
    .row { display:grid; grid-template-columns: 1fr 140px; gap: 12px; }
    .buttons { display:flex; gap: 10px; flex-wrap: wrap; }
    button {
      border: 0;
      border-radius: 10px;
      padding: 12px 18px;
      font-size: 15px;
      font-weight: 600;
      cursor: pointer;
    }
    .start { background: var(--accent); color: white; }
    .stop { background: #2a3142; color: var(--text); }
    .save { background: #223046; color: var(--text); }
    button:disabled { opacity: 0.45; cursor: not-allowed; }
    .status-pill {
      display:inline-flex; align-items:center; gap:8px;
      padding: 6px 12px; border-radius: 999px;
      background: #121722; border: 1px solid var(--border);
      font-size: 14px;
    }
    .dot { width:8px; height:8px; border-radius:50%; background: var(--muted); }
    .dot.on { background: var(--ok); box-shadow: 0 0 8px var(--ok); }
    .meta { color: var(--muted); margin-top: 10px; line-height: 1.6; }
    .addr-list { margin-top: 8px; }
    .addr-item { margin-top: 8px; }
    .addr-item a { color: var(--accent); text-decoration: none; font-weight: 600; }
    .addr-item a:hover { text-decoration: underline; }
    .addr-hint { font-size: 12px; color: var(--muted); margin-top: 2px; }
    .field-hint { font-size: 12px; color: var(--muted); margin: -8px 0 14px; }
    .qr-grid { margin-top: 12px; }
    .qr-options {
      display: flex;
      flex-direction: row;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 12px;
    }
    .qr-option {
      flex: 1 1 180px;
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 10px 12px;
      background: #0d1017;
      border: 1px solid var(--border);
      border-radius: 10px;
      cursor: pointer;
    }
    .qr-option input[type="radio"] {
      width: auto;
      margin: 0;
      padding: 0;
      border: none;
      background: transparent;
      flex-shrink: 0;
      accent-color: var(--accent);
    }
    .qr-option:has(input:checked) { border-color: var(--accent); background: #111827; }
    .qr-option .qr-title { font-size: 14px; margin-bottom: 2px; }
    .qr-option .addr-hint { font-size: 11px; line-height: 1.35; }
    .qr-item {
      display: grid;
      grid-template-columns: 180px 1fr;
      gap: 16px;
      align-items: center;
      padding: 14px;
      margin-top: 14px;
      background: #0d1017;
      border: 1px solid var(--border);
      border-radius: 12px;
    }
    .qr-item img {
      width: 168px;
      height: 168px;
      border-radius: 8px;
      background: white;
      padding: 6px;
    }
    .qr-title { font-weight: 600; color: var(--text); margin-bottom: 4px; }
    .qr-url { color: var(--accent); font-size: 14px; word-break: break-all; }
    .qr-note { font-size: 12px; color: var(--muted); margin-top: 8px; line-height: 1.5; }
    @media (max-width: 520px) {
      .qr-options { flex-direction: column; }
      .qr-option { flex: 1 1 auto; }
      .qr-item { grid-template-columns: 1fr; justify-items: center; text-align: center; }
    }
    #logs {
      height: 220px;
      overflow:auto;
      background: #0b0e14;
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 12px;
      font-family: Consolas, monospace;
      font-size: 12px;
      white-space: pre-wrap;
      color: #c9d2e3;
    }
    .error { color: var(--danger); margin-top: 8px; min-height: 20px; }
  </style>
</head>
<body>
  <div class="wrap">
    <h1>Go Music Server</h1>
    <div class="sub">Панель управления локальным музыкальным сервером</div>

    <div class="card">
      <div class="status-pill"><span class="dot" id="dot"></span><span id="statusText">Проверка…</span></div>
      <div class="meta">
        <div id="addrLine">Подключение: —</div>
        <div id="tracksLine">Треков: —</div>
      </div>
    </div>

    <div class="card" id="qrCard" hidden>
      <label>Подключение по QR</label>
      <div class="qr-note">Выберите сеть и отсканируйте QR в iOS-приложении.</div>
      <div class="qr-options" id="qrOptions"></div>
      <div class="qr-grid" id="qrDisplay"></div>
    </div>

    <div class="card">
      <label>Папка музыки</label>
      <input id="music_path" placeholder="D:\Music"/>

      <label>Bearer Token</label>
      <input id="token" type="password" placeholder="token"/>

      <div class="row">
        <div>
          <label>Host</label>
          <input id="host" placeholder="0.0.0.0"/>
          <div class="field-hint">Оставьте 0.0.0.0 — сервер будет доступен по Wi‑Fi и VPN</div>
        </div>
        <div>
          <label>Port</label>
          <input id="port" placeholder="8080"/>
        </div>
      </div>

      <div class="buttons">
        <button class="start" id="startBtn" onclick="startServer()">Запустить</button>
        <button class="stop" id="stopBtn" onclick="stopServer()" disabled>Остановить</button>
        <button class="save" onclick="saveSettings()">Сохранить</button>
      </div>
      <div class="error" id="error"></div>
    </div>

    <div class="card">
      <label>Лог</label>
      <div id="logs"></div>
    </div>
  </div>

<script>
async function api(path, opts) {
  const res = await fetch(path, opts);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.message || ('HTTP ' + res.status));
  return data;
}

function values() {
  return {
    music_path: document.getElementById('music_path').value.trim(),
    host: document.getElementById('host').value.trim(),
    port: parseInt(document.getElementById('port').value, 10) || 0,
    token: document.getElementById('token').value.trim(),
  };
}

function setError(msg) {
  document.getElementById('error').textContent = msg || '';
}

function renderAddresses(s) {
  const el = document.getElementById('addrLine');
  if (!s.running) {
    el.innerHTML = 'Подключение: —';
    return;
  }

  const listen = s.listen_addr || (s.host + ':' + s.port);
  let html = '<div>Слушает: <code>' + listen + '</code> (все интерфейсы)</div>';
  const eps = s.endpoints || [];

  if (eps.length === 0) {
    html += '<div class="addr-list">Подключайтесь: <a href="http://127.0.0.1:' + s.port + '">http://127.0.0.1:' + s.port + '</a></div>';
  } else {
    html += '<div class="addr-list">Подключайтесь с телефона:</div>';
    html += eps.map(ep =>
      '<div class="addr-item"><a href="http://' + ep.addr + '">http://' + ep.addr + '</a>' +
      '<div class="addr-hint">' + ep.label + (ep.hint ? ' · ' + ep.hint : '') + '</div></div>'
    ).join('');
  }

  el.innerHTML = html;
  renderQR(s);
}

function renderQR(s) {
  const card = document.getElementById('qrCard');
  const options = document.getElementById('qrOptions');
  const display = document.getElementById('qrDisplay');
  if (!s.running) {
    card.hidden = true;
    options.innerHTML = '';
    display.innerHTML = '';
    qrEndpoints = [];
    return;
  }

  const eps = s.endpoints || [];
  const items = eps.length > 0 ? eps : [{
    label: 'Локально',
    addr: '127.0.0.1:' + s.port,
    hint: 'На этом ПК',
    qr_text: JSON.stringify({ server: 'http://127.0.0.1:' + s.port, token: s.token || '' }),
  }];

  const prev = qrSelectedAddr;
  qrEndpoints = items;
  if (!prev || !items.some(ep => ep.addr === prev)) {
    qrSelectedAddr = items[0].addr;
  }

  const addrKey = items.map(ep => ep.addr).join('|');
  const qrKey = items.map(ep => ep.qr_text).join('|');
  if (addrKey !== qrAddrKey) {
    qrAddrKey = addrKey;
    options.innerHTML = items.map((ep, i) =>
      '<label class="qr-option">' +
        '<input type="radio" name="qrEndpoint" value="' + i + '"' +
        (ep.addr === qrSelectedAddr ? ' checked' : '') +
        ' onchange="selectQR(' + i + ')"/>' +
        '<span>' +
          '<div class="qr-title">' + ep.label + '</div>' +
          '<div class="addr-hint">' + (ep.hint || '') + ' · ' + ep.addr + '</div>' +
        '</span>' +
      '</label>'
    ).join('');
  }

  if (qrKey !== qrPayloadKey) {
    qrPayloadKey = qrKey;
    renderQRDisplay();
  }

  card.hidden = false;
}

let qrEndpoints = [];
let qrSelectedAddr = null;
let qrAddrKey = '';
let qrPayloadKey = '';

function selectQR(index) {
  const ep = qrEndpoints[index];
  if (!ep) return;
  qrSelectedAddr = ep.addr;
  qrPayloadKey = '';
  renderQRDisplay();
}

function renderQRDisplay() {
  const display = document.getElementById('qrDisplay');
  const ep = qrEndpoints.find(e => e.addr === qrSelectedAddr) || qrEndpoints[0];
  if (!ep) {
    display.innerHTML = '';
    return;
  }

  const url = 'http://' + ep.addr;
  const src = '/api/qrcode?text=' + encodeURIComponent(ep.qr_text);
  display.innerHTML =
    '<div class="qr-item">' +
      '<img src="' + src + '" alt="QR ' + url + '" width="168" height="168"/>' +
      '<div>' +
        '<div class="qr-title">' + ep.label + '</div>' +
        '<div class="qr-url">' + url + '</div>' +
        '<div class="addr-hint">' + (ep.hint || '') + '</div>' +
      '</div>' +
    '</div>';
}

function setRunning(running) {
  document.getElementById('dot').className = 'dot' + (running ? ' on' : '');
  document.getElementById('statusText').textContent = running ? 'Работает' : 'Остановлен';
  document.getElementById('startBtn').disabled = running;
  document.getElementById('stopBtn').disabled = !running;
  ['music_path','token','host','port'].forEach(id => {
    document.getElementById(id).disabled = running;
  });
}

async function refresh() {
  try {
    const s = await api('/api/status');
    document.getElementById('music_path').value = s.music_path || '';
    document.getElementById('host').value = s.host || '0.0.0.0';
    document.getElementById('port').value = s.port || 8080;
    document.getElementById('token').value = s.token || '';
    renderAddresses(s);
    document.getElementById('tracksLine').textContent = s.running ? ('Треков: ' + s.tracks) : 'Треков: —';
    setRunning(!!s.running);

    const logs = await api('/api/logs');
    document.getElementById('logs').textContent = (logs.logs || []).join('\n');
    const box = document.getElementById('logs');
    box.scrollTop = box.scrollHeight;
  } catch (e) {
    setError(e.message);
  }
}

async function saveSettings() {
  setError('');
  try {
    await api('/api/save', {
      method: 'POST',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify(values()),
    });
    await refresh();
  } catch (e) {
    setError(e.message);
  }
}

async function startServer() {
  setError('');
  document.getElementById('startBtn').disabled = true;
  try {
    await api('/api/start', {
      method: 'POST',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify(values()),
    });
    await refresh();
  } catch (e) {
    setError(e.message);
    document.getElementById('startBtn').disabled = false;
  }
}

async function stopServer() {
  setError('');
  document.getElementById('stopBtn').disabled = true;
  try {
    await api('/api/stop', { method: 'POST' });
    await refresh();
  } catch (e) {
    setError(e.message);
  }
}

refresh();
setInterval(refresh, 2000);
</script>
</body>
</html>
`))
