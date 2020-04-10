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
		cancel()
	}()

	if _, err := listener.Subscribe(0, 0); err != nil {
		log.Fatal(err)
	}

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
					log.Printf("%+v %s\n", event, event.Data)
				}
			}
		}
	}(parentContext, events)

	<-done

	if _, err := listener.Unsubscribe(0); err != nil {
		log.Fatal(err)
	}
}
