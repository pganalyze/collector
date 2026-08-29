package util_test

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pganalyze/collector/util"
)

// Verifies we reconnect if the server closed the connection
func TestSocketReconnect(t *testing.T) {
	var connectsReceived int32
	ctx, cancel := context.WithCancel(context.Background())

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		atomic.AddInt32(&connectsReceived, 1)
		// Close socket right away, to trigger a reconnect
		c.Close()
	})
	s := &http.Server{Addr: "localhost:9123", Handler: serverMux}
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		t.Errorf("TestSocketReconnect: failed to start socket: %v", err)
		cancel()
		return
	}
	go s.Serve(ln)

	dialer := websocket.Dialer{}
	url := "ws://localhost:9123"
	socket := util.NewReconnectingSocket(
		ctx, &util.Logger{Destination: log.New(os.Stderr, "", 0)},
		dialer, url, make(map[string][]string),
		1*time.Second,
		1*time.Second,
	)

	err = socket.Connect()
	if err != nil {
		t.Errorf("TestSocketReconnect: failed initial socket connection: %v", err)
	}

	time.Sleep(2 * time.Second)

	cancel()
	s.Shutdown(ctx)

	if atomic.LoadInt32(&connectsReceived) < 2 {
		t.Errorf("TestSocketReconnect: expected: 2+ connects; actual: %d", connectsReceived)
	}
}

// Verifies that Connect() does not block forever once the socket's context has
// been canceled (e.g. during a SIGHUP config reload). Regression test for a hang
// where the internal manager goroutine exited on its <-ctx.Done() branch while a
// Connect() call was left blocked forever on the startWait channel. That blocked
// Connect() (reached via EnsureGrant during activity collection) holds a WaitGroup
// entry, so the reload's wg.Wait() never returns and the collector wedges.
func TestSocketConnectReturnsAfterContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	socket := util.NewReconnectingSocket(
		ctx, &util.Logger{Destination: log.New(os.Stderr, "", 0)},
		websocket.Dialer{}, "ws://localhost:9124", make(map[string][]string),
		1*time.Second,
		1*time.Second,
	)

	// Cancel the context and give the internal manager goroutine time to take its
	// <-ctx.Done() branch and exit. After this, no goroutine is left to answer a
	// Connect() request.
	cancel()
	time.Sleep(100 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		done <- socket.Connect()
	}()

	select {
	case <-done:
		// Connect() returned (with or without an error) instead of hanging.
	case <-time.After(5 * time.Second):
		t.Fatal("TestSocketConnectReturnsAfterContextCancel: Connect() blocked indefinitely after context cancellation")
	}
}

// Verifies that WriteMessage() only returns once the message has actually been
// written to the socket, so callers don't report success for a write that is
// still in flight (or about to fail)
func TestSocketWriteMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	received := make(chan []byte, 1)
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		_, data, err := c.ReadMessage()
		if err != nil {
			return
		}
		received <- data
		<-ctx.Done()
	})
	s := &http.Server{Addr: "localhost:9125", Handler: serverMux}
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		t.Fatalf("TestSocketWriteMessage: failed to start socket: %v", err)
	}
	go s.Serve(ln)
	defer s.Shutdown(context.Background())

	socket := util.NewReconnectingSocket(
		ctx, &util.Logger{Destination: log.New(os.Stderr, "", 0)},
		websocket.Dialer{}, "ws://localhost:9125", make(map[string][]string),
		1*time.Second,
		1*time.Second,
	)

	err = socket.Connect()
	if err != nil {
		t.Fatalf("TestSocketWriteMessage: failed initial socket connection: %v", err)
	}

	err = socket.WriteMessage(ctx, []byte("test message"))
	if err != nil {
		t.Fatalf("TestSocketWriteMessage: unexpected write error: %v", err)
	}

	// Since WriteMessage() waits for the write to complete, the data must already
	// be on its way, and the server must see it without further delay
	select {
	case data := <-received:
		if string(data) != "test message" {
			t.Errorf("TestSocketWriteMessage: expected: %q; actual: %q", "test message", data)
		}
	case <-time.After(5 * time.Second):
		t.Error("TestSocketWriteMessage: server did not receive the message")
	}
}

// Verifies that WriteMessage() does not block forever when there is no writer
// goroutine to take the message (e.g. the connection went away after the caller
// checked Connected(), or the socket's context was canceled)
func TestSocketWriteMessageReturnsAfterContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	socket := util.NewReconnectingSocket(
		ctx, &util.Logger{Destination: log.New(os.Stderr, "", 0)},
		websocket.Dialer{}, "ws://localhost:9126", make(map[string][]string),
		1*time.Second,
		1*time.Second,
	)

	cancel()

	done := make(chan error, 1)
	go func() {
		done <- socket.WriteMessage(ctx, []byte("test message"))
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("TestSocketWriteMessageReturnsAfterContextCancel: expected an error for a message that was never written")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("TestSocketWriteMessageReturnsAfterContextCancel: WriteMessage() blocked indefinitely after context cancellation")
	}
}
