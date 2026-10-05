package tournaments

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"
)

// Settings are saved with the event so rule changes cannot rewrite old events.
type TournamentSettings struct {
	Scoring   ScoringSettings   `json:"scoring"`
	WeighIn   WeighInSettings   `json:"weighIn"`
	Draw      DrawSettings      `json:"draw"`
	Schedule  ScheduleSettings  `json:"schedule"`
	Numbering NumberingSettings `json:"numbering"`
	Hogu      HoguSettings      `json:"hogu"`
}

// HoguSettings map required electronic protector sizes to the courts equipped for them.
type HoguSettings struct {
	Enabled       bool              `json:"enabled"`
	CourtSizes    map[int][]string  `json:"courtSizes"`
	CategorySizes map[string]string `json:"categorySizes"`
}

type ScoringSettings struct {
	PointGap             int `json:"pointGap"`
	PointGapFromRound    int `json:"pointGapFromRound"`
	GamJeomLimitPerRound int `json:"gamJeomLimitPerRound"`
	RoundsToWin          int `json:"roundsToWin"`
	MaxRounds            int `json:"maxRounds"`
}

func defaultScoring() ScoringSettings {
	return ScoringSettings{PointGap: 15, PointGapFromRound: 1, GamJeomLimitPerRound: 5, RoundsToWin: 2, MaxRounds: 3}
}

type WeighInSettings struct {
	MaxAttempts int     `json:"maxAttempts"`
	ToleranceKg float64 `json:"toleranceKg"`
}
type DrawSettings struct {
	Type          string `json:"type"`
	SeparateTeams bool   `json:"separateTeams"`
}
type EventDay struct {
	Day   int    `json:"day"`
	Date  string `json:"date"`
	Label string `json:"label"`
}
type ScheduleSettings struct {
	Mode             string     `json:"mode"`
	SingleCourt      int        `json:"singleCourt"`
	FinalCourtPolicy string     `json:"finalCourtPolicy"`
	FinalCourt       int        `json:"finalCourt"`
	DayAssignment    string     `json:"dayAssignment"`
	Days             []EventDay `json:"days"`
	// Maps an actual weight category (e.g. -54), never its numeric parity, to a day.
	CategoryDays map[string]int `json:"categoryDays"`
}

type NumberingSettings struct {
	StartAt int    `json:"startAt"`
	Scope   string `json:"scope"`
	Order   string `json:"order"`
}

func DefaultSettings(date time.Time) TournamentSettings {
	return TournamentSettings{
		Scoring:   defaultScoring(),
		WeighIn:   WeighInSettings{MaxAttempts: 2, ToleranceKg: 0.2},
		Draw:      DrawSettings{Type: "random", SeparateTeams: true},
		Schedule:  ScheduleSettings{Mode: "balanced", SingleCourt: 1, FinalCourtPolicy: "primary", FinalCourt: 1, DayAssignment: "manual", Days: []EventDay{{Day: 1, Date: date.Format("2006-01-02"), Label: "روز اول"}}, CategoryDays: map[string]int{}},
		Numbering: NumberingSettings{StartAt: 1, Scope: "tournament", Order: "rounds"},
		Hogu:      HoguSettings{CourtSizes: map[int][]string{}, CategorySizes: map[string]string{}},
	}
}
func settingsFor(t *Tournament) TournamentSettings {
	if t.Settings == nil {
		return DefaultSettings(t.Date)
	}
	return *t.Settings
}
func dayForCategory(t *Tournament, category string) int {
	day := settingsFor(t).Schedule.CategoryDays[category]
	if day == 0 {
		return 1
	}
	return day
}
func validHoguSize(size string) bool {
	switch size {
	case "1", "2", "3", "4":
		return true
	default:
		return false
	}
}

func validateSettings(t *Tournament, s TournamentSettings) error {
	invalid := func(message string) error { return fmt.Errorf("%w: %s", ErrInvalidTournamentInput, message) }
	if s.Scoring.PointGap < 1 || s.Scoring.PointGap > 1000 || s.Scoring.PointGapFromRound < 1 || s.Scoring.PointGapFromRound > s.Scoring.MaxRounds || s.Scoring.GamJeomLimitPerRound < 1 || s.Scoring.GamJeomLimitPerRound > 100 || s.Scoring.RoundsToWin < 1 || s.Scoring.RoundsToWin > 5 || s.Scoring.MaxRounds != 2*s.Scoring.RoundsToWin-1 {
		return invalid("invalid scoring rules")
	}
	switch s.Schedule.Mode {
	case "single_court", "split_halves", "balanced":
	default:
		return invalid("unknown schedule mode")
	}
	if s.Schedule.Mode == "split_halves" && t.Courts < 2 {
		return invalid("split_halves requires at least two courts")
	}
	if s.Schedule.SingleCourt < 1 || s.Schedule.SingleCourt > t.Courts || s.Schedule.FinalCourt < 1 || s.Schedule.FinalCourt > t.Courts {
		return invalid("configured court is outside the tournament")
	}
	if s.Schedule.FinalCourtPolicy != "primary" && s.Schedule.FinalCourtPolicy != "fixed" {
		return invalid("finalCourtPolicy must be primary or fixed")
	}
	if s.Schedule.DayAssignment != "manual" && s.Schedule.DayAssignment != "alternating" {
		return invalid("dayAssignment must be manual or alternating")
	}
	if s.Schedule.DayAssignment == "alternating" && len(s.Schedule.Days) != 2 {
		return invalid("alternating assignment requires two days")
	}
	if s.Numbering.StartAt < 1 || s.Numbering.StartAt > 1000000 {
		return invalid("startAt must be 1–1000000")
	}
	switch s.Numbering.Scope {
	case "tournament", "day", "court_day":
	default:
		return invalid("unknown numbering scope")
	}
	if s.Numbering.Order != "rounds" && s.Numbering.Order != "category" {
		return invalid("numbering order must be rounds or category")
	}
	if s.Hogu.Enabled {
		for court := 1; court <= t.Courts; court++ {
			if len(s.Hogu.CourtSizes[court]) == 0 {
				return invalid(fmt.Sprintf("choose at least one hogu size for court %d", court))
			}
			seen := map[string]bool{}
			for _, size := range s.Hogu.CourtSizes[court] {
				size = strings.TrimSpace(size)
				if !validHoguSize(size) || seen[size] {
					return invalid(fmt.Sprintf("invalid or repeated hogu size on court %d (allowed sizes: 1–4)", court))
				}
				seen[size] = true
			}
		}
		for category, size := range s.Hogu.CategorySizes {
			if !IsValidWeightCategory(t.AgeCategory, t.Gender, category) || !validHoguSize(strings.TrimSpace(size)) {
				return invalid("invalid category hogu size (allowed sizes: 1–4)")
			}
		}
		for _, match := range t.Matches {
			if lifecycleIsBye(&match) {
				continue
			}
			if strings.TrimSpace(s.Hogu.CategorySizes[match.WeightCategory]) == "" {
				return invalid(fmt.Sprintf("choose a hogu size for category %s", match.WeightCategory))
			}
		}
	}
	if s.WeighIn.MaxAttempts < 1 || s.WeighIn.MaxAttempts > 10 {
		return invalid("weigh-in maxAttempts must be 1–10")
	}
	if math.IsNaN(s.WeighIn.ToleranceKg) || math.IsInf(s.WeighIn.ToleranceKg, 0) || s.WeighIn.ToleranceKg < 0 || s.WeighIn.ToleranceKg > 5 {
		return invalid("weigh-in toleranceKg must be 0–5")
	}
	if _, err := normalizeBracketDrawType(s.Draw.Type); err != nil {
		return err
	}
	if len(s.Schedule.Days) < 1 || len(s.Schedule.Days) > 2 {
		return invalid("provide one or two event days")
	}
	var previous time.Time
	for i, day := range s.Schedule.Days {
		date, err := time.Parse("2006-01-02", day.Date)
		if err != nil || day.Day != i+1 {
			return invalid("days must be numbered from 1 and have YYYY-MM-DD dates")
		}
		if i == 0 && day.Date != t.Date.Format("2006-01-02") {
			return invalid("first day must match tournament date")
		}
		if i > 0 && !date.After(previous) {
			return invalid("second day must be after first day")
		}
		previous = date
	}
	for cat, day := range s.Schedule.CategoryDays {
		if !IsValidWeightCategory(t.AgeCategory, t.Gender, cat) || day < 1 || day > len(s.Schedule.Days) {
			return invalid("invalid category day assignment")
		}
	}
	if len(s.Schedule.Days) == 2 {
		for _, cat := range WeightOptions(t.AgeCategory, t.Gender) {
			if s.Schedule.CategoryDays[cat] == 0 {
				return invalid("two-day events require a day for every weight category")
			}
		}
	}
	return nil
}
func (s *tournamentService) UpdateSettings(ctx context.Context, id string, input TournamentSettings) (*Tournament, error) {
	t, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Schedule.DayAssignment == "alternating" {
		input.Schedule.CategoryDays = map[string]int{}
		for i, cat := range WeightOptions(t.AgeCategory, t.Gender) {
			input.Schedule.CategoryDays[cat] = i%2 + 1
		}
	}
	if err = validateSettings(t, input); err != nil {
		return nil, err
	}
	old := settingsFor(t)
	if reflect.DeepEqual(old, input) {
		return t, nil
	}
	if hasStartedMatches(t) && old.Scoring != input.Scoring {
		return nil, ErrMatchLocked
	}
	if hasStartedMatches(t) && (!reflect.DeepEqual(old.Schedule, input.Schedule) || old.Numbering != input.Numbering || old.Draw != input.Draw || !reflect.DeepEqual(old.Hogu, input.Hogu)) {
		return nil, ErrMatchLocked
	}
	if old.WeighIn != input.WeighIn {
		for _, a := range t.Athletes {
			if a.WeighIn != nil && (len(a.WeighIn.Attempts) > 0 || a.WeighIn.Status != WeighInPending) {
				return nil, fmt.Errorf("%w: weigh-in rules are locked after first attempt", ErrAthleteLocked)
			}
		}
	}
	// Do not move played or ongoing matches into a different event day.
	for _, m := range t.Matches {
		newDay := input.Schedule.CategoryDays[m.WeightCategory]
		if newDay == 0 {
			newDay = 1
		}
		if !lifecycleIsBye(&m) && (lifecycleHasResult(&m) || lifecycleMatchStatus(&m) == MatchStatusOngoing) {
			oldDay := dayForCategory(t, m.WeightCategory)
			if oldDay != newDay || old.Schedule.Days[oldDay-1].Date != input.Schedule.Days[newDay-1].Date {
				return nil, ErrMatchLocked
			}
		}
	}
	if len(t.Matches) > 0 && (old.Draw.Type != input.Draw.Type || old.Draw.SeparateTeams != input.Draw.SeparateTeams) {
		return nil, fmt.Errorf("%w: reset brackets before changing draw settings", ErrBracketAlreadyDrawn)
	}
	if !reflect.DeepEqual(old.Schedule, input.Schedule) {
		for i := range t.Matches {
			t.Matches[i].MovedAt = nil
		}
	}
	t.Settings = &input
	if !reflect.DeepEqual(old, input) && !hasStartedMatches(t) {
		clearMatchNumbers(t)
	}
	for i := range t.Matches {
		t.Matches[i].Day = dayForCategory(t, t.Matches[i].WeightCategory)
	}
	if err = applyCourtPlan(t); err != nil {
		return nil, err
	}
	t.UpdatedAt = time.Now().UTC()
	if err = s.repository.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}
