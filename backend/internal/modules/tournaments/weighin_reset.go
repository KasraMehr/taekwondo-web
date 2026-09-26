package tournaments

import (
	"context"
	"time"
)

func (s *tournamentService) ResetWeighIn(ctx context.Context, id, athleteID string) (*TournamentAthlete, error) {
	t, i, err := s.loadTournamentAthleteForWeighIn(ctx, id, athleteID)
	if err != nil {
		return nil, err
	}
	if athleteIsReferencedByMatches(t.Matches, athleteID) {
		return nil, ErrAthleteLocked
	}
	t.Athletes[i].WeighIn = newPendingWeighIn()
	t.Athletes[i].WeighedIn = false
	t.UpdatedAt = time.Now().UTC()
	if err = s.repository.Update(ctx, t); err != nil {
		return nil, err
	}
	return &t.Athletes[i], nil
}
func (s *tournamentService) ClearWeighInSignature(ctx context.Context, id, athleteID string) (*TournamentAthlete, error) {
	t, i, err := s.loadTournamentAthleteForWeighIn(ctx, id, athleteID)
	if err != nil {
		return nil, err
	}
	a := &t.Athletes[i]
	if a.WeighIn == nil || a.WeighIn.Signature == nil {
		return nil, ErrInvalidSignature
	}
	w := *a.WeighIn
	w.Signature = nil
	a.WeighIn = &w
	t.UpdatedAt = time.Now().UTC()
	if err = s.repository.Update(ctx, t); err != nil {
		return nil, err
	}
	return a, nil
}
