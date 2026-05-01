package httpapi

import (
	"context"
	"net/http"

	"nhooyr.io/websocket"

	"github.com/refreshcoder/musicbox-unraid/internal/ws"
)

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	c := ws.NewClient(conn)
	s.ws.Add(c)
	defer s.ws.Remove(c)

	conn.SetReadLimit(1024 * 32)

	_ = c.Run(ctx)
}
