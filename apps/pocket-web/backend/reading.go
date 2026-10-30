package main

import (
	"net/http"
	"strings"
	"time"
)

func (s *Server) reading(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		q := `SELECT id,title,url,read,offline_metadata,created_at FROM reading_list`
		args := []any{}
		if search := r.URL.Query().Get("q"); search != "" {
			q += ` WHERE title LIKE ? OR url LIKE ?`
			args = append(args, "%"+search+"%", "%"+search+"%")
		}
		if r.URL.Query().Get("sort") == "title" {
			q += ` ORDER BY title COLLATE NOCASE`
		} else {
			q += ` ORDER BY created_at DESC`
		}
		rows, err := s.db.Query(q, args...)
		if err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, title, url, meta, created string
			var read int
			if err = rows.Scan(&id, &title, &url, &read, &meta, &created); err != nil {
				fail(w, 500, "db_error", err.Error())
				return
			}
			out = append(out, map[string]any{"id": id, "title": title, "url": url, "read": read != 0, "offlineMetadata": meta, "createdAt": created})
		}
		jsonOut(w, 200, out)
		return
	}
	if r.Method != http.MethodPost {
		fail(w, 405, "method_not_allowed", "GET or POST required")
		return
	}
	var x readingInput
	if decode(r, &x) != nil || x.URL == "" {
		fail(w, 400, "invalid_json", "url required")
		return
	}
	id := newID()
	created := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`INSERT INTO reading_list(id,title,url,read,offline_metadata,created_at) VALUES(?,?,?,?,?,?)`, id, x.Title, x.URL, boolInt(x.Read), x.Offline, created); err != nil {
		fail(w, 500, "db_error", err.Error())
		return
	}
	jsonOut(w, 201, map[string]any{"id": id, "title": x.Title, "url": x.URL, "read": x.Read, "offlineMetadata": x.Offline, "createdAt": created})
}
func (s *Server) readingItem(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/reading-list/")
	if r.Method == http.MethodDelete {
		res, err := s.db.Exec(`DELETE FROM reading_list WHERE id=?`, id)
		if err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			fail(w, 404, "not_found", "reading item not found")
			return
		}
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	if r.Method != http.MethodPut {
		fail(w, 405, "method_not_allowed", "PUT or DELETE required")
		return
	}
	var x readingInput
	if decode(r, &x) != nil || x.URL == "" {
		fail(w, 400, "invalid_json", "invalid item")
		return
	}
	res, err := s.db.Exec(`UPDATE reading_list SET title=?,url=?,read=?,offline_metadata=? WHERE id=?`, x.Title, x.URL, boolInt(x.Read), x.Offline, id)
	if err != nil {
		fail(w, 500, "db_error", err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		fail(w, 404, "not_found", "reading item not found")
		return
	}
	jsonOut(w, 200, map[string]any{"id": id, "title": x.Title, "url": x.URL, "read": x.Read, "offlineMetadata": x.Offline})
}
