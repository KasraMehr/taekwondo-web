package tournaments

import (
	"context"
	"fmt"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"sort"
	"strings"
	"time"
)

type categoryLoad struct {
	category    string
	load, first int
	pinned      bool
	courts      []int
}

func categoryStats(t *Tournament) []categoryLoad {
	stats := []categoryLoad{}
	positions := map[string]int{}
	for _, m := range t.Matches {
		i, ok := positions[m.WeightCategory]
		if !ok {
			i = len(stats)
			positions[m.WeightCategory] = i
			stats = append(stats, categoryLoad{category: m.WeightCategory})
		}
		if lifecycleIsBye(&m) {
			continue
		}
		s := &stats[i]
		s.load++
		if m.Round == 1 {
			s.first++
		}
		if m.WinnerID != nil || m.MovedAt != nil || lifecycleMatchStatus(&m) == MatchStatusOngoing {
			s.pinned = true
		}
		if m.Court > 0 {
			found := false
			for _, c := range s.courts {
				if c == m.Court {
					found = true
				}
			}
			if !found {
				s.courts = append(s.courts, m.Court)
			}
		}
	}
	for i := range stats {
		sort.Ints(stats[i].courts)
	}
	return stats
}
func applyCourtPlanSingleDay(t *Tournament) {
	stats := categoryStats(t)
	loads := make([]int, t.Courts+1)
	assignment := map[string][]int{}
	free := []categoryLoad{}
	settings := settingsFor(t).Schedule
	for _, stat := range stats {
		if stat.pinned {
			assignment[stat.category] = stat.courts
			for _, m := range t.Matches {
				if m.WeightCategory == stat.category && !lifecycleIsBye(&m) && m.Court > 0 && m.Court <= t.Courts {
					loads[m.Court]++
				}
			}
		} else {
			free = append(free, stat)
		}
	}
	sort.SliceStable(free, func(i, j int) bool { return free[i].load > free[j].load })
	for _, stat := range free {
		if stat.load == 0 {
			assignment[stat.category] = []int{}
			continue
		}
		courts := make([]int, t.Courts)
		for i := range courts {
			courts[i] = i + 1
		}
		sort.SliceStable(courts, func(i, j int) bool { return loads[courts[i]] < loads[courts[j]] })
		switch settings.Mode {
		case "single_court":
			courts = []int{settings.SingleCourt}
		case "split_halves":
			courts = courts[:min(2, t.Courts)]
		default:
			courts = courts[:1]
		}
		assignCategoryCourts(t.Matches, stat.category, courts)
		used := map[int]bool{}
		maxRound := 0
		for _, m := range t.Matches {
			if m.WeightCategory == stat.category {
				maxRound = max(maxRound, m.Round)
			}
		}
		for i := range t.Matches {
			m := &t.Matches[i]
			if m.WeightCategory != stat.category || lifecycleIsBye(m) {
				continue
			}
			if settings.Mode != "single_court" && settings.FinalCourtPolicy == "fixed" && m.Round == maxRound {
				m.Court = settings.FinalCourt
			}
			loads[m.Court]++
			used[m.Court] = true
		}
		assignment[stat.category] = []int{}
		for c := 1; c <= t.Courts; c++ {
			if used[c] {
				assignment[stat.category] = append(assignment[stat.category], c)
			}
		}
	}
	if settingsFor(t).Numbering.Order == "category" {
		syncCategoryOrders(t)
	} else {
		syncMatchOrders(t.Matches)
	}
	t.CourtAssignment = assignment
}
func syncCategoryOrders(t *Tournament) {
	ranks := map[string]int{}
	for i, cat := range WeightOptions(t.AgeCategory, t.Gender) {
		ranks[cat] = i
	}
	indices := []int{}
	for i, m := range t.Matches {
		if !lifecycleIsBye(&m) {
			indices = append(indices, i)
		}
	}
	sort.SliceStable(indices, func(i, j int) bool {
		a, b := t.Matches[indices[i]], t.Matches[indices[j]]
		if a.Court != b.Court {
			return a.Court < b.Court
		}
		if a.WeightCategory != b.WeightCategory {
			return ranks[a.WeightCategory] < ranks[b.WeightCategory]
		}
		if a.Round != b.Round {
			return a.Round < b.Round
		}
		if a.BracketIndex != nil && b.BracketIndex != nil {
			return *a.BracketIndex < *b.BracketIndex
		}
		return false
	})
	counters := map[int]int{}
	for _, i := range indices {
		m := &t.Matches[i]
		counters[m.Court]++
		m.Order = counters[m.Court]
	}
}

func assignCategoryCourts(matches []Match, category string, courts []int) {
	if len(courts) == 0 {
		courts = []int{1}
	}
	rounds := map[int][]int{}
	maxRound := 0
	for i, m := range matches {
		if m.WeightCategory != category {
			continue
		}
		rounds[m.Round] = append(rounds[m.Round], i)
		maxRound = max(maxRound, m.Round)
	}
	for round, indices := range rounds {
		sort.SliceStable(indices, func(i, j int) bool {
			a, b := matches[indices[i]].BracketIndex, matches[indices[j]].BracketIndex
			return a != nil && b != nil && *a < *b
		})
		for position, i := range indices {
			m := &matches[i]
			if lifecycleIsBye(m) {
				m.Court = 0
				m.Order = 0
				continue
			}
			if round == maxRound {
				m.Court = courts[0]
			} else {
				m.Court = courts[min(len(courts)-1, position*len(courts)/len(indices))]
			}
		}
	}
}
func syncMatchOrders(matches []Match) {
	maxima := map[string]int{}
	active := []int{}
	for i, m := range matches {
		if m.Court < 1 || lifecycleIsBye(&m) {
			continue
		}
		active = append(active, i)
		maxima[m.WeightCategory] = max(maxima[m.WeightCategory], m.Round)
	}
	sizes := map[string]map[int]int{}
	for _, i := range active {
		m := matches[i]
		phase := maxima[m.WeightCategory] - m.Round
		if sizes[m.WeightCategory] == nil {
			sizes[m.WeightCategory] = map[int]int{}
		}
		sizes[m.WeightCategory][phase]++
	}
	comparer := collate.New(language.Persian)
	sort.SliceStable(active, func(i, j int) bool {
		a, b := matches[active[i]], matches[active[j]]
		ad := a.WinnerID != nil || lifecycleMatchStatus(&a) == MatchStatusOngoing
		bd := b.WinnerID != nil || lifecycleMatchStatus(&b) == MatchStatusOngoing
		if ad != bd {
			return ad
		}
		if ad {
			return a.Order < b.Order
		}
		pa, pb := maxima[a.WeightCategory]-a.Round, maxima[b.WeightCategory]-b.Round
		if pa != pb {
			return pa > pb
		}
		if a.Court != b.Court {
			return a.Court < b.Court
		}
		if sizes[a.WeightCategory][pa] != sizes[b.WeightCategory][pb] {
			return sizes[a.WeightCategory][pa] > sizes[b.WeightCategory][pb]
		}
		if a.WeightCategory != b.WeightCategory {
			return comparer.CompareString(a.WeightCategory, b.WeightCategory) < 0
		}
		return a.Side < b.Side
	})
	orders := map[[2]int]int{}
	for _, i := range active {
		m := &matches[i]
		key := [2]int{m.Day, m.Court}
		orders[key]++
		m.Order = orders[key]
	}
}
func (s *tournamentService) RebalanceCourts(ctx context.Context, id string) (*Tournament, error) {
	t, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if hasStartedMatches(t) {
		return nil, ErrMatchLocked
	}
	clearMatchNumbers(t)
	if err := applyCourtPlan(t); err != nil {
		return nil, err
	}
	t.UpdatedAt = time.Now().UTC()
	if err = s.repository.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func applyCourtPlan(t *Tournament) error {
	assignment := map[string][]int{}
	for _, day := range settingsFor(t).Schedule.Days {
		subset := *t
		subset.Matches = []Match{}
		indices := []int{}
		for i, m := range t.Matches {
			if dayForCategory(t, m.WeightCategory) == day.Day {
				m.Day = day.Day
				subset.Matches = append(subset.Matches, m)
				indices = append(indices, i)
			}
		}
		applyCourtPlanSingleDay(&subset)
		if err := applyHoguCourtPlan(&subset); err != nil {
			return err
		}
		for i, index := range indices {
			t.Matches[index] = subset.Matches[i]
		}
		for cat, courts := range subset.CourtAssignment {
			assignment[cat] = courts
		}
	}
	t.CourtAssignment = assignment
	return nil
}

// Keep each weight category on courts carrying its required protector size.
func applyHoguCourtPlan(t *Tournament) error {
	hogu := settingsFor(t).Hogu
	if !hogu.Enabled {
		return nil
	}
	loads := make([]int, t.Courts+1)
	categories := make([]string, 0, len(t.CourtAssignment))
	for category := range t.CourtAssignment {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	for _, category := range categories {
		size := strings.TrimSpace(hogu.CategorySizes[category])
		if size == "" {
			continue
		}
		compatible := []int{}
		for court := 1; court <= t.Courts; court++ {
			for _, available := range hogu.CourtSizes[court] {
				if strings.TrimSpace(available) == size {
					compatible = append(compatible, court)
					break
				}
			}
		}
		if len(compatible) == 0 {
			return fmt.Errorf("%w: no court has hogu size %s required for category %s", ErrInvalidTournamentInput, size, category)
		}
		moved := map[string]int{}
		for _, match := range t.Matches {
			if match.WeightCategory == category && match.MovedAt != nil && !lifecycleIsBye(&match) {
				if !containsInt(compatible, match.Court) {
					return fmt.Errorf("%w: manually moved match in category %s is on a court without hogu size %s", ErrInvalidTournamentInput, category, size)
				}
				moved[match.ID] = match.Court
			}
		}
		if assigned := t.CourtAssignment[category]; len(assigned) > 0 {
			kept := []int{}
			for _, court := range assigned {
				if containsInt(compatible, court) {
					kept = append(kept, court)
				}
			}
			if len(kept) > 0 {
				compatible = kept
			}
		}
		sort.SliceStable(compatible, func(i, j int) bool {
			if loads[compatible[i]] == loads[compatible[j]] {
				return compatible[i] < compatible[j]
			}
			return loads[compatible[i]] < loads[compatible[j]]
		})
		if len(compatible) > 2 {
			compatible = compatible[:2]
		}
		assignCategoryCourts(t.Matches, category, compatible)
		for i := range t.Matches {
			if court, ok := moved[t.Matches[i].ID]; ok {
				t.Matches[i].Court = court
			}
		}
		used := []int{}
		for _, m := range t.Matches {
			if m.WeightCategory == category && !lifecycleIsBye(&m) && m.Court > 0 {
				loads[m.Court]++
				if !containsInt(used, m.Court) {
					used = append(used, m.Court)
				}
			}
		}
		t.CourtAssignment[category] = used
	}
	if settingsFor(t).Numbering.Order == "category" {
		syncCategoryOrders(t)
	} else {
		syncMatchOrders(t.Matches)
	}
	return nil
}

func containsInt(values []int, value int) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
