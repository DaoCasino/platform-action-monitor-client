package eventlistener

import (
	"context"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"net/url"
	"time"
)

func listenAndServe(parentContext context.Context, listener *EventListener) {
	log := pumpsLog.Named("reconnect")

	u := url.URL{Scheme: "ws", Host: listener.Addr, Path: "/"}
	g, ctx := errgroup.WithContext(parentContext)

	for attempts := 0; attempts < listener.ReconnectionAttempts; attempts++ {
		log.Debug("connection", zap.Int("attempts", attempts))

		var err error

		listener.conn, _, err = websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
		if err == nil {
			log.Debug("connected", zap.String("url", u.String()))
			g.Go(func() error {
				return listener.readPump(ctx)
			})
			g.Go(func() error {
				return listener.writePump(ctx)
			})
			g.Go(func() error {
				for eventType, offset := range listener.subscriptions {
					if _, err := listener.Subscribe(eventType, offset); err != nil {
						return err
					}
				}
				return nil
			})

			err = g.Wait()
			if err != nil {
				log.Error("wait error", zap.Error(err))
			}
		} else {
			log.Error("connection error", zap.String("url", u.String()), zap.Error(err))
		}

		select {
		case <-parentContext.Done():
			break
		case <-time.After(reconnectionDelay):
			continue
		}
	}
}
