package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type DownloadCenterClient interface {
	Available(context.Context) (bool, string)
	List(context.Context) ([]Download, error)
	Create(context.Context, string, string, string) (Download, error)
}
type Download struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Status   string `json:"status"`
	Category string `json:"category"`
}
type HTTPDownloadCenter struct {
	BaseURL string
	Client  *http.Client
}

func NewDownloadCenterFromEnv() *HTTPDownloadCenter {
	return &HTTPDownloadCenter{BaseURL: strings.TrimRight(os.Getenv("DOWNLOADSD_URL"), "/"), Client: &http.Client{Timeout: 5 * time.Second}}
}
func (c *HTTPDownloadCenter) Available(ctx context.Context) (bool, string) {
	if c.BaseURL == "" {
		return false, "DOWNLOADSD_URL is unset; offline mock mode"
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/health", nil)
	if e != nil {
		return false, e.Error()
	}
	resp, e := c.client().Do(req)
	if e != nil {
		return false, "downloadsd unavailable: " + e.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Sprintf("downloadsd health returned HTTP %d", resp.StatusCode)
	}
	return true, "downloadsd reachable"
}
func (c *HTTPDownloadCenter) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 5 * time.Second}
}
func (c *HTTPDownloadCenter) List(ctx context.Context) ([]Download, error) {
	if c.BaseURL == "" {
		return nil, errors.New("downloadsd unavailable")
	}
	req, e := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/downloads", nil)
	if e != nil {
		return nil, e
	}
	resp, e := c.client().Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("downloadsd returned HTTP %d", resp.StatusCode)
	}
	var a []Download
	e = json.NewDecoder(resp.Body).Decode(&a)
	return a, e
}
func (c *HTTPDownloadCenter) Create(ctx context.Context, u, title, category string) (Download, error) {
	if c.BaseURL == "" {
		return Download{}, errors.New("downloadsd unavailable")
	}
	body := strings.NewReader(fmt.Sprintf(`{"url":%q,"title":%q,"category":%q}`, u, title, category))
	req, e := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/downloads", body)
	if e != nil {
		return Download{}, e
	}
	req.Header.Set("Content-Type", "application/json")
	resp, e := c.client().Do(req)
	if e != nil {
		return Download{}, e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Download{}, fmt.Errorf("downloadsd returned HTTP %d", resp.StatusCode)
	}
	var d Download
	e = json.NewDecoder(resp.Body).Decode(&d)
	return d, e
}
