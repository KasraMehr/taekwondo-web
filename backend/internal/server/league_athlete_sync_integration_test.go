package server

import (
	"backend/internal/modules/leagues"
	"backend/internal/modules/tournaments"
	"backend/internal/testutil"
	"net/http"
	"testing"
)

func TestPostgresLeagueTournamentAthleteEditsStayInSync(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "league-athlete-sync@example.test")
	base := "/api/v1/organizations/" + org
	l := decode[leagues.League](t, requestJSON(t, router, http.MethodPost, base+"/leagues", token, map[string]any{"name": "Sync", "gender": "male", "ageCategory": "بزرگسالان", "stageCount": 1, "weeksPerStage": 1}, 200))
	lp := base + "/leagues/" + l.ID
	a := decode[leagues.Athlete](t, requestJSON(t, router, http.MethodPost, lp+"/athletes", token, map[string]any{"name": "Original", "gender": "male", "weightCategory": "-54", "coachName": "First"}, 200))
	tour := decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, lp+"/tournaments", token, map[string]any{"name": "Week", "date": "2026-11-18", "courts": 1, "stageId": l.Stages[0].ID, "weekId": l.Stages[0].Weeks[0].ID}, 200))
	tp := base + "/tournaments/" + tour.ID
	requestJSON(t, router, http.MethodPut, lp+"/athletes/"+a.ID, token, map[string]any{"weightCategory": "-58", "coachName": "Second", "active": true}, 200)
	tour = decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodGet, tp, token, nil, 200))
	if len(tour.Athletes) != 1 || tour.Athletes[0].WeightCategory != "-58" || tour.Athletes[0].Coach != "Second" {
		t.Fatalf("league edit did not reach tournament: %+v", tour.Athletes)
	}
	requestJSON(t, router, http.MethodPut, tp+"/athletes/"+a.ID, token, map[string]any{"name": "Renamed", "club": "", "coach": "Third", "weightCategory": "-63"}, 200)
	l = decode[leagues.League](t, requestJSON(t, router, http.MethodGet, lp, token, nil, 200))
	if len(l.Athletes) != 1 || l.Athletes[0].Name != "Renamed" || l.Athletes[0].CoachName != "Third" || l.Athletes[0].WeightCategory != "-63" {
		t.Fatalf("tournament edit did not reach league: %+v", l.Athletes)
	}
}
