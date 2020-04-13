package eventlistener

import (
	"context"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"time"
)

const (
	msgPumpStopped       = "pump stopped"
	msgPumpRunning       = "pump running"
	msgParentContextDone = "parent context done"
)

func (e *EventListener) readPump(parentContext context.Context) {
	log := pumpsLog.Named("readPump")

	defer func() {
		_ = e.conn.Close()
		log.Info(msgPumpStopped)
	}()

	log.Info(msgPumpRunning)

	e.conn.SetReadLimit(e.MessageSizeLimit)
	if err := e.conn.SetReadDeadline(time.Now().Add(e.PongWait)); err != nil {
		log.Error("conn.SetReadDeadline", zap.Error(err))
	}
	e.conn.SetPongHandler(func(string) error { return e.conn.SetReadDeadline(time.Now().Add(e.PongWait)) })

	for {
		select {
		case <-parentContext.Done():
			log.Debug(msgParentContextDone)
			return
		default:
			_, message, err := e.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error("conn.ReadMessage", zap.Error(err))
				}
				return
			}

			if err := e.processMessage(message); err != nil {
				log.Error("processMessage", zap.Error(err))
			}
		}
	}
}

func (e *EventListener) responsePump(ctx context.Context, send <-chan *responseQueue) {
	log := pumpsLog.Named("responsePump")
	process := make(map[string]chan *responseMessage)
	defer func() {
		for ID, ch := range process {
			if ch != nil {
				close(ch)
			}
			delete(process, ID)
		}

		log.Info(msgPumpStopped)
	}()

	log.Info(msgPumpStopped)
	for {
		select {
		case <-ctx.Done():
			log.Debug(msgParentContextDone)
			return
		case message, ok := <-send:
			if !ok {
				log.Debug("close send channel")
				return
			}
			if message.response != nil { // Add wait response
				process[message.ID] = message.response
			}

		case response, ok := <-e.response:
			if !ok {
				log.Debug("close response channel")
				return
			}
			ID := *response.ID
			if ch, ok := process[ID]; ok {
				if ch != nil {
					ch <- response
					close(ch)
				}
				delete(process, ID)
			}
		}
	}
}

func (e *EventListener) writePump(parentContext context.Context) {
	log := pumpsLog.Named("writePump")

	ticker := time.NewTicker(e.PingPeriod)
	waitResponse := make(chan *responseQueue)

	go e.responsePump(parentContext, waitResponse)

	defer func() {
		close(waitResponse)

		if e.event != nil {
			close(e.event) // <- events can not be expected
			log.Debug("close event channel")
		}

		ticker.Stop()
		_ = e.conn.Close()

		log.Info(msgPumpStopped)
	}()

	log.Info(msgPumpRunning)

	for {
		select {
		case <-parentContext.Done():
			log.Debug(msgParentContextDone)
			return

		case message, ok := <-e.send:
			if !ok {
				// The session closed the channel.
				log.Debug("close send channel")
				if err := closeMessage(e.conn, e.WriteWait); err != nil {
					log.Error("closeMessage", zap.Error(err))
				}
				return
			}
			err := writeMessage(e.conn, e.WriteWait, message.message)
			if err != nil {
				log.Error("writeMessage", zap.Error(err))
				return
			}
			if message.response != nil {
				waitResponse <- message
			}
		case <-ticker.C:
			if err := pingMessage(e.conn, e.WriteWait); err != nil {
				log.Error("pingMessage", zap.Error(err))
			}
		}
	}
}
