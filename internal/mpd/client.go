package mpd

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Addr string
}

func (c Client) cmdLines(ctx context.Context, cmd string) ([]string, error) {
	if c.Addr == "" {
		return nil, fmt.Errorf("mpd addr is empty")
	}

	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", c.Addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "OK") {
		return nil, fmt.Errorf("mpd handshake: %s", strings.TrimSpace(line))
	}

	if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
		return nil, err
	}

	var out []string
	for {
		l, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		l = strings.TrimSpace(l)
		if l == "OK" {
			return out, nil
		}
		if strings.HasPrefix(l, "ACK") {
			return nil, fmt.Errorf("mpd ack: %s", l)
		}
		out = append(out, l)
	}
}

func (c Client) cmd(ctx context.Context, cmd string) (map[string]string, error) {
	if c.Addr == "" {
		return nil, fmt.Errorf("mpd addr is empty")
	}

	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", c.Addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "OK") {
		return nil, fmt.Errorf("mpd handshake: %s", strings.TrimSpace(line))
	}

	if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
		return nil, err
	}

	out := map[string]string{}
	for {
		l, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		l = strings.TrimSpace(l)
		if l == "OK" {
			return out, nil
		}
		if strings.HasPrefix(l, "ACK") {
			return nil, fmt.Errorf("mpd ack: %s", l)
		}
		parts := strings.SplitN(l, ": ", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
}

type Status struct {
	State       string
	Volume      int
	Song        int
	QueueLength int
	ElapsedMs   int64
	DurationMs  int64
	BitrateKbps int
}

func (c Client) Status(ctx context.Context) (Status, error) {
	m, err := c.cmd(ctx, "status")
	if err != nil {
		return Status{}, err
	}

	var s Status
	s.State = m["state"]
	s.Volume, _ = strconv.Atoi(m["volume"])
	s.BitrateKbps, _ = strconv.Atoi(m["bitrate"])
	s.Song, _ = strconv.Atoi(m["song"])
	s.QueueLength, _ = strconv.Atoi(m["playlistlength"])

	if v := m["elapsed"]; v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			s.ElapsedMs = int64(f * 1000)
		}
	}
	if v := m["duration"]; v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			s.DurationMs = int64(f * 1000)
		}
	}
	if v := m["time"]; v != "" && (s.DurationMs == 0 || s.ElapsedMs == 0) {
		parts := strings.SplitN(v, ":", 2)
		if len(parts) == 2 {
			el, _ := strconv.ParseInt(parts[0], 10, 64)
			du, _ := strconv.ParseInt(parts[1], 10, 64)
			if s.ElapsedMs == 0 {
				s.ElapsedMs = el * 1000
			}
			if s.DurationMs == 0 {
				s.DurationMs = du * 1000
			}
		}
	}

	return s, nil
}

type QueueItem struct {
	Pos        int    `json:"pos"`
	Path       string `json:"path"`
	Title      string `json:"title,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
}

func (c Client) Queue(ctx context.Context) ([]QueueItem, error) {
	lines, err := c.cmdLines(ctx, "playlistinfo")
	if err != nil {
		return nil, err
	}

	var out []QueueItem
	var cur *QueueItem

	flush := func() {
		if cur == nil {
			return
		}
		out = append(out, *cur)
		cur = nil
	}

	for _, l := range lines {
		parts := strings.SplitN(l, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		k, v := parts[0], parts[1]
		switch k {
		case "file":
			flush()
			cur = &QueueItem{Path: v}
		case "Pos":
			if cur != nil {
				cur.Pos, _ = strconv.Atoi(v)
			}
		case "Title":
			if cur != nil {
				cur.Title = v
			}
		case "Time":
			if cur != nil {
				if sec, err := strconv.ParseInt(v, 10, 64); err == nil {
					cur.DurationMs = sec * 1000
				}
			}
		}
	}
	flush()
	return out, nil
}

func (c Client) Add(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path is empty")
	}
	_, err := c.cmd(ctx, "add "+quote(path))
	return err
}

func (c Client) Clear(ctx context.Context) error {
	_, err := c.cmd(ctx, "clear")
	return err
}

func (c Client) Delete(ctx context.Context, pos int) error {
	if pos < 0 {
		return fmt.Errorf("invalid pos: %d", pos)
	}
	_, err := c.cmd(ctx, fmt.Sprintf("delete %d", pos))
	return err
}

func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		if r == '\\' || r == '"' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
	return b.String()
}

func (c Client) Play(ctx context.Context) error {
	_, err := c.cmd(ctx, "play")
	return err
}

func (c Client) Pause(ctx context.Context, pause bool) error {
	arg := "0"
	if pause {
		arg = "1"
	}
	_, err := c.cmd(ctx, "pause "+arg)
	return err
}

func (c Client) Next(ctx context.Context) error {
	_, err := c.cmd(ctx, "next")
	return err
}

func (c Client) Prev(ctx context.Context) error {
	_, err := c.cmd(ctx, "previous")
	return err
}

func (c Client) SetVol(ctx context.Context, vol int) error {
	if vol < 0 || vol > 100 {
		return fmt.Errorf("invalid volume: %d", vol)
	}
	_, err := c.cmd(ctx, fmt.Sprintf("setvol %d", vol))
	return err
}

func (c Client) SeekMs(ctx context.Context, positionMs int64) error {
	sec := positionMs / 1000
	_, err := c.cmd(ctx, fmt.Sprintf("seekcur %d", sec))
	return err
}
