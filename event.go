package eventlistener

import (
	"encoding/json"
	"fmt"
)

type EventType int

type Event struct {
	Offset    uint64          `json:"offset"`
	Sender    string          `json:"sender"`
	CasinoID  uint64          `json:"casino_id"`
	GameID    uint64          `json:"game_id"`
	RequestID uint64          `json:"req_id"`
	EventType EventType       `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type EventMessage struct {
	Offset uint64   `json:"offset"` // last event.offset
	Events []*Event `json:"events"`
}

func (e EventType) ToString() string {
	return fmt.Sprintf("event_%d", e)
}
