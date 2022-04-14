module github.com/cyverse-de/de-webhooks

go 1.16

require (
	github.com/buger/jsonparser v0.0.0-20170803100442-fda8192cc4f1
	github.com/cyverse-de/configurate v0.0.0-20190318152107-8f767cb828d9
	github.com/cyverse-de/dbutil v1.0.1
	github.com/cyverse-de/go-mod/otelutils v0.0.2
	github.com/cyverse-de/messaging/v9 v9.1.5
	github.com/cyverse-de/queries v1.0.1
	github.com/lib/pq v1.10.4
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/viper v1.4.0
	github.com/streadway/amqp v1.0.1-0.20200716223359-e6b33f460591
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.1.12 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.31.0
	go.opentelemetry.io/otel v1.6.3
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
