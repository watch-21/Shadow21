package modes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConflictingAndInterruptedStateBlocks(t *testing.T) {
	p := filepath.Join(t.TempDir(), "marker")
	if e := RequireAbsent([]string{p}); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(p, []byte("starting"), 0600)
	if RequireAbsent([]string{p}) == nil {
		t.Fatal("unfinished operation accepted")
	}
	os.Remove(p)
	if e := RequireAbsent([]string{p}); e != nil {
		t.Fatal(e)
	}
}
func TestLockRejectsConcurrentOperations(t *testing.T) {
	p := filepath.Join(t.TempDir(), "lock")
	release, e := Lock(p)
	if e != nil {
		t.Fatal(e)
	}
	if r, e := Lock(p); e == nil {
		r()
		t.Fatal("concurrent operation accepted")
	}
	release()
	r, e := Lock(p)
	if e != nil {
		t.Fatal(e)
	}
	r()
}
