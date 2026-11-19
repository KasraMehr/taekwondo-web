package tournaments

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestSavedCourtSettingsOverrideOldManualQueue(t *testing.T) {
	for _, mode := range []string{"single_court", "split_halves", "balanced"} {
		t.Run(mode, func(t *testing.T) {
			s, r, id := newFixture(t)
			settings := settingsFor(r.value)
			settings.Schedule.Mode = "single_court"
			if mode == "single_court" {
				settings.Schedule.Mode = "balanced"
			}
			if _, err := s.UpdateSettings(context.Background(), id, settings); err != nil {
				t.Fatal(err)
			}
			enter(t, s, id, "-54", 16)
			if _, err := s.DrawBracket(context.Background(), id, "ranking"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.NumberMatches(context.Background(), id, false); err != nil {
				t.Fatal(err)
			}
			// An earlier manual adjustment must not silently override new preferences.
			now := time.Now()
			for i, m := range r.value.Matches {
				if !lifecycleIsBye(&m) {
					r.value.Matches[i].MovedAt = &now
					break
				}
			}
			settings = settingsFor(r.value)
			settings.Schedule.Mode = mode
			settings.Schedule.SingleCourt = 3
			settings.Schedule.FinalCourtPolicy = "fixed"
			settings.Schedule.FinalCourt = 4
			settings.Numbering.StartAt = 41
			settings.Numbering.Scope = "court_day"
			updated, err := s.UpdateSettings(context.Background(), id, settings)
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := s.GetByID(context.Background(), id)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(updated, loaded) || !reflect.DeepEqual(*loaded.Settings, settings) {
				t.Fatal("settings/schedule did not survive reload")
			}
			courts := map[int]bool{}
			for _, m := range loaded.Matches {
				if lifecycleIsBye(&m) {
					continue
				}
				if m.MatchNumber == nil || *m.MatchNumber < 41 {
					t.Fatal("settings save left a match unnumbered")
				}
				if m.MovedAt != nil {
					t.Fatal("old manual pin survived explicit new layout")
				}
				if mode == "single_court" && m.Court != 3 {
					t.Fatal("single court setting ignored")
				}
				if mode != "single_court" && m.NextMatchID == nil && m.Court != 4 {
					t.Fatal("final court setting ignored")
				}
				if m.NextMatchID != nil {
					courts[m.Court] = true
				}
			}
			if mode == "split_halves" && len(courts) != 2 {
				t.Fatalf("expected two halves, got %v", courts)
			}
			if mode == "balanced" && len(courts) != 4 {
				t.Fatalf("balanced ignored available courts: %v", courts)
			}
			assertTimeline(t, loaded)
			preview, err := s.NumberMatches(context.Background(), id, true)
			if err != nil {
				t.Fatal(err)
			}
			applied, err := s.NumberMatches(context.Background(), id, false)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(preview.Matches, applied.Matches) {
				t.Fatal("preview differs from saved numbering")
			}
		})
	}
}

func TestOneDaySettingsClearAlternatingSecondDay(t *testing.T) {
	s, r, id := newFixture(t)
	settings := settingsFor(r.value)
	settings.Schedule.Days = append(settings.Schedule.Days, EventDay{Day: 2, Date: "2026-10-02", Label: "day 2"})
	settings.Schedule.DayAssignment = "alternating"
	if _, err := s.UpdateSettings(context.Background(), id, settings); err != nil {
		t.Fatal(err)
	}
	settings = settingsFor(r.value)
	settings.Schedule.Days = settings.Schedule.Days[:1]
	updated, err := s.UpdateSettings(context.Background(), id, settings)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Settings.Schedule.DayAssignment != "manual" || len(updated.Settings.Schedule.CategoryDays) != 0 {
		t.Fatal("hidden second-day assignments survived")
	}
}

func TestRebalanceSavesNumbersWithoutSecondAction(t *testing.T) {
	s, r, id := newFixture(t)
	enter(t, s, id, "-54", 8)
	if _, err := s.DrawBracket(context.Background(), id, "ranking"); err != nil {
		t.Fatal(err)
	}
	result, err := s.RebalanceCourts(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range result.Matches {
		if !lifecycleIsBye(&m) && m.MatchNumber == nil {
			t.Fatal("rebalance left unnumbered games")
		}
	}
	if !reflect.DeepEqual(result.Matches, r.value.Matches) {
		t.Fatal("rebalance not persisted")
	}
}

func TestHoguSettingsReassignOldManualCourt(t *testing.T) {
	s, r, id := newFixture(t)
	enter(t, s, id, "-54", 8)
	if _, err := s.DrawBracket(context.Background(), id, "ranking"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.NumberMatches(context.Background(), id, false); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for i, m := range r.value.Matches {
		if !lifecycleIsBye(&m) {
			r.value.Matches[i].MovedAt = &now
		}
	}
	settings := settingsFor(r.value)
	settings.Hogu = HoguSettings{Enabled: true, CategorySizes: map[string]string{"-54": "2"}, CourtSizes: map[int][]string{1: {"1"}, 2: {"1"}, 3: {"2"}, 4: {"4"}}}
	updated, err := s.UpdateSettings(context.Background(), id, settings)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range updated.Matches {
		if !lifecycleIsBye(&m) && (m.Court != 3 || m.MatchNumber == nil) {
			t.Fatal("new hogu configuration was ignored")
		}
	}
}

func TestTwoDayWeightParityDirections(t *testing.T) {
	for _, mode := range []string{"alternating", "alternating_reverse"} {
		t.Run(mode, func(t *testing.T) {
			s, r, id := newFixture(t)
			enter(t, s, id, "-54", 4)
			enter(t, s, id, "-58", 4)
			if _, err := s.DrawBracket(context.Background(), id, "ranking"); err != nil {
				t.Fatal(err)
			}
			settings := settingsFor(r.value)
			settings.Schedule.Days = append(settings.Schedule.Days, EventDay{Day: 2, Date: "2026-10-02", Label: "day 2"})
			settings.Schedule.DayAssignment = mode
			updated, err := s.UpdateSettings(context.Background(), id, settings)
			if err != nil {
				t.Fatal(err)
			}
			for i, category := range WeightOptions(updated.AgeCategory, updated.Gender) {
				expected := i%2 + 1
				if mode == "alternating_reverse" {
					expected = 3 - expected
				}
				if updated.Settings.Schedule.CategoryDays[category] != expected {
					t.Fatalf("%s assigned to wrong day", category)
				}
				for _, match := range updated.Matches {
					if match.WeightCategory == category && match.Day != expected {
						t.Fatalf("match %s day=%d want %d", match.ID, match.Day, expected)
					}
				}
			}
			loaded, err := s.GetByID(context.Background(), id)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(loaded.Settings.Schedule, updated.Settings.Schedule) {
				t.Fatal("day selection not persisted")
			}
			settings = settingsFor(loaded)
			settings.Schedule.Days = settings.Schedule.Days[:1]
			oneDay, err := s.UpdateSettings(context.Background(), id, settings)
			if err != nil {
				t.Fatal(err)
			}
			for _, match := range oneDay.Matches {
				if match.Day != 1 {
					t.Fatal("second-day match survived one-day conversion")
				}
			}
		})
	}
}

func TestDelayedSecondDayWeighInDrawAndNumberingPreservesDayOne(t *testing.T) {
	ctx := context.Background()
	s, r, id := newFixture(t)
	settings := settingsFor(r.value)
	settings.Schedule.Days = append(settings.Schedule.Days, EventDay{Day: 2, Date: "2026-10-02", Label: "روز دوم"})
	settings.Schedule.DayAssignment = "alternating"
	settings.Numbering.Scope = "tournament"
	if _, err := s.UpdateSettings(ctx, id, settings); err != nil {
		t.Fatal(err)
	}
	enter(t, s, id, "-54", 4)
	dayTwoAthletes := []string{}
	for _, name := range []string{"day-two-a", "day-two-b", "day-two-c", "day-two-d"} {
		athlete, err := s.AddAthlete(ctx, id, AddAthleteInput{Name: name, Club: "day two", WeightCategory: "-58"})
		if err != nil {
			t.Fatal(err)
		}
		dayTwoAthletes = append(dayTwoAthletes, athlete.ID)
	}
	if _, err := s.DrawBracketForCategory(ctx, id, "-54", "ranking"); err != nil {
		t.Fatal(err)
	}
	readiness := CheckDrawReadiness(r.value)
	if !readiness.Ready || !reflect.DeepEqual(readiness.ReadyDays, []int{1}) {
		t.Fatalf("day one should be independently ready: %#v", readiness)
	}
	if _, err := s.NumberMatches(ctx, id, false); err != nil {
		t.Fatal(err)
	}
	var firstMatch string
	for _, match := range r.value.Matches {
		if match.Day == 1 && !lifecycleIsBye(&match) && match.Athlete1ID != nil && match.Athlete2ID != nil {
			firstMatch = match.ID
			break
		}
	}
	if firstMatch == "" {
		t.Fatal("day one has no startable match")
	}
	if _, err := s.StartMatch(ctx, id, firstMatch); err != nil {
		t.Fatal(err)
	}
	dayOneBefore := []Match{}
	for _, match := range r.value.Matches {
		if match.Day == 1 {
			dayOneBefore = append(dayOneBefore, match)
		}
	}
	for _, athleteID := range dayTwoAthletes {
		if _, err := s.RecordWeighIn(ctx, id, athleteID, RecordWeighInInput{WeightKg: 58}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.DrawBracketForCategory(ctx, id, "-58", "ranking"); err != nil {
		t.Fatal("day two draw must remain available after day one starts:", err)
	}
	if _, err := s.NumberMatches(ctx, id, false); err != nil {
		t.Fatal("day two numbering must remain available after day one starts:", err)
	}
	dayOneAfter := []Match{}
	dayTwoNumbered := false
	dayOneMaximum := 0
	dayTwoMinimum := int(^uint(0) >> 1)
	for _, match := range r.value.Matches {
		if match.Day == 1 {
			dayOneAfter = append(dayOneAfter, match)
			if match.MatchNumber != nil && *match.MatchNumber > dayOneMaximum {
				dayOneMaximum = *match.MatchNumber
			}
		} else if match.Day == 2 && !lifecycleIsBye(&match) {
			if match.MatchNumber == nil {
				t.Fatal("day two match was not numbered")
			}
			dayTwoNumbered = true
			if *match.MatchNumber < dayTwoMinimum {
				dayTwoMinimum = *match.MatchNumber
			}
		}
	}
	if !reflect.DeepEqual(dayOneBefore, dayOneAfter) {
		t.Fatal("preparing day two changed day-one matches")
	}
	if !dayTwoNumbered || dayTwoMinimum <= dayOneMaximum {
		t.Fatal("tournament-wide numbers did not continue on day two")
	}
}
