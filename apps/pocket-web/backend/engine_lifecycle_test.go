package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestChromiumViewCloseKillsProcessGroupAndRemovesProfile(t *testing.T) {
	if _, err := exec.LookPath("chromium"); err != nil {
		t.Skip("chromium is not installed")
	}
	view, err := (&ChromiumHeadlessAdapter{}).NewView(context.Background(), "cleanup-test", ViewOptions{})
	if err != nil {
		t.Fatalf("launch view: %v", err)
	}
	chromium := view.(*chromiumView)
	profile := chromium.profile
	pid := chromium.cmd.Process.Pid
	if err = chromium.Navigate(context.Background(), "about:blank"); err != nil {
		_ = view.Close()
		t.Fatalf("navigate: %v", err)
	}
	if err = view.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err = syscall.Kill(-pid, 0); err == nil || err != syscall.ESRCH {
		t.Fatalf("engine process group still alive after close: %v", err)
	}
	if _, err = os.Stat(profile); !os.IsNotExist(err) {
		t.Fatalf("engine profile still present after close: %v", err)
	}
	for _, line := range processLinesWithProfile(profile) {
		t.Fatalf("child process survived close: %s", strings.TrimSpace(line))
	}
}

func processLinesWithProfile(profile string) []string {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	matches := []string{}
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err == nil && strings.Contains(string(cmdline), profile) {
			matches = append(matches, entry.Name())
		}
	}
	time.Sleep(10 * time.Millisecond)
	return matches
}
