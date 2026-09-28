package tournaments

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"testing"
)

type parityMatch struct {
	A, B, Winner *string
	Round        int
	Side         MatchSide
	Bye          bool
	Index        int
	Next         int
	Slot         *NextSlot
}

func TestActualDesktopFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/desktop-parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Categories map[AgeCategory]map[Gender][]string
		Weights    []struct {
			Category      string
			Weight        float64
			OK, Tolerance bool
		}
		Brackets []struct {
			Count   int
			Matches []parityMatch
		}
		Rounds []struct {
			Round   MatchRound
			Outcome struct {
				Winner  *Corner
				EndedBy *RoundEndReason
			}
		}
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(weightCategories, fixture.Categories) {
		t.Fatal("age/gender/weight definitions differ from desktop")
	}
	for _, test := range fixture.Weights {
		got, err := evaluateWeighInWeight(test.Category, test.Weight, true, WeightOptions(AgeCategoryBozorgsalan, GenderMale))
		if err != nil || got.ok != test.OK || got.usedTolerance != test.Tolerance {
			t.Fatalf("weight parity %+v: %+v %v", test, got, err)
		}
	}
	for _, test := range fixture.Brackets {
		athletes := []TournamentAthlete{}
		for i := 1; i <= test.Count; i++ {
			rank := i
			athletes = append(athletes, TournamentAthlete{ID: strconv.Itoa(i), Club: strconv.Itoa(i), Ranking: &rank})
		}
		got, _, err := buildCategoryBracket(athletes, "-54", "ranking", 1, map[int]int{})
		if err != nil {
			t.Fatal(err)
		}
		indices := map[string]int{}
		for i, m := range got {
			indices[m.ID] = i
		}
		normalized := []parityMatch{}
		for _, m := range got {
			next := -1
			if m.NextMatchID != nil {
				next = indices[*m.NextMatchID]
			}
			normalized = append(normalized, parityMatch{A: m.Athlete1ID, B: m.Athlete2ID, Winner: m.WinnerID, Round: m.Round, Side: m.Side, Bye: lifecycleIsBye(&m), Index: *m.BracketIndex, Next: next, Slot: m.NextSlot})
		}
		if !reflect.DeepEqual(normalized, test.Matches) {
			a, _ := json.Marshal(normalized)
			b, _ := json.Marshal(test.Matches)
			t.Fatalf("%d-entrant bracket differs\nGo: %s\nTS: %s", test.Count, a, b)
		}
	}
	for _, test := range fixture.Rounds {
		winner, reason := resolveRound(test.Round)
		expectedWinner := Corner("")
		expectedReason := RoundEndReason("")
		if test.Outcome.Winner != nil {
			expectedWinner = *test.Outcome.Winner
		}
		if test.Outcome.EndedBy != nil {
			expectedReason = *test.Outcome.EndedBy
		}
		if winner != expectedWinner || reason != expectedReason {
			t.Fatal("round scoring differs from desktop")
		}
	}
}
