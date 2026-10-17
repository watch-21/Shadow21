package proxymode

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixture struct {
	m      Manager
	calls  []string
	active map[string]bool
	fail   string
}

func TestTorNamedSessionUsesOnlyOwnedSocketAndService(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if err := f.m.Start(ctx, 1000, "tor"); err != nil {
		t.Fatal(err)
	}
	if err := f.m.Stop(ctx, 1000, "tor"); err != nil {
		t.Fatal(err)
	}
	calls := strings.Join(f.calls, "\n")
	for _, expected := range []string{"-S " + f.m.socket("tor"), "new-session -d -s tor", "set-option -g exit-empty on", "-e HISTFILE=/dev/null", "kill-session -t =tor", "systemctl stop " + unit("tor")} {
		if !strings.Contains(calls, expected) {
			t.Fatalf("missing %s in %s", expected, calls)
		}
	}
	if strings.Contains(calls, "kill-server") || strings.Contains(calls, "pkill") {
		t.Fatal("unrelated process control")
	}
}

func setup(t *testing.T) *fixture {
	t.Helper()
	d := t.TempDir()
	f := &fixture{active: map[string]bool{}}
	f.m = Manager{StateDir: d, RuntimeDir: d, ConfigPath: filepath.Join(d, "torrc"), PublicPath: filepath.Join(d, "public.json"), Preflight: func(context.Context) error { return nil }, Ready: func(context.Context) error { return nil }, BindCheck: func(context.Context) error { return nil }, RoutingCheck: func() error { return nil }}
	os.WriteFile(f.m.ConfigPath, []byte("# prior config\nSafeSocks 1\n"), 0640)
	f.m.Execute = func(_ context.Context, p string, a ...string) ([]byte, error) {
		c := p + " " + strings.Join(a, " ")
		f.calls = append(f.calls, c)
		if f.fail != "" && strings.Contains(c, f.fail) {
			return []byte("injected failure"), errors.New("fail")
		}
		if p == "systemd-run" {
			for _, x := range a {
				if strings.HasPrefix(x, "--unit=") {
					f.active[strings.TrimPrefix(x, "--unit=")] = true
				}
			}
		}
		if p == "systemctl" && a[0] == "stop" {
			f.active[a[1]] = false
		}
		if p == "systemctl" && a[0] == "show" {
			if f.active[a[1]] {
				return []byte("active\n"), nil
			}
			return []byte("inactive\n"), nil
		}
		return nil, nil
	}
	return f
}
func TestStartStopRestoresConfigAndTerminatesOwnedServices(t *testing.T) {
	f := setup(t)
	before, _ := os.ReadFile(f.m.ConfigPath)
	ctx := context.Background()
	if e := f.m.Start(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	s, e := f.m.read()
	if e != nil || s.Phase != "active" || s.Owner != 1000 {
		t.Fatal(s, e)
	}
	b, _ := os.ReadFile(f.m.ConfigPath)
	if !strings.Contains(string(b), "SocksPort 9050\nSocksPolicy accept *") {
		t.Fatal(string(b))
	}
	if e = f.m.Stop(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(f.m.ConfigPath)
	if string(before) != string(after) {
		t.Fatal("config not restored")
	}
	if _, e = os.Stat(f.m.statePath()); !os.IsNotExist(e) {
		t.Fatal("state remains", e)
	}
	text := strings.Join(f.calls, "\n")
	for _, need := range []string{"--property=KillMode=control-group", "--setenv=ALL_PROXY=socks5h://127.0.0.1:9050", "kill-session -t =shadow21-test", "tmux -N -f /dev/null -S", "systemctl stop " + unit("shadow21-test"), "systemctl stop " + TorUnit} {
		if !strings.Contains(text, need) {
			t.Fatal(need)
		}
	}
	if strings.Contains(text, "kill-server") || strings.Contains(text, "pkill") {
		t.Fatal("global kill")
	}
	if strings.Index(text, "kill-session") > strings.Index(text, "systemctl stop "+unit("shadow21-test")) {
		t.Fatal("wrong shutdown order")
	}
}
func TestConfigExternalEditBlocksRestorationAndModeSwitch(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.m.Start(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(f.m.ConfigPath, []byte("# external edit\n"), 0600)
	if f.m.Stop(ctx, 1000, "shadow21-test") == nil {
		t.Fatal("overwrote external change")
	}
	b, _ := os.ReadFile(f.m.ConfigPath)
	if string(b) != "# external edit\n" {
		t.Fatal("external edit lost")
	}
	s, e := f.m.read()
	if e != nil || s.Phase != "failed" {
		t.Fatal(s, e)
	}
	if f.m.Start(ctx, 1000, "shadow21-next") == nil {
		t.Fatal("failed cleanup allowed restart")
	}
}
func TestPartialFailureRollsBack(t *testing.T) {
	for _, fail := range []string{"--verify-config", "--unit=" + TorUnit, "new-session", "has-session", "set-option"} {
		t.Run(fail, func(t *testing.T) {
			f := setup(t)
			before, _ := os.ReadFile(f.m.ConfigPath)
			f.fail = fail
			if f.m.Start(context.Background(), 1000, "shadow21-test") == nil {
				t.Fatal("failure accepted")
			}
			after, _ := os.ReadFile(f.m.ConfigPath)
			if string(before) != string(after) {
				t.Fatal("config not restored")
			}
			if _, e := os.Stat(f.m.statePath()); !os.IsNotExist(e) {
				t.Fatal("recovery state remains", e)
			}
		})
	}
}
func TestStopFailureRetainsRecoveryState(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.m.Start(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	f.fail = "systemctl stop " + unit("shadow21-test")
	if f.m.Stop(ctx, 1000, "shadow21-test") == nil {
		t.Fatal("failed stop accepted")
	}
	s, e := f.m.read()
	if e != nil || s.Phase != "failed" {
		t.Fatal(s, e)
	}
	f.fail = ""
	if e = f.m.Recover(ctx, 1000); e != nil {
		t.Fatal(e)
	}
}
func TestOwnerAndRepeatedLifecycle(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.m.Start(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	if f.m.Stop(ctx, 1001, "shadow21-test") == nil {
		t.Fatal("other owner accepted")
	}
	if f.m.Action(ctx, 1001, "detach", "shadow21-test") == nil {
		t.Fatal("unauthorized detach")
	}
	if f.m.Start(ctx, 1000, "shadow21-next") == nil {
		t.Fatal("second session accepted")
	}
	if e := f.m.Action(ctx, 1000, "detach", "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	if e := f.m.Stop(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	if e := f.m.Stop(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
}
func TestRoutingConflictDoesNotMutate(t *testing.T) {
	f := setup(t)
	f.m.RoutingCheck = func() error { return errors.New("routing active") }
	if f.m.Start(context.Background(), 1000, "shadow21-test") == nil {
		t.Fatal("conflict accepted")
	}
	if len(f.calls) != 0 {
		t.Fatal("mutated on conflict")
	}
}
func TestBindingOrReadinessFailureRollsBack(t *testing.T) {
	for _, bind := range []bool{false, true} {
		f := setup(t)
		fail := func(context.Context) error { return errors.New("not ready") }
		if bind {
			f.m.BindCheck = fail
		} else {
			f.m.Ready = fail
		}
		if f.m.Start(context.Background(), 1000, "shadow21-test") == nil {
			t.Fatal("readiness accepted")
		}
		if _, e := os.Stat(f.m.statePath()); !os.IsNotExist(e) {
			t.Fatal("state remains", e)
		}
	}
}
func TestMissingOriginalIsRestoredToAbsence(t *testing.T) {
	f := setup(t)
	os.Remove(f.m.ConfigPath)
	ctx := context.Background()
	if e := f.m.Start(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	if e := f.m.Stop(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(f.m.ConfigPath); !os.IsNotExist(e) {
		t.Fatal("created config retained")
	}
}
func TestConfigDirectivesAndConflicts(t *testing.T) {
	text, e := RenderConfig("SocksPort 9050\nSocksPolicy accept *\nSafeSocks 1\n")
	if e != nil {
		t.Fatal(e)
	}
	if strings.Count(text, "SocksPort") != 1 || strings.Count(text, "SocksPolicy accept *") != 1 {
		t.Fatal(text)
	}
	for _, x := range []string{"SocksPort 0.0.0.0:9050", "%include /tmp/custom", "HiddenServiceDir /tmp/x", "+SocksPort 9000", "SocksPort 9050 IsolateSOCKSAuth"} {
		if _, e = RenderConfig(x); e == nil {
			t.Fatal(x)
		}
	}
}
func TestLoopbackBindingVerification(t *testing.T) {
	if e := ParseBindings("LISTEN 0 128 127.0.0.1:9050 0.0.0.0:*\n"); e != nil {
		t.Fatal(e)
	}
	for _, x := range []string{"", "LISTEN 0 128 0.0.0.0:9050 0.0.0.0:*", "LISTEN 0 128 [::]:9050 [::]:*", "LISTEN 0 128 127.0.0.1:9050 0.0.0.0:*\nLISTEN 0 128 [::]:9050 [::]:*", "bad"} {
		if ParseBindings(x) == nil {
			t.Fatal(x)
		}
	}
}
func TestCanceledStartupStillRunsRollback(t *testing.T) {
	f := setup(t)
	ctx, cancel := context.WithCancel(context.Background())
	f.m.Ready = func(context.Context) error { cancel(); return context.Canceled }
	original := f.m.Execute
	f.m.Execute = func(c context.Context, p string, a ...string) ([]byte, error) {
		if c.Err() != nil {
			return nil, c.Err()
		}
		return original(c, p, a...)
	}
	if f.m.Start(ctx, 1000, "shadow21-test") == nil {
		t.Fatal("canceled start accepted")
	}
	if _, e := os.Stat(f.m.statePath()); !os.IsNotExist(e) {
		t.Fatal("rollback canceled with startup", e)
	}
}
func TestBootRecoveryHandlesAbsentTransientUnits(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.m.Start(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	f.active = map[string]bool{}
	if e := f.m.Recover(ctx, 0); e != nil {
		t.Fatal(e)
	}
}
func TestCollectedUnitsAreInactiveButPermissionErrorsRemainFailures(t *testing.T) {
	f := setup(t)
	f.m.Execute = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("Unit shadow21-proxy-tor.service could not be found."), errors.New("not found")
	}
	a, e := f.m.active(context.Background(), TorUnit)
	if e != nil || a {
		t.Fatal(a, e)
	}
	f.m.Execute = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("Access denied"), errors.New("denied")
	}
	if _, e = f.m.active(context.Background(), TorUnit); e == nil {
		t.Fatal("permission error treated as absence")
	}
}
func TestInterruptedFinalStatusCleanup(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.m.Start(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	public, _ := os.ReadFile(f.m.PublicPath)
	if e := f.m.Stop(ctx, 1000, "shadow21-test"); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(f.m.PublicPath, public, 0644)
	if f.m.Start(ctx, 1000, "shadow21-next") == nil {
		t.Fatal("stale status ignored")
	}
	if e := f.m.Recover(ctx, 1000); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(f.m.PublicPath); !os.IsNotExist(e) {
		t.Fatal("stale completed status retained")
	}
}
func TestTmuxVersionBoundary(t *testing.T) {
	for _, v := range []string{"tmux 3.2a", "tmux 3.4", "tmux 3.6"} {
		if e := ValidateTmuxVersion(v); e != nil {
			t.Fatal(v, e)
		}
	}
	for _, v := range []string{"tmux 2.9", "tmux 3.1", "unknown"} {
		if ValidateTmuxVersion(v) == nil {
			t.Fatal(v)
		}
	}
}

func TestDefaultNumericSessionHasNoExplicitName(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.m.Start(ctx, 1000, "0"); e != nil {
		t.Fatal(e)
	}
	calls := strings.Join(f.calls, "\n")
	if !strings.Contains(calls, "new-session -d -e ALL_PROXY=") || strings.Contains(calls, "new-session -d -s") {
		t.Fatal("explicit session name was supplied", calls)
	}
	if e := f.m.Stop(ctx, 1000, "0"); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(strings.Join(f.calls, "\n"), "kill-session -t =0") {
		t.Fatal("wrong stop target")
	}
}

func TestConsoleProgressDoesNotReplaceReadiness(t *testing.T) {
	f := setup(t)
	var progress []string
	f.m.Report = func(s string) { progress = append(progress, s) }
	if e := f.m.Start(context.Background(), 1000, "0"); e != nil {
		t.Fatal(e)
	}
	joined := strings.Join(progress, "\n")
	if !strings.Contains(joined, "Checking session ownership") || !strings.Contains(joined, "Waiting for Tor bootstrap") || !strings.Contains(joined, "Creating the owned tmux") {
		t.Fatal(joined)
	}
	if strings.Index(joined, "Waiting for Tor") > strings.Index(joined, "Creating the owned tmux") {
		t.Fatal("Session reported before Tor readiness")
	}
}
