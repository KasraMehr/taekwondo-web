package tournaments

import (
	"context"
	"fmt"
	"sort"
	"time"
)

func hasStartedMatches(t *Tournament) bool {
	for _, m := range t.Matches {
		if !lifecycleIsBye(&m) && (lifecycleHasResult(&m) || lifecycleMatchStatus(&m) == MatchStatusOngoing) {
			return true
		}
	}
	return false
}
func clearMatchNumbers(t *Tournament) {
	for i := range t.Matches {
		t.Matches[i].MatchNumber = nil
		t.Matches[i].ScheduledStartMinute = nil
		t.Matches[i].ScheduledEndMinute = nil
		t.Matches[i].ScheduledWaitMinutes = 0
	}
}

type DrawReadiness struct {
	Ready               bool     `json:"ready"`
	PendingWeighIns     []string `json:"pendingWeighIns"`
	UndrawnCategories   []string `json:"undrawnCategories"`
	NoContestCategories []string `json:"noContestCategories"`
}

func CheckDrawReadiness(t *Tournament) DrawReadiness {
	result := DrawReadiness{PendingWeighIns: []string{}, UndrawnCategories: []string{}, NoContestCategories: []string{}}
	for _, cat := range WeightOptions(t.AgeCategory, t.Gender) {
		total, passed := 0, 0
		pending := false
		for _, a := range t.Athletes {
			if a.WeightCategory != cat {
				continue
			}
			total++
			if athletePassedWeighIn(a) {
				passed++
			} else if a.WeighIn == nil || a.WeighIn.Status != WeighInFailed {
				pending = true
			}
		}
		if total == 0 {
			continue
		}
		if pending {
			result.PendingWeighIns = append(result.PendingWeighIns, cat)
		}
		if passed >= 2 && !bracketExistsForCategory(t.Matches, cat) {
			result.UndrawnCategories = append(result.UndrawnCategories, cat)
		}
		if !pending && passed < 2 {
			result.NoContestCategories = append(result.NoContestCategories, cat)
		}
	}
	result.Ready = len(t.Matches) > 0 && len(result.PendingWeighIns) == 0 && len(result.UndrawnCategories) == 0
	return result
}
func allDrawsComplete(t *Tournament) bool { return CheckDrawReadiness(t).Ready }

// Numbering runs only after the entire event's draws are complete. Preview has no writes.
func (s *tournamentService) NumberMatches(ctx context.Context, id string, preview bool) (*Tournament, error) {
	return s.numberMatchesWithPriority(ctx, id, preview, nil, nil, nil)
}

// NumberMatchesWithPriority applies a validated AI suggested category priority.
// All bracket, rest, court and number constraints still use the normal scheduler.
func (s *tournamentService) NumberMatchesWithPriority(ctx context.Context, id string, preview bool, priority []string) (*Tournament, error) {
	return s.numberMatchesWithPriority(ctx, id, preview, priority, nil, nil)
}

type GuidedNumberingOptions struct {
	MatchMinutes int            `json:"matchMinutes"`
	RestMinutes  int            `json:"restMinutes"`
	StartAt      int            `json:"startAt"`
	LimitHogu    bool           `json:"limitHogu"`
	HoguPairs    map[string]int `json:"hoguPairs"`
}

func (s *tournamentService) NumberMatchesWithPlan(ctx context.Context, id string, preview bool, priority []string, courts map[string][]int, options *GuidedNumberingOptions) (*Tournament, error) {
	return s.numberMatchesWithPriority(ctx, id, preview, priority, courts, options)
}

func (s *tournamentService) numberMatchesWithPriority(ctx context.Context, id string, preview bool, priority []string, courts map[string][]int, options *GuidedNumberingOptions) (*Tournament, error) {
	t, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !allDrawsComplete(t) {
		return nil, fmt.Errorf("%w: finish all weigh-ins and category draws before numbering", ErrMatchLocked)
	}
	if hasStartedMatches(t) {
		return nil, fmt.Errorf("%w: numbering is locked after the first match starts", ErrMatchLocked)
	}
	if options != nil {
		if options.MatchMinutes < 1 || options.MatchMinutes > 60 || options.RestMinutes < 0 || options.RestMinutes > 120 || options.StartAt < 1 || options.StartAt > 1000000 {
			return nil, fmt.Errorf("%w: invalid guided numbering settings", ErrInvalidTournamentInput)
		}
		settings := settingsFor(t)
		settings.Schedule.MatchMinutes = options.MatchMinutes
		settings.Schedule.RestMinutes = options.RestMinutes
		settings.Numbering.StartAt = options.StartAt
		applyDefaultHoguSizes(t, &settings.Hogu)
		settings.Hogu.Enabled = options.LimitHogu
		if options.LimitHogu {
			settings.Hogu.PairCounts = map[string]int{}
			settings.Hogu.CourtSizes = map[int][]string{}
			for size, count := range options.HoguPairs {
				if !validHoguSize(size) || count < 0 || count > 100 {
					return nil, fmt.Errorf("%w: invalid hogu inventory", ErrInvalidTournamentInput)
				}
				settings.Hogu.PairCounts[size] = count
				if count > 0 {
					for court := 1; court <= t.Courts; court++ {
						settings.Hogu.CourtSizes[court] = append(settings.Hogu.CourtSizes[court], size)
					}
				}
			}
			for _, m := range t.Matches {
				if !lifecycleIsBye(&m) && settings.Hogu.PairCounts[settings.Hogu.CategorySizes[m.WeightCategory]] < 1 {
					return nil, fmt.Errorf("%w: hogu size %s is unavailable", ErrInvalidTournamentInput, settings.Hogu.CategorySizes[m.WeightCategory])
				}
			}
		} else {
			settings.Hogu.PairCounts = nil
		}
		t.Settings = &settings
	}
	if courts != nil {
		if hasManualSchedule(t) {
			return nil, fmt.Errorf("%w: guided numbering requires an automatic court schedule", ErrMatchLocked)
		}
		playable := map[string]bool{}
		for _, m := range t.Matches {
			if !lifecycleIsBye(&m) {
				playable[m.WeightCategory] = true
			}
		}
		if len(courts) != len(playable) {
			return nil, fmt.Errorf("%w: select courts for every playable category", ErrInvalidTournamentInput)
		}
		settings := settingsFor(t)
		for category, selected := range courts {
			if !playable[category] || len(selected) < 1 || len(selected) > 2 {
				return nil, fmt.Errorf("%w: invalid court selection", ErrInvalidTournamentInput)
			}
			seen := map[int]bool{}
			for _, court := range selected {
				if court < 1 || court > t.Courts || seen[court] {
					return nil, fmt.Errorf("%w: invalid court selection", ErrInvalidTournamentInput)
				}
				if settings.Hogu.Enabled && !containsString(settings.Hogu.CourtSizes[court], settings.Hogu.CategorySizes[category]) {
					return nil, fmt.Errorf("%w: selected court has incompatible hogu", ErrInvalidCourt)
				}
				seen[court] = true
			}
		}
	}
	if len(priority) > 0 {
		allowed := map[string]bool{}
		for _, category := range WeightOptions(t.AgeCategory, t.Gender) {
			allowed[category] = true
		}
		seen := map[string]bool{}
		for _, category := range priority {
			if !allowed[category] || seen[category] {
				return nil, fmt.Errorf("%w: invalid category priority", ErrInvalidTournamentInput)
			}
			seen[category] = true
		}
		if hasManualSchedule(t) {
			return nil, fmt.Errorf("%w: AI numbering requires an automatic court schedule", ErrMatchLocked)
		}
	}
	// Never redistribute an approved manual queue during ordinary numbering.
	if !hasManualSchedule(t) {
		clearMatchNumbers(t)
		if courts == nil {
			if err = applyCourtPlan(t); err != nil {
				return nil, err
			}
		} else {
			for category, selected := range courts {
				assignCategoryCourts(t.Matches, category, selected)
			}
			syncCategoryOrders(t)
		}
	}
	if err = buildScheduleTimelineWithPlan(t, hasManualSchedule(t), priority, courts); err != nil {
		return nil, err
	}
	needsNumbers := !hasManualSchedule(t)
	for _, m := range t.Matches {
		if !lifecycleIsBye(&m) && m.MatchNumber == nil {
			needsNumbers = true
		}
	}
	if needsNumbers {
		if err = assignMatchNumbers(t); err != nil {
			return nil, err
		}
	}
	if preview {
		return t, nil
	}
	t.UpdatedAt = time.Now().UTC()
	if err = s.repository.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Automatic scheduling uses shared minutes across all courts.
func scheduleRestAwareOrders(t *Tournament) error {
	return buildScheduleTimeline(t, hasManualSchedule(t))
}

func assignMatchNumbers(t *Tournament) error {
	if err := lifecycleValidateBracket(t); err != nil {
		return err
	}
	// Each court queue and each bracket feeder imposes an ordering dependency.
	indices := []int{}
	deps := map[string][]string{}
	done := map[string]bool{}
	for i, m := range t.Matches {
		if lifecycleIsBye(&m) {
			done[m.ID] = true
			continue
		}
		indices = append(indices, i)
	}
	sort.SliceStable(indices, func(i, j int) bool {
		a, b := t.Matches[indices[i]], t.Matches[indices[j]]
		if a.Day != b.Day {
			return a.Day < b.Day
		}
		if a.ScheduledStartMinute != nil && b.ScheduledStartMinute != nil && *a.ScheduledStartMinute != *b.ScheduledStartMinute {
			return *a.ScheduledStartMinute < *b.ScheduledStartMinute
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		if a.Court != b.Court {
			return a.Court < b.Court
		}
		return a.ID < b.ID
	})
	previous := map[[2]int]string{}
	for _, i := range indices {
		m := t.Matches[i]
		key := [2]int{m.Day, m.Court}
		if prev := previous[key]; prev != "" {
			deps[m.ID] = append(deps[m.ID], prev)
		}
		previous[key] = m.ID
	}
	for _, m := range t.Matches {
		if m.NextMatchID != nil {
			deps[*m.NextMatchID] = append(deps[*m.NextMatchID], m.ID)
		}
	}
	counters := map[string]int{}
	remaining := len(indices)
	settings := settingsFor(t).Numbering
	for remaining > 0 {
		selected := -1
		for _, i := range indices {
			m := t.Matches[i]
			if done[m.ID] {
				continue
			}
			ready := true
			for _, dep := range deps[m.ID] {
				if !done[dep] {
					ready = false
					break
				}
			}
			if ready {
				selected = i
				break
			}
		}
		if selected < 0 {
			return fmt.Errorf("%w: court order contradicts bracket dependencies", ErrInvalidMatchPosition)
		}
		m := &t.Matches[selected]
		key := "all"
		switch settings.Scope {
		case "day":
			key = fmt.Sprint(m.Day)
		case "court_day":
			key = fmt.Sprintf("%d/%d", m.Day, m.Court)
		}
		number := settings.StartAt + counters[key]
		counters[key]++
		m.MatchNumber = &number
		done[m.ID] = true
		remaining--
	}
	return nil
}

func sameNumberScope(t *Tournament, day, court int, other Match) bool {
	switch settingsFor(t).Numbering.Scope {
	case "day":
		return other.Day == day
	case "court_day":
		return other.Day == day && other.Court == court
	default:
		return true
	}
}
