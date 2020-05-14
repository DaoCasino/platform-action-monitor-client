package main

import (
	"context"
	"flag"
	"github.com/DaoCasino/platform-action-monitor-client"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var addr = flag.String("addr", ":8888", "action monitor service address")

func main() {
	flag.Parse()

	parentContext, cancel := context.WithCancel(context.Background())
	events := make(chan *eventlistener.EventMessage)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	listener := eventlistener.NewEventListener(*addr, events)
	if err := listener.ListenAndServe(parentContext); err != nil {
		log.Fatal(err)
	}

	defer func() {
		listener.Close()
		cancel()
	}()

	go func(ctx context.Context, events <-chan *eventlistener.EventMessage) {
		for {
			select {
			case <-ctx.Done():
				return
			case eventMessage, ok := <-events:
				if !ok {
					return
				}
				for _, event := range eventMessage.Events {
					log.Printf("event: offset=%d, type=%d\n", event.Offset, event.EventType)
				}
			}
		}
	}(parentContext, events)

	eventTypes := []eventlistener.EventType{0, 1, 2, 3, 4, 5, 6}

	if _, err := listener.BatchSubscribe(eventTypes, 0); err != nil {
		log.Fatal(err)
	}

	<-done

	if _, err := listener.BatchUnsubscribe(eventTypes); err != nil {
		log.Fatal(err)
	}
}
