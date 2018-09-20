#!/bin/bash

set -e

service_pid=

cleanup() {
	rc=$?
	rm -f test/out.log
	if [ ! -z "${service_pid}" ]; then
		kill $service_pid
	fi
	exit $rc
}
trap cleanup INT TERM

echo "=> Starting exporter"
./build/mqtt_blackbox_exporter -config.file config.yaml.dist &
service_pid=$!

echo "=> Waiting 5s"
sleep 5

echo "=> Requesting /metrics"
curl --silent --max-time 2 http://localhost:9214/metrics > test/out.log

echo "=> Killing exporter (pid=${service_pid})"
kill $service_pid

echo "=> Checking result"
grep 'probe_mqtt_started_total{broker="ssl://broker.mqttdashboard.com:8883",name="mqtt broker SSL"} [[:digit:]]' test/out.log
grep 'probe_mqtt_started_total{broker="tcp://test.mosquitto.org:1883",name="mqtt broker NONSSL"} [[:digit:]]' test/out.log
