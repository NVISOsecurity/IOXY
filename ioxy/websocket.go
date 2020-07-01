package main

import (
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

type MqttReaderBuffer struct {
	//conn *websocket.Conn
	msg []byte
	i   int
}

type MqttReader struct {
	*websocket.Conn
	b *MqttReaderBuffer
}

func (r MqttReader) Read(p []byte) (int, error) {
	if r.b.i == -1 {
		_, msg, err := r.ReadMessage()
		if err != nil {
			log.Warnln("ws: reader message", err)

			return 0, err
		}
		r.b.i = 0
		r.b.msg = msg
		log.Println("ws: reader message", len(msg), string(r.b.msg))
	}

	p[0] = r.b.msg[r.b.i]
	if r.b.i < len(r.b.msg)-1 {
		r.b.i++
	} else {
		r.b.i = -1
	}
	//log.Println("ws: reader byte", p[0], r.b.i, len(r.b.msg))
	return 1, nil
}

func (r MqttReader) SetDeadline(time time.Time) error {
	return r.SetDeadline(time)
}

func (r MqttReader) Write(data []byte) (int, error) {
	err := r.WriteMessage(websocket.BinaryMessage, data)
	return len(data), err
}

// Init MQTT
func wsMqttPrepare() {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  10240,
		WriteBufferSize: 10240,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		Subprotocols: []string{"mqtt"},

		EnableCompression: true,
	}

	http.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		log.Println("ws: incoming connection", r.Header)

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("ws: ", err)
			return
		}
		log.Println("ws: client subscribed")
		wsrb := MqttReaderBuffer{nil, -1}
		wsr := MqttReader{conn, &wsrb}

		session := NewSession()
		session.Stream(wsr)
	})
}

// Create WebSocket Server
func wsMqttListen() {
	addr := httpHost + ":" + strconv.Itoa(httpPort)
	log.Println("ws: listening " + addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Errorln("ws: Error listening ws://"+addr, err.Error())
		os.Exit(1)
	}
}

// Create Secure WebSocket Server
func wssMqttListen() {
	addr := httpsHost + ":" + strconv.Itoa(httpsPort)
	log.Println("wss: listening wss://" + addr)
	err := http.ListenAndServeTLS(addr, httpsCert, httpsKey, nil)
	if err != nil {
		log.Errorln("wss: Error listening wss://"+addr, err.Error())
		os.Exit(1)
	}
}
