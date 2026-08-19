package reticles

import (
	"context"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"strings"
	"time"
)

type MountService struct {
	registry *MountRegistry
	events   platform.EventSink
	clock    platform.Clock
}

func NewMountService(r *MountRegistry, e platform.EventSink, c platform.Clock) *MountService {
	return &MountService{registry: r, events: e, clock: c}
}
func (s *MountService) MountForExposure(ctx context.Context, reticleID, toolID, slotID, zone string) (Mount, error) {
	if strings.TrimSpace(reticleID) == "" || strings.TrimSpace(toolID) == "" || strings.TrimSpace(slotID) == "" || strings.TrimSpace(zone) == "" {
		return Mount{}, platform.ErrInvariant
	}
	var registry interface {
		Mount(context.Context, Mount) (Mount, error)
	} = s.registry
	if registry == nil {
		registry = NewMountRegistry()
	}
	m, err := registry.Mount(ctx, Mount{ReticleID: reticleID, ToolID: toolID, SlotID: slotID, Zone: zone})
	if err != nil {
		return Mount{}, platform.Wrap("mount", "reticle", reticleID, err)
	}
	if err = s.events.Publish(ctx, platform.Event{Topic: "reticles.mounted", Key: reticleID, Version: m.Generation, At: s.clock.Now(), Attributes: map[string]string{"slot": slotID, "tool": toolID, "zone": zone}}); err != nil {
		rbCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.registry.Unmount(rbCtx, m)
		return Mount{}, platform.Wrap("publish", "reticle", reticleID, err)
	}
	return m, nil
}
