package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func (m *Manager) runBV(ctx context.Context, t *Task, onUpdate func(*Task)) error {
	m.setStage(t, "download", 0.05, onUpdate)

	url := bvToURL(t.Input)
	if url == "" {
		return errors.New("invalid bv")
	}

	outTpl := filepath.Join(m.IncomingDir(), "%(title)s.%(ext)s")
	stdout, err := m.runner.Run(
		ctx,
		"yt-dlp",
		"--no-playlist",
		"--no-warnings",
		"-f",
		"bestaudio",
		"--extract-audio",
		"--audio-format",
		"m4a",
		"--print",
		"after_move:filepath",
		"-o",
		outTpl,
		url,
	)
	if err != nil {
		return fmt.Errorf("yt-dlp: %w", err)
	}

	src := parseLastNonEmptyLine(stdout)
	if src == "" {
		return errors.New("yt-dlp: missing output path")
	}

	m.setStage(t, "move", 0.85, onUpdate)

	dstName := filepath.Base(src)
	dst := filepath.Join(m.MusicDir(), dstName)
	if err := moveFile(src, dst); err != nil {
		return err
	}

	rel := dstName
	t.ResultPath = rel

	if m.mpd != nil {
		m.setStage(t, "rescan", 0.95, onUpdate)
		if err := m.mpd.Update(ctx); err != nil {
			return fmt.Errorf("mpd update: %w", err)
		}
	}

	m.setStage(t, "", 1, onUpdate)
	return nil
}

func bvToURL(bv string) string {
	bv = strings.TrimSpace(bv)
	if bv == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(bv), "http://") || strings.HasPrefix(strings.ToLower(bv), "https://") {
		return bv
	}
	if !strings.HasPrefix(strings.ToUpper(bv), "BV") {
		return ""
	}
	return "https://www.bilibili.com/video/" + bv
}

func parseLastNonEmptyLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		v := strings.TrimSpace(lines[i])
		if v != "" {
			return v
		}
	}
	return ""
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		return err
	}
	return nil
}

