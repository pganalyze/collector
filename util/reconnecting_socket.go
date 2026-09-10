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
	start     chan connectRequest
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

// connectRequest - A single request to establish the connection, together with
// the channel used to hand the outcome back to the caller
type connectRequest struct {
	// Bounds the connection attempt made on behalf of this request
	ctx context.Context
	// Must be buffered (capacity 1), so the manager goroutine never blocks
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
		Read:     make(chan []byte),
		write:    make(chan socketWrite),
		ctx:      ctx,
		dialer:   dialer,
		url:      url,
		headers:  headers,
		logger:   logger,
		start:    make(chan connectRequest),
		shutdown: make(chan struct{}),
	}

	// Manager goroutine: serializes connection attempts and shutdowns
	go func() {
		var skipConnectUntil time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-w.start:
				var err error
				if w.Connected() || !w.requested.Load() {
					// Nothing to do
				} else if req.ctx.Err() != nil {
					// The caller gave up while waiting for an earlier attempt to finish
					err = req.ctx.Err()
				} else if time.Now().After(skipConnectUntil) {
					var connectStatus int
					connectStatus, err = w.connect(req.ctx)
					if connectStatus >= 400 && connectStatus < 500 {
						skipConnectUntil = time.Now().Add(clientErrorTimeout) // Delay reconnect when server responds with 4xx errors
					}
				} else {
					err = ErrorConnectRateLimited
				}
				req.result <- err
			case <-w.shutdown:
				w.closeConnection(w.conn.Load())
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
					w.requestConnect(ctx)
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
// The passed context bounds the connection attempt: when it is canceled, the
// attempt is abandoned and ctx.Err() is returned. This does not stop the socket
// from reconnecting in the background later on.
//
// Does nothing if the WebSocket is already connected
func (w *ReconnectingSocket) Connect(ctx context.Context) error {
	w.requested.Store(true)
	if w.Connected() {
		return nil
	}
	return w.requestConnect(ctx)
}

// requestConnect - Hands a connection request to the manager goroutine and
// waits for its outcome, giving up when either the passed context or the
// socket's own context is canceled
func (w *ReconnectingSocket) requestConnect(ctx context.Context) error {
	// Must be buffered with capacity 1, see connectRequest definition
	req := connectRequest{ctx: ctx, result: make(chan error, 1)}
	select {
	case w.start <- req:
	case <-ctx.Done():
		return ctx.Err()
	case <-w.ctx.Done():
		return w.ctx.Err()
	}
	select {
	case err := <-req.result:
		return err
	case <-ctx.Done():
		return ctx.Err()
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
		// The channel reader is stopped once the context is canceled, so we must
		// not block on the send when the collector is shutting down or reloading
		select {
		case w.shutdown <- struct{}{}:
		case <-w.ctx.Done():
		}
	}
}

// connect - Makes a single connection attempt, bounded by the passed context
// (in addition to the socket's own context and the dialer's handshake timeout)
//
// Returns the HTTP status code of the handshake response, if there was one.
func (w *ReconnectingSocket) connect(ctx context.Context) (int, error) {
	var connectStatus int

	// The dial only lives as long as the request that triggered it, but must also
	// stop when the socket as a whole is shut down
	dialCtx, cancelDial := context.WithCancel(ctx)
	defer cancelDial()
	stopAfterFunc := context.AfterFunc(w.ctx, cancelDial)
	defer stopAfterFunc()

	conn, response, err := w.dialer.DialContext(dialCtx, w.url, w.headers)
	if response != nil {
		connectStatus = response.StatusCode
	}
	if err != nil {
		if response != nil {
			w.logger.PrintWarning("Error starting websocket: %s (HTTP status %d)", err, connectStatus)
		} else {
			w.logger.PrintWarning("Error starting websocket: %s", err)
		}
		return connectStatus, err
	}
	w.conn.Store(conn)

	// The established connection is independent of the request that started it,
	// and lives until it fails or the socket is shut down
	connCtx, cancelConn := context.WithCancel(w.ctx)
	// Writer goroutine
	go func() {
		ticker := time.NewTicker(socketPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-connCtx.Done():
				w.closeConnection(conn)
				return
			case msg := <-w.write:
				conn.SetWriteDeadline(time.Now().Add(socketWriteTimeout))
				err := conn.WriteMessage(websocket.BinaryMessage, msg.data)
				msg.result <- err
				if err != nil {
					w.closeConnection(conn)
					return
				}
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(socketWriteTimeout))
				err := conn.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					w.logger.PrintWarning("Error sending websocket ping: %s", err)
					w.closeConnection(conn)
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

// closeConnection - Closes the given connection, unless it was already closed,
// or a newer connection has been established in the meantime
//
// Callers pass the connection they are working with, so a lingering goroutine
// from an earlier connection can't tear down its replacement.
func (w *ReconnectingSocket) closeConnection(conn *websocket.Conn) {
	if conn == nil || !w.conn.CompareAndSwap(conn, nil) {
		return
	}
	err := conn.Close()
	if err != nil {
		w.logger.PrintWarning("Error closing websocket: %s", err)
	}
}
