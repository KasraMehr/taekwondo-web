package tournaments

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *tournamentService) DrawBracket(
	ctx context.Context,
	tournamentID string,
	drawType string,
) (*Tournament, error) {
	if s == nil || s.repository == nil {
		return nil, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	if err := validateRequiredUUID(
		"tournament id",
		tournamentID,
	); err != nil {
		return nil, err
	}

	normalizedDrawType, err := normalizeBracketDrawType(drawType)
	if err != nil {
		return nil, err
	}

	tournament, err := s.repository.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf(
			"get tournament %s before drawing bracket: %w",
			tournamentID,
			err,
		)
	}

	if tournament == nil {
		return nil, ErrTournamentNotFound
	}

	normalizeTournamentCollections(tournament)
	if strings.TrimSpace(drawType)=="" { normalizedDrawType,err=normalizeBracketDrawType(settingsFor(tournament).Draw.Type);if err!=nil{return nil,err} }

	if tournament.Courts <= 0 {
		return nil, fmt.Errorf(
			"%w: tournament must have at least one court",
			ErrInvalidCourt,
		)
	}

	if len(tournament.Matches) > 0 {
		return nil, ErrBracketAlreadyDrawn
	}

	for _, a := range tournament.Athletes {
		if !athletePassedWeighIn(a) && (a.WeighIn == nil || a.WeighIn.Status != WeighInFailed) {
			return nil, fmt.Errorf("%w: complete weigh-ins before drawing", ErrWeighInNotPassed)
		}
	}
	weightCategories := tournamentEligibleWeightCategories(tournament)
	if len(weightCategories) == 0 {
		return nil, ErrNoEligibleAthletes
	}

	matches := make([]Match, 0)
	courtAssignment := make(map[string][]int)
	nextOrderByCourt := make(map[int]int)

	for _, weightCategory := range weightCategories {
		categoryAthletes := eligibleAthletesForBracket(
			tournament,
			weightCategory,
		)

		if len(categoryAthletes) < 2 {
			continue
		}

		categoryMatches, categoryCourts, err :=
			buildCategoryBracket(
				categoryAthletes,
				weightCategory,
				normalizedDrawType,
				tournament.Courts,
				nextOrderByCourt, settingsFor(tournament).Draw.SeparateTeams,
			)
		if err != nil {
			return nil, fmt.Errorf(
				"draw bracket for weight category %s: %w",
				weightCategory,
				err,
			)
		}

		matches = append(matches, categoryMatches...)
		courtAssignment[weightCategory] = categoryCourts
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf(
			"%w: at least two eligible athletes are required in one weight category",
			ErrNoEligibleAthletes,
		)
	}

	tournament.Matches = matches
	tournament.CourtAssignment = courtAssignment
	applyCourtPlan(tournament)
	tournament.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf(
			"update tournament %s after drawing bracket: %w",
			tournamentID,
			err,
		)
	}

	return tournament, nil
}

func (s *tournamentService) DrawBracketForCategory(
	ctx context.Context,
	tournamentID string,
	weightCategory string,
	drawType string,
) (*Tournament, error) {
	if s == nil || s.repository == nil {
		return nil, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	if err := validateRequiredUUID(
		"tournament id",
		tournamentID,
	); err != nil {
		return nil, err
	}

	weightCategory = strings.TrimSpace(weightCategory)
	if weightCategory == "" {
		return nil, fmt.Errorf(
			"%w: weight category is required",
			ErrInvalidTournamentInput,
		)
	}

	normalizedDrawType, err := normalizeBracketDrawType(drawType)
	if err != nil {
		return nil, err
	}

	tournament, err := s.repository.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf(
			"get tournament %s before drawing category bracket: %w",
			tournamentID,
			err,
		)
	}

	if tournament == nil {
		return nil, ErrTournamentNotFound
	}

	normalizeTournamentCollections(tournament)
	if strings.TrimSpace(drawType)=="" { normalizedDrawType,err=normalizeBracketDrawType(settingsFor(tournament).Draw.Type);if err!=nil{return nil,err} }

	if tournament.Courts <= 0 {
		return nil, fmt.Errorf(
			"%w: tournament must have at least one court",
			ErrInvalidCourt,
		)
	}

	if !IsValidWeightCategory(
		tournament.AgeCategory,
		tournament.Gender,
		weightCategory,
	) {
		return nil, fmt.Errorf(
			"%w: invalid weight category %q",
			ErrInvalidTournamentInput,
			weightCategory,
		)
	}

	if bracketExistsForCategory(
		tournament.Matches,
		weightCategory,
	) {
		return nil, fmt.Errorf(
			"%w: bracket already exists for weight category %s",
			ErrBracketAlreadyDrawn,
			weightCategory,
		)
	}

	if hasStartedMatches(tournament) {
		return nil, ErrMatchLocked
	}
	for _, a := range tournament.Athletes {
		if a.WeightCategory == weightCategory && !athletePassedWeighIn(a) && (a.WeighIn == nil || a.WeighIn.Status != WeighInFailed) {
			return nil, fmt.Errorf("%w: complete category weigh-ins before drawing", ErrWeighInNotPassed)
		}
	}
	athletes := eligibleAthletesForBracket(
		tournament,
		weightCategory,
	)
	if len(athletes) < 2 {
		return nil, fmt.Errorf(
			"%w: weight category %s requires at least two eligible athletes",
			ErrNoEligibleAthletes,
			weightCategory,
		)
	}

	nextOrderByCourt := calculateNextOrderByCourt(
		tournament.Matches,
		tournament.Courts,
	)

	categoryMatches, categoryCourts, err := buildCategoryBracket(
		athletes,
		weightCategory,
		normalizedDrawType,
		tournament.Courts,
		nextOrderByCourt, settingsFor(tournament).Draw.SeparateTeams,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"draw bracket for weight category %s: %w",
			weightCategory,
			err,
		)
	}

	clearMatchNumbers(tournament)
	tournament.Matches = append(
		tournament.Matches,
		categoryMatches...,
	)

	if tournament.CourtAssignment == nil {
		tournament.CourtAssignment = make(map[string][]int)
	}

	tournament.CourtAssignment[weightCategory] = categoryCourts
	applyCourtPlan(tournament)
	tournament.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf(
			"update tournament %s after drawing category bracket: %w",
			tournamentID,
			err,
		)
	}

	return tournament, nil
}

func buildCategoryBracket(
	athletes []TournamentAthlete,
	weightCategory string,
	drawType string,
	courtCount int,
	nextOrderByCourt map[int]int,
	separateTeams ...bool,
) ([]Match, []int, error) {
	if len(athletes) < 2 {
		return nil, nil, ErrNoEligibleAthletes
	}

	if courtCount <= 0 {
		return nil, nil, ErrInvalidCourt
	}

	orderedAthletes := prepareAthletesForDraw(
		athletes,
		drawType,
	)

	bracketSize := nextPowerOfTwo(len(orderedAthletes))
	totalRounds := bracketRoundCount(bracketSize)

	seedOrder := bracketSeedOrder(bracketSize)
	slots := make([]*string, bracketSize)

	for slotIndex, seed := range seedOrder {
		if seed > len(orderedAthletes) {
			continue
		}

		slots[slotIndex] = stringPointer(orderedAthletes[seed-1].ID)
	}

	if len(separateTeams) == 0 || separateTeams[0] {
		optimizeTeamSeparation(slots, orderedAthletes, drawType, totalRounds)
	}

	roundMatches := make([][]Match, totalRounds)

	firstRoundMatchCount := bracketSize / 2
	roundMatches[0] = make([]Match, firstRoundMatchCount)

	for matchIndex := 0; matchIndex < firstRoundMatchCount; matchIndex++ {
		athlete1ID := cloneStringPointer(
			slots[matchIndex*2],
		)
		athlete2ID := cloneStringPointer(
			slots[(matchIndex*2)+1],
		)

		isBye := exactlyOneAthleteExists(
			athlete1ID,
			athlete2ID,
		)

		status := MatchStatusPending
		var winnerID *string

		if isBye {
			status = MatchStatusCompleted

			if athlete1ID != nil {
				winnerID = cloneStringPointer(athlete1ID)
			} else {
				winnerID = cloneStringPointer(athlete2ID)
			}
		}

		court := assignBracketCourt(
			matchIndex,
			courtCount,
		)
		order := reserveCourtOrder(
			nextOrderByCourt,
			court,
		)

		currentBracketIndex := matchIndex
		currentIsBye := isBye
		if isBye {
			court = 0
			order = 0
		}

		roundMatches[0][matchIndex] = Match{
			ID:             uuid.NewString(),
			Athlete1ID:     athlete1ID,
			Athlete2ID:     athlete2ID,
			WinnerID:       winnerID,
			Court:          court,
			Order:          order,
			Status:         &status,
			Result:         nil,
			BracketIndex:   &currentBracketIndex,
			WeightCategory: weightCategory,
			Round:          1,
			Side: bracketMatchSide(
				1,
				totalRounds,
				matchIndex,
				firstRoundMatchCount,
			),
			IsBye: &currentIsBye,
		}

	}

	previousRoundMatchCount := firstRoundMatchCount

	for roundIndex := 1; roundIndex < totalRounds; roundIndex++ {
		matchCount := previousRoundMatchCount / 2
		roundMatches[roundIndex] = make([]Match, matchCount)

		for matchIndex := 0; matchIndex < matchCount; matchIndex++ {
			court := assignBracketCourt(
				matchIndex+roundIndex,
				courtCount,
			)
			order := reserveCourtOrder(
				nextOrderByCourt,
				court,
			)

			status := MatchStatusPending
			isBye := false

			currentBracketIndex := matchIndex

			roundMatches[roundIndex][matchIndex] = Match{
				ID:             uuid.NewString(),
				Court:          court,
				Order:          order,
				Status:         &status,
				BracketIndex:   &currentBracketIndex,
				WeightCategory: weightCategory,
				Round:          roundIndex + 1,
				Side: bracketMatchSide(
					roundIndex+1,
					totalRounds,
					matchIndex,
					matchCount,
				),
				IsBye: &isBye,
			}

		}

		previousRoundMatchCount = matchCount
	}

	for roundIndex := 0; roundIndex < totalRounds-1; roundIndex++ {
		for matchIndex := range roundMatches[roundIndex] {
			nextMatchIndex := matchIndex / 2
			nextMatchID :=
				roundMatches[roundIndex+1][nextMatchIndex].ID

			nextSlot := NextSlotAthlete1
			if matchIndex%2 == 1 {
				nextSlot = NextSlotAthlete2
			}

			roundMatches[roundIndex][matchIndex].NextMatchID =
				stringPointer(nextMatchID)
			roundMatches[roundIndex][matchIndex].NextSlot =
				nextSlotPointer(nextSlot)
		}
	}

	propagateInitialByes(roundMatches)

	result := make([]Match, 0, bracketSize-1)

	for roundIndex := range roundMatches {
		result = append(result, roundMatches[roundIndex]...)
	}

	usedCourts := collectUsedCourts(result)

	return result, usedCourts, nil
}

func prepareAthletesForDraw(athletes []TournamentAthlete, drawType string) []TournamentAthlete {
	result := append([]TournamentAthlete(nil), athletes...)
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	random.Shuffle(len(result), func(i, j int) { result[i], result[j] = result[j], result[i] })
	if drawType == "ranking" {
		sort.SliceStable(result, func(i, j int) bool {
			left, right := result[i], result[j]
			l, r := left.Ranking != nil && *left.Ranking > 0, right.Ranking != nil && *right.Ranking > 0
			if !l {
				return false
			}
			if !r {
				return true
			}
			if *left.Ranking == *right.Ranking {
				return left.ID < right.ID
			}
			return *left.Ranking < *right.Ranking
		})
	}
	return result
}

func bracketSeedOrder(size int) []int {
	if size <= 1 {
		return []int{1}
	}
	half := bracketSeedOrder(size / 2)
	out := make([]int, 0, size)
	for _, r := range half {
		if r%2 == 1 {
			out = append(out, 2*r-1)
		} else {
			out = append(out, 2*r)
		}
	}
	for i := len(half) - 1; i >= 0; i-- {
		r := half[i]
		if r%2 == 1 {
			out = append(out, 2*r)
		} else {
			out = append(out, 2*r-1)
		}
	}
	return out
}

func nextPowerOfTwo(value int) int {
	if value <= 1 {
		return 1
	}

	result := 1

	for result < value {
		result *= 2
	}

	return result
}

func bracketRoundCount(bracketSize int) int {
	rounds := 0

	for bracketSize > 1 {
		bracketSize /= 2
		rounds++
	}

	return rounds
}

func bracketMatchSide(
	round int,
	totalRounds int,
	matchIndex int,
	matchCount int,
) MatchSide {
	if round == totalRounds {
		return MatchSideFinal
	}

	if matchIndex < matchCount/2 {
		return MatchSideLeft
	}

	return MatchSideRight
}

func propagateInitialByes(roundMatches [][]Match) {
	if len(roundMatches) < 2 {
		return
	}

	for index := range roundMatches[0] {
		current := &roundMatches[0][index]
		if current.IsBye == nil || !*current.IsBye ||
			current.WinnerID == nil {
			continue
		}

		next := &roundMatches[1][index/2]
		if index%2 == 0 {
			next.Athlete1ID = cloneStringPointer(current.WinnerID)
		} else {
			next.Athlete2ID = cloneStringPointer(current.WinnerID)
		}
	}
}

func exactlyOneAthleteExists(
	athlete1ID *string,
	athlete2ID *string,
) bool {
	return (athlete1ID != nil && athlete2ID == nil) ||
		(athlete1ID == nil && athlete2ID != nil)
}

func assignBracketCourt(
	matchIndex int,
	courtCount int,
) int {
	return (matchIndex % courtCount) + 1
}

func reserveCourtOrder(
	nextOrderByCourt map[int]int,
	court int,
) int {
	order := nextOrderByCourt[court]

	if order <= 0 {
		order = 1
	}

	nextOrderByCourt[court] = order + 1

	return order
}

func calculateNextOrderByCourt(
	matches []Match,
	courtCount int,
) map[int]int {
	result := make(map[int]int, courtCount)

	for court := 1; court <= courtCount; court++ {
		result[court] = 1
	}

	for _, match := range matches {
		nextOrder := match.Order + 1

		if nextOrder > result[match.Court] {
			result[match.Court] = nextOrder
		}
	}

	return result
}

func collectUsedCourts(matches []Match) []int {
	seen := make(map[int]struct{})

	for _, match := range matches {
		if match.Court <= 0 {
			continue
		}

		seen[match.Court] = struct{}{}
	}

	result := make([]int, 0, len(seen))

	for court := range seen {
		result = append(result, court)
	}

	sort.Ints(result)

	return result
}

func normalizeBracketDrawType(
	drawType string,
) (string, error) {
	drawType = strings.ToLower(
		strings.TrimSpace(drawType),
	)

	switch drawType {
	case "", "random":
		return "random", nil

	case "ranking", "ranked", "seeded":
		return "ranking", nil

	default:
		return "", fmt.Errorf(
			"%w: unsupported draw type %q",
			ErrInvalidTournamentInput,
			drawType,
		)
	}
}

func eligibleAthletesForBracket(
	tournament *Tournament,
	weightCategory string,
) []TournamentAthlete {
	result := make([]TournamentAthlete, 0)

	for _, athlete := range tournament.Athletes {
		if athlete.WeightCategory != weightCategory {
			continue
		}

		/*
			فعلاً فقط ورزشکارانی وارد قرعه‌کشی می‌شوند
			که وزن‌کشی را پاس کرده‌اند.
		*/
		if !athletePassedWeighIn(athlete) {
			continue
		}

		result = append(result, athlete)
	}

	return result
}

func tournamentEligibleWeightCategories(
	tournament *Tournament,
) []string {
	counts := make(map[string]int)

	for _, athlete := range tournament.Athletes {
		if !athlete.WeighedIn {
			continue
		}

		counts[athlete.WeightCategory]++
	}

	availableOptions := WeightOptions(
		tournament.AgeCategory,
		tournament.Gender,
	)

	result := make([]string, 0)

	for _, weightCategory := range availableOptions {
		if counts[weightCategory] >= 2 {
			result = append(result, weightCategory)
		}
	}

	return result
}

func bracketExistsForCategory(
	matches []Match,
	weightCategory string,
) bool {
	for _, match := range matches {
		if match.WeightCategory == weightCategory {
			return true
		}
	}

	return false
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	result := *value
	return &result
}

func stringPointer(value string) *string {
	result := value
	return &result
}

func nextSlotPointer(value NextSlot) *NextSlot {
	result := value
	return &result
}

func (s *tournamentService) ResetRankings(
	ctx context.Context,
	tournamentID string,
	weightCategory string,
) (*Tournament, error) {
	tournament, err := s.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("load tournament before resetting rankings: %w", err)
	}

	if !IsValidWeightCategory(
		tournament.AgeCategory,
		tournament.Gender,
		weightCategory,
	) {
		return nil, fmt.Errorf(
			"%w: invalid weight category %q",
			ErrInvalidTournamentInput,
			weightCategory,
		)
	}

	updated := *tournament
	updated.Athletes = make([]TournamentAthlete, len(tournament.Athletes))
	copy(updated.Athletes, tournament.Athletes)
	changed := false
	for i := range updated.Athletes {
		athlete := &updated.Athletes[i]
		if athlete.WeightCategory != weightCategory || athlete.Ranking == nil {
			continue
		}

		athlete.Ranking = nil
		changed = true
	}

	if !changed {
		return tournament, nil
	}

	updated.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf(
			"reset rankings for weight category %q: %w",
			weightCategory,
			err,
		)
	}

	return &updated, nil
}
