package server

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"backend/internal/modules/leagues"
	"backend/internal/modules/tournaments"
	"backend/internal/testutil"
)

func TestPostgresLeagueFourAthletesThroughFinal(t *testing.T) {
	pool := testutil.Database(t)
	var today string
	if err := pool.QueryRow(context.Background(), `SELECT ((now() AT TIME ZONE 'Asia/Tehran')::date)::text`).Scan(&today); err != nil {
		t.Fatal(err)
	}
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "medals@example.test")
	base := "/api/v1/organizations/" + org
	league := decode[leagues.League](t, requestJSON(t, router, http.MethodPost, base+"/leagues", token, map[string]any{"name": "Medal test", "gender": "male", "ageCategory": "بزرگسالان", "seasonName": "2026", "stageCount": 1, "weeksPerStage": 2, "groupNames": []string{"A"}}, 200))
	lp := base + "/leagues/" + league.ID
	for i := 1; i <= 4; i++ {
		club := decode[map[string]any](t, requestJSON(t, router, http.MethodPost, base+"/clubs", token, map[string]any{"name": fmt.Sprintf("Team %d", i)}, 200))
		league = decode[leagues.League](t, requestJSON(t, router, http.MethodPost, lp+"/teams", token, map[string]any{"clubId": club["id"], "groupId": league.Groups[0].ID}, 200))
		var teamID string
		for _, team := range league.Teams {
			if team.ClubID == club["id"] {
				teamID = team.ID
			}
		}
		requestJSON(t, router, http.MethodPost, lp+"/athletes", token, map[string]any{"name": fmt.Sprintf("Player %d", i), "gender": "male", "clubId": club["id"], "teamId": teamID, "groupId": league.Groups[0].ID, "weightCategory": "-54", "ranking": 5 - i}, 200)
	}
	event := decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, lp+"/tournaments", token, map[string]any{"name": "Four-player test", "date": today, "courts": 1, "stageId": league.Stages[0].ID, "weekId": league.Stages[0].Weeks[0].ID, "groupId": league.Groups[0].ID}, 200))
	path := base + "/tournaments/" + event.ID
	nextWeek := decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, lp+"/tournaments", token, map[string]any{"name": "Next week", "date": "2026-11-24", "courts": 1, "stageId": league.Stages[0].ID, "weekId": league.Stages[0].Weeks[1].ID, "groupId": league.Groups[0].ID}, 200))
	for _, a := range event.Athletes {
		requestJSON(t, router, http.MethodPost, path+"/athletes/"+a.ID+"/weigh-in", token, map[string]any{"weightKg": 54.0}, 200)
	}
	event = decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, path+"/draw", token, map[string]any{"type": "ranked", "weightCategory": "-54"}, 200))
	if len(event.Matches) != 3 {
		t.Fatalf("expected two semifinals and a final: %d", len(event.Matches))
	}
	nameByID := map[string]string{}
	for _, a := range event.Athletes {
		nameByID[a.ID] = a.Name
	}
	wins := map[string]int{}
	bronze := map[string]bool{}
	var gold, silver string
	for played := 0; played < 3; played++ {
		var ready *tournaments.Match
		for i := range event.Matches {
			m := &event.Matches[i]
			if m.Athlete1ID != nil && m.Athlete2ID != nil && (m.Status == nil || *m.Status == tournaments.MatchStatusPending) {
				ready = m
				break
			}
		}
		if ready == nil {
			t.Fatal("no playable match")
		}
		winner, loser := *ready.Athlete1ID, *ready.Athlete2ID
		if nameByID[winner] > nameByID[loser] {
			winner, loser = loser, winner
		}
		requestJSON(t, router, http.MethodPost, path+"/matches/"+ready.ID+"/start", token, nil, 200)
		livePath := path + "/matches/" + ready.ID + "/live"
		liveInput := map[string]any{"version": 0, "blueId": *ready.Athlete1ID, "redId": *ready.Athlete2ID, "activeRound": 0, "rounds": []tournaments.MatchRound{{Number: 1, Red: tournaments.RoundScore{Punch: 1}}}}
		requestJSON(t, router, http.MethodPut, livePath, "", liveInput, 401)
		requestJSON(t, router, http.MethodPut, livePath, token, liveInput, 200)
		requestJSON(t, router, http.MethodPut, livePath, token, liveInput, 409)
		live := decode[map[string]struct {
			Score struct {
				Rounds []tournaments.MatchRound `json:"rounds"`
			} `json:"score"`
		}](t, requestJSON(t, router, http.MethodGet, "/api/v1/public/competitions/"+event.ID+"/live", "", nil, 200))
		if len(live[ready.ID].Score.Rounds) != 1 || live[ready.ID].Score.Rounds[0].Red.Punch != 1 {
			t.Fatal("red punch not published live")
		}
		ongoing := decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodGet, path, token, nil, 200))
		for _, m := range ongoing.Matches {
			if m.ID == ready.ID && (m.WinnerID != nil || m.Result != nil) {
				t.Fatal("live punch finalized the match")
			}
		}
		rounds := []tournaments.MatchRound{}
		// First semifinal is 2-0; second semifinal and final are 2-1.
		count := 2
		if played > 0 {
			count = 3
		}
		for n := 1; n <= count; n++ {
			r := tournaments.MatchRound{Number: n}
			blueWins := winner == *ready.Athlete1ID
			if count == 3 && n == 2 {
				blueWins = !blueWins
			}
			if blueWins {
				r.Blue.BodyKick = 2
			} else {
				r.Red.BodyKick = 2
			}
			rounds = append(rounds, r)
		}
		requestJSON(t, router, http.MethodPost, path+"/matches/"+ready.ID+"/result", token, map[string]any{"winType": "PTF", "blueId": *ready.Athlete1ID, "redId": *ready.Athlete2ID, "winnerId": winner, "rounds": rounds}, 200)
		liveAfter := decode[map[string]any](t, requestJSON(t, router, http.MethodGet, "/api/v1/public/competitions/"+event.ID+"/live", "", nil, 200))
		if _, ok := liveAfter[ready.ID]; ok {
			t.Fatal("finished match still publishes draft")
		}
		wins[nameByID[winner]]++
		if ready.NextMatchID == nil {
			gold = nameByID[winner]
			silver = nameByID[loser]
		} else {
			bronze[nameByID[loser]] = true
		}
		standings := decode[leagues.Standings](t, requestJSON(t, router, http.MethodGet, lp+"/standings", token, nil, 200))
		if len(standings.Players) != 4 || len(standings.Teams) != 4 {
			t.Fatalf("missing standings: %#v", standings)
		}
		for _, p := range standings.Players {
			want := 1 + 2*wins[p.Name]
			g, s, b := 0, 0, 0
			if played == 2 {
				if p.Name == gold {
					want += 10
					g = 1
				} else if p.Name == silver {
					want += 6
					s = 1
				} else if bronze[p.Name] {
					want += 3
					b = 1
				}
			}
			if p.TotalPoints != want || p.Gold != g || p.Silver != s || p.Bronze != b || p.Wins != wins[p.Name] {
				t.Fatalf("after match %d: unexpected player %+v; want points=%d medals=%d/%d/%d", played+1, p, want, g, s, b)
			}
			found := false
			for _, team := range standings.Teams {
				if team.Name == p.ClubName {
					found = true
					if team.TotalPoints != p.TotalPoints || team.Gold != p.Gold || team.Silver != p.Silver || team.Bronze != p.Bronze {
						t.Fatalf("team mismatch: %+v / %+v", team, p)
					}
				}
			}
			if !found {
				t.Fatalf("missing team for %s", p.Name)
			}
		}
		t.Logf("after match %d: standings=%+v", played+1, standings.Players)
		publicBytes := requestJSON(t, router, http.MethodGet, "/api/v1/public/competitions/"+event.ID, "", nil, 200)
		type publicResponse struct {
			Matches []struct {
				ID          string `json:"id"`
				MatchNumber *int   `json:"matchNumber"`
				Order       int    `json:"order"`
			} `json:"matches"`
			Standings struct {
				Players []struct {
					Name        string `json:"name"`
					TotalPoints int    `json:"totalPoints"`
				} `json:"players"`
			} `json:"standings"`
		}
		public := decode[publicResponse](t, publicBytes)
		for _, m := range public.Matches {
			found := false
			for _, original := range event.Matches {
				if original.ID == m.ID {
					found = true
					if m.Order != original.Order || !reflect.DeepEqual(m.MatchNumber, original.MatchNumber) {
						t.Fatal("public match numbering differs")
					}
				}
			}
			if !found {
				t.Fatal("unexpected public match")
			}
		}
		if len(public.Matches) != 3 || len(public.Standings.Players) != 4 {
			t.Fatalf("incomplete public competition: %s", publicBytes)
		}
		for i, p := range public.Standings.Players {
			if p.Name != standings.Players[i].Name || p.TotalPoints != standings.Players[i].TotalPoints {
				t.Fatal("public standings differ from organizer standings")
			}
		}
		for _, sensitive := range []string{`"profileId"`, `"weighIn"`, `"coach"`, `"refereeId"`, `"note"`, `"signature"`} {
			if strings.Contains(string(publicBytes), sensitive) {
				t.Fatalf("public response leaked %s", sensitive)
			}
		}
		event = decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodGet, path, token, nil, 200))
	}
	before := decode[leagues.Standings](t, requestJSON(t, router, http.MethodGet, lp+"/standings", token, nil, 200))
	for i := 0; i < 2; i++ {
		requestJSON(t, router, http.MethodPost, lp+"/tournaments/"+event.ID+"/publish", token, nil, 200)
	}
	after := decode[leagues.Standings](t, requestJSON(t, router, http.MethodGet, lp+"/standings", token, nil, 200))
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("re-publishing duplicated or changed points: before=%+v after=%+v", before, after)
	}
	for i, p := range after.Players {
		if p.Rank != i+1 {
			t.Fatalf("wrong ranking: %+v", p)
		}
	}
	if after.Players[0].TotalPoints != 15 || after.Players[1].TotalPoints != 9 || after.Players[2].TotalPoints != 4 || after.Players[3].TotalPoints != 4 {
		t.Fatalf("unexpected final standings: %+v", after.Players)
	}
	for _, tc := range []struct {
		name                         string
		medals                       []string
		crossWeek                    bool
		points, gold, silver, bronze int
	}{
		{"gold and silver", []string{"gold", "silver"}, false, 20, 1, 0, 0},
		{"silver and bronze", []string{"silver", "bronze"}, false, 16, 0, 1, 0},
		{"two bronzes", []string{"bronze"}, false, 13, 0, 0, 1},
		{"gold and bronze", []string{"gold", "bronze"}, false, 20, 1, 0, 0},
		{"gold and bronze across weeks", []string{"gold", "bronze"}, true, 23, 1, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			// Put the fixture athletes on one team; each subtest rolls back.
			if _, err = tx.Exec(ctx, `UPDATE league_point_events SET team_id=$2,club_name='Shared team' WHERE tournament_id=$1`, event.ID, league.Teams[0].ID); err != nil {
				t.Fatal(err)
			}
			if _, err = tx.Exec(ctx, `DELETE FROM league_point_events WHERE tournament_id=$1 AND event_type IN ('gold','silver','bronze') AND NOT (event_type=ANY($2::text[]))`, event.ID, tc.medals); err != nil {
				t.Fatal(err)
			}
			if tc.crossWeek {
				if _, err = tx.Exec(ctx, `UPDATE league_point_events SET tournament_id=$2 WHERE id=(SELECT id FROM league_point_events WHERE tournament_id=$1 AND event_type='bronze' ORDER BY event_key LIMIT 1)`, event.ID, nextWeek.ID); err != nil {
					t.Fatal(err)
				}
			}
			svc := leagues.NewService(tx, org, "")
			got, err := svc.Standings(ctx, league.ID, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Teams) != 1 {
				t.Fatalf("unexpected teams: %+v", got.Teams)
			}
			team := got.Teams[0]
			if team.TotalPoints != tc.points || team.Gold != tc.gold || team.Silver != tc.silver || team.Bronze != tc.bronze || team.WeighInPoints != 4 || team.WinPoints != 6 {
				t.Fatalf("incorrect best-medal team score: %+v", team)
			}
			medals := 0
			for _, player := range got.Players {
				medals += player.Gold + player.Silver + player.Bronze
			}
			wantMedals := len(tc.medals)
			for _, medal := range tc.medals {
				if medal == "bronze" {
					wantMedals++
				}
			}
			if medals != wantMedals {
				t.Fatalf("individual medals changed: got %d want %d", medals, wantMedals)
			}
		})
	}
	list := decode[struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}](t, requestJSON(t, router, http.MethodGet, "/api/v1/public/competitions?q=Four-player", "", nil, 200))
	if len(list.Items) != 1 || list.Items[0].ID != event.ID {
		t.Fatal("public competition search failed")
	}
	requestJSON(t, router, http.MethodGet, "/api/v1/public/competitions?offset=-1", "", nil, 400)
	requestJSON(t, router, http.MethodGet, "/api/v1/public/competitions/not-a-uuid", "", nil, 400)
	requestJSON(t, router, http.MethodGet, "/api/v1/public/competitions/00000000-0000-0000-0000-000000000000", "", nil, 404)
	requestJSON(t, router, http.MethodPost, path+"/draw", "", map[string]any{"type": "ranked", "weightCategory": "-54"}, 401)
	// An early-created next week must refresh seeds even after weighing in.
	nextPath := base + "/tournaments/" + nextWeek.ID
	for _, p := range nextWeek.Athletes {
		requestJSON(t, router, http.MethodPost, nextPath+"/athletes/"+p.ID+"/weigh-in", token, map[string]any{"weightKg": 54.0}, 200)
	}
	nextWeek = decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, nextPath+"/draw", token, map[string]any{"type": "ranked", "weightCategory": "-54"}, 200))
	for _, p := range nextWeek.Athletes {
		if p.Name == gold && (p.Ranking == nil || *p.Ranking != 1) {
			t.Fatalf("previous champion must be seed 1: %+v", p)
		}
		if p.Name == silver && (p.Ranking == nil || *p.Ranking != 2) {
			t.Fatalf("previous silver must be seed 2: %+v", p)
		}
	}
}
