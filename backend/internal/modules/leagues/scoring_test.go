package leagues

import (
	"testing"

	"backend/internal/modules/tournaments"
)

func ptr[T any](v T) *T { return &v }

func TestScoreTournamentUsesBracketGraphAndProducesStableKeys(t *testing.T) {
	status := tournaments.MatchStatusCompleted
	blue := tournaments.CornerBlue
	entries := []tournaments.TournamentAthlete{
		{ID: "a", Name: "A", Club: "One", WeightCategory: "-54", WeighIn: &tournaments.WeighIn{Status: tournaments.WeighInPassed}},
		{ID: "b", Name: "B", Club: "Two", WeightCategory: "-54", WeighIn: &tournaments.WeighIn{Status: tournaments.WeighInPassed}},
		{ID: "c", Name: "C", Club: "Three", WeightCategory: "-54", WeighIn: &tournaments.WeighIn{Status: tournaments.WeighInPassed}},
		{ID: "d", Name: "D", Club: "Four", WeightCategory: "-54", WeighIn: &tournaments.WeighIn{Status: tournaments.WeighInPassed}},
	}
	rounds20 := []tournaments.MatchRound{{Winner: &blue}, {Winner: &blue}}
	matches := []tournaments.Match{
		{ID: "s1", Athlete1ID: ptr("a"), Athlete2ID: ptr("b"), WinnerID: ptr("a"), Status: &status, WeightCategory: "-54", NextMatchID: ptr("f"), Result: &tournaments.MatchResult{WinnerCorner: &blue, Rounds: rounds20}},
		{ID: "s2", Athlete1ID: ptr("c"), Athlete2ID: ptr("d"), WinnerID: ptr("c"), Status: &status, WeightCategory: "-54", NextMatchID: ptr("f"), Result: &tournaments.MatchResult{WinnerCorner: &blue, Rounds: rounds20}},
		// Deliberately use round 1 for the final: scoring must follow NextMatchID.
		{ID: "f", Athlete1ID: ptr("a"), Athlete2ID: ptr("c"), WinnerID: ptr("a"), Status: &status, WeightCategory: "-54", Round: 1, Result: &tournaments.MatchResult{WinnerCorner: &blue, Rounds: rounds20}},
	}
	events, err := ScoreTournament(&tournaments.Tournament{Athletes: entries, Matches: matches}, DefaultScoring(), map[string]*string{})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]PointEvent{}
	for _, e := range events {
		if _, ok := seen[e.EventKey]; ok {
			t.Fatalf("duplicate event key %s", e.EventKey)
		}
		seen[e.EventKey] = e
	}
	if seen["medal:-54:gold"].EntryID != "a" || seen["medal:-54:silver"].EntryID != "c" {
		t.Fatalf("wrong final medals: %#v", seen)
	}
	if seen["medal:-54:bronze:s1"].EntryID != "b" || seen["medal:-54:bronze:s2"].EntryID != "d" {
		t.Fatalf("wrong bronze medals: %#v", seen)
	}
	if seen["match:s1:winner"].RoundDiff != 2 || seen["match:s1:loser"].RoundDiff != -2 {
		t.Fatal("round statistics were not derived")
	}
}

func TestScoreTournamentPublishesCompletedMatchesBeforeCategoryFinishes(t *testing.T) {
	pending, completed := tournaments.MatchStatusPending, tournaments.MatchStatusCompleted
	blue := tournaments.CornerBlue
	rounds20 := []tournaments.MatchRound{{Winner: &blue}, {Winner: &blue}}
	events, err := ScoreTournament(&tournaments.Tournament{
		Athletes: []tournaments.TournamentAthlete{
			{ID: "a", Name: "A", WeightCategory: "-54", WeighIn: &tournaments.WeighIn{Status: tournaments.WeighInPassed}},
			{ID: "b", Name: "B", WeightCategory: "-54", WeighIn: &tournaments.WeighIn{Status: tournaments.WeighInPassed}},
			{ID: "c", Name: "C", WeightCategory: "-54", WeighIn: &tournaments.WeighIn{Status: tournaments.WeighInPassed}},
		},
		Matches: []tournaments.Match{
			{ID: "semi", Athlete1ID: ptr("a"), Athlete2ID: ptr("b"), WinnerID: ptr("a"), Status: &completed, WeightCategory: "-54", NextMatchID: ptr("final"), Result: &tournaments.MatchResult{WinnerCorner: &blue, Rounds: rounds20}},
			{ID: "final", Athlete1ID: ptr("a"), Athlete2ID: ptr("c"), Status: &pending, WeightCategory: "-54"},
		},
	}, DefaultScoring(), nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]PointEvent{}
	for _, event := range events {
		seen[event.EventKey] = event
	}
	if seen["match:semi:winner"].Points != DefaultScoring().WinPoint || seen["match:semi:loser"].Losses != 1 {
		t.Fatalf("completed match was not scored: %#v", seen)
	}
	for key := range seen {
		if len(key) >= 6 && key[:6] == "medal:" {
			t.Fatalf("incomplete category received a medal event: %s", key)
		}
	}
}

func TestRankTieBreakOrder(t *testing.T) {
	rows := []Standing{{Name: "round", TotalPoints: 10, Gold: 1, RoundDiff: 4}, {Name: "gold", TotalPoints: 10, Gold: 2}, {Name: "points", TotalPoints: 11}}
	Rank(rows)
	if rows[0].Name != "points" || rows[1].Name != "gold" || rows[2].Name != "round" {
		t.Fatalf("unexpected order %#v", rows)
	}
}
