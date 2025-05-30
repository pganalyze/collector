<p align="center">
    <a href="https://cloud.google.com/alloydb/docs/connect-language-connectors#go-pgx">
        <img src="docs/images/alloydb-go-connector.png" alt="alloydb-go-connector image">
    </a>
</p>

# AlloyDB Go Connector

[![CI][ci-badge]][ci-build]
[![Go Reference][pkg-badge]][pkg-docs]

[ci-badge]: https://github.com/GoogleCloudPlatform/alloydb-go-connector/actions/workflows/tests.yaml/badge.svg?event=push
[ci-build]: https://github.com/GoogleCloudPlatform/alloydb-go-connector/actions/workflows/tests.yaml?query=event%3Apush+branch%3Amain
[pkg-badge]: https://pkg.go.dev/badge/cloud.google.com/go/alloydbconn.svg
[pkg-docs]: https://pkg.go.dev/cloud.google.com/go/alloydbconn

The _AlloyDB Go Connector_ is an AlloyDB connector designed for use with the Go
language. Using an AlloyDB connector provides the following benefits:

* **IAM Authorization:** uses IAM permissions to control who/what can connect to
  your AlloyDB instances

* **Improved Security:** uses TLS 1.3 encryption and identity verification
  between the client connector and the server-side proxy, independent of the
  database protocol.

* **Convenience:** removes the requirement to use and distribute SSL
  certificates, as well as manage firewalls or source/destination IP addresses.

* (optionally) **IAM DB Authentication:** provides support for
  [AlloyDB’s automatic IAM DB AuthN][iam-db-authn] feature.

[iam-db-authn]: https://cloud.google.com/alloydb/docs/manage-iam-authn

## Installation

You can install this repo with `go get`:

```sh
go get cloud.google.com/go/alloydbconn
```

## Usage

This package provides several functions for authorizing and encrypting
connections. These functions can be used with your database driver to connect to
your AlloyDB instance.

AlloyDB supports network connectivity through public IP addresses and private,
internal IP addresses. By default this package will attempt to connect over a
private IP connection. When doing so, this package must be run in an
environment that is connected to the [VPC Network][vpc] that hosts your
AlloyDB private IP address.

Please see [Configuring AlloyDB Connectivity][alloydb-connectivity] for more details.

[vpc]: https://cloud.google.com/vpc/docs/vpc
[alloydb-connectivity]: https://cloud.google.com/alloydb/docs/configure-connectivity

### APIs and Services

This package requires the following to connect successfully:

* IAM principal (user, service account, etc.) with the [AlloyDB
  Client and Service Usage Consumer][client-role] roles or equivalent
  permissions. [Credentials](#credentials) for the IAM principal are
  used to authorize connections to an AlloyDB instance.

* The [AlloyDB API][admin-api] to be enabled within your Google Cloud
  Project. By default, the API will be called in the project associated with the
  IAM principal.

[admin-api]:   https://console.cloud.google.com/apis/api/alloydb.googleapis.com
[client-role]: https://cloud.google.com/alloydb/docs/auth-proxy/overview#how-authorized

### Credentials

This repo uses the [Application Default Credentials (ADC)][adc] strategy for
resolving credentials. Please see [these instructions for how to set your ADC][set-adc]
(Google Cloud Application vs Local Development, IAM user vs service account credentials),
or consult the [golang.org/x/oauth2/google][google-auth] documentation.

To explicitly set a specific source for the Credentials, see [Using
Options](#using-options) below.

[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc
[google-auth]: https://pkg.go.dev/golang.org/x/oauth2/google#hdr-Credentials

### Connecting with pgx

To use the dialer with [pgx](https://github.com/jackc/pgx), use
[pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v4/pgxpool) by configuring a
[Config.DialFunc][dial-func] like so:

``` go
// Configure the driver to connect to the database
dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", pgUser, pgPass, pgDB)
config, err := pgxpool.ParseConfig(dsn)
if err != nil {
    log.Fatalf("failed to parse pgx config: %v", err)
}

// Create a new dialer with any options
d, err := alloydbconn.NewDialer(ctx)
if err != nil {
    log.Fatalf("failed to initialize dialer: %v", err)
}
// Don't close the dialer until you're done with the database connection
// e.g. at the end of your main function
defer d.Close()

// Tell the driver to use the AlloyDB Go Connector to create connections
config.ConnConfig.DialFunc = func(ctx context.Context, _ string, instance string) (net.Conn, error) {
    return d.Dial(ctx, "projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>")
}

// Interact with the driver directly as you normally would
conn, err := pgxpool.ConnectConfig(context.Background(), config)
if err != nil {
    log.Fatalf("failed to connect: %v", connErr)
}
defer conn.Close()
```

[dial-func]: https://pkg.go.dev/github.com/jackc/pgconn#Config

### Using Options

If you need to customize something about the `Dialer`, you can initialize
directly with `NewDialer`:

```go
ctx := context.Background()
d, err := alloydbconn.NewDialer(
    ctx,
    alloydbconn.WithCredentialsFile("key.json"),
)
if err != nil {
    log.Fatalf("unable to initialize dialer: %s", err)
}

conn, err := d.Dial(ctx, "projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>")
```

For a full list of customizable behavior, see alloydbconn.Option.

### Using DialOptions

If you want to customize how the connection is created, use a DialOption.

For example, to connect over public IP, use:

```go
conn, err := d.Dial(
    ctx,
    "projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>",
    alloydbconn.WithPublicIP(),
)
```

Or to use PSC, use:

``` go
conn, err := d.Dial(
    ctx,
    "projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>",
    alloydbconn.WithPSC(),
)
```

You can also use the `WithDefaultDialOptions` Option to specify DialOptions to
be used by default:

```go
d, err := alloydbconn.NewDialer(
    ctx,
    alloydbconn.WithDefaultDialOptions(
        alloydbconn.WithPublicIP(),
    ),
)
```

### Using the dialer with database/sql

Using the dialer directly will expose more configuration options. However, it is
possible to use the dialer with the `database/sql` package.

To use `database/sql`, use `pgxv5.RegisterDriver` with any necessary Dialer
configuration. Note: the connection string must use the keyword/value format
with host set to the instance connection name.

``` go
package foo

import (
    "database/sql"

    "cloud.google.com/go/alloydbconn"
    "cloud.google.com/go/alloydbconn/driver/pgxv5"
)

func Connect() {
    cleanup, err := pgxv5.RegisterDriver("alloydb", alloydbconn.WithPublicIP())
    if err != nil {
        // ... handle error
    }
    defer cleanup()

    db, err := sql.Open(
        "alloydb",
        "host=projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE> user=myuser password=mypass dbname=mydb sslmode=disable",
	)
    // ... etc
}
```

### Automatic IAM Database Authentication

The Go Connector supports [Automatic IAM database authentication][].

Make sure to [configure your AlloyDB Instance to allow IAM authentication][configure-iam-authn]
and [add an IAM database user][add-iam-user].

A `Dialer` can be configured to connect to an AlloyDB instance using
automatic IAM database authentication with the `WithIAMAuthN` Option.

```go
d, err := alloydbconn.NewDialer(ctx, alloydbconn.WithIAMAuthN())
```

When configuring the DSN for IAM authentication, the `password` field can be
omitted and the `user` field should be formatted as follows:

- For an IAM user account, this is the user's email address.
- For a service account, it is the service account's email without the
`.gserviceaccount.com` domain suffix.

For example, to connect using the `test-sa@test-project.iam.gserviceaccount.com`
service account, the DSN would look like:

```go
dsn := "user=test-sa@test-project.iam dbname=mydb sslmode=disable"
```

[Automatic IAM database authentication]: https://cloud.google.com/alloydb/docs/manage-iam-authn
[configure-iam-authn]: https://cloud.google.com/alloydb/docs/manage-iam-authn#enable
[add-iam-user]: https://cloud.google.com/alloydb/docs/manage-iam-authn#create-user

### Enabling Metrics and Tracing

This library includes support for metrics and tracing using [OpenCensus][]. To
enable metrics or tracing, you need to configure an [exporter][]. OpenCensus
supports many backends for exporters.

Supported metrics include:

- `alloydbconn/dial_latency`: The distribution of dialer latencies (ms)
- `alloydbconn/open_connections`: The current number of open AlloyDB
  connections
- `alloydbconn/dial_failure_count`: The number of failed dial attempts
- `alloydbconn/refresh_success_count`: The number of successful certificate
  refresh operations
- `alloydbconn/refresh_failure_count`: The number of failed refresh
  operations.
- `alloydbconn/bytes_sent`: The number of bytes sent to an AlloyDB instance.
- `alloydbconn/bytes_received`: The number of bytes received from an AlloyDB
  instance.

Supported traces include:

- `cloud.google.com/go/alloydbconn.Dial`: The dial operation including
  refreshing an ephemeral certificate and connecting to the instance
- `cloud.google.com/go/alloydbconn/internal.InstanceInfo`: The call to retrieve
  instance metadata (e.g., IP address, etc)
- `cloud.google.com/go/alloydbconn/internal.Connect`: The connection attempt
  using the ephemeral certificate
- AlloyDB API client operations

For example, to use [Cloud Monitoring][] and [Cloud Trace][], you would
configure an exporter like so:

```golang
package main

import (
    "contrib.go.opencensus.io/exporter/stackdriver"
    "go.opencensus.io/trace"
)

func main() {
    sd, err := stackdriver.NewExporter(stackdriver.Options{
        ProjectID: "mycoolproject",
    })
    if err != nil {
        // handle error
    }
    defer sd.Flush()
    trace.RegisterExporter(sd)

    sd.StartMetricsExporter()
    defer sd.StopMetricsExporter()

    // Use alloydbconn as usual.
    // ...
}
```

[OpenCensus]: https://opencensus.io/
[exporter]: https://opencensus.io/exporters/
[Cloud Monitoring]: https://cloud.google.com/monitoring
[Cloud Trace]: https://cloud.google.com/trace

### Debug Logging

The Go Connector supports optional debug logging to help diagnose problems with
the background certificate refresh. To enable it, provide a logger that
implements the `debug.ContextLogger` interface when initializing the Dialer.

For example:

``` go
import (
    "context"
    "net"

    "cloud.google.com/go/alloydbconn"
)

type myLogger struct{}

func (l *myLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
    // Log as you like here
}

func connect() {
    l := &myLogger{}

    d, err := NewDialer(
        context.Background(),
        alloydbconn.WithContextDebugLogger(l),
    )
    // use dialer as usual...
}
```

## Support policy

### Major version lifecycle

This project uses [semantic versioning](https://semver.org/), and uses the
following lifecycle regarding support for a major version:

**Active** - Active versions get all new features and security fixes (that
wouldn’t otherwise introduce a breaking change). New major versions are
guaranteed to be "active" for a minimum of 1 year.

**Deprecated** - Deprecated versions continue to receive security and critical
bug fixes, but do not receive new features. Deprecated versions will be
supported for 1 year.

**Unsupported** - Any major version that has been deprecated for >=1 year is
considered unsupported.

## Supported Go Versions

We follow the [Go Version Support Policy][go-policy] used by Google Cloud
Libraries for Go.

[go-policy]: https://github.com/googleapis/google-cloud-go#go-versions-supported

### Release cadence

This project aims for a release on at least a monthly basis. If no new features
or fixes have been added, a new PATCH version with the latest dependencies is
released.
