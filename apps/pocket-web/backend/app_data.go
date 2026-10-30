package main

import (
	"context"
	"database/sql"
	"time"
)

// seedDemoData inserts stable examples once so offline mode can exercise the UI.
func seedDemoData(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM settings WHERE key='demo_seeded'`).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return nil
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	folder := "demo-reading"
	_, err = tx.Exec(`INSERT INTO bookmark_folders(id,name,parent_id) VALUES(?,?,NULL)`, folder, "PJOS")
	if err != nil {
		return err
	}
	for _, b := range []struct{ id, title, url, folder string }{{"demo-bm-home", "Pocket Web project", "https://example.org/", folder}, {"demo-bm-docs", "Web standards", "https://developer.mozilla.org/", folder}} {
		if _, err = tx.Exec(`INSERT INTO bookmarks(id,title,url,folder_id,created_at) VALUES(?,?,?,?,?)`, b.id, b.title, b.url, b.folder, now); err != nil {
			return err
		}
	}
	for _, r := range []struct {
		id, title, url string
		read           int
	}{{"demo-read-1", "A short guide to the open web", "https://example.org/guide", 0}, {"demo-read-2", "Reading on small screens", "https://example.org/reading", 1}} {
		if _, err = tx.Exec(`INSERT INTO reading_list(id,title,url,read,offline_metadata,created_at) VALUES(?,?,?,?,?,?)`, r.id, r.title, r.url, r.read, "Saved metadata · offline content not downloaded", now); err != nil {
			return err
		}
	}
	for _, h := range []struct {
		url, title string
		visits     int
	}{{"https://example.org/", "Example Domain", 3}, {"https://developer.mozilla.org/", "MDN Web Docs", 1}} {
		if _, err = tx.Exec(`INSERT INTO history(url,title,visit_count,last_visit) VALUES(?,?,?,?)`, h.url, h.title, h.visits, now); err != nil {
			return err
		}
	}
	_, err = tx.Exec(`INSERT INTO settings(key,value) VALUES('demo_seeded','true')`)
	if err != nil {
		return err
	}
	return tx.Commit()
}
