package transport

import (
	"context"
	"sync"
)

type MoveStep struct{ ID, From, To string }
type StepResult struct {
	ID  string
	Err error
}
type ResultLedger struct {
	mu      sync.Mutex
	results []StepResult
}

func (l *ResultLedger) Append(r StepResult) {
	l.mu.Lock()
	l.results = append(l.results, r)
	l.mu.Unlock()
}
func (l *ResultLedger) Snapshot() []StepResult {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]StepResult, len(l.results))
	copy(out, l.results)
	return out
}

type StepWorker func(context.Context, MoveStep) error
