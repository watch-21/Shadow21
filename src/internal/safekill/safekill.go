package safekill

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	BleachBitPath = "/usr/bin/bleachbit"
	procRootPath  = "/proc"
)

var processNames = []string{
	"chrome",
	"dropbox",
	"skype",
	"icedove",
	"thunderbird",
	"firefox",
	"firefox-esr",
	"chromium",
	"xchat",
	"hexchat",
	"transmission",
	"steam",
	"firejail",
}

var cacheTargets = []string{
	"adobe_reader.cache",
	"chromium.cache",
	"chromium.session",
	"chromium.history",
	"chromium.form_history",
	"elinks.history",
	"emesene.cache",
	"epiphany.cache",
	"firefox.cache",
	"firefox.crash_reports",
	"firefox.url_history",
	"firefox.forms",
	"flash.cache",
	"flash.cookies",
	"google_chrome.cache",
	"google_chrome.history",
	"google_chrome.form_history",
	"google_chrome.search_engines",
	"google_chrome.session",
	"google_earth.temporary_files",
	"links2.history",
	"opera.cache",
	"opera.form_history",
	"opera.history",
}

func killProcesses(ctx context.Context, names []string) error {
	if err := validateProcessNamespace(); err != nil {
		return err
	}
	type target struct {
		pid     int
		start   string
		process *os.Process
	}
	var pending []target
	defer func() {
		for _, p := range pending {
			p.process.Release()
		}
	}()
	nameTargets := make(map[string]bool)
	for _, target := range names {
		nameTargets[target] = true
	}

	entries, err := os.ReadDir(procRootPath)
	if err != nil {
		return fmt.Errorf("read process table: %w", err)
	}

	euid := os.Geteuid()
	var firstFailure error
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || pid == os.Getpid() {
			continue
		}

		processDir := filepath.Join(procRootPath, entry.Name())
		info, err := os.Stat(processDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			if firstFailure == nil {
				firstFailure = fmt.Errorf("inspect process %d: %w", pid, err)
			}
			continue
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			if firstFailure == nil {
				firstFailure = fmt.Errorf("inspect process %d owner", pid)
			}
			continue
		}
		if int(stat.Uid) != euid {
			continue
		}

		comm, err := os.ReadFile(filepath.Join(processDir, "comm"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			if firstFailure == nil {
				firstFailure = fmt.Errorf("read process %d name: %w", pid, err)
			}
			continue
		}
		processName := strings.TrimSpace(string(comm))

		if !nameTargets[processName] {
			continue
		}

		stamp, err := processStamp(processDir)
		if err != nil {
			continue
		}
		process, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		// A process handle and start stamp prevent a later PID reuse from
		// becoming evidence that the selected application is still alive.
		current, err := processStamp(processDir)
		if err != nil || stamp != current {
			process.Release()
			continue
		}
		err = process.Signal(syscall.SIGTERM)
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			process.Release()
			continue
		}
		if err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, syscall.ESRCH) {
			process.Release()
			if firstFailure == nil {
				firstFailure = fmt.Errorf("terminate selected application: %w", err)
			}
			continue
		}
		pending = append(pending, target{pid, stamp, process})
	}
	if firstFailure != nil {
		return firstFailure
	}
	waitCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	for {
		alive := false
		for _, p := range pending {
			stamp, err := processStamp(filepath.Join(procRootPath, strconv.Itoa(p.pid)))
			if err == nil && stamp == p.start {
				alive = true
				break
			}
		}
		if !alive {
			return nil
		}
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("selected applications did not finish closing; cache cleanup was not started: %w", waitCtx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func validateProcessNamespace() error {
	b, err := os.ReadFile(filepath.Join(procRootPath, "self", "stat"))
	if err != nil {
		return err
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return fmt.Errorf("process namespace is unavailable")
	}
	pid, err := strconv.Atoi(fields[0])
	if err != nil || pid != os.Getpid() {
		return fmt.Errorf("process namespace does not match the mounted process table; cleanup refused")
	}
	return nil
}
func processStamp(dir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(dir, "stat"))
	if err != nil {
		return "", err
	}
	end := strings.LastIndex(string(b), ") ")
	if end < 0 {
		return "", fmt.Errorf("invalid process status")
	}
	fields := strings.Fields(string(b)[end+2:])
	if len(fields) < 20 {
		return "", fmt.Errorf("incomplete process status")
	}
	if fields[0] == "Z" || fields[0] == "X" {
		return "", os.ErrProcessDone
	}
	return fields[19], nil
}

type Runner struct{}

func (Runner) Run() error {
	// Cleanup must operate on the desktop account, never another user's apps
	// or root's browser profiles.
	if os.Geteuid() == 0 {
		return fmt.Errorf("run SafeKill as the desktop user, without sudo")
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, 25*time.Second)
	defer cancel()
	return cleanupPipeline(ctx, runCleaner, func(ctx context.Context) error { return killProcesses(ctx, processNames) })
}
func cleanupPipeline(ctx context.Context, execute func(context.Context, ...string) (string, error), terminate func(context.Context) error) error {
	listCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	listed, err := execute(listCtx, "--list-cleaners")
	cancel()
	if err != nil {
		return fmt.Errorf("check BleachBit before closing applications: %w", err)
	}
	available := make(map[string]bool)
	for _, name := range strings.Fields(listed) {
		available[name] = true
	}
	args := []string{"--clean"}
	for _, name := range cacheTargets {
		if available[name] {
			args = append(args, name)
		}
	}
	if len(args) == 1 {
		return fmt.Errorf("no supported BleachBit cleanup targets are available")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = terminate(ctx); err != nil {
		return fmt.Errorf("close selected applications: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if _, err = execute(ctx, args...); err != nil {
		return fmt.Errorf("clear application caches: %w", err)
	}
	return nil
}

// A child process group bounds cleanup descendants as well as the immediate
// cleaner. Raw cleaner output can contain personal file paths and is discarded.
func runCleaner(ctx context.Context, args ...string) (string, error) {
	return cleanerCommand(ctx, BleachBitPath, args...)
}
func cleanerCommand(ctx context.Context, path string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = time.Second
	var output bytes.Buffer
	if len(args) == 1 && args[0] == "--list-cleaners" {
		cmd.Stdout = &boundedOutput{buffer: &output, limit: 1 << 20}
	} else {
		cmd.Stdout = io.Discard
	}
	cmd.Stderr = io.Discard
	err := cmd.Run()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		return "", err
	}
	return output.String(), nil
}

type boundedOutput struct {
	buffer *bytes.Buffer
	limit  int
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.buffer.Len()+len(p) > b.limit {
		return 0, fmt.Errorf("cleaner list exceeds limit")
	}
	return b.buffer.Write(p)
}
