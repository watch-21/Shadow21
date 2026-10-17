package systemd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

const notifySocketEnvironment = "NOTIFY_SOCKET"

func NotifyStatus(status string) error {
	socket, enabled := os.LookupEnv(notifySocketEnvironment)
	if !enabled || socket == "" {
		return nil
	}

	status = sanitizeStatus(status)
	if status == "" {
		return errors.New("systemd status cannot be empty")
	}
	if strings.HasPrefix(socket, "@") {
		socket = "\x00" + socket[1:]
	}

	address := &net.UnixAddr{Name: socket, Net: "unixgram"}
	connection, err := net.DialUnix("unixgram", nil, address)
	if err != nil {
		return fmt.Errorf("connect to systemd notification socket: %w", err)
	}
	defer connection.Close()

	if _, err := connection.Write([]byte("STATUS=" + status)); err != nil {
		return fmt.Errorf("publish systemd status: %w", err)
	}
	return nil
}

func sanitizeStatus(status string) string {
	status = strings.ReplaceAll(status, "\r", " ")
	status = strings.ReplaceAll(status, "\n", " ")
	return strings.TrimSpace(status)
}
