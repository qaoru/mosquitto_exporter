package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/qaoru/mosquitto_exporter/internal"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

var (
	webListenAddress = kingpin.Flag("web.listen-address", "Address on which the web server will listen.").Default(":9344").String()
	webTelemetryPath = kingpin.Flag("web.telemetry-path", "Path on which metrics will be served.").Default("/metrics").String()

	broker            = kingpin.Flag("mqtt.broker", "Broker connection string.").Short('b').Default("tcp://127.0.0.1:1883").Envar("MQTT_BROKER").String()
	clientID          = kingpin.Flag("mqtt.client-id", "Client ID to use when connected to the broker.").Default("mosquitto-exporter").Envar("MQTT_CLIENT_ID").String()
	username          = kingpin.Flag("mqtt.username", "Broker username").Short('u').Envar("MQTT_USERNAME").String()
	password          = kingpin.Flag("mqtt.password", "Broker password").Short('p').Envar("MQTT_PASSWORD").String()
	clientsCollector  = kingpin.Flag("collector.clients", "Enable the clients collector.").Bool()
	messagesCollector = kingpin.Flag("collector.messages", "Enable the messages collector.").Bool()
	loadCollector     = kingpin.Flag("collector.load", "Enable the load collector.").Bool()

	constLabels = make(prometheus.Labels, 4)
)

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Version(fmt.Sprintf("%s (commit %s, built %s by %s)", version, commit, date, builtBy))
	kingpin.Parse()
	mqttOptions := mqtt.NewClientOptions().AddBroker(*broker)
	constLabels["broker"] = *broker
	mqttOptions.SetClientID(*clientID)
	mqttOptions.SetAutoReconnect(true)
	mqttOptions.SetConnectRetry(true)
	mqttOptions.SetResumeSubs(true)
	mqttOptions.SetCleanSession(false)
	mqttOptions.SetMaxReconnectInterval(30 * time.Second)
	mqttOptions.SetConnectTimeout(5 * time.Second)
	if *username != "" {
		mqttOptions.SetUsername(*username)
	}
	if *password != "" {
		mqttOptions.SetPassword(*password)
	}

	// Create and register the up collector
	upCollector := internal.NewUpCollector(constLabels)
	prometheus.MustRegister(upCollector)

	// Create and register the metric collectors up front so they are always
	// present in /metrics (with zero values until data arrives). Subscriptions
	// are (re)established in the OnConnectHandler below, which only fires once
	// the broker connection is open. This keeps the HTTP server available even
	// when the broker is unreachable (graceful degradation: mosquitto_up=0).
	var (
		clientsColl  *internal.ClientsCollector
		messagesColl *internal.MessagesCollector
		loadColl     *internal.LoadCollector
	)
	if *clientsCollector {
		clientsColl = internal.NewClientsCollector(constLabels)
		prometheus.MustRegister(clientsColl)
	}
	if *messagesCollector {
		messagesColl = internal.NewMessagesCollector(constLabels)
		prometheus.MustRegister(messagesColl)
	}
	if *loadCollector {
		loadColl = internal.NewLoadCollector(constLabels)
		prometheus.MustRegister(loadColl)
	}
	defaultColl := internal.NewDefaultCollector(constLabels)
	prometheus.MustRegister(defaultColl)

	// Set up connection handlers. OnConnect runs in its own goroutine, so it
	// is safe to subscribe synchronously here without blocking main.
	mqttOptions.SetOnConnectHandler(func(client mqtt.Client) {
		log.Println("Connected to broker")
		upCollector.SetUp(true)
		defaultColl.Subscribe(client)
		if clientsColl != nil {
			clientsColl.Subscribe(client)
		}
		if messagesColl != nil {
			messagesColl.Subscribe(client)
		}
		if loadColl != nil {
			loadColl.Subscribe(client)
		}
	})
	mqttOptions.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("Connection lost: %v", err)
		upCollector.SetUp(false)
	})

	client := mqtt.NewClient(mqttOptions)
	log.Printf("Connecting to broker %s", *broker)
	client.Connect() // non-blocking: connection is retried in the background

	defer client.Disconnect(250)

	// Health endpoint
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	http.Handle(*webTelemetryPath, promhttp.Handler())

	srv := &http.Server{Addr: *webListenAddress}
	go func() {
		log.Printf("Starting server on %s", *webListenAddress)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
}
