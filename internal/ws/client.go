package ws

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"nhooyr.io/websocket"
)

type Client struct {
	conn   *websocket.Conn
	closed atomic.Bool
	ch     chan Event
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
		ch:   make(chan Event, 64),
	}
}

func (c *Client) Send(evt Event) {
	if c.closed.Load() {
		return
	}
	select {
	case c.ch <- evt:
	default:
	}
}

func (c *Client) Run(ctx context.Context) error {
	defer c.closed.Store(true)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt := <-c.ch:
			b, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			if err := c.conn.Write(ctx, websocket.MessageText, b); err != nil {
				return err
			}
		}
	}
}

