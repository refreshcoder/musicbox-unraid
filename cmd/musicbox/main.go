package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/refreshcoder/musicbox-unraid/internal/httpapi"
)

func main() {
	addr := envOr("MUSICBOX_ADDR", ":8080")
	staticDir := envOr("MUSICBOX_STATIC_DIR", "web/dist")
	mpdAddr := os.Getenv("MUSICBOX_MPD_ADDR")
	musicDir := envOr("MUSICBOX_MUSIC_DIR", "/srv/music")

	s, err := httpapi.NewServer(httpapi.Options{
		StaticDir: staticDir,
		MPDAddr:   mpdAddr,
		MusicDir:  musicDir,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.Start(ctx)

	server := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
			cancel()
		}
	}()

	waitForSignal()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
}

func envOr(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func waitForSignal() {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
