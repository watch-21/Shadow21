package tor

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
	"gitlab.com/parrotsec/packages/anonsurf/internal/netcheck"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var bootstrapPattern = regexp.MustCompile(`(?:^|\s)PROGRESS=([0-9]+)(?:\s|$)`)

func Bootstrap() (int, error) {
	data, e := os.ReadFile(TorrcPath)
	if e != nil {
		return -1, e
	}
	cfg, e := parseControlConfig(strings.NewReader(string(data)))
	if e != nil {
		return -1, e
	}
	return BootstrapAt("tcp", cfg.Address, cfg.CookieAuthFile)
}

func BootstrapAt(network, address, cookiePath string) (int, error) {
	cookie, e := os.ReadFile(cookiePath)
	if e != nil || len(cookie) != 32 {
		return -1, fmt.Errorf("control authentication unavailable")
	}
	conn, e := net.DialTimeout(network, address, 2*time.Second)
	if e != nil {
		return -1, e
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	reader := bufio.NewReader(conn)
	if _, e = fmt.Fprintf(conn, "AUTHENTICATE %s\r\n", hex.EncodeToString(cookie)); e != nil {
		return -1, e
	}
	if e = readControlOK(reader, "authenticate"); e != nil {
		return -1, e
	}
	if _, e = fmt.Fprint(conn, "GETINFO status/bootstrap-phase\r\n"); e != nil {
		return -1, e
	}
	progress := -1
	for i := 0; i < 16; i++ {
		line, e := reader.ReadString('\n')
		if e != nil {
			return -1, e
		}
		if !strings.HasPrefix(line, "250") {
			return -1, fmt.Errorf("bootstrap query failed")
		}
		if m := bootstrapPattern.FindStringSubmatch(line); len(m) == 2 {
			progress, _ = strconv.Atoi(m[1])
		}
		if len(line) > 3 && line[3] == ' ' {
			if progress < 0 || progress > 100 {
				return -1, fmt.Errorf("bootstrap observation missing")
			}
			return progress, nil
		}
	}
	return -1, fmt.Errorf("bootstrap response too long")
}
func WaitReady(seconds int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		if n, err := Bootstrap(); err == nil {
			b, _ := json.Marshal(map[string]any{"progress": n, "timestamp": time.Now().UTC().Format(time.RFC3339)})
			if err = fileops.WriteAtomic("/run/shadow21-bootstrap.json", append(b, '\n'), 0644); err != nil {
				return err
			}
			if n == 100 && netcheck.ProbeSOCKS(ctx, "127.0.0.1:9050") == nil {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("Tor authenticated bootstrap readiness timed out")
		case <-timer.C:
		}
	}
}
func ValidateRendered(content string) error {
	f, err := os.CreateTemp("/run/shadow21", "torrc-check-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.WriteString(content); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err = exec.CommandContext(ctx, "tor", "--verify-config", "-f", name).Run(); err != nil {
		return fmt.Errorf("Tor configuration validation failed")
	}
	return nil
}
