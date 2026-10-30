package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestOAuthPKCEStateIsValidatedAndSingleUse(t *testing.T) {
	s := testServer(t)
	start := postJSON(t, s, "/api/oauth/google/start", `{}`)
	if start.Code != http.StatusOK {
		t.Fatalf("start status %d: %s", start.Code, start.Body)
	}
	var flow map[string]string
	if err := json.Unmarshal(start.Body.Bytes(), &flow); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(flow["codeVerifier"]))
	if got := base64.RawURLEncoding.EncodeToString(sum[:]); got != flow["codeChallenge"] {
		t.Fatalf("challenge mismatch: %s", got)
	}
	bad := postJSON(t, s, "/api/oauth/google/callback", `{"state":"`+flow["state"]+`","codeVerifier":"wrong"}`)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("bad verifier status %d", bad.Code)
	}
	replayed := postJSON(t, s, "/api/oauth/google/callback", `{"state":"`+flow["state"]+`","codeVerifier":"`+flow["codeVerifier"]+`"}`)
	if replayed.Code != http.StatusBadRequest {
		t.Fatalf("replayed state unexpectedly accepted: %d", replayed.Code)
	}
}

func TestCredentialStoreEncryptsAndAPIExposesNoPlaintext(t *testing.T) {
	s := testServer(t)
	token := "secret-token-value-for-test"
	created := postJSON(t, s, "/api/auth/mock", `{"provider":"google","email":"user@example.test","token":"`+token+`"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("mock auth status %d: %s", created.Code, created.Body)
	}
	if strings.Contains(created.Body.String(), token) {
		t.Fatal("mock auth response leaked plaintext token")
	}
	var accountID string
	if err := s.db.QueryRow(`SELECT id FROM accounts WHERE email='user@example.test'`).Scan(&accountID); err != nil {
		t.Fatal(err)
	}
	if s.key == nil || len(s.key) != 32 {
		t.Fatal("test server has no credential key")
	}
	var ciphertext string
	if err := s.db.QueryRow(`SELECT ciphertext FROM oauth_tokens WHERE account_id=?`, accountID).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ciphertext, token) || ciphertext == token {
		t.Fatal("token was persisted in plaintext")
	}
	store, err := NewCredentialStore(s.db, s.key)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := store.Get(accountID)
	if err != nil || opened != token {
		t.Fatalf("credential decrypt mismatch: %q %v", opened, err)
	}
	accounts := request(t, s, http.MethodGet, "/api/accounts")
	if strings.Contains(accounts.Body.String(), token) {
		t.Fatal("accounts endpoint leaked plaintext token")
	}
}
