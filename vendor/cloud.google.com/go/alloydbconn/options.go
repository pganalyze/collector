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
	"crypto/rsa"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/alloydbconn/debug"
	"cloud.google.com/go/alloydbconn/errtype"
	"cloud.google.com/go/alloydbconn/internal/alloydb"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	apiopt "google.golang.org/api/option"
)

// CloudPlatformScope is the default OAuth2 scope set on the API client.
const CloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

// An Option is an option for configuring a Dialer.
type Option func(d *dialerConfig)

type dialerConfig struct {
	rsaKey *rsa.PrivateKey
	// alloydbClientOpts are options to configure only the AlloyDB Rest API
	// client. Configuration that should apply to all Google Cloud API clients
	// should be included in clientOpts.
	alloydbClientOpts []apiopt.ClientOption
	// clientOpts are options to configure any Google Cloud API client. They
	// should not include any AlloyDB-specific configuration.
	clientOpts     []apiopt.ClientOption
	dialOpts       []DialOption
	dialFunc       func(ctx context.Context, network, addr string) (net.Conn, error)
	refreshTimeout time.Duration
	tokenSource    oauth2.TokenSource
	userAgents     []string
	useIAMAuthN    bool
	logger         debug.ContextLogger
	lazyRefresh    bool

	// disableMetadataExchange is a temporary addition and will be removed in
	// future versions.
	disableMetadataExchange bool
	// disableBuiltInTelemetry disables the internal metric exporter.
	disableBuiltInTelemetry bool

	staticConnInfo io.Reader
	// err tracks any dialer options that may have failed.
	err error
}

// WithOptions turns a list of Option's into a single Option.
func WithOptions(opts ...Option) Option {
	return func(d *dialerConfig) {
		for _, opt := range opts {
			opt(d)
		}
	}
}

// WithCredentialsFile returns an Option that specifies a service account
// or refresh token JSON credentials file to be used as the basis for
// authentication.
func WithCredentialsFile(filename string) Option {
	return func(d *dialerConfig) {
		b, err := os.ReadFile(filename)
		if err != nil {
			d.err = errtype.NewConfigError(err.Error(), "n/a")
			return
		}
		opt := WithCredentialsJSON(b)
		opt(d)
	}
}

// WithCredentialsJSON returns an Option that specifies a service account
// or refresh token JSON credentials to be used as the basis for authentication.
func WithCredentialsJSON(b []byte) Option {
	return func(d *dialerConfig) {
		// TODO: Use AlloyDB-specfic scope
		c, err := google.CredentialsFromJSON(context.Background(), b, CloudPlatformScope)
		if err != nil {
			d.err = errtype.NewConfigError(err.Error(), "n/a")
			return
		}
		d.tokenSource = c.TokenSource
		d.clientOpts = append(d.clientOpts, apiopt.WithCredentials(c))
	}
}

// WithUserAgent returns an Option that sets the User-Agent.
func WithUserAgent(ua string) Option {
	return func(d *dialerConfig) {
		d.userAgents = append(d.userAgents, ua)
	}
}

// WithDefaultDialOptions returns an Option that specifies the default
// DialOptions used.
func WithDefaultDialOptions(opts ...DialOption) Option {
	return func(d *dialerConfig) {
		d.dialOpts = append(d.dialOpts, opts...)
	}
}

// WithTokenSource returns an Option that specifies an OAuth2 token source
// to be used as the basis for authentication.
func WithTokenSource(s oauth2.TokenSource) Option {
	return func(d *dialerConfig) {
		d.tokenSource = s
		d.clientOpts = append(d.clientOpts, apiopt.WithTokenSource(s))
	}
}

// WithRSAKey returns an Option that specifies a rsa.PrivateKey used to
// represent the client.
func WithRSAKey(k *rsa.PrivateKey) Option {
	return func(d *dialerConfig) {
		d.rsaKey = k
	}
}

// WithRefreshTimeout returns an Option that sets a timeout on refresh
// operations. Defaults to 60s.
func WithRefreshTimeout(t time.Duration) Option {
	return func(d *dialerConfig) {
		d.refreshTimeout = t
	}
}

// WithHTTPClient configures the underlying AlloyDB Admin API client with the
// provided HTTP client. This option is generally unnecessary except for
// advanced use-cases.
func WithHTTPClient(client *http.Client) Option {
	return func(d *dialerConfig) {
		d.clientOpts = append(d.clientOpts, apiopt.WithHTTPClient(client))
	}
}

// WithAdminAPIEndpoint configures the underlying AlloyDB Admin API client to
// use the provided URL.
func WithAdminAPIEndpoint(url string) Option {
	return func(d *dialerConfig) {
		d.alloydbClientOpts = append(d.alloydbClientOpts, apiopt.WithEndpoint(url))
	}
}

// WithDialFunc configures the function used to connect to the address on the
// named network. This option is generally unnecessary except for advanced
// use-cases. The function is used for all invocations of Dial. To configure
// a dial function per individual calls to dial, use WithOneOffDialFunc.
func WithDialFunc(dial func(ctx context.Context, network, addr string) (net.Conn, error)) Option {
	return func(d *dialerConfig) {
		d.dialFunc = dial
	}
}

// WithIAMAuthN enables automatic IAM Authentication. If no token source has
// been configured (such as with WithTokenSource, WithCredentialsFile, etc),
// the dialer will use the default token source as defined by
// https://pkg.go.dev/golang.org/x/oauth2/google#FindDefaultCredentialsWithParams.
func WithIAMAuthN() Option {
	return func(d *dialerConfig) {
		d.useIAMAuthN = true
	}
}

type debugLoggerWithoutContext struct {
	logger debug.Logger
}

// Debugf implements debug.ContextLogger.
func (d *debugLoggerWithoutContext) Debugf(_ context.Context, format string, args ...any) {
	d.logger.Debugf(format, args...)
}

var _ debug.ContextLogger = new(debugLoggerWithoutContext)

// WithDebugLogger configures a debug logger for reporting on internal
// operations. By default the debug logger is disabled.
// Prefer WithContextLogger.
func WithDebugLogger(l debug.Logger) Option {
	return func(d *dialerConfig) {
		d.logger = &debugLoggerWithoutContext{l}
	}
}

// WithContextLogger configures a debug lgoger for reporting on internal
// operations. By default the debug logger is disabled.
func WithContextLogger(l debug.ContextLogger) Option {
	return func(d *dialerConfig) {
		d.logger = l
	}
}

// WithLazyRefresh configures the dialer to refresh certificates on an
// as-needed basis. If a certificate is expired when a connection request
// occurs, the Go Connector will block the attempt and refresh the certificate
// immediately. This option is useful when running the Go Connector in
// environments where the CPU may be throttled, thus preventing a background
// goroutine from running consistently (e.g., in Cloud Run the CPU is throttled
// outside of a request context causing the background refresh to fail).
func WithLazyRefresh() Option {
	return func(d *dialerConfig) {
		d.lazyRefresh = true
	}
}

// WithStaticConnectionInfo specifies an io.Reader from which to read static
// connection info. This is a *dev-only* option and should not be used in
// production as it will result in failed connections after the client
// certificate expires. It is also subject to breaking changes in the format.
// NOTE: The static connection info is not refreshed by the dialer. The JSON
// format supports multiple instances, regardless of cluster.
//
// The reader should hold JSON with the following format:
//
//	{
//	    "publicKey": "<PEM Encoded public RSA key>",
//	    "privateKey": "<PEM Encoded private RSA key>",
//	    "projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>": {
//	        "ipAddress": "<PSA-based private IP address>",
//	        "publicIpAddress": "<public IP address>",
//	        "pscInstanceConfig": {
//	            "pscDnsName": "<PSC DNS name>"
//	        },
//	        "pemCertificateChain": [
//	            "<client cert>", "<intermediate cert>", "<CA cert>"
//	        ],
//	        "caCert": "<CA cert>"
//	    }
//	}
func WithStaticConnectionInfo(r io.Reader) Option {
	return func(d *dialerConfig) {
		d.staticConnInfo = r
	}
}

// WithOptOutOfAdvancedConnectionCheck disables the dataplane permission check.
// It is intended only for clients who are running in an environment where the
// workload's IP address is otherwise unknown and cannot be allow-listed in a
// VPC Service Control security perimeter. This option is incompatible with IAM
// Authentication.
//
// NOTE: This option is for internal usage only and is meant to ease the
// migration when the advanced check will be required on the server. In future
// versions this will revert to a no-op and should not be used. If you think
// you need this option, open an issue on
// https://github.com/GoogleCloudPlatform/alloydb-go-connector for design
// advice.
func WithOptOutOfAdvancedConnectionCheck() Option {
	return func(d *dialerConfig) {
		d.disableMetadataExchange = true
	}
}

// WithOptOutOfBuiltInTelemetry disables the internal metric export. By
// default, the Dialer will report on its internal operations to the
// alloydb.googleapis.com system metric prefix. These metrics help AlloyDB
// improve performance and identify client connectivity problems. Presently,
// these metrics aren't public, but will be made public in the future. To
// disable this telemetry, provide this option when initializing a Dialer.
func WithOptOutOfBuiltInTelemetry() Option {
	return func(d *dialerConfig) {
		d.disableBuiltInTelemetry = true
	}
}

// A DialOption is an option for configuring how a Dialer's Dial call is
// executed.
type DialOption func(d *dialCfg)

type dialCfg struct {
	dialFunc     func(ctx context.Context, network, addr string) (net.Conn, error)
	ipType       string
	tcpKeepAlive time.Duration
}

// DialOptions turns a list of DialOption instances into an DialOption.
func DialOptions(opts ...DialOption) DialOption {
	return func(cfg *dialCfg) {
		for _, opt := range opts {
			opt(cfg)
		}
	}
}

// WithOneOffDialFunc configures the dial function on a one-off basis for an
// individual call to Dial. To configure a dial function across all invocations
// of Dial, use WithDialFunc.
func WithOneOffDialFunc(dial func(ctx context.Context, network, addr string) (net.Conn, error)) DialOption {
	return func(c *dialCfg) {
		c.dialFunc = dial
	}
}

// WithTCPKeepAlive returns a DialOption that specifies the tcp keep alive
// period for the connection returned by Dial.
func WithTCPKeepAlive(d time.Duration) DialOption {
	return func(cfg *dialCfg) {
		cfg.tcpKeepAlive = d
	}
}

// WithPublicIP returns a DialOption that specifies a public IP will be used to
// connect.
func WithPublicIP() DialOption {
	return func(cfg *dialCfg) {
		cfg.ipType = alloydb.PublicIP
	}
}

// WithPrivateIP returns a DialOption that specifies a private IP (VPC) will be
// used to connect.
func WithPrivateIP() DialOption {
	return func(cfg *dialCfg) {
		cfg.ipType = alloydb.PrivateIP
	}
}

// WithPSC returns a DialOption that specifies a PSC endpoint will be used to
// connect.
func WithPSC() DialOption {
	return func(cfg *dialCfg) {
		cfg.ipType = alloydb.PSC
	}
}
