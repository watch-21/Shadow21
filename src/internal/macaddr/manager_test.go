package macaddr

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const id = "11111111-2222-4333-8444-555555555555"
const boot = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"

type fixture struct {
	m                    Manager
	policy, active, kind string
	calls                []string
	failUp               bool
}

func setup(t *testing.T) *fixture {
	t.Helper()
	d := t.TempDir()
	f := &fixture{policy: "preserve", active: id, kind: "802-3-ethernet"}
	f.m = Manager{Dir: filepath.Join(d, "state"), Sys: filepath.Join(d, "net"), BootPath: filepath.Join(d, "boot"), Guard: func(context.Context) error { return nil }, Random: func() (string, error) { return "02:aa:bb:cc:dd:ee", nil }}
	os.MkdirAll(filepath.Join(f.m.Sys, "eth0", "device"), 0700)
	os.WriteFile(filepath.Join(f.m.Sys, "eth0", "type"), []byte("1\n"), 0600)
	os.WriteFile(filepath.Join(f.m.Sys, "eth0", "address"), []byte("00:11:22:33:44:55\n"), 0600)
	os.WriteFile(f.m.BootPath, []byte(boot), 0600)
	f.m.Execute = func(_ context.Context, a ...string) (string, error) {
		text := strings.Join(a, " ")
		f.calls = append(f.calls, text)
		if len(a) > 1 && a[0] == "-g" {
			switch a[1] {
			case "GENERAL.CON-UUID":
				return f.active, nil
			case "connection.type":
				return f.kind, nil
			case "connection.multi-connect":
				return "0 (default)", nil
			case "connection.master":
				return "", nil
			case "802-3-ethernet.cloned-mac-address", "802-11-wireless.cloned-mac-address":
				return f.policy, nil
			}
			return "", errors.New("unexpected query")
		}
		if strings.HasPrefix(text, "connection modify --temporary uuid ") {
			f.policy = a[len(a)-1]
			return "", nil
		}
		if strings.HasPrefix(text, "device disconnect ") {
			f.active = ""
			return "", nil
		}
		if strings.HasPrefix(text, "connection up uuid ") {
			if f.failUp {
				f.failUp = false
				return "", errors.New("injected activation failure")
			}
			f.active = id
			if validMAC(f.policy) {
				os.WriteFile(filepath.Join(f.m.Sys, "eth0", "address"), []byte(f.policy), 0600)
			}
			return "", nil
		}
		return "", errors.New("unexpected mutation")
	}
	return f
}
func TestTemporaryApplyAndExactRestore(t *testing.T) {
	for _, kind := range []string{"802-3-ethernet", "802-11-wireless"} {
		t.Run(kind, func(t *testing.T) {
			f := setup(t)
			f.kind = kind
			ctx := context.Background()
			if e := f.m.Apply(ctx, 1000, "eth0"); e != nil {
				t.Fatal(e)
			}
			s, e := f.m.read("eth0")
			if e != nil || s.Phase != "active" || s.Owner != 1000 {
				t.Fatal(s, e)
			}
			info, _ := os.Stat(f.m.path("eth0"))
			if info.Mode().Perm() != 0600 {
				t.Fatal(info.Mode())
			}
			if f.m.Apply(ctx, 1000, "eth0") == nil {
				t.Fatal("overwrote initial backup")
			}
			if f.m.Restore(ctx, 1001, "eth0") == nil {
				t.Fatal("other user restored state")
			}
			if e = f.m.Restore(ctx, 1000, "eth0"); e != nil {
				t.Fatal(e)
			}
			got, _ := f.m.address("eth0")
			if got != "00:11:22:33:44:55" || f.policy != "preserve" {
				t.Fatal(got, f.policy)
			}
			if _, e = os.Stat(f.m.path("eth0")); !os.IsNotExist(e) {
				t.Fatal(e)
			}
			for _, call := range f.calls {
				if strings.Contains(call, "connection modify ") && !strings.Contains(call, "--temporary") {
					t.Fatal("persistent profile write", call)
				}
			}
		})
	}
}
func TestActivationFailureRollsBack(t *testing.T) {
	f := setup(t)
	f.failUp = true
	if f.m.Apply(context.Background(), 1000, "eth0") == nil {
		t.Fatal("ignored failure")
	}
	got, _ := f.m.address("eth0")
	if got != "00:11:22:33:44:55" || f.policy != "preserve" {
		t.Fatal(got, f.policy)
	}
	if _, e := os.Stat(f.m.path("eth0")); !os.IsNotExist(e) {
		t.Fatal("rollback incomplete", e)
	}
}
func TestExternalPolicyAndConnectionArePreserved(t *testing.T) {
	for _, change := range []string{"policy", "connection"} {
		t.Run(change, func(t *testing.T) {
			f := setup(t)
			ctx := context.Background()
			if e := f.m.Apply(ctx, 1000, "eth0"); e != nil {
				t.Fatal(e)
			}
			if change == "policy" {
				f.policy = "02:00:00:00:00:10"
			} else {
				f.active = "99999999-2222-4333-8444-555555555555"
			}
			f.calls = nil
			if f.m.Restore(ctx, 1000, "eth0") == nil {
				t.Fatal("external edit overwritten")
			}
			for _, c := range f.calls {
				if strings.HasPrefix(c, "connection modify") || strings.HasPrefix(c, "connection up") {
					t.Fatal("mutated external state", c)
				}
			}
			s, _ := f.m.read("eth0")
			if s.Phase != "failed" {
				t.Fatal(s)
			}
		})
	}
}
func TestGuardAndInvalidInterfaceCauseNoMutation(t *testing.T) {
	f := setup(t)
	f.m.Guard = func(context.Context) error { return errors.New("active mode") }
	if f.m.Apply(context.Background(), 1000, "eth0") == nil || len(f.calls) > 0 {
		t.Fatal("guard bypass")
	}
	for _, n := range []string{"lo", "../eth0", "eth0;id", "", "--help", "eth0\n"} {
		if ValidInterface(n) {
			t.Fatal(n)
		}
	}
}
func TestRebootDropsOnlyExpiredTemporaryState(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.m.Apply(ctx, 1000, "eth0"); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(f.m.BootPath, []byte("bbbbbbbb-bbbb-4ccc-8ddd-eeeeeeeeeeee"), 0600)
	f.policy = "preserve"
	f.calls = nil
	if e := f.m.Restore(ctx, 1000, "eth0"); e != nil {
		t.Fatal(e)
	}
	for _, c := range f.calls {
		if strings.HasPrefix(c, "connection up") || strings.HasPrefix(c, "connection modify") {
			t.Fatal("replayed old runtime MAC after boot", c)
		}
	}
}
func TestInterruptedStateBlocksStartAndKeepsRecovery(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if e := f.m.Apply(ctx, 1000, "eth0"); e != nil {
		t.Fatal(e)
	}
	if e := f.m.RequireReady(); e != nil {
		t.Fatal(e)
	}
	s, _ := f.m.read("eth0")
	s.Phase = "failed"
	f.m.save(s)
	if f.m.RequireReady() == nil {
		t.Fatal("interrupted MAC accepted")
	}
	if e := f.m.RecoverInterrupted(ctx); e != nil {
		t.Fatal(e)
	}
	if e := f.m.RequireReady(); e != nil {
		t.Fatal(e)
	}
}
func TestRandomAddressIsLocalUnicast(t *testing.T) {
	for i := 0; i < 64; i++ {
		s, e := RandomMAC()
		if e != nil || !validMAC(s) {
			t.Fatal(s, e)
		}
		b, _ := net.ParseMAC(s)
		if b[0]&3 != 2 {
			t.Fatal("not locally administered unicast", s)
		}
	}
}

func TestRoutingServiceInspection(t *testing.T) {
	for _, text := range []string{"LoadState=not-found\nActiveState=inactive\n", "LoadState=loaded\nActiveState=failed\n", "ActiveState=inactive\nLoadState=loaded\n"} {
		if e := inactiveService(text); e != nil {
			t.Fatal(text, e)
		}
	}
	for _, text := range []string{"", "LoadState=not-found\nActiveState=active", "LoadState=loaded\nActiveState=activating", "LoadState=loaded\nActiveState=active", "LoadState=error\nActiveState=inactive"} {
		if inactiveService(text) == nil {
			t.Fatal("unsafe/unknown service accepted", text)
		}
	}
}
func TestMACChangeDisconnectsBeforeActivation(t *testing.T) {
	f := setup(t)
	if e := f.m.Apply(context.Background(), 1000, "eth0"); e != nil {
		t.Fatal(e)
	}
	calls := strings.Join(f.calls, "\n")
	down := strings.Index(calls, "device disconnect eth0")
	up := strings.Index(calls, "connection up uuid ")
	if down < 0 || up < down {
		t.Fatal("no explicit reconnect", calls)
	}
}

func TestUppercasePolicyRestores(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if err := f.m.Apply(ctx, 1000, "eth0"); err != nil {
		t.Fatal(err)
	}
	f.policy = strings.ToUpper(f.policy)
	if err := f.m.Restore(ctx, 1000, "eth0"); err != nil {
		t.Fatal(err)
	}
	got, _ := f.m.address("eth0")
	if got != "00:11:22:33:44:55" || f.policy != "preserve" {
		t.Fatal(got, f.policy)
	}
}
func TestRepeatedRandomizePreservesOriginal(t *testing.T) {
	for _, kind := range []string{"802-3-ethernet", "802-11-wireless"} {
		t.Run(kind, func(t *testing.T) {
			f := setup(t)
			f.kind = kind
			ctx := context.Background()
			for _, mac := range []string{"02:aa:bb:cc:dd:01", "02:aa:bb:cc:dd:02", "02:aa:bb:cc:dd:03"} {
				f.m.Random = func() (string, error) { return mac, nil }
				if err := f.m.Apply(ctx, 1000, "eth0"); err != nil {
					t.Fatal(err)
				}
				got, _ := f.m.address("eth0")
				s, err := f.m.read("eth0")
				if err != nil || got != mac || s.PreviousMAC != "00:11:22:33:44:55" || s.PreviousPolicy != "preserve" {
					t.Fatal(got, s, err)
				}
				f.policy = strings.ToUpper(f.policy)
			}
			if err := f.m.Restore(ctx, 1000, "eth0"); err != nil {
				t.Fatal(err)
			}
			got, _ := f.m.address("eth0")
			if got != "00:11:22:33:44:55" || f.policy != "preserve" {
				t.Fatal(got, f.policy)
			}
		})
	}
}
func TestRepeatedRandomizePreservesConflicts(t *testing.T) {
	for _, change := range []string{"policy", "owner", "connection", "phase"} {
		t.Run(change, func(t *testing.T) {
			f := setup(t)
			ctx := context.Background()
			if err := f.m.Apply(ctx, 1000, "eth0"); err != nil {
				t.Fatal(err)
			}
			uid := 1000
			switch change {
			case "policy":
				f.policy = "random"
			case "owner":
				uid = 1001
			case "connection":
				f.active = "99999999-2222-4333-8444-555555555555"
			case "phase":
				s, _ := f.m.read("eth0")
				s.Phase = "failed"
				f.m.save(s)
			}
			f.m.Random = func() (string, error) { return "02:aa:bb:cc:dd:02", nil }
			f.calls = nil
			if err := f.m.Apply(ctx, uid, "eth0"); err == nil {
				t.Fatal("conflict accepted")
			}
			for _, c := range f.calls {
				if strings.HasPrefix(c, "connection modify") || strings.HasPrefix(c, "device disconnect") || strings.HasPrefix(c, "connection up") {
					t.Fatal("external settings changed", c)
				}
			}
			if _, err := f.m.read("eth0"); err != nil {
				t.Fatal("journal lost", err)
			}
		})
	}
}
func TestFailedSecondChangeRollsBack(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if err := f.m.Apply(ctx, 1000, "eth0"); err != nil {
		t.Fatal(err)
	}
	count := 0
	execute := f.m.Execute
	f.m.Execute = func(ctx context.Context, a ...string) (string, error) {
		if strings.HasPrefix(strings.Join(a, " "), "connection up uuid") {
			count++
			if count == 2 {
				f.failUp = true
			}
		}
		return execute(ctx, a...)
	}
	f.m.Random = func() (string, error) { return "02:aa:bb:cc:dd:02", nil }
	if err := f.m.Apply(ctx, 1000, "eth0"); err == nil {
		t.Fatal("failure ignored")
	}
	got, _ := f.m.address("eth0")
	if got != "00:11:22:33:44:55" || f.policy != "preserve" {
		t.Fatal(got, f.policy)
	}
	if _, err := os.Stat(f.m.path("eth0")); !os.IsNotExist(err) {
		t.Fatal("rollback incomplete", err)
	}
}
