package qualifications

import "context"

type ProbeGate struct{ ctx context.Context }

func NewProbeGate(ctx context.Context) *ProbeGate { return &ProbeGate{ctx: context.Background()} }
func (g *ProbeGate) Allow() error                 { return g.ctx.Err() }
