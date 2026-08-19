package shifts

import (
	"context"
	"fmt"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
)

type Service struct {
	repo   *Repository
	policy Policy
	clock  platform.Clock
	events platform.EventSink
}

func NewService(r *Repository, p Policy, c platform.Clock, e platform.EventSink) *Service {
	return &Service{repo: r, policy: p, clock: c, events: e}
}
func (s *Service) Register(ctx context.Context, x ShiftHandoff) (ShiftHandoff, error) {
	if x.State == "" {
		x.State = StatePrepared
	}
	now := s.clock.Now()
	if x.ReadyAt.IsZero() {
		x.ReadyAt = now
	}
	x.UpdatedAt = now
	if err := s.policy.Validate(x); err != nil {
		return ShiftHandoff{}, platform.Wrap("register", "shifts", x.ID, err)
	}
	saved, err := s.repo.Create(ctx, x)
	if err != nil {
		return ShiftHandoff{}, err
	}
	if err = s.events.Publish(ctx, platform.Event{Topic: "shifts.registered", Key: saved.ID, Version: saved.Version, At: now, Attributes: platform.CloneMap(saved.Constraints)}); err != nil {
		_ = s.repo.Delete(context.Background(), saved.ID, saved.Version)
		return ShiftHandoff{}, platform.Wrap("publish", "shifts", saved.ID, err)
	}
	return saved, nil
}
func (s *Service) Apply(ctx context.Context, cmd Command) (ShiftHandoff, error) {
	before, err := s.repo.Get(ctx, cmd.ID)
	if err != nil {
		return ShiftHandoff{}, err
	}
	target := cmd.TargetState
	if target == "" {
		target = before.State
	}
	if !s.policy.CanTransition(before.State, target) {
		return ShiftHandoff{}, platform.Wrap("transition", "shifts", cmd.ID, platform.ErrInvalidTransition)
	}
	updated, err := s.repo.Update(ctx, cmd.ID, cmd.ExpectedVersion, func(next *ShiftHandoff) error {
		next.State = target
		if cmd.Priority != 0 {
			next.Priority = cmd.Priority
		}
		if cmd.Quantity != 0 {
			next.Quantity = cmd.Quantity
		}
		if !cmd.Deadline.IsZero() {
			next.Deadline = cmd.Deadline
		}
		for k, v := range cmd.Constraints {
			if next.Constraints == nil {
				next.Constraints = map[string]string{}
			}
			next.Constraints[k] = v
		}
		next.UpdatedAt = s.clock.Now()
		return s.policy.Validate(*next)
	})
	if err != nil {
		return ShiftHandoff{}, err
	}
	attrs := map[string]string{"from": before.State, "to": updated.State, "reason": cmd.Reason}
	if err = s.events.Publish(ctx, platform.Event{Topic: "shifts.transitioned", Key: updated.ID, Version: updated.Version, At: updated.UpdatedAt, Attributes: attrs}); err != nil {
		_, rb := s.repo.Restore(context.Background(), before, updated.Version)
		if rb != nil {
			return ShiftHandoff{}, fmt.Errorf("publish: %w; rollback: %v", err, rb)
		}
		return ShiftHandoff{}, platform.Wrap("publish", "shifts", updated.ID, err)
	}
	return updated, nil
}
func (s *Service) Candidates(ctx context.Context, fab string) ([]Candidate, error) {
	items, err := s.repo.List(ctx, fab)
	if err != nil {
		return nil, err
	}
	return Rank(s.policy, items, s.clock.Now()), nil
}
func (s *Service) ReserveBest(ctx context.Context, fab, target string) (Candidate, ShiftHandoff, error) {
	ranked, err := s.Candidates(ctx, fab)
	if err != nil {
		return Candidate{}, ShiftHandoff{}, err
	}
	for _, c := range ranked {
		if !c.Feasible {
			continue
		}
		x, e := s.Apply(ctx, Command{ID: c.ID, ExpectedVersion: c.Version, TargetState: target, Reason: "dispatch-reservation"})
		if e == nil {
			return c, x, nil
		}
		if ctx.Err() != nil {
			return Candidate{}, ShiftHandoff{}, ctx.Err()
		}
	}
	return Candidate{}, ShiftHandoff{}, platform.Wrap("reserve", "shifts", fab, platform.ErrCapacity)
}
func (s *Service) Snapshot(ctx context.Context, fab string) (Snapshot, error) {
	x, err := s.repo.Snapshot(ctx, fab)
	if err != nil {
		return Snapshot{}, err
	}
	x.GeneratedAt = s.clock.Now()
	return x, nil
}
