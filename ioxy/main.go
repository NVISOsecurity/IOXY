package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

/* Channels */
var wspipe chan []byte
var start chan bool
var stop chan bool
var newConns chan net.Conn
var wsIntercept chan map[string]interface{}

/* Proxy status */

var isStarted int
var listenerMode string
var brokerMode string

/* Var for parameter parser */
var mqttFlagSet *flag.FlagSet
var mqttsFlagSet *flag.FlagSet
var httpFlagSet *flag.FlagSet
var httpsFlagSet *flag.FlagSet
var brokerFlagSet *flag.FlagSet
var mitmFlagSet *flag.FlagSet
var guiFlagSet *flag.FlagSet

/* Options var */

// MITM MQTT
var mqttHost string // Default : 0.0.0.0
var mqttPort int    // Default : 1883
var mqttEnable = false

// MITM MQTTS
var mqttsHost string // Default : 0.0.0.0
var mqttsPort int    // Default : 8883
var mqttsCert string // Default : "certs/server.server.pem"
var mqttsKey string  // Default : "certs/server.key"
var mqttsCA string
var mqttsEnable = false

// MITM HTTP
var httpHost string // Default : 0.0.0.0
var httpPort int    // Default : 8080
var httpEnable = false

// MITM HTTPS
var httpsHost string // Default : 0.0.0.0
var httpsPort int    // Default : 8081
var httpsCert string // Default : "certs/server.server.pem"
var httpsKey string  // Default : "certs/server.key"
var httpsEnable = false

// Broker connection settings
var mqttBrokerHost string
var mqttBrokerPort int
var mqttBrokerUsername string // Default : using the creds from the client
var mqttBrokerPassword string // Default : using the creds from the client
var mqttBrokerTLS bool        // Default : false
var mqttBrokerClientKey string
var mqttBrokerClientCert string
var authURL string
var authCAFile string
var amazonMqttProtocol bool // If ALPN needed

// MiTM-OPT
var intercept bool   // Default : false
var verbosity string // Default : info

// GUI
var guiEnabled = false
var guiHost string // Default : "0.0.0.0"
var guiPort string // Default : "1111"

// Std Routing
var stdr *os.File
var stdw *os.File
var oldStd *os.File

var authClient *http.Client

var subscribed = false

var defaultMessage = `
Usage : ioxy ACOMMAND BCOMMAND [CCOMMAND] [DCOMMAND]

[] = optional

ACommands:
  mqtt	  	Create a mqtt server (0.0.0.0:1883 by default)
  mqtts	  	Create a mqtts server (0.0.0.0:8883 by default)
  http	  	Create a http server (0.0.0.0:8080 by default)
  https	  	Create a https server	(0.0.0.0:8081 by default)

BCommands :
  broker  	Used to set up the distant broker settings

CCommands :
  mitm-opt 	Mitm options like intercept

DCOMMAND :
  gui

Run 'ioxy COMMAND -h' for more information on a command.`

var db *sql.DB

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func startProxy() {
	log.Printf("Starting ioxy")
	if listenerMode == "mqtts" {
		log.Printf("MiTM Broker Settings | Mode : %s | Host : %s | Port : %d", listenerMode, mqttsHost, mqttsPort)
	} else {
		log.Printf("MiTM Broker Settings | Mode : %s | Host : %s | Port : %d", listenerMode, mqttHost, mqttPort)
	}
	if mqttBrokerUsername != "" && mqttBrokerHost != "" {
		log.Printf("Distant Broker Settings | Host : %s | Port : %d | Username : %s | Password : %s", mqttBrokerHost, mqttBrokerPort, mqttBrokerUsername, mqttBrokerPassword)
	} else {
		log.Printf("Distant Broker Settings | Host : %s | Port : %d", mqttBrokerHost, mqttBrokerPort)
	}
	var a string
	if intercept {
		a = "\033[0;36mactive\033[0m"
	} else {
		a = "\033[1;33mdisabled\033[0m"
	}
	log.Printf("Broker Misc | Payload Intercept : %s", a)
	if httpEnable {
		go wsMqttListen()
		isStarted = 2
	}

	if httpsEnable {
		go wssMqttListen()
		isStarted = 2
	}

	if mqttEnable {
		go mqttListen()
		isStarted = 2
	}

	if mqttsEnable {
		go mqttsListen()
		isStarted = 2
	}
}

func main() {
	listenerMode = "-"
	brokerMode = "-"
	isStarted = 0
	newConns = make(chan net.Conn)
	wspipe = make(chan []byte)
	stop = make(chan bool)
	start = make(chan bool)
	wsIntercept = make(chan map[string]interface{})
	initFlags() // Init the command args
	oldStd = os.Stdout
	stdr, stdw, _ = os.Pipe()
	os.Stdout = stdw
	// Check if there is args given
	if len(os.Args) < 2 {
		fmt.Fprintln(oldStd, defaultMessage)
		os.Exit(1)
	}

	var lastElement = 0 // Used to acces to next parsed element

	// Look for broker keywords in the args
	for index, element := range os.Args {
		if element == "broker" {
			lastElement = index
		}
	}

	// Parse ACOMMAND
	switch os.Args[1] {
	case "gui":
		guiEnabled = true
		guiFlagSet.Parse(os.Args[2:])
		initHTTP(guiHost, guiPort)
	case "mqtt":
		mqttEnable = true
		listenerMode = "mqtt"
		if lastElement != 0 {
			mqttFlagSet.Parse(os.Args[2:lastElement])
		} else {
			mqttFlagSet.Parse(os.Args[2:])
		}
	case "mqtts":
		mqttsEnable = true
		listenerMode = "mqtts"
		if lastElement != 0 {
			mqttsFlagSet.Parse(os.Args[2:lastElement])
		} else {
			mqttsFlagSet.Parse(os.Args[2:])
		}
		if mqttsCert != "" && mqttsKey != "" {
			var err error
			dat, err := ioutil.ReadFile(mqttsCert)
			if err != nil {
				fmt.Fprintln(oldStd, "Error reading mqtts certificate\n")
				fmt.Fprintln(oldStd, defaultMessage)
				os.Exit(1)
			}
			mqttsCert = string(dat)
			dat, err = ioutil.ReadFile(mqttsKey)
			if err != nil {
				fmt.Fprintln(oldStd, "Error reading mqtts key\n")
				fmt.Fprintln(oldStd, defaultMessage)
				os.Exit(1)
			}
			mqttsKey = string(dat)
		} else {
			if mqttsKey != "" {
				fmt.Fprintln(oldStd, "Missing mqtts key\n")
			} else {
				fmt.Fprintln(oldStd, "Missing mqtts cert\n")
			}
			fmt.Fprintln(oldStd, defaultMessage)
			os.Exit(1)
		}
	case "http":
		httpEnable = true
		if lastElement != 0 {
			httpFlagSet.Parse(os.Args[2:lastElement])
		} else {
			httpFlagSet.Parse(os.Args[2:])
		}
	case "https":
		httpsEnable = true
		if lastElement != 0 {
			httpsFlagSet.Parse(os.Args[2:lastElement])
		} else {
			httpsFlagSet.Parse(os.Args[2:])
		}
	case "broker":
		if os.Args[2] == "-h" || os.Args[2] == "--help" {
			brokerFlagSet.Parse(os.Args[2:3])
			os.Exit(1)
		}
	case "mitm-opt":
		if os.Args[2] == "-h" || os.Args[2] == "--help" {
			mitmFlagSet.Parse(os.Args[2:3])
			os.Exit(1)
		}

	default:
		fmt.Fprintln(oldStd, defaultMessage)
		os.Exit(1)
	}

	// Load CA cert
	if mqttsEnable || httpsEnable {
		authCACertPool := x509.NewCertPool()
		if authURL != "" && authCAFile != "" {
			caCert, err := ioutil.ReadFile(authCAFile)
			if err != nil {
				log.Fatal(err)
			}

			pemBlock, _ := pem.Decode(caCert)
			clientCert, err := x509.ParseCertificate(pemBlock.Bytes)
			if err != nil {
				log.Fatal(err)
			}

			clientCert.BasicConstraintsValid = true
			clientCert.IsCA = true
			clientCert.KeyUsage = x509.KeyUsageCertSign
			//clientCert.DNSNames = append(clientCert.DNSNames, "policy")
			authCACertPool.AddCert(clientCert)
			//log.Println("auth using CA  : '" + authCAFile + "'")
		}

		tlsConfig := &tls.Config{RootCAs: authCACertPool}
		//tlsConfig.BuildNameToCertificate()
		tr := &http.Transport{
			TLSClientConfig: tlsConfig,
			//DisableCompression: true,
		}
		authClient = &http.Client{Transport: tr}
	}
	if os.Args[1] != "gui" {
		if lastElement == 0 {
			fmt.Fprintln(oldStd, "\n/!\\ BCOMMAND NEEDED /!\\")
			fmt.Fprintln(oldStd, defaultMessage)
			os.Exit(1)
		}
		// Parse BCOMMAND
		if os.Args[lastElement] == "broker" {
			brokerFlagSet.Parse(os.Args[lastElement+1:])
			if mqttBrokerTLS {
				brokerMode = "mqtts"
				flagset := make(map[string]bool)
				brokerFlagSet.Visit(func(f *flag.Flag) { flagset[f.Name] = true })
				if flagset["mqtt-broker-key"] {
					dat, err := ioutil.ReadFile(mqttBrokerClientKey)
					if err != nil {
						fmt.Fprintln(oldStd, "Error reading mqtts key")
						fmt.Fprintln(oldStd, defaultMessage)
						os.Exit(1)
					}
					mqttBrokerClientKey = string(dat)
				}
				if flagset["mqtt-broker-cert"] {
					dat, err := ioutil.ReadFile(mqttBrokerClientCert)
					if err != nil {
						fmt.Fprintln(oldStd, "Error reading mqtts client")
						fmt.Fprintln(oldStd, defaultMessage)
						os.Exit(1)
					}
					mqttBrokerClientCert = string(dat)
				}
			} else {
				brokerMode = "mqtt"
			}
		} else {
			fmt.Fprintln(oldStd, defaultMessage)
		}

		// Look for mitm-opt keywords in the args
		lastElement = 0
		for index, element := range os.Args {
			if element == "mitm-opt" {
				lastElement = index
			}
		}

		// Parse CCOMMAND
		if os.Args[lastElement] == "mitm-opt" {
			mitmFlagSet.Parse(os.Args[lastElement+1:])
		}

		// Look for gui keywords in the args
		lastElement = 0
		for index, element := range os.Args {
			if element == "gui" {
				lastElement = index
			}
		}
		// Parse DCOMMAND
		if os.Args[lastElement] == "gui" {
			guiFlagSet.Parse(os.Args[lastElement+1:])
			guiEnabled = true
			initHTTP(guiHost, guiPort)
		}
	}
	/* Log setup */
	initializeLogging("file.log")
	exist := stringInSlice(verbosity, []string{"panic", "fatal", "error", "warning", "info", "debug", "trace"})
	if exist {
		lvl, _ := log.ParseLevel(verbosity)
		log.SetLevel(lvl)
		fmt.Fprintln(oldStd, "Log Level Set To : "+verbosity)
	} else {
		fmt.Fprintln(oldStd, "Only the following levels are allowed : \n\n\tpanic\n\tfatal\n\terror\n\twarning\n\tinfo\n\tdebug\n\ttrace")
		os.Exit(1)
	}
	/* End Log setup*/

	if mqttBrokerHost == "0.0.0.0" && brokerFlagSet.Parsed() {
		fmt.Fprintln(oldStd, "\n/!\\ Missing option for broker command : -mqtt-broker-host /!\\")
		fmt.Fprintln(oldStd, defaultMessage)
		os.Exit(1)
	}

	// Print the auth url
	if authURL != "" {
		log.Println("auth connect   : ", authURL+"/connect")
		log.Println("auth publish   : ", authURL+"/publish")
		log.Println("auth subscribe : ", authURL+"/subscribe")
	} else {
		log.Println("auth : no auth url configured : bypassing!")
	}

	db = initDB()
	defer db.Close()

	if httpEnable || httpsEnable {
		wsMqttPrepare()
	}
	if guiEnabled {
		log.Printf("GUI listening on %s:%s", guiHost, guiPort)
		for {
			select {
			case v, ok := <-start:
				if v == true && ok == true && (isStarted == 0 || isStarted == 1) {
					startProxy()
				}
			}
		}
	} else {
		startProxy()
	}

	// Exit gracefully
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
