package main

import (
	"flag"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Probes []ProbeConfig `yaml:"probes"`
}

type ProbeConfig struct {
	Name         string        `yaml:"name"`
	Broker       string        `yaml:"broker_url"`
	Topic        string        `yaml:"topic"`
	ClientPrefix string        `yaml:"client_prefix"`
	Username     string        `yaml:"username"`
	Password     string        `yaml:"password"`
	Messages     int           `yaml:"messages"`
	TestInterval time.Duration `yaml:"interval"`
}

var build string

var (
	messagesPublished = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_messages_published",
			Help: "Number of published messages.",
		}, []string{"name", "broker"})

	messagesReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_messages_received",
			Help: "Number of received messages.",
		}, []string{"name", "broker"})

	timedoutTests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_timeouts",
			Help: "Number of timed out tests.",
		}, []string{"name", "broker"})

	probeStarted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_started",
			Help: "Number of started probes.",
		}, []string{"name", "broker"})

	probeCompleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_completed",
			Help: "Number of completed probes.",
		}, []string{"name", "broker"})

	errors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_errors",
			Help: "Number of errors occured during test execution.",
		}, []string{"name", "broker"})

	probeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "probe_mqtt_duration",
			Help: "Time taken to execute probe.",
		}, []string{"name", "broker"})

	logger = log.New(os.Stderr, "", log.Lmicroseconds|log.Ltime|log.Lshortfile)

	configFile    = flag.String("config.file", "config.yaml", "Exporter configuration file.")
	listenAddress = flag.String("web.listen-address", ":9142", "The address to listen on for HTTP requests.")
)

func init() {
	prometheus.MustRegister(messagesPublished)
	prometheus.MustRegister(messagesReceived)
	prometheus.MustRegister(probeStarted)
	prometheus.MustRegister(probeCompleted)
	prometheus.MustRegister(probeDuration)

	prometheus.MustRegister(timedoutTests)
	prometheus.MustRegister(errors)
}

func startProbe(probeConfig *ProbeConfig) {
	num := probeConfig.Messages
	testTimeout := 10 * time.Second
	qos := byte(0)
	t0 := time.Now()
	logger.Printf("foo")

	probeStarted.WithLabelValues(probeConfig.Name, probeConfig.Broker).Inc()
	defer func() {
		probeCompleted.WithLabelValues(probeConfig.Name, probeConfig.Broker).Inc()
		probeDuration.WithLabelValues(probeConfig.Name, probeConfig.Broker).Observe(time.Since(t0).Seconds())
	}()

	queue := make(chan [2]string)
	reportError := func(error error) {
		errors.WithLabelValues(probeConfig.Name, probeConfig.Broker).Inc()
		logger.Print(error)
	}

	publisherOptions := mqtt.NewClientOptions().SetClientID(fmt.Sprintf("%s-p", probeConfig.ClientPrefix)).SetUsername(probeConfig.Username).SetPassword(probeConfig.Password).AddBroker(probeConfig.Broker)

	subscriberOptions := mqtt.NewClientOptions().SetClientID(fmt.Sprintf("%s-s", probeConfig.ClientPrefix)).SetUsername(probeConfig.Username).SetPassword(probeConfig.Password).AddBroker(probeConfig.Broker)

	subscriberOptions.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		queue <- [2]string{msg.Topic(), string(msg.Payload())}
	})

	publisher := mqtt.NewClient(publisherOptions)
	subscriber := mqtt.NewClient(subscriberOptions)

	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		reportError(token.Error())
		return
	}
	defer publisher.Disconnect(5)

	if token := subscriber.Connect(); token.Wait() && token.Error() != nil {
		reportError(token.Error())
		return
	}
	defer subscriber.Disconnect(5)

	if token := subscriber.Subscribe(probeConfig.Topic, qos, nil); token.Wait() && token.Error() != nil {
		reportError(token.Error())
		return
	}
	defer subscriber.Unsubscribe(probeConfig.Topic)

	timeout := time.After(testTimeout)
	timeoutTriggered := false
	receiveCount := 0

	for i := 0; i < num; i++ {
		text := fmt.Sprintf("this is msg #%d!", i)
		token := publisher.Publish(probeConfig.Topic, qos, false, text)
		token.Wait()
		messagesPublished.WithLabelValues(probeConfig.Name, probeConfig.Broker).Inc()
	}

	for receiveCount < num && !timeoutTriggered {
		select {
		case <-queue:
			receiveCount++
			messagesReceived.WithLabelValues(probeConfig.Name, probeConfig.Broker).Inc()
		case <-timeout:
			timedoutTests.WithLabelValues(probeConfig.Name, probeConfig.Broker).Inc()
			timeoutTriggered = true
		}
	}
}

func main() {
	flag.Parse()
	yamlFile, err := ioutil.ReadFile(*configFile)

	if err != nil {
		logger.Fatalf("Error reading config file: %s", err)
	}

	config := Config{}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		logger.Fatalf("Error parsing config file: %s", err)
	}

	logger.Printf("Starting mqtt_blackbox_exporter (build %s)\n", build)

	for _, probe := range config.Probes {

		delay := probe.TestInterval
		if delay == 0 {
			delay = 60 * time.Second
		}
		go func(probeConfig ProbeConfig) {
			for {
				startProbe(&probeConfig)
				time.Sleep(delay)
			}
		}(probe)
	}

	http.Handle("/metrics", prometheus.Handler())
	http.ListenAndServe(*listenAddress, nil)
}
