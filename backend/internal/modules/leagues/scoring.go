package leagues

import (
	"fmt"
	"sort"

	"backend/internal/modules/tournaments"
)

// ScoreTournament derives a fresh ledger. Callers replace the old tournament ledger,
// which makes publishing idempotent and allows a corrected result to be republished.
func ScoreTournament(t *tournaments.Tournament, cfg ScoringConfig, teams map[string]*string) ([]PointEvent, error) {
	athletes := map[string]tournaments.TournamentAthlete{}
	for _, a := range t.Athletes {
		athletes[a.ID] = a
	}
	events := []PointEvent{}
	add := func(a tournaments.TournamentAthlete, kind, key string) *PointEvent {
		profile := ""
		if a.ProfileID != nil {
			profile = *a.ProfileID
		}
		events = append(events, PointEvent{EntryID: a.ID, ProfileID: profile, AthleteName: a.Name, ClubName: a.Club, WeightCategory: a.WeightCategory, TeamID: teams[a.ID], EventType: kind, EventKey: key})
		return &events[len(events)-1]
	}
	if cfg.CountWeighIn {
		for _, a := range t.Athletes {
			if a.WeighIn != nil && a.WeighIn.Status == tournaments.WeighInPassed {
				add(a, "weigh_in", "weigh-in:"+a.ID).Points = cfg.WeighInPoint
			}
		}
	}
	byCategory := map[string][]tournaments.Match{}
	for _, m := range t.Matches {
		if m.IsBye != nil && *m.IsBye {
			continue
		}
		byCategory[m.WeightCategory] = append(byCategory[m.WeightCategory], m)
		if m.Status == nil || *m.Status != tournaments.MatchStatusCompleted || m.WinnerID == nil || m.Result == nil || m.Athlete1ID == nil || m.Athlete2ID == nil {
			continue
		}
		winner, wok := athletes[*m.WinnerID]
		loserID := *m.Athlete1ID
		if loserID == *m.WinnerID {
			loserID = *m.Athlete2ID
		}
		loser, lok := athletes[loserID]
		if !wok || !lok {
			return nil, fmt.Errorf("match %s references an unknown athlete", m.ID)
		}
		wr, lr := roundRecord(m)
		we := add(winner, "match_stats", "match:"+m.ID+":winner")
		we.Wins, we.RoundDiff = 1, wr
		if wr == 2 {
			we.Wins20 = 1
		} else if wr == 1 {
			we.Wins21 = 1
		}
		if cfg.CountWin {
			we.Points = cfg.WinPoint
		}
		le := add(loser, "match_stats", "match:"+m.ID+":loser")
		le.Losses, le.RoundDiff = 1, lr
		if lr == -2 {
			le.Losses02 = 1
		} else if lr == -1 {
			le.Losses12 = 1
		}
	}
	for category, matches := range byCategory {
		if len(matches) == 0 {
			continue
		}
		complete := true
		for _, m := range matches {
			if m.Status == nil || *m.Status != tournaments.MatchStatusCompleted || m.WinnerID == nil {
				complete = false
				break
			}
		}
		if !complete {
			// Match and weigh-in events are useful while a weight category is still
			// running. Medal events are added only after its whole bracket finishes.
			continue
		}
		var final *tournaments.Match
		for i := range matches {
			if matches[i].NextMatchID == nil {
				if final != nil {
					return nil, fmt.Errorf("weight category %s has multiple finals", category)
				}
				final = &matches[i]
			}
		}
		if final == nil || final.Athlete1ID == nil || final.Athlete2ID == nil {
			return nil, fmt.Errorf("weight category %s has no completed final", category)
		}
		gold := athletes[*final.WinnerID]
		silverID := *final.Athlete1ID
		if silverID == *final.WinnerID {
			silverID = *final.Athlete2ID
		}
		silver := athletes[silverID]
		ge := add(gold, "gold", "medal:"+category+":gold")
		ge.Points, ge.Gold = cfg.Gold, 1
		se := add(silver, "silver", "medal:"+category+":silver")
		se.Points, se.Silver = cfg.Silver, 1
		for _, semi := range matches {
			if semi.NextMatchID == nil || *semi.NextMatchID != final.ID || semi.Athlete1ID == nil || semi.Athlete2ID == nil || semi.WinnerID == nil {
				continue
			}
			loserID := *semi.Athlete1ID
			if loserID == *semi.WinnerID {
				loserID = *semi.Athlete2ID
			}
			bronze := athletes[loserID]
			be := add(bronze, "bronze", "medal:"+category+":bronze:"+semi.ID)
			be.Points, be.Bronze = cfg.Bronze, 1
		}
	}
	return events, nil
}

func roundRecord(m tournaments.Match) (int, int) {
	if m.Result == nil || m.Result.WinnerCorner == nil {
		return 0, 0
	}
	w, l := 0, 0
	for _, r := range m.Result.Rounds {
		if r.Winner == nil {
			continue
		}
		if *r.Winner == *m.Result.WinnerCorner {
			w++
		} else {
			l++
		}
	}
	if w == 2 && l == 0 {
		return 2, -2
	}
	if w == 2 && l == 1 {
		return 1, -1
	}
	return 0, 0
}

func Rank(rows []Standing) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.TotalPoints != b.TotalPoints {
			return a.TotalPoints > b.TotalPoints
		}
		if a.Gold != b.Gold {
			return a.Gold > b.Gold
		}
		if a.Silver != b.Silver {
			return a.Silver > b.Silver
		}
		if a.Bronze != b.Bronze {
			return a.Bronze > b.Bronze
		}
		if a.RoundDiff != b.RoundDiff {
			return a.RoundDiff > b.RoundDiff
		}
		if a.Wins != b.Wins {
			return a.Wins > b.Wins
		}
		return a.Name < b.Name
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
}
