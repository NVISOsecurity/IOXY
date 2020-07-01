package main

import "flag"

func initFlags() {

	// Init MQTT Flags
	mqttFlagSet = flag.NewFlagSet("mqtt", flag.ExitOnError)
	mqttFlagSet.IntVar(&mqttPort, "mqtt-port", 1883, "Mqtt port to listen to.")
	mqttFlagSet.StringVar(&mqttHost, "mqtt-host", "0.0.0.0", "Mqtt interface to listen to.")

	// Init MQTTS Flags
	mqttsFlagSet = flag.NewFlagSet("mqtts", flag.ExitOnError)
	mqttsFlagSet.IntVar(&mqttsPort, "mqtts-port", 8883, "Mqtts port to listen to.")
	mqttsFlagSet.StringVar(&mqttsHost, "mqtts-host", "0.0.0.0", "Mqtts interface to listen to.")
	mqttsFlagSet.StringVar(&mqttsCert, "mqtts-cert", "certs/server.pem", "Certificate used for mqtt TLS.")
	mqttsFlagSet.StringVar(&mqttsKey, "mqtts-key", "certs/server.key", "Key used for mqtt TLS.")
	mqttsFlagSet.StringVar(&mqttsCA, "mqtts-ca", "", "CA certificate to verify peer.")

	// Init HTTP Flags
	httpFlagSet = flag.NewFlagSet("http", flag.ExitOnError)
	httpFlagSet.StringVar(&httpHost, "http-host", "0.0.0.0", "Listen http port (for http and websockets)")
	httpFlagSet.IntVar(&httpPort, "http-port", 8080, "Listen http port (for http and websockets)")

	// Init HTTPS Flags
	httpsFlagSet = flag.NewFlagSet("http", flag.ExitOnError)
	httpsFlagSet.StringVar(&httpsHost, "https-host", "0.0.0.0", "Listen https port (for https and websockets tls)")
	httpsFlagSet.IntVar(&httpsPort, "https-port", 8081, "Listen https port (for https and websockets tls)")
	httpsFlagSet.StringVar(&httpsCert, "https-cert", "certs/server.pem", "Certificate used for https.")
	httpsFlagSet.StringVar(&httpsKey, "https-key", "certs/server.key", "Key used for https.")

	// Init Broker Flags
	brokerFlagSet = flag.NewFlagSet("broker", flag.ExitOnError)
	brokerFlagSet.IntVar(&mqttBrokerPort, "mqtt-broker-port", 1883, "Port of the mqtt server")
	brokerFlagSet.StringVar(&mqttBrokerHost, "mqtt-broker-host", "/", "Host the mqtt server.")
	brokerFlagSet.StringVar(&mqttBrokerUsername, "mqtt-broker-username", "", "Username of the mqtt server. Reuse incoming one if empty")
	brokerFlagSet.StringVar(&mqttBrokerPassword, "mqtt-broker-password", "", "Password the mqtt server.")
	brokerFlagSet.BoolVar(&mqttBrokerTLS, "mqtt-broker-tls", false, "Enable tls protocol")
	brokerFlagSet.StringVar(&mqttBrokerClientCert, "mqtt-broker-cert", "", "Certificate used to connect to the server.")
	brokerFlagSet.StringVar(&mqttBrokerClientKey, "mqtt-broker-key", "", "Key used to connect to the server.")
	brokerFlagSet.BoolVar(&amazonMqttProtocol, "x-amzn-mqtt-ca", false, "Enable ALPN for mqtt on amazon servers")
	brokerFlagSet.StringVar(&authURL, "auth-url", "", "URL to the authz/authn service")
	brokerFlagSet.StringVar(&authCAFile, "auth-ca-file", "", "PEM encoded CA's certificate file for the authz/authn service")

	// MiTM Options
	mitmFlagSet = flag.NewFlagSet("mitm-opt", flag.ExitOnError)
	mitmFlagSet.BoolVar(&intercept, "intercept", false, "Enable live message modification")
	mitmFlagSet.StringVar(&verbosity, "verbosity", "info", "Set log verbosity several allowed [panic, fatal, error, warning, info, debug, trace]")

	// GUI
	guiFlagSet = flag.NewFlagSet("gui", flag.ExitOnError)
	guiFlagSet.StringVar(&guiHost, "host", "0.0.0.0", "Set gui host")
	guiFlagSet.StringVar(&guiPort, "port", "1111", "Set gui port")
}
