package server

import (
	"backend/internal/modules/tournaments"
	"backend/internal/provision"
	"backend/internal/testutil"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func requestJSON(t *testing.T, h http.Handler, method, path, token string, body any, want int) []byte {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	h.ServeHTTP(response, req)
	if response.Code != want {
		t.Fatalf("%s %s: got %d want %d: %s", method, path, response.Code, want, response.Body.String())
	}
	return response.Body.Bytes()
}
func decode[T any](t *testing.T, b []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(b, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
func setupAccount(t *testing.T, h http.Handler, pool *pgxpool.Pool, email string) (string, string) {
	t.Helper()
	org, err := provision.CreateAdmin(context.Background(), pool, "Organizer", email, "integration-only-password", "Test organization")
	if err != nil {
		t.Fatal(err)
	}
	auth := decode[map[string]any](t, requestJSON(t, h, "POST", "/api/v1/auth/login", "", map[string]any{"email": email, "password": "integration-only-password"}, 200))
	return auth["token"].(string), org
}
func TestPostgresTournamentHTTPFlow(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "organizer@example.test")
	otherToken, otherOrg := setupAccount(t, router, pool, "other@example.test")
	base := "/api/v1/organizations/" + org
	event := decode[tournaments.Tournament](t, requestJSON(t, router, "POST", base+"/tournaments", token, map[string]any{"name": "جام", "date": "2026-10-01", "courts": 4, "gender": "male", "ageCategory": "بزرگسالان", "format": "grandPrix"}, 200))
	path := base + "/tournaments/" + event.ID
	requestJSON(t, router, "GET", path, "", nil, 401)
	requestJSON(t, router, "GET", path, otherToken, nil, 403)
	requestJSON(t, router, "GET", "/api/v1/organizations/"+otherOrg+"/tournaments/"+event.ID, otherToken, nil, 404)
	var firstProfile string
	for i := 0; i < 4; i++ {
		p := decode[map[string]any](t, requestJSON(t, router, "POST", base+"/athletes", token, map[string]any{"name": fmt.Sprint("Athlete ", i), "gender": "male"}, 200))
		pid := p["id"].(string)
		if i == 0 {
			firstProfile = pid
		}
		entry := decode[tournaments.TournamentAthlete](t, requestJSON(t, router, "POST", path+"/athletes", token, map[string]any{"profileId": pid, "weightCategory": "-54", "ranking": i + 1}, 200))
		if entry.ProfileID == nil || *entry.ProfileID != pid {
			t.Fatal("profile not linked")
		}
		requestJSON(t, router, "POST", path+"/athletes/"+entry.ID+"/weigh-in", token, map[string]any{"weightKg": 54.2}, 200)
	}
	requestJSON(t, router, "POST", path+"/numbering", token, nil, 409)
	requestJSON(t, router, "POST", "/api/v1/organizations/"+otherOrg+"/tournaments/"+event.ID+"/athletes", otherToken, map[string]any{"profileId": firstProfile, "weightCategory": "-54"}, 404)
	requestJSON(t, router, "POST", path+"/draw", token, map[string]any{"type": "ranked"}, 200)
	stored := decode[tournaments.Tournament](t, requestJSON(t, router, "GET", path, token, nil, 200))
	if len(stored.Matches) != 3 || stored.Settings == nil {
		t.Fatal("aggregate round trip lost data")
	}
	for _, m := range stored.Matches {
		if m.MatchNumber != nil {
			t.Fatal("draw already numbered")
		}
	}
	// A stale UI must not overwrite a newer version when using If-Match.
	stale := httptest.NewRequest("POST", path+"/numbering", nil)
	stale.Header.Set("Authorization", "Bearer "+token)
	stale.Header.Set("If-Match", "1")
	staleResponse := httptest.NewRecorder()
	router.ServeHTTP(staleResponse, stale)
	if staleResponse.Code != 409 {
		t.Fatalf("stale revision accepted: %d", staleResponse.Code)
	}
	numbered := decode[tournaments.Tournament](t, requestJSON(t, router, "POST", path+"/numbering", token, nil, 200))
	if len(numbered.Athletes) != 4 || numbered.Matches[0].MatchNumber == nil {
		t.Fatal("numbering failed")
	}
	// Two officials cannot start two matches on the same court concurrently.
	playable := []tournaments.Match{}
	for _, m := range numbered.Matches {
		if m.Round == 1 {
			playable = append(playable, m)
		}
	}
	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	for _, m := range playable {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			req := httptest.NewRequest("POST", path+"/matches/"+id+"/start", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			statuses <- rec.Code
		}(m.ID)
	}
	wg.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatalf("concurrent court starts: %v", counts)
	}
	requestJSON(t, router, "POST", path+"/numbering", token, nil, 409)
	var links int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM tournament_entries WHERE tournament_id=$1 AND profile_id IS NOT NULL`, event.ID).Scan(&links); err != nil || links != 4 {
		t.Fatalf("relational links %d: %v", links, err)
	}
}

func TestPostgresSettingsPersistAndRollback(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "settings@example.test")
	base := "/api/v1/organizations/" + org + "/tournaments"
	event := decode[tournaments.Tournament](t, requestJSON(t, router, "POST", base, token, map[string]any{"name": "Two days", "date": "2026-10-01", "courts": 3, "gender": "male", "ageCategory": "بزرگسالان", "format": "grandPrix"}, 200))
	path := base + "/" + event.ID
	settings := *event.Settings
	settings.Schedule.Mode = "split_halves"
	settings.Schedule.DayAssignment = "alternating"
	settings.Schedule.Days = append(settings.Schedule.Days, tournaments.EventDay{Day: 2, Date: "2026-10-02"})
	requestJSON(t, router, "PUT", path+"/settings", token, settings, 200)
	saved := decode[tournaments.TournamentSettings](t, requestJSON(t, router, "GET", path+"/settings", token, nil, 200))
	if saved.Schedule.CategoryDays["-54"] != 1 || saved.Schedule.CategoryDays["-58"] != 2 {
		t.Fatal("day assignment not persisted")
	}
	settings.Schedule.Days[1].Date = "2026-09-01"
	requestJSON(t, router, "PUT", path+"/settings", token, settings, 400)
	after := decode[tournaments.TournamentSettings](t, requestJSON(t, router, "GET", path+"/settings", token, nil, 200))
	if after.Schedule.Days[1].Date != "2026-10-02" {
		t.Fatal("invalid update was not rolled back")
	}
}
