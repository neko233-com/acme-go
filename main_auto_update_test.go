package main

import (
	"errors"
	"io"
	"path/filepath"
	"testing"
)

func TestRunTriggersAutoUpdateForOperationalCommands(t *testing.T) {
	originalAutoUpdate := autoUpdate
	originalLocateExecPath := locateExecPath
	autoUpdateCalls := 0
	autoUpdate = func(currentVersion, executablePath string, out io.Writer) error {
		autoUpdateCalls++
		return nil
	}
	locateExecPath = func() (string, error) { return filepath.Join(t.TempDir(), "acme233"), nil }
	t.Cleanup(func() {
		autoUpdate = originalAutoUpdate
		locateExecPath = originalLocateExecPath
	})

	if err := run([]string{"providers"}); err != nil {
		t.Fatalf("run providers: %v", err)
	}
	if autoUpdateCalls != 1 {
		t.Fatalf("auto update calls: got %d want 1", autoUpdateCalls)
	}
}

func TestRunSkipsAutoUpdateForHelpTopics(t *testing.T) {
	originalAutoUpdate := autoUpdate
	autoUpdateCalls := 0
	autoUpdate = func(currentVersion, executablePath string, out io.Writer) error {
		autoUpdateCalls++
		return nil
	}
	t.Cleanup(func() { autoUpdate = originalAutoUpdate })

	if err := run([]string{"help", "version"}); err != nil {
		t.Fatalf("run help version: %v", err)
	}
	if autoUpdateCalls != 0 {
		t.Fatalf("auto update calls: got %d want 0", autoUpdateCalls)
	}
}

func TestRunIgnoresAutoUpdateFailure(t *testing.T) {
	originalAutoUpdate := autoUpdate
	originalLocateExecPath := locateExecPath
	autoUpdate = func(currentVersion, executablePath string, out io.Writer) error {
		return errors.New("boom")
	}
	locateExecPath = func() (string, error) { return filepath.Join(t.TempDir(), "acme233"), nil }
	t.Cleanup(func() {
		autoUpdate = originalAutoUpdate
		locateExecPath = originalLocateExecPath
	})

	if err := run([]string{"providers"}); err != nil {
		t.Fatalf("run providers: %v", err)
	}
}
