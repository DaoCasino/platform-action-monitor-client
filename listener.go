package eventlistener

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"net/url"
	"sync"
	"time"
)

const (
	writeWait            = 10 * time.Second
	pongWait             = 60 * time.Second
	responseWait         = 10 * time.Second
	pingPeriod           = (pongWait * 9) / 10
	messageSizeLimit     = 0
	reconnectionAttempts = 5
	reconnectionDelay    = 2 * time.Second
)

var ListenerClosed = errors.New("listener closed")

type EventListener struct {
	Addr             string        // TCP address to listen.
	MessageSizeLimit int64         // Maximum message size allowed from client.
	WriteWait        time.Duration // Time allowed to write a message to the client.
	PongWait         time.Duration // Time allowed to read the next pong message from the peer.
	PingPeriod       time.Duration // Send pings to peer with this period. Must be less than pongWait.
	ResponseWait     time.Duration // Time allowed to wait response from server.

	ReconnectionDelay    time.Duration // Delay between connection attempts, used in RunListener
	ReconnectionAttempts int           // used in RunListener

	conn  *websocket.Conn
	event chan<- *EventMessage

	send     chan *responseQueue
	response chan *responseMessage

	done chan struct{}

	sync.Mutex
	subscriptions map[EventType]uint64
}

func NewEventListener(addr string, event chan<- *EventMessage) *EventListener {
	return &EventListener{
		Addr:             addr,
		MessageSizeLimit: messageSizeLimit,
		WriteWait:        writeWait,
		PongWait:         pongWait,
		PingPeriod:       pingPeriod,
		ResponseWait:     responseWait,

		ReconnectionDelay:    reconnectionDelay,
		ReconnectionAttempts: reconnectionAttempts,

		event:         event,
		send:          make(chan *responseQueue),
		response:      make(chan *responseMessage),
		subscriptions: make(map[EventType]uint64),
		done:          make(chan struct{}),
	}
}

// ListenAndServe starts the action listener. Returns an error if unable to connect.
// This method is non-blocking but does not support reconnections. If you need to maintain a connection, use Run
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

	if err == nil && result {
		e.Lock()
		e.subscriptions[eventType] = offset
		e.Unlock()
	}

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

	if err == nil && result {
		e.Lock()
		delete(e.subscriptions, eventType)
		e.Unlock()
	}

	return result, err
}

func (e *EventListener) addr() string {
	u := url.URL{Scheme: "ws", Host: e.Addr, Path: "/"}
	return u.String()
}

func (e *EventListener) Close() {
	close(e.done)
	close(e.event)
}
