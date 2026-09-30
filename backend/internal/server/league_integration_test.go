package server

import (
	"net/http"
	"testing"

	"backend/internal/modules/leagues"
	"backend/internal/modules/tournaments"
	"backend/internal/testutil"
)

func TestPostgresLeagueRosterCreatesTournamentSnapshot(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "league@example.test")
	base := "/api/v1/organizations/" + org
	club := decode[map[string]any](t, requestJSON(t, router, http.MethodPost, base+"/clubs", token, map[string]any{"name": "Club One"}, 200))
	league := decode[leagues.League](t, requestJSON(t, router, http.MethodPost, base+"/leagues", token, map[string]any{"name": "Season 1", "gender": "male", "ageCategory": "بزرگسالان", "seasonName": "2026", "stageCount": 1, "weeksPerStage": 1, "groupNames": []string{"A"}}, 200))
	if len(league.Stages) != 1 || len(league.Stages[0].Weeks) != 1 || len(league.Groups) != 1 {
		t.Fatalf("league structure missing: %#v", league)
	}
	league = decode[leagues.League](t, requestJSON(t, router, http.MethodPost, base+"/leagues/"+league.ID+"/teams", token, map[string]any{"clubId": club["id"], "groupId": league.Groups[0].ID}, 200))
	athlete := decode[leagues.Athlete](t, requestJSON(t, router, http.MethodPost, base+"/leagues/"+league.ID+"/athletes", token, map[string]any{"name": "League Athlete", "gender": "male", "clubId": club["id"], "teamId": league.Teams[0].ID, "groupId": league.Groups[0].ID, "weightCategory": "-54"}, 200))
	reloaded := decode[leagues.League](t, requestJSON(t, router, http.MethodGet, base+"/leagues/"+league.ID, token, nil, 200))
	if len(reloaded.Athletes) != 1 || reloaded.Athletes[0].Name != "League Athlete" || reloaded.Athletes[0].TeamID == nil || reloaded.Athletes[0].GroupID == nil {
		t.Fatalf("league athlete did not survive a database reload: %#v", reloaded.Athletes)
	}
	requestJSON(t, router, http.MethodPut, base+"/leagues/"+league.ID+"/groups/"+league.Groups[0].ID, token, map[string]any{"name": "Group One"}, 200)
	event := decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, base+"/leagues/"+league.ID+"/tournaments", token, map[string]any{"name": "Week A", "date": "2026-10-01", "courts": 2, "stageId": league.Stages[0].ID, "weekId": league.Stages[0].Weeks[0].ID, "groupId": league.Groups[0].ID}, 200))
	if len(event.Athletes) != 1 || event.Athletes[0].ProfileID == nil || *event.Athletes[0].ProfileID != athlete.ProfileID {
		t.Fatalf("league roster was not imported: %#v", event.Athletes)
	}
	path := base + "/tournaments/" + event.ID + "/athletes/" + event.Athletes[0].ID + "/weigh-in"
	requestJSON(t, router, http.MethodPost, path, token, map[string]any{"weightKg": 54.2}, 200)
	requestJSON(t, router, http.MethodPost, base+"/leagues/"+league.ID+"/tournaments/"+event.ID+"/sync-roster", token, nil, 409)
}

func TestPostgresLeagueStandingsRefreshAfterEachTournamentResult(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "live-league@example.test")
	base := "/api/v1/organizations/" + org
	league := decode[leagues.League](t, requestJSON(t, router, http.MethodPost, base+"/leagues", token, map[string]any{"name": "Live season", "gender": "male", "ageCategory": "بزرگسالان", "seasonName": "2026", "stageCount": 1, "weeksPerStage": 1, "groupNames": []string{"A"}}, 200))

	for i, name := range []string{"Blue", "Red"} {
		club := decode[map[string]any](t, requestJSON(t, router, http.MethodPost, base+"/clubs", token, map[string]any{"name": name + " Club"}, 200))
		profile := decode[map[string]any](t, requestJSON(t, router, http.MethodPost, base+"/athletes", token, map[string]any{"name": name, "gender": "male", "clubId": club["id"]}, 200))
		league = decode[leagues.League](t, requestJSON(t, router, http.MethodPost, base+"/leagues/"+league.ID+"/teams", token, map[string]any{"clubId": club["id"], "groupId": league.Groups[0].ID}, 200))
		requestJSON(t, router, http.MethodPost, base+"/leagues/"+league.ID+"/athletes", token, map[string]any{"profileId": profile["id"], "teamId": league.Teams[i].ID, "groupId": league.Groups[0].ID, "weightCategory": "-54", "ranking": i + 1}, 200)
	}

	event := decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, base+"/leagues/"+league.ID+"/tournaments", token, map[string]any{"name": "Week 1", "date": "2026-10-01", "courts": 2, "stageId": league.Stages[0].ID, "weekId": league.Stages[0].Weeks[0].ID, "groupId": league.Groups[0].ID}, 200))
	path := base + "/tournaments/" + event.ID
	// Tournament operators may add a guest that is not managed in the league
	// roster. Its profile_id is NULL and must not break the automatic refresh.
	requestJSON(t, router, http.MethodPost, path+"/athletes", token, map[string]any{"name": "Guest", "club": "Guest Club", "weightCategory": "-58"}, 200)
	event = decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodGet, path, token, nil, 200))
	for _, athlete := range event.Athletes {
		weight := 54.2
		if athlete.WeightCategory == "-58" {
			weight = 57.2
		}
		requestJSON(t, router, http.MethodPost, path+"/athletes/"+athlete.ID+"/weigh-in", token, map[string]any{"weightKg": weight}, 200)
	}
	event = decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, path+"/draw", token, map[string]any{"type": "ranked", "weightCategory": "-54"}, 200))
	if len(event.Matches) != 1 {
		t.Fatalf("expected one final, got %d", len(event.Matches))
	}
	ovr := decode[map[string]any](t, requestJSON(t, router, http.MethodGet, "/api/v1/public/ovr?tournamentId="+event.ID, "", nil, 200))
	if ovr["tournament"] == nil || len(ovr["courts"].([]any)) != event.Courts {
		t.Fatalf("public OVR did not expose the tournament courts: %#v", ovr)
	}
	match := event.Matches[0]
	requestJSON(t, router, http.MethodPost, path+"/matches/"+match.ID+"/start", token, nil, 200)
	requestJSON(t, router, http.MethodPost, path+"/matches/"+match.ID+"/result", token, map[string]any{"winType": "WDR", "blueId": *match.Athlete1ID, "redId": *match.Athlete2ID, "winnerId": *match.Athlete1ID, "rounds": []any{}}, 200)

	standings := decode[leagues.Standings](t, requestJSON(t, router, http.MethodGet, base+"/leagues/"+league.ID+"/standings", token, nil, 200))
	if len(standings.Players) != 3 || standings.Players[0].TotalPoints != 13 || standings.Players[0].Gold != 1 || standings.Players[1].TotalPoints != 7 || standings.Players[1].Silver != 1 || standings.Players[2].Name != "Guest" || standings.Players[2].ID == "" {
		t.Fatalf("standings were not refreshed from the result: %#v", standings.Players)
	}
	if standings.Players[0].Wins != 1 || standings.Players[1].Losses != 1 || standings.Players[0].GroupID == "" || len(standings.Teams) != 3 || standings.Teams[0].Wins != 1 || standings.Teams[1].Losses != 1 {
		t.Fatalf("grouped individual/team win-loss totals are incomplete: players=%#v teams=%#v", standings.Players, standings.Teams)
	}
}
