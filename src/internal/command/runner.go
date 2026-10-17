package command

import (
	"bytes"
	"os"
	"os/exec"
)

type Runner struct{}

func (Runner) Run(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	_, err := runCaptured(cmd, command, args, true)
	return err
}

func (Runner) Output(command string, args ...string) ([]byte, error) {
	return runCaptured(exec.Command(command, args...), command, args, true)
}

func (Runner) QuietOutput(command string, args ...string) ([]byte, error) {
	return runCaptured(exec.Command(command, args...), command, args, false)
}

func (Runner) RunWithInput(command string, input []byte, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = bytes.NewReader(input)
	_, err := runCaptured(cmd, command, args, true)
	return err
}

func (Runner) RunInteractive(command string, args ...string) error {
	writeVerboseInvocation(command, args, nil, nil)
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCaptured(cmd *exec.Cmd, command string, args []string, verbose bool) ([]byte, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if verbose {
		writeVerboseInvocation(command, args, stdout.Bytes(), stderr.Bytes())
	}
	if err != nil {
		return stdout.Bytes(), commandError{
			err:        err,
			diagnostic: joinCommandOutput(stderr.String(), stdout.String()),
		}
	}
	return stdout.Bytes(), nil
}
