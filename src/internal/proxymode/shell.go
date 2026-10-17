package proxymode

import (
	"os"
	"strings"
	"syscall"
)

// The root account chooses its login shell. User environment and shell text are
// never interpreted to select the executable for the privileged session.
func configuredShell(passwd string) string {
	for _, line := range strings.Split(passwd, "\n") {
		f := strings.Split(line, ":")
		if len(f) != 7 || f[0] != "root" || f[2] != "0" {
			continue
		}
		switch f[6] {
		case "/bin/bash", "/usr/bin/bash", "/bin/zsh", "/usr/bin/zsh", "/bin/sh", "/usr/bin/sh", "/bin/fish", "/usr/bin/fish":
			return f[6]
		}
	}
	return "/bin/bash"
}
func rootShell() string {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return "/bin/bash"
	}
	shell := configuredShell(string(data))
	info, err := os.Stat(shell)
	if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 && info.Mode().Perm()&0022 == 0 {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Uid == 0 {
			return shell
		}
	}
	return "/bin/bash"
}
