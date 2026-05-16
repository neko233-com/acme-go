package testspec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func LoadJSON(relativePath string, target any) error {
	_, filePath, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("resolve spec path: runtime caller unavailable")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filePath), "..", ".."))
	fullPath := filepath.Join(repoRoot, relativePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read spec %s: %w", relativePath, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse spec %s: %w", relativePath, err)
	}
	return nil
}
