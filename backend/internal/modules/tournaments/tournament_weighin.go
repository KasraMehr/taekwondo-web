package tournaments

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	maxWeighInAttempts = 2

	// میزان ارفاق برای دسته‌های کاهشی مانند -52
	weighInToleranceKg = 0.2
)

var (
	ErrWeighInNotPassed = errors.New(
		"weigh-in has not passed",
	)

	ErrWeighInAlreadySigned = errors.New(
		"weigh-in already signed",
	)
)

// WeighInPreview نتیجه بررسی وزن بدون ذخیره در دیتابیس است.
type WeighInPreview struct {
	MinKg             *float64      `json:"minKg"`
	MaxKg             *float64      `json:"maxKg"`
	WeightKg          float64       `json:"weightKg"`
	WeightCategory    string        `json:"weightCategory"`
	WithTolerance     bool          `json:"withTolerance"`
	ToleranceKg       float64       `json:"toleranceKg"`
	CategoryLimitKg   float64       `json:"categoryLimitKg"`
	EffectiveLimitKg  float64       `json:"effectiveLimitKg"`
	LimitIsMinimum    bool          `json:"limitIsMinimum"`
	OK                bool          `json:"ok"`
	AttemptNo         int           `json:"attemptNo"`
	AttemptsUsed      int           `json:"attemptsUsed"`
	AttemptsRemaining int           `json:"attemptsRemaining"`
	CurrentStatus     WeighInStatus `json:"currentStatus"`
	ResultStatus      WeighInStatus `json:"resultStatus"`
}

type weighInEvaluation struct {
	minKg            *float64
	maxKg            *float64
	usedTolerance    bool
	ok               bool
	categoryLimitKg  float64
	effectiveLimitKg float64
	toleranceKg      float64
	limitIsMinimum   bool
}

// ApproveAllWeighIns marks every registered athlete as passed without
// fabricating a scale reading. It is intended for events where the head
// referee has approved the complete roster outside the individual workflow.
func (s *tournamentService) ApproveAllWeighIns(ctx context.Context, tournamentID string) (*Tournament, error) {
	if s == nil || s.repository == nil {
		return nil, fmt.Errorf("%w: tournament repository is nil", ErrInvalidTournamentInput)
	}
	tournament, err := s.repository.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("get tournament %s before bulk weigh-in approval: %w", tournamentID, err)
	}
	if tournament == nil {
		return nil, ErrTournamentNotFound
	}
	normalizeTournamentCollections(tournament)
	if len(tournament.Matches) > 0 {
		return nil, fmt.Errorf("%w: reset brackets before approving all weigh-ins", ErrBracketAlreadyDrawn)
	}
	changed := false
	for i := range tournament.Athletes {
		athlete := &tournament.Athletes[i]
		if err := validateWeighInAthleteCategory(tournament, *athlete); err != nil {
			return nil, err
		}
		if athlete.WeighIn.Status == WeighInPassed {
			continue
		}
		athlete.WeighIn.Status = WeighInPassed
		athlete.WeighedIn = true
		changed = true
	}
	if !changed {
		return tournament, nil
	}
	tournament.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf("update tournament %s after bulk weigh-in approval: %w", tournamentID, err)
	}
	return tournament, nil
}

// PreviewWeighIn وزن را بررسی می‌کند، اما چیزی در دیتابیس ذخیره نمی‌کند.
func (s *tournamentService) PreviewWeighIn(
	ctx context.Context,
	tournamentID string,
	athleteID string,
	input PreviewWeighInInput,
) (*WeighInPreview, error) {
	if s == nil || s.repository == nil {
		return nil, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	if err := validateWeighInWeight(input.WeightKg); err != nil {
		return nil, err
	}

	tournament, index, err :=
		s.loadTournamentAthleteForWeighIn(
			ctx,
			tournamentID,
			athleteID,
		)
	if err != nil {
		return nil, err
	}

	athlete := tournament.Athletes[index]

	if err := validateWeighInAthleteCategory(
		tournament,
		athlete,
	); err != nil {
		return nil, err
	}

	attemptNo, err := nextWeighInAttemptNo(
		athlete.WeighIn, settingsFor(tournament).WeighIn.MaxAttempts,
	)
	if err != nil {
		return nil, err
	}

	attemptsUsed := 0
	currentStatus := WeighInPending

	if athlete.WeighIn != nil {
		attemptsUsed = len(athlete.WeighIn.Attempts)
		currentStatus = effectiveWeighInStatus(
			athlete.WeighIn,
		)
	}

	withTolerance := resolveWeighInTolerance(
		input.WithTolerance,
		athlete.WeighIn,
	)

	evaluation, err := evaluateWeighInWeight(
		athlete.WeightCategory,
		input.WeightKg,
		withTolerance,
		WeightOptions(tournament.AgeCategory, tournament.Gender), settingsFor(tournament).WeighIn.ToleranceKg,
	)
	if err != nil {
		return nil, err
	}

	resultStatus := WeighInPending

	if evaluation.ok {
		resultStatus = WeighInPassed
	} else if attemptsUsed+1 >= settingsFor(tournament).WeighIn.MaxAttempts {
		resultStatus = WeighInFailed
	}

	attemptsRemaining :=
		settingsFor(tournament).WeighIn.MaxAttempts - (attemptsUsed + 1)

	if attemptsRemaining < 0 {
		attemptsRemaining = 0
	}

	return &WeighInPreview{
		MinKg:             evaluation.minKg,
		MaxKg:             evaluation.maxKg,
		WeightKg:          input.WeightKg,
		WeightCategory:    athlete.WeightCategory,
		WithTolerance:     evaluation.usedTolerance,
		ToleranceKg:       evaluation.toleranceKg,
		CategoryLimitKg:   evaluation.categoryLimitKg,
		EffectiveLimitKg:  evaluation.effectiveLimitKg,
		LimitIsMinimum:    evaluation.limitIsMinimum,
		OK:                evaluation.ok,
		AttemptNo:         attemptNo,
		AttemptsUsed:      attemptsUsed,
		AttemptsRemaining: attemptsRemaining,
		CurrentStatus:     currentStatus,
		ResultStatus:      resultStatus,
	}, nil
}

// RecordWeighIn یک تلاش وزن‌کشی را ذخیره می‌کند.
func (s *tournamentService) RecordWeighIn(
	ctx context.Context,
	tournamentID string,
	athleteID string,
	input RecordWeighInInput,
) (*TournamentAthlete, error) {
	if s == nil || s.repository == nil {
		return nil, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	if err := validateWeighInWeight(input.WeightKg); err != nil {
		return nil, err
	}

	tournament, index, err :=
		s.loadTournamentAthleteForWeighIn(
			ctx,
			tournamentID,
			athleteID,
		)
	if err != nil {
		return nil, err
	}

	current := tournament.Athletes[index]

	if err := validateWeighInAthleteCategory(
		tournament,
		current,
	); err != nil {
		return nil, err
	}

	attemptNo, err := nextWeighInAttemptNo(
		current.WeighIn,
	)
	if err != nil {
		return nil, err
	}

	withTolerance := resolveWeighInTolerance(
		input.WithTolerance,
		current.WeighIn,
	)

	evaluation, err := evaluateWeighInWeight(
		current.WeightCategory,
		input.WeightKg,
		withTolerance,
		WeightOptions(tournament.AgeCategory, tournament.Gender), settingsFor(tournament).WeighIn.ToleranceKg,
	)
	if err != nil {
		return nil, err
	}

	if current.WeighIn == nil {
		current.WeighIn = newPendingWeighIn()
	}

	if current.WeighIn.Status == "" {
		current.WeighIn.Status = WeighInPending
	}

	if current.WeighIn.Attempts == nil {
		current.WeighIn.Attempts =
			make([]WeighInAttempt, 0)
	}

	now := time.Now().UTC()

	current.WeighIn.Attempts = append(
		current.WeighIn.Attempts,
		WeighInAttempt{
			No:       attemptNo,
			WeightKg: input.WeightKg,
			OK:       evaluation.ok,
			At:       now,
		},
	)

	current.WeighIn.WeightKg =
		cloneFloatPointer(input.WeightKg)

	current.WeighIn.WithTolerance = evaluation.usedTolerance

	switch {
	case evaluation.ok:
		current.WeighIn.Status = WeighInPassed
		current.WeighedIn = true

	case len(current.WeighIn.Attempts) >= settingsFor(tournament).WeighIn.MaxAttempts:
		current.WeighIn.Status = WeighInFailed
		current.WeighedIn = false

	default:
		current.WeighIn.Status = WeighInPending
		current.WeighedIn = false
	}

	tournament.Athletes[index] = current
	tournament.UpdatedAt = now

	if err := s.repository.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf(
			"update tournament %s after recording weigh-in: %w",
			tournamentID,
			err,
		)
	}

	return &current, nil
}

// SignWeighIn امضای نهایی وزن‌کشی را ثبت می‌کند.
func (s *tournamentService) SignWeighIn(
	ctx context.Context,
	tournamentID string,
	athleteID string,
	input SignWeighInInput,
) (*TournamentAthlete, error) {
	if s == nil || s.repository == nil {
		return nil, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	imagePath := strings.TrimSpace(input.ImagePath)

	if imagePath == "" {
		return nil, fmt.Errorf(
			"%w: image path is required",
			ErrInvalidSignature,
		)
	}

	tournament, index, err :=
		s.loadTournamentAthleteForWeighIn(
			ctx,
			tournamentID,
			athleteID,
		)
	if err != nil {
		return nil, err
	}

	current := tournament.Athletes[index]

	if current.WeighIn == nil ||
		current.WeighIn.Status != WeighInPassed {
		return nil, fmt.Errorf(
			"%w: athlete must pass weigh-in before signing",
			ErrWeighInNotPassed,
		)
	}

	if current.WeighIn.Signature != nil {
		return nil, ErrWeighInAlreadySigned
	}

	now := time.Now().UTC()

	current.WeighIn.Signature = &WeighInSignature{
		ImagePath: imagePath,
		SignedAt:  now,
	}

	current.WeighedIn = true

	tournament.Athletes[index] = current
	tournament.UpdatedAt = now

	if err := s.repository.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf(
			"update tournament %s after signing weigh-in: %w",
			tournamentID,
			err,
		)
	}

	return &current, nil
}

func (s *tournamentService) loadTournamentAthleteForWeighIn(
	ctx context.Context,
	tournamentID string,
	athleteID string,
) (*Tournament, int, error) {
	if s == nil || s.repository == nil {
		return nil, -1, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	if err := validateRequiredUUID(
		"tournament id",
		tournamentID,
	); err != nil {
		return nil, -1, err
	}

	if err := validateRequiredUUID(
		"athlete id",
		athleteID,
	); err != nil {
		return nil, -1, err
	}

	tournament, err := s.repository.GetByID(
		ctx,
		tournamentID,
	)
	if err != nil {
		return nil, -1, fmt.Errorf(
			"get tournament %s before weigh-in: %w",
			tournamentID,
			err,
		)
	}

	if tournament == nil {
		return nil, -1, ErrTournamentNotFound
	}

	normalizeTournamentCollections(tournament)

	index := findTournamentAthleteIndex(
		tournament.Athletes,
		athleteID,
	)
	if index == -1 {
		return nil, -1, ErrAthleteNotFound
	}

	return tournament, index, nil
}

func validateWeighInAthleteCategory(
	tournament *Tournament,
	athlete TournamentAthlete,
) error {
	if tournament == nil {
		return fmt.Errorf(
			"%w: tournament is nil",
			ErrInvalidTournamentInput,
		)
	}

	if !IsValidWeightCategory(
		tournament.AgeCategory,
		tournament.Gender,
		athlete.WeightCategory,
	) {
		return fmt.Errorf(
			"%w: invalid athlete weight category %q",
			ErrInvalidTournamentInput,
			athlete.WeightCategory,
		)
	}

	return nil
}

func validateWeighInWeight(
	weightKg float64,
) error {
	if math.IsNaN(weightKg) ||
		math.IsInf(weightKg, 0) ||
		weightKg <= 0 {
		return fmt.Errorf(
			"%w: weight must be a positive finite number",
			ErrInvalidWeight,
		)
	}

	return nil
}

func nextWeighInAttemptNo(
	weighIn *WeighIn,
	limits ...int,
) (int, error) {
	limit := maxWeighInAttempts
	if len(limits) > 0 {
		limit = limits[0]
	}
	status := effectiveWeighInStatus(weighIn)
	attempts := 0

	if weighIn != nil {
		attempts = len(weighIn.Attempts)
	}

	switch status {
	case WeighInPending:
		if attempts >= limit {
			return 0, ErrMaxWeighInAttempts
		}

		return attempts + 1, nil

	case WeighInPassed:
		return 0, ErrWeighInAlreadyPassed

	case WeighInFailed:
		return 0, ErrMaxWeighInAttempts

	default:
		return 0, fmt.Errorf(
			"%w: invalid weigh-in status %q",
			ErrInvalidTournamentEnum,
			status,
		)
	}
}

func effectiveWeighInStatus(
	weighIn *WeighIn,
) WeighInStatus {
	if weighIn == nil || weighIn.Status == "" {
		return WeighInPending
	}

	return weighIn.Status
}

func resolveWeighInTolerance(
	input *bool,
	weighIn *WeighIn,
) bool {
	if input != nil {
		return *input
	}

	// WithTolerance on the saved record means tolerance was used, not allowed.
	// The desktop applies the 200g allowance by default on every attempt.
	return true
}

func evaluateWeighInWeight(
	weightCategory string,
	weightKg float64,
	withTolerance bool,
	categories []string,
	allowances ...float64,
) (weighInEvaluation, error) {
	if err := validateWeighInWeight(weightKg); err != nil {
		return weighInEvaluation{}, err
	}
	weightCategory = strings.TrimSpace(weightCategory)

	if len(weightCategory) < 2 {
		return weighInEvaluation{}, fmt.Errorf(
			"%w: invalid weight category %q",
			ErrInvalidTournamentInput,
			weightCategory,
		)
	}

	operator := weightCategory[0]

	limit, err := strconv.ParseFloat(
		strings.TrimSpace(weightCategory[1:]),
		64,
	)
	if err != nil ||
		math.IsNaN(limit) ||
		math.IsInf(limit, 0) ||
		limit <= 0 {
		return weighInEvaluation{}, fmt.Errorf(
			"%w: invalid weight category %q",
			ErrInvalidTournamentInput,
			weightCategory,
		)
	}

	switch operator {
	case '-':
		tolerance := 0.0

		if withTolerance {
			tolerance = weighInToleranceKg
			if len(allowances) > 0 {
				tolerance = allowances[0]
			}
		}

		effectiveLimit := limit + tolerance

		var minimum *float64
		found := false
		for i, category := range categories {
			if category != weightCategory {
				continue
			}
			found = true
			if i > 0 {
				v, e := strconv.ParseFloat(categories[i-1][1:], 64)
				if e != nil {
					return weighInEvaluation{}, ErrInvalidWeight
				}
				minimum = &v
			}
			break
		}
		if !found {
			return weighInEvaluation{}, ErrInvalidWeight
		}
		grams := math.Round(weightKg * 1000)
		ok := grams <= math.Round(effectiveLimit*1000) && (minimum == nil || grams > math.Round(*minimum*1000))
		return weighInEvaluation{
			ok:               ok,
			minKg:            minimum,
			maxKg:            &limit,
			usedTolerance:    ok && grams > math.Round(limit*1000),
			categoryLimitKg:  limit,
			effectiveLimitKg: effectiveLimit,
			toleranceKg:      tolerance,
			limitIsMinimum:   false,
		}, nil

	case '+':
		// The lower boundary belongs to the preceding category, as in the app.
		// tolerance روی دسته‌های افزایشی اعمال نمی‌شود.
		return weighInEvaluation{
			ok:               math.Round(weightKg*1000) > math.Round(limit*1000),
			minKg:            &limit,
			categoryLimitKg:  limit,
			effectiveLimitKg: limit,
			toleranceKg:      0,
			limitIsMinimum:   true,
		}, nil

	default:
		return weighInEvaluation{}, fmt.Errorf(
			"%w: invalid weight category %q",
			ErrInvalidTournamentInput,
			weightCategory,
		)
	}
}

func cloneFloatPointer(value float64) *float64 {
	result := value
	return &result
}

func (s *tournamentService) ResetCategoryBracketAndWeighIns(
	ctx context.Context,
	tournamentID string,
	weightCategory string,
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

	tournament, err := s.repository.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf(
			"get tournament %s before resetting category: %w",
			tournamentID,
			err,
		)
	}

	if tournament == nil {
		return nil, ErrTournamentNotFound
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

	// Identify matches that will be removed.
	removedMatchIDs := make(map[string]struct{})

	for _, match := range tournament.Matches {
		if match.WeightCategory == weightCategory {
			removedMatchIDs[match.ID] = struct{}{}
		}
	}

	// Reject cross-category bracket links in either direction.
	if len(removedMatchIDs) == 0 {
		return nil, fmt.Errorf("%w: category has no bracket", ErrMatchNotFound)
	}
	for _, match := range tournament.Matches {
		if match.NextMatchID == nil {
			continue
		}

		sourceRemoved := match.WeightCategory == weightCategory
		_, targetRemoved := removedMatchIDs[*match.NextMatchID]

		if sourceRemoved != targetRemoved {
			return nil, fmt.Errorf(
				"%w: cannot reset category %q: match %s links outside the category or to a missing match",
				ErrInvalidTournamentInput,
				weightCategory,
				match.ID,
			)
		}
	}

	if hasStartedMatches(tournament) {
		return nil, ErrMatchLocked
	}
	updated := *tournament
	changed := false

	// Use a new slice so filtering does not modify the original backing array.
	if len(removedMatchIDs) > 0 {
		updated.Matches = make([]Match, 0, len(tournament.Matches))

		for _, match := range tournament.Matches {
			if match.WeightCategory == weightCategory {
				continue
			}

			updated.Matches = append(updated.Matches, match)
		}

		changed = true
	}

	// Copy the map before deleting the category assignment.
	if _, exists := tournament.CourtAssignment[weightCategory]; exists {
		updated.CourtAssignment = make(
			map[string][]int,
			len(tournament.CourtAssignment),
		)

		for category, courts := range tournament.CourtAssignment {
			if category == weightCategory {
				continue
			}

			// Court slices are preserved; this method never mutates them.
			updated.CourtAssignment[category] = courts
		}

		changed = true
	}

	// Allocate an athlete slice only when a weigh-in needs resetting.
	athletesCopied := false

	for i, athlete := range tournament.Athletes {
		if athlete.WeightCategory != weightCategory {
			continue
		}

		if !athlete.WeighedIn && athlete.WeighIn == nil {
			continue
		}

		if !athletesCopied {
			updated.Athletes = make(
				[]TournamentAthlete,
				len(tournament.Athletes),
			)
			copy(updated.Athletes, tournament.Athletes)
			athletesCopied = true
		}

		updated.Athletes[i].WeighedIn = false
		updated.Athletes[i].WeighIn = newPendingWeighIn()
		changed = true
	}

	if !changed {
		return tournament, nil
	}

	clearMatchNumbers(&updated)
	syncMatchOrders(updated.Matches)
	updated.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf(
			"reset bracket and weigh-ins for tournament %s weight category %q: %w",
			tournamentID,
			weightCategory,
			err,
		)
	}

	return &updated, nil
}
