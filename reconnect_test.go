package eventlistener

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEventListener_reconnectError(t *testing.T) {
	parentContext, cancel := context.WithCancel(context.Background())
	events := make(chan *EventMessage)
	defer func() {
		cancel()
	}()

	listener := NewEventListener(":12345", events)
	listener.ReconnectionAttempts = 1
	go listener.Run(parentContext)

	ok, err := listener.Subscribe(0, 0)
	assert.Equal(t, ListenerClosed, err)
	assert.False(t, ok)
}
