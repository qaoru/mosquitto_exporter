package main

import (
	"log"
	"net/http"

	"github.com/alecthomas/kingpin"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/qaoru/mosquitto_exporter/internal"
)

func connectHandler(client mqtt.Client) {
	log.Println("Connected")
}

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
	kingpin.Parse()
	mqttOptions := mqtt.NewClientOptions().AddBroker(*broker)
	constLabels["broker"] = *broker
	mqttOptions.SetClientID(*clientID)
	mqttOptions.SetOnConnectHandler(connectHandler)
	if username != nil {
		mqttOptions.SetUsername(*username)
	}
	if password != nil {
		mqttOptions.SetPassword(*password)
	}

	client := mqtt.NewClient(mqttOptions)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

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
	http.Handle(*webTelemetryPath, promhttp.Handler())
	http.ListenAndServe(*webListenAddress, nil)
}
