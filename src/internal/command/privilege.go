package command

import (
	"os"
)

const SudoPath = "/usr/bin/sudo"

func IsAdmin() bool {
	return os.Geteuid() == 0
}

func PrivilegedCommand(command string, args ...string) (string, []string) {
	if IsAdmin() {
		return command, args
	}
	return SudoPath, append([]string{command}, args...)
}
