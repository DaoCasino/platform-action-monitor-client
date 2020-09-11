package eventlistener

import (
	"context"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"net/url"
	"time"
)

// Run starts the action listener tries to reconnect and restore subscriptions in case of an error.
// Run in goroutine because this method is blocking
func (e *EventListener) Run(parentContext context.Context) {
	log := pumpsLog.Named("reconnect")
	defer func() {
		e.Close()
		log.Debug("listener close")
	}()

	u := url.URL{Scheme: "ws", Host: e.Addr, Path: "/"}
	attempt := 1

	for {
		log.Debug("connection", zap.Int("attempt", attempt))

		g, ctx := errgroup.WithContext(parentContext)
		var err error

		e.conn, _, err = websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
		if err == nil {
			attempt = 0

			log.Debug("connected", zap.String("url", u.String()))
			g.Go(func() error {
				return e.readPump(ctx)
			})
			g.Go(func() error {
				return e.writePump(ctx)
			})

			// Group by offset
			subscriptions := make(map[uint64][]EventType)

			e.Lock()
			for eventType, offset := range e.subscriptions {
				if eventTypes, ok := subscriptions[offset]; ok {
					subscriptions[offset] = append(eventTypes, eventType)
				} else {
					subscriptions[offset] = []EventType{eventType}
				}
			}
			e.Unlock()

			for offset, eventTypes := range subscriptions {
				if _, err := e.BatchSubscribe(eventTypes, offset); err != nil {
					log.Error("batchSubscribe error", zap.Error(err))
					break
				}
			}

			err = g.Wait()
			if err != nil {
				log.Error("wait error", zap.Error(err))
			}
		} else {
			log.Error("connection error", zap.String("url", u.String()), zap.Error(err))
		}

		attempt++
		if attempt > e.ReconnectionAttempts {
			return
		}

		select {
		case <-parentContext.Done():
			log.Debug("parent context done")
			return
		case <-time.After(e.ReconnectionDelay):
		}
	}
}
