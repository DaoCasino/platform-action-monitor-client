package eventlistener

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEventListener_reconnect(t *testing.T) {
	parentContext, cancel := context.WithCancel(context.Background())
	events := make(chan *EventMessage)
	defer func() {
		cancel()
	}()

	listener := NewEventListener(":8888", events)
	go listenAndServe(parentContext, listener)

	ok, err := listener.Subscribe(0, 0)
	require.NoError(t, err)
	assert.True(t, ok)

	for event := range events {
		t.Logf("%d %+v\n", len(event.Events), event.Events[0])
		t.Logf("%s\n", event.Events[0].Data)
	}
}
