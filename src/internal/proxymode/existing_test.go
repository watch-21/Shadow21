package proxymode

import (
	"context"
	"errors"
	"gitlab.com/parrotsec/packages/anonsurf/internal/netcheck"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExistingTorIsNotReconfiguredOrStopped(t *testing.T) {
	f := setup(t)
	f.m.Existing = func(context.Context) (string, error) { return "tor@default.service", nil }
	f.m.Ready = func(context.Context) error { t.Fatal("owned readiness used for borrowed Tor"); return nil }
	before, _ := os.ReadFile(f.m.ConfigPath)
	if e := f.m.Start(context.Background(), 1000, "tor"); e != nil {
		t.Fatal(e)
	}
	s, e := f.m.read()
	if e != nil || s.TorStarted || s.ConfigChanged || s.ExistingTorUnit != "tor@default.service" {
		t.Fatal(s, e)
	}
	if e = f.m.Stop(context.Background(), 1000, "tor"); e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(f.m.ConfigPath)
	if string(before) != string(after) {
		t.Fatal("changed borrowed configuration")
	}
	calls := strings.Join(f.calls, "\n")
	if strings.Contains(calls, "systemctl stop tor@default.service") || strings.Contains(calls, "--unit="+TorUnit) || strings.Contains(calls, "--verify-config") {
		t.Fatal(calls)
	}
}
func TestUnverifiedExistingProxyCreatesNoState(t *testing.T) {
	f := setup(t)
	f.m.Existing = func(context.Context) (string, error) { return "", errors.New("unverified listener") }
	if f.m.Start(context.Background(), 1000, "tor") == nil {
		t.Fatal("accepted unverified listener")
	}
	if _, e := os.Stat(f.m.statePath()); !os.IsNotExist(e) {
		t.Fatal(e)
	}
	if len(f.calls) != 0 {
		t.Fatal(f.calls)
	}
}
func TestBorrowedProxyFailureCleansOnlyOwnedSession(t *testing.T) {
	f := setup(t)
	f.m.Existing = func(context.Context) (string, error) { return "tor.service", nil }
	f.fail = "new-session"
	if f.m.Start(context.Background(), 1000, "tor") == nil {
		t.Fatal("ignored failure")
	}
	if strings.Contains(strings.Join(f.calls, "\n"), "systemctl stop tor.service") {
		t.Fatal("stopped borrowed Tor")
	}
	if _, e := os.Stat(f.m.statePath()); !os.IsNotExist(e) {
		t.Fatal("rollback incomplete", e)
	}
}
func TestListenerIdentityParsing(t *testing.T) {
	valid := `LISTEN 0 4096 127.0.0.1:9050 0.0.0.0:* users:(("tor",pid=123,fd=6))`
	if pid, e := listenerPID(valid); e != nil || pid != 123 {
		t.Fatal(pid, e)
	}
	for _, bad := range []string{strings.Replace(valid, "127.0.0.1:9050", "0.0.0.0:9050", 1), strings.Replace(valid, `"tor"`, `"python3"`, 1), valid + "\n" + valid, strings.Replace(valid, "pid=123", "pid=0", 1), ""} {
		if _, e := listenerPID(bad); e == nil {
			t.Fatal("accepted unsafe listener", bad)
		}
	}
}

func TestStopClearsCompletedPublicStatusAfterJournalRemoval(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.m.Start(ctx, 1000, "tor"); e != nil {
		t.Fatal(e)
	}
	status, _ := os.ReadFile(f.m.PublicPath)
	if e := f.m.Stop(ctx, 1000, "tor"); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(f.m.PublicPath, status, 0644); e != nil {
		t.Fatal(e)
	}
	if e := f.m.Stop(ctx, 1000, "tor"); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(f.m.PublicPath); !os.IsNotExist(e) {
		t.Fatal("stale public status was not removed", e)
	}
}

func TestExistingUnitAllowlist(t *testing.T) {
	for _, u := range []string{"", "tor.service", "tor@default.service"} {
		if e := ValidateExistingUnit(u); e != nil {
			t.Fatal(e)
		}
	}
	for _, u := range []string{"unrelated.service", "--all", "tor@other.service", "tor.service\n"} {
		if ValidateExistingUnit(u) == nil {
			t.Fatal("accepted arbitrary unit", u)
		}
	}
}

func TestExistingReadinessRetriesAndKeepsCause(t *testing.T) {
	calls := 0
	err := waitExistingReadiness(context.Background(), func(context.Context) (netcheck.Result, error) {
		calls++
		if calls < 3 {
			return netcheck.Result{}, errors.New("SOCKS destination timeout")
		}
		return netcheck.Result{IsTor: true}, nil
	}, 0)
	if err != nil || calls != 3 {
		t.Fatal(err, calls)
	}
	calls = 0
	cause := errors.New("TLS connection timed out")
	err = waitExistingReadiness(context.Background(), func(context.Context) (netcheck.Result, error) { calls++; return netcheck.Result{}, cause }, 0)
	var typed *ProxyReadinessError
	if !errors.As(err, &typed) || !errors.Is(err, cause) || calls != 3 || !strings.Contains(err.Error(), cause.Error()) {
		t.Fatal(err, calls)
	}
}
func TestExistingReadinessRejectsDirectResultWithoutRetry(t *testing.T) {
	calls := 0
	err := waitExistingReadiness(context.Background(), func(context.Context) (netcheck.Result, error) { calls++; return netcheck.Result{IsTor: false}, nil }, 0)
	if err == nil || calls != 1 {
		t.Fatal(err, calls)
	}
}
func TestExistingReadinessCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := waitExistingReadiness(ctx, func(context.Context) (netcheck.Result, error) {
		calls++
		cancel()
		return netcheck.Result{}, errors.New("connection pending")
	}, time.Hour)
	if !errors.Is(err, context.Canceled) || calls != 1 || !strings.Contains(err.Error(), "connection pending") {
		t.Fatal(err, calls)
	}
}

func TestStopStillCleansOwnedServiceWhenTmuxCommandTimesOut(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.m.Existing = func(context.Context) (string, error) { return "tor@default.service", nil }
	if err := f.m.Start(ctx, 1000, "tor"); err != nil {
		t.Fatal(err)
	}
	execute := f.m.Execute
	f.m.Execute = func(ctx context.Context, program string, a ...string) ([]byte, error) {
		if program == "tmux" && strings.Contains(strings.Join(a, " "), "kill-session") {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return execute(ctx, program, a...)
	}
	started := time.Now()
	if err := f.m.Stop(ctx, 1000, "tor"); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) > 5*time.Second {
		t.Fatal("owned stop did not bound the stalled tmux command")
	}
	if strings.Contains(strings.Join(f.calls, "\n"), "systemctl stop tor@default.service") {
		t.Fatal("borrowed Tor service was stopped")
	}
}
