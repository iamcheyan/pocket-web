package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
)

type CredentialStore interface {
	Put(accountID, token string) error
	Get(accountID string) (string, error)
	Delete(accountID string) error
}
type AESGCMCredentialStore struct {
	db  *sql.DB
	key []byte
}

func NewCredentialStore(db *sql.DB, key []byte) (*AESGCMCredentialStore, error) {
	if len(key) != 32 {
		return nil, errors.New("credential key must be 32 bytes")
	}
	return &AESGCMCredentialStore{db: db, key: append([]byte(nil), key...)}, nil
}
func LoadCredentialKey(dir string) ([]byte, error) {
	p := filepath.Join(dir, "credential.key")
	key, err := os.ReadFile(p)
	if err == nil {
		if len(key) != 32 {
			return nil, errors.New("credential key must be 32 bytes")
		}
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	key = make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	_, err = f.Write(key)
	closeErr := f.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return key, nil
}
func (c *AESGCMCredentialStore) seal(token string) (string, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(append(nonce, g.Seal(nil, nonce, []byte(token), nil)...)), nil
}
func (c *AESGCMCredentialStore) open(ciphertext string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(b) < g.NonceSize() {
		return "", errors.New("credential ciphertext truncated")
	}
	plain, err := g.Open(nil, b[:g.NonceSize()], b[g.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
func (c *AESGCMCredentialStore) Put(id, token string) error {
	sealed, err := c.seal(token)
	if err != nil {
		return err
	}
	_, err = c.db.Exec(`INSERT INTO oauth_tokens(account_id,ciphertext) VALUES(?,?) ON CONFLICT(account_id) DO UPDATE SET ciphertext=excluded.ciphertext`, id, sealed)
	return err
}
func (c *AESGCMCredentialStore) Get(id string) (string, error) {
	var sealed string
	if err := c.db.QueryRow(`SELECT ciphertext FROM oauth_tokens WHERE account_id=?`, id).Scan(&sealed); err != nil {
		return "", err
	}
	return c.open(sealed)
}
func (c *AESGCMCredentialStore) Delete(id string) error {
	_, err := c.db.Exec(`DELETE FROM oauth_tokens WHERE account_id=?`, id)
	return err
}
