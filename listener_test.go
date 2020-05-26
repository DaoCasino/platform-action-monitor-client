package eventlistener

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	waitEventsTimeout = time.Second
	event             = 0
	offset            = 0
)

func TestNewEventListener(t *testing.T) {
	addr := ":1234"
	listener := NewEventListener(addr, nil)

	assert.Equal(t, addr, listener.Addr)
	assert.Equal(t, int64(messageSizeLimit), listener.MessageSizeLimit)
	assert.Equal(t, writeWait, listener.WriteWait)
	assert.Equal(t, pongWait, listener.PongWait)
	assert.Equal(t, pingPeriod, listener.PingPeriod)
	assert.Nil(t, listener.event)
}

func TestEventListener_ListenAndServe(t *testing.T) {
	parentContext := context.Background()

	listener := NewEventListener(":1234", nil)
	err := listener.ListenAndServe(parentContext)
	require.Error(t, err)
}

func TestEventListener_Subscribe(t *testing.T) {
	parentContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener := NewEventListener(*addr, nil)
	listener.SetToken(*token)

	if err := listener.ListenAndServe(parentContext); err != nil {
		t.Skip("listen error", err.Error())
		return
	}

	ok, err := listener.Subscribe(event, offset)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEventListener_Unsubscribe(t *testing.T) {
	parentContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener := NewEventListener(*addr, nil)
	listener.SetToken(*token)

	if err := listener.ListenAndServe(parentContext); err != nil {
		t.Skip("listen error", err.Error())
		return
	}

	// Unsubscribing from a topic that is not subscribed to
	ok, err := listener.Unsubscribe(666)
	require.Error(t, err)
	assert.False(t, ok)

	ok, err = listener.Subscribe(event, offset)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = listener.Unsubscribe(event)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEventListener_BatchSubscribe(t *testing.T) {
	parentContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener := NewEventListener(*addr, nil)
	listener.SetToken(*token)

	if err := listener.ListenAndServe(parentContext); err != nil {
		t.Skip("listen error", err.Error())
		return
	}

	ok, err := listener.BatchSubscribe([]EventType{event}, offset)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEventListener_BatchUnsubscribe(t *testing.T) {
	parentContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener := NewEventListener(*addr, nil)
	listener.SetToken(*token)

	if err := listener.ListenAndServe(parentContext); err != nil {
		t.Skip("listen error", err.Error())
		return
	}

	// Unsubscribing from a topic that is not subscribed to
	ok, err := listener.BatchUnsubscribe([]EventType{666})
	require.Error(t, err)
	assert.False(t, ok)

	types := []EventType{event}

	ok, err = listener.BatchSubscribe(types, offset)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = listener.BatchUnsubscribe(types)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEventListener_EventsMessage(t *testing.T) {
	parentContext, cancel := context.WithCancel(context.Background())
	events := make(chan *EventMessage)
	defer func() {
		cancel()
	}()

	listener := NewEventListener(*addr, events)
	listener.SetToken(*token)

	if err := listener.ListenAndServe(parentContext); err != nil {
		t.Skip("listen error", err.Error())
		return
	}

	ok, err := listener.Subscribe(event, offset)
	require.NoError(t, err)
	assert.True(t, ok)

	waitContext, cancelWait := context.WithTimeout(parentContext, waitEventsTimeout)

loop:
	for {
		select {
		case <-waitContext.Done():
			t.Log("no events")
			break loop
		case event := <-events:
			if len(event.Events) == 0 {
				t.Error("received 0 events; want more")
				break loop
			}

			t.Logf("%d %+v\n", len(event.Events), event.Events[0])
			t.Logf("%s\n", event.Events[0].Data)
			break loop

		}
	}

	cancelWait()
}
