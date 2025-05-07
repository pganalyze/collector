// +heroku goVersion go1.23

module github.com/pganalyze/collector

require (
	cloud.google.com/go v0.112.2 // indirect
	cloud.google.com/go/pubsub v1.36.1
	github.com/AlecAivazis/survey/v2 v2.2.1
	github.com/aws/aws-sdk-go v1.55.3
	github.com/certifi/gocertifi v0.0.0-20210507211836-431795d63e8d // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/getsentry/raven-go v0.0.0-20161115135411-3f7439d3e74d
	github.com/go-ini/ini v1.62.0
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/gorhill/cronexpr v0.0.0-20160318121724-f0984319b442
	github.com/guregu/null v0.0.0-20160228005316-41961cea0328
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/jtolds/gls v4.2.0+incompatible // indirect
	github.com/juju/syslog v0.0.0-20150205155936-6be94e8b7187
	github.com/kr/logfmt v0.0.0-20140226030751-b84e30acd515
	github.com/kylelemons/godebug v1.1.0
	github.com/lib/pq v1.10.7
	github.com/ogier/pflag v0.0.0-20160129220114-45c278ab3607
	github.com/papertrail/go-tail v0.0.0-20180509224916-973c153b0431
	github.com/pkg/errors v0.9.1
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/smartystreets/assertions v0.0.0-20160707190355-2063fd1cc7c9 // indirect
	github.com/smartystreets/goconvey v0.0.0-20160704134950-4622128e06c7 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.40.0
	google.golang.org/api v0.231.0
	google.golang.org/protobuf v1.36.6
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/mcuadros/go-syslog.v2 v2.3.0
)

require (
	cloud.google.com/go/cloudsqlconn v1.17.0
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.14.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.7.0
	github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery v1.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmosforpostgresql/armcosmosforpostgresql v1.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v4 v4.0.0-beta.5
	github.com/fatih/color v1.16.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/pganalyze/pg_query_go/v6 v6.1.0
	github.com/prometheus/procfs v0.7.3
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.19.0
	go.opentelemetry.io/otel/sdk v1.35.0
	go.opentelemetry.io/otel/trace v1.35.0
	go.opentelemetry.io/proto/otlp v1.0.0
)

require (
	cloud.google.com/go/auth v0.16.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.6.0 // indirect
	cloud.google.com/go/iam v1.1.6 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/Azure/go-amqp v1.0.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.4 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	google.golang.org/genproto v0.0.0-20240213162025-012b6fc9bca9 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250505200425-f936aa4a68b2 // indirect
	google.golang.org/grpc v1.72.0 // indirect
)

go 1.23.0

toolchain go1.23.4
