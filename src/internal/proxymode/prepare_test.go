package proxymode

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func prepFixture(t *testing.T) (Manager, *[]string) {
	t.Helper()
	calls := []string{}
	m := Manager{ServiceUnit: "tor@default.service", StateDir: t.TempDir(), RoutingCheck: func() error { return nil }, IdleCheck: func() error { return nil }, Preflight: func(context.Context) error { return nil }}
	m.Execute = func(_ context.Context, p string, a ...string) ([]byte, error) {
		calls = append(calls, p+" "+strings.Join(a, " "))
		if strings.Contains(strings.Join(a, " "), "MainPID") {
			return []byte("123\n"), nil
		}
		return []byte("active\n"), nil
	}
	return m, &calls
}
func TestPrepareTorServiceBeforeAnySession(t *testing.T) {
	for _, unit := range []string{"tor.service", "tor@default.service"} {
		m, calls := prepFixture(t)
		m.ServiceUnit = unit
		got, e := m.PrepareTor(context.Background())
		if e != nil || got != unit {
			t.Fatal(got, e)
		}
		if (*calls)[0] != "systemctl start tor.service" {
			t.Fatal(*calls)
		}
		if unit != "tor.service" && (*calls)[1] != "systemctl start tor@default.service" {
			t.Fatal(*calls)
		}
		for _, c := range *calls {
			if !strings.HasPrefix(c, "systemctl ") {
				t.Fatal("Unexpected process", c)
			}
		}
		if _, e := os.Stat(filepath.Join(m.StateDir, "proxy-state.json")); !os.IsNotExist(e) {
			t.Fatal("Selection created a session journal")
		}
	}
}
func TestPrepareTorGuardsAndFailure(t *testing.T) {
	for _, failure := range []string{"routing", "proxy", "preflight", "command", "state", "pid", "owned"} {
		t.Run(failure, func(t *testing.T) {
			m, calls := prepFixture(t)
			switch failure {
			case "routing":
				m.RoutingCheck = func() error { return errors.New("active routing") }
			case "proxy":
				m.IdleCheck = func() error { return errors.New("active proxy") }
			case "preflight":
				m.Preflight = func(context.Context) error { return errors.New("missing tor") }
			case "owned":
				os.WriteFile(filepath.Join(m.StateDir, "proxy-state.json"), []byte("{}"), 0600)
			default:
				old := m.Execute
				m.Execute = func(ctx context.Context, p string, a ...string) ([]byte, error) {
					text := strings.Join(a, " ")
					if failure == "command" && a[0] == "start" {
						return []byte("service failure"), errors.New("failed")
					}
					if failure == "state" && strings.Contains(text, "ActiveState") {
						return []byte("failed"), nil
					}
					if failure == "pid" && strings.Contains(text, "MainPID") {
						return []byte("0"), nil
					}
					return old(ctx, p, a...)
				}
			}
			if _, e := m.PrepareTor(context.Background()); e == nil {
				t.Fatal("Failure accepted")
			}
			if (failure == "routing" || failure == "proxy" || failure == "preflight" || failure == "owned") && len(*calls) > 0 {
				t.Fatal("Started before guards", *calls)
			}
		})
	}
}

func TestPrepareTorRefusesNonlocalConfigurationBeforeStart(t *testing.T) {
	m, calls := prepFixture(t)
	m.ConfigPath = filepath.Join(m.StateDir, "torrc")
	os.WriteFile(m.ConfigPath, []byte("SocksPort 0.0.0.0:9050\n"), 0600)
	if _, e := m.PrepareTor(context.Background()); e == nil || len(*calls) != 0 {
		t.Fatal("Unsafe config was started", e, *calls)
	}
}
func TestPrepareTorVerifiesLocalBinding(t *testing.T) {
	m, _ := prepFixture(t)
	m.BindCheck = func(context.Context) error { return errors.New("SOCKS listener is not loopback-only") }
	if _, e := m.PrepareTor(context.Background()); e == nil {
		t.Fatal("Nonlocal listener accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.BindCheck = func(context.Context) error { return errors.New("not listening yet") }
	if _, e := m.PrepareTor(ctx); e == nil {
		t.Fatal("No listener accepted")
	}
}
