package tournaments

import (
	"fmt"
	"sort"
)

func hasManualSchedule(t *Tournament) bool {
	for _, m := range t.Matches {
		if m.MovedAt != nil && !lifecycleIsBye(&m) {
			return true
		}
	}
	return false
}

// A shared timeline is essential: order 10 on A is not a time on B.
// Preserve mode keeps the entire approved queue, including manual edits.
// Automatic mode fills available court time with independent bouts first.
func buildScheduleTimeline(t *Tournament, preserve bool) error {
	settings := settingsFor(t)
	duration, rest := settings.Schedule.MatchMinutes, settings.Schedule.RestMinutes
	if duration == 0 {
		duration, rest = 5, 10
	}
	byID := map[string]int{}
	feeders := map[string][]int{}
	maxRound := map[string]int{}
	weightRank := map[string]int{}
	for rank, category := range WeightOptions(t.AgeCategory, t.Gender) {
		weightRank[category] = rank
	}
	indices := []int{}
	for i := range t.Matches {
		m := &t.Matches[i]
		if _, exists := byID[m.ID]; exists {
			return fmt.Errorf("%w: duplicate match id", ErrInvalidMatchPosition)
		}
		byID[m.ID] = i
		maxRound[m.WeightCategory] = max(maxRound[m.WeightCategory], m.Round)
		m.ScheduledStartMinute, m.ScheduledEndMinute, m.ScheduledWaitMinutes = nil, nil, 0
		if !lifecycleIsBye(m) {
			if m.Court < 1 || m.Court > t.Courts {
				return ErrInvalidCourt
			}
			if settings.Hogu.Enabled {
				allowed := false
				for _, size := range settings.Hogu.CourtSizes[m.Court] {
					if size == settings.Hogu.CategorySizes[m.WeightCategory] {
						allowed = true
					}
				}
				if !allowed {
					return fmt.Errorf("%w: هوگوی زمین برای وزن %s مناسب نیست", ErrInvalidCourt, m.WeightCategory)
				}
			}
			indices = append(indices, i)
		}
	}
	for i, m := range t.Matches {
		if m.NextMatchID != nil {
			if _, exists := byID[*m.NextMatchID]; !exists {
				return fmt.Errorf("%w: missing next match", ErrInvalidMatchPosition)
			}
			feeders[*m.NextMatchID] = append(feeders[*m.NextMatchID], i)
		}
	}
	// Flatten bye chains rather than treating an unplayed feeder as completed.
	var realFeeders func(int, map[int]bool) ([]int, error)
	realFeeders = func(i int, path map[int]bool) ([]int, error) {
		if path[i] {
			return nil, fmt.Errorf("%w: cyclic bracket", ErrInvalidMatchPosition)
		}
		path[i] = true
		defer delete(path, i)
		result := []int{}
		for _, f := range feeders[t.Matches[i].ID] {
			if lifecycleIsBye(&t.Matches[f]) {
				ancestors, err := realFeeders(f, path)
				if err != nil {
					return nil, err
				}
				result = append(result, ancestors...)
			} else {
				result = append(result, f)
			}
		}
		return result, nil
	}
	deps := map[int][]int{}
	for _, i := range indices {
		var err error
		deps[i], err = realFeeders(i, map[int]bool{})
		if err != nil {
			return err
		}
	}
	sort.SliceStable(indices, func(a, b int) bool {
		x, y := t.Matches[indices[a]], t.Matches[indices[b]]
		if x.Day != y.Day {
			return x.Day < y.Day
		}
		if x.Order != y.Order {
			return x.Order < y.Order
		}
		if x.Court != y.Court {
			return x.Court < y.Court
		}
		return x.ID < y.ID
	})
	previous := map[int]int{}
	last := map[[2]int]int{}
	if preserve {
		for _, i := range indices {
			m := t.Matches[i]
			key := [2]int{m.Day, m.Court}
			if p, ok := last[key]; ok {
				if t.Matches[p].Order == m.Order {
					return fmt.Errorf("%w: duplicate court position", ErrInvalidMatchPosition)
				}
				previous[i] = p
			}
			last[key] = i
		}
	}
	done := map[int]bool{}
	available := map[[2]int]int{}
	nextOrder := map[[2]int]int{}
	for len(done) < len(indices) {
		selected, start, selectedCourt := -1, 0, 0
		for _, i := range indices {
			if done[i] {
				continue
			}
			m := t.Matches[i]
			if p, ok := previous[i]; ok && !done[p] {
				continue
			}
			earliest := 0
			ready := true
			for _, f := range deps[i] {
				feeder := t.Matches[f]
				if feeder.Day > m.Day {
					return fmt.Errorf("%w: feeder is on a later day", ErrInvalidMatchPosition)
				}
				if !done[f] {
					ready = false
					break
				}
				if feeder.Day == m.Day {
					earliest = max(earliest, *feeder.ScheduledEndMinute+rest)
				}
			}
			if !ready {
				continue
			}
			courts := []int{m.Court}
			if !preserve && settings.Schedule.Mode == "balanced" && !(settings.Schedule.FinalCourtPolicy == "fixed" && m.Round == maxRound[m.WeightCategory]) {
				courts = nil
				for court := 1; court <= t.Courts; court++ {
					if !settings.Hogu.Enabled || containsString(settings.Hogu.CourtSizes[court], settings.Hogu.CategorySizes[m.WeightCategory]) {
						courts = append(courts, court)
					}
				}
			}
			for _, court := range courts {
				at := max(earliest, available[[2]int{m.Day, court}])
				better := selected < 0
				if !better {
					other := t.Matches[selected]
					stage, otherStage := maxRound[m.WeightCategory]-m.Round, maxRound[other.WeightCategory]-other.Round
					priority := stage > otherStage
					if stage == otherStage {
						if settings.Numbering.Order == "category" && m.WeightCategory != other.WeightCategory {
							priority = weightRank[m.WeightCategory] < weightRank[other.WeightCategory]
						} else if i == selected {
							priority = nextOrder[[2]int{m.Day, court}] < nextOrder[[2]int{m.Day, selectedCourt}]
						}
					}
					better = m.Day < other.Day || m.Day == other.Day && (at < start || at == start && priority)
				}
				if better {
					selected, start, selectedCourt = i, at, court
				}
			}
		}
		if selected < 0 {
			return fmt.Errorf("%w: ترتیب دستی با مسیر صعود بازیکنان تناقض دارد", ErrInvalidMatchPosition)
		}
		m := &t.Matches[selected]
		m.Court = selectedCourt
		key := [2]int{m.Day, m.Court}
		end := start + duration
		m.ScheduledStartMinute, m.ScheduledEndMinute = &start, &end
		m.ScheduledWaitMinutes = start - available[key]
		available[key] = end
		nextOrder[key]++
		if !preserve {
			m.Order = nextOrder[key]
		}
		done[selected] = true
	}
	t.CourtAssignment = map[string][]int{}
	for _, i := range indices {
		m := t.Matches[i]
		if !containsInt(t.CourtAssignment[m.WeightCategory], m.Court) {
			t.CourtAssignment[m.WeightCategory] = append(t.CourtAssignment[m.WeightCategory], m.Court)
		}
	}
	for _, courts := range t.CourtAssignment {
		sort.Ints(courts)
	}
	return nil
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
