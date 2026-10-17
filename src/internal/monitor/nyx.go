package monitor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

const (
	NyxPath       = "/usr/bin/nyx"
	NyxConfigPath = "/etc/shadow21/nyxrc"
)

type NyxMonitor struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func NewNyxMonitor(stdin io.Reader, stdout, stderr io.Writer) NyxMonitor {
	return NyxMonitor{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
}

func (m NyxMonitor) Run() error {
	stdin := m.stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := m.stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := m.stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	cmd := exec.Command(NyxPath, "--config", NyxConfigPath)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run nyx monitor: %w", err)
	}
	return nil
}
