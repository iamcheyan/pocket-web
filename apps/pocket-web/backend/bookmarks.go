package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *Server) bookmarks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := `SELECT id,title,url,COALESCE(folder_id,''),created_at FROM bookmarks`
		args := []any{}
		if search := r.URL.Query().Get("q"); search != "" {
			q += ` WHERE title LIKE ? OR url LIKE ?`
			args = append(args, "%"+search+"%", "%"+search+"%")
		}
		q += ` ORDER BY created_at DESC`
		rows, err := s.db.Query(q, args...)
		if err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, title, url, folder, created string
			if err = rows.Scan(&id, &title, &url, &folder, &created); err != nil {
				fail(w, 500, "db_error", err.Error())
				return
			}
			out = append(out, map[string]any{"id": id, "title": title, "url": url, "folderId": folder, "createdAt": created})
		}
		if err = rows.Err(); err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		jsonOut(w, 200, out)
	case http.MethodPost:
		var x bookmarkInput
		if decode(r, &x) != nil || x.URL == "" {
			fail(w, 400, "invalid_json", "url required")
			return
		}
		id := newID()
		created := time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := s.db.Exec(`INSERT INTO bookmarks(id,title,url,folder_id,created_at) VALUES(?,?,?,?,?)`, id, x.Title, x.URL, x.FolderID, created); err != nil {
			fail(w, 400, "invalid_bookmark", err.Error())
			return
		}
		jsonOut(w, 201, map[string]any{"id": id, "title": x.Title, "url": x.URL, "folderId": x.FolderID, "createdAt": created})
	default:
		fail(w, 405, "method_not_allowed", "GET or POST required")
	}
}
func (s *Server) bookmark(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/bookmarks/")
	if id == "import" || id == "export" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var x bookmarkInput
		if decode(r, &x) != nil || x.URL == "" {
			fail(w, 400, "invalid_json", "url required")
			return
		}
		res, err := s.db.Exec(`UPDATE bookmarks SET title=?,url=?,folder_id=? WHERE id=?`, x.Title, x.URL, x.FolderID, id)
		if err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			fail(w, 404, "not_found", "bookmark not found")
			return
		}
		jsonOut(w, 200, map[string]any{"id": id, "title": x.Title, "url": x.URL, "folderId": x.FolderID})
	case http.MethodDelete:
		res, err := s.db.Exec(`DELETE FROM bookmarks WHERE id=?`, id)
		if err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			fail(w, 404, "not_found", "bookmark not found")
			return
		}
		jsonOut(w, 200, map[string]bool{"ok": true})
	default:
		fail(w, 405, "method_not_allowed", "PUT or DELETE required")
	}
}
func (s *Server) folders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := s.db.Query(`SELECT id,name,COALESCE(parent_id,'') FROM bookmark_folders ORDER BY name`)
		if err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		defer rows.Close()
		out := []map[string]string{}
		for rows.Next() {
			var id, name, parent string
			if err = rows.Scan(&id, &name, &parent); err != nil {
				fail(w, 500, "db_error", err.Error())
				return
			}
			out = append(out, map[string]string{"id": id, "name": name, "parentId": parent})
		}
		jsonOut(w, 200, out)
	case http.MethodPost:
		var x struct {
			Name     string `json:"name"`
			ParentID string `json:"parentId"`
		}
		if decode(r, &x) != nil || strings.TrimSpace(x.Name) == "" {
			fail(w, 400, "invalid_json", "folder name required")
			return
		}
		id := newID()
		if _, err := s.db.Exec(`INSERT INTO bookmark_folders(id,name,parent_id) VALUES(?,?,NULLIF(?,''))`, id, x.Name, x.ParentID); err != nil {
			fail(w, 400, "invalid_folder", err.Error())
			return
		}
		jsonOut(w, 201, map[string]string{"id": id, "name": x.Name, "parentId": x.ParentID})
	default:
		fail(w, 405, "method_not_allowed", "GET or POST required")
	}
}
func (s *Server) importBookmarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, 405, "method_not_allowed", "POST required")
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, 400, "invalid_json", "could not read bookmark import")
		return
	}
	var in []bookmarkInput
	if err = json.Unmarshal(data, &in); err != nil {
		var wrapped struct {
			Bookmarks []bookmarkInput `json:"bookmarks"`
		}
		if err = json.Unmarshal(data, &wrapped); err != nil {
			fail(w, 400, "invalid_json", "expected bookmark array")
			return
		}
		in = wrapped.Bookmarks
	}
	tx, err := s.db.Begin()
	if err != nil {
		fail(w, 500, "db_error", err.Error())
		return
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, x := range in {
		if x.URL == "" {
			fail(w, 400, "invalid_bookmark", "url required")
			return
		}
		if _, err = tx.Exec(`INSERT INTO bookmarks(id,title,url,folder_id,created_at) VALUES(?,?,?,?,?)`, newID(), x.Title, x.URL, x.FolderID, now); err != nil {
			fail(w, 400, "invalid_bookmark", err.Error())
			return
		}
	}
	if err = tx.Commit(); err != nil {
		fail(w, 500, "db_error", err.Error())
		return
	}
	jsonOut(w, 200, map[string]int{"imported": len(in)})
}
func (s *Server) exportBookmarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fail(w, 405, "method_not_allowed", "GET required")
		return
	}
	rows, err := s.db.Query(`SELECT title,url FROM bookmarks ORDER BY created_at`)
	if err != nil {
		fail(w, 500, "db_error", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var title, url string
		if err = rows.Scan(&title, &url); err != nil {
			fail(w, 500, "db_error", err.Error())
			return
		}
		out = append(out, map[string]string{"title": title, "url": url})
	}
	jsonOut(w, 200, out)
}
