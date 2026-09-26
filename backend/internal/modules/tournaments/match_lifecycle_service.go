package tournaments

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// StartMatch starts a pending, non-bye match.
//
// Concurrency requirement:
// Loading, validation and persistence must run inside the same
// repository transaction/locking boundary in production.
func (s *tournamentService) StartMatch(
	ctx context.Context,
	tournamentID string,
	matchID string,
) (*Match, error) {
	if err := lifecycleValidateMatchID(matchID); err != nil {
		return nil, err
	}

	tournament, err := s.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf(
			"load tournament before starting match: %w",
			err,
		)
	}

	if err := lifecycleValidateBracket(tournament); err != nil {
		return nil, err
	}

	index, err := lifecycleFindMatchIndex(
		tournament.Matches,
		matchID,
	)
	if err != nil {
		return nil, err
	}

	match := &tournament.Matches[index]

	if lifecycleIsBye(match) {
		return nil, ErrByeMatch
	}

	if lifecycleHasResult(match) {
		return nil, ErrResultAlreadyExists
	}

	if status := lifecycleMatchStatus(match); status != MatchStatusPending {
		return nil, fmt.Errorf(
			"%w: cannot start match %s with status %q",
			ErrMatchLocked,
			match.ID,
			status,
		)
	}

	if err := lifecycleValidateParticipants(
		tournament,
		match,
	); err != nil {
		return nil, err
	}

	if err := lifecycleValidateIncomingMatches(
		tournament,
		match,
	); err != nil {
		return nil, err
	}

	if err := lifecycleValidateCourt(
		tournament,
		match,
	); err != nil {
		return nil, err
	}

	if err := lifecycleValidateAvailability(
		tournament,
		index,
	); err != nil {
		return nil, err
	}

	lifecycleSetStatus(match, MatchStatusOngoing)
	tournament.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf(
			"start match %s in tournament %s: %w",
			matchID,
			tournamentID,
			err,
		)
	}

	return match, nil
}

// RecordMatchResult completes an ongoing, non-bye match
// and propagates its winner to the next match.
//
// Concurrency requirement:
// Loading, validation and persistence must run inside the same
// repository transaction/locking boundary in production.
func (s *tournamentService) RecordMatchResult(
	ctx context.Context,
	tournamentID string,
	matchID string,
	input MatchResultInput,
) (*Match, error) {
	if err := lifecycleValidateMatchID(matchID); err != nil {
		return nil, err
	}

	tournament, err := s.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf(
			"load tournament before recording match result: %w",
			err,
		)
	}

	if err := lifecycleValidateBracket(tournament); err != nil {
		return nil, err
	}

	index, err := lifecycleFindMatchIndex(
		tournament.Matches,
		matchID,
	)
	if err != nil {
		return nil, err
	}

	match := &tournament.Matches[index]

	if lifecycleIsBye(match) {
		return nil, ErrByeMatch
	}

	if lifecycleHasResult(match) {
		return nil, ErrResultAlreadyExists
	}

	if status := lifecycleMatchStatus(match); status != MatchStatusOngoing {
		return nil, fmt.Errorf(
			"%w: cannot record result for match %s with status %q",
			ErrMatchLocked,
			match.ID,
			status,
		)
	}

	if err := lifecycleValidateParticipants(
		tournament,
		match,
	); err != nil {
		return nil, err
	}

	if err := lifecycleValidateIncomingMatches(
		tournament,
		match,
	); err != nil {
		return nil, err
	}

	result, err := lifecycleBuildMatchResult(match, input, settingsFor(tournament).Scoring)
	if err != nil {
		return nil, err
	}

	if err := lifecycleValidateDownstreamClearable(
		tournament,
		index,
	); err != nil {
		return nil, err
	}

	nextIndex := -1

	if match.NextMatchID != nil {
		nextIndex, err = lifecycleFindMatchIndex(
			tournament.Matches,
			*match.NextMatchID,
		)
		if err != nil {
			return nil, err
		}

		next := &tournament.Matches[nextIndex]

		if lifecycleIsBye(next) {
			return nil, fmt.Errorf(
				"%w: cannot propagate winner into bye match %s",
				ErrInvalidTournamentInput,
				next.ID,
			)
		}

		// Bracket validation guarantees a valid, non-nil NextSlot.
		slot, err := lifecycleSlotPointer(
			next,
			*match.NextSlot,
		)
		if err != nil {
			return nil, err
		}

		// An ongoing source match must not have already
		// propagated an athlete into its destination slot.
		if *slot != nil {
			return nil, fmt.Errorf(
				"%w: destination slot in match %s is already occupied",
				ErrInvalidTournamentInput,
				next.ID,
			)
		}

		if lifecycleMatchHasAthlete(next, *result.WinnerID) {
			return nil, fmt.Errorf(
				"%w: winner %s already occupies the other slot of match %s",
				ErrInvalidTournamentInput,
				*result.WinnerID,
				next.ID,
			)
		}
	}

	// Copy the tournament value and match slice before mutation.
	// Other nested data is shared but is not modified here.
	updated := *tournament
	updated.Matches = make([]Match, len(tournament.Matches))
	copy(updated.Matches, tournament.Matches)

	match = &updated.Matches[index]

	now := time.Now().UTC()
	result.RecordedAt = now

	match.Result = result
	match.WinnerID = lifecycleResultStringCopy(result.WinnerID)
	lifecycleSetStatus(match, MatchStatusCompleted)

	if nextIndex != -1 {
		next := &updated.Matches[nextIndex]

		slot, err := lifecycleSlotPointer(
			next,
			*match.NextSlot,
		)
		if err != nil {
			return nil, err
		}

		*slot = lifecycleResultStringCopy(result.WinnerID)
	}

	updated.UpdatedAt = now

	if err := s.repository.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf(
			"record result of match %s in tournament %s: %w",
			matchID,
			tournamentID,
			err,
		)
	}

	return match, nil
}

// ClearMatchResult removes a completed, non-bye match result.
//
// Policy:
//   - The match returns to pending, not ongoing.
//   - Its athletes and court remain unchanged.
//   - Its winner is removed from the immediate next-match slot.
//   - Started/completed/result-bearing downstream matches block clearing.
//   - Downstream results are never deleted automatically.
//
// Concurrency requirement:
// Loading, validation and persistence must run inside the same
// repository transaction/locking boundary in production.
func (s *tournamentService) ClearMatchResult(
	ctx context.Context,
	tournamentID string,
	matchID string,
) (*Match, error) {
	if err := lifecycleValidateMatchID(matchID); err != nil {
		return nil, err
	}

	tournament, err := s.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf(
			"load tournament before clearing match result: %w",
			err,
		)
	}

	if err := lifecycleValidateBracket(tournament); err != nil {
		return nil, err
	}

	index, err := lifecycleFindMatchIndex(
		tournament.Matches,
		matchID,
	)
	if err != nil {
		return nil, err
	}

	match := &tournament.Matches[index]

	if lifecycleIsBye(match) {
		return nil, ErrByeMatch
	}

	if lifecycleMatchStatus(match) != MatchStatusCompleted {
		return nil, fmt.Errorf(
			"%w: only completed matches can have their result cleared",
			ErrMatchLocked,
		)
	}

	// Do not silently repair a partially corrupted result.
	if err := lifecycleValidateCompletedWinner(match); err != nil {
		return nil, err
	}

	if err := lifecycleValidateDownstreamClearable(
		tournament,
		index,
	); err != nil {
		return nil, err
	}

	// All validation is done before any mutation.
	if match.NextMatchID != nil {
		nextIndex, err := lifecycleFindMatchIndex(
			tournament.Matches,
			*match.NextMatchID,
		)
		if err != nil {
			return nil, err
		}

		next := &tournament.Matches[nextIndex]

		slot, err := lifecycleSlotPointer(
			next,
			*match.NextSlot,
		)
		if err != nil {
			return nil, err
		}

		// A nil slot is acceptable: no propagated winner to remove.
		// A different athlete indicates inconsistent bracket data.
		if *slot != nil && **slot != *match.WinnerID {
			return nil, fmt.Errorf(
				"%w: next match %s contains an unexpected athlete "+
					"in the slot supplied by match %s",
				ErrInvalidTournamentInput,
				next.ID,
				match.ID,
			)
		}

		*slot = nil
	}

	match.Result = nil
	match.WinnerID = nil
	lifecycleSetStatus(match, MatchStatusPending)

	tournament.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf(
			"clear result of match %s in tournament %s: %w",
			matchID,
			tournamentID,
			err,
		)
	}

	return match, nil
}

func lifecycleValidateMatchID(matchID string) error {
	if strings.TrimSpace(matchID) == "" {
		return fmt.Errorf(
			"%w: match id is required",
			ErrInvalidTournamentInput,
		)
	}

	return nil
}

// lifecycleFindMatchIndex also rejects duplicate identifiers.
func lifecycleFindMatchIndex(
	matches []Match,
	matchID string,
) (int, error) {
	found := -1

	for i := range matches {
		if matches[i].ID != matchID {
			continue
		}

		if found != -1 {
			return -1, fmt.Errorf(
				"%w: duplicate match id %q",
				ErrInvalidTournamentInput,
				matchID,
			)
		}

		found = i
	}

	if found == -1 {
		return -1, ErrMatchNotFound
	}

	return found, nil
}

func lifecycleMatchStatus(match *Match) MatchStatus {
	if match.Status == nil {
		return MatchStatusPending
	}

	return *match.Status
}

func lifecycleSetStatus(
	match *Match,
	status MatchStatus,
) {
	match.Status = &status
}

func lifecycleIsBye(match *Match) bool {
	return match.IsBye != nil && *match.IsBye
}

func lifecycleHasResult(match *Match) bool {
	return match.Result != nil || match.WinnerID != nil
}

func lifecycleMatchHasAthlete(
	match *Match,
	athleteID string,
) bool {
	if strings.TrimSpace(athleteID) == "" {
		return false
	}

	return (match.Athlete1ID != nil &&
		*match.Athlete1ID == athleteID) ||
		(match.Athlete2ID != nil &&
			*match.Athlete2ID == athleteID)
}

// lifecycleSlotPointer returns a pointer to the destination slot,
// allowing the caller to inspect or clear it.
func lifecycleSlotPointer(
	match *Match,
	slot NextSlot,
) (**string, error) {
	switch slot {
	case NextSlotAthlete1:
		return &match.Athlete1ID, nil

	case NextSlotAthlete2:
		return &match.Athlete2ID, nil

	default:
		return nil, fmt.Errorf(
			"%w: invalid slot for match %s",
			ErrInvalidTournamentInput,
			match.ID,
		)
	}
}

func lifecycleValidateParticipants(
	tournament *Tournament,
	match *Match,
) error {
	if match.Athlete1ID == nil ||
		match.Athlete2ID == nil ||
		strings.TrimSpace(*match.Athlete1ID) == "" ||
		strings.TrimSpace(*match.Athlete2ID) == "" {
		return fmt.Errorf(
			"%w: match %s requires two athletes",
			ErrInvalidTournamentInput,
			match.ID,
		)
	}

	if *match.Athlete1ID == *match.Athlete2ID {
		return fmt.Errorf(
			"%w: match %s cannot contain the same athlete twice",
			ErrInvalidTournamentInput,
			match.ID,
		)
	}

	for _, athleteID := range []string{
		*match.Athlete1ID,
		*match.Athlete2ID,
	} {
		index := findTournamentAthleteIndex(
			tournament.Athletes,
			athleteID,
		)
		if index == -1 {
			return fmt.Errorf(
				"%w: athlete %s assigned to match %s",
				ErrAthleteNotFound,
				athleteID,
				match.ID,
			)
		}

		athlete := &tournament.Athletes[index]
		if !athletePassedWeighIn(*athlete) {
			return ErrWeighInNotPassed
		}

		if athlete.WeightCategory != match.WeightCategory {
			return fmt.Errorf(
				"%w: athlete %s weight category "+
					"does not match match %s",
				ErrInvalidTournamentInput,
				athleteID,
				match.ID,
			)
		}
	}

	return nil
}

func lifecycleValidateCourt(
	tournament *Tournament,
	match *Match,
) error {
	if tournament.Courts < 1 {
		return fmt.Errorf(
			"%w: tournament must have at least one court",
			ErrInvalidCourt,
		)
	}

	if match.Court < 1 || match.Court > tournament.Courts {
		return fmt.Errorf(
			"%w: court must be between 1 and %d",
			ErrInvalidCourt,
			tournament.Courts,
		)
	}

	return nil
}

// Availability is checked within this tournament only.
// Shared courts or athletes across tournaments require repository-level checks.
func lifecycleValidateAvailability(
	tournament *Tournament,
	matchIndex int,
) error {
	match := &tournament.Matches[matchIndex]

	for i := range tournament.Matches {
		if i == matchIndex {
			continue
		}

		other := &tournament.Matches[i]

		if lifecycleMatchStatus(other) != MatchStatusOngoing {
			continue
		}

		if other.Day == match.Day && other.Court == match.Court {
			return fmt.Errorf(
				"%w: court %d is occupied by match %s",
				ErrMatchLocked,
				match.Court,
				other.ID,
			)
		}

		if lifecycleMatchHasAthlete(other, *match.Athlete1ID) ||
			lifecycleMatchHasAthlete(other, *match.Athlete2ID) {
			return fmt.Errorf(
				"%w: an athlete is already participating in match %s",
				ErrAthleteLocked,
				other.ID,
			)
		}
	}

	return nil
}

// A completed bye may have a WinnerID without a MatchResult.

func lifecycleValidateCompletedWinner(match *Match) error {
	if lifecycleMatchStatus(match) != MatchStatusCompleted {
		return fmt.Errorf("%w: match %s is not completed", ErrMatchLocked, match.ID)
	}
	if match.WinnerID == nil || strings.TrimSpace(*match.WinnerID) == "" {
		return fmt.Errorf("%w: match %s has no winner", ErrInvalidTournamentInput, match.ID)
	}
	if !lifecycleMatchHasAthlete(match, *match.WinnerID) {
		return fmt.Errorf("%w: winner is outside match %s participants", ErrInvalidTournamentInput, match.ID)
	}
	if match.Result == nil {
		if lifecycleIsBye(match) {
			return nil
		}
		return fmt.Errorf("%w: match %s has no result", ErrInvalidTournamentInput, match.ID)
	}
	if match.Result.WinnerID == nil || *match.Result.WinnerID != *match.WinnerID {
		return fmt.Errorf("%w: match %s has inconsistent winner data", ErrInvalidTournamentInput, match.ID)
	}
	return nil
}

// Every incoming edge must supply a completed match's winner
// to the correct destination slot.
func lifecycleValidateIncomingMatches(tournament *Tournament, match *Match) error {
	for i := range tournament.Matches {
		source := &tournament.Matches[i]
		if source.NextMatchID == nil || *source.NextMatchID != match.ID {
			continue
		}
		if source.NextSlot == nil {
			return fmt.Errorf("%w: source match %s has no next slot", ErrInvalidTournamentInput, source.ID)
		}
		slot, err := lifecycleSlotPointer(match, *source.NextSlot)
		if err != nil {
			return nilOrSourceError(source.ID, err)
		}
		if err := lifecycleValidateCompletedWinner(source); err != nil {
			return fmt.Errorf("validate incoming match %s: %w", source.ID, err)
		}
		if *slot == nil || **slot != *source.WinnerID {
			return fmt.Errorf("%w: winner of %s is not in the expected slot of %s", ErrInvalidTournamentInput, source.ID, match.ID)
		}
	}
	return nil
}

func nilOrSourceError(sourceID string, err error) error {
	return fmt.Errorf(
		"validate next slot of source match %s: %w",
		sourceID,
		err,
	)
}

// lifecycleValidateBracket validates structural integrity:
//   - unique, non-empty match IDs;
//   - paired NextMatchID and NextSlot;
//   - existing destinations;
//   - valid and uniquely occupied destination slots;
//   - matching weight categories across edges;
//   - no cycles.
//
// This deliberately rejects corrupted bracket data instead of repairing it.
func lifecycleValidateBracket(tournament *Tournament) error {
	if tournament == nil {
		return fmt.Errorf(
			"%w: tournament is nil",
			ErrInvalidTournamentInput,
		)
	}

	indexByID := make(map[string]int, len(tournament.Matches))

	for i := range tournament.Matches {
		match := &tournament.Matches[i]

		if strings.TrimSpace(match.ID) == "" {
			return fmt.Errorf(
				"%w: match at index %d has an empty id",
				ErrInvalidTournamentInput,
				i,
			)
		}

		if _, exists := indexByID[match.ID]; exists {
			return fmt.Errorf(
				"%w: duplicate match id %q",
				ErrInvalidTournamentInput,
				match.ID,
			)
		}

		indexByID[match.ID] = i
	}

	type destinationSlot struct {
		matchID string
		slot    NextSlot
	}

	occupied := make(map[destinationSlot]string)

	for i := range tournament.Matches {
		source := &tournament.Matches[i]

		if source.NextMatchID == nil {
			if source.NextSlot != nil {
				return fmt.Errorf(
					"%w: match %s has a next slot without a next match",
					ErrInvalidTournamentInput,
					source.ID,
				)
			}

			continue
		}

		if strings.TrimSpace(*source.NextMatchID) == "" {
			return fmt.Errorf(
				"%w: match %s has an empty next match id",
				ErrInvalidTournamentInput,
				source.ID,
			)
		}

		if source.NextSlot == nil {
			return fmt.Errorf(
				"%w: match %s has a next match without a next slot",
				ErrInvalidTournamentInput,
				source.ID,
			)
		}

		targetIndex, exists := indexByID[*source.NextMatchID]
		if !exists {
			return fmt.Errorf(
				"%w: match %s references missing next match %s",
				ErrInvalidTournamentInput,
				source.ID,
				*source.NextMatchID,
			)
		}

		target := &tournament.Matches[targetIndex]

		if source.ID == target.ID {
			return fmt.Errorf(
				"%w: match %s points to itself",
				ErrInvalidTournamentInput,
				source.ID,
			)
		}

		if _, err := lifecycleSlotPointer(
			target,
			*source.NextSlot,
		); err != nil {
			return err
		}

		if source.WeightCategory != target.WeightCategory {
			return fmt.Errorf(
				"%w: matches %s and %s have different weight categories",
				ErrInvalidTournamentInput,
				source.ID,
				target.ID,
			)
		}

		key := destinationSlot{
			matchID: target.ID,
			slot:    *source.NextSlot,
		}

		if previousID, exists := occupied[key]; exists {
			return fmt.Errorf(
				"%w: matches %s and %s feed the same slot of match %s",
				ErrInvalidTournamentInput,
				previousID,
				source.ID,
				target.ID,
			)
		}

		occupied[key] = source.ID
	}

	// Iterative cycle detection:
	// 0 = unvisited, 1 = current path, 2 = finished.
	state := make([]uint8, len(tournament.Matches))

	for start := range tournament.Matches {
		if state[start] != 0 {
			continue
		}

		path := make([]int, 0)
		current := start

		for {
			if state[current] == 1 {
				return fmt.Errorf(
					"%w: bracket contains a cycle at match %s",
					ErrInvalidTournamentInput,
					tournament.Matches[current].ID,
				)
			}

			if state[current] == 2 {
				break
			}

			state[current] = 1
			path = append(path, current)

			nextID := tournament.Matches[current].NextMatchID
			if nextID == nil {
				break
			}

			current = indexByID[*nextID]
		}

		for _, index := range path {
			state[index] = 2
		}
	}

	return nil
}

// lifecycleValidateDownstreamClearable applies a conservative policy:
// every dependent match must still be pending and have no result.
//
// It also rejects stale winner propagation from a pending downstream match.
// It does not delete unrelated participants or recursively clear results.
//
// Precondition: lifecycleValidateBracket has succeeded.
func lifecycleValidateDownstreamClearable(
	tournament *Tournament,
	sourceIndex int,
) error {
	source := &tournament.Matches[sourceIndex]
	nextID := source.NextMatchID

	for nextID != nil {
		index, err := lifecycleFindMatchIndex(
			tournament.Matches,
			*nextID,
		)
		if err != nil {
			return nilOrSourceError(source.ID, err)
		}

		match := &tournament.Matches[index]

		if lifecycleMatchStatus(match) != MatchStatusPending ||
			lifecycleHasResult(match) {
			return fmt.Errorf(
				"%w: downstream match %s must have its result "+
					"cleared and be pending first",
				ErrMatchLocked,
				match.ID,
			)
		}

		// A pending match without a result must not already have
		// supplied an athlete to its own next-match slot.
		if match.NextMatchID != nil {
			targetIndex, err := lifecycleFindMatchIndex(
				tournament.Matches,
				*match.NextMatchID,
			)
			if err != nil {
				return err
			}

			target := &tournament.Matches[targetIndex]

			slot, err := lifecycleSlotPointer(
				target,
				*match.NextSlot,
			)
			if err != nil {
				return err
			}

			if *slot != nil {
				return fmt.Errorf(
					"%w: pending match %s has an occupied "+
						"outgoing slot in match %s",
					ErrInvalidTournamentInput,
					match.ID,
					target.ID,
				)
			}
		}

		nextID = match.NextMatchID
	}

	return nil
}

// lifecycleBuildMatchResult validates and copies caller-owned data.
// It does not modify the tournament or infer winners from scores.
func lifecycleBuildMatchResult(
	match *Match,
	input MatchResultInput,
	scoring ...ScoringSettings,
) (*MatchResult, error) {
	switch input.WinType {
	case WinTypePTF,
		WinTypePTG,
		WinTypeRSC,
		WinTypeSUP,
		WinTypeGDP,
		WinTypeWDR,
		WinTypeDSQ,
		WinTypePUN,
		WinTypeRSCInj,
		WinTypeWO:
		// Supported enum value.
	default:
		return nil, fmt.Errorf(
			"%w: invalid win type %q",
			ErrInvalidTournamentEnum,
			input.WinType,
		)
	}

	if err := lifecycleValidateResultRounds(input.Rounds); err != nil {
		return nil, err
	}
	if err := normalizeScoredResult(&input, scoring...); err != nil {
		return nil, err
	}
	blueID := strings.TrimSpace(input.BlueID)
	redID := strings.TrimSpace(input.RedID)
	winnerID := strings.TrimSpace(input.WinnerID)

	if blueID == "" || redID == "" || winnerID == "" {
		return nil, fmt.Errorf(
			"%w: blue id, red id and winner id are required",
			ErrInvalidTournamentInput,
		)
	}

	if blueID == redID {
		return nil, fmt.Errorf(
			"%w: blue and red athletes must be different",
			ErrInvalidTournamentInput,
		)
	}

	if !lifecycleMatchHasAthlete(match, blueID) ||
		!lifecycleMatchHasAthlete(match, redID) {
		return nil, fmt.Errorf(
			"%w: blue and red athletes must match the participants of match %s",
			ErrInvalidTournamentInput,
			match.ID,
		)
	}

	var winnerCorner Corner

	switch winnerID {
	case blueID:
		winnerCorner = CornerBlue
	case redID:
		winnerCorner = CornerRed
	default:
		return nil, fmt.Errorf(
			"%w: winner must be one of the match participants",
			ErrInvalidTournamentInput,
		)
	}

	if input.WinnerCorner != nil &&
		*input.WinnerCorner != winnerCorner {
		return nil, fmt.Errorf(
			"%w: winner corner does not match winner id",
			ErrInvalidTournamentInput,
		)
	}

	if err := lifecycleValidateResultRounds(input.Rounds); err != nil {
		return nil, err
	}

	refereeID, err := normalizeUUIDPointer(
		"referee id",
		input.RefereeID,
	)
	if err != nil {
		return nil, err
	}

	var note *string
	if input.Note != nil {
		value := strings.TrimSpace(*input.Note)
		if value != "" {
			note = &value
		}
	}

	rounds := make([]MatchRound, len(input.Rounds))
	for i := range input.Rounds {
		rounds[i] = input.Rounds[i]

		if input.Rounds[i].Winner != nil {
			value := *input.Rounds[i].Winner
			rounds[i].Winner = &value
		}

		if input.Rounds[i].EndedBy != nil {
			value := *input.Rounds[i].EndedBy
			rounds[i].EndedBy = &value
		}
	}

	return &MatchResult{
		WinType:      input.WinType,
		BlueID:       &blueID,
		RedID:        &redID,
		WinnerID:     &winnerID,
		WinnerCorner: &winnerCorner,
		Rounds:       rounds,
		Note:         note,
		RefereeID:    lifecycleResultStringCopy(refereeID),
		// RecordedAt is assigned before persistence.
	}, nil
}

// lifecycleValidateResultRounds performs structural validation.
// Empty rounds are allowed; scoring rules are validated separately.
func lifecycleValidateResultRounds(rounds []MatchRound) error {
	for i := range rounds {
		round := &rounds[i]

		if round.Number != i+1 {
			return fmt.Errorf(
				"%w: rounds must be ordered and numbered from 1; expected %d, got %d",
				ErrInvalidTournamentInput,
				i+1,
				round.Number,
			)
		}

		if err := lifecycleValidateResultScore(
			round.Number,
			CornerBlue,
			round.Blue,
		); err != nil {
			return err
		}

		if err := lifecycleValidateResultScore(
			round.Number,
			CornerRed,
			round.Red,
		); err != nil {
			return err
		}

		if round.Winner != nil &&
			*round.Winner != CornerBlue &&
			*round.Winner != CornerRed {
			return fmt.Errorf(
				"%w: invalid winner corner in round %d",
				ErrInvalidTournamentEnum,
				round.Number,
			)
		}

		if round.ManualWinner && round.Winner == nil {
			return fmt.Errorf(
				"%w: round %d has a manual winner flag without a winner",
				ErrInvalidTournamentInput,
				round.Number,
			)
		}

		if round.EndedBy != nil {
			switch *round.EndedBy {
			case RoundEndPTF,
				RoundEndPTG,
				RoundEndPUN,
				RoundEndGDP,
				RoundEndSUP:
				// Supported enum value.
			default:
				return fmt.Errorf(
					"%w: invalid end reason in round %d",
					ErrInvalidTournamentEnum,
					round.Number,
				)
			}

			if *round.EndedBy == RoundEndGDP && !round.IsGoldenPoint {
				return fmt.Errorf(
					"%w: round %d ended by GDP but is not a golden-point round",
					ErrInvalidTournamentInput,
					round.Number,
				)
			}
		}

		if round.IsGoldenPoint && i != len(rounds)-1 {
			return fmt.Errorf(
				"%w: golden-point round must be the last round",
				ErrInvalidTournamentInput,
			)
		}
	}

	return nil
}

func lifecycleValidateResultScore(
	roundNumber int,
	corner Corner,
	score RoundScore,
) error {
	for _, value := range []int{score.Punch, score.BodyKick, score.HeadKick, score.TurningBodyKick, score.TurningHeadKick, score.GamJeom, score.GamJeomLate} {
		if value > 10000 {
			return fmt.Errorf("%w: technique and penalty counts cannot exceed 10000", ErrInvalidTournamentInput)
		}
	}
	if score.Punch < 0 ||
		score.BodyKick < 0 ||
		score.HeadKick < 0 ||
		score.TurningBodyKick < 0 ||
		score.TurningHeadKick < 0 ||
		score.GamJeom < 0 ||
		score.GamJeomLate < 0 {
		return fmt.Errorf(
			"%w: round %d %s scores and penalties cannot be negative",
			ErrInvalidTournamentInput,
			roundNumber,
			corner,
		)
	}

	return nil
}

func lifecycleResultStringCopy(value *string) *string {
	if value == nil {
		return nil
	}

	copied := *value
	return &copied
}
