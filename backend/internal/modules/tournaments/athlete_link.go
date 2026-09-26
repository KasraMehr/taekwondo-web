package tournaments

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
)

// AthleteProfile is identity data; weight, ranking and weigh-in belong to an entry.
// Identity and club are snapshotted on entry to preserve historical results.
type AthleteProfile struct {
	ID, Name, Club, Coach string
	Gender                *string
	Active                bool
}
type AthleteProfileRepository interface {
	GetAthleteProfile(context.Context, string) (*AthleteProfile, error)
}

func (r *tournamentRepository) GetAthleteProfile(ctx context.Context, id string) (*AthleteProfile, error) {
	p := &AthleteProfile{}
	err := r.db.QueryRow(ctx, `SELECT a.id::text,a.name,coalesce(c.name,''),coalesce(c.coach_id::text,''),a.gender,a.is_active FROM athletes a LEFT JOIN clubs c ON c.id=a.club_id AND c.organization_id=a.organization_id WHERE a.id=$1 AND a.organization_id=$2 FOR SHARE OF a`, id, r.organizationID).Scan(&p.ID, &p.Name, &p.Club, &p.Coach, &p.Gender, &p.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAthleteNotFound
	}
	return p, err
}
func (s *tournamentService) resolveProfile(ctx context.Context, t *Tournament, in *AddAthleteInput) error {
	if in.ProfileID == nil {
		return nil
	}
	if err := validateRequiredUUID("profile id", *in.ProfileID); err != nil {
		return err
	}
	source, ok := s.repository.(AthleteProfileRepository)
	if !ok {
		return fmt.Errorf("%w: profile lookup is unavailable", ErrInvalidTournamentInput)
	}
	p, err := source.GetAthleteProfile(ctx, *in.ProfileID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrAthleteNotFound
	}
	if !p.Active {
		return fmt.Errorf("%w: athlete profile is inactive", ErrInvalidTournamentInput)
	}
	if p.Gender == nil || *p.Gender != string(t.Gender) {
		return fmt.Errorf("%w: profile gender does not match tournament", ErrInvalidTournamentInput)
	}
	for _, a := range t.Athletes {
		if a.ProfileID != nil && *a.ProfileID == p.ID {
			return ErrDuplicateAthlete
		}
	}
	in.Name = p.Name
	in.Club = p.Club
	in.Coach = p.Coach
	if in.ID == "" {
		in.ID = p.ID
	}
	return nil
}
