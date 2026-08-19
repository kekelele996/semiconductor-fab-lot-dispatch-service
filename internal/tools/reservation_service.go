package tools

import (
	"context"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"time"
)

type ReservationService struct {
	table  *ReservationTable
	events platform.EventSink
	clock  platform.Clock
}

func NewReservationService(t *ReservationTable, e platform.EventSink, c platform.Clock) *ReservationService {
	return &ReservationService{table: t, events: e, clock: c}
}
func (s *ReservationService) ReserveRoute(ctx context.Context, owner string, primary, backup []string) (ToolReservation, error) {
	ids := append(append([]string(nil), primary...), backup...)
	r, err := s.table.ReserveAll(ctx, owner, ids)
	if err != nil {
		return ToolReservation{}, platform.Wrap("reserve", "tool-route", owner, err)
	}
	attrs := map[string]string{"owner": owner, "tool_count": string(rune(len(r.ToolIDs) + '0'))}
	if err = s.events.Publish(ctx, platform.Event{Topic: "tools.route-reserved", Key: owner, Version: r.Generation, At: s.clock.Now(), Attributes: attrs}); err != nil {
		rbCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_ = s.table.Release(rbCtx, r)
		return ToolReservation{}, platform.Wrap("publish", "tool-route", owner, err)
	}
	return r, nil
}
func (s *ReservationService) ReleaseRoute(ctx context.Context, r ToolReservation) error {
	if err := s.table.Release(ctx, r); err != nil {
		return err
	}
	return s.events.Publish(ctx, platform.Event{Topic: "tools.route-released", Key: r.Owner, Version: r.Generation, At: s.clock.Now()})
}
