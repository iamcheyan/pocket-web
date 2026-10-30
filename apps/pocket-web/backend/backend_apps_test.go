package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

type fakeLaunch struct{ installed bool }

func (f fakeLaunch) Installed(context.Context, string) (bool, string) {
	if f.installed {
		return true, "installed for test"
	}
	return false, "not installed for test"
}
func (f fakeLaunch) Launch(context.Context, string, string) error { return nil }

func TestContextualActionsUseReplaceableRegistry(t *testing.T) {
	s := testServer(t)
	s.apps = DefaultPJAppRegistry{Apps: DefaultApps, Launch: fakeLaunch{installed: false}}
	w := request(t, s, http.MethodGet, "/api/apps/actions?url=https%3A%2F%2Fexample.test%2Fgame.gba")
	if w.Code != http.StatusOK {
		t.Fatalf("actions status %d: %s", w.Code, w.Body)
	}
	var actions []AppAction
	if err := json.Unmarshal(w.Body.Bytes(), &actions); err != nil {
		t.Fatal(err)
	}
	if len(actions) != 4 {
		t.Fatalf("expected four app actions, got %d", len(actions))
	}
	for _, action := range actions {
		if action.Available {
			t.Fatalf("uninstalled app reported available: %#v", action)
		}
	}
	if actions[3].Reason != "not installed for test" {
		t.Fatalf("GameHub should use ROM eligibility then install state: %#v", actions[3])
	}
}

func TestGameHubRejectsNonROMLink(t *testing.T) {
	s := testServer(t)
	s.apps = DefaultPJAppRegistry{Apps: DefaultApps, Launch: fakeLaunch{installed: true}}
	w := request(t, s, http.MethodGet, "/api/apps/actions?url=https%3A%2F%2Fexample.test%2F")
	var actions []AppAction
	if err := json.Unmarshal(w.Body.Bytes(), &actions); err != nil {
		t.Fatal(err)
	}
	if actions[3].Available || actions[3].Reason != "not a recognized game ROM link" {
		t.Fatalf("unexpected GameHub action: %#v", actions[3])
	}
}
