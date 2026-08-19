package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"semiconductor-fab-lot-dispatch-service/internal/api"
	"semiconductor-fab-lot-dispatch-service/internal/dispatch"
	"semiconductor-fab-lot-dispatch-service/internal/lots"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"semiconductor-fab-lot-dispatch-service/internal/tools"
	"syscall"
	"time"
)

func main() {
	l := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	c := platform.RealClock{}
	e := &platform.MemoryEventSink{}
	s := api.NewServer(l, lots.NewService(lots.NewRepository(), lots.DefaultPolicy(), c, e), tools.NewService(tools.NewRepository(), tools.DefaultPolicy(), c, e), dispatch.NewService(dispatch.NewRepository(), dispatch.DefaultPolicy(), c, e))
	h := &http.Server{Addr: env("HTTP_ADDR", ":8080"), Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		l.Info("listening", "addr", h.Addr)
		if err := h.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Error("serve", "error", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	x, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = h.Shutdown(x)
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
