// +heroku goVersion go1.19

module github.com/pganalyze/collector

require (
	cloud.google.com/go v0.68.0 // indirect
	cloud.google.com/go/pubsub v1.8.1
	github.com/AlecAivazis/survey/v2 v2.2.1
	github.com/Azure/azure-amqp-common-go/v3 v3.2.1
	github.com/Azure/azure-event-hubs-go/v3 v3.3.14
	github.com/Azure/azure-sdk-for-go v58.0.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.21
	github.com/Azure/go-autorest/autorest/adal v0.9.16 // indirect
	github.com/StackExchange/wmi v0.0.0-20150520194626-f3e2bae1e0cb // indirect
	github.com/aws/aws-sdk-go v1.36.10
	github.com/bmizerany/lpx v0.0.0-20130503172629-af85cf24c156
	github.com/certifi/gocertifi v0.0.0-20160926115448-a61bf5eafa3a // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gedex/inflector v0.0.0-20161103042756-046f2c312046
	github.com/getsentry/raven-go v0.0.0-20161115135411-3f7439d3e74d
	github.com/go-ini/ini v1.62.0
	github.com/go-ole/go-ole v0.0.0-20160708033836-be49f7c07711 // indirect
	github.com/golang-jwt/jwt/v4 v4.1.0 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/gorhill/cronexpr v0.0.0-20160318121724-f0984319b442
	github.com/guregu/null v0.0.0-20160228005316-41961cea0328
	github.com/hashicorp/go-retryablehttp v0.7.0
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/jtolds/gls v4.2.0+incompatible // indirect
	github.com/juju/syslog v0.0.0-20150205155936-6be94e8b7187
	github.com/kr/logfmt v0.0.0-20140226030751-b84e30acd515
	github.com/kr/pretty v0.2.1 // indirect
	github.com/kylelemons/godebug v0.0.0-20170224010052-a616ab194758
	github.com/lib/pq v1.10.7
	github.com/mitchellh/mapstructure v1.4.2 // indirect
	github.com/ogier/pflag v0.0.0-20160129220114-45c278ab3607
	github.com/papertrail/go-tail v0.0.0-20180509224916-973c153b0431
	github.com/pganalyze/pg_query_go/v2 v2.2.0
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v0.0.0-20160713180306-0aa62d5ddceb
	github.com/shirou/gopsutil v3.21.10+incompatible
	github.com/smartystreets/assertions v0.0.0-20160707190355-2063fd1cc7c9 // indirect
	github.com/smartystreets/goconvey v0.0.0-20160704134950-4622128e06c7 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/net v0.0.0-20210929193557-e81a3d93ecf6
	google.golang.org/api v0.32.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/mcuadros/go-syslog.v2 v2.3.0
)

require github.com/prometheus/procfs v0.7.3

require (
	github.com/Azure/go-amqp v0.16.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/devigned/tab v0.1.1 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/googleapis/gax-go/v2 v2.0.5 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/mattn/go-isatty v0.0.8 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	go.opencensus.io v0.22.4 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20210816074244-15123e1e1f71 // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
	golang.org/x/text v0.3.6 // indirect
	golang.org/x/tools v0.0.0-20201002184944-ecd9fd270d5d // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/genproto v0.0.0-20201002142447-3860012362da // indirect
	google.golang.org/grpc v1.32.0 // indirect
)

go 1.19
