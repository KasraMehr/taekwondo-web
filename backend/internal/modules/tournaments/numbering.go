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
	clearMatchNumbers(t)
	applyCourtPlan(t)
	if err = assignMatchNumbers(t); err != nil {
		return nil, err
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
