package eventlistener

import (
	"encoding/json"
	"fmt"
	"github.com/lucsky/cuid"
	"go.uber.org/zap"
	"time"
)

const (
	methodSubscribe        = "subscribe"
	methodUnsubscribe      = "unsubscribe"
	methodBatchSubscribe   = "batchSubscribe"
	methodBatchUnsubscribe = "batchUnsubscribe"
)

type requestMessage struct {
	ID     string      `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

func (req *requestMessage) toJSON() ([]byte, error) {
	return json.Marshal(req)
}

type responseErrorMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type responseMessage struct {
	ID     *string               `json:"id"`
	Result json.RawMessage       `json:"result"`
	Error  *responseErrorMessage `json:"error"`
}

type responseQueue struct {
	ID       string
	message  []byte
	response chan *responseMessage
}

func newResponseQueue(ID string, message []byte) *responseQueue {
	return &responseQueue{
		ID:       ID,
		message:  message,
		response: make(chan *responseMessage),
	}
}

func newRequestMessage(method string, params interface{}) *requestMessage {
	return &requestMessage{
		ID:     cuid.New(),
		Method: method,
		Params: params,
	}
}

func (e *EventListener) sendRequest(req *requestMessage) (*responseMessage, error) {
	message, err := req.toJSON()
	if err != nil {
		return nil, err
	}
	messageLog.Debug("sendRequest", zap.Any("request", json.RawMessage(message)))

	wait := newResponseQueue(req.ID, message)

	select {
	case e.send <- wait:
	case _, ok := <-e.done:
		if !ok {
			return nil, ListenerClosed
		}
	}

	select {
	case response := <-wait.response:
		return response, nil
	case <-time.After(e.ResponseWait):
		return nil, fmt.Errorf("request timeout: %+v", req)
	}
}

func (e *EventListener) processMessage(message []byte) error {
	response := new(responseMessage)
	if err := json.Unmarshal(message, response); err != nil {
		return err
	}
	messageLog.Debug("processMessage", zap.Any("response", json.RawMessage(message)))

	if response.ID != nil {
		e.response <- response
	} else {
		eventMessage := new(EventMessage)
		if err := json.Unmarshal(response.Result, eventMessage); err != nil {
			return err
		}

		if e.event != nil {
			e.event <- eventMessage
		}

		e.updateOffset(eventMessage.Events)
	}
	return nil
}

func (e *EventListener) updateOffset(events []*Event) {
	e.Lock()
	defer e.Unlock()

	for _, event := range events {
		event := event
		if _, ok := e.subscriptions[event.EventType]; ok {
			e.subscriptions[event.EventType] = event.Offset + 1
		}
	}
}
