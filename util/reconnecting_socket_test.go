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
