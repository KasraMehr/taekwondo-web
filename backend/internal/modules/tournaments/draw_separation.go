package tournaments

import (
	"math/rand"
	"strings"
)

func athletePassedWeighIn(a TournamentAthlete) bool {
	// A structured weigh-in is authoritative. Never trust a contradictory legacy flag.
	if a.WeighIn != nil {
		return a.WeighIn.Status == WeighInPassed
	}
	return a.WeighedIn
}
func teamConflictProfile(slots []*string, teams map[string]string, rounds int) []int {
	p := make([]int, rounds)
	for i, a := range slots {
		if a == nil || teams[*a] == "" {
			continue
		}
		for j := i + 1; j < len(slots); j++ {
			b := slots[j]
			if b == nil || teams[*a] != teams[*b] {
				continue
			}
			for r := 1; r <= rounds; r++ {
				if i/(1<<r) == j/(1<<r) {
					p[r-1]++
					break
				}
			}
		}
	}
	return p
}
func profileLess(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}
func profileResolved(p []int) bool {
	for i := 0; i < len(p)-1; i++ {
		if p[i] > 0 {
			return false
		}
	}
	return true
}

// Same constraints as desktop: lock top four ranked seeds and preserve bye recipients.
func optimizeTeamSeparation(slots []*string, athletes []TournamentAthlete, drawType string, rounds int) {
	teams := map[string]string{}
	sizes := map[string]int{}
	fixed := map[string]bool{}
	ranked := 0
	for _, a := range athletes {
		key := strings.TrimSpace(a.Coach)
		if key == "" {
			key = strings.TrimSpace(a.Club)
		}
		teams[a.ID] = key
		if key != "" {
			sizes[key]++
		}
		if drawType == "ranking" && a.Ranking != nil && *a.Ranking > 0 && ranked < 4 {
			fixed[a.ID] = true
			ranked++
		}
	}
	groups := [][]int{{}, {}}
	movable := []int{}
	for i, id := range slots {
		if id == nil || fixed[*id] {
			continue
		}
		movable = append(movable, i)
		g := 1
		if slots[i^1] == nil {
			g = 0
		}
		groups[g] = append(groups[g], i)
	}
	climb := func() []int {
		profile := teamConflictProfile(slots, teams, rounds)
		for pass := 0; pass < 30 && !profileResolved(profile); pass++ {
			improved := false
			for _, group := range groups {
				for _, i := range group {
					if sizes[teams[*slots[i]]] < 2 {
						continue
					}
					for _, j := range group {
						if i == j {
							continue
						}
						slots[i], slots[j] = slots[j], slots[i]
						candidate := teamConflictProfile(slots, teams, rounds)
						if profileLess(candidate, profile) {
							profile = candidate
							improved = true
						} else {
							slots[i], slots[j] = slots[j], slots[i]
						}
					}
				}
			}
			if !improved {
				break
			}
		}
		return profile
	}
	best := climb()
	values := append([]*string(nil), slots...)
	for attempt := 0; attempt < 8 && !profileResolved(best); attempt++ {
		copy(slots, values)
		for _, group := range groups {
			rand.Shuffle(len(group), func(i, j int) { slots[group[i]], slots[group[j]] = slots[group[j]], slots[group[i]] })
		}
		candidate := climb()
		if profileLess(candidate, best) {
			best = candidate
			copy(values, slots)
		}
	}
	for _, i := range movable {
		slots[i] = values[i]
	}
}
