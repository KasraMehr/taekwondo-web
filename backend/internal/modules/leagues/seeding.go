package leagues

import (
	"backend/internal/modules/tournaments"
	"context"
)

// Refresh just seeds before drawing, including weeks created/weighed in early.
func (s *Service) RefreshTournamentSeeds(ctx context.Context, id string) error {
	var locked string
	if err := s.db.QueryRow(ctx, `SELECT id::text FROM tournaments WHERE id=$1 AND organization_id=$2 FOR UPDATE`, id, s.organizationID).Scan(&locked); err != nil {
		return err
	}
	repo := tournaments.NewTournamentRepository(s.db, s.organizationID)
	t, err := repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if t.LeagueID == nil {
		return nil
	}
	previous, err := s.previousWeekSeeds(ctx, t)
	if err != nil {
		return err
	}
	drawn := map[string]bool{}
	for _, m := range t.Matches {
		drawn[m.WeightCategory] = true
	}
	changed := false
	for i := range t.Athletes {
		p := &t.Athletes[i]
		ranks, ok := previous[p.WeightCategory]
		if !ok || drawn[p.WeightCategory] || p.ProfileID == nil {
			continue
		}
		p.Ranking = nil
		if rank, found := ranks[*p.ProfileID]; found {
			n := rank
			p.Ranking = &n
		}
		changed = true
	}
	if changed {
		return repo.Update(ctx, t)
	}
	return nil
}

// Previous completed weight brackets seed the next week. Week/stage order,
// not creation date or future results, determines the source.
func (s *Service) previousWeekSeeds(ctx context.Context, t *tournaments.Tournament) (map[string]map[string]int, error) {
	rows, err := s.db.Query(ctx, `WITH previous AS (
 SELECT w.id FROM league_weeks w JOIN league_stages st ON st.id=w.stage_id
 JOIN league_weeks current_week ON current_week.id=$2 JOIN league_stages current_stage ON current_stage.id=current_week.stage_id
 WHERE w.league_id=$1 AND (st.position,w.position)<(current_stage.position,current_week.position)
 AND EXISTS(SELECT 1 FROM tournaments p WHERE p.week_id=w.id AND p.group_id IS NOT DISTINCT FROM $3::uuid)
 ORDER BY st.position DESC,w.position DESC LIMIT 1
 ), totals AS (
 SELECT e.profile_id::text AS profile,e.weight_category,sum(e.points) AS points,sum(e.gold) AS gold,sum(e.silver) AS silver,sum(e.bronze) AS bronze,sum(e.round_diff) AS diff,sum(e.wins) AS wins
 FROM league_point_events e JOIN tournaments p ON p.id=e.tournament_id
 WHERE p.week_id=(SELECT id FROM previous) AND p.group_id IS NOT DISTINCT FROM $3::uuid AND e.profile_id IS NOT NULL
 AND EXISTS(SELECT 1 FROM league_point_events medal WHERE medal.tournament_id=p.id AND medal.weight_category=e.weight_category AND medal.gold>0)
 GROUP BY e.profile_id,e.weight_category
 ) SELECT profile,weight_category,row_number() OVER(PARTITION BY weight_category ORDER BY gold DESC,silver DESC,bronze DESC,points DESC,diff DESC,wins DESC,profile) FROM totals`, *t.LeagueID, t.WeekID, t.GroupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]map[string]int{}
	for rows.Next() {
		var profile, weight string
		var rank int
		if err = rows.Scan(&profile, &weight, &rank); err != nil {
			return nil, err
		}
		if out[weight] == nil {
			out[weight] = map[string]int{}
		}
		out[weight][profile] = rank
	}
	return out, rows.Err()
}
