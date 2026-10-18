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
