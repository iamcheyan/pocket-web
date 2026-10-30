package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownloadCenterUnavailableIsExplicit(t *testing.T) {
	s := testServer(t)
	s.downloadClient = NewDownloadCenterFromEnv()
	w := request(t, s, http.MethodGet, "/api/downloads")
	if w.Code != http.StatusOK {
		t.Fatalf("downloads status %d", w.Code)
	}
	var response struct {
		Available bool       `json:"available"`
		Reason    string     `json:"reason"`
		Items     []Download `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Available || response.Reason == "" || len(response.Items) != 0 {
		t.Fatalf("unexpected unavailable response: %#v", response)
	}
}

func TestDownloadCenterSyncsRemoteTasks(t *testing.T) {
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"remote-1","title":"Archive","url":"https://example.test/a.zip","status":"completed","category":"Archives"}]`))
	}))
	defer remote.Close()
	s := testServer(t)
	s.downloadClient = &HTTPDownloadCenter{BaseURL: remote.URL, Client: remote.Client()}
	response := request(t, s, http.MethodGet, "/api/downloads")
	if response.Code != http.StatusOK {
		t.Fatalf("downloads status %d: %s", response.Code, response.Body)
	}
	var payload struct {
		Available bool       `json:"available"`
		Items     []Download `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Available || len(payload.Items) != 1 || payload.Items[0].ID != "remote-1" {
		t.Fatalf("remote tasks not surfaced: %#v", payload)
	}
	var status, category string
	if err := s.db.QueryRow(`SELECT status,category FROM downloads WHERE id='remote-1'`).Scan(&status, &category); err != nil {
		t.Fatal(err)
	}
	if status != "completed" || category != "Archives" {
		t.Fatalf("download mirror mismatch: %s/%s", status, category)
	}
}
