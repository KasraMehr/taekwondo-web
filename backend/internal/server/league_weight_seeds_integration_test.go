package server

import (
	"backend/internal/modules/leagues"
	"backend/internal/modules/tournaments"
	"backend/internal/testutil"
	"fmt"
	"net/http"
	"testing"
)

func TestPostgresSeedsStayWithinWeightAndRefreshExistingWeek(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "weight-seeds@example.test")
	base := "/api/v1/organizations/" + org
	league := decode[leagues.League](t, requestJSON(t, router, http.MethodPost, base+"/leagues", token, map[string]any{"name": "Weight seeds", "gender": "male", "ageCategory": "بزرگسالان", "seasonName": "2026", "stageCount": 1, "weeksPerStage": 2, "groupNames": []string{"A"}}, 200))
	lp := base + "/leagues/" + league.ID
	for _, weight := range []string{"-54", "-58"} {
		for rank := 1; rank <= 2; rank++ {
			requestJSON(t, router, http.MethodPost, lp+"/athletes", token, map[string]any{"name": fmt.Sprintf("%s seed %d", weight, rank), "gender": "male", "groupId": league.Groups[0].ID, "weightCategory": weight, "ranking": rank, "coachName": "Test Coach"}, 200)
		}
	}
	create := func(week int) tournaments.Tournament {
		return decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, lp+"/tournaments", token, map[string]any{"name": fmt.Sprintf("Week %d", week+1), "date": "2026-11-18", "courts": 2, "stageId": league.Stages[0].ID, "weekId": league.Stages[0].Weeks[week].ID, "groupId": league.Groups[0].ID}, 200))
	}
	first := create(0)
	path := base + "/tournaments/" + first.ID
	for _, a := range first.Athletes {
		kg := 54.0
		if a.WeightCategory == "-58" {
			kg = 58
		}
		requestJSON(t, router, http.MethodPost, path+"/athletes/"+a.ID+"/weigh-in", token, map[string]any{"weightKg": kg}, 200)
	}
	for _, weight := range []string{"-54", "-58"} {
		first = decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, path+"/draw", token, map[string]any{"type": "ranked", "weightCategory": weight}, 200))
	}
	finish := func(weight string) {
		var winner string
		for _, a := range first.Athletes {
			if a.WeightCategory == weight && a.Ranking != nil && *a.Ranking == 2 {
				winner = a.ID
			}
		}
		for _, m := range first.Matches {
			if m.WeightCategory != weight {
				continue
			}
			requestJSON(t, router, http.MethodPost, path+"/matches/"+m.ID+"/start", token, nil, 200)
			requestJSON(t, router, http.MethodPost, path+"/matches/"+m.ID+"/result", token, map[string]any{"winType": "WDR", "blueId": *m.Athlete1ID, "redId": *m.Athlete2ID, "winnerId": winner, "rounds": []any{}}, 200)
		}
	}
	finish("-54")
	second := create(1)
	check := func(event tournaments.Tournament, completed58 bool) {
		for _, a := range event.Athletes {
			want := 1
			if a.Name == fmt.Sprintf("%s seed 2", a.WeightCategory) {
				want = 2
			}
			if a.WeightCategory == "-54" || completed58 {
				want = 3 - want
			}
			if a.Ranking == nil || *a.Ranking != want {
				t.Fatalf("incorrect seed for %s: got %v want %d", a.Name, a.Ranking, want)
			}
		}
	}
	check(second, false)
	finish("-58")
	second = decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, lp+"/tournaments/"+second.ID+"/sync-roster", token, nil, 200))
	check(second, true)
}
