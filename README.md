# mosquitto_exporter
Prometheus exporter for Mosquitto MQTT broker.

This exporter subscribes to topics under the `$SYS` tree and exposes values as prometheus metrics.

Tested against Mosquitto v2.0.20.

Due to limitations of the client library, this exporter can only connect using MQTTv3 / MQTTv3.1.

## Running the exporter

```sh
docker run -d -p 9344:9344 ghcr.io/qaoru/mosquitto_exporter \
    --mqtt.broker=tcp://127.0.0.1:1883 \
    --collector.clients \
    --collector.messages \
    --collector.load
```