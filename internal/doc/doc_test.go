package doc

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveGuidePathPrefersWorkingDirectory(t *testing.T) {
	workingDir := t.TempDir()
	executableDir := t.TempDir()
	workingGuide := filepath.Join(workingDir, GuideFileName)
	executableGuide := filepath.Join(executableDir, GuideFileName)
	if err := osWriteFile(workingGuide, []byte("guide")); err != nil {
		t.Fatalf("write working guide: %v", err)
	}
	if err := osWriteFile(executableGuide, []byte("guide")); err != nil {
		t.Fatalf("write executable guide: %v", err)
	}

	path, err := ResolveGuidePath(workingDir, filepath.Join(executableDir, "acme233"))
	if err != nil {
		t.Fatalf("ResolveGuidePath: %v", err)
	}
	if path != workingGuide {
		t.Fatalf("path: got %q want %q", path, workingGuide)
	}
}

func TestResolveGuidePathFallsBackToExecutableDirectory(t *testing.T) {
	executableDir := t.TempDir()
	executableGuide := filepath.Join(executableDir, GuideFileName)
	if err := osWriteFile(executableGuide, []byte("guide")); err != nil {
		t.Fatalf("write executable guide: %v", err)
	}

	path, err := ResolveGuidePath("", filepath.Join(executableDir, "acme233"))
	if err != nil {
		t.Fatalf("ResolveGuidePath: %v", err)
	}
	if path != executableGuide {
		t.Fatalf("path: got %q want %q", path, executableGuide)
	}
}

func TestOpenGuideUsesPlatformCommand(t *testing.T) {
	workingDir := t.TempDir()
	guidePath := filepath.Join(workingDir, GuideFileName)
	if err := osWriteFile(guidePath, []byte("guide")); err != nil {
		t.Fatalf("write guide: %v", err)
	}

	var command string
	var args []string
	original := startCommand
	startCommand = func(name string, values ...string) error {
		command = name
		args = append([]string(nil), values...)
		return nil
	}
	t.Cleanup(func() { startCommand = original })

	path, err := OpenGuide(workingDir, "")
	if err != nil {
		t.Fatalf("OpenGuide: %v", err)
	}
	if path != guidePath {
		t.Fatalf("path: got %q want %q", path, guidePath)
	}
	if command == "" {
		t.Fatal("expected opener command to be invoked")
	}
	switch runtime.GOOS {
	case "windows":
		if command != "cmd" || len(args) < 4 || !strings.EqualFold(args[0], "/C") {
			t.Fatalf("unexpected windows command: %s %v", command, args)
		}
	case "darwin":
		if command != "open" {
			t.Fatalf("unexpected macOS command: %s %v", command, args)
		}
	default:
		if command != "xdg-open" {
			t.Fatalf("unexpected linux command: %s %v", command, args)
		}
	}
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
