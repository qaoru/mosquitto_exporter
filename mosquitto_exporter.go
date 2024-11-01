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
	broker            = kingpin.Flag("broker", "Broker connection string.").Short('b').Default("tcp://127.0.0.1:1883").String()
	clientID          = kingpin.Flag("client-id", "Client ID to use when connected to the broker.").Default("mosquitto-exporter").String()
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
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
