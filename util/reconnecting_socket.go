package util

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type ReconnectingSocket struct {
	// Channels shared with the caller
	Read  chan []byte
	Write chan []byte

	// Initial arguments
	dialer  websocket.Dialer
	url     string
	headers map[string][]string
	logger  *Logger

	// Internal state
	ctx       context.Context
	requested atomic.Bool
	conn      atomic.Pointer[websocket.Conn]
	start     chan struct{}
	startWait chan error
	shutdown  chan struct{}
}

var ErrorConnectRateLimited = errors.New("Skipping connection attempt because of previous 4XX error")

// Timeouts to detect dead connections (e.g. a NAT/firewall silently dropping
// the TCP session), which would otherwise only fail after the kernel's TCP
// retransmission timeout (which can exceed 15 minutes)
const (
	// Time allowed to write a message to the peer before considering the connection dead
	socketWriteTimeout = 30 * time.Second
	// Interval at which pings are sent when the connection is otherwise idle
	socketPingInterval = 30 * time.Second
	// Time allowed to receive any message (including pong replies) from the
	// peer; must exceed socketPingInterval
	socketPongTimeout = 70 * time.Second
)

// NewReconnectingSocket - Initializes a new reconnecting WebSocket
//
// The passed context must eventually be canceled in order for internal Goroutines to be stopped.
func NewReconnectingSocket(ctx context.Context, logger *Logger, dialer websocket.Dialer, url string, headers map[string][]string, reconnectInterval time.Duration, clientErrorTimeout time.Duration) *ReconnectingSocket {
	w := &ReconnectingSocket{
		Read:      make(chan []byte),
		Write:     make(chan []byte),
		ctx:       ctx,
		dialer:    dialer,
		url:       url,
		headers:   headers,
		logger:    logger,
		start:     make(chan struct{}, 1),
		startWait: make(chan error, 1),
		shutdown:  make(chan struct{}),
	}

	go func() {
		var skipConnectUntil time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.start:
				if w.Connected() || !w.requested.Load() {
					w.startWait <- nil
				} else if time.Now().After(skipConnectUntil) {
					connectStatus, err := w.connect(ctx)
					if connectStatus >= 400 && connectStatus < 500 {
						skipConnectUntil = time.Now().Add(clientErrorTimeout) // Delay reconnect when server responds with 4xx errors
					}
					w.startWait <- err
				} else {
					w.startWait <- ErrorConnectRateLimited
				}
			case <-w.shutdown:
				if w.Connected() {
					w.closeConnection()
				}
			}
		}
	}()

	// Try reconnecting outside of requested starts in case of disconnects
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(reconnectInterval):
				if !w.Connected() && w.requested.Load() {
					select {
					case w.start <- struct{}{}:
					case <-ctx.Done():
						return
					}
					select {
					case <-w.startWait:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return w
}

func (w *ReconnectingSocket) Connected() bool {
	return w.conn.Load() != nil
}

// Connect - Blocks until connection is either established, or fails to be established
//
// Does nothing if the WebSocket is already connected
func (w *ReconnectingSocket) Connect() error {
	w.requested.Store(true)
	if w.Connected() {
		return nil
	}
	select {
	case w.start <- struct{}{}:
	case <-w.ctx.Done():
		return w.ctx.Err()
	}
	select {
	case err := <-w.startWait:
		return err
	case <-w.ctx.Done():
		return w.ctx.Err()
	}
}

// Disconnect - Shuts down the WebSocket connection
//
// Does nothing if the WebSocket is already disconnected. If needed the WebSocket
// can be started again by calling Connect() after this.
func (w *ReconnectingSocket) Disconnect() {
	w.requested.Store(false)
	if w.Connected() {
		w.shutdown <- struct{}{}
	}
}

func (w *ReconnectingSocket) connect(ctx context.Context) (int, error) {
	var connectStatus int
	connCtx, cancelConn := context.WithCancel(ctx)
	conn, response, err := w.dialer.DialContext(ctx, w.url, w.headers)
	if response != nil {
		connectStatus = response.StatusCode
	}
	if err != nil {
		cancelConn()
		w.logger.PrintWarning("Error starting websocket: %s %v", err, response)
		return 0, err
	}
	w.conn.Store(conn)
	// Writer goroutine
	go func() {
		ticker := time.NewTicker(socketPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-connCtx.Done():
				w.closeConnection()
				return
			case data := <-w.Write:
				conn.SetWriteDeadline(time.Now().Add(socketWriteTimeout))
				err := conn.WriteMessage(websocket.BinaryMessage, data)
				if err != nil {
					w.logger.PrintError("Error writing to websocket: %s", err)
					w.closeConnection()
					return
				}
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(socketWriteTimeout))
				err := conn.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					w.logger.PrintWarning("Error sending websocket ping: %s", err)
					w.closeConnection()
					return
				}
			}
		}
	}()
	// Reader goroutine
	conn.SetReadDeadline(time.Now().Add(socketPongTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(socketPongTimeout))
	})
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				serverClosed := websocket.IsCloseError(err, websocket.CloseNoStatusReceived) // The server shut down the websocket
				shutdown := errors.Is(err, net.ErrClosed)                                    // The collector process is shutting down
				if !serverClosed && !shutdown {
					w.logger.PrintWarning("Error reading from websocket: %s", err)
				}
				cancelConn()
				return
			}
			conn.SetReadDeadline(time.Now().Add(socketPongTimeout))

			w.Read <- data
		}
	}()
	return connectStatus, nil
}

func (w *ReconnectingSocket) closeConnection() {
	conn := w.conn.Swap(nil)
	if conn != nil {
		err := conn.Close()
		if err != nil {
			w.logger.PrintWarning("Error closing websocket: %s", err)
		}
	}
}
