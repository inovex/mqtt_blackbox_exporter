# MQTT Blackbox Exporter

[![Build Status](https://travis-ci.org/inovex/mqtt_blackbox_exporter.png?branch=master)](https://travis-ci.org/inovex/mqtt_blackbox_exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/inovex/mqtt_blackbox_exporter)](https://goreportcard.com/report/github.com/inovex/mqtt_blackbox_exporter)
[![Docker Pulls](https://img.shields.io/docker/pulls/inovex/mqtt_blackbox_exporter.svg?maxAge=604800)](https://hub.docker.com/r/inovex/mqtt_blackbox_exporter/)

Tests MQTT messaging roundtrips (publish/subscribe on same topic).

Definition of roundtrip:

- start subscriber on $topic
- start publisher on $topic
- publish $messages on $topic
- receive $message on $topic

## Build

```
$ mkdir -p ${GOPATH}/src/github.com/inovex/
$ git clone https://github.com/inovex/mqtt_blackbox_exporter.git ${GOPATH}/src/github.com/inovex/mqtt_blackbox_exporter/
$ cd ${GOPATH}/src/github.com/inovex/mqtt_blackbox_exporter/
$ make
```

This will build the mqtt_blackbox_exporter for all target platforms and write them to the ``build/`` directory.

Binaries are provided on Github, see https://github.com/inovex/mqtt_blackbox_exporter.

## Install

Place the binary somewhere in a ``PATH`` directory and make it executable (``chmod +x mqtt_blackbox_exporter``).

### Deploy on Openshift

Deploy the exporter on Openshift cluster just by running:

```
oc project <my-project>
oc process -f https://raw.githubusercontent.com/ivanovaleksandar/mqtt_blackbox_exporter/master/contrib/openshift-template.yaml \
    -p NAMESPACE=$(oc project -q) \
    | oc create -f -
```

This will create `ImageStream`, `Service`, `DeploymentConfig` and `ConfigMap`. The `ConfigMap` keeps the `config.yaml` file that is initiated on start up.  

*NOTE*: Please edit the configuration to point to the correct connection string, username and password that points to the application that you want to export the metrics from.

```
# Sample config for non SSL deployment
probes:
  - name: mqtt broker NONSSL
    broker_url: tcp://<application-url>:1883
    username: <username>
    password: <password>
    messages: 10
    interval: 30s
```

## Configure

See ``config.yaml.dist`` for a configuration example.

## Run

Native:

```
$ ./mqtt_blackbox_exporter -config.file config.yaml
```

Using Docker:

```
docker run --rm -it -p 9214:9214 -v ${PWD}/:/data/ inovex/mqtt_blackbox_exporter:latest -config.file /data/config.yaml
```

```
$ curl -s http://127.0.0.1:9214/metrics
...
# HELP probe_mqtt_completed_total Number of completed probes.
# TYPE probe_mqtt_completed_total counter
probe_mqtt_completed_total{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 64

...

# HELP probe_mqtt_duration_seconds Time taken to execute probe.
# TYPE probe_mqtt_duration_seconds histogram
probe_mqtt_duration_seconds_bucket{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL",le="0.005"} 0
probe_mqtt_duration_seconds_bucket{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL",le="0.01"} 0
probe_mqtt_duration_seconds_sum{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 50.09346619300002
probe_mqtt_duration_seconds_count{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 64
...

# HELP probe_mqtt_messages_published_total Number of published messages.
# TYPE probe_mqtt_messages_published_total counter
probe_mqtt_messages_published_total{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 640
...

# HELP probe_mqtt_messages_received_total Number of received messages.
# TYPE probe_mqtt_messages_received_total counter
probe_mqtt_messages_received_total{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 640
...

# HELP probe_mqtt_started_total Number of started probes.
# TYPE probe_mqtt_started_total counter
probe_mqtt_started_total{broker="ssl://mqtt.example.net:8883",name="mqtt broker SSL"} 64
...
```
