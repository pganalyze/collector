// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alloydbconn

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/alloydb/connectors/apiv1alpha/connectorspb"
	"cloud.google.com/go/alloydbconn/debug"
	"cloud.google.com/go/alloydbconn/errtype"
	"cloud.google.com/go/alloydbconn/internal/alloydb"
	"cloud.google.com/go/alloydbconn/internal/tel"
	"github.com/google/uuid"
	"golang.org/x/net/proxy"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"

	alloydbadmin "cloud.google.com/go/alloydb/apiv1alpha"
	telv2 "cloud.google.com/go/alloydbconn/internal/tel/v2"
)

const (
	// defaultTCPKeepAlive is the default keep alive value used on connections
	// to a AlloyDB instance
	defaultTCPKeepAlive = 30 * time.Second
	// serverProxyPort is the port the server-side proxy receives connections on.
	serverProxyPort = "5433"
	// ioTimeout is the maximum amount of time to wait before aborting a
	// metadata exhange
	ioTimeout = 30 * time.Second
	// metricShutdownTimeout is the maximum amount of time to wait to flush any
	// remaining metrics when the dialer closes.
	metricShutdownTimeout = 3 * time.Second
)

var (
	// ErrDialerClosed is used when a caller invokes Dial after closing the
	// Dialer.
	ErrDialerClosed = errors.New("alloydbconn: dialer is closed")
	// versionString indicates the version of this library.
	//go:embed version.txt
	versionString string
	userAgent     = "alloydb-go-connector/" + strings.TrimSpace(versionString)
)

// keyGenerator encapsulates the details of RSA key generation to provide lazy
// generation, custom keys, or a default RSA generator.
type keyGenerator struct {
	once    sync.Once
	key     *rsa.PrivateKey
	err     error
	genFunc func() (*rsa.PrivateKey, error)
}

// newKeyGenerator initializes a keyGenerator that will (in order):
// - always return the RSA key if one is provided, or
// - generate an RSA key lazily when it's requested, or
// - (default) immediately generate an RSA key as part of the initializer.
func newKeyGenerator(
	k *rsa.PrivateKey, lazy bool, genFunc func() (*rsa.PrivateKey, error),
) (*keyGenerator, error) {
	g := &keyGenerator{genFunc: genFunc}
	switch {
	case k != nil:
		// If the caller has provided a key, initialize the key and consume the
		// sync.Once now.
		g.once.Do(func() { g.key, g.err = k, nil })
	case lazy:
		// If lazy refresh is enabled, do nothing and wait for the call to
		// rsaKey.
	default:
		// If no key has been provided and lazy refresh isn't enabled, generate
		// the key and consume the sync.Once now.
		g.once.Do(func() { g.key, g.err = g.genFunc() })
	}
	return g, g.err
}

// rsaKey will generate an RSA key if one is not already cached. Otherwise, it
// will return the cached key.
func (g *keyGenerator) rsaKey() (*rsa.PrivateKey, error) {
	g.once.Do(func() { g.key, g.err = g.genFunc() })

	return g.key, g.err
}

type connectionInfoCache interface {
	ConnectionInfo(context.Context) (alloydb.ConnectionInfo, error)
	ForceRefresh()
	io.Closer
}

// monitoredCache is a wrapper around a connectionInfoCache that tracks the
// number of connections to the associated instance.
type monitoredCache struct {
	openConns *uint64
	connectionInfoCache
}

// A Dialer is used to create connections to AlloyDB instance.
//
// Use NewDialer to initialize a Dialer.
type Dialer struct {
	lock           sync.RWMutex
	cache          map[alloydb.InstanceURI]monitoredCache
	keyGenerator   *keyGenerator
	refreshTimeout time.Duration
	// closed reports if the dialer has been closed.
	closed chan struct{}

	// lazyRefresh determines what kind of caching is used for ephemeral
	// certificates. When lazyRefresh is true, the dialer will use a lazy
	// cache, refresh certificates only when a connection attempt needs a fresh
	// certificate. Otherwise, a refresh ahead cache will be used. The refresh
	// ahead cache assumes a background goroutine may run consistently.
	lazyRefresh bool

	// disableMetadataExchange is a temporary addition to help clients who
	// cannot use the metadata exchange yet. In future versions, this field
	// should be removed.
	disableMetadataExchange bool

	// disableBuiltInMetrics turns the internal metric export into a no-op.
	disableBuiltInMetrics bool

	staticConnInfo io.Reader

	client *alloydbadmin.AlloyDBAdminClient
	// clientOpts are options for all Google Cloud API clients. There should be
	// no AlloyDB-specific configuration in these options.
	clientOpts []option.ClientOption
	logger     debug.ContextLogger

	// defaultDialCfg holds the constructor level DialOptions, so that it can
	// be copied and mutated by the Dial function.
	defaultDialCfg dialCfg

	// dialerID uniquely identifies a Dialer. Used for monitoring purposes,
	// *only* when a client has configured OpenCensus exporters.
	dialerID        string
	metricsMu       sync.Mutex
	metricRecorders map[alloydb.InstanceURI]telv2.MetricRecorder

	// dialFunc is the function used to connect to the address on the named
	// network. By default it is golang.org/x/net/proxy#Dial.
	dialFunc func(cxt context.Context, network, addr string) (net.Conn, error)

	useIAMAuthN    bool
	iamTokenSource oauth2.TokenSource
	userAgent      string

	buffer *buffer
}

type nullLogger struct{}

func (nullLogger) Debugf(context.Context, string, ...any) {}

// NewDialer creates a new Dialer.
//
// Initial calls to NewDialer make take longer than normal because generation of an
// RSA keypair is performed. Calls with a WithRSAKeyPair DialOption or after a default
// RSA keypair is generated will be faster.
func NewDialer(ctx context.Context, opts ...Option) (*Dialer, error) {
	cfg := &dialerConfig{
		refreshTimeout: alloydb.RefreshTimeout,
		dialFunc:       proxy.Dial,
		logger:         nullLogger{},
		userAgents:     []string{userAgent},
	}
	for _, opt := range opts {
		opt(cfg)
		if cfg.err != nil {
			return nil, cfg.err
		}
	}
	if cfg.disableMetadataExchange && cfg.useIAMAuthN {
		return nil, errors.New("incompatible options: WithOptOutOfAdvancedConnection " +
			"check cannot be used with WithIAMAuthN")
	}
	userAgent := strings.Join(cfg.userAgents, " ")
	// Add user agent to the end to make sure it's not overridden.
	cfg.clientOpts = append(cfg.clientOpts, option.WithUserAgent(userAgent))

	// If no token source is configured, use ADC's token source.
	ts := cfg.tokenSource
	if ts == nil {
		var err error
		ts, err = google.DefaultTokenSource(ctx, CloudPlatformScope)
		if err != nil {
			return nil, err
		}
	}

	cOpts := append(cfg.alloydbClientOpts, cfg.clientOpts...)
	client, err := alloydbadmin.NewAlloyDBAdminRESTClient(ctx, cOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create AlloyDB Admin API client: %v", err)
	}

	dialCfg := dialCfg{
		ipType:       alloydb.PrivateIP,
		tcpKeepAlive: defaultTCPKeepAlive,
	}
	for _, opt := range cfg.dialOpts {
		opt(&dialCfg)
	}

	if err := tel.InitMetrics(); err != nil {
		return nil, err
	}
	dialerID := uuid.New().String()
	g, err := newKeyGenerator(cfg.rsaKey, cfg.lazyRefresh,
		func() (*rsa.PrivateKey, error) {
			return rsa.GenerateKey(rand.Reader, 2048)
		})
	if err != nil {
		return nil, err
	}
	d := &Dialer{
		closed:                  make(chan struct{}),
		cache:                   make(map[alloydb.InstanceURI]monitoredCache),
		lazyRefresh:             cfg.lazyRefresh,
		disableMetadataExchange: cfg.disableMetadataExchange,
		disableBuiltInMetrics:   cfg.disableBuiltInTelemetry,
		staticConnInfo:          cfg.staticConnInfo,
		keyGenerator:            g,
		refreshTimeout:          cfg.refreshTimeout,
		client:                  client,
		clientOpts:              cfg.clientOpts,
		logger:                  cfg.logger,
		defaultDialCfg:          dialCfg,
		dialerID:                dialerID,
		metricRecorders:         map[alloydb.InstanceURI]telv2.MetricRecorder{},
		dialFunc:                cfg.dialFunc,
		useIAMAuthN:             cfg.useIAMAuthN,
		iamTokenSource:          ts,
		userAgent:               userAgent,
		buffer:                  newBuffer(),
	}
	return d, nil
}

// metricRecorder does a lazy initialization of the metric exporter.
func (d *Dialer) metricRecorder(ctx context.Context, inst alloydb.InstanceURI) telv2.MetricRecorder {
	d.metricsMu.Lock()
	defer d.metricsMu.Unlock()
	if mr, ok := d.metricRecorders[inst]; ok {
		return mr
	}
	cfg := telv2.Config{
		Enabled:   !d.disableBuiltInMetrics,
		Version:   versionString,
		ClientID:  d.dialerID,
		ProjectID: inst.Project(),
		Location:  inst.Region(),
		Cluster:   inst.Cluster(),
		Instance:  inst.Name(),
	}
	mr := telv2.NewMetricRecorder(ctx, d.logger, cfg, d.clientOpts...)
	d.metricRecorders[inst] = mr
	return mr
}

// Dial returns a net.Conn connected to the specified AlloyDB instance. The
// instance argument must be the instance's URI, which is in the format
// projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>
func (d *Dialer) Dial(ctx context.Context, instance string, opts ...DialOption) (conn net.Conn, err error) {
	select {
	case <-d.closed:
		return nil, ErrDialerClosed
	default:
	}

	inst, err := alloydb.ParseInstURI(instance)
	if err != nil {
		return nil, err
	}
	mr := d.metricRecorder(ctx, inst)

	var (
		startTime = time.Now()
		endDial   tel.EndSpanFunc
		attrs     = telv2.Attributes{
			IAMAuthN:    d.useIAMAuthN,
			UserAgent:   d.userAgent,
			RefreshType: telv2.RefreshAheadType,
		}
	)
	if d.lazyRefresh {
		attrs.RefreshType = telv2.RefreshLazyType
	}
	ctx, endDial = tel.StartSpan(ctx, "cloud.google.com/go/alloydbconn.Dial",
		tel.AddInstanceName(instance),
		tel.AddDialerID(d.dialerID),
	)
	defer func() {
		go tel.RecordDialError(context.Background(), instance, d.dialerID, err)
		go mr.RecordDialCount(ctx, attrs)
		endDial(err)
	}()

	cfg := d.defaultDialCfg
	for _, opt := range opts {
		opt(&cfg)
	}

	var endInfo tel.EndSpanFunc
	ctx, endInfo = tel.StartSpan(ctx, "cloud.google.com/go/alloydbconn/internal.InstanceInfo")
	cache, cacheHit, err := d.connectionInfoCache(ctx, inst, mr)
	attrs.CacheHit = cacheHit
	if err != nil {
		attrs.DialStatus = telv2.DialCacheError
		endInfo(err)
		return nil, err
	}
	ci, err := cache.ConnectionInfo(ctx)
	if err != nil {
		attrs.DialStatus = telv2.DialCacheError
		d.removeCached(ctx, inst, cache, err)
		endInfo(err)
		return nil, err
	}
	endInfo(err)

	// If the client certificate has expired (as when the computer goes to
	// sleep, and the refresh cycle cannot run), force a refresh immediately.
	// The TLS handshake will not fail on an expired client certificate. It's
	// not until the first read where the client cert error will be surfaced.
	// So check that the certificate is valid before proceeding.
	if invalidClientCert(ctx, inst, d.logger, ci.Expiration) {
		d.logger.Debugf(ctx, "[%v] Refreshing certificate now", inst.String())
		cache.ForceRefresh()
		// Block on refreshed connection info
		ci, err = cache.ConnectionInfo(ctx)
		if err != nil {
			d.removeCached(ctx, inst, cache, err)
			attrs.DialStatus = telv2.DialCacheError
			return nil, err
		}
	}
	addr, ok := ci.IPAddrs[cfg.ipType]
	if !ok {
		d.removeCached(ctx, inst, cache, err)
		err := errtype.NewConfigError(
			fmt.Sprintf("instance does not have IP of type %q", cfg.ipType),
			inst.String(),
		)
		attrs.DialStatus = telv2.DialUserError
		return nil, err
	}

	var connectEnd tel.EndSpanFunc
	ctx, connectEnd = tel.StartSpan(ctx, "cloud.google.com/go/alloydbconn/internal.Connect")
	defer func() { connectEnd(err) }()
	hostPort := net.JoinHostPort(addr, serverProxyPort)
	f := d.dialFunc
	if cfg.dialFunc != nil {
		f = cfg.dialFunc
	}
	d.logger.Debugf(ctx, "[%v] Dialing %v", inst.String(), hostPort)
	conn, err = f(ctx, "tcp", hostPort)
	if err != nil {
		d.logger.Debugf(ctx, "[%v] Dialing %v failed: %v", inst.String(), hostPort, err)
		// refresh the instance info in case it caused the connection failure
		cache.ForceRefresh()
		attrs.DialStatus = telv2.DialTCPError
		return nil, errtype.NewDialError("failed to dial", inst.String(), err)
	}
	if c, ok := conn.(*net.TCPConn); ok {
		if err := c.SetKeepAlive(true); err != nil {
			attrs.DialStatus = telv2.DialTCPError
			return nil, errtype.NewDialError("failed to set keep-alive", inst.String(), err)
		}
		if err := c.SetKeepAlivePeriod(cfg.tcpKeepAlive); err != nil {
			attrs.DialStatus = telv2.DialTCPError
			return nil, errtype.NewDialError("failed to set keep-alive period", inst.String(), err)
		}
	}

	c := &tls.Config{
		Certificates: []tls.Certificate{ci.ClientCert},
		RootCAs:      ci.RootCAs,
		// The PSC, private, and public IP all appear in the certificate as
		// SAN. Use the server name that corresponds to the requested
		// connection path.
		ServerName: addr,
		MinVersion: tls.VersionTLS13,
	}
	tlsConn := tls.Client(conn, c)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		d.logger.Debugf(ctx, "[%v] TLS handshake failed: %v", inst.String(), err)
		// refresh the instance info in case it caused the handshake failure
		cache.ForceRefresh()
		_ = tlsConn.Close() // best effort close attempt
		attrs.DialStatus = telv2.DialTLSError
		return nil, errtype.NewDialError("handshake failed", inst.String(), err)
	}

	if !d.disableMetadataExchange {
		// The metadata exchange must occur after the TLS connection is established
		// to avoid leaking sensitive information.
		err = d.metadataExchange(tlsConn)
		if err != nil {
			_ = tlsConn.Close() // best effort close attempt
			attrs.DialStatus = telv2.DialMDXError
			return nil, err
		}
	}
	attrs.DialStatus = telv2.DialSuccess

	latency := time.Since(startTime).Milliseconds()
	go func() {
		n := atomic.AddUint64(cache.openConns, 1)
		tel.RecordOpenConnections(ctx, int64(n), d.dialerID, inst.String())
		tel.RecordDialLatency(ctx, instance, d.dialerID, latency)
		mr.RecordOpenConnection(ctx, attrs)
		mr.RecordDialLatency(ctx, latency, attrs)
	}()

	return newInstrumentedConn(tlsConn, mr, attrs, func() {
		n := atomic.AddUint64(cache.openConns, ^uint64(0))
		tel.RecordOpenConnections(context.Background(), int64(n), d.dialerID, inst.String())
		mr.RecordClosedConnection(context.Background(), attrs)
	}, d.dialerID, inst.String()), nil
}

// removeCached stops all background refreshes and deletes the connection
// info cache from the map of caches.
func (d *Dialer) removeCached(
	ctx context.Context,
	i alloydb.InstanceURI, c connectionInfoCache, err error,
) {
	d.logger.Debugf(
		ctx,
		"[%v] Removing connection info from cache: %v",
		i.String(),
		err,
	)
	d.lock.Lock()
	defer d.lock.Unlock()
	c.Close()
	delete(d.cache, i)
}

func invalidClientCert(
	ctx context.Context,
	inst alloydb.InstanceURI, l debug.ContextLogger, expiration time.Time,
) bool {
	now := time.Now().UTC()
	notAfter := expiration.UTC()
	invalid := now.After(notAfter)
	l.Debugf(
		ctx,
		"[%v] Now = %v, Current cert expiration = %v",
		inst.String(),
		now.Format(time.RFC3339),
		notAfter.Format(time.RFC3339),
	)
	l.Debugf(ctx, "[%v] Cert is valid = %v", inst.String(), !invalid)
	return invalid
}

// metadataExchange sends metadata about the connection prior to the database
// protocol taking over. The exchange consists of four steps:
//
//  1. Prepare a MetadataExchangeRequest including the IAM Principal's OAuth2
//     token, the user agent, and the requested authentication type.
//
//  2. Write the size of the message as a big endian uint32 (4 bytes) to the
//     server followed by the marshaled message. The length does not include the
//     initial four bytes.
//
//  3. Read a big endian uint32 (4 bytes) from the server. This is the
//     MetadataExchangeResponse message length and does not include the initial
//     four bytes.
//
//  4. Unmarshal the response using the message length in step 3. If the
//     response is not OK, return the response's error. If there is no error, the
//     metadata exchange has succeeded and the connection is complete.
//
// Subsequent interactions with the server use the database protocol.
func (d *Dialer) metadataExchange(conn net.Conn) error {
	tok, err := d.iamTokenSource.Token()
	if err != nil {
		return err
	}
	authType := connectorspb.MetadataExchangeRequest_DB_NATIVE
	if d.useIAMAuthN {
		authType = connectorspb.MetadataExchangeRequest_AUTO_IAM
	}
	req := &connectorspb.MetadataExchangeRequest{
		UserAgent:   d.userAgent,
		AuthType:    authType,
		Oauth2Token: tok.AccessToken,
	}
	m, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	b := d.buffer.get()
	defer d.buffer.put(b)

	buf := *b
	reqSize := proto.Size(req)
	binary.BigEndian.PutUint32(buf, uint32(reqSize))
	buf = append(buf[:4], m...)

	// Set IO deadline before write
	err = conn.SetDeadline(time.Now().Add(ioTimeout))
	if err != nil {
		return err
	}
	defer conn.SetDeadline(time.Time{})

	_, err = conn.Write(buf)
	if err != nil {
		return err
	}

	// Reset IO deadline before read
	err = conn.SetDeadline(time.Now().Add(ioTimeout))
	if err != nil {
		return err
	}
	defer conn.SetDeadline(time.Time{})

	buf = buf[:4]
	_, err = conn.Read(buf)
	if err != nil {
		return err
	}

	respSize := binary.BigEndian.Uint32(buf)
	resp := buf[:respSize]
	_, err = conn.Read(resp)
	if err != nil {
		return err
	}

	var mdxResp connectorspb.MetadataExchangeResponse
	err = proto.Unmarshal(resp, &mdxResp)
	if err != nil {
		return err
	}

	if mdxResp.GetResponseCode() != connectorspb.MetadataExchangeResponse_OK {
		return errors.New(mdxResp.GetError())
	}

	return nil
}

const maxMessageSize = 16 * 1024 // 16 kb

type buffer struct {
	pool sync.Pool
}

func newBuffer() *buffer {
	return &buffer{
		pool: sync.Pool{
			New: func() any {
				buf := make([]byte, maxMessageSize)
				return &buf
			},
		},
	}
}

func (b *buffer) get() *[]byte {
	return b.pool.Get().(*[]byte)
}

func (b *buffer) put(buf *[]byte) {
	b.pool.Put(buf)
}

// newInstrumentedConn initializes an instrumentedConn that on closing will
// decrement the number of open connects and record the result.
func newInstrumentedConn(conn net.Conn, mr telv2.MetricRecorder, a telv2.Attributes, closeFunc func(), dialerID, instance string) *instrumentedConn {
	return &instrumentedConn{
		Conn:           conn,
		closeFunc:      closeFunc,
		dialerID:       dialerID,
		instance:       instance,
		metricRecorder: mr,
		attrs:          a,
	}
}

// instrumentedConn wraps a net.Conn and invokes closeFunc when the connection
// is closed.
type instrumentedConn struct {
	net.Conn
	closeFunc      func()
	dialerID       string
	instance       string
	metricRecorder telv2.MetricRecorder
	attrs          telv2.Attributes
}

// Read delegates to the underlying net.Conn interface and records number of
// bytes read.
func (i *instrumentedConn) Read(b []byte) (int, error) {
	bytesRead, err := i.Conn.Read(b)
	if err == nil {
		go tel.RecordBytesReceived(context.Background(), int64(bytesRead), i.instance, i.dialerID)
		go i.metricRecorder.RecordBytesRxCount(context.Background(), int64(bytesRead), i.attrs)
	}
	return bytesRead, err
}

// Write delegates to the underlying net.Conn interface and records number of
// bytes written.
func (i *instrumentedConn) Write(b []byte) (int, error) {
	bytesWritten, err := i.Conn.Write(b)
	if err == nil {
		go tel.RecordBytesSent(context.Background(), int64(bytesWritten), i.instance, i.dialerID)
		go i.metricRecorder.RecordBytesTxCount(context.Background(), int64(bytesWritten), i.attrs)
	}
	return bytesWritten, err
}

// Close delegates to the underlying net.Conn interface and reports the close
// to the provided closeFunc only when Close returns no error.
func (i *instrumentedConn) Close() error {
	err := i.Conn.Close()
	if err != nil {
		return err
	}
	go i.closeFunc()
	return nil
}

// Close closes the Dialer; it prevents the Dialer from refreshing the information
// needed to connect.
func (d *Dialer) Close() error {
	// Check if Close has already been called.
	select {
	case <-d.closed:
		return nil
	default:
	}
	close(d.closed)

	d.lock.Lock()
	for _, i := range d.cache {
		_ = i.Close()
	}
	d.lock.Unlock()

	d.metricsMu.Lock()
	ctx, cancel := context.WithTimeout(context.Background(), metricShutdownTimeout)
	defer cancel()
	for _, mr := range d.metricRecorders {
		// If a metric recorder doesn't shutdown cleanly, log the error and
		// keep going. An error here isn't actionable and should not be
		// returned to the caller.
		if err := mr.Shutdown(ctx); err != nil {
			d.logger.Debugf(context.Background(), "internal metric exporter failed to shutdown: %v", err)
		}
	}
	d.metricsMu.Unlock()
	return nil

}

func (d *Dialer) connectionInfoCache(ctx context.Context, uri alloydb.InstanceURI, mr telv2.MetricRecorder) (monitoredCache, bool, error) {
	d.lock.RLock()
	c, ok := d.cache[uri]
	d.lock.RUnlock()
	if !ok {
		d.lock.Lock()
		defer d.lock.Unlock()
		// Recheck to ensure instance wasn't created between locks
		c, ok = d.cache[uri]
		if !ok {
			d.logger.Debugf(ctx, "[%v] Connection info added to cache", uri.String())
			k, err := d.keyGenerator.rsaKey()
			if err != nil {
				return monitoredCache{}, ok, err
			}
			var cache connectionInfoCache
			switch {
			case d.lazyRefresh:
				cache = alloydb.NewLazyRefreshCache(
					uri,
					d.logger,
					d.client, k,
					d.refreshTimeout, d.dialerID,
					d.disableMetadataExchange,
					d.userAgent,
					mr,
				)
			case d.staticConnInfo != nil:
				var err error
				cache, err = alloydb.NewStaticConnectionInfoCache(
					uri,
					d.logger,
					d.staticConnInfo,
				)
				if err != nil {
					return monitoredCache{}, ok, err
				}
			default:
				cache = alloydb.NewRefreshAheadCache(
					uri,
					d.logger,
					d.client, k,
					d.refreshTimeout, d.dialerID,
					d.disableMetadataExchange,
					d.userAgent,
					mr,
				)
			}
			var open uint64
			c = monitoredCache{openConns: &open, connectionInfoCache: cache}
			d.cache[uri] = c
		}
	}
	return c, ok, nil
}
