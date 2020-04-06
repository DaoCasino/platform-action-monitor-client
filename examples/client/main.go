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

	listener.Subscribe(0, 0)

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
					log.Printf("%+v", event)
				}
			}
		}
	}(parentContext, events)

	<-done
	listener.Unsubscribe(0)
	cancel()
}
