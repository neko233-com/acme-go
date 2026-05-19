package doc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const GuideFileName = "how-to-use.html"

var startCommand = func(name string, args ...string) error {
	command := exec.Command(name, args...)
	return command.Start()
}

func ResolveGuidePath(workingDir, executablePath string) (string, error) {
	candidates := []string{}
	if workingDir != "" {
		candidates = append(candidates, filepath.Join(workingDir, GuideFileName))
	}
	if executablePath != "" {
		exeDir := filepath.Dir(executablePath)
		candidates = append(candidates,
			filepath.Join(exeDir, GuideFileName),
			filepath.Join(filepath.Dir(exeDir), GuideFileName),
		)
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		cleaned := filepath.Clean(candidate)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		if info, err := os.Stat(cleaned); err == nil && !info.IsDir() {
			return cleaned, nil
		}
	}

	return "", fmt.Errorf("%s not found in working directory or next to the executable", GuideFileName)
}

func OpenGuide(workingDir, executablePath string) (string, error) {
	guidePath, err := ResolveGuidePath(workingDir, executablePath)
	if err != nil {
		return "", err
	}
	if err := openPath(guidePath); err != nil {
		return "", err
	}
	return guidePath, nil
}

func openPath(path string) error {
	switch runtime.GOOS {
	case "windows":
		return startCommand("cmd", "/C", "start", "", path)
	case "darwin":
		return startCommand("open", path)
	default:
		return startCommand("xdg-open", path)
	}
}
