package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"semiconductor-fab-lot-dispatch-service/internal/dispatch"
	"semiconductor-fab-lot-dispatch-service/internal/lots"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"semiconductor-fab-lot-dispatch-service/internal/tools"
	"strings"
	"time"
)

type Server struct {
	logger   *slog.Logger
	lots     *lots.Service
	tools    *tools.Service
	dispatch *dispatch.Service
	started  time.Time
}

func NewServer(l *slog.Logger, ls *lots.Service, ts *tools.Service, ds *dispatch.Service) *Server {
	return &Server{logger: l, lots: ls, tools: ts, dispatch: ds, started: time.Now().UTC()}
}
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", s.health)
	m.HandleFunc("GET /api/dashboard", s.dashboard)
	m.HandleFunc("POST /api/lots", s.createLot)
	m.HandleFunc("POST /api/lots/{id}/dispatch", s.dispatchLot)
	m.Handle("/", http.FileServer(http.Dir("frontend/dist")))
	return requestLog(s.logger, m)
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok", "uptime_seconds": int(time.Since(s.started).Seconds())})
}
func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	a, e := s.lots.Snapshot(r.Context(), r.URL.Query().Get("fab"))
	if e != nil {
		writeError(w, e)
		return
	}
	b, e := s.tools.Snapshot(r.Context(), r.URL.Query().Get("fab"))
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"lots": a, "tools": b})
}
func (s *Server) createLot(w http.ResponseWriter, r *http.Request) {
	var x lots.Lot
	if e := json.NewDecoder(r.Body).Decode(&x); e != nil {
		writeJSON(w, 400, map[string]string{"error": e.Error()})
		return
	}
	x, e := s.lots.Register(r.Context(), x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 201, x)
}
func (s *Server) dispatchLot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		writeJSON(w, 400, map[string]string{"error": "missing lot id"})
		return
	}
	var c lots.Command
	if e := json.NewDecoder(r.Body).Decode(&c); e != nil {
		writeJSON(w, 400, map[string]string{"error": e.Error()})
		return
	}
	c.ID = id
	if c.TargetState == "" {
		c.TargetState = lots.StateRunning
	}
	x, e := s.lots.Apply(r.Context(), c)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, x)
}
func writeError(w http.ResponseWriter, e error) {
	n := 500
	switch {
	case errors.Is(e, platform.ErrNotFound):
		n = 404
	case errors.Is(e, platform.ErrConflict):
		n = 409
	case errors.Is(e, platform.ErrInvalidTransition), errors.Is(e, platform.ErrInvariant):
		n = 422
	case errors.Is(e, platform.ErrCapacity):
		n = 429
	}
	writeJSON(w, n, map[string]string{"error": e.Error()})
}
func writeJSON(w http.ResponseWriter, n int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(n)
	_ = json.NewEncoder(w).Encode(v)
}
func requestLog(l *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		at := time.Now()
		next.ServeHTTP(w, r)
		l.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(at))
	})
}
