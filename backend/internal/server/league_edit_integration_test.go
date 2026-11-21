package server

import (
	"backend/internal/modules/leagues"
	"backend/internal/modules/tournaments"
	"backend/internal/testutil"
	"net/http"
	"testing"
)

func TestPostgresLeagueEditLockAndCoach(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "league-edit@example.test")
	base := "/api/v1/organizations/" + org
	l := decode[leagues.League](t, requestJSON(t, router, http.MethodPost, base+"/leagues", token, map[string]any{"name": "Before", "gender": "male", "ageCategory": "بزرگسالان", "stageCount": 1, "weeksPerStage": 2, "groupNames": []string{"A", "B"}}, 200))
	path := base + "/leagues/" + l.ID
	firstStage := l.Stages[0].ID
	edit := map[string]any{"name": "After", "gender": "male", "ageCategory": "امیدها", "seasonName": "New season", "stageCount": 3}
	l = decode[leagues.League](t, requestJSON(t, router, http.MethodPut, path, token, edit, 200))
	if l.Name != "After" || l.AgeCategory != "امیدها" || len(l.Stages) != 3 || l.Stages[0].ID != firstStage || len(l.Stages[2].Weeks) != 2 {
		t.Fatalf("edit did not preserve structure: %+v", l)
	}
	edit["stageCount"] = 2
	l = decode[leagues.League](t, requestJSON(t, router, http.MethodPut, path, token, edit, 200))
	if len(l.Stages) != 2 {
		t.Fatal("stage reduction failed")
	}
	group := l.Groups[0].ID
	addTeam := func(name string) leagues.Team {
		club := decode[map[string]any](t, requestJSON(t, router, http.MethodPost, base+"/clubs", token, map[string]any{"name": name}, 200))
		updated := decode[leagues.League](t, requestJSON(t, router, http.MethodPost, path+"/teams", token, map[string]any{"clubId": club["id"], "groupId": group}, 200))
		for _, team := range updated.Teams {
			if team.ClubID == club["id"] {
				return team
			}
		}
		t.Fatal("team not found")
		return leagues.Team{}
	}
	team := addTeam("Team A")
	addAthlete := func(name string, teamID any) leagues.Athlete {
		return decode[leagues.Athlete](t, requestJSON(t, router, http.MethodPost, path+"/athletes", token, map[string]any{"name": name, "gender": "male", "teamId": teamID, "groupId": group, "weightCategory": "-54", "coachName": "Coach " + name}, 200))
	}
	free := addAthlete("Free", nil)
	member := addAthlete("Member", team.ID)
	tour := decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, path+"/tournaments", token, map[string]any{"name": "Week 1", "date": "2026-11-18", "courts": 1, "stageId": l.Stages[0].ID, "weekId": l.Stages[0].Weeks[0].ID, "groupId": group}, 200))
	tp := base + "/tournaments/" + tour.ID
	for _, a := range tour.Athletes {
		requestJSON(t, router, http.MethodPost, tp+"/athletes/"+a.ID+"/weigh-in", token, map[string]any{"weightKg": 54}, 200)
	}
	tour = decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, tp+"/draw", token, map[string]any{"type": "random"}, 200))
	l = decode[leagues.League](t, requestJSON(t, router, http.MethodGet, path, token, nil, 200))
	if l.StructureLocked {
		t.Fatal("draw alone locked league")
	}
	requestJSON(t, router, http.MethodPost, tp+"/matches/"+tour.Matches[0].ID+"/start", token, nil, 200)
	l = decode[leagues.League](t, requestJSON(t, router, http.MethodGet, path, token, nil, 200))
	if !l.StructureLocked {
		t.Fatal("starting match did not lock structure")
	}
	requestJSON(t, router, http.MethodPut, path, token, edit, 409)
	l.Settings.Scoring.WinPoint = 7
	l.Settings.Scoring.WeighInPoint = 5
	requestJSON(t, router, http.MethodPut, path+"/settings", token, l.Settings, 200)
	standings := decode[leagues.Standings](t, requestJSON(t, router, http.MethodGet, path+"/standings", token, nil, 200))
	if len(standings.Teams) != 1 || standings.Teams[0].ID != team.ID || len(standings.Players) != 2 {
		t.Fatalf("free athlete must remain individual, not a team: %+v", standings)
	}
	for _, player := range standings.Players {
		if player.WeighInPoints != 5 {
			t.Fatalf("published score was not recalculated: %+v", player)
		}
	}
	lockedSettings := l.Settings
	lockedSettings.TeamsToPromote++
	requestJSON(t, router, http.MethodPut, path+"/settings", token, lockedSettings, 409)
	requestJSON(t, router, http.MethodPost, path+"/groups", token, map[string]any{"name": "C"}, 409)
	requestJSON(t, router, http.MethodPut, path+"/groups/"+group, token, map[string]any{"name": "C"}, 409)
	requestJSON(t, router, http.MethodDelete, path+"/groups/"+group, token, nil, 409)
	requestJSON(t, router, http.MethodPut, path+"/teams/"+team.ID, token, map[string]any{"groupId": l.Groups[1].ID, "active": true}, 409)
	requestJSON(t, router, http.MethodPost, path+"/stages/"+firstStage+"/weeks", token, map[string]any{"name": "Extra"}, 409)
	requestJSON(t, router, http.MethodDelete, path+"/athletes/"+member.ID, token, nil, 409)
	update := map[string]any{"teamId": nil, "groupId": group, "weightCategory": "-54", "coachName": "Updated Coach", "active": true}
	requestJSON(t, router, http.MethodPut, path+"/athletes/"+free.ID, token, update, 200)
	update["teamId"] = team.ID
	requestJSON(t, router, http.MethodPut, path+"/athletes/"+free.ID, token, update, 409)
	added := addTeam("Late Team")
	addAthlete("Late Member", added.ID)
	addAthlete("Late Free", nil)
	// Late arrivals can be registered first and assigned to an existing group later.
	club := decode[map[string]any](t, requestJSON(t, router, http.MethodPost, base+"/clubs", token, map[string]any{"name": "Unassigned team"}, 200))
	unassignedLeague := decode[leagues.League](t, requestJSON(t, router, http.MethodPost, path+"/teams", token, map[string]any{"clubId": club["id"]}, 200))
	for _, team := range unassignedLeague.Teams {
		if team.ClubID == club["id"] {
			requestJSON(t, router, http.MethodPut, path+"/teams/"+team.ID, token, map[string]any{"groupId": group, "active": true}, 200)
			requestJSON(t, router, http.MethodPut, path+"/teams/"+team.ID, token, map[string]any{"groupId": l.Groups[1].ID, "active": true}, 409)
		}
	}
	input := map[string]any{"name": "Unassigned free", "gender": "male", "weightCategory": "-54", "coachName": "   "}
	requestJSON(t, router, http.MethodPost, path+"/athletes", token, input, 400)
	input["coachName"] = "Required Coach"
	unassigned := decode[leagues.Athlete](t, requestJSON(t, router, http.MethodPost, path+"/athletes", token, input, 200))
	assign := map[string]any{"groupId": group, "weightCategory": "-54", "coachName": "Required Coach", "active": true}
	requestJSON(t, router, http.MethodPut, path+"/athletes/"+unassigned.ID, token, assign, 200)
	assign["coachName"] = " "
	requestJSON(t, router, http.MethodPut, path+"/athletes/"+unassigned.ID, token, assign, 400)
	assign["coachName"] = "Required Coach"
	assign["groupId"] = l.Groups[1].ID
	requestJSON(t, router, http.MethodPut, path+"/athletes/"+unassigned.ID, token, assign, 409)
	public := decode[map[string]any](t, requestJSON(t, router, http.MethodGet, "/api/v1/public/competitions/"+tour.ID, "", nil, 200))
	labels := map[string]string{}
	for _, raw := range public["athletes"].([]any) {
		a := raw.(map[string]any)
		labels[a["name"].(string)] = a["club"].(string)
	}
	if labels["Free"] != "Updated Coach" || labels["Member"] != "Team A" {
		t.Fatalf("wrong affiliations: %v", labels)
	}
	m := tour.Matches[0]
	requestJSON(t, router, http.MethodPost, tp+"/matches/"+m.ID+"/result", token, map[string]any{"winType": "WDR", "blueId": *m.Athlete1ID, "redId": *m.Athlete2ID, "winnerId": *m.Athlete1ID, "rounds": []any{}}, 200)
	requestJSON(t, router, http.MethodDelete, tp+"/matches/"+m.ID+"/result", token, nil, 200)
	l = decode[leagues.League](t, requestJSON(t, router, http.MethodGet, path, token, nil, 200))
	if !l.StructureLocked {
		t.Fatal("clearing results unlocked league")
	}
}
