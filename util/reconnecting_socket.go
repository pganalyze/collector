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
	// Channel shared with the caller for incoming messages
	Read chan []byte

	// Channel for handing outgoing messages to the writer goroutine
	// (use WriteMessage to send out messages)
	write chan socketWrite

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

// socketWrite - A single outgoing message, together with the channel used to
// hand the write's outcome back to the caller
type socketWrite struct {
	data []byte
	// Must be buffered (capacity 1), so the writer goroutine never blocks
	// handing back the result to a caller that has stopped waiting
	result chan error
}

var ErrorConnectRateLimited = errors.New("Skipping connection attempt because of previous 4XX error")
var ErrorWriteNotAccepted = errors.New("Timeout waiting for websocket connection to accept message")

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
		write:     make(chan socketWrite),
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

// WriteMessage - Sends the given data over the WebSocket, waiting for the write
// to actually complete.
//
// Returns an error if the write failed, or if no connection was available to
// take the message in time. If the context is canceled while waiting, ctx.Err()
// is returned and the message may or may not have been sent.
func (w *ReconnectingSocket) WriteMessage(ctx context.Context, data []byte) error {
	// Must be buffered with capacity 1, see socketWrite definition
	result := make(chan error, 1)

	// A healthy writer goroutine picks the message up right away. Waiting longer
	// than a single write deadline means there is no writer goroutine to take it
	// (e.g. the connection was torn down after the caller checked Connected()).
	// Give up in that case, instead of blocking until the next reconnect and then
	// sending data that is stale by then.
	timeout := time.NewTimer(socketWriteTimeout)
	defer timeout.Stop()

	select {
	case w.write <- socketWrite{data: data, result: result}:
	case <-timeout.C:
		return ErrorWriteNotAccepted
	case <-ctx.Done():
		return ctx.Err()
	}

	// Wait on the message send, bounded by the write deadline set by the writer goroutine
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
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
			case msg := <-w.write:
				conn.SetWriteDeadline(time.Now().Add(socketWriteTimeout))
				err := conn.WriteMessage(websocket.BinaryMessage, msg.data)
				msg.result <- err
				if err != nil {
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
