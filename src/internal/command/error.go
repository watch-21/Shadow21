package command

import (
	"fmt"
	"strings"
)

type commandError struct {
	err        error
	diagnostic string
}

func (e commandError) Error() string {
	if e.diagnostic == "" {
		return e.err.Error()
	}
	indented := strings.ReplaceAll(e.diagnostic, "\n", "\n  ")
	return fmt.Sprintf("%v\n  %s", e.err, indented)
}

func (e commandError) Unwrap() error {
	return e.err
}

func joinCommandOutput(stderr, stdout string) string {
	parts := make([]string, 0, 2)
	if output := strings.TrimSpace(stderr); output != "" {
		parts = append(parts, output)
	}
	if output := strings.TrimSpace(stdout); output != "" {
		parts = append(parts, output)
	}
	return strings.Join(parts, "\n")
}
