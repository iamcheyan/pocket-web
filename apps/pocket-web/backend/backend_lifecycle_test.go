package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type trackingEngine struct {
	opened  int
	closed  int
	lastURL string
}

func (e *trackingEngine) Name() string              { return "tracking-engine" }
func (e *trackingEngine) Available() (bool, string) { return true, "tracking engine" }
func (e *trackingEngine) NewView(_ context.Context, _ string, _ ViewOptions) (BrowserView, error) {
	e.opened++
	return &trackingView{engine: e}, nil
}

type trackingView struct{ engine *trackingEngine }

func (v *trackingView) Navigate(_ context.Context, url string) error {
	v.engine.lastURL = url
	return nil
}
func (v *trackingView) Back(context.Context) error    { return nil }
func (v *trackingView) Forward(context.Context) error { return nil }
func (v *trackingView) Reload(context.Context) error  { return nil }
func (v *trackingView) Stop(context.Context) error    { return nil }
func (v *trackingView) Close() error                  { v.engine.closed++; return nil }

func TestTabDuplicateAndRestoreUseURL(t *testing.T) {
	engine := &trackingEngine{}
	s := testServerWithEngine(t, engine)
	created := postJSON(t, s, "/api/tabs", `{"url":"https://example.test/page"}`)
	var original map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &original); err != nil {
		t.Fatal(err)
	}
	id, _ := original["id"].(string)
	duplicated := postJSON(t, s, "/api/tabs/"+id+"/duplicate", `{}`)
	var duplicate map[string]any
	if err := json.Unmarshal(duplicated.Body.Bytes(), &duplicate); err != nil {
		t.Fatal(err)
	}
	if duplicate["url"] != "https://example.test/page" {
		t.Fatalf("duplicate lost source URL: %#v", duplicate)
	}
	if res := request(t, s, http.MethodDelete, "/api/tabs/"+id); res.Code != http.StatusOK {
		t.Fatalf("close status %d: %s", res.Code, res.Body)
	}
	restored := postJSON(t, s, "/api/tabs/restore-closed", `{}`)
	var restoredTab map[string]any
	if err := json.Unmarshal(restored.Body.Bytes(), &restoredTab); err != nil {
		t.Fatal(err)
	}
	if restored.Code != http.StatusCreated || restoredTab["url"] != "https://example.test/page" {
		t.Fatalf("restore failed: %d %#v", restored.Code, restoredTab)
	}
}

func TestPrivateModeSuppressesHistoryAndSessionRows(t *testing.T) {
	engine := &trackingEngine{}
	s := testServerWithEngine(t, engine)
	if res := postJSON(t, s, "/api/privacy/private-mode", `{"enabled":true}`); res.Code != http.StatusOK {
		t.Fatalf("private mode status %d", res.Code)
	}
	if res := postJSON(t, s, "/api/tabs", `{"url":"https://private.example/"}`); res.Code != http.StatusCreated {
		t.Fatalf("private tab status %d: %s", res.Code, res.Body)
	}
	var historyCount, tabCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM history WHERE url='https://private.example/'`).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM tabs`).Scan(&tabCount); err != nil {
		t.Fatal(err)
	}
	if historyCount != 0 {
		t.Fatalf("private navigation persisted %d history rows", historyCount)
	}
	if tabCount != 0 {
		t.Fatalf("private navigation persisted %d tab rows", tabCount)
	}
	if res := postJSON(t, s, "/api/sessions/save", `{}`); res.Code != http.StatusOK {
		t.Fatalf("session save status %d", res.Code)
	}
	var sessionCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&sessionCount); err != nil {
		t.Fatal(err)
	}
	if sessionCount != 0 {
		t.Fatalf("private mode persisted %d sessions", sessionCount)
	}
}

func testServerWithEngine(t *testing.T, engine BrowserEngine) *Server {
	t.Helper()
	s := testServer(t)
	s.engine = engine
	return s
}

func postJSON(t *testing.T, s *Server, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
	return w
}
