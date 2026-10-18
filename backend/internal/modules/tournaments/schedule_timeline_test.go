package tournaments

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func timelineEvent(matches ...Match) *Tournament {
	s := DefaultSettings(time.Now())
	s.Schedule.Mode = "split_halves"
	return &Tournament{Courts: 2, Settings: &s, Matches: matches}
}

func timelineMatch(id string, court, order int, next string) Match {
	m := Match{ID: id, Court: court, Order: order, Day: 1, Round: 1, WeightCategory: "-54"}
	if next != "" {
		m.NextMatchID = &next
	}
	return m
}

func assertTimeline(t *testing.T, event *Tournament) {
	t.Helper()
	byID := map[string]Match{}
	for _, m := range event.Matches {
		byID[m.ID] = m
	}
	for _, m := range event.Matches {
		if lifecycleIsBye(&m) {
			continue
		}
		if m.ScheduledStartMinute == nil || m.ScheduledEndMinute == nil {
			t.Fatalf("missing time: %s", m.ID)
		}
		if m.NextMatchID != nil {
			n := byID[*m.NextMatchID]
			if !lifecycleIsBye(&n) && n.Day == m.Day && *n.ScheduledStartMinute < *m.ScheduledEndMinute+event.Settings.Schedule.RestMinutes {
				t.Fatalf("insufficient rest %s -> %s", m.ID, n.ID)
			}
		}
		for _, n := range event.Matches {
			if n.ID == m.ID || lifecycleIsBye(&n) || n.Court != m.Court || n.Day != m.Day {
				continue
			}
			if *m.ScheduledStartMinute < *n.ScheduledEndMinute && *n.ScheduledStartMinute < *m.ScheduledEndMinute {
				t.Fatalf("court overlap %s/%s", m.ID, n.ID)
			}
		}
	}
}

func TestTimelineCrossCourtRestAndFillerBouts(t *testing.T) {
	e := timelineEvent(timelineMatch("feeder", 1, 1, "final"), timelineMatch("final", 2, 1, ""), timelineMatch("filler1", 2, 2, ""), timelineMatch("filler2", 2, 3, ""), timelineMatch("filler3", 2, 4, ""))
	e.Matches[1].Round = 2
	if err := buildScheduleTimeline(e, false); err != nil {
		t.Fatal(err)
	}
	assertTimeline(t, e)
	if *e.Matches[1].ScheduledStartMinute != 15 || e.Matches[1].Order != 4 || e.Matches[1].ScheduledWaitMinutes != 0 {
		t.Fatalf("ready bouts did not fill rest: %+v", e.Matches[1])
	}
}

func TestTimelineReportsUnavoidableWait(t *testing.T) {
	e := timelineEvent(timelineMatch("first", 1, 1, "next"), timelineMatch("next", 2, 1, ""))
	if err := buildScheduleTimeline(e, false); err != nil {
		t.Fatal(err)
	}
	assertTimeline(t, e)
	if e.Matches[1].ScheduledWaitMinutes != 15 {
		t.Fatal("waiting was hidden")
	}
}

func TestTimelineManualCrossCourtCycleRejected(t *testing.T) {
	// A waits for B2, while B waits for A2. Each bracket alone is valid.
	e := timelineEvent(timelineMatch("a1", 1, 1, ""), timelineMatch("a2", 1, 2, "b1"), timelineMatch("b1", 2, 1, ""), timelineMatch("b2", 2, 2, "a1"))
	if err := buildScheduleTimeline(e, true); err == nil {
		t.Fatal("accepted cyclic court queues")
	}
}

func TestTimelineByeChainAndDays(t *testing.T) {
	e := timelineEvent(timelineMatch("first", 1, 1, "bye"), timelineMatch("bye", 0, 0, "next"), timelineMatch("next", 2, 1, ""))
	bye := true
	e.Matches[1].IsBye = &bye
	if err := buildScheduleTimeline(e, false); err != nil {
		t.Fatal(err)
	}
	if *e.Matches[2].ScheduledStartMinute != 15 {
		t.Fatal("bye chain discarded feeder rest")
	}
	e.Matches[2].Day = 2
	if err := buildScheduleTimeline(e, false); err != nil {
		t.Fatal(err)
	}
	if *e.Matches[2].ScheduledStartMinute != 0 {
		t.Fatal("rest leaked into next day")
	}
}

func TestTimelineBalancedUsesAvailableCompatibleCourt(t *testing.T) {
	e := timelineEvent(timelineMatch("a", 1, 1, ""), timelineMatch("b", 1, 2, ""), timelineMatch("c", 1, 3, ""), timelineMatch("d", 1, 4, ""))
	e.Settings.Schedule.Mode = "balanced"
	e.Settings.Hogu = HoguSettings{Enabled: true, CategorySizes: map[string]string{"-54": "2"}, CourtSizes: map[int][]string{1: {"2"}, 2: {"2"}}}
	if err := buildScheduleTimeline(e, false); err != nil {
		t.Fatal(err)
	}
	assertTimeline(t, e)
	loads := map[int]int{}
	for _, m := range e.Matches {
		loads[m.Court]++
		if m.ScheduledWaitMinutes != 0 {
			t.Fatal("idle court with ready work")
		}
	}
	if loads[1] != 2 || loads[2] != 2 {
		t.Fatalf("uneven load %v", loads)
	}
	e.Settings.Hogu.CourtSizes[2] = []string{"4"}
	// Replan normally assigns all bouts to compatible courts before scheduling.
	for i := range e.Matches {
		e.Matches[i].Court = 1
	}
	if err := buildScheduleTimeline(e, false); err != nil {
		t.Fatal(err)
	}
	for _, m := range e.Matches {
		if m.Court != 1 {
			t.Fatal("incompatible hogu")
		}
	}
}

func TestManualScheduleSavedAndPreservedByNumbering(t *testing.T) {
	s, r, id := newFixture(t)
	enter(t, s, id, "-54", 8)
	if _, err := s.DrawBracket(context.Background(), id, "ranking"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.NumberMatches(context.Background(), id, false); err != nil {
		t.Fatal(err)
	}
	first := []Match{}
	for _, m := range r.value.Matches {
		if m.Round == 1 && !lifecycleIsBye(&m) {
			first = append(first, m)
		}
	}
	if len(first) < 2 {
		t.Fatal("fixture missing bouts")
	}
	if _, err := s.SwapMatchPositions(context.Background(), id, first[0].ID, SwapMatchPositionsInput{OtherMatchID: first[1].ID}); err != nil {
		t.Fatal(err)
	}
	saved := copyTournament(r.value)
	preview, err := s.NumberMatches(context.Background(), id, true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.Matches, r.value.Matches) {
		t.Fatal("preview mutated persisted schedule")
	}
	result, err := s.NumberMatches(context.Background(), id, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.Matches, result.Matches) || !reflect.DeepEqual(preview.Matches, result.Matches) {
		t.Fatal("renumbering lost manual schedule or preview differs")
	}
	settings := *r.value.Settings
	settings.Schedule.RestMinutes = 12
	if _, err := s.UpdateSettings(context.Background(), id, settings); err != nil {
		t.Fatal(err)
	}
	for i, m := range r.value.Matches {
		before := saved.Matches[i]
		if m.Court != before.Court || m.Order != before.Order || !reflect.DeepEqual(m.MatchNumber, before.MatchNumber) || !reflect.DeepEqual(m.MovedAt, before.MovedAt) {
			t.Fatal("settings lost manual edits")
		}
	}
	assertTimeline(t, r.value)
}

func TestManualMoveInsertsIntoOccupiedPosition(t *testing.T) {
	s, r, id := newFixture(t)
	settings := *r.value.Settings
	settings.Schedule.Mode = "single_court"
	if _, err := s.UpdateSettings(context.Background(), id, settings); err != nil {
		t.Fatal(err)
	}
	enter(t, s, id, "-54", 8)
	if _, err := s.DrawBracket(context.Background(), id, "ranking"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.NumberMatches(context.Background(), id, false); err != nil {
		t.Fatal(err)
	}
	var moved Match
	for _, m := range r.value.Matches {
		if m.Round == 1 && m.Order > moved.Order {
			moved = m
		}
	}
	if _, err := s.MoveMatch(context.Background(), id, moved.ID, MoveMatchInput{Court: 1, Order: 1}); err != nil {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	for _, m := range r.value.Matches {
		if lifecycleIsBye(&m) {
			continue
		}
		if seen[m.Order] || m.MatchNumber == nil || *m.MatchNumber != m.Order {
			t.Fatal("inconsistent saved numbering")
		}
		seen[m.Order] = true
		if m.ID == moved.ID && (m.Order != 1 || m.MovedAt == nil) {
			t.Fatal("move not saved")
		}
	}
	assertTimeline(t, r.value)
}
