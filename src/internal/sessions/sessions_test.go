package sessions

import (
	"context"
	"encoding/json"
	"gitlab.com/parrotsec/packages/anonsurf/internal/preferences"
	"gitlab.com/parrotsec/packages/anonsurf/internal/proxymode"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDisabledDoesNotExecute(t *testing.T) {
	m := New(preferences.Default())
	m.Execute = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("executed disabled start")
		return nil, nil
	}
	if m.Start(context.Background(), "shadow21-work") == nil {
		t.Fatal("disabled accepted")
	}
}
func TestOwnedActionsOnly(t *testing.T) {
	for _, n := range []string{"other", "shadow21-a;touch /tmp/a", "shadow21-", "shadow21-a.b", "shadow21-:0", "shadow21-\n"} {
		if ValidateName(n) == nil {
			t.Fatal(n)
		}
	}
	m := New(preferences.Default())
	if m.Action(context.Background(), "kill-server", "shadow21-test") == nil {
		t.Fatal("bad action")
	}
}
func TestStartUsesAuthorizedHelper(t *testing.T) {
	s := preferences.Default()
	s.TmuxEnabled = true
	s.Mode = "proxy"
	m := New(s)
	m.Execute = func(_ context.Context, p string, a ...string) ([]byte, error) {
		text := p + " " + strings.Join(a, " ")
		if !strings.Contains(text, Helper+" session-start shadow21-test") || strings.Contains(text, "kill-server") {
			t.Fatal(text)
		}
		return nil, nil
	}
	if e := m.Start(context.Background(), "shadow21-test"); e != nil {
		t.Fatal(e)
	}
}
func TestSystemSelectionBlocksSession(t *testing.T) {
	s := preferences.Default()
	s.TmuxEnabled = true
	m := New(s)
	m.Execute = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("executed with system selection")
		return nil, nil
	}
	if m.Start(context.Background(), "shadow21-test") == nil {
		t.Fatal("system selection accepted")
	}
}
func TestListUsesOwnedJournalWithoutStartingServer(t *testing.T) {
	m := New(preferences.Default())
	m.UID = 1000
	m.PublicPath = filepath.Join(t.TempDir(), "status.json")
	b, _ := json.Marshal(proxymode.Public{Owner: 1000, Sessions: []string{"shadow21-test"}, Phase: "active", Mode: "proxy"})
	os.WriteFile(m.PublicPath, b, 0644)
	m.Execute = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("listing started server")
		return nil, nil
	}
	n, e := m.List(context.Background())
	if e != nil || len(n) != 1 {
		t.Fatal(n, e)
	}
	m.UID = 1001
	n, e = m.List(context.Background())
	if e != nil || len(n) != 0 {
		t.Fatal(n, e)
	}
}
