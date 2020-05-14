package main

import (
	"context"
	"flag"
	"github.com/DaoCasino/platform-action-monitor-client"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var addr = flag.String("addr", ":8888", "action monitor service address")

func init() {
	if os.Getenv("DEBUG") != "" {
		eventlistener.EnableDebugLogging()
	}
}

func main() {
	flag.Parse()

	parentContext, cancel := context.WithCancel(context.Background())
	events := make(chan *eventlistener.EventMessage)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	listener := eventlistener.NewEventListener(*addr, events)
	// setup reconnection options
	listener.ReconnectionAttempts = 5
	listener.ReconnectionDelay = 5 * time.Second

	defer func() {
		cancel()
		time.Sleep(time.Second) // wait close routines
	}()

	go listener.Run(parentContext)

	canceled := make(chan struct{}, 1)
	go func(ctx context.Context, events <-chan *eventlistener.EventMessage, done chan struct{}) {
		defer func() {
			done <- struct{}{}
			close(done)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case eventMessage, ok := <-events:
				if !ok {
					return
				}
				for _, event := range eventMessage.Events {
					log.Printf("%+v %s\n", event, event.Data)
				}
			}
		}
	}(parentContext, events, canceled)

	eventTypes := []eventlistener.EventType{0, 1, 2, 3, 4, 5, 6}

	if _, err := listener.BatchSubscribe(eventTypes, 0); err != nil {
		log.Fatal(err)
	}

	select {
	case <-done:
	case <-canceled:
	}

	if _, err := listener.BatchUnsubscribe(eventTypes); err != nil {
		log.Println(err)
	}
}
