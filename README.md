# MQTT Blackbox Exporter

Tests MQTT messaging roundtrips (publish/subscribe on same topic).

Definition of roundtrip:

- start subscriber on $topic
- start publisher on $topc
- publish $messages on $topic
- receive $message on $topic

## Build

```
$ mkdir -p ${GOPATH}/src/github.com/inovex/
$ git clone https://github.com/inovex/mqtt_blackbox_exporter.git ${GOPATH}/src/github.com/inovex/mqtt_blackbox_exporter/
$ cd ${GOPATH}/src/github.com/inovex/mqtt_blackbox_exporter/
$ make
```

This will build the exporter and installs it to your GOPATH/bin directory.

## Configure

See ``config.yaml.dist`` for a configuration example.

## Run

```
$ ./mqtt_blackbox_exporter -config.file config.yaml
```

```
$ curl -s http://127.0.0.1:9142/metrics
...
# HELP probe_mqtt_completed Number of completed probes.
# TYPE probe_mqtt_completed counter
probe_mqtt_completed{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 64

...

# HELP probe_mqtt_duration Time taken to execute probe.
# TYPE probe_mqtt_duration histogram
probe_mqtt_duration_bucket{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL",le="0.005"} 0
probe_mqtt_duration_bucket{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL",le="0.01"} 0
probe_mqtt_duration_sum{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 50.09346619300002
probe_mqtt_duration_count{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 64
...

# HELP probe_mqtt_messages_published Number of published messages.
# TYPE probe_mqtt_messages_published counter
probe_mqtt_messages_published{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 640
...

# HELP probe_mqtt_messages_received Number of received messages.
# TYPE probe_mqtt_messages_received counter
probe_mqtt_messages_received{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 640
...

# HELP probe_mqtt_started Number of started probes.
# TYPE probe_mqtt_started counter
probe_mqtt_started{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 64
...
```
