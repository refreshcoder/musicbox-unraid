package ws

import "time"

type Event struct {
	Type string `json:"type"`
	TS   int64  `json:"ts"`
	Data any    `json:"data"`
}

func NewEvent(typ string, data any) Event {
	return Event{Type: typ, TS: time.Now().Unix(), Data: data}
}

