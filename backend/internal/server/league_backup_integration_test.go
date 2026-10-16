package server

import (
	"net/http"
	"testing"

	"backend/internal/modules/leagues"
	p "backend/internal/modules/poomsae"
	"backend/internal/testutil"
)

func TestPostgresLeagueBackupRoundTrip(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "backup-league@example.test")
	base := "/api/v1/organizations/" + org
	league := decode[leagues.League](t, requestJSON(t, router, http.MethodPost, base+"/leagues", token, map[string]any{
		"name": "Season", "gender": "male", "ageCategory": "بزرگسالان", "seasonName": "1405", "stageCount": 1, "weeksPerStage": 1,
	}, 200))
	club := decode[map[string]any](t, requestJSON(t, router, http.MethodPost, base+"/clubs", token, map[string]any{"name": "Backup Club"}, 200))
	league = decode[leagues.League](t, requestJSON(t, router, http.MethodPost, base+"/leagues/"+league.ID+"/teams", token, map[string]any{"clubId": club["id"]}, 200))
	requestJSON(t, router, http.MethodPost, base+"/leagues/"+league.ID+"/athletes", token, map[string]any{
		"name": "Backup Athlete", "gender": "male", "clubId": club["id"], "teamId": league.Teams[0].ID, "weightCategory": "-54",
	}, 200)
	requestJSON(t, router, http.MethodPost, base+"/leagues/"+league.ID+"/tournaments", token, map[string]any{
		"name": "Week One", "date": "2026-10-01", "courts": 2, "stageId": league.Stages[0].ID, "weekId": league.Stages[0].Weeks[0].ID,
	}, 200)
	backup := decode[leagues.Backup](t, requestJSON(t, router, http.MethodGet, base+"/leagues/"+league.ID+"/export", token, nil, 200))
	if backup.League.Settings.SeasonName != "1405" || len(backup.Tournaments) != 1 || len(backup.Tournaments[0].Athletes) != 1 {
		t.Fatalf("incomplete export: %#v", backup)
	}
	imported := decode[leagues.League](t, requestJSON(t, router, http.MethodPost, base+"/leagues/import", token, leagues.ImportBackupInput{Backup: backup, Mode: "add"}, 200))
	if imported.ID == league.ID || len(imported.Athletes) != 1 || imported.Settings.SeasonName != "1405" {
		t.Fatalf("imported league is incomplete: %#v", imported)
	}
	reloaded := decode[leagues.Backup](t, requestJSON(t, router, http.MethodGet, base+"/leagues/"+imported.ID+"/export", token, nil, 200))
	if len(reloaded.Tournaments) != 1 || len(reloaded.Tournaments[0].Athletes) != 1 || reloaded.Tournaments[0].Athletes[0].ID == backup.Tournaments[0].Athletes[0].ID {
		t.Fatalf("import did not persist or remap tournament athletes: %#v", reloaded.Tournaments)
	}
	requestJSON(t, router, http.MethodPost, base+"/leagues/import", token, leagues.ImportBackupInput{Backup: backup, Mode: "replace"}, 200)
	requestJSON(t, router, http.MethodGet, base+"/leagues/"+league.ID, token, nil, 200)
}

func TestPostgresPoomsaeLeagueWeeksPersist(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "poomsae-league@example.test")
	base := "/api/v1/organizations/" + org
	event := decode[p.Event](t, poomsaeJSON(t, router, http.MethodPost, base+"/poomsae-events", token, poomsaeEventBody(), 0, 200))
	league := decode[p.PoomsaeLeague](t, requestJSON(t, router, http.MethodPost, base+"/poomsae-leagues", token, map[string]any{
		"name": "Poomsae Season", "seasonName": "1405", "stageCount": 2, "weeksPerStage": 2,
	}, 200))
	if len(league.Weeks) != 4 {
		t.Fatalf("expected four weeks, got %d", len(league.Weeks))
	}
	requestJSON(t, router, http.MethodPut, base+"/poomsae-leagues/"+league.ID+"/weeks/"+league.Weeks[0].ID, token, map[string]any{"eventId": event.ID}, 200)
	detail := decode[struct {
		League    p.PoomsaeLeague          `json:"league"`
		Standings p.PoomsaeLeagueStandings `json:"standings"`
	}](t, requestJSON(t, router, http.MethodGet, base+"/poomsae-leagues/"+league.ID, token, nil, 200))
	if detail.League.Weeks[0].EventID == nil || *detail.League.Weeks[0].EventID != event.ID || len(detail.Standings.Players) != 0 {
		t.Fatalf("poomsae league did not persist: %#v", detail)
	}
}
