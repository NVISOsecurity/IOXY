package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	log "github.com/sirupsen/logrus"
)

type WsNormal struct {
	Way     string
	Topic   string
	Payload []byte
	Qos     byte
}

type WsIntercept struct {
	Intercept string
	Way       string
	Topic     string
	Payload   []byte
}

func (session *Session) ForwardMQTTPacket(way string, r net.Conn, w net.Conn) error {
	cp, err := packets.ReadPacket(r)
	if err != nil {
		if !session.closed {
			log.Errorln("Session", session.id, way, "- Error reading MQTT packet")
			log.Println("The client may have closed the connection")
		}
		return err
	}
	log.Debugln("Session", session.id, way, "- Forward MQTT packet", cp.String())
	switch p := cp.(type) {
	case *packets.ConnectPacket: /*Outbound only*/
		addClient(db, SessionMgmt{p.ClientIdentifier, session.id, string(p.Username), string(p.Password)})
	case *packets.PublishPacket: /*Inbound/Outbound only*/
		if intercept {
			if guiEnabled {
				b, err := json.Marshal(WsIntercept{Intercept: "a", Way: string(way), Topic: p.TopicName, Payload: p.Payload})
				checkError(err)
				log.Printf("%s", b)
				wspipe <- b
				quit := false
				for quit == false {
					select {
					case dat, status := <-wsIntercept:
						if status {
							p.TopicName = dat["topic"].(string)
							p.Payload = []byte(dat["payload"].(string))
							quit = true
						}
					default:
					}
				}
			} else {
				quit := false
				for quit == false {
					if way == ">" {
						fmt.Printf("Client \033[1;34m%s--%s\033[0m Proxy \033[1;34m%s--%s\033[0m Server\n", "", way, "", way)
					} else {
						fmt.Printf("Client \033[1;34m%s--%s\033[0m Proxy \033[1;34m%s--%s\033[0m Server\n", way, "", way, "")
					}
					fmt.Printf("Topic\t: %s\n", p.TopicName)
					fmt.Printf("Payload\t: %s\n", p.Payload)
					fmt.Printf("1. Change topic\n2. Change payload\n3. Send payload\n> ")
					reader := bufio.NewReader(os.Stdin)
					text, _ := reader.ReadString('\n')
					log.Printf("%v", text)
					text = strings.Replace(text, "\n", "", -1)
					log.Printf("%v", text)
					switch text {
					case "1":
						fmt.Printf("New Topic : ")
						text, _ := reader.ReadString('\n')
						text = strings.Replace(text, "\n", "", -1)
						p.Payload = []byte(text)
					case "2":
						fmt.Printf("New Payload : ")
						text, _ := reader.ReadString('\n')
						text = strings.Replace(text, "\n", "", -1)
						p.Payload = []byte(text)
					case "3":
						log.Println("Payload sent !")
						quit = true
					default:
						fmt.Printf("Bad arg !\n")
					}
				}
			}
		}
		log.Printf("client %s broker | Publish | packet : SessionId : %s, Topic : %s , Payload : %s, Dup : %t, QoS : %d, Retain : %t", way, session.id, p.TopicName, string(p.Payload), p.Dup, int(p.Qos), p.Retain)
		b, err := json.Marshal(WsNormal{Way: way, Topic: p.TopicName, Payload: p.Payload, Qos: p.Qos})
		checkError(err)
		wspipe <- b
		addMessage(db, Messages{session.id, p.TopicName, string(p.Payload), p.Dup, int(p.Qos), p.Retain, time.Now().Format("2006-01-02 15:04:05")})
	case *packets.SubscribePacket:
		log.Printf("client %s broker | Subscribe | packet : SessionId : %s, Topic : %s, Dup : %t, QoS : %d, Retain : %t", way, session.id, strings.Join(p.Topics, ","), p.Dup, int(p.Qos), p.Retain)
	default:
		err = nil
	}
	if authURL != "" {
		switch p := cp.(type) {
		case *packets.ConnectPacket: /*Outbound only*/
			err = session.HandleConnect(way, p, r, w)
		case *packets.SubscribePacket: /*Outbound only*/
			err = session.HandleSubscribe(way, p, r, w)
		case *packets.PublishPacket: /*Inbound/Outbound only*/
			err = session.HandlePublish(way, p, r, w)
		default:
			fmt.Printf("%v\n", p)
			err = nil
		}
	} else {
		err = nil
	}
	if err != nil {
		log.Debugln("Session", session.id, way, "- Forward MQTT packet", err)
		return err
	}
	err = cp.Write(w)
	if err != nil {
		log.Errorln("Session", session.id, way, "- Error writing MQTT packet", err)
		return err
	}
	return nil
}
