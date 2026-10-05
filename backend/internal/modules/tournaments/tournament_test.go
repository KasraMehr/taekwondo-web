package tournaments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"math"
	"reflect"
	"testing"
	"time"
)

type memoryRepository struct {
	value    *Tournament
	profiles map[string]*AthleteProfile
	writes   int
}

func copyTournament(t *Tournament) *Tournament {
	if t == nil {
		return nil
	}
	b, _ := json.Marshal(t)
	var out Tournament
	_ = json.Unmarshal(b, &out)
	return &out
}
func (r *memoryRepository) Create(_ context.Context, t *Tournament) error {
	r.value = copyTournament(t)
	r.value.Revision = 1
	t.Revision = 1
	r.writes++
	return nil
}
func (r *memoryRepository) GetByID(_ context.Context, id string) (*Tournament, error) {
	if r.value == nil || r.value.ID != id {
		return nil, ErrTournamentNotFound
	}
	return copyTournament(r.value), nil
}
func (r *memoryRepository) List(context.Context, TournamentListFilter) ([]Tournament, error) {
	return []Tournament{*copyTournament(r.value)}, nil
}
func (r *memoryRepository) Update(_ context.Context, t *Tournament) error {
	if r.value.Revision != t.Revision {
		return ErrConflict
	}
	t.Revision++
	r.value = copyTournament(t)
	r.writes++
	return nil
}
func (r *memoryRepository) Delete(context.Context, string) error { r.value = nil; return nil }
func (r *memoryRepository) GetAthleteProfile(_ context.Context, id string) (*AthleteProfile, error) {
	p := r.profiles[id]
	if p == nil {
		return nil, ErrAthleteNotFound
	}
	return p, nil
}
func newFixture(t *testing.T) (TournamentService, *memoryRepository, string) {
	t.Helper()
	r := &memoryRepository{profiles: map[string]*AthleteProfile{}}
	s := NewTournamentService(r)
	event, err := s.Create(context.Background(), CreateTournamentInput{Name: "جام تست", Date: "2026-10-01", Courts: 4, Gender: GenderMale, AgeCategory: AgeCategoryBozorgsalan, Format: TournamentFormatGrandPrix})
	if err != nil {
		t.Fatal(err)
	}
	return s, r, event.ID
}
func enter(t *testing.T, s TournamentService, id, cat string, n int) {
	t.Helper()
	for i := 1; i <= n; i++ {
		rank := i
		a, err := s.AddAthlete(context.Background(), id, AddAthleteInput{Name: fmt.Sprint("athlete ", cat, " ", i), Club: fmt.Sprint("club ", i), WeightCategory: cat, Ranking: &rank})
		if err != nil {
			t.Fatal(err)
		}
		limit := 54.0
		if cat == "-58" {
			limit = 58
		}
		if cat == "-63" {
			limit = 63
		}
		if cat == "-68" {
			limit = 68
		}
		if _, err = s.RecordWeighIn(context.Background(), id, a.ID, RecordWeighInInput{WeightKg: limit}); err != nil {
			t.Fatal(err)
		}
	}
}
func TestWeightBoundaries(t *testing.T) {
	categories := WeightOptions(AgeCategoryBozorgsalan, GenderMale)
	cases := []struct {
		cat           string
		weight        float64
		ok, tolerance bool
	}{{"-58", 54, false, false}, {"-58", 54.001, true, false}, {"-58", 58, true, false}, {"-58", 58.2, true, true}, {"-58", 58.201, false, false}, {"-54", 20, true, false}, {"+87", 87, false, false}, {"+87", 87.001, true, false}}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s_%g", tc.cat, tc.weight), func(t *testing.T) {
			got, err := evaluateWeighInWeight(tc.cat, tc.weight, true, categories)
			if err != nil || got.ok != tc.ok || got.usedTolerance != tc.tolerance {
				t.Fatalf("got %+v %v", got, err)
			}
		})
	}
	for _, invalid := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := evaluateWeighInWeight("-58", invalid, true, categories); err == nil {
			t.Fatal("accepted invalid weight")
		}
	}
}
func TestWeighInAttemptsResetAndSignature(t *testing.T) {
	s, r, id := newFixture(t)
	a, err := s.AddAthlete(context.Background(), id, AddAthleteInput{Name: "A", WeightCategory: "-58"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = s.RecordWeighIn(context.Background(), id, a.ID, RecordWeighInInput{WeightKg: 59}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.RecordWeighIn(context.Background(), id, a.ID, RecordWeighInInput{WeightKg: 58}); !errors.Is(err, ErrMaxWeighInAttempts) {
		t.Fatalf("third attempt: %v", err)
	}
	if _, err = s.ResetWeighIn(context.Background(), id, a.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.RecordWeighIn(context.Background(), id, a.ID, RecordWeighInInput{WeightKg: 58.2})
	if err != nil || !got.WeighIn.WithTolerance {
		t.Fatalf("tolerance: %+v %v", got, err)
	}
	if _, err = s.SignWeighIn(context.Background(), id, a.ID, SignWeighInInput{ImagePath: "signature.png"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SignWeighIn(context.Background(), id, a.ID, SignWeighInInput{ImagePath: "signature2.png"}); !errors.Is(err, ErrWeighInAlreadySigned) {
		t.Fatal(err)
	}
	before := r.writes
	if _, err = s.ResetCategoryBracketAndWeighIns(context.Background(), id, "-58"); !errors.Is(err, ErrMatchNotFound) || r.writes != before {
		t.Fatal("reset without a bracket must not destroy weigh-ins")
	}
}

func TestLinkedProfileIdentityAndIsolation(t *testing.T) {
	s, r, id := newFixture(t)
	profileID := uuid.NewString()
	gender := "male"
	r.profiles[profileID] = &AthleteProfile{ID: profileID, Name: "Real Name", Club: "Club A", Gender: &gender, Active: true}
	a, err := s.AddAthlete(context.Background(), id, AddAthleteInput{ProfileID: &profileID, Name: "Forged", Club: "Forged", WeightCategory: "-54"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "Real Name" || a.Club != "Club A" || a.ProfileID == nil {
		t.Fatalf("identity was not snapshotted: %+v", a)
	}
	r.profiles[profileID].Club = "Club B"
	if r.value.Athletes[0].Club != "Club A" {
		t.Fatal("historical club changed")
	}
	if _, err = s.AddAthlete(context.Background(), id, AddAthleteInput{ProfileID: &profileID, WeightCategory: "-58"}); !errors.Is(err, ErrDuplicateAthlete) {
		t.Fatal(err)
	}
	missing := uuid.NewString()
	if _, err = s.AddAthlete(context.Background(), id, AddAthleteInput{ProfileID: &missing, WeightCategory: "-54"}); !errors.Is(err, ErrAthleteNotFound) {
		t.Fatal(err)
	}
	r.profiles[profileID].Active = false
	r.value.Athletes = nil
	if _, err = s.AddAthlete(context.Background(), id, AddAthleteInput{ProfileID: &profileID, WeightCategory: "-54"}); err == nil {
		t.Fatal("inactive profile accepted")
	}
}
func TestSeedOrderAndByes(t *testing.T) {
	if got := bracketSeedOrder(8); !reflect.DeepEqual(got, []int{1, 8, 5, 4, 3, 6, 7, 2}) {
		t.Fatal(got)
	}
	for n := 2; n <= 33; n++ {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			s, r, id := newFixture(t)
			enter(t, s, id, "-54", n)
			draw, err := s.DrawBracket(context.Background(), id, "ranking")
			if err != nil {
				t.Fatal(err)
			}
			if len(draw.Matches) != nextPowerOfTwo(n)-1 {
				t.Fatal("wrong match count")
			}
			seen := map[string]bool{}
			playable := 0
			for _, m := range draw.Matches {
				if m.MatchNumber != nil {
					t.Fatal("number assigned during draw")
				}
				if !lifecycleIsBye(&m) {
					playable++
				} else if m.Court != 0 || m.Order != 0 || m.WinnerID == nil {
					t.Fatalf("invalid bye %+v", m)
				}
				if m.Round == 1 {
					for _, id := range []*string{m.Athlete1ID, m.Athlete2ID} {
						if id != nil {
							if seen[*id] {
								t.Fatal("duplicate entrant")
							}
							seen[*id] = true
						}
					}
				}
			}
			if len(seen) != n || playable != n-1 {
				t.Fatalf("participants %d, games %d", len(seen), playable)
			}
			if err = lifecycleValidateBracket(r.value); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestDrawUsesStructuredWeighInAndLocksEntries(t *testing.T) {
	s, r, id := newFixture(t)
	enter(t, s, id, "-54", 3)
	r.value.Athletes[0].WeighedIn = true
	r.value.Athletes[0].WeighIn.Status = WeighInFailed
	draw, err := s.DrawBracket(context.Background(), id, "ranking")
	if err != nil {
		t.Fatal(err)
	}
	if len(draw.Matches) != 1 {
		t.Fatal("legacy flag admitted failed athlete")
	}
	if _, err = s.AddAthlete(context.Background(), id, AddAthleteInput{Name: "Late", WeightCategory: "-54"}); !errors.Is(err, ErrAthleteLocked) {
		t.Fatal(err)
	}
	for _, a := range draw.Athletes {
		if athleteIsReferencedByMatches(draw.Matches, a.ID) {
			if err = s.RemoveAthlete(context.Background(), id, a.ID); !errors.Is(err, ErrAthleteLocked) {
				t.Fatal(err)
			}
			break
		}
	}
}
func TestSettingsTwoDaysAndNumberingModes(t *testing.T) {
	for _, mode := range []string{"single_court", "split_halves", "balanced"} {
		t.Run(mode, func(t *testing.T) {
			s, r, id := newFixture(t)
			settings := *r.value.Settings
			settings.Schedule.Mode = mode
			settings.Schedule.SingleCourt = 3
			settings.Schedule.DayAssignment = "alternating"
			settings.Schedule.Days = append(settings.Schedule.Days, EventDay{Day: 2, Date: "2026-10-02", Label: "روز دوم"})
			if _, err := s.UpdateSettings(context.Background(), id, settings); err != nil {
				t.Fatal(err)
			}
			for _, cat := range []string{"-54", "-58", "-63", "-68"} {
				enter(t, s, id, cat, 8)
			}
			if _, err := s.DrawBracketForCategory(context.Background(), id, "-54", "ranking"); err != nil {
				t.Fatal(err)
			}
			beforeNumbering := r.writes
			if _, err := s.NumberMatches(context.Background(), id, true); !errors.Is(err, ErrMatchLocked) {
				t.Fatalf("previewed numbering before all categories were drawn: %v", err)
			}
			if _, err := s.NumberMatches(context.Background(), id, false); !errors.Is(err, ErrMatchLocked) {
				t.Fatalf("numbered before all categories were drawn: %v", err)
			}
			if r.writes != beforeNumbering {
				t.Fatal("incomplete draw wrote match numbers")
			}
			for _, match := range r.value.Matches {
				if match.MatchNumber != nil {
					t.Fatal("incomplete draw received a match number")
				}
			}
			for _, cat := range []string{"-58", "-63", "-68"} {
				if _, err := s.DrawBracketForCategory(context.Background(), id, cat, "ranking"); err != nil {
					t.Fatal(err)
				}
			}
			before := r.writes
			preview, err := s.NumberMatches(context.Background(), id, true)
			if err != nil {
				t.Fatal(err)
			}
			if r.writes != before {
				t.Fatal("preview wrote to repository")
			}
			result, err := s.NumberMatches(context.Background(), id, false)
			if err != nil {
				t.Fatal(err)
			}
			numbers := map[int]bool{}
			positions := map[[3]int]bool{}
			byID := map[string]Match{}
			for i, m := range result.Matches {
				if m.MatchNumber == nil || numbers[*m.MatchNumber] {
					t.Fatal("missing/duplicate number")
				}
				numbers[*m.MatchNumber] = true
				if preview.Matches[i].MatchNumber == nil || *m.MatchNumber != *preview.Matches[i].MatchNumber {
					t.Fatal("preview differs")
				}
				key := [3]int{m.Day, m.Court, m.Order}
				if positions[key] {
					t.Fatal("duplicate day/court/order")
				}
				positions[key] = true
				byID[m.ID] = m
				expected := 1
				if m.WeightCategory == "-58" || m.WeightCategory == "-68" {
					expected = 2
				}
				if m.Day != expected {
					t.Fatal("wrong day")
				}
				if mode == "single_court" && m.Court != 3 {
					t.Fatal("single court violated")
				}
			}
			for _, m := range result.Matches {
				if m.NextMatchID != nil {
					next := byID[*m.NextMatchID]
					if *m.MatchNumber >= *next.MatchNumber {
						t.Fatal("numbering reversed bracket")
					}
					if mode == "split_halves" && next.Side != MatchSideFinal && m.Court != next.Court {
						t.Fatal("athlete switches courts before final")
					}
				}
			}
			if mode == "balanced" {
				loads := map[int][]int{}
				for _, match := range result.Matches {
					if !lifecycleIsBye(&match) {
						if loads[match.Day] == nil {
							loads[match.Day] = make([]int, result.Courts+1)
						}
						loads[match.Day][match.Court]++
					}
				}
				for day, dayLoads := range loads {
					minimum, maximum := dayLoads[1], dayLoads[1]
					for court := 2; court <= result.Courts; court++ {
						minimum = min(minimum, dayLoads[court])
						maximum = max(maximum, dayLoads[court])
					}
					if maximum-minimum > 1 {
						t.Fatalf("balanced mode produced uneven court loads on day %d: %v", day, dayLoads[1:])
					}
				}
			}
		})
	}
}

func TestBalancedCourtsWithFivePopulatedCadetCategories(t *testing.T) {
	r := &memoryRepository{profiles: map[string]*AthleteProfile{}}
	s := NewTournamentService(r)
	event, err := s.Create(context.Background(), CreateTournamentInput{Name: "نوجوانان", Date: "2026-10-05", Courts: 4, Gender: GenderMale, AgeCategory: AgeCategoryNojavanan, Format: TournamentFormatGrandPrix})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{"-45": 21, "-51": 22, "-59": 21, "-68": 21, "-78": 16}
	weights := map[string]float64{"-45": 45, "-51": 50, "-59": 58, "-68": 67, "-78": 77}
	for category, count := range counts {
		for i := 0; i < count; i++ {
			athlete, addErr := s.AddAthlete(context.Background(), event.ID, AddAthleteInput{Name: fmt.Sprintf("%s-%d", category, i), Club: fmt.Sprintf("club-%d", i%12), WeightCategory: category})
			if addErr != nil {
				t.Fatal(addErr)
			}
			if _, weighErr := s.RecordWeighIn(context.Background(), event.ID, athlete.ID, RecordWeighInInput{WeightKg: weights[category]}); weighErr != nil {
				t.Fatal(weighErr)
			}
		}
	}
	drawn, err := s.DrawBracket(context.Background(), event.ID, "random")
	if err != nil {
		t.Fatal(err)
	}
	loads := make([]int, drawn.Courts+1)
	for _, match := range drawn.Matches {
		if !lifecycleIsBye(&match) {
			loads[match.Court]++
		}
	}
	minimum, maximum := loads[1], loads[1]
	for court := 2; court <= drawn.Courts; court++ {
		minimum = min(minimum, loads[court])
		maximum = max(maximum, loads[court])
	}
	if maximum-minimum > 1 {
		t.Fatalf("expected nearly equal court loads, got %v", loads[1:])
	}
}
func TestNumberingSettingsAndLocks(t *testing.T) {
	s, r, id := newFixture(t)
	enter(t, s, id, "-54", 4)
	if _, err := s.DrawBracket(context.Background(), id, "ranking"); err != nil {
		t.Fatal(err)
	}
	settings := *r.value.Settings
	settings.Numbering.StartAt = 101
	settings.Numbering.Scope = "court_day"
	if _, err := s.UpdateSettings(context.Background(), id, settings); err != nil {
		t.Fatal(err)
	}
	result, err := s.NumberMatches(context.Background(), id, false)
	if err != nil {
		t.Fatal(err)
	}
	if *result.Matches[0].MatchNumber != 101 {
		t.Fatal("start number ignored")
	}
	match := result.Matches[0]
	if _, err = s.StartMatch(context.Background(), id, match.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.NumberMatches(context.Background(), id, false); !errors.Is(err, ErrMatchLocked) {
		t.Fatal("renumbered ongoing competition")
	}
	settings.Numbering.StartAt = 1
	if _, err = s.UpdateSettings(context.Background(), id, settings); !errors.Is(err, ErrMatchLocked) {
		t.Fatal("changed live schedule")
	}
}

func TestHoguSizesRestrictCourtAssignmentAndNumbering(t *testing.T) {
	s, r, id := newFixture(t)
	enter(t, s, id, "-54", 4)
	if _, err := s.DrawBracket(context.Background(), id, "ranking"); err != nil {
		t.Fatal(err)
	}
	settings := *r.value.Settings
	settings.Hogu = HoguSettings{
		Enabled:       true,
		CourtSizes:    map[int][]string{1: {"1"}, 2: {"2"}, 3: {"1"}, 4: {"1"}},
		CategorySizes: map[string]string{"-54": "2"},
	}
	if _, err := s.UpdateSettings(context.Background(), id, settings); err != nil {
		t.Fatal(err)
	}
	preview, err := s.NumberMatches(context.Background(), id, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range preview.Matches {
		if !lifecycleIsBye(&match) && match.Court != 2 {
			t.Fatalf("size-2 category assigned to court %d", match.Court)
		}
	}
	settings = *r.value.Settings
	settings.Hogu.CourtSizes = map[int][]string{1: {"1"}, 2: {"1"}, 3: {"1"}, 4: {"1"}}
	if _, err := s.UpdateSettings(context.Background(), id, settings); err == nil {
		t.Fatal("expected rejection when no court carries the required size")
	}
}
func TestScoringRejectsForgedWinnerAndPropagates(t *testing.T) {
	s, r, id := newFixture(t)
	enter(t, s, id, "-54", 4)
	draw, err := s.DrawBracket(context.Background(), id, "ranking")
	if err != nil {
		t.Fatal(err)
	}
	m := draw.Matches[0]
	if _, err = s.StartMatch(context.Background(), id, m.ID); err != nil {
		t.Fatal(err)
	}
	rounds := []MatchRound{{Number: 1, Blue: RoundScore{BodyKick: 2}}, {Number: 2, Blue: RoundScore{Punch: 1}}}
	in := MatchResultInput{WinType: WinTypePTF, BlueID: *m.Athlete1ID, RedID: *m.Athlete2ID, WinnerID: *m.Athlete2ID, Rounds: rounds}
	if _, err = s.RecordMatchResult(context.Background(), id, m.ID, in); err == nil {
		t.Fatal("forged winner accepted")
	}
	in.WinnerID = ""
	result, err := s.RecordMatchResult(context.Background(), id, m.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if result.WinnerID == nil || *result.WinnerID != in.BlueID {
		t.Fatal("wrong calculated winner")
	}
	if _, err = s.RecordMatchResult(context.Background(), id, m.ID, in); !errors.Is(err, ErrResultAlreadyExists) {
		t.Fatal(err)
	}
	next := r.value.Matches[len(r.value.Matches)-1]
	if next.Athlete1ID == nil || *next.Athlete1ID != in.BlueID {
		t.Fatal("winner not advanced")
	}
	if _, err = s.ClearMatchResult(context.Background(), id, m.ID); err != nil {
		t.Fatal(err)
	}
	if r.value.Matches[len(r.value.Matches)-1].Athlete1ID != nil {
		t.Fatal("clearing winner did not clear feeder")
	}
}
func TestSettingsRejectInvalidDays(t *testing.T) {
	s, r, id := newFixture(t)
	settings := *r.value.Settings
	settings.Schedule.Days = append(settings.Schedule.Days, EventDay{Day: 2, Date: "2026-09-30"})
	if _, err := s.UpdateSettings(context.Background(), id, settings); err == nil {
		t.Fatal("accepted reversed dates")
	}
	settings = DefaultSettings(time.Now())
	if _, err := s.UpdateSettings(context.Background(), id, settings); err == nil {
		t.Fatal("accepted first day unrelated to event")
	}
}

func TestTeamSeparationKeepsTopSeedsAndByeRecipients(t *testing.T) {
	athletes := []TournamentAthlete{}
	for i := 1; i <= 8; i++ {
		rank := i
		club := fmt.Sprint(min(i, 9-i))
		athletes = append(athletes, TournamentAthlete{ID: fmt.Sprint(i), Club: club, Ranking: &rank})
	}
	matches, _, err := buildCategoryBracket(athletes, "-54", "ranking", 2, map[int]int{})
	if err != nil {
		t.Fatal(err)
	}
	clubs := map[string]string{}
	for _, a := range athletes {
		clubs[a.ID] = a.Club
	}
	for _, m := range matches {
		if m.Round == 1 && clubs[*m.Athlete1ID] == clubs[*m.Athlete2ID] {
			t.Fatal("avoidable first-round club clash")
		}
	}
	for position, expected := range map[int]string{0: "1", 3: "4", 4: "3", 7: "2"} {
		m := matches[position/2]
		actual := m.Athlete1ID
		if position%2 == 1 {
			actual = m.Athlete2ID
		}
		if actual == nil || *actual != expected {
			t.Fatal("top seed moved")
		}
	}
	matches, _, err = buildCategoryBracket(athletes[:5], "-54", "ranking", 2, map[int]int{})
	if err != nil {
		t.Fatal(err)
	}
	byes := map[string]bool{}
	for _, m := range matches {
		if lifecycleIsBye(&m) {
			byes[*m.WinnerID] = true
		}
	}
	if !reflect.DeepEqual(byes, map[string]bool{"1": true, "2": true, "3": true}) {
		t.Fatalf("bye recipients changed: %v", byes)
	}
}

func TestPendingWeighInBlocksDraw(t *testing.T) {
	s, _, id := newFixture(t)
	enter(t, s, id, "-54", 2)
	if _, err := s.AddAthlete(context.Background(), id, AddAthleteInput{Name: "Pending", WeightCategory: "-54"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DrawBracketForCategory(context.Background(), id, "-54", "ranking"); !errors.Is(err, ErrWeighInNotPassed) {
		t.Fatal("partial category draw accepted", err)
	}
}

func TestNumberingScopesAcrossDays(t *testing.T) {
	for _, scope := range []string{"tournament", "day", "court_day"} {
		t.Run(scope, func(t *testing.T) {
			s, r, id := newFixture(t)
			settings := *r.value.Settings
			settings.Schedule.DayAssignment = "alternating"
			settings.Schedule.Days = append(settings.Schedule.Days, EventDay{Day: 2, Date: "2026-10-02"})
			settings.Numbering.Scope = scope
			settings.Numbering.StartAt = 10
			settings.Numbering.Order = "category"
			if _, err := s.UpdateSettings(context.Background(), id, settings); err != nil {
				t.Fatal(err)
			}
			for _, cat := range []string{"-54", "-58", "-63", "-68"} {
				enter(t, s, id, cat, 4)
			}
			if _, err := s.DrawBracket(context.Background(), id, "ranking"); err != nil {
				t.Fatal(err)
			}
			got, err := s.NumberMatches(context.Background(), id, false)
			if err != nil {
				t.Fatal(err)
			}
			minima := map[string]int{}
			seen := map[string]bool{}
			for _, m := range got.Matches {
				key := "all"
				if scope == "day" {
					key = fmt.Sprint(m.Day)
				}
				if scope == "court_day" {
					key = fmt.Sprintf("%d/%d", m.Day, m.Court)
				}
				number := *m.MatchNumber
				k := fmt.Sprintf("%s/%d", key, number)
				if seen[k] {
					t.Fatal("duplicate scoped number")
				}
				seen[k] = true
				if minima[key] == 0 || number < minima[key] {
					minima[key] = number
				}
			}
			for _, minimum := range minima {
				if minimum != 10 {
					t.Fatal("scope did not restart at configured start")
				}
			}
		})
	}
}

func TestConfigurableScoringAndFrozenRules(t *testing.T) {
	s, r, id := newFixture(t)
	settings := *r.value.Settings
	settings.Scoring.RoundsToWin = 1
	settings.Scoring.MaxRounds = 1
	settings.Scoring.PointGap = 8
	if _, err := s.UpdateSettings(context.Background(), id, settings); err != nil {
		t.Fatal(err)
	}
	enter(t, s, id, "-54", 2)
	got, err := s.DrawBracket(context.Background(), id, "ranking")
	if err != nil {
		t.Fatal(err)
	}
	m := got.Matches[0]
	if _, err = s.StartMatch(context.Background(), id, m.ID); err != nil {
		t.Fatal(err)
	}
	result, err := s.RecordMatchResult(context.Background(), id, m.ID, MatchResultInput{WinType: WinTypePTF, BlueID: *m.Athlete1ID, RedID: *m.Athlete2ID, Rounds: []MatchRound{{Number: 1, Blue: RoundScore{BodyKick: 4}}}})
	if err != nil || result.Result.WinType != WinTypePTG {
		t.Fatalf("custom rules not applied: %+v %v", result, err)
	}
	settings.Scoring.PointGap = 15
	if _, err = s.UpdateSettings(context.Background(), id, settings); !errors.Is(err, ErrMatchLocked) {
		t.Fatal("rules changed after results")
	}
}

func TestBalancedLoadsAndFixedFinal(t *testing.T) {
	s, r, id := newFixture(t)
	for _, cat := range []string{"-54", "-58", "-63", "-68"} {
		enter(t, s, id, cat, 8)
	}
	result, err := s.DrawBracket(context.Background(), id, "ranking")
	if err != nil {
		t.Fatal(err)
	}
	loads := map[int]int{}
	for _, m := range result.Matches {
		if !lifecycleIsBye(&m) {
			loads[m.Court]++
		}
	}
	if !reflect.DeepEqual(loads, map[int]int{1: 7, 2: 7, 3: 7, 4: 7}) {
		t.Fatalf("unbalanced equal brackets: %v", loads)
	}
	settings := *r.value.Settings
	settings.Schedule.Mode = "split_halves"
	settings.Schedule.FinalCourtPolicy = "fixed"
	settings.Schedule.FinalCourt = 4
	result, err = s.UpdateSettings(context.Background(), id, settings)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range result.Matches {
		if m.Side == MatchSideFinal && m.Court != 4 {
			t.Fatal("fixed final court ignored")
		}
	}
}
