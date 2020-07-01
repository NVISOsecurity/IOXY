package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/eclipse/paho.mqtt.golang/packets"
	log "github.com/sirupsen/logrus"
)

type MQTTConnect struct {
	Uuid             string
	Username         string
	Password         string
	ClientIdentifier string
	CleanSession     bool
	ProtocolName     string
	ProtocolVersion  int
}

type MQTTConnectResponse struct {
	Username         string
	Password         string
	ClientIdentifier string
}

type MQTTSubscribe struct {
	Uuid             string
	Username         string
	ClientIdentifier string
	Topic            string
	Qos              int
}

type MQTTSubscribeResponse struct {
	Topic string
}

type MQTTPublish struct {
	Uuid             string
	Username         string
	ClientIdentifier string
	Topic            string
	Qos              int
	Payload          string
}

type MQTTPublishResponse struct {
	Topic   string
	Payload string
}

func (session *Session) request(way string, uri string, request interface{}, response interface{}) (int, error) {
	//req.Header.Add("API-KEY", "tenant:admin")
	jData, err := json.Marshal(request)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", authURL+uri, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Add("Content-Type", "application/json")
	//req.Header.Add("API-KEY", "tenant:admin")
	//req.Header.Add("Authorization", "")
	req.Body = ioutil.NopCloser(bytes.NewReader(jData))

	resp, err := authClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	log.Println("Session", session.id, way, "- Auth response", authURL+uri, resp.StatusCode, string(body))
	if response != nil {
		err = json.Unmarshal(body, response)
		if err != nil {
			return 0, err
		}
	}

	return resp.StatusCode, nil
}

func (session *Session) HandleConnect(way string, p *packets.ConnectPacket, r net.Conn, w net.Conn) error {
	log.Println("Session", session.id, "- CONNECT")
	var resp MQTTConnectResponse
	rq := MQTTConnect{session.id, p.Username, string(p.Password), p.ClientIdentifier, p.CleanSession, p.ProtocolName, int(p.ProtocolVersion)}
	code, err := session.request(way, "/connect", rq, &resp)
	if err != nil {
		log.Errorln("Session", session.id, way, "- Error getting connect authorization", err)
		return err
	}

	if code != 200 {
		return errors.New("Connect Not Authorized")
	}
	if mqttBrokerUsername != "" {
		p.Username = mqttBrokerUsername
		p.Password = []byte(mqttBrokerPassword)
	}

	//Override information
	if resp.ClientIdentifier != "" && resp.ClientIdentifier != p.ClientIdentifier {
		log.Println("Session", session.id, way, "- CONNECT alter ClientIdentifier", p.ClientIdentifier, "-->", resp.ClientIdentifier)
		p.ClientIdentifier = resp.ClientIdentifier
	}
	if resp.Username != "" && resp.Username != p.Username {
		log.Println("Session", session.id, way, "- CONNECT alter Username", p.Username, "-->", resp.Username)
		p.Username = resp.Username
	}
	if resp.Password != "" && resp.Password != string(p.Password) {
		log.Println("Session", session.id, way, "- CONNECT alter Password")
		p.Password = []byte(resp.Password)
	}
	session.Username = p.Username
	session.ClientIdentifier = p.ClientIdentifier
	return nil
}

func (session *Session) HandleSubscribe(way string, p *packets.SubscribePacket, r net.Conn, w net.Conn) error {
	log.Println("Session", session.id, way, "- SUBSCRIBE", p.Topics, p.Qos)
	var resp MQTTSubscribeResponse
	topics := p.Topics
	for i := range p.Topics {
		rq := MQTTSubscribe{session.id, session.Username, session.ClientIdentifier, p.Topics[i], int(p.Qos)}
		code, err := session.request(way, "/subscribe", rq, &resp)

		if err != nil {
			log.Errorln("Session", session.id, way, "- Error getting subscribe authorization", err)
			return err
		}
		if code != 200 {
			cp2 := packets.NewControlPacket(packets.Suback)
			suback := cp2.(*packets.SubackPacket)
			suback.ReturnCodes = []byte{packets.ErrRefusedNotAuthorised}
			err := suback.Write(r)
			if err != nil {
				log.Errorln("Session", session.id, way, "- Error writing subscribe ack error message", err)
			}
			return errors.New("Subscribe Not Authorized")
		}

		if resp.Topic != "" && resp.Topic != topics[i] {
			log.Println("Session", session.id, way, "- SUBSCRIBE alter topic", i, topics[i], "-->", resp.Topic)
			topics[i] = resp.Topic
		}
	}

	p.Topics = topics

	return nil
}

func (session *Session) HandlePublish(way string, p *packets.PublishPacket, r net.Conn, w net.Conn) error {
	action := "PUBLISH"
	uri := "/publish"
	if w == session.inbound {
		action = "RECEIVE"
		uri = "/receive"
	}

	log.Println("Session", session.id, way, "- "+action, r.RemoteAddr().String(), w.RemoteAddr().String())
	log.Println("Session", session.id, way, "- "+action, p.TopicName, p.Qos, string(p.Payload))
	rq := MQTTPublish{session.id, session.Username, session.ClientIdentifier, p.TopicName, int(p.Qos), string(p.Payload)}
	var resp MQTTPublishResponse
	code, err := session.request(way, uri, rq, &resp)

	if err != nil {
		log.Errorln("Session", session.id, way, "- Error getting Publish authorization", err)
		return err
	}
	if code != 200 {
		return errors.New(action + " Not Authorized")
	}
	if resp.Topic != "" && resp.Topic != p.TopicName {
		log.Println("Session", session.id, way, "- "+action+" alter topic", p.TopicName, "-->", resp.Topic)
		p.TopicName = resp.Topic
	}
	if resp.Payload != "" && resp.Payload != string(p.Payload) {
		log.Println("Session", session.id, way, "- "+action+"alter topic", p.Payload, "-->", resp.Payload)
		p.Payload = []byte(resp.Payload)
	}
	return nil
}
