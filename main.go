package main

import (
	"context"
	"database/sql"
	"flag"
	"log"

	"github.com/cyverse-de/configurate"
	"github.com/cyverse-de/go-mod/otelutils"
	"github.com/cyverse-de/messaging/v9"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

const serviceName = "de-webhooks"

//Log define a logrus logger
var Log = logrus.WithFields(logrus.Fields{
	"service": serviceName,
	"art-id":  serviceName,
	"group":   "org.cyverse",
})

//DBConnection db connection to DE database
type DBConnection struct {
	db *sql.DB
}

//Queue name
const queuename = "notification-queue"

var config *viper.Viper

func main() {

	logrus.SetFormatter(&logrus.JSONFormatter{})

	var (
		cfgPath = flag.String("config", "/etc/iplant/de/webhooks.yml", "The path to the config file")
	)

	flag.Parse()

	var tracerCtx, cancel = context.WithCancel(context.Background())
	defer cancel()
	shutdown := otelutils.TracerProviderFromEnv(tracerCtx, serviceName, func(e error) { log.Fatal(e) })
	defer shutdown()

	if *cfgPath == "" {
		Log.Fatal("--config must be set")
	}

	cfg, err := configurate.InitDefaults(*cfgPath, configurate.JobServicesDefaults)
	if err != nil {
		Log.Fatal(err)
	}
	config = cfg

	Log.Print("Connecting to amqp...")
	client, err := messaging.NewClient(config.GetString("amqp.uri"), true)
	if err != nil {
		Log.Fatal(err)
	}
	defer client.Close()

	go client.Listen()

	DBConnection := NewDBConnection(Init())
	defer DBConnection.db.Close()

	client.AddConsumerMulti(
		cfg.GetString("amqp.exchange.name"),
		"topic",
		queuename,
		[]string{config.GetString("amqp.routing")},
		func(ctx context.Context, del amqp.Delivery) {
			err := ProcessMessage(ctx, DBConnection, del)
			if err != nil {
				Log.Error(err)
				err = del.Reject(!del.Redelivered)
				if err != nil {
					Log.Error(err)
				}
			} else {
				err = del.Ack(false)
				if err != nil {
					Log.Error(err)
				}
			}
		},
		1)

	forever := make(chan bool)
	Log.Print("****Waiting for notifications. Press Ctrl + c to quit!****")
	<-forever
}

//NewDBConnection makes a new DBConnection
func NewDBConnection(db *sql.DB) *DBConnection {
	return &DBConnection{
		db: db,
	}
}
