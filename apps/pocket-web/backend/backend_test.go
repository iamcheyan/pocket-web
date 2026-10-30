package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

type testEngine struct{ available bool }

func (e testEngine) Name() string              { return "test-engine" }
func (e testEngine) Available() (bool, string) { return e.available, "test engine" }
func (e testEngine) NewView(context.Context, string, ViewOptions) (BrowserView, error) {
	return nil, nil
}

func testServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	s := NewServer(db, testEngine{true}, dir)
	if err = s.migrate(); err != nil {
		t.Fatal(err)
	}
	if err = seedDemoData(db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
func request(t *testing.T, s *Server, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}
func TestMigrationCreatesCoreTables(t *testing.T) {
	s := testServer(t)
	for _, table := range []string{"tabs", "sessions", "bookmarks", "bookmark_folders", "reading_list", "history", "downloads", "settings", "accounts", "oauth_tokens"} {
		var name string
		if err := s.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name); err != nil {
			t.Errorf("table %s missing: %v", table, err)
		}
	}
}
func TestHealthReportsEngineAvailability(t *testing.T) {
	s := testServer(t)
	w := request(t, s, http.MethodGet, "/api/health")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["status"] != "ok" {
		t.Fatalf("unexpected health response: %#v", got)
	}
}
func TestReadingListEndpointIsIndependent(t *testing.T) {
	s := testServer(t)
	var bookmarkURLs, readingURLs []string
	for _, query := range []struct {
		table  string
		target *[]string
	}{{"bookmarks", &bookmarkURLs}, {"reading_list", &readingURLs}} {
		rows, err := s.db.Query(`SELECT url FROM ` + query.table)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var value string
			if err = rows.Scan(&value); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			*query.target = append(*query.target, value)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		_ = rows.Close()
	}
	if len(readingURLs) == 0 {
		t.Fatal("expected deterministic reading list seed")
	}
	for _, bookmark := range bookmarkURLs {
		for _, reading := range readingURLs {
			if bookmark == reading {
				t.Fatalf("reading entry %q duplicated a bookmark", reading)
			}
		}
	}
	if w := request(t, s, http.MethodGet, "/api/bookmarks"); w.Code != http.StatusOK {
		t.Fatalf("bookmarks status %d", w.Code)
	}
	if w := request(t, s, http.MethodGet, "/api/reading-list?sort=title"); w.Code != http.StatusOK {
		t.Fatalf("reading status %d", w.Code)
	}
}

func TestBookmarkImportExportRoundTrip(t *testing.T) {
	s := testServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/bookmarks/import", strings.NewReader(`[{"title":"Example","url":"https://example.test"}]`)))
	if w.Code != http.StatusOK {
		t.Fatalf("import status %d: %s", w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/bookmarks/export", nil))
	var got []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, entry := range got {
		if entry["url"] == "https://example.test" {
			found = true
		}
	}
	if w.Code != http.StatusOK || !found {
		t.Fatalf("export missing imported bookmark, status=%d: %#v", w.Code, got)
	}
}

func TestHistoryClearRemovesPersistedRows(t *testing.T) {
	s := testServer(t)
	if _, err := s.db.Exec(`INSERT INTO history(url,title,visit_count,last_visit) VALUES(?,?,?,?)`, "https://example.test", "Example", 2, "now"); err != nil {
		t.Fatal(err)
	}
	w := request(t, s, http.MethodDelete, "/api/history")
	if w.Code != http.StatusOK {
		t.Fatalf("clear status %d: %s", w.Code, w.Body)
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM history`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("history rows remain: %d", count)
	}
}
