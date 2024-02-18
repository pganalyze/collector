// +heroku goVersion go1.20

module github.com/pganalyze/collector

require (
	cloud.google.com/go v0.110.8 // indirect
	cloud.google.com/go/pubsub v1.33.0
	github.com/AlecAivazis/survey/v2 v2.2.1
	github.com/StackExchange/wmi v0.0.0-20150520194626-f3e2bae1e0cb // indirect
	github.com/aws/aws-sdk-go v1.36.10
	github.com/certifi/gocertifi v0.0.0-20210507211836-431795d63e8d // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gedex/inflector v0.0.0-20161103042756-046f2c312046
	github.com/getsentry/raven-go v0.0.0-20161115135411-3f7439d3e74d
	github.com/go-ini/ini v1.62.0
	github.com/go-ole/go-ole v0.0.0-20160708033836-be49f7c07711 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/gorhill/cronexpr v0.0.0-20160318121724-f0984319b442
	github.com/guregu/null v0.0.0-20160228005316-41961cea0328
	github.com/hashicorp/go-retryablehttp v0.7.0
	github.com/jtolds/gls v4.2.0+incompatible // indirect
	github.com/juju/syslog v0.0.0-20150205155936-6be94e8b7187
	github.com/kr/logfmt v0.0.0-20140226030751-b84e30acd515
	github.com/kylelemons/godebug v1.1.0
	github.com/lib/pq v1.10.7
	github.com/ogier/pflag v0.0.0-20160129220114-45c278ab3607
	github.com/papertrail/go-tail v0.0.0-20180509224916-973c153b0431
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v1.2.0
	github.com/shirou/gopsutil v3.21.10+incompatible
	github.com/smartystreets/assertions v0.0.0-20160707190355-2063fd1cc7c9 // indirect
	github.com/smartystreets/goconvey v0.0.0-20160704134950-4622128e06c7 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/net v0.23.0
	google.golang.org/api v0.143.0
	google.golang.org/protobuf v1.33.0
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/mcuadros/go-syslog.v2 v2.3.0
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.6.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs v1.0.0
	github.com/fatih/color v1.16.0
	github.com/gorilla/websocket v1.5.1
	github.com/pganalyze/pg_query_go/v5 v5.1.0
	github.com/prometheus/procfs v0.7.3
	go.opentelemetry.io/otel v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.19.0
	go.opentelemetry.io/otel/sdk v1.19.0
	go.opentelemetry.io/otel/trace v1.19.0
	go.opentelemetry.io/proto/otlp v1.0.0
)

require (
	cloud.google.com/go/compute v1.23.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.3.0 // indirect
	github.com/Azure/go-amqp v1.0.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.1 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.19.0 // indirect
	golang.org/x/oauth2 v0.12.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/term v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20231002182017-d307bd883b97 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231002182017-d307bd883b97 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231002182017-d307bd883b97 // indirect
	google.golang.org/grpc v1.58.3 // indirect
)

go 1.20
