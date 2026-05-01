package bluetooth

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

type Ctl struct {
	Path string
}

func (c Ctl) Run(ctx context.Context, args ...string) (string, error) {
	path := c.Path
	if path == "" {
		path = "bluetoothctl"
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

