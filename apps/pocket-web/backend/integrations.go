package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

func (s *Server) privacy(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed", "POST required")
		return
	}
	a := strings.TrimPrefix(r.URL.Path, "/api/privacy/")
	switch a {
	case "private-mode":
		var x struct {
			Enabled bool `json:"enabled"`
		}
		if decode(r, &x) != nil {
			fail(w, 400, "invalid_json", "enabled required")
			return
		}
		s.mu.Lock()
		s.private = x.Enabled
		s.mu.Unlock()
		if _, err := s.db.Exec(`INSERT INTO settings(key,value) VALUES('privateMode',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, boolJSON(x.Enabled)); err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		jsonOut(w, 200, map[string]bool{"private": x.Enabled})
	case "clear-history":
		_, _ = s.db.Exec(`DELETE FROM history`)
		jsonOut(w, 200, map[string]bool{"ok": true})
	case "clear-cookies", "clear-cache":
		s.mu.Lock()
		for id, v := range s.views {
			_ = v.Close()
			delete(s.views, id)
		}
		s.mu.Unlock()
		_, _ = s.db.Exec(`DELETE FROM tabs`)
		_, _ = s.db.Exec(`DELETE FROM sessions`)
		jsonOut(w, 200, map[string]any{"ok": true, "cleared": a, "profilesRemoved": true})
	case "delete-account-data":
		_, _ = s.db.Exec(`DELETE FROM oauth_tokens`)
		_, _ = s.db.Exec(`DELETE FROM accounts`)
		jsonOut(w, 200, map[string]bool{"ok": true})
	default:
		fail(w, 404, "not_found", "unknown action")
	}
}
func (s *Server) accounts(w http.ResponseWriter, r *http.Request) {
	if r.Method == "DELETE" {
		id := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
		if id == "" {
			_, _ = s.db.Exec(`DELETE FROM oauth_tokens`)
			_, _ = s.db.Exec(`DELETE FROM accounts`)
		} else {
			_, _ = s.db.Exec(`DELETE FROM oauth_tokens WHERE account_id=?`, id)
			_, _ = s.db.Exec(`DELETE FROM accounts WHERE id=?`, id)
		}
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	if r.Method != "GET" {
		fail(w, 405, "method_not_allowed", "GET or DELETE required")
		return
	}
	rows, e := s.db.Query(`SELECT id,provider,email,created_at FROM accounts`)
	if e != nil {
		fail(w, 500, "db_error", e.Error())
		return
	}
	defer rows.Close()
	a := []map[string]string{}
	for rows.Next() {
		var id, p, m, c string
		_ = rows.Scan(&id, &p, &m, &c)
		a = append(a, map[string]string{"id": id, "provider": p, "email": m, "createdAt": c})
	}
	jsonOut(w, 200, a)
}
func (s *Server) oauthRoute(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/oauth/"), "/"), "/")
	if len(p) != 2 {
		fail(w, 404, "not_found", "invalid OAuth path")
		return
	}
	provider, op := p[0], p[1]
	if provider != "google" && provider != "github" {
		fail(w, 404, "not_found", "unsupported provider")
		return
	}
	if op == "start" && r.Method == "POST" {
		verifierBytes := make([]byte, 32)
		_, _ = rand.Read(verifierBytes)
		ver := base64.RawURLEncoding.EncodeToString(verifierBytes)
		h := sha256.Sum256([]byte(ver))
		state := newID()
		s.mu.Lock()
		s.oauth[state] = oauthState{ver, provider, time.Now().Add(10 * time.Minute)}
		s.mu.Unlock()
		jsonOut(w, 200, map[string]string{"state": state, "codeVerifier": ver, "codeChallenge": base64.RawURLEncoding.EncodeToString(h[:]), "challengeMethod": "S256", "mode": "mock"})
		return
	}
	if op == "callback" && r.Method == "POST" {
		var x struct {
			State        string `json:"state"`
			CodeVerifier string `json:"codeVerifier"`
		}
		if decode(r, &x) != nil {
			fail(w, 400, "invalid_json", "callback required")
			return
		}
		s.mu.Lock()
		o, ok := s.oauth[x.State]
		delete(s.oauth, x.State)
		s.mu.Unlock()
		if !ok || time.Now().After(o.expires) || x.CodeVerifier != o.verifier {
			fail(w, 400, "invalid_state", "state or verifier invalid")
			return
		}
		jsonOut(w, 503, map[string]string{"error": "provider credentials unavailable", "provider": provider})
		return
	}
	fail(w, 404, "not_found", "unknown OAuth action")
}
func (s *Server) mockAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed", "POST required")
		return
	}
	var x struct {
		Provider string `json:"provider"`
		Email    string `json:"email"`
		Token    string `json:"token"`
	}
	if decode(r, &x) != nil {
		fail(w, 400, "invalid_json", "invalid account")
		return
	}
	if x.Provider == "" {
		x.Provider = "mock"
	}
	if x.Email == "" {
		x.Email = "demo@pocketweb.local"
	}
	id := newID()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, e := s.db.Exec(`INSERT INTO accounts VALUES(?,?,?,?)`, id, x.Provider, x.Email, now)
	if e != nil {
		fail(w, 500, "db_error", e.Error())
		return
	}
	if x.Token != "" {
		store, e := NewCredentialStore(s.db, s.key)
		if e == nil {
			e = store.Put(id, x.Token)
		}
		if e != nil {
			_, _ = s.db.Exec(`DELETE FROM accounts WHERE id=?`, id)
			fail(w, 500, "credential_error", "could not encrypt credential")
			return
		}
	}
	jsonOut(w, 201, map[string]string{"id": id, "provider": x.Provider, "email": x.Email, "createdAt": now, "mode": "mock"})
}
func (s *Server) actions(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")
	if target == "" {
		fail(w, 400, "invalid_url", "url query parameter required")
		return
	}
	jsonOut(w, 200, s.apps.Actions(r.Context(), target))
}

func (s *Server) launchAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 405, "method_not_allowed", "POST required")
		return
	}
	var input struct {
		App string `json:"app"`
		URL string `json:"url"`
	}
	if decode(r, &input) != nil || input.App == "" || input.URL == "" {
		fail(w, 400, "invalid_json", "app and url required")
		return
	}
	if err := s.launch.Launch(r.Context(), input.App, input.URL); err != nil {
		fail(w, 503, "app_unavailable", err.Error())
		return
	}
	jsonOut(w, 202, map[string]string{"status": "launched", "app": input.App})
}
