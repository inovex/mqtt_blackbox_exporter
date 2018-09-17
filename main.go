package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
)

type config struct {
	Probes []probeConfig `yaml:"probes"`
}

type probeConfig struct {
	Name         string        `yaml:"name"`
	Broker       string        `yaml:"broker_url"`
	Topic        string        `yaml:"topic"`
	ClientPrefix string        `yaml:"client_prefix"`
	Username     string        `yaml:"username"`
	Password     string        `yaml:"password"`
	ClientCert   string        `yaml:"client_cert"`
	ClientKey    string        `yaml:"client_key"`
	CAChain      string        `yaml:"ca_chain"`
	Messages     int           `yaml:"messages"`
	TestInterval time.Duration `yaml:"interval"`
}

var build string

var (
	messagesPublished = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_messages_published_total",
			Help: "Number of published messages.",
		}, []string{"name", "broker"})

	messagesPublishTimeout = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_messages_publish_timeout_total",
			Help: "Number of published messages.",
		}, []string{"name", "broker"})

	messagesReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_messages_received_total",
			Help: "Number of received messages.",
		}, []string{"name", "broker"})

	timedoutTests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_timeouts_total",
			Help: "Number of timed out tests.",
		}, []string{"name", "broker"})

	probeStarted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_started_total",
			Help: "Number of started probes.",
		}, []string{"name", "broker"})

	probeCompleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_completed_total",
			Help: "Number of completed probes.",
		}, []string{"name", "broker"})

	errors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_mqtt_errors_total",
			Help: "Number of errors occurred during test execution.",
		}, []string{"name", "broker"})

	probeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "probe_mqtt_duration_seconds",
			Help: "Time taken to execute probe.",
		}, []string{"name", "broker"})

	logger = log.New(os.Stderr, "", log.Lmicroseconds|log.Ltime|log.Lshortfile)

	configFile    = flag.String("config.file", "config.yaml", "Exporter configuration file.")
	listenAddress = flag.String("web.listen-address", ":9214", "The address to listen on for HTTP requests.")
	enableTrace   = flag.Bool("trace.enable", false, "set this flag to enable mqtt tracing")
)

func init() {
	prometheus.MustRegister(probeStarted)
	prometheus.MustRegister(probeDuration)
	prometheus.MustRegister(probeCompleted)
	prometheus.MustRegister(messagesPublished)
	prometheus.MustRegister(messagesReceived)
	prometheus.MustRegister(messagesPublishTimeout)
	prometheus.MustRegister(timedoutTests)
	prometheus.MustRegister(errors)
}

// Stolen from https://github.com/shoenig/go-mqtt/blob/master/samples/ssl.go
func NewTlsConfig(probeConfig *probeConfig) *tls.Config {
	// Import trusted certificates from CAChain - purely for verification - not sent to TLS server
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(probeConfig.CAChain)
	if err == nil {
		certpool.AppendCertsFromPEM(pemCerts)
	}

	// Import client certificate/key pair
	// If you want the chain certs to be sent to the server, concatenate the leaf,
	//  intermediate and root into the ClientCert file
	cert, err := tls.LoadX509KeyPair(probeConfig.ClientCert, probeConfig.ClientKey)
	if err != nil {
		return &tls.Config{}
	}

	// Create tls.Config with desired tls properties
	return &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certpool,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: false,
		// Certificates = list of certs client sends to server.
		Certificates: []tls.Certificate{cert},
	}
}

func connectClient(probeConfig *probeConfig, timeout time.Duration, opts *mqtt.ClientOptions) (mqtt.Client, error) {
	tlsconfig := NewTlsConfig(probeConfig)
	baseOptions := mqtt.NewClientOptions()
	if opts != nil {
		baseOptions = opts
	}
	baseOptions = baseOptions.SetAutoReconnect(false).
		SetUsername(probeConfig.Username).
		SetPassword(probeConfig.Password).
		SetTLSConfig(tlsconfig).
		AddBroker(probeConfig.Broker)
	client := mqtt.NewClient(baseOptions)
	token := client.Connect()
	success := token.WaitTimeout(timeout)
	if !success {
		return nil, fmt.Errorf("reached connect timeout")
	}
	if token.Error() != nil {
		return nil, fmt.Errorf("failed to connect client: %s", token.Error().Error())
	}
	return client, nil

}

func startProbe(probeConfig *probeConfig) {
	num := probeConfig.Messages
	setupTimeout := probeConfig.TestInterval / 3
	probeTimeout := probeConfig.TestInterval / 3
	setupDeadLine := time.Now().Add(setupTimeout)
	qos := byte(0)
	t0 := time.Now()

	// Initialize optional metrics with initial values to have them present from the beginning
	messagesPublished.WithLabelValues(probeConfig.Name, probeConfig.Broker).Add(0)
	messagesReceived.WithLabelValues(probeConfig.Name, probeConfig.Broker).Add(0)
	errors.WithLabelValues(probeConfig.Name, probeConfig.Broker).Add(0)
	timedoutTests.WithLabelValues(probeConfig.Name, probeConfig.Broker).Add(0)

	// Starting to fill metric vectors with meaningful values
	probeStarted.WithLabelValues(probeConfig.Name, probeConfig.Broker).Inc()
	defer func() {
		probeCompleted.WithLabelValues(probeConfig.Name, probeConfig.Broker).Inc()
		probeDuration.WithLabelValues(probeConfig.Name, probeConfig.Broker).Observe(time.Since(t0).Seconds())
	}()

	queue := make(chan [2]string)
	reportError := func(error error) {
		errors.WithLabelValues(probeConfig.Name, probeConfig.Broker).Inc()
		logger.Printf("Probe %s: %s", probeConfig.Name, error.Error())
	}

	publisherOptions := mqtt.NewClientOptions().SetClientID(fmt.Sprintf("%s-p", probeConfig.ClientPrefix))

	subscriberOptions := mqtt.NewClientOptions().SetClientID(fmt.Sprintf("%s-s", probeConfig.ClientPrefix)).
		SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
			queue <- [2]string{msg.Topic(), string(msg.Payload())}
		})

	publisher, err := connectClient(probeConfig, setupDeadLine.Sub(time.Now()), publisherOptions)
	if err != nil {
		reportError(err)
		return
	}
	defer publisher.Disconnect(5)

	subscriber, err := connectClient(probeConfig, setupDeadLine.Sub(time.Now()), subscriberOptions)
	if err != nil {
		reportError(err)
		return
	}
	defer subscriber.Disconnect(5)

	if token := subscriber.Subscribe(probeConfig.Topic, qos, nil); token.WaitTimeout(setupDeadLine.Sub(time.Now())) && token.Error() != nil {
		reportError(token.Error())
		return
	}
	defer subscriber.Unsubscribe(probeConfig.Topic)

	probeDeadline := time.Now().Add(probeTimeout)
	timeout := time.After(probeTimeout)
	timeoutTriggered := false
	receiveCount := 0

	for i := 0; i < num; i++ {
		text := fmt.Sprintf("this is msg #%d!", i)
		token := publisher.Publish(probeConfig.Topic, qos, false, text)
		if !token.WaitTimeout(probeDeadline.Sub(time.Now())) {
			messagesPublishTimeout.WithLabelValues(probeConfig.Name, probeConfig.Broker).Inc()
		}
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

	mqtt.ERROR = logger
	mqtt.CRITICAL = logger

	if *enableTrace {
		mqtt.WARN = logger
		mqtt.DEBUG = logger
	}

	config := config{}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		logger.Fatalf("Error parsing config file: %s", err)
	}

	logger.Printf("Starting mqtt_blackbox_exporter (build: %s)\n", build)

	for _, probe := range config.Probes {

		delay := probe.TestInterval
		if delay == 0 {
			delay = 60 * time.Second
		}
		go func(probeConfig probeConfig) {
			for {
				startProbe(&probeConfig)
				time.Sleep(delay)
			}
		}(probe)
	}

	http.Handle("/metrics", prometheus.Handler())
	http.ListenAndServe(*listenAddress, nil)
}
