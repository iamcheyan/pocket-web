package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) downloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, 405, "method_not_allowed", "GET required")
		return
	}
	available, reason := s.downloadClient.Available(r.Context())
	if !available {
		jsonOut(w, 200, map[string]any{"available": false, "reason": reason, "items": []any{}})
		return
	}
	items, err := s.downloadClient.List(r.Context())
	if err != nil {
		jsonOut(w, 200, map[string]any{"available": false, "reason": err.Error(), "items": []any{}})
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		fail(w, 500, "db_error", err.Error())
		return
	}
	for _, item := range items {
		if _, err = tx.Exec(`INSERT INTO downloads(id,title,url,status,category) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET title=excluded.title,url=excluded.url,status=excluded.status,category=excluded.category`, item.ID, item.Title, item.URL, item.Status, item.Category); err != nil {
			_ = tx.Rollback()
			fail(w, 500, "db_error", err.Error())
			return
		}
	}
	if err = tx.Commit(); err != nil {
		fail(w, 500, "db_error", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"available": true, "items": items})
}
func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rows, err := s.db.Query(`SELECT key,value FROM settings`)
		if err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		defer rows.Close()
		out := map[string]any{}
		for rows.Next() {
			var key, value string
			if err = rows.Scan(&key, &value); err != nil {
				fail(w, 500, "db_error", err.Error())
				return
			}
			var decoded any
			if json.Unmarshal([]byte(value), &decoded) == nil {
				out[key] = decoded
			} else {
				out[key] = value
			}
		}
		jsonOut(w, 200, out)
		return
	}
	if r.Method != http.MethodPut {
		fail(w, 405, "method_not_allowed", "GET or PUT required")
		return
	}
	var values map[string]json.RawMessage
	if decode(r, &values) != nil {
		fail(w, 400, "invalid_json", "expected settings object")
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		fail(w, 500, "db_error", err.Error())
		return
	}
	for key, value := range values {
		if key == "" || strings.ContainsAny(key, "\x00") {
			_ = tx.Rollback()
			fail(w, 400, "invalid_setting", "invalid setting key")
			return
		}
		if _, err = tx.Exec(`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, string(value)); err != nil {
			_ = tx.Rollback()
			fail(w, 500, "db_error", err.Error())
			return
		}
		if key == "privateMode" {
			var enabled bool
			if json.Unmarshal(value, &enabled) == nil {
				s.mu.Lock()
				s.private = enabled
				s.mu.Unlock()
			}
		}
	}
	if err = tx.Commit(); err != nil {
		fail(w, 500, "db_error", err.Error())
		return
	}
	jsonOut(w, 200, values)
}
