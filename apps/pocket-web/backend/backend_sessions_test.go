package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestSessionSaveAndRestoreRebuildsTabState(t *testing.T) {
	engine := &trackingEngine{}
	s := testServerWithEngine(t, engine)
	created := postJSON(t, s, "/api/tabs", `{"url":"https://example.test/restored"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", created.Code, created.Body)
	}
	if saved := postJSON(t, s, "/api/sessions/save", `{}`); saved.Code != http.StatusOK {
		t.Fatalf("save status %d: %s", saved.Code, saved.Body)
	}
	if _, err := s.db.Exec(`DELETE FROM tabs`); err != nil {
		t.Fatal(err)
	}
	openedBefore := engine.opened
	restored := postJSON(t, s, "/api/sessions/restore-previous", `{}`)
	var tabs []SessionTab
	if err := json.Unmarshal(restored.Body.Bytes(), &tabs); err != nil {
		t.Fatal(err)
	}
	if restored.Code != http.StatusOK || len(tabs) != 1 || tabs[0].URL != "https://example.test/restored" {
		t.Fatalf("restore result %d: %#v", restored.Code, tabs)
	}
	if engine.opened != openedBefore+1 {
		t.Fatalf("restore did not create a fresh engine view: %d -> %d", openedBefore, engine.opened)
	}
	var persisted int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM tabs WHERE url='https://example.test/restored'`).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != 1 {
		t.Fatalf("restored tab rows = %d", persisted)
	}
}
