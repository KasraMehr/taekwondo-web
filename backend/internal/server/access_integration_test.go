package server

import (
	"net/http"
	"testing"

	"backend/internal/modules/tournaments"
	"backend/internal/testutil"
)

func TestScopedAdminCanDelegateOnlyOwnPermissionsAndCourt(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	ownerToken, org := setupAccount(t, router, pool, "rbac-owner@example.test")
	base := "/api/v1/organizations/" + org
	event := decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, base+"/tournaments", ownerToken, map[string]any{"name": "Scoped", "date": "2026-10-01", "courts": 2, "gender": "male", "ageCategory": "بزرگسالان", "format": "grandPrix"}, 200))
	other := decode[tournaments.Tournament](t, requestJSON(t, router, http.MethodPost, base+"/tournaments", ownerToken, map[string]any{"name": "Other", "date": "2026-10-02", "courts": 2, "gender": "male", "ageCategory": "بزرگسالان", "format": "grandPrix"}, 200))
	password := "scoped-admin-password"
	requestJSON(t, router, http.MethodPost, base+"/members/accounts", ownerToken, map[string]any{"name": "Court A Admin", "email": "court-a@example.test", "password": password, "role": "admin", "permissions": []string{"members.manage", "tournaments.read", "sheets.ta"}, "scope": map[string]any{"tournamentIds": []string{event.ID}, "courts": []string{"A"}, "stations": []string{"ta"}}}, 200)
	login := decode[map[string]any](t, requestJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]any{"email": "court-a@example.test", "password": password}, 200))
	token := login["token"].(string)
	requestJSON(t, router, http.MethodGet, base+"/tournaments/"+event.ID, token, nil, 200)
	requestJSON(t, router, http.MethodGet, base+"/tournaments/"+other.ID, token, nil, 403)
	requestJSON(t, router, http.MethodGet, base+"/tournaments/"+event.ID+"/sheets/ta?court=A", token, nil, 200)
	requestJSON(t, router, http.MethodGet, base+"/tournaments/"+event.ID+"/sheets/ta?court=B", token, nil, 403)
	requestJSON(t, router, http.MethodPost, base+"/members/accounts", token, map[string]any{"name": "Child Admin", "email": "child-admin@example.test", "password": "child-admin-password", "role": "admin", "permissions": []string{"sheets.ta"}, "scope": map[string]any{"tournamentIds": []string{event.ID}, "courts": []string{"A"}, "stations": []string{"ta"}}}, 200)
	requestJSON(t, router, http.MethodPost, base+"/members/accounts", token, map[string]any{"name": "Escalation", "email": "escalation@example.test", "password": "escalation-password", "role": "admin", "permissions": []string{"athletes.manage"}}, 403)
}
