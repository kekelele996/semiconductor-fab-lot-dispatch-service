package platform

import (
	"context"
	"sync"
	"time"
)

type Event struct {
	Topic, Key string
	Version    int64
	At         time.Time
	Attributes map[string]string
}
type EventSink interface {
	Publish(context.Context, Event) error
}
type MemoryEventSink struct {
	mu     sync.RWMutex
	events []Event
}

func (s *MemoryEventSink) Publish(ctx context.Context, e Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e.Attributes = CloneMap(e.Attributes)
	s.events = append(s.events, e)
	return nil
}
func (s *MemoryEventSink) Events() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Event, len(s.events))
	copy(out, s.events)
	return out
}
func CloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
