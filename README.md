# Semiconductor Fab Lot Dispatch Service

A Go control-plane for wafer-lot dispatch in a semiconductor fabrication facility. The service models lot readiness, tools, chambers, recipes, reticles, carriers, AMHS transport, reservations, qualifications, maintenance, contamination zones, holds, rework, sampling, routing, batching, capacity, energy windows, purge cycles, shift handoff, alarms, recovery, and audit records.

## Architecture
- `cmd/fab-dispatch`: HTTP service entry point.
- `internal/api`: health, dashboard, lot registration and dispatch endpoints.
- `internal/<context>`: domain model, versioned repository, policy and transactional service.
- `internal/platform`: clocks, typed errors and event abstractions.
- `frontend`: dependency-free operations console.

## Run
```bash
npm --prefix frontend run build
go run ./cmd/fab-dispatch
```
The service listens on `:8080`; set `HTTP_ADDR` to override it.

## Test
```bash
go test ./...
go build ./...
```

## HTTP endpoints
- `GET /healthz`
- `GET /api/dashboard?fab=fab-a`
- `POST /api/lots`
- `POST /api/lots/{id}/dispatch`
