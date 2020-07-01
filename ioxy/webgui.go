package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"

	rice "github.com/GeertJohan/go.rice"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var (
	templateBox *rice.Box
	assetsBox   *rice.Box
	clients     = make(map[*websocket.Conn]bool)
	upgrader    = websocket.Upgrader{} // use default options
)

type Cert struct {
	CA         string
	ServerKey  string
	ClientCert string
}

type AlertBoostrap struct {
	Reason  string
	Message string
	Color   string
}

type Status struct {
	Proxy        string
	ListenerAddr string
	BrokerAddr   string
	ListenerMode string
	BrokerMode   string
	LivePayload  bool
	MITMCerts    Cert
	BrokerCerts  Cert
	Intercept    bool
	Alert        AlertBoostrap
}

type Logs struct {
	Proxy   string
	LogDump []string
}

func fsToString(filename string) string {
	s, err := templateBox.String(filename)
	if err != nil {
		log.Fatal(err)
	}
	return s
}

func serverStatus(isStarted int) string {
	// 0 = off, 1 = starting, 2 = started
	switch isStarted {
	case 0:
		return "Not running"
	case 1:
		return "Starting . . ."
	case 2:
		return "Running"
	}
	return ""
}

func fileToString(r *http.Request, id string) string {
	file, _, err := r.FormFile(id)
	if err != nil {
		return ""
	}
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	return fmt.Sprintf("%s", fileBytes)
}

func listenerAddress() string {
	switch listenerMode {
	case "mqtt":
		return mqttHost + ":" + strconv.Itoa(mqttPort)
	case "mqtts":
		return mqttsHost + ":" + strconv.Itoa(mqttsPort)
	case "http":
	case "https":
	default:
		return ""
	}
	return ""
}

func routeTraffic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if isStarted == 2 {
		http.Redirect(w, r, "/app?redirect_from=/", 307)
		return
	} else {
		http.Redirect(w, r, "/settings?redirect_from=/", 307)
		return
	}
}

func serveSettings(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/settings" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	tmpl, err := template.New("settings").Parse(fsToString("settings.html"))
	if err != nil {
		panic(err)
	}
	switch r.Method {
	case "GET":
		alert := AlertBoostrap{}
		key := r.URL.Query().Get("redirect_from")
		if len(key) > 0 {
			if key == "/app" {
				alert.Reason = "Server not started"
				alert.Message = "the server must be started before accessing the app"
				alert.Color = "danger"
			}
		}
		tmplVar := Status{serverStatus(isStarted), listenerAddress(), mqttBrokerHost + ":" + strconv.Itoa(mqttBrokerPort), listenerMode, brokerMode, intercept, Cert{mqttsCA, mqttsKey, mqttsCert}, Cert{authCAFile, mqttBrokerClientKey, mqttBrokerClientCert}, intercept, alert}
		tmpl.Execute(w, tmplVar)
	case "POST":
		if err := r.ParseMultipartForm(0); err != nil { // max upload size 3mb
			if err := r.ParseForm(); err == nil {
				//
			} else {
				fmt.Fprintf(w, "err: %v", err)
				return
			}
		}
		if r.FormValue("start") == "true" {
			if isStarted == 0 {
				log.Printf("Starting the server")
				start <- true
				isStarted = 1
			} else {
				log.Printf("Server already started, status %d", isStarted)
			}
		} else if r.FormValue("start") == "false" {
			if isStarted == 2 {
				log.Printf("Stopping the server")
				stop <- true
				isStarted = 0
			} else {
				log.Printf("Server already stopped, status %d", isStarted)
			}
		}
		if val := r.FormValue("mqtt-eb-host"); val != "" {
			listenerMode = "mqtt"
			mqttEnable = true
			mqttsEnable = false
			httpEnable = false
			httpsEnable = false
			log.Printf("mqtt-eb-host : %s", val)
			mqttHost = val
		}
		if val, err := strconv.Atoi(r.FormValue("mqtt-eb-port")); val != 0 && err == nil {
			listenerMode = "mqtt"
			mqttEnable = true
			mqttsEnable = false
			httpEnable = false
			httpsEnable = false
			log.Printf("mqtt-eb-port : %d", val)
			mqttPort = val
		}
		if val := r.FormValue("mqtts-eb-host"); val != "" {
			listenerMode = "mqtts"
			mqttEnable = false
			mqttsEnable = true
			httpEnable = false
			httpsEnable = false
			log.Printf("mqtts-eb-host : %s", r.FormValue("mqtts-eb-host"))
			mqttsHost = val
		}
		if val, err := strconv.Atoi(r.FormValue("mqtts-eb-port")); val != 0 && err == nil {
			listenerMode = "mqtts"
			mqttEnable = false
			mqttsEnable = true
			httpEnable = false
			httpsEnable = false
			log.Printf("mqtts-eb-port : %d", val)
			mqttsPort = val
		}
		if cert := fileToString(r, "mqtts-eb-ca"); cert != "" { // Cert{mqttsCA, mqttsKey, mqttsCert},
			log.Printf("%s", cert)
			mqttsCA = cert
		}
		if key := fileToString(r, "mqtts-eb-serv"); key != "" {
			log.Printf("%s", key)
			mqttsKey = key
		}
		if cert := fileToString(r, "mqtts-eb-client"); cert != "" {
			log.Printf("%s", cert)
			mqttsCert = cert
		}
		if val := r.FormValue("mqtt-b-host"); val != "" {
			brokerMode = "mqtt"
			mqttBrokerTLS = false
			log.Printf("mqtt-b-host : %s", val)
			mqttBrokerHost = val
		}
		if val, err := strconv.Atoi(r.FormValue("mqtt-b-port")); val != 0 && err == nil {
			brokerMode = "mqtt"
			mqttBrokerTLS = false
			log.Printf("mqtt-b-port : %d", val)
			mqttBrokerPort = val
		}
		if val := r.FormValue("mqtts-b-host"); val != "" {
			brokerMode = "mqtts"
			mqttBrokerTLS = true
			log.Printf("mqtts-b-host : %s", val)
			mqttBrokerHost = val
		}
		if val, err := strconv.Atoi(r.FormValue("mqtts-b-port")); val != 0 && err == nil {
			brokerMode = "mqtts"
			mqttBrokerTLS = true
			log.Printf("mqtts-b-port : %d", val)
			mqttBrokerPort = val
		}
		if cert := fileToString(r, "mqtts-b-ca"); cert != "" { // Cert{authCAFile, mqttBrokerClientKey, mqttBrokerClientCert}
			log.Printf("%s", cert)
			authCAFile = cert
		}
		if key := fileToString(r, "mqtts-b-serv"); key != "" {
			log.Printf("%s", key)
			mqttBrokerClientKey = key
		}
		if cert := fileToString(r, "mqtts-b-client"); cert != "" {
			log.Printf("%s", cert)
			mqttBrokerClientCert = cert
		}
		if r.FormValue("intercept") == "true" {
			intercept = true
		} else if r.FormValue("intercept") == "false" {
			intercept = false
		}
		tmplVar := Status{serverStatus(isStarted), listenerAddress(), mqttBrokerHost + ":" + strconv.Itoa(mqttBrokerPort), listenerMode, brokerMode, intercept, Cert{mqttsCA, mqttsKey, mqttsCert}, Cert{authCAFile, mqttBrokerClientKey, mqttBrokerClientCert}, intercept, AlertBoostrap{}}
		tmpl.Execute(w, tmplVar)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func serveAPP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/app" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if isStarted != 2 {
		http.Redirect(w, r, "/settings?redirect_from=/app", 307)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tmpl, err := template.New("app").Parse(fsToString("index.html"))
	if err != nil {
		panic(err)
	}
	tmplVar := Status{serverStatus(isStarted), listenerAddress(), mqttBrokerHost + ":" + strconv.Itoa(mqttBrokerPort), listenerMode, brokerMode, intercept, Cert{mqttsCA, mqttsKey, mqttsCert}, Cert{authCAFile, mqttBrokerClientKey, mqttBrokerClientCert}, intercept, AlertBoostrap{}}
	tmpl.Execute(w, tmplVar)
}

func serveLOG(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/logs" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tmpl, err := template.New("log").Parse(fsToString("log.html"))
	if err != nil {
		panic(err)
	}
	tmpl.Execute(w, Logs{serverStatus(isStarted), sessionLogs("file.log")})
}

func serveAssets(w http.ResponseWriter, r *http.Request) {
	reqFile := r.RequestURI
	dotExt := filepath.Ext(reqFile)
	var contentType string
	switch dotExt {
	case ".js":
		contentType = "application/javascript"
	case ".css":
		contentType = "text/css"
	case ".min.css":
		contentType = "text/css"
	case ".map":
		contentType = "application/json"
	case ".jpg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".svg":
		contentType = "image/svg+xml"
	}
	file, err := assetsBox.String(reqFile[8:])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Error 404 : ressource not found"))
	}
	w.Header().Add("Content-Type", contentType)
	w.Write([]byte(file))
}

func wsHandler(wspipe chan []byte, w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	log.Printf("New Websocket Connection Created")
	defer c.Close()
	stopStoudToWs := make(chan bool)
	go func() {
		for {
			select {
			case quit, active := <-stopStoudToWs:
				if active {
					if quit {
						return
					}
				}
			default:
				defer func() {
				}()
				s := bufio.NewScanner(stdr)
				for s.Scan() {
					wspipe <- []byte(s.Text())
				}
			}
		}
	}()
	go func() {
		for {
			var dat map[string]interface{}
			_, p, err := c.ReadMessage()
			if err != nil {
				log.Println(err)
				stopStoudToWs <- true
				c.Close()
				return
			}
			err = json.Unmarshal([]byte(p), &dat)
			if err != nil {
				log.Println(err)
				return
			}
			if val, ok := dat["intercept"]; ok {
				if val.(bool) {
					intercept = true
				} else {
					intercept = false
				}
			} else {
				wsIntercept <- dat
			}
		}
	}()
	for {
		select {
		case message := <-wspipe:
			err := c.WriteMessage(1, message)
			if err != nil {
				log.Println(err)
				c.Close()
				stopStoudToWs <- true
				return
			}
		default:
		}
	}
}

func initHTTP(host string, port string) {
	conf := rice.Config{
		LocateOrder: []rice.LocateMethod{rice.LocateEmbedded, rice.LocateAppended, rice.LocateFS},
	}
	riceBox, err := conf.FindBox("web/templates")
	if err != nil {
		log.Fatal(err)
	}
	templateBox = riceBox
	riceBox, err = conf.FindBox("web/assets")
	if err != nil {
		log.Fatal(err)
	}
	assetsBox = riceBox
	http.HandleFunc("/", routeTraffic)
	http.HandleFunc("/app", serveAPP)
	http.HandleFunc("/logs", serveLOG)
	http.HandleFunc("/settings", serveSettings)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) { wsHandler(wspipe, w, r) })
	http.HandleFunc("/assets/", serveAssets)
	http.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fsToString("manifest.json"))
	})
	go http.ListenAndServe(host+":"+port, nil)
}
