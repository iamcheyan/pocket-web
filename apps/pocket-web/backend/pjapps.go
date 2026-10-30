package main

import (
	"context"
	"errors"
	"net/url"
	"os/exec"
	"strings"
)

// AppLaunchAPI is the boundary through which Pocket Web hands a URL to another PJOS app.
type AppLaunchAPI interface {
	Installed(context.Context, string) (bool, string)
	Launch(context.Context, string, string) error
}

// PJAppRegistry describes which external PJOS applications Pocket Web can present.
type PJAppRegistry interface {
	Actions(context.Context, string) []AppAction
}

type AppAction struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	URL       string `json:"url"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
	Kind      string `json:"kind"`
}

type PocketApp struct {
	ID      string
	Label   string
	Binary  string
	Kind    string
	Schemes []string
}

type DefaultPJAppRegistry struct {
	Apps   []PocketApp
	Launch AppLaunchAPI
}

type ExecAppLaunchAPI struct{}

func (ExecAppLaunchAPI) Installed(_ context.Context, appID string) (bool, string) {
	for _, app := range DefaultApps {
		if app.ID == appID {
			if _, err := exec.LookPath(app.Binary); err != nil {
				return false, app.Label + " is not installed"
			}
			return true, app.Label + " is installed"
		}
	}
	return false, "unknown Pocket application"
}
func (ExecAppLaunchAPI) Launch(ctx context.Context, appID, target string) error {
	if _, err := url.ParseRequestURI(target); err != nil {
		return errors.New("invalid target URL")
	}
	for _, app := range DefaultApps {
		if app.ID != appID {
			continue
		}
		if _, err := exec.LookPath(app.Binary); err != nil {
			return errors.New(app.Label + " is not installed")
		}
		return exec.CommandContext(ctx, app.Binary, target).Start()
	}
	return errors.New("unknown Pocket application")
}

var DefaultApps = []PocketApp{
	{ID: "pocket-reader", Label: "Save to Pocket Reader", Binary: "pocket-reader", Kind: "reader"},
	{ID: "pocket-video", Label: "Download with Pocket Video", Binary: "pocket-video", Kind: "download-video"},
	{ID: "pocket-music", Label: "Download Audio with Pocket Music", Binary: "pocket-music", Kind: "download-audio"},
	{ID: "gamehub", Label: "Open Game ROM Link with GameHub", Binary: "pocket-gamehub", Kind: "game-rom"},
}

func (registry DefaultPJAppRegistry) Actions(ctx context.Context, target string) []AppAction {
	actions := make([]AppAction, 0, len(registry.Apps))
	for _, app := range registry.Apps {
		available, reason := registry.Launch.Installed(ctx, app.ID)
		if app.Kind == "game-rom" && !isROMURL(target) {
			available, reason = false, "not a recognized game ROM link"
		}
		actions = append(actions, AppAction{ID: app.ID, Label: app.Label, URL: target, Available: available, Reason: reason, Kind: app.Kind})
	}
	return actions
}

func isROMURL(target string) bool {
	parsed, err := url.Parse(target)
	if err != nil {
		return false
	}
	path := strings.ToLower(parsed.Path)
	for _, extension := range []string{".zip", ".7z", ".gba", ".gb", ".gbc", ".nds", ".3ds", ".nes", ".sfc", ".smc", ".iso", ".xci", ".nsp"} {
		if strings.HasSuffix(path, extension) {
			return true
		}
	}
	return false
}
