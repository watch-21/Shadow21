package proxymode

import (
	"strings"
	"testing"
)

func TestAttachmentWithoutCallerTERM(t *testing.T) {
	t.Setenv("TERM", "")
	c := attachmentCommand("/run/shadow21/tor.sock", "tor")
	if !strings.Contains(strings.Join(c.Env, "\n"), "TERM=xterm-256color") {
		t.Fatal("pkexec environment must not leave tmux without a terminal type")
	}
	if strings.Join(c.Args[1:], " ") != "-N -f /dev/null -S /run/shadow21/tor.sock attach-session -t =tor" {
		t.Fatal("attachment must target the owned dedicated socket and exact session")
	}
}
