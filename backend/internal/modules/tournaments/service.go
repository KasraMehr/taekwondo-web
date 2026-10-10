package tournaments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidTournamentInput = errors.New(
		"invalid tournament input",
	)

	ErrInvalidTournamentContext = errors.New(
		"invalid tournament context",
	)

	ErrInvalidTournamentDate = errors.New(
		"invalid tournament date",
	)

	ErrInvalidTournamentEnum = errors.New(
		"invalid tournament enum value",
	)

	ErrAthleteNotFound = errors.New(
		"athlete not found",
	)

	ErrDuplicateAthlete = errors.New(
		"duplicate athlete",
	)

	ErrAthleteLocked = errors.New(
		"athlete is locked",
	)

	ErrMatchNotFound = errors.New(
		"match not found",
	)

	ErrMatchLocked = errors.New(
		"match is locked",
	)

	ErrInvalidWeight = errors.New(
		"invalid weight",
	)

	ErrMaxWeighInAttempts = errors.New(
		"maximum weigh-in attempts reached",
	)

	ErrWeighInAlreadyPassed = errors.New(
		"weigh-in already passed",
	)

	ErrInvalidSignature = errors.New(
		"invalid weigh-in signature",
	)

	ErrBracketAlreadyDrawn = errors.New(
		"bracket already drawn",
	)

	ErrNoEligibleAthletes = errors.New(
		"no eligible athletes",
	)

	ErrResultAlreadyExists = errors.New(
		"match result already exists",
	)

	ErrDownstreamResult = errors.New(
		"downstream match has a result",
	)

	ErrByeMatch = errors.New(
		"bye match cannot be operated",
	)

	ErrInvalidCourt = errors.New(
		"invalid court",
	)

	ErrInvalidMatchPosition = errors.New(
		"invalid match position",
	)
)

type TournamentService interface {
	NumberMatches(context.Context, string, bool) (*Tournament, error)
	UpdateSettings(context.Context, string, TournamentSettings) (*Tournament, error)
	RebalanceCourts(context.Context, string) (*Tournament, error)
	ResetWeighIn(context.Context, string, string) (*TournamentAthlete, error)
	ClearWeighInSignature(context.Context, string, string) (*TournamentAthlete, error)
	ApproveAllWeighIns(context.Context, string) (*Tournament, error)
	Create(
		ctx context.Context,
		input CreateTournamentInput,
	) (*Tournament, error)

	GetByID(
		ctx context.Context,
		id string,
	) (*Tournament, error)

	List(
		ctx context.Context,
		filter TournamentListFilter,
	) ([]Tournament, error)

	Update(
		ctx context.Context,
		id string,
		input UpdateTournamentInput,
	) (*Tournament, error)

	Delete(
		ctx context.Context,
		id string,
	) error

	AddAthlete(
		ctx context.Context,
		tournamentID string,
		input AddAthleteInput,
	) (*TournamentAthlete, error)

	UpdateAthlete(
		ctx context.Context,
		tournamentID string,
		athleteID string,
		input UpdateAthleteInput,
	) (*TournamentAthlete, error)

	RemoveAthlete(
		ctx context.Context,
		tournamentID string,
		athleteID string,
	) error

	PreviewWeighIn(
		ctx context.Context,
		tournamentID string,
		athleteID string,
		input PreviewWeighInInput,
	) (*WeighInPreview, error)

	RecordWeighIn(
		ctx context.Context,
		tournamentID string,
		athleteID string,
		input RecordWeighInInput,
	) (*TournamentAthlete, error)

	SignWeighIn(
		ctx context.Context,
		tournamentID string,
		athleteID string,
		input SignWeighInInput,
	) (*TournamentAthlete, error)

	DrawBracket(
		ctx context.Context,
		tournamentID string,
		drawType string,
	) (*Tournament, error)

	DrawBracketForCategory(
		ctx context.Context,
		tournamentID string,
		weightCategory string,
		drawType string,
	) (*Tournament, error)

	StartMatch(
		ctx context.Context,
		tournamentID string,
		matchID string,
	) (*Match, error)

	RecordMatchResult(
		ctx context.Context,
		tournamentID string,
		matchID string,
		input MatchResultInput,
	) (*Match, error)

	ClearMatchResult(
		ctx context.Context,
		tournamentID string,
		matchID string,
	) (*Match, error)

	SetMatchNumber(
		ctx context.Context,
		tournamentID string,
		matchID string,
		input SetMatchNumberInput,
	) (*Match, error)

	MoveMatch(
		ctx context.Context,
		tournamentID string,
		matchID string,
		input MoveMatchInput,
	) (*Match, error)

	ReassignMatchCourt(
		ctx context.Context,
		tournamentID string,
		matchID string,
		input ReassignMatchCourtInput,
	) (*Match, error)

	SwapMatchPositions(
		ctx context.Context,
		tournamentID string,
		matchID string,
		input SwapMatchPositionsInput,
	) (*Tournament, error)

	SwapAthletes(
		ctx context.Context,
		tournamentID string,
		matchID string,
		input SwapAthletesInput,
	) (*Tournament, error)

	ResetRankings(
		ctx context.Context,
		tournamentID string,
		weightCategory string,
	) (*Tournament, error)

	ResetCategoryBracketAndWeighIns(
		ctx context.Context,
		tournamentID string,
		weightCategory string,
	) (*Tournament, error)
}

type tournamentService struct {
	repository TournamentRepository
}

// بررسی انطباق پیاده‌سازی با interface در زمان کامپایل.
var _ TournamentService = (*tournamentService)(nil)

func NewTournamentService(
	repository TournamentRepository,
) TournamentService {
	return &tournamentService{
		repository: repository,
	}
}

type CreateTournamentInput struct {
	Name        string           `json:"name"`
	Date        string           `json:"date"`
	Courts      int              `json:"courts"`
	Gender      Gender           `json:"gender"`
	AgeCategory AgeCategory      `json:"ageCategory"`
	Format      TournamentFormat `json:"format"`

	LeagueID         *string `json:"leagueId"`
	StageOrder       *int    `json:"stageOrder"`
	WeekID           *string `json:"weekId"`
	GroupID          *string `json:"groupId"`
	TeamTournamentID *string `json:"teamTournamentId"`
}

type UpdateTournamentInput struct {
	Name        string           `json:"name"`
	Date        string           `json:"date"`
	Courts      int              `json:"courts"`
	Gender      Gender           `json:"gender"`
	AgeCategory AgeCategory      `json:"ageCategory"`
	Format      TournamentFormat `json:"format"`

	LeagueID         *string `json:"leagueId"`
	StageOrder       *int    `json:"stageOrder"`
	WeekID           *string `json:"weekId"`
	GroupID          *string `json:"groupId"`
	TeamTournamentID *string `json:"teamTournamentId"`
}

type AddAthleteInput struct {
	ProfileID      *string `json:"profileId"`
	Coach          string  `json:"coach"`
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Club           string  `json:"club"`
	WeightCategory string  `json:"weightCategory"`
	Ranking        *int    `json:"ranking"`
}

type UpdateAthleteInput struct {
	Coach          string `json:"coach"`
	Name           string `json:"name"`
	Club           string `json:"club"`
	WeightCategory string `json:"weightCategory"`
	Ranking        *int   `json:"ranking"`
}

type PreviewWeighInInput struct {
	WeightKg      float64 `json:"weightKg"`
	WithTolerance *bool   `json:"withTolerance"`
}

type RecordWeighInInput struct {
	WeightKg      float64 `json:"weightKg"`
	WithTolerance *bool   `json:"withTolerance"`
}

type SignWeighInInput struct {
	ImagePath string `json:"imagePath"`
}

type MatchResultInput struct {
	WinType      WinType      `json:"winType"`
	WinnerID     string       `json:"winnerId"`
	WinnerCorner *Corner      `json:"winnerCorner"`
	BlueID       string       `json:"blueId"`
	RedID        string       `json:"redId"`
	Rounds       []MatchRound `json:"rounds"`
	Note         *string      `json:"note"`
	RefereeID    *string      `json:"refereeId"`
}

type MoveMatchInput struct {
	Court       int     `json:"court"`
	Order       int     `json:"order"`
	SourceLabel *string `json:"sourceLabel"`
}

type ReassignMatchCourtInput struct {
	Court       int     `json:"court"`
	SourceLabel *string `json:"sourceLabel"`
}

type SwapMatchPositionsInput struct {
	OtherMatchID string `json:"otherMatchId"`
}

type SwapAthletesInput struct {
	Slot         int    `json:"slot"`
	OtherMatchID string `json:"otherMatchId"`
	OtherSlot    int    `json:"otherSlot"`
}

func (s *tournamentService) Create(
	ctx context.Context,
	input CreateTournamentInput,
) (*Tournament, error) {
	if s == nil || s.repository == nil {
		return nil, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	date, err := parseTournamentDate(input.Date)
	if err != nil {
		return nil, err
	}
	groupID, err := normalizeUUIDPointer("group id", input.GroupID)
	if err != nil {
		return nil, err
	}

	leagueID, weekID, teamTournamentID, stageOrder, err :=
		normalizeTournamentContext(
			input.LeagueID,
			input.StageOrder,
			input.WeekID,
			input.TeamTournamentID,
		)
	if err != nil {
		return nil, err
	}

	if err := validateTournamentDefinition(
		input.Name,
		input.Courts,
		input.Gender,
		input.AgeCategory,
		input.Format,
	); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	tournament := &Tournament{
		ID:               uuid.NewString(),
		Name:             strings.TrimSpace(input.Name),
		Date:             date,
		Courts:           input.Courts,
		Gender:           input.Gender,
		AgeCategory:      input.AgeCategory,
		Format:           input.Format,
		LeagueID:         leagueID,
		StageOrder:       stageOrder,
		WeekID:           weekID,
		GroupID:          groupID,
		TeamTournamentID: teamTournamentID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	normalizeTournamentCollections(tournament)

	if err := s.repository.Create(ctx, tournament); err != nil {
		return nil, fmt.Errorf("create tournament: %w", err)
	}

	return tournament, nil
}

func (s *tournamentService) GetByID(
	ctx context.Context,
	id string,
) (*Tournament, error) {
	if err := validateRequiredUUID("tournament id", id); err != nil {
		return nil, err
	}

	if s == nil || s.repository == nil {
		return nil, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	tournament, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get tournament %s: %w", id, err)
	}

	if tournament == nil {
		return nil, ErrTournamentNotFound
	}

	normalizeTournamentCollections(tournament)

	return tournament, nil
}

func (s *tournamentService) List(
	ctx context.Context,
	filter TournamentListFilter,
) ([]Tournament, error) {
	if filter.Gender != nil && !filter.Gender.IsValid() {
		return nil, fmt.Errorf(
			"%w: invalid gender",
			ErrInvalidTournamentEnum,
		)
	}

	if s == nil || s.repository == nil {
		return nil, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	if filter.AgeCategory != nil && !filter.AgeCategory.IsValid() {
		return nil, fmt.Errorf(
			"%w: invalid age category",
			ErrInvalidTournamentEnum,
		)
	}

	if filter.Format != nil && !isValidTournamentFormat(*filter.Format) {
		return nil, fmt.Errorf(
			"%w: invalid tournament format",
			ErrInvalidTournamentEnum,
		)
	}

	var err error

	filter.LeagueID, err = normalizeFilterUUID(
		"league id",
		filter.LeagueID,
	)
	if err != nil {
		return nil, err
	}
	filter.WeekID, err = normalizeFilterUUID(
		"week id",
		filter.WeekID,
	)
	if err != nil {
		return nil, err
	}

	filter.TeamTournamentID, err = normalizeFilterUUID(
		"team tournament id",
		filter.TeamTournamentID,
	)
	if err != nil {
		return nil, err
	}

	if filter.EventDateFrom != nil {
		from := filter.EventDateFrom.UTC()
		filter.EventDateFrom = &from
	}

	if filter.EventDateTo != nil {
		to := filter.EventDateTo.UTC()
		filter.EventDateTo = &to
	}

	if filter.EventDateFrom != nil &&
		filter.EventDateTo != nil &&
		filter.EventDateFrom.After(*filter.EventDateTo) {
		return nil, fmt.Errorf(
			"%w: event date range is reversed",
			ErrInvalidTournamentInput,
		)
	}

	tournaments, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list tournaments: %w", err)
	}

	for index := range tournaments {
		normalizeTournamentCollections(&tournaments[index])
	}

	return tournaments, nil
}

func (s *tournamentService) Update(
	ctx context.Context,
	id string,
	input UpdateTournamentInput,
) (*Tournament, error) {
	if err := validateRequiredUUID("tournament id", id); err != nil {
		return nil, err
	}

	date, err := parseTournamentDate(input.Date)
	if err != nil {
		return nil, err
	}

	if s == nil || s.repository == nil {
		return nil, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	leagueID, weekID, teamTournamentID, stageOrder, err :=
		normalizeTournamentContext(
			input.LeagueID,
			input.StageOrder,
			input.WeekID,
			input.TeamTournamentID,
		)
	if err != nil {
		return nil, err
	}
	groupID, err := normalizeUUIDPointer("group id", input.GroupID)
	if err != nil {
		return nil, err
	}

	if err := validateTournamentDefinition(
		input.Name,
		input.Courts,
		input.Gender,
		input.AgeCategory,
		input.Format,
	); err != nil {
		return nil, err
	}

	tournament, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get tournament %s: %w", id, err)
	}

	if tournament == nil {
		return nil, ErrTournamentNotFound
	}

	if len(tournament.Athletes) > 0 && (tournament.Gender != input.Gender || tournament.AgeCategory != input.AgeCategory || tournament.Format != input.Format) {
		return nil, fmt.Errorf("%w: category settings are locked after registration", ErrAthleteLocked)
	}
	for _, m := range tournament.Matches {
		if m.Court > input.Courts {
			return nil, ErrInvalidCourt
		}
	}
	oldSettings := settingsFor(tournament)
	if hasStartedMatches(tournament) && (tournament.Courts != input.Courts || !tournament.Date.Equal(date)) {
		return nil, ErrMatchLocked
	}
	delta := date.Sub(tournament.Date)
	for i := range oldSettings.Schedule.Days {
		d, e := time.Parse("2006-01-02", oldSettings.Schedule.Days[i].Date)
		if e != nil {
			return nil, ErrInvalidTournamentDate
		}
		oldSettings.Schedule.Days[i].Date = d.Add(delta).Format("2006-01-02")
	}
	tournament.Settings = &oldSettings
	tournament.Name = strings.TrimSpace(input.Name)
	tournament.Date = date
	tournament.Courts = input.Courts
	tournament.Gender = input.Gender
	tournament.AgeCategory = input.AgeCategory
	tournament.Format = input.Format
	tournament.LeagueID = leagueID
	tournament.StageOrder = stageOrder
	tournament.WeekID = weekID
	tournament.GroupID = groupID
	tournament.TeamTournamentID = teamTournamentID
	tournament.UpdatedAt = time.Now().UTC()
	if err := validateSettings(tournament, settingsFor(tournament)); err != nil {
		return nil, err
	}
	if !hasStartedMatches(tournament) {
		clearMatchNumbers(tournament)
		if err := applyCourtPlan(tournament); err != nil {
			return nil, err
		}
	}
	normalizeTournamentCollections(tournament)

	if err := s.repository.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf("update tournament %s: %w", id, err)
	}

	return tournament, nil
}

func (s *tournamentService) Delete(
	ctx context.Context,
	id string,
) error {
	if err := validateRequiredUUID("tournament id", id); err != nil {
		return err
	}

	if s == nil || s.repository == nil {
		return fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	// ابتدا وجود aggregate بررسی می‌شود تا رفتار سرویس
	// مستقل از پیاده‌سازی repository باشد.
	tournament, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get tournament %s before delete: %w", id, err)
	}

	if tournament == nil {
		return ErrTournamentNotFound
	}

	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete tournament %s: %w", id, err)
	}

	return nil
}

func parseTournamentDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf(
			"%w: date is required",
			ErrInvalidTournamentDate,
		)
	}

	if date, err := time.ParseInLocation(
		"2006-01-02",
		value,
		time.UTC,
	); err == nil {
		return date.UTC(), nil
	}

	if date, err := time.Parse(time.RFC3339, value); err == nil {
		return date.UTC(), nil
	}

	return time.Time{}, fmt.Errorf(
		"%w: expected YYYY-MM-DD or RFC3339",
		ErrInvalidTournamentDate,
	)
}

func validateTournamentDefinition(
	name string,
	courts int,
	gender Gender,
	ageCategory AgeCategory,
	format TournamentFormat,
) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf(
			"%w: tournament name is required",
			ErrInvalidTournamentInput,
		)
	}

	if courts <= 0 || courts > 64 {
		return fmt.Errorf(
			"%w: courts must be between 1 and 64",
			ErrInvalidTournamentInput,
		)
	}

	if !gender.IsValid() {
		return fmt.Errorf(
			"%w: invalid gender %q",
			ErrInvalidTournamentEnum,
			gender,
		)
	}

	if !ageCategory.IsValid() {
		return fmt.Errorf(
			"%w: invalid age category %q",
			ErrInvalidTournamentEnum,
			ageCategory,
		)
	}

	if !isValidTournamentFormat(format) {
		return fmt.Errorf(
			"%w: invalid tournament format %q",
			ErrInvalidTournamentEnum,
			format,
		)
	}

	return nil
}

func isValidTournamentFormat(
	format TournamentFormat,
) bool {
	switch format {
	case TournamentFormatGrandPrix,
		TournamentFormatTeam:
		return true
	default:
		return false
	}
}

func validateRequiredUUID(
	name string,
	value string,
) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf(
			"%w: %s is required",
			ErrInvalidTournamentInput,
			name,
		)
	}

	if _, err := uuid.Parse(value); err != nil {
		return fmt.Errorf(
			"%w: invalid %s: %v",
			ErrInvalidTournamentInput,
			name,
			err,
		)
	}

	return nil
}

func normalizeUUIDPointer(
	name string,
	value *string,
) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(strings.TrimSpace(*value))
	if err != nil {
		return nil, fmt.Errorf(
			"%w: invalid %s: %v",
			ErrInvalidTournamentInput,
			name,
			err,
		)
	}

	result := parsed.String()

	return &result, nil
}

func normalizeFilterUUID(
	name string,
	value *string,
) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return value, nil
	}

	parsed, err := uuid.Parse(strings.TrimSpace(*value))
	if err != nil {
		return nil, fmt.Errorf(
			"%w: invalid %s: %v",
			ErrInvalidTournamentInput,
			name,
			err,
		)
	}

	result := parsed.String()

	return &result, nil
}

func normalizeTournamentContext(
	leagueID *string,
	stageOrder *int,
	weekID *string,
	TeamTournamentID *string,
) (
	normalizedLeagueID *string,
	normalizedWeekID *string,
	normalizedTeamTournamentID *string,
	normalizedStageOrder *int,
	err error,
) {
	normalizedLeagueID, err = normalizeUUIDPointer(
		"league id",
		leagueID,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	normalizedWeekID, err = normalizeUUIDPointer(
		"week id",
		weekID,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	normalizedTeamTournamentID, err = normalizeUUIDPointer(
		"team tournament id",
		TeamTournamentID,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	hasLeague := normalizedLeagueID != nil
	hasWeek := normalizedWeekID != nil
	hasStage := stageOrder != nil

	// LeagueID، WeekID و StageOrder یک context واحد هستند.
	// یا هر سه مقداردهی می‌شوند یا هیچ‌کدام.
	if hasLeague || hasWeek || hasStage {
		if !hasLeague || !hasWeek || !hasStage {
			return nil, nil, nil, nil, fmt.Errorf(
				"%w: league id, week id and stage order must be provided together",
				ErrInvalidTournamentContext,
			)
		}

		if *stageOrder <= 0 {
			return nil, nil, nil, nil, fmt.Errorf(
				"%w: stage order must be greater than zero",
				ErrInvalidTournamentContext,
			)
		}

		stage := *stageOrder
		normalizedStageOrder = &stage
	}

	return normalizedLeagueID,
		normalizedWeekID,
		normalizedTeamTournamentID,
		normalizedStageOrder,
		nil
}

func (s *tournamentService) AddAthlete(
	ctx context.Context,
	tournamentID string,
	input AddAthleteInput,
) (*TournamentAthlete, error) {
	if s == nil || s.repository == nil {
		return nil, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	if err := validateRequiredUUID("tournament id", tournamentID); err != nil {
		return nil, err
	}

	tournament, err := s.repository.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf(
			"get tournament %s before adding athlete: %w",
			tournamentID,
			err,
		)
	}

	if tournament == nil {
		return nil, ErrTournamentNotFound
	}

	normalizeTournamentCollections(tournament)

	if err := s.resolveProfile(ctx, tournament, &input); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	club := strings.TrimSpace(input.Club)
	weightCategory := strings.TrimSpace(input.WeightCategory)

	if name == "" {
		return nil, fmt.Errorf(
			"%w: athlete name is required",
			ErrInvalidTournamentInput,
		)
	}

	if weightCategory == "" {
		return nil, fmt.Errorf(
			"%w: weight category is required",
			ErrInvalidTournamentInput,
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

	if input.Ranking != nil && *input.Ranking <= 0 {
		return nil, fmt.Errorf(
			"%w: ranking must be greater than zero",
			ErrInvalidTournamentInput,
		)
	}

	if input.Ranking != nil {
		for _, athlete := range tournament.Athletes {
			if athlete.WeightCategory == weightCategory &&
				athlete.Ranking != nil &&
				*athlete.Ranking == *input.Ranking {
				return nil, fmt.Errorf(
					"%w: ranking %d is already assigned to %s",
					ErrDuplicateAthlete,
					*input.Ranking,
					athlete.Name,
				)
			}
		}
	}

	athleteID := strings.TrimSpace(input.ID)
	if athleteID == "" {
		athleteID = uuid.NewString()
	}

	if err := validateRequiredUUID("athlete id", athleteID); err != nil {
		return nil, err
	}

	for _, athlete := range tournament.Athletes {
		if athlete.ID == athleteID {
			return nil, fmt.Errorf(
				"%w: athlete id %s already exists in tournament",
				ErrDuplicateAthlete,
				athleteID,
			)
		}
	}

	if bracketExistsForCategory(tournament.Matches, weightCategory) || hasStartedMatches(tournament) {
		return nil, ErrAthleteLocked
	}
	clearMatchNumbers(tournament)

	athlete := TournamentAthlete{
		ProfileID:      cloneStringPointer(input.ProfileID),
		Coach:          strings.TrimSpace(input.Coach),
		ID:             athleteID,
		Number:         nextAthleteNumber(tournament.Athletes),
		Name:           name,
		Club:           club,
		WeightCategory: weightCategory,
		Ranking:        cloneIntPointer(input.Ranking),
		WeighedIn:      false,
		WeighIn:        newPendingWeighIn(),
	}

	tournament.Athletes = append(tournament.Athletes, athlete)
	tournament.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf(
			"update tournament %s after adding athlete: %w",
			tournamentID,
			err,
		)
	}

	return &athlete, nil
}

func (s *tournamentService) UpdateAthlete(
	ctx context.Context,
	tournamentID string,
	athleteID string,
	input UpdateAthleteInput,
) (*TournamentAthlete, error) {
	if s == nil || s.repository == nil {
		return nil, fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	if err := validateRequiredUUID("tournament id", tournamentID); err != nil {
		return nil, err
	}

	if err := validateRequiredUUID("athlete id", athleteID); err != nil {
		return nil, err
	}

	tournament, err := s.repository.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf(
			"get tournament %s before updating athlete: %w",
			tournamentID,
			err,
		)
	}

	if tournament == nil {
		return nil, ErrTournamentNotFound
	}

	normalizeTournamentCollections(tournament)

	index := findTournamentAthleteIndex(
		tournament.Athletes,
		athleteID,
	)
	if index == -1 {
		return nil, ErrAthleteNotFound
	}

	current := tournament.Athletes[index]

	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf(
			"%w: athlete name is required",
			ErrInvalidTournamentInput,
		)
	}

	weightCategory := strings.TrimSpace(input.WeightCategory)

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

	if input.Ranking != nil && *input.Ranking <= 0 {
		return nil, fmt.Errorf(
			"%w: ranking must be greater than zero",
			ErrInvalidTournamentInput,
		)
	}

	if input.Ranking != nil {
		for i, athlete := range tournament.Athletes {
			if i == index {
				continue
			}

			if athlete.WeightCategory == weightCategory &&
				athlete.Ranking != nil &&
				*athlete.Ranking == *input.Ranking {
				return nil, fmt.Errorf(
					"%w: ranking %d is already assigned to %s",
					ErrDuplicateAthlete,
					*input.Ranking,
					athlete.Name,
				)
			}
		}
	}

	if current.WeightCategory != weightCategory && bracketExistsForCategory(tournament.Matches, weightCategory) {
		return nil, ErrAthleteLocked
	}
	if current.WeightCategory != weightCategory &&
		current.WeighIn != nil &&
		(current.WeighIn.Status != WeighInPending ||
			len(current.WeighIn.Attempts) > 0) {
		return nil, fmt.Errorf(
			"%w: weight category cannot change after weigh-in has started",
			ErrAthleteLocked,
		)
	}

	if current.WeightCategory != weightCategory &&
		athleteIsReferencedByMatches(tournament.Matches, athleteID) {
		return nil, fmt.Errorf(
			"%w: weight category cannot change after athlete is assigned to matches",
			ErrAthleteLocked,
		)
	}

	if current.ProfileID != nil && (current.Name != strings.TrimSpace(input.Name) || current.Club != strings.TrimSpace(input.Club) || current.Coach != strings.TrimSpace(input.Coach)) {
		return nil, fmt.Errorf("%w: linked identity snapshot cannot be overwritten", ErrAthleteLocked)
	}
	current.Coach = strings.TrimSpace(input.Coach)
	current.Name = strings.TrimSpace(input.Name)
	current.Club = strings.TrimSpace(input.Club)
	current.WeightCategory = weightCategory
	current.Ranking = cloneIntPointer(input.Ranking)

	if current.WeighIn == nil {
		current.WeighIn = newPendingWeighIn()
	}

	current.WeighedIn = current.WeighIn.Status == WeighInPassed
	tournament.Athletes[index] = current
	tournament.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, tournament); err != nil {
		return nil, fmt.Errorf(
			"update tournament %s after updating athlete: %w",
			tournamentID,
			err,
		)
	}

	return &current, nil
}

func (s *tournamentService) RemoveAthlete(
	ctx context.Context,
	tournamentID string,
	athleteID string,
) error {
	if s == nil || s.repository == nil {
		return fmt.Errorf(
			"%w: tournament repository is nil",
			ErrInvalidTournamentInput,
		)
	}

	if err := validateRequiredUUID("tournament id", tournamentID); err != nil {
		return err
	}

	if err := validateRequiredUUID("athlete id", athleteID); err != nil {
		return err
	}

	tournament, err := s.repository.GetByID(ctx, tournamentID)
	if err != nil {
		return fmt.Errorf(
			"get tournament %s before removing athlete: %w",
			tournamentID,
			err,
		)
	}

	if tournament == nil {
		return ErrTournamentNotFound
	}

	normalizeTournamentCollections(tournament)

	index := findTournamentAthleteIndex(
		tournament.Athletes,
		athleteID,
	)
	if index == -1 {
		return ErrAthleteNotFound
	}

	// قفل باید پیش از تغییر aggregate و ذخیره بررسی شود.
	if athleteIsReferencedByMatches(tournament.Matches, athleteID) {
		return fmt.Errorf(
			"%w: athlete %s is referenced by a match",
			ErrAthleteLocked,
			athleteID,
		)
	}

	tournament.Athletes = append(
		tournament.Athletes[:index],
		tournament.Athletes[index+1:]...,
	)
	tournament.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, tournament); err != nil {
		return fmt.Errorf(
			"update tournament %s after removing athlete: %w",
			tournamentID,
			err,
		)
	}

	return nil
}

func findTournamentAthleteIndex(
	athletes []TournamentAthlete,
	athleteID string,
) int {
	for index := range athletes {
		if athletes[index].ID == athleteID {
			return index
		}
	}

	return -1
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}

	result := *value
	return &result
}

func newPendingWeighIn() *WeighIn {
	return &WeighIn{
		Status:   WeighInPending,
		Attempts: make([]WeighInAttempt, 0),
	}
}

func athleteIsReferencedByMatches(
	matches []Match,
	athleteID string,
) bool {
	for _, match := range matches {
		if match.Athlete1ID != nil &&
			*match.Athlete1ID == athleteID {
			return true
		}

		if match.Athlete2ID != nil &&
			*match.Athlete2ID == athleteID {
			return true
		}

		if match.WinnerID != nil &&
			*match.WinnerID == athleteID {
			return true
		}

		if match.Result != nil {
			if match.Result.BlueID != nil &&
				*match.Result.BlueID == athleteID {
				return true
			}

			if match.Result.RedID != nil &&
				*match.Result.RedID == athleteID {
				return true
			}

			if match.Result.WinnerID != nil &&
				*match.Result.WinnerID == athleteID {
				return true
			}
		}
	}

	return false
}

func nextAthleteNumber(athletes []TournamentAthlete) int {
	maxNumber := 0
	for _, a := range athletes {
		if a.Number > maxNumber {
			maxNumber = a.Number
		}
	}
	return maxNumber + 1
}
