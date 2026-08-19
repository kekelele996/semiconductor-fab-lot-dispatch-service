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
	l.results = append(l.results, r)
}
func (l *ResultLedger) Snapshot() []StepResult {
	return l.results
}

type StepWorker func(context.Context, MoveStep) error
