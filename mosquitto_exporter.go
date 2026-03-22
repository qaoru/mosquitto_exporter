package main

import (
	"fmt"
	"log"
	"net/http"
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
	if username != nil {
		mqttOptions.SetUsername(*username)
	}
	if password != nil {
		mqttOptions.SetPassword(*password)
	}

	// Create up collector and register it
	upCollector := internal.NewUpCollector(constLabels)
	prometheus.MustRegister(upCollector)

	// Set up connection handlers
	mqttOptions.SetOnConnectHandler(func(client mqtt.Client) {
		log.Println("Connected to broker")
		upCollector.SetUp(true)
	})
	mqttOptions.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("Connection lost: %v", err)
		upCollector.SetUp(false)
	})

	client := mqtt.NewClient(mqttOptions)
	client.Connect()
	log.Println("Attempting to connect to broker (async)")
	// Connection result will be handled by OnConnectHandler and ConnectionLostHandler

	defer client.Disconnect(250)

	if *clientsCollector {
		clientsCollector := internal.NewClientsCollector(constLabels)
		clientsCollector.Subscribe(client)
		prometheus.MustRegister(clientsCollector)
	}
	if *messagesCollector {
		messagesCollector := internal.NewMessagesCollector(constLabels)
		messagesCollector.Subscribe(client)
		prometheus.MustRegister(messagesCollector)
	}
	if *loadCollector {
		loadCollector := internal.NewLoadCollector(constLabels)
		loadCollector.Subscribe(client)
		prometheus.MustRegister(loadCollector)
	}

	defaultCollector := internal.NewDefaultCollector(constLabels)
	defaultCollector.Subscribe(client)
	prometheus.MustRegister(defaultCollector)

	// Health endpoint
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	http.Handle(*webTelemetryPath, promhttp.Handler())
	log.Printf("Starting server on %s", *webListenAddress)
	http.ListenAndServe(*webListenAddress, nil)
}
