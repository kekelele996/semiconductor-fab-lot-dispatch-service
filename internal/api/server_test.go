package api

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"semiconductor-fab-lot-dispatch-service/internal/dispatch"
	"semiconductor-fab-lot-dispatch-service/internal/lots"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"semiconductor-fab-lot-dispatch-service/internal/tools"
	"testing"
)

func TestHealth(t *testing.T) {
	c := platform.RealClock{}
	e := &platform.MemoryEventSink{}
	s := NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)), lots.NewService(lots.NewRepository(), lots.DefaultPolicy(), c, e), tools.NewService(tools.NewRepository(), tools.DefaultPolicy(), c, e), dispatch.NewService(dispatch.NewRepository(), dispatch.DefaultPolicy(), c, e))
	r := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}
}
