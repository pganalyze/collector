package selfhosted

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/papertrail/go-tail/follower"
	"github.com/pganalyze/collector/util"
)

// Verifies that tailFile reads new lines appended to a file and outputs them.
func TestTailFile_BasicLineReading(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	if err := os.WriteFile(logPath, nil, 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	linesCh := make(chan SelfHostedLogStreamItem, 10)
	if err := tailFile(ctx, logPath, linesCh, testLogger()); err != nil {
		t.Fatalf("tailFile: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(logPath, []byte("hello tail\n"), 0644); err != nil {
		t.Fatalf("append line: %v", err)
	}

	lines := drainLines(linesCh, 1*time.Second)
	if len(lines) == 0 {
		t.Error("expected at least one line, got none")
	}
}

// Verifies that when a log file is renamed (simulating logrotate), the follower detects the
// change and reads from the new file.
func TestTailFile_LogRotation(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "postgresql.log")

	if err := os.WriteFile(logPath, nil, 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	linesCh := make(chan SelfHostedLogStreamItem, 100)
	if err := tailFile(ctx, logPath, linesCh, testLogger()); err != nil {
		t.Fatalf("tailFile: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// Simulate logrotate: rename + create new file
	if err := os.Rename(logPath, logPath+".1"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("after rotation\n"), 0644); err != nil {
		t.Fatalf("create new: %v", err)
	}

	lines := drainLines(linesCh, 2*time.Second)
	found := false
	for _, line := range lines {
		if line == "after rotation" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'after rotation' after log rotation, got: %v", lines)
	}
}

// Verifies that Close() does not block when the follower's reader is stuck waiting for a newline
// (e.g. file has content without a trailing newline and was rotated).
func TestFollower_Close_NoDeadlock_BlockedRead(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	// No trailing newline, so reader blocks on ReadBytes('\n')
	if err := os.WriteFile(logPath, []byte("no newline at end"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	f, err := follower.New(logPath, follower.Config{Whence: io.SeekEnd, Offset: 0, Reopen: true})
	if err != nil {
		t.Fatalf("follower.New: %v", err)
	}

	// Rotate the file so the follower holds an fd to a now-deleted file
	if err := os.Rename(logPath, logPath+".1"); err != nil {
		t.Fatalf("rename: %v", err)
	}

	// Close must return promptly; a deadlock here means the fd leak is present
	done := make(chan struct{})
	go func() { f.Close(); close(done) }()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Close() blocked")
	}
}

// Verifies that Close() returns when the follower's sendLine goroutine is blocked on the unbuffered
// lines channel. Closing the underlying file unblocks the reader, which lets the goroutine exit.
func TestFollower_Close_NoDeadlock_SendLineBlocks(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	// A complete line will cause sendLine to block on the unbuffered channel
	if err := os.WriteFile(logPath, []byte("line to trigger sendLine\n"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	f, err := follower.New(logPath, follower.Config{Whence: io.SeekStart, Offset: 0, Reopen: true})
	if err != nil {
		t.Fatalf("follower.New: %v", err)
	}

	// Intentionally don't read from f.Lines() so sendLine blocks
	done := make(chan struct{})
	go func() { f.Close(); close(done) }()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Close() blocked - sendLine deadlock not resolved")
	}
}

func testLogger() *util.Logger {
	return &util.Logger{
		Verbose:     true,
		Destination: log.New(os.Stderr, "", 0),
	}
}

func drainLines(ch chan SelfHostedLogStreamItem, timeout time.Duration) []string {
	var lines []string
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case item, ok := <-ch:
			if !ok {
				return lines
			}
			lines = append(lines, item.Line)
		case <-timer.C:
			return lines
		}
	}
}
