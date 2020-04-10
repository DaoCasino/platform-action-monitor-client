package eventlistener

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"net/url"
	"time"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	responseWait   = 10 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 0
)

type EventListener struct {
	Addr           string        // TCP address to listen.
	MaxMessageSize int64         // Maximum message size allowed from client.
	WriteWait      time.Duration // Time allowed to write a message to the client.
	PongWait       time.Duration // Time allowed to read the next pong message from the peer.
	PingPeriod     time.Duration // Send pings to peer with this period. Must be less than pongWait.
	ResponseWait   time.Duration // Time allowed to wait response from server.

	conn  *websocket.Conn
	event chan<- *EventMessage

	send     chan *responseQueue
	response chan *responseMessage
}

func NewEventListener(addr string, event chan<- *EventMessage) *EventListener {
	return &EventListener{
		Addr:           addr,
		MaxMessageSize: maxMessageSize,
		WriteWait:      writeWait,
		PongWait:       pongWait,
		PingPeriod:     pingPeriod,
		ResponseWait:   responseWait,

		event:    event,
		send:     make(chan *responseQueue, 512),
		response: make(chan *responseMessage, 512),
	}
}

func (e *EventListener) ListenAndServe(parentContext context.Context) error {
	u := url.URL{Scheme: "ws", Host: e.Addr, Path: "/"}

	var err error
	e.conn, _, err = websocket.DefaultDialer.DialContext(parentContext, u.String(), nil)
	if err != nil {
		return err
	}

	go e.readPump(parentContext)
	go e.writePump(parentContext)
	return nil
}

func (e *EventListener) Subscribe(eventType EventType, offset uint64) (bool, error) {
	params := struct {
		Topic  string `json:"topic"`
		Offset uint64 `json:"offset"`
	}{
		eventType.ToString(),
		offset,
	}

	request := newRequestMessage(methodSubscribe, params)
	response, err := e.sendRequest(request)
	if err != nil {
		return false, err
	}

	if response.Error != nil {
		return false, errors.New(response.Error.Message)
	}

	result := false
	err = json.Unmarshal(response.Result, &result)
	return result, err
}

func (e *EventListener) Unsubscribe(eventType EventType) (bool, error) {
	params := struct {
		Topic string `json:"topic"`
	}{
		eventType.ToString(),
	}

	request := newRequestMessage(methodUnsubscribe, params)
	response, err := e.sendRequest(request)
	if err != nil {
		return false, err
	}

	if response.Error != nil {
		return false, errors.New(response.Error.Message)
	}

	result := false
	err = json.Unmarshal(response.Result, &result)
	return result, err
}
