package tournaments

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ImportTournamentInput contains the portable tournament data from an Excel backup.
// Identity, organization, league linkage and revision always come from the target.
type ImportTournamentInput struct {
	Name               string              `json:"name"`
	Date               string              `json:"date"`
	Courts             int                 `json:"courts"`
	Gender             Gender              `json:"gender"`
	AgeCategory        AgeCategory         `json:"ageCategory"`
	Format             TournamentFormat    `json:"format"`
	Athletes           []TournamentAthlete `json:"athletes"`
	Matches            []Match             `json:"matches"`
	CourtAssignment    map[string][]int    `json:"courtAssignment"`
	EliminationMatches []EliminationMatch  `json:"eliminationMatches"`
	Settings           *TournamentSettings `json:"settings"`
}

func (s *tournamentService) ImportSnapshot(ctx context.Context, id string, input ImportTournamentInput) (*Tournament, error) {
	date, err := parseTournamentDate(input.Date)
	if err != nil {
		return nil, err
	}
	if err := validateTournamentDefinition(input.Name, input.Courts, input.Gender, input.AgeCategory, input.Format); err != nil {
		return nil, err
	}
	t, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	athleteIDs := make(map[string]bool, len(input.Athletes))
	for _, athlete := range input.Athletes {
		if _, err := uuid.Parse(athlete.ID); err != nil || athleteIDs[athlete.ID] {
			return nil, fmt.Errorf("%w: invalid or duplicate athlete id", ErrInvalidTournamentInput)
		}
		if strings.TrimSpace(athlete.Name) == "" || !IsValidWeightCategory(input.AgeCategory, input.Gender, athlete.WeightCategory) {
			return nil, fmt.Errorf("%w: invalid athlete name or weight category", ErrInvalidTournamentInput)
		}
		athleteIDs[athlete.ID] = true
	}
	for _, match := range input.Matches {
		if _, err := uuid.Parse(match.ID); err != nil || !IsValidWeightCategory(input.AgeCategory, input.Gender, match.WeightCategory) {
			return nil, fmt.Errorf("%w: invalid match id or weight category", ErrInvalidTournamentInput)
		}
		if match.Court < 0 || match.Court > input.Courts {
			return nil, ErrInvalidCourt
		}
		for _, athleteID := range []*string{match.Athlete1ID, match.Athlete2ID, match.WinnerID} {
			if athleteID != nil && !athleteIDs[*athleteID] {
				return nil, fmt.Errorf("%w: match references an unknown athlete", ErrInvalidTournamentInput)
			}
		}
	}

	t.Name = strings.TrimSpace(input.Name)
	t.Date = date
	t.Courts = input.Courts
	t.Gender = input.Gender
	t.AgeCategory = input.AgeCategory
	t.Format = input.Format
	t.Athletes = input.Athletes
	t.Matches = input.Matches
	t.CourtAssignment = input.CourtAssignment
	t.EliminationMatches = input.EliminationMatches
	t.Settings = input.Settings
	normalizeTournamentCollections(t)
	assignAthleteNumbers(t.Athletes)
	if err := validateSettings(t, *t.Settings); err != nil {
		return nil, err
	}
	if err := lifecycleValidateBracket(t); err != nil {
		return nil, err
	}
	if err := s.repository.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("import tournament %s: %w", id, err)
	}
	return t, nil
}
