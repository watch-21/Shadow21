package safekill

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCleanerPreflightAndSupportedTargets(t *testing.T) {
	var events []string
	execute := func(_ context.Context, a ...string) (string, error) {
		events = append(events, strings.Join(a, " "))
		if a[0] == "--list-cleaners" {
			return "firefox.cache\nfirefox.url_history\nunrelated.cleaner", nil
		}
		return "", nil
	}
	err := cleanupPipeline(context.Background(), execute, func(context.Context) error { events = append(events, "close"); return nil })
	if err != nil || strings.Join(events, "|") != "--list-cleaners|close|--clean firefox.cache firefox.url_history" {
		t.Fatal(events, err)
	}
}
func TestPreflightFailureNeverClosesApplications(t *testing.T) {
	for _, listed := range []string{"", "unrelated.cleaner"} {
		err := cleanupPipeline(context.Background(), func(context.Context, ...string) (string, error) { return listed, nil }, func(context.Context) error { t.Fatal("apps closed before valid cleaner preflight"); return nil })
		if err == nil {
			t.Fatal("missing supported cleaners accepted")
		}
	}
	err := cleanupPipeline(context.Background(), func(context.Context, ...string) (string, error) { return "", errors.New("not installed") }, func(context.Context) error { t.Fatal("apps closed without BleachBit"); return nil })
	if err == nil {
		t.Fatal("missing dependency ignored")
	}
}
func TestClosingFailureAndCancellationDoNotClean(t *testing.T) {
	calls := 0
	execute := func(context.Context, ...string) (string, error) {
		calls++
		if calls > 1 {
			t.Fatal("cleanup after closing failure")
		}
		return "firefox.cache", nil
	}
	if err := cleanupPipeline(context.Background(), execute, func(context.Context) error { return errors.New("still running") }); err == nil {
		t.Fatal("close failure ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls = 0
	if err := cleanupPipeline(ctx, execute, func(context.Context) error { t.Fatal("termination after cancel"); return nil }); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestCleanerTimeoutKillsOwnedChildGroup(t *testing.T) {
	dir := t.TempDir()
	pulse := filepath.Join(dir, "pulse")
	script := filepath.Join(dir, "cleaner")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n(while :; do printf x >> \"$1\"; sleep 0.02; done) &\nwait\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := cleanerCommand(ctx, script, pulse)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 3*time.Second {
		t.Fatal(err, time.Since(started))
	}
	before, err := os.Stat(pulse)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	after, err := os.Stat(pulse)
	if err != nil || before.Size() != after.Size() {
		t.Fatal("cleanup descendant survived cancellation", err)
	}
}
func TestSelectedOwnedApplicationCloses(t *testing.T) {
	if err := validateProcessNamespace(); err != nil {
		t.Skip(err)
	}
	name := fmt.Sprintf("sh21-%d", os.Getpid())
	path := filepath.Join(t.TempDir(), name)
	if err := os.Symlink("/bin/sleep", path); err != nil {
		t.Fatal(err)
	}
	child := exec.Command(path, "60")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { child.Process.Kill(); child.Wait() }()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := killProcesses(ctx, []string{name}); err != nil {
		t.Fatal(err)
	}
	if _, err := processStamp(filepath.Join("/proc", fmt.Sprint(child.Process.Pid))); err == nil {
		t.Fatal("selected application still running")
	}
}
