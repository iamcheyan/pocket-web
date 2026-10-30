package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"
)

type ViewOptions struct{ Private bool }
type BrowserEngine interface {
	Name() string
	Available() (bool, string)
	NewView(context.Context, string, ViewOptions) (BrowserView, error)
}
type BrowserView interface {
	Navigate(context.Context, string) error
	Back(context.Context) error
	Forward(context.Context) error
	Reload(context.Context) error
	Stop(context.Context) error
	Close() error
}

type ChromiumHeadlessAdapter struct{ Binary string }

func (e *ChromiumHeadlessAdapter) Name() string { return "ChromiumHeadlessAdapter" }
func (e *ChromiumHeadlessAdapter) binary() string {
	if e.Binary != "" {
		return e.Binary
	}
	return "chromium"
}
func (e *ChromiumHeadlessAdapter) Available() (bool, string) {
	if _, err := exec.LookPath(e.binary()); err != nil {
		return false, "Chromium executable unavailable"
	}
	return true, "headless development engine using Chromium CDP"
}

type chromiumView struct {
	mu                         sync.Mutex
	cmd                        *exec.Cmd
	profile, debugAddr, target string
	conn                       *websocket.Conn
	nextID                     int64
	pending                    map[int64]chan cdpResult
	closed                     bool
	waitDone                   chan struct{}
	processErr                 error
}
type cdpResult struct {
	result json.RawMessage
	err    error
}
type cdpEnvelope struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}
type cdpTarget struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func (e *ChromiumHeadlessAdapter) NewView(ctx context.Context, _ string, _ ViewOptions) (BrowserView, error) {
	if ok, why := e.Available(); !ok {
		return nil, errors.New(why)
	}
	p, err := os.MkdirTemp("", "pocket-web-profile-")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(e.binary(), "--headless=new", "--no-sandbox", "--disable-gpu", "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=0", "--user-data-dir="+p, "about:blank")
	if err = cmd.Start(); err != nil {
		_ = os.RemoveAll(p)
		return nil, err
	}
	v := &chromiumView{cmd: cmd, profile: p, pending: map[int64]chan cdpResult{}, waitDone: make(chan struct{})}
	go func() {
		v.processErr = cmd.Wait()
		close(v.waitDone)
		v.failPending(errors.New("Chromium process exited"))
	}()
	if err = v.initialize(ctx); err != nil {
		_ = v.Close()
		return nil, err
	}
	return v, nil
}
func (v *chromiumView) initialize(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	marker := filepath.Join(v.profile, "DevToolsActivePort")
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if b, err := os.ReadFile(marker); err == nil {
			lines := strings.Split(strings.TrimSpace(string(b)), "\n")
			if len(lines) >= 2 {
				v.debugAddr = net.JoinHostPort("127.0.0.1", strings.TrimSpace(lines[0]))
				break
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for Chromium debug endpoint: %w", ctx.Err())
		case <-v.waitDone:
			return fmt.Errorf("Chromium exited before debug endpoint: %w", v.processErr)
		case <-ticker.C:
		}
	}
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+v.debugAddr+"/json/list", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			var targets []cdpTarget
			de := json.NewDecoder(resp.Body).Decode(&targets)
			_ = resp.Body.Close()
			if de == nil {
				for _, t := range targets {
					if t.Type == "page" && t.WebSocketDebuggerURL != "" {
						v.target = t.ID
						return v.attach(ctx, t.WebSocketDebuggerURL)
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for Chromium page target: %w", ctx.Err())
		case <-v.waitDone:
			return fmt.Errorf("Chromium exited before page target: %w", v.processErr)
		case <-ticker.C:
		}
	}
}
func (v *chromiumView) attach(ctx context.Context, endpoint string) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("connecting to CDP: %w", err)
	}
	v.mu.Lock()
	v.conn = conn
	v.mu.Unlock()
	go v.readLoop()
	return nil
}
func (v *chromiumView) readLoop() {
	for {
		var msg cdpEnvelope
		if err := v.conn.ReadJSON(&msg); err != nil {
			v.failPending(fmt.Errorf("CDP connection closed: %w", err))
			return
		}
		if msg.ID == 0 {
			continue
		}
		v.mu.Lock()
		ch := v.pending[msg.ID]
		delete(v.pending, msg.ID)
		v.mu.Unlock()
		if ch != nil {
			if msg.Error != nil {
				ch <- cdpResult{err: errors.New(msg.Error.Message)}
			} else {
				ch <- cdpResult{result: msg.Result}
			}
		}
	}
}
func (v *chromiumView) failPending(err error) {
	v.mu.Lock()
	p := v.pending
	v.pending = map[int64]chan cdpResult{}
	v.mu.Unlock()
	for _, ch := range p {
		ch <- cdpResult{err: err}
	}
}
func (v *chromiumView) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v.mu.Lock()
	if v.closed || v.conn == nil {
		v.mu.Unlock()
		return nil, errors.New("Chromium view is closed")
	}
	v.nextID++
	id := v.nextID
	ch := make(chan cdpResult, 1)
	v.pending[id] = ch
	err := v.conn.WriteJSON(map[string]any{"id": id, "method": method, "params": params})
	if err != nil {
		delete(v.pending, id)
		v.mu.Unlock()
		return nil, err
	}
	v.mu.Unlock()
	select {
	case r := <-ch:
		return r.result, r.err
	case <-ctx.Done():
		v.mu.Lock()
		delete(v.pending, id)
		v.mu.Unlock()
		return nil, ctx.Err()
	}
}
func (v *chromiumView) Navigate(ctx context.Context, u string) error {
	_, err := v.call(ctx, "Page.navigate", map[string]string{"url": u})
	return err
}
func (v *chromiumView) history(ctx context.Context, d int) error {
	raw, err := v.call(ctx, "Page.getNavigationHistory", map[string]any{})
	if err != nil {
		return err
	}
	var h struct {
		Entries []struct {
			ID int64 `json:"id"`
		} `json:"entries"`
		Current int `json:"currentIndex"`
	}
	if err = json.Unmarshal(raw, &h); err != nil {
		return err
	}
	i := h.Current + d
	if i < 0 || i >= len(h.Entries) {
		return nil
	}
	_, err = v.call(ctx, "Page.navigateToHistoryEntry", map[string]int64{"entryId": h.Entries[i].ID})
	return err
}
func (v *chromiumView) Back(ctx context.Context) error    { return v.history(ctx, -1) }
func (v *chromiumView) Forward(ctx context.Context) error { return v.history(ctx, 1) }
func (v *chromiumView) Reload(ctx context.Context) error {
	_, err := v.call(ctx, "Page.reload", map[string]bool{"ignoreCache": false})
	return err
}
func (v *chromiumView) Stop(ctx context.Context) error {
	_, err := v.call(ctx, "Page.stopLoading", map[string]any{})
	return err
}
func (v *chromiumView) Screenshot(ctx context.Context) ([]byte, error) {
	raw, err := v.call(ctx, "Page.captureScreenshot", map[string]any{"format": "png", "captureBeyondViewport": false})
	if err != nil {
		return nil, err
	}
	var x struct {
		Data string `json:"data"`
	}
	if err = json.Unmarshal(raw, &x); err != nil {
		return nil, err
	}
	if x.Data == "" {
		return nil, errors.New("screenshot contained no data")
	}
	return base64.StdEncoding.DecodeString(x.Data)
}
func (v *chromiumView) CurrentPage(ctx context.Context) (string, string, error) {
	raw, err := v.call(ctx, "Runtime.evaluate", map[string]any{"expression": "({url:location.href,title:document.title})", "returnByValue": true})
	if err != nil {
		return "", "", err
	}
	var x struct {
		Result struct {
			Value struct {
				URL   string `json:"url"`
				Title string `json:"title"`
			} `json:"value"`
		} `json:"result"`
	}
	if err = json.Unmarshal(raw, &x); err != nil {
		return "", "", err
	}
	return x.Result.Value.URL, x.Result.Value.Title, nil
}
func (v *chromiumView) Close() error {
	v.mu.Lock()
	if v.closed {
		v.mu.Unlock()
		return nil
	}
	v.closed = true
	c := v.conn
	v.mu.Unlock()
	if c != nil {
		_ = c.Close()
	}
	v.failPending(errors.New("Chromium view closed"))
	if v.cmd.Process != nil {
		_ = v.cmd.Process.Kill()
	}
	select {
	case <-v.waitDone:
	case <-time.After(5 * time.Second):
	}
	return os.RemoveAll(v.profile)
}

type oauthState struct {
	verifier, provider string
	expires            time.Time
}
type Server struct {
	db             *sql.DB
	mux            *http.ServeMux
	engine         BrowserEngine
	downloadClient DownloadCenterClient
	views          map[string]BrowserView
	mu             sync.Mutex
	private        bool
	oauth          map[string]oauthState
	key            []byte
	dataDir        string
	closedTabs     []map[string]any
}

func NewServer(db *sql.DB, engine BrowserEngine, dir string) *Server {
	server := &Server{db: db, engine: engine, downloadClient: NewDownloadCenterFromEnv(), dataDir: dir, views: map[string]BrowserView{}, oauth: map[string]oauthState{}}
	server.mux = http.NewServeMux()
	server.routes()
	if key, err := LoadCredentialKey(dir); err == nil {
		server.key = key
	}
	return server
}
func OpenServer(dir string, engine BrowserEngine) (*Server, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "pocketweb.db"))
	if err != nil {
		return nil, err
	}
	server := NewServer(db, engine, dir)
	if err = server.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	server.key, err = LoadCredentialKey(dir)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if err = seedDemoData(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return server, nil
}
func (s *Server) Close() error {
	s.mu.Lock()
	for _, view := range s.views {
		_ = view.Close()
	}
	s.mu.Unlock()
	return s.db.Close()
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	localCORS(http.HandlerFunc(s.mux.ServeHTTP)).ServeHTTP(w, r)
}
func (s *Server) migrate() error {
	_, e := s.db.Exec(`PRAGMA journal_mode=WAL;CREATE TABLE IF NOT EXISTS tabs(id TEXT PRIMARY KEY,title TEXT NOT NULL DEFAULT '',url TEXT NOT NULL DEFAULT '',position INTEGER NOT NULL,active INTEGER NOT NULL DEFAULT 0,private INTEGER NOT NULL DEFAULT 0);CREATE TABLE IF NOT EXISTS sessions(id INTEGER PRIMARY KEY,created_at TEXT NOT NULL,tabs_json TEXT NOT NULL);CREATE TABLE IF NOT EXISTS bookmarks(id TEXT PRIMARY KEY,title TEXT NOT NULL,url TEXT NOT NULL,folder_id TEXT,created_at TEXT NOT NULL);CREATE TABLE IF NOT EXISTS bookmark_folders(id TEXT PRIMARY KEY,name TEXT NOT NULL,parent_id TEXT);CREATE TABLE IF NOT EXISTS reading_list(id TEXT PRIMARY KEY,title TEXT NOT NULL,url TEXT NOT NULL,read INTEGER NOT NULL DEFAULT 0,offline_metadata TEXT NOT NULL DEFAULT '',created_at TEXT NOT NULL);CREATE TABLE IF NOT EXISTS history(id INTEGER PRIMARY KEY,url TEXT UNIQUE NOT NULL,title TEXT NOT NULL,visit_count INTEGER NOT NULL,last_visit TEXT NOT NULL);CREATE TABLE IF NOT EXISTS downloads(id TEXT PRIMARY KEY,title TEXT NOT NULL,url TEXT NOT NULL,status TEXT NOT NULL,category TEXT NOT NULL DEFAULT '');CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY,value TEXT NOT NULL);CREATE TABLE IF NOT EXISTS accounts(id TEXT PRIMARY KEY,provider TEXT NOT NULL,email TEXT NOT NULL,created_at TEXT NOT NULL);CREATE TABLE IF NOT EXISTS oauth_tokens(account_id TEXT PRIMARY KEY,ciphertext TEXT NOT NULL)`)
	return e
}
func (s *Server) routes() {
	s.mux.HandleFunc("/api/health", s.health)
	s.mux.HandleFunc("/api/tabs", s.tabs)
	s.mux.HandleFunc("/api/tabs/", s.tab)
	s.mux.HandleFunc("/api/sessions/", s.sessionRoutes)
	s.mux.HandleFunc("/api/bookmarks", s.bookmarks)
	s.mux.HandleFunc("/api/bookmarks/", s.bookmark)
	s.mux.HandleFunc("/api/bookmark-folders", s.folders)
	s.mux.HandleFunc("/api/bookmarks/import", s.importBookmarks)
	s.mux.HandleFunc("/api/bookmarks/export", s.exportBookmarks)
	s.mux.HandleFunc("/api/reading-list", s.reading)
	s.mux.HandleFunc("/api/reading-list/", s.readingItem)
	s.mux.HandleFunc("/api/history", s.history)
	s.mux.HandleFunc("/api/history/", s.historyItem)
	s.mux.HandleFunc("/api/downloads", s.downloads)
	s.mux.HandleFunc("/api/settings", s.settings)
	s.mux.HandleFunc("/api/privacy/", s.privacy)
	s.mux.HandleFunc("/api/accounts", s.accounts)
	s.mux.HandleFunc("/api/oauth/", s.oauthRoute)
	s.mux.HandleFunc("/api/auth/mock", s.mockAuth)
	s.mux.HandleFunc("/api/apps/actions", s.actions)
	s.mux.Handle("/", s.staticHandler(os.Getenv("POCKET_WEB_UI")))
}
func jsonOut(w http.ResponseWriter, n int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(n)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, n int, c, m string) {
	jsonOut(w, n, map[string]any{"error": map[string]string{"code": c, "message": m}})
}
func decode(r *http.Request, v any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func newID() string { b := make([]byte, 16); _, _ = rand.Read(b); return fmt.Sprintf("%x", b) }
func boolJSON(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ok, why := s.engine.Available()
	da, dr := s.downloadAvailability()
	jsonOut(w, 200, map[string]any{"status": "ok", "engine": map[string]any{"name": s.engine.Name(), "available": ok, "reason": why}, "downloads": map[string]any{"available": da, "reason": dr}, "mockMode": true})
}
func (s *Server) downloadAvailability() (bool, string) {
	return s.downloadClient.Available(context.Background())
}
func (s *Server) tabs(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.mu.Lock()
		privateMode := s.private
		s.mu.Unlock()
		query := `SELECT id,title,url,active,private FROM tabs ORDER BY position`
		if privateMode {
			query = `SELECT id,title,url,active,private FROM tabs WHERE private=0 ORDER BY position`
		}
		rows, err := s.db.Query(query)
		if err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, title, url string
			var active, private int
			if err = rows.Scan(&id, &title, &url, &active, &private); err != nil {
				fail(w, 500, "db_error", err.Error())
				return
			}
			items = append(items, map[string]any{"id": id, "title": title, "url": url, "active": active == 1, "private": private == 1, "loading": false})
		}
		if err = rows.Err(); err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		jsonOut(w, 200, items)
		return
	}
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed", "GET or POST required")
		return
	}
	var input struct {
		URL     string `json:"url"`
		Private bool   `json:"private"`
	}
	if decode(r, &input) != nil {
		fail(w, 400, "invalid_json", "invalid tab")
		return
	}
	if input.URL == "" {
		input.URL = "about:blank"
	}
	id := newID()
	s.mu.Lock()
	private := s.private || input.Private
	s.mu.Unlock()
	if !private {
		_, _ = s.db.Exec(`UPDATE tabs SET active=0`)
	}
	view, err := s.engine.NewView(r.Context(), id, ViewOptions{Private: private})
	if err != nil {
		fail(w, 503, "engine_unavailable", err.Error())
		return
	}
	s.putView(id, view)
	if !private {
		if _, err = s.db.Exec(`INSERT INTO tabs(id,title,url,position,active,private) VALUES(?,'',?,(SELECT COALESCE(MAX(position)+1,0) FROM tabs),1,0)`, id, input.URL); err != nil {
			_ = view.Close()
			s.mu.Lock()
			delete(s.views, id)
			s.mu.Unlock()
			fail(w, 500, "db_error", err.Error())
			return
		}
	}
	var navigationErr error
	if input.URL != "about:blank" {
		navigationErr = view.Navigate(r.Context(), input.URL)
		if navigationErr == nil {
			title := input.URL
			if page, ok := view.(interface {
				CurrentPage(context.Context) (string, string, error)
			}); ok {
				if actual, titleFromPage, pageErr := page.CurrentPage(r.Context()); pageErr == nil {
					if actual != "" {
						input.URL = actual
					}
					if titleFromPage != "" {
						title = titleFromPage
					}
				}
			}
			if !private {
				_, _ = s.db.Exec(`UPDATE tabs SET title=?,url=? WHERE id=?`, title, input.URL, id)
				_, _ = s.db.Exec(`INSERT INTO history(url,title,visit_count,last_visit) VALUES(?,?,1,?) ON CONFLICT(url) DO UPDATE SET title=excluded.title,visit_count=history.visit_count+1,last_visit=excluded.last_visit`, input.URL, title, time.Now().UTC().Format(time.RFC3339Nano))
			}
		}
	}
	jsonOut(w, 201, map[string]any{"id": id, "title": "", "url": input.URL, "active": true, "private": private, "loading": false, "engineError": errText(navigationErr)})
}
func (s *Server) tab(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/api/tabs/")
	if p == "restore-closed" && r.Method == "POST" {
		s.mu.Lock()
		if len(s.closedTabs) == 0 {
			s.mu.Unlock()
			fail(w, 404, "not_found", "no recently closed tab")
			return
		}
		last := s.closedTabs[len(s.closedTabs)-1]
		s.closedTabs = s.closedTabs[:len(s.closedTabs)-1]
		s.mu.Unlock()
		restoredURL, _ := last["url"].(string)
		if restoredURL == "" {
			fail(w, 404, "not_found", "closed tab has no URL")
			return
		}
		r.Body = io.NopCloser(strings.NewReader(`{"url":` + quote(restoredURL) + `}`))
		s.tabs(w, r)
		return
	}
	if p == "order" && r.Method == "PUT" {
		var x struct {
			IDs []string `json:"ids"`
		}
		if decode(r, &x) != nil {
			fail(w, 400, "invalid_json", "ids required")
			return
		}
		for i, id := range x.IDs {
			if _, err := s.db.Exec(`UPDATE tabs SET position=? WHERE id=?`, i, id); err != nil {
				fail(w, 500, "db_error", err.Error())
				return
			}
		}
		if err := s.saveSession(); err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	parts := strings.Split(p, "/")
	id := parts[0]
	if len(parts) > 1 {
		switch parts[1] {
		case "navigate":
			var x struct {
				URL string `json:"url"`
			}
			if decode(r, &x) != nil {
				fail(w, 400, "invalid_json", "url required")
				return
			}
			v := s.viewFor(id)
			if v == nil {
				fail(w, 503, "engine_unavailable", "tab view unavailable")
				return
			}
			if e := v.Navigate(r.Context(), x.URL); e != nil {
				fail(w, 503, "engine_unavailable", e.Error())
				return
			}
			_, _ = s.db.Exec(`UPDATE tabs SET url=?,title=? WHERE id=? AND private=0`, x.URL, x.URL, id)
			var private int
			_ = s.db.QueryRow(`SELECT private FROM tabs WHERE id=?`, id).Scan(&private)
			if private == 0 {
				_, _ = s.db.Exec(`INSERT INTO history(url,title,visit_count,last_visit) VALUES(?,?,1,?) ON CONFLICT(url) DO UPDATE SET title=excluded.title,visit_count=history.visit_count+1,last_visit=excluded.last_visit`, x.URL, x.URL, time.Now().UTC().Format(time.RFC3339Nano))
			}
			jsonOut(w, 200, map[string]any{"id": id, "title": x.URL, "url": x.URL})
			return
		case "back", "forward", "reload":
			v := s.viewFor(id)
			if v == nil {
				fail(w, 503, "engine_unavailable", "tab view unavailable")
				return
			}
			var e error
			if parts[1] == "back" {
				e = v.Back(r.Context())
			} else if parts[1] == "forward" {
				e = v.Forward(r.Context())
			} else {
				e = v.Reload(r.Context())
			}
			if e != nil {
				fail(w, 503, "engine_unavailable", e.Error())
				return
			}
			jsonOut(w, 200, map[string]bool{"ok": true})
			return
		case "screenshot":
			if r.Method != "GET" {
				fail(w, 405, "method_not_allowed", "GET required")
				return
			}
			v := s.viewFor(id)
			if v == nil {
				fail(w, 503, "engine_unavailable", "tab view unavailable")
				return
			}
			shot, ok := v.(interface {
				Screenshot(context.Context) ([]byte, error)
			})
			if !ok {
				fail(w, 503, "engine_unavailable", "engine does not support screenshots")
				return
			}
			png, e := shot.Screenshot(r.Context())
			if e != nil {
				fail(w, 503, "engine_unavailable", e.Error())
				return
			}
			if len(png) == 0 {
				fail(w, 503, "engine_unavailable", "engine returned an empty screenshot")
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(200)
			_, _ = w.Write(png)
			return
		case "duplicate":
			duplicateURL := ""
			_ = s.db.QueryRow(`SELECT url FROM tabs WHERE id=?`, id).Scan(&duplicateURL)
			if duplicateURL == "" {
				if view := s.viewFor(id); view != nil {
					if page, ok := view.(interface {
						CurrentPage(context.Context) (string, string, error)
					}); ok {
						duplicateURL, _, _ = page.CurrentPage(r.Context())
					}
				}
			}
			if duplicateURL == "" {
				fail(w, 404, "not_found", "tab not found")
				return
			}
			r.Body = io.NopCloser(strings.NewReader(`{"url":` + quote(duplicateURL) + `}`))
			r.URL.Path = "/api/tabs"
			s.tabs(w, r)
			return
		}
	}
	if r.Method == "DELETE" {
		var t, u string
		var pr int
		_ = s.db.QueryRow(`SELECT title,url,private FROM tabs WHERE id=?`, id).Scan(&t, &u, &pr)
		if pr == 0 && u != "" {
			s.mu.Lock()
			s.closedTabs = append(s.closedTabs, map[string]any{"title": t, "url": u})
			if len(s.closedTabs) > 10 {
				s.closedTabs = s.closedTabs[1:]
			}
			s.mu.Unlock()
		}
		s.mu.Lock()
		view := s.views[id]
		delete(s.views, id)
		s.mu.Unlock()
		if view != nil {
			_ = view.Close()
		}
		if _, err := s.db.Exec(`DELETE FROM tabs WHERE id=?`, id); err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		if err := s.saveSession(); err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	if r.Method == "GET" {
		var t, u string
		var a, pr int
		e := s.db.QueryRow(`SELECT title,url,active,private FROM tabs WHERE id=?`, id).Scan(&t, &u, &a, &pr)
		if e != nil {
			fail(w, 404, "not_found", "tab not found")
			return
		}
		jsonOut(w, 200, map[string]any{"id": id, "title": t, "url": u, "active": a == 1, "private": pr == 1})
		return
	}
	fail(w, 405, "method_not_allowed", "method not allowed")
}
func errText(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}
func quote(s string) string { b, _ := json.Marshal(s); return string(b) }
func main() {
	d := os.Getenv("POCKET_WEB_DATA")
	if d == "" {
		c, _ := os.UserConfigDir()
		d = filepath.Join(c, "pocket-web")
	}
	s, e := OpenServer(d, &ChromiumHeadlessAdapter{Binary: os.Getenv("CHROMIUM_BIN")})
	if e != nil {
		log.Fatal(e)
	}
	defer s.Close()
	addr := os.Getenv("POCKET_WEB_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8088"
	}
	log.Printf("webd listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, s))
}
