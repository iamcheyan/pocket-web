package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type SessionTab struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

func (s *Server) saveSession() error {
	s.mu.Lock()
	privateMode := s.private
	s.mu.Unlock()
	if privateMode {
		return nil
	}
	rows, err := s.db.Query(`SELECT id,title,url,active FROM tabs WHERE private=0 ORDER BY position`)
	if err != nil {
		return err
	}
	tabs := []SessionTab{}
	for rows.Next() {
		var tab SessionTab
		var active int
		if err = rows.Scan(&tab.ID, &tab.Title, &tab.URL, &active); err != nil {
			rows.Close()
			return err
		}
		tab.Active = active == 1
		tabs = append(tabs, tab)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	data, err := json.Marshal(tabs)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO sessions(created_at,tabs_json) VALUES(?,?)`, time.Now().UTC().Format(time.RFC3339Nano), string(data))
	return err
}
func (s *Server) sessionRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/sessions/save":
		if r.Method != http.MethodPost {
			fail(w, 405, "method_not_allowed", "POST required")
			return
		}
		if err := s.saveSession(); err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		jsonOut(w, 200, map[string]bool{"ok": true})
	case "/api/sessions/restore-previous", "/api/sessions/restore-tabs":
		if r.Method != http.MethodPost {
			fail(w, 405, "method_not_allowed", "POST required")
			return
		}
		var raw string
		if err := s.db.QueryRow(`SELECT tabs_json FROM sessions ORDER BY id DESC LIMIT 1`).Scan(&raw); err != nil {
			if err == sql.ErrNoRows {
				jsonOut(w, 200, []SessionTab{})
				return
			}
			fail(w, 500, "db_error", err.Error())
			return
		}
		var tabs []SessionTab
		if err := json.Unmarshal([]byte(raw), &tabs); err != nil {
			fail(w, 500, "invalid_session", "stored session corrupt")
			return
		}
		restored := []SessionTab{}
		for index, tab := range tabs {
			if tab.URL == "" || tab.ID == "" {
				continue
			}
			view, err := s.engine.NewView(r.Context(), tab.ID, ViewOptions{})
			if err != nil {
				continue
			}
			if err = view.Navigate(r.Context(), tab.URL); err != nil {
				_ = view.Close()
				continue
			}
			s.mu.Lock()
			s.views[tab.ID] = view
			s.mu.Unlock()
			if _, err = s.db.Exec(`INSERT INTO tabs(id,title,url,position,active,private) VALUES(?,?,?,?,?,0) ON CONFLICT(id) DO UPDATE SET title=excluded.title,url=excluded.url,position=excluded.position,active=excluded.active`, tab.ID, tab.Title, tab.URL, index, boolInt(tab.Active)); err != nil {
				_ = view.Close()
				fail(w, 500, "db_error", err.Error())
				return
			}
			restored = append(restored, tab)
		}
		jsonOut(w, 200, restored)
	default:
		fail(w, 404, "not_found", "unknown session route")
	}
}
