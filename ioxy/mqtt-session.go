package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type Session struct {
	id string
	//id int64
	wg sync.WaitGroup
	//deviceName string
	Username         string
	ClientIdentifier string
	inbound          net.Conn
	outbound         net.Conn
	closed           bool
}

var globalSessionID int64
var globalSessionCount int32

var sessions []*Session

func RemoveIndex(s []*Session, index int) []*Session {
	return append(s[:index], s[index+1:]...)
}

// Create MQTT Session
func NewSession() *Session {
	var session Session
	//g := atomic.AddInt64(&globalSessionID, 1)
	id := uuid.NewV4()
	session.id = id.String()
	atomic.AddInt32(&globalSessionCount, 1)
	return &session
}

// Handle the forwading
func (session *Session) forwardHalf(way string, c1 net.Conn, c2 net.Conn) {
	defer c1.Close()
	defer c2.Close()
	defer session.wg.Done()
	//io.Copy(c1, c2)
	for {
		select {
		case v, ok := <-stop:
			if v == true && ok == true {
				log.Printf("Stopping forward server")
				c1.Close()
				c2.Close()
				session.closed = true
				return
			}
		default:
			log.Debugln("Session", session.id, way, "- Wait Packet", c1.RemoteAddr().String(), c2.RemoteAddr().String())
			err := session.ForwardMQTTPacket(way, c1, c2)
			if err != nil {
				session.closed = true
				return
			}
		}
	}
}

// Establish a connection to the distant broker
func (session *Session) DialOutbound() error {
	addr := mqttBrokerHost + ":" + strconv.Itoa(mqttBrokerPort)
	if mqttBrokerTLS {
		cert, err := tls.X509KeyPair([]byte(mqttBrokerClientCert), []byte(mqttBrokerClientKey))
		if err != nil {
			log.Fatalf("server: loadkeys: %s", err)
			return err
		}
		var config tls.Config
		if amazonMqttProtocol {
			// Check if CA is needed
			config = tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true, NextProtos: []string{"x-amzn-mqtt-ca"}}
		} else {
			config = tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}
		}
		client, err := tls.Dial("tcp", addr, &config)
		if err != nil {
			log.Fatalf("client: dial: %s", err)
		}
		client.Handshake()
		session.outbound = client
	} else {
		log.Println("Session", session.id, "- Dialing...", session.id, addr)
		client, err := net.Dial("tcp", addr)
		if err != nil {
			log.Errorln("Session", session.id, "- Dial failed :", addr, err)
			return err
		}
		log.Println("Session", session.id, "- Connected", session.inbound.RemoteAddr().String(), addr)
		session.outbound = client
	}
	return nil
}

/*Forward messages between :
* Client -> Proxy -> Broker
* Client <- Proxy <- Broker
**/
func (session *Session) Stream(conn net.Conn) {
	session.inbound = conn
	session.wg.Add(2)
	err := session.DialOutbound()
	if err != nil {
		return
	}
	go session.forwardHalf("<", session.outbound, session.inbound)
	go session.forwardHalf(">", session.inbound, session.outbound)
	session.wg.Wait()

	atomic.AddInt32(&globalSessionCount, -1)
	log.Println("Session", session.id, "Closed", conn.LocalAddr().String(), globalSessionCount)
}

// Accept Client Request
func mqttAccept(l net.Listener) {
	for {
		select {
		case v, ok := <-stop:
			if v == true && ok == true {
				log.Printf("Clossing opened sessions")
				for i, session := range sessions {
					session.inbound.Close()
					session.outbound.Close()
					sessions = RemoveIndex(sessions, i)
				}
				log.Printf("Stopping listener")
				l.Close()
				isStarted = 0
			}
			return
		case c := <-newConns:
			log.Println("New client connected")
			session := NewSession()
			sessions = append(sessions, session)
			go session.Stream(c)
		}
	}
}

// Create MQTT Server
func mqttListen() {
	// Listen for incoming connections.
	addr := mqttHost + ":" + strconv.Itoa(mqttPort)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println("mqtt: Error listening mqtt://"+addr, err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	log.Println("mqtt: listening on mqtt://" + addr)
	go func(l net.Listener) {
		for {
			c, err := l.Accept()
			if err != nil {
				// handle error (and then for example indicate acceptor is down)
				log.Printf("Closing listener goroutine")
				//newConns <- nil
				return
			}
			newConns <- c
		}
	}(l)

	mqttAccept(l)
}

// Create MQTTS Server
func mqttsListen() {
	// Listen for incoming connections.
	cert, err := tls.X509KeyPair([]byte(mqttsCert), []byte(mqttsKey))
	if err != nil {
		log.Fatalf("mqtts: loadkeys: %s", err)
		os.Exit(1)
	}
	var serverConf *tls.Config
	if mqttsCA != "" {
		rootCAs := configureRootCAs(&mqttsCA)
		if amazonMqttProtocol {
			serverConf = &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
				NextProtos:   []string{"x-amzn-mqtt-ca"}, // Can be removed
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    rootCAs,
			}

		} else {
			serverConf = &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    rootCAs,
			}
		}
	} else {
		serverConf = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}
	addr := mqttsHost + ":" + strconv.Itoa(mqttsPort)
	l, err := tls.Listen("tcp", addr, serverConf)
	if err != nil {
		log.Println("mqtts: Error listening mqtts://"+addr, err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	go func(l net.Listener) {
		for {
			c, err := l.Accept()
			if err != nil {
				// handle error (and then for example indicate acceptor is down)
				log.Printf("Closing listener goroutine")
				//newConns <- nil
				return
			}
			newConns <- c
		}
	}(l)
	log.Println("mqtts: listening on mqtts://" + addr)
	mqttAccept(l)
}

func configureRootCAs(caCertPathFlag *string) *x509.CertPool {
	// also load as bytes for x509
	// Read in the cert file
	x509certs, err := ioutil.ReadFile(*caCertPathFlag)
	if err != nil {
		log.Fatalf("Failed to append certificate to RootCAs: %v", err)
	}

	// Get the SystemCertPool, continue with an empty pool on error
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	// append the local cert to the in-memory system CA pool
	if ok := rootCAs.AppendCertsFromPEM(x509certs); !ok {
		log.Warning("No certs appended, using system certs only")
	}
	return rootCAs
}
