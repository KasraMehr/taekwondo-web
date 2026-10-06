package tournaments

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SetMatchNumberInput changes only the displayed match number.
// Execution order and bracket identity remain unchanged.
type SetMatchNumberInput struct {
	MatchNumber int `json:"matchNumber"`
}

// SetMatchNumber changes the displayed number of a pending,
// non-bye match.
//
// Numbers are unique within a tournament court.
// Match ID, execution order and bracket links are not modified.
func (s *tournamentService) SetMatchNumber(
	ctx context.Context,
	tournamentID string,
	matchID string,
	input SetMatchNumberInput,
) (*Match, error) {
	if err := lifecycleValidateMatchID(matchID); err != nil {
		return nil, err
	}

	if input.MatchNumber <= 0 {
		return nil, fmt.Errorf(
			"%w: match number must be greater than zero",
			ErrInvalidMatchPosition,
		)
	}

	tournament, err := s.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf(
			"load tournament before setting match number: %w",
			err,
		)
	}

	index, err := lifecycleFindMatchIndex(
		tournament.Matches,
		matchID,
	)
	if err != nil {
		return nil, err
	}

	if hasStartedMatches(tournament) {
		return nil, ErrMatchLocked
	}
	if !allDrawsComplete(tournament) {
		return nil, fmt.Errorf("%w: finish weigh-ins and draws for all categories before numbering", ErrMatchLocked)
	}
	match := &tournament.Matches[index]

	if err := schedulingValidateEditable(match); err != nil {
		return nil, err
	}

	if err := lifecycleValidateCourt(
		tournament,
		match,
	); err != nil {
		return nil, err
	}

	if err := schedulingValidateNumberAvailable(
		tournament,
		index,
		match.Court,
		input.MatchNumber,
	); err != nil {
		return nil, err
	}

	// A repeated request does not require another write.
	if match.MatchNumber != nil &&
		*match.MatchNumber == input.MatchNumber {
		result := *match
		result.MatchNumber = cloneIntPointer(match.MatchNumber)

		return &result, nil
	}

	updated := schedulingCloneMatches(tournament)
	match = &updated.Matches[index]

	number := input.MatchNumber
	match.MatchNumber = &number

	updated.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, updated); err != nil {
		return nil, fmt.Errorf(
			"set number of match %s in tournament %s: %w",
			matchID,
			tournamentID,
			err,
		)
	}

	return match, nil
}

// schedulingValidateEditable limits scheduling changes
// to pending, non-bye matches without any result.
func schedulingValidateEditable(match *Match) error {
	if match == nil {
		return fmt.Errorf(
			"%w: match is nil",
			ErrInvalidTournamentInput,
		)
	}

	if lifecycleIsBye(match) {
		return ErrByeMatch
	}

	if lifecycleHasResult(match) {
		return fmt.Errorf(
			"%w: match %s already has result data",
			ErrMatchLocked,
			match.ID,
		)
	}

	if status := lifecycleMatchStatus(match); status != MatchStatusPending {
		return fmt.Errorf(
			"%w: cannot edit scheduling of match %s with status %q",
			ErrMatchLocked,
			match.ID,
			status,
		)
	}

	return nil
}

// schedulingValidateNumberAvailable checks uniqueness within
// one court of the current tournament.
//
// A nil MatchNumber means no displayed number has been assigned.
// Existing numbers remain reserved regardless of match status.
func schedulingValidateNumberAvailable(
	tournament *Tournament,
	excludedIndex int,
	court int,
	number int,
) error {
	if court < 1 || court > tournament.Courts {
		return fmt.Errorf(
			"%w: court must be between 1 and %d",
			ErrInvalidCourt,
			tournament.Courts,
		)
	}

	if number <= 0 {
		return fmt.Errorf(
			"%w: match number must be greater than zero",
			ErrInvalidMatchPosition,
		)
	}

	for i := range tournament.Matches {
		if i == excludedIndex {
			continue
		}

		other := &tournament.Matches[i]

		if other.MatchNumber == nil || !sameNumberScope(tournament, tournament.Matches[excludedIndex].Day, court, *other) {
			continue
		}

		if *other.MatchNumber == number {
			return fmt.Errorf(
				"%w: match number %d on court %d "+
					"is already assigned to match %s",
				ErrInvalidMatchPosition,
				number,
				court,
				other.ID,
			)
		}
	}

	return nil
}

// schedulingCloneMatches copies the tournament value and its
// match slice.
//
// Nested pointers and other collections are still shared.
// Callers must replace pointer fields rather than mutate
// their referenced values.
func schedulingCloneMatches(
	tournament *Tournament,
) *Tournament {
	updated := *tournament

	updated.Matches = make([]Match, len(tournament.Matches))
	copy(updated.Matches, tournament.Matches)

	return &updated
}

// MoveMatch moves a match to an unoccupied court/order position.
// MatchNumber is preserved and must be available on the target court.
func (s *tournamentService) MoveMatch(
	ctx context.Context,
	tournamentID string,
	matchID string,
	input MoveMatchInput,
) (*Match, error) {
	if err := lifecycleValidateMatchID(matchID); err != nil {
		return nil, err
	}

	tournament, err := s.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("load tournament before moving match: %w", err)
	}

	index, err := lifecycleFindMatchIndex(tournament.Matches, matchID)
	if err != nil {
		return nil, err
	}

	return s.schedulingMoveLoadedMatch(
		ctx,
		tournament,
		index,
		input.Court,
		input.Order,
		input.SourceLabel,
	)
}

// ReassignMatchCourt appends a match after the highest order
// currently assigned on the target court.
func (s *tournamentService) ReassignMatchCourt(
	ctx context.Context,
	tournamentID string,
	matchID string,
	input ReassignMatchCourtInput,
) (*Match, error) {
	if err := lifecycleValidateMatchID(matchID); err != nil {
		return nil, err
	}

	tournament, err := s.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf(
			"load tournament before reassigning match court: %w",
			err,
		)
	}

	index, err := lifecycleFindMatchIndex(tournament.Matches, matchID)
	if err != nil {
		return nil, err
	}

	match := &tournament.Matches[index]

	if err := schedulingValidateEditable(match); err != nil {
		return nil, err
	}

	if err := lifecycleValidateCourt(tournament, match); err != nil {
		return nil, err
	}

	if err := schedulingValidateTargetCourt(tournament, input.Court); err != nil {
		return nil, err
	}

	// Reassigning to the current court preserves the current position.
	if match.Court == input.Court {
		return schedulingCopyMatch(match), nil
	}

	maxOrder := 0

	for i := range tournament.Matches {
		other := &tournament.Matches[i]

		if other.Court == input.Court && other.Order > maxOrder {
			maxOrder = other.Order
		}
	}

	maxInt := int(^uint(0) >> 1)
	if maxOrder == maxInt {
		return nil, fmt.Errorf(
			"%w: no order available after %d on court %d",
			ErrInvalidMatchPosition,
			maxOrder,
			input.Court,
		)
	}

	return s.schedulingMoveLoadedMatch(
		ctx,
		tournament,
		index,
		input.Court,
		maxOrder+1,
		input.SourceLabel,
	)
}

// SwapMatchPositions exchanges court, order and displayed number.
// Athlete assignments and bracket links remain attached to their match.
func (s *tournamentService) SwapMatchPositions(
	ctx context.Context,
	tournamentID string,
	matchID string,
	input SwapMatchPositionsInput,
) (*Tournament, error) {
	if err := lifecycleValidateMatchID(matchID); err != nil {
		return nil, err
	}

	if err := lifecycleValidateMatchID(input.OtherMatchID); err != nil {
		return nil, err
	}

	if matchID == input.OtherMatchID {
		return nil, fmt.Errorf(
			"%w: cannot swap a match with itself",
			ErrInvalidMatchPosition,
		)
	}

	tournament, err := s.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf(
			"load tournament before swapping match positions: %w",
			err,
		)
	}

	firstIndex, err := lifecycleFindMatchIndex(tournament.Matches, matchID)
	if err != nil {
		return nil, err
	}

	secondIndex, err := lifecycleFindMatchIndex(
		tournament.Matches,
		input.OtherMatchID,
	)
	if err != nil {
		return nil, err
	}

	first := &tournament.Matches[firstIndex]
	second := &tournament.Matches[secondIndex]

	for _, match := range []*Match{first, second} {
		if err := schedulingValidateEditable(match); err != nil {
			return nil, err
		}

		if err := lifecycleValidateCourt(tournament, match); err != nil {
			return nil, err
		}

		if match.Order <= 0 {
			return nil, fmt.Errorf(
				"%w: match %s has invalid order %d",
				ErrInvalidMatchPosition,
				match.ID,
				match.Order,
			)
		}
	}

	updated := schedulingCloneMatches(tournament)
	updatedFirst := &updated.Matches[firstIndex]
	updatedSecond := &updated.Matches[secondIndex]

	updatedFirst.Court = second.Court
	updatedFirst.Order = second.Order
	updatedFirst.MatchNumber = cloneIntPointer(second.MatchNumber)

	updatedSecond.Court = first.Court
	updatedSecond.Order = first.Order
	updatedSecond.MatchNumber = cloneIntPointer(first.MatchNumber)

	// Validate the final state, so each swapped match has already
	// vacated its previous position and displayed number.
	for _, index := range []int{firstIndex, secondIndex} {
		match := &updated.Matches[index]

		if err := schedulingValidatePositionAvailable(
			updated,
			index,
			match.Court,
			match.Order,
		); err != nil {
			return nil, err
		}

		if match.MatchNumber != nil {
			if err := schedulingValidateNumberAvailable(
				updated,
				index,
				match.Court,
				*match.MatchNumber,
			); err != nil {
				return nil, err
			}
		}

		if err := schedulingValidateBracketOrder(updated, index); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()

	schedulingRecordMovement(updatedFirst, first, nil, now)
	schedulingRecordMovement(updatedSecond, second, nil, now)
	updated.UpdatedAt = now

	if err := s.repository.Update(ctx, updated); err != nil {
		return nil, fmt.Errorf(
			"swap positions of matches %s and %s in tournament %s: %w",
			matchID,
			input.OtherMatchID,
			tournamentID,
			err,
		)
	}

	return updated, nil
}

// schedulingMoveLoadedMatch operates on a tournament already loaded
// by the caller and persists the resulting change once.
func (s *tournamentService) schedulingMoveLoadedMatch(
	ctx context.Context,
	tournament *Tournament,
	index int,
	court int,
	order int,
	sourceLabel *string,
) (*Match, error) {
	match := &tournament.Matches[index]

	if err := schedulingValidateEditable(match); err != nil {
		return nil, err
	}

	if err := lifecycleValidateCourt(tournament, match); err != nil {
		return nil, err
	}

	if err := schedulingValidatePositionAvailable(
		tournament,
		index,
		court,
		order,
	); err != nil {
		return nil, err
	}

	if match.MatchNumber != nil {
		if err := schedulingValidateNumberAvailable(
			tournament,
			index,
			court,
			*match.MatchNumber,
		); err != nil {
			return nil, err
		}
	}

	if match.Court == court && match.Order == order {
		return schedulingCopyMatch(match), nil
	}

	updated := schedulingCloneMatches(tournament)
	moved := &updated.Matches[index]

	moved.Court = court
	moved.Order = order

	if err := schedulingValidateBracketOrder(updated, index); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	schedulingRecordMovement(moved, match, sourceLabel, now)
	updated.UpdatedAt = now

	if err := s.repository.Update(ctx, updated); err != nil {
		return nil, fmt.Errorf(
			"move match %s in tournament %s: %w",
			match.ID,
			tournament.ID,
			err,
		)
	}

	return schedulingCopyMatch(moved), nil
}

func schedulingValidateTargetCourt(
	tournament *Tournament,
	court int,
) error {
	if court < 1 || court > tournament.Courts {
		return fmt.Errorf(
			"%w: court must be between 1 and %d",
			ErrInvalidCourt,
			tournament.Courts,
		)
	}

	return nil
}

// All existing matches reserve their court/order position,
// regardless of status.
func schedulingValidatePositionAvailable(
	tournament *Tournament,
	excludedIndex int,
	court int,
	order int,
) error {
	if err := schedulingValidateTargetCourt(tournament, court); err != nil {
		return err
	}

	if order <= 0 {
		return fmt.Errorf(
			"%w: order must be greater than zero",
			ErrInvalidMatchPosition,
		)
	}

	for i := range tournament.Matches {
		if i == excludedIndex {
			continue
		}

		other := &tournament.Matches[i]

		if other.Day == tournament.Matches[excludedIndex].Day && other.Court == court && other.Order == order {
			return fmt.Errorf(
				"%w: order %d on court %d is occupied by match %s",
				ErrInvalidMatchPosition,
				order,
				court,
				other.ID,
			)
		}
	}

	return nil
}

// Check direct bracket dependencies when both matches are on
// the same court. Orders on different courts are not comparable.
func schedulingValidateBracketOrder(
	tournament *Tournament,
	index int,
) error {
	match := &tournament.Matches[index]

	for i := range tournament.Matches {
		other := &tournament.Matches[i]

		if i == index || other.Day != match.Day || other.Court != match.Court {
			continue
		}

		// A bye does not require an execution slot before its successor.
		if match.NextMatchID != nil &&
			*match.NextMatchID == other.ID &&
			!lifecycleIsBye(match) &&
			match.Order >= other.Order {
			return fmt.Errorf(
				"%w: match %s must run before next match %s on court %d",
				ErrInvalidMatchPosition,
				match.ID,
				other.ID,
				match.Court,
			)
		}

		if other.NextMatchID != nil &&
			*other.NextMatchID == match.ID &&
			!lifecycleIsBye(other) &&
			other.Order >= match.Order {
			return fmt.Errorf(
				"%w: match %s must run after source match %s on court %d",
				ErrInvalidMatchPosition,
				match.ID,
				other.ID,
				match.Court,
			)
		}
	}

	return nil
}

// Record the position immediately preceding this movement.
// Replace pointers instead of mutating values shared with the original.
func schedulingRecordMovement(
	target *Match,
	previous *Match,
	sourceLabel *string,
	at time.Time,
) {
	court := previous.Court
	order := previous.Order

	target.SourceCourt = &court
	target.SourceOrder = &order
	target.MovedAt = &at

	target.SourceLabel = nil
	if sourceLabel != nil {
		label := *sourceLabel
		target.SourceLabel = &label
	}
}

// Copy scheduling pointers. Other nested data remains shared,
// consistent with schedulingCloneMatches.
func schedulingCopyMatch(match *Match) *Match {
	result := *match

	result.MatchNumber = cloneIntPointer(match.MatchNumber)
	result.SourceCourt = cloneIntPointer(match.SourceCourt)
	result.SourceOrder = cloneIntPointer(match.SourceOrder)

	if match.SourceLabel != nil {
		label := *match.SourceLabel
		result.SourceLabel = &label
	}

	if match.MovedAt != nil {
		at := *match.MovedAt
		result.MovedAt = &at
	}

	return &result
}

// SwapAthletes exchanges two first-round athlete slots while preserving the
// bracket positions. Auto-completed bye winners are allowed and their
// propagated participant in the next round is refreshed after the swap.
func (s *tournamentService) SwapAthletes(
	ctx context.Context,
	tournamentID string,
	matchID string,
	input SwapAthletesInput,
) (*Tournament, error) {
	for _, id := range []string{matchID, input.OtherMatchID} {
		if err := lifecycleValidateMatchID(id); err != nil {
			return nil, err
		}
	}
	if (input.Slot != 1 && input.Slot != 2) || (input.OtherSlot != 1 && input.OtherSlot != 2) {
		return nil, fmt.Errorf("%w: athlete slots must be 1 or 2", ErrInvalidTournamentInput)
	}
	if matchID == input.OtherMatchID && input.Slot == input.OtherSlot {
		return nil, fmt.Errorf("%w: cannot swap an athlete slot with itself", ErrInvalidTournamentInput)
	}
	tournament, err := s.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("load tournament before swapping athletes: %w", err)
	}
	firstIndex, err := lifecycleFindMatchIndex(tournament.Matches, matchID)
	if err != nil {
		return nil, err
	}
	secondIndex, err := lifecycleFindMatchIndex(tournament.Matches, input.OtherMatchID)
	if err != nil {
		return nil, err
	}
	first, second := &tournament.Matches[firstIndex], &tournament.Matches[secondIndex]
	if first.WeightCategory != second.WeightCategory || first.Round != 1 || second.Round != 1 {
		return nil, fmt.Errorf("%w: athlete swaps require first-round matches in the same weight category", ErrInvalidTournamentInput)
	}
	for i := range tournament.Matches {
		match := &tournament.Matches[i]
		if match.WeightCategory == first.WeightCategory && !lifecycleIsBye(match) &&
			(lifecycleHasResult(match) || lifecycleMatchStatus(match) != MatchStatusPending) {
			return nil, fmt.Errorf("%w: category has started or contains results", ErrMatchLocked)
		}
	}
	updated := schedulingCloneMatches(tournament)
	a, b := &updated.Matches[firstIndex], &updated.Matches[secondIndex]
	aSlot := athleteSwapSlot(a, input.Slot)
	bSlot := athleteSwapSlot(b, input.OtherSlot)
	if aSlot == nil || bSlot == nil || *aSlot == nil || *bSlot == nil ||
		strings.TrimSpace(**aSlot) == "" || strings.TrimSpace(**bSlot) == "" {
		return nil, fmt.Errorf("%w: both selected slots must contain an athlete", ErrInvalidTournamentInput)
	}
	*aSlot, *bSlot = *bSlot, *aSlot
	refreshSwappedBye(updated, a)
	if a.ID != b.ID {
		refreshSwappedBye(updated, b)
	}
	updated.UpdatedAt = time.Now().UTC()
	if err := lifecycleValidateBracket(updated); err != nil {
		return nil, fmt.Errorf("validate bracket after athlete swap: %w", err)
	}
	if err := s.repository.Update(ctx, updated); err != nil {
		return nil, fmt.Errorf("swap athletes between matches %s and %s: %w", matchID, input.OtherMatchID, err)
	}
	return updated, nil
}

func athleteSwapSlot(match *Match, slot int) **string {
	if slot == 1 {
		return &match.Athlete1ID
	}
	if slot == 2 {
		return &match.Athlete2ID
	}
	return nil
}

func refreshSwappedBye(tournament *Tournament, match *Match) {
	if !lifecycleIsBye(match) {
		return
	}
	lone := match.Athlete1ID
	if lone == nil {
		lone = match.Athlete2ID
	}
	match.WinnerID = cloneStringPointer(lone)
	lifecycleSetStatus(match, MatchStatusCompleted)
	if match.NextMatchID == nil || lone == nil {
		return
	}
	index, err := lifecycleFindMatchIndex(tournament.Matches, *match.NextMatchID)
	if err != nil {
		return
	}
	next := &tournament.Matches[index]
	if match.NextSlot != nil && *match.NextSlot == NextSlotAthlete2 {
		next.Athlete2ID = cloneStringPointer(lone)
	} else {
		next.Athlete1ID = cloneStringPointer(lone)
	}
}
