package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	p "backend/internal/modules/poomsae"
	"backend/internal/testutil"
	"github.com/google/uuid"
)

func poomsaeRequest(h http.Handler, method, path, token string, body any, revision int64) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if revision > 0 {
		req.Header.Set("If-Match", fmt.Sprintf(`"%d"`, revision))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}
func poomsaeJSON(t *testing.T, h http.Handler, method, path, token string, body any, rev int64, want int) []byte {
	t.Helper()
	w := poomsaeRequest(h, method, path, token, body, rev)
	if w.Code != want {
		t.Fatalf("%s %s: got %d want %d: %s", method, path, w.Code, want, w.Body.String())
	}
	return w.Body.Bytes()
}
func poomsaeEventBody() map[string]any {
	return map[string]any{"name": "جام پومسه", "date": "2026-10-03", "rules": map[string]any{"accuracyMax": "3.00", "presentationMax": "7.00", "precision": 2, "tiePolicy": "shared", "formSelection": "division", "allowRepeatedForm": false, "version": "test rules"}}
}
func poomsaeDivisionBody() map[string]any {
	return map[string]any{"name": "زیر ۱۲", "competitionType": "individual", "gender": "male", "birthDateFrom": "2015-01-01", "birthDateTo": "2020-12-31", "allowedFormCodes": []int{2, 3, 5}, "form1Code": 2, "form2Code": 3}
}
func poomsaeScores() map[string]any {
	return map[string]any{"form1": map[string]any{"accuracy": "2.50", "presentation": "6.00"}, "form2": map[string]any{"accuracy": "2.60", "presentation": "6.10"}}
}

func TestPostgresPoomsaeLifecycle(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "poomsae@example.test")
	base := "/api/v1/organizations/" + org
	events := base + "/poomsae-events"
	event := decode[p.Event](t, poomsaeJSON(t, router, "POST", events, token, poomsaeEventBody(), 0, 200))
	eventPath := events + "/" + event.ID
	poomsaeJSON(t, router, "POST", eventPath+"/divisions", token, poomsaeDivisionBody(), 0, 428)
	created := decode[struct {
		Division      p.Division `json:"division"`
		EventRevision int64      `json:"eventRevision"`
	}](t, poomsaeJSON(t, router, "POST", eventPath+"/divisions", token, poomsaeDivisionBody(), event.Revision, 200))
	d := created.Division
	path := eventPath + "/divisions/" + d.ID
	if created.EventRevision != 2 {
		t.Fatal("event revision not advanced")
	}
	poomsaeJSON(t, router, "POST", eventPath+"/divisions", token, poomsaeDivisionBody(), 1, 409)
	changedRules := poomsaeEventBody()
	changedRules["date"] = "2026-10-04"
	poomsaeJSON(t, router, "PUT", eventPath, token, changedRules, 2, 409)
	poomsaeJSON(t, router, "POST", path+"/draw", token, map[string]any{}, d.Revision, 409)
	var firstBody map[string]any
	for i := 0; i < 3; i++ {
		profile := decode[map[string]any](t, requestJSON(t, router, "POST", base+"/athletes", token, map[string]any{"name": fmt.Sprintf("Athlete %d", i), "gender": "male", "birthDate": "2016-01-01"}, 200))
		body := map[string]any{"athleteId": profile["id"], "firstName": fmt.Sprint("علی", i), "lastName": "نمونه", "teamName": "باشگاه", "birthDate": "2016-01-01", "gender": "male"}
		if i == 0 {
			firstBody = body
		}
		d = decode[p.Division](t, poomsaeJSON(t, router, "POST", path+"/entries", token, body, d.Revision, 200))
	}
	poomsaeJSON(t, router, "POST", path+"/entries", token, firstBody, d.Revision, 409)
	d = decode[p.Division](t, poomsaeJSON(t, router, "POST", path+"/draw", token, map[string]any{}, d.Revision, 200))
	original := append([]string{}, d.ActiveDraw().EntryIDs...)
	reloaded := decode[p.Division](t, poomsaeJSON(t, router, "GET", path, token, nil, 0, 200))
	if !reflect.DeepEqual(original, reloaded.ActiveDraw().EntryIDs) {
		t.Fatal("draw lost on reload")
	}
	poomsaeJSON(t, router, "POST", path+"/finalize", token, nil, d.Revision, 409)
	for i := 0; i < 2; i++ {
		d = decode[p.Division](t, poomsaeJSON(t, router, "PUT", path+"/entries/"+original[i]+"/scores", token, poomsaeScores(), d.Revision, 200))
	}
	poomsaeJSON(t, router, "POST", path+"/draw", token, map[string]any{"reason": "redraw"}, d.Revision, 409)
	poomsaeJSON(t, router, "POST", path+"/invalidate-draw", token, map[string]any{"reason": "reset"}, d.Revision, 409)
	third, _ := d.Entry(original[2])
	absent := third.EntryInput
	absent.Status = "absent"
	absent.Reason = "did not attend"
	d = decode[p.Division](t, poomsaeJSON(t, router, "PUT", path+"/entries/"+third.ID, token, absent, d.Revision, 200))
	standings := decode[p.Standings](t, poomsaeJSON(t, router, "GET", path+"/standings", token, nil, 0, 200))
	if len(standings.Ranked) != 2 || len(standings.Unranked) != 1 || *standings.Ranked[0].Total != 1720 || *standings.Ranked[1].Rank != 1 {
		t.Fatalf("bad standings: %+v", standings)
	}
	blank := decode[struct {
		Rows []p.Result `json:"rows"`
	}](t, poomsaeJSON(t, router, "GET", path+"/sheet?mode=blank", token, nil, 0, 200))
	for _, row := range blank.Rows {
		if row.Total != nil || p.HasScores(row.Entry.Scores) || row.Rank != nil {
			t.Fatal("blank sheet contains scores")
		}
	}
	d = decode[p.Division](t, poomsaeJSON(t, router, "POST", path+"/finalize", token, nil, d.Revision, 200))
	poomsaeJSON(t, router, "PUT", path+"/entries/"+original[0]+"/scores", token, poomsaeScores(), d.Revision, 409)
	d = decode[p.Division](t, poomsaeJSON(t, router, "POST", path+"/reopen", token, map[string]any{"reason": "review score"}, d.Revision, 200))
	correction := poomsaeScores()
	correction["form1"] = map[string]any{"accuracy": "2.00", "presentation": "6.00"}
	poomsaeJSON(t, router, "PUT", path+"/entries/"+original[0]+"/scores", token, correction, d.Revision, 400)
	correction["reason"] = "transcription error"
	d = decode[p.Division](t, poomsaeJSON(t, router, "PUT", path+"/entries/"+original[0]+"/scores", token, correction, d.Revision, 200))
	standings = decode[p.Standings](t, poomsaeJSON(t, router, "GET", path+"/standings", token, nil, 0, 200))
	if standings.Ranked[1].Entry.ID != original[0] || *standings.Ranked[1].Rank != 2 {
		t.Fatal("correction did not update ranking")
	}
	history := decode[[]map[string]any](t, poomsaeJSON(t, router, "GET", path+"/history", token, nil, 0, 200))
	if history[0]["reason"] != "transcription error" || history[0]["before"] == nil {
		t.Fatal("correction history missing")
	}
	var auditCount int
	err := pool.QueryRow(context.Background(), `SELECT count(*) FROM poomsae_change_log WHERE division_id=$1`, d.ID).Scan(&auditCount)
	if err != nil || auditCount != int(d.Revision) {
		t.Fatalf("failed requests changed audit/revision: %d %d %v", auditCount, d.Revision, err)
	}
	// Concurrent score submissions with the same revision: exactly one commits.
	body := poomsaeScores()
	body["reason"] = "concurrent correction"
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- poomsaeRequest(router, "PUT", path+"/entries/"+original[0]+"/scores", token, body, d.Revision).Code
		}()
	}
	wg.Wait()
	close(codes)
	counts := map[int]int{}
	for code := range codes {
		counts[code]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatalf("concurrent writes: %v", counts)
	}
}

func TestPostgresPoomsaeAuthorizationAndRollback(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, org := setupAccount(t, router, pool, "poomsae-owner@example.test")
	other, otherOrg := setupAccount(t, router, pool, "poomsae-other@example.test")
	base := "/api/v1/organizations/" + org
	events := base + "/poomsae-events"
	event := decode[p.Event](t, poomsaeJSON(t, router, "POST", events, token, poomsaeEventBody(), 0, 200))
	ep := events + "/" + event.ID
	created := decode[struct {
		Division p.Division `json:"division"`
	}](t, poomsaeJSON(t, router, "POST", ep+"/divisions", token, poomsaeDivisionBody(), 1, 200))
	d := created.Division
	path := ep + "/divisions/" + d.ID
	poomsaeJSON(t, router, "GET", path, "", nil, 0, 401)
	poomsaeJSON(t, router, "GET", path, other, nil, 0, 403)
	poomsaeJSON(t, router, "GET", "/api/v1/organizations/"+otherOrg+"/poomsae-events/"+event.ID, other, nil, 0, 404)
	foreign := decode[map[string]any](t, requestJSON(t, router, "POST", "/api/v1/organizations/"+otherOrg+"/athletes", other, map[string]any{"name": "foreign", "gender": "male"}, 200))
	body := map[string]any{"athleteId": foreign["id"], "firstName": "x", "lastName": "y", "teamName": "z", "birthDate": "2016-01-01", "gender": "male"}
	poomsaeJSON(t, router, "POST", path+"/entries", token, body, d.Revision, 400)
	reloaded := decode[p.Division](t, poomsaeJSON(t, router, "GET", path, token, nil, 0, 200))
	if reloaded.Revision != 1 || len(reloaded.Entries) != 0 {
		t.Fatal("failed registration was persisted")
	}
	// Add the second user as a referee assigned to exactly this division.
	ctx := context.Background()
	var user string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM users WHERE email='poomsae-other@example.test'`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	scope, _ := json.Marshal(map[string]any{"poomsaeEventIds": []string{event.ID}, "poomsaeDivisionIds": []string{d.ID}})
	if _, err := pool.Exec(ctx, `INSERT INTO memberships(organization_id,user_id,role,scope) VALUES($1,$2,'referee',$3)`, org, user, scope); err != nil {
		t.Fatal(err)
	}
	poomsaeJSON(t, router, "GET", path, other, nil, 0, 200)
	poomsaeJSON(t, router, "POST", path+"/draw", other, map[string]any{}, 1, 403)
	poomsaeJSON(t, router, "POST", path+"/reopen", other, map[string]any{"reason": "test"}, 1, 403)
	poomsaeJSON(t, router, "GET", ep+"/divisions/"+uuid.NewString(), other, nil, 0, 403)
	listed := decode[[]map[string]any](t, poomsaeJSON(t, router, "GET", events, other, nil, 0, 200))
	if len(listed) != 1 {
		t.Fatal("assigned event missing")
	}
	scope, _ = json.Marshal(map[string]any{"tournamentIds": []string{uuid.NewString()}})
	if _, err := pool.Exec(ctx, `UPDATE memberships SET scope=$3 WHERE organization_id=$1 AND user_id=$2`, org, user, scope); err != nil {
		t.Fatal(err)
	}
	poomsaeJSON(t, router, "GET", path, other, nil, 0, 403)
	listed = decode[[]map[string]any](t, poomsaeJSON(t, router, "GET", events, other, nil, 0, 200))
	if len(listed) != 0 {
		t.Fatal("legacy scoped referee inherited global poomsae access")
	}
}
