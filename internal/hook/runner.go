package hook

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func Run(commands []string, env map[string]string, out io.Writer) error {
	for _, command := range commands {
		if err := runCommand(command, env, out); err != nil {
			return err
		}
	}
	return nil
}

func runCommand(command string, env map[string]string, out io.Writer) error {
	if command == "" {
		return nil
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run hook %q: %w", command, err)
	}
	return nil
}
