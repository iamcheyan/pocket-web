package main

import (
	"database/sql"
	"net/http"
	"strings"
)

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		if _, err := s.db.Exec(`DELETE FROM history`); err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	if r.Method != http.MethodGet {
		fail(w, 405, "method_not_allowed", "GET or DELETE required")
		return
	}
	q := "%" + r.URL.Query().Get("q") + "%"
	rows, err := s.db.Query(`SELECT id,url,title,visit_count,last_visit FROM history WHERE url LIKE ? OR title LIKE ? ORDER BY last_visit DESC`, q, q)
	if err != nil {
		fail(w, 500, "db_error", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, count int
		var url, title, last string
		if err = rows.Scan(&id, &url, &title, &count, &last); err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		out = append(out, map[string]any{"id": id, "url": url, "title": title, "visitCount": count, "lastVisit": last})
	}
	jsonOut(w, 200, out)
}
func (s *Server) historyItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		fail(w, 405, "method_not_allowed", "DELETE required")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/history/")
	var result sql.Result
	var err error
	if id == "" {
		result, err = s.db.Exec(`DELETE FROM history`)
	} else {
		result, err = s.db.Exec(`DELETE FROM history WHERE id=?`, id)
	}
	if err != nil {
		fail(w, 500, "db_error", err.Error())
		return
	}
	if id != "" {
		n, _ := result.RowsAffected()
		if n == 0 {
			fail(w, 404, "not_found", "history entry not found")
			return
		}
	}
	jsonOut(w, 200, map[string]bool{"ok": true})
}
