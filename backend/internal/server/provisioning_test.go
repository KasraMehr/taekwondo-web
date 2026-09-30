package server

import (
	"backend/internal/provision"
	"backend/internal/testutil"
	"context"
	"net/http"
	"testing"
)

func TestPublicProvisioningIsClosedWithoutDatabase(t *testing.T) {
	router := NewRouter(nil)
	for _, path := range []string{"/api/v1/auth/register", "/api/v1/organizations"} {
		requestJSON(t, router, http.MethodPost, path, "", map[string]string{}, 403)
	}
}

func TestProvisionedOwnerCanLoginButCannotCreateOrganizationOverHTTP(t *testing.T) {
	pool := testutil.Database(t)
	router := NewRouter(pool)
	token, _ := setupAccount(t, router, pool, "provision@example.test")
	requestJSON(t, router, http.MethodPost, "/api/v1/auth/register", token, map[string]string{}, 403)
	requestJSON(t, router, http.MethodPost, "/api/v1/organizations", token, map[string]string{"name": "Unauthorized"}, 403)
	requestJSON(t, router, http.MethodGet, "/api/v1/organizations", token, nil, 200)
	_, err := provision.CreateAdmin(context.Background(), pool, "Replacement", "provision@example.test", "replacement-password", "Extra")
	if err == nil {
		t.Fatal("duplicate owner must fail without changing password or creating another organization")
	}
	requestJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "provision@example.test", "password": "integration-only-password"}, 200)
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM organizations").Scan(&count); err != nil || count != 1 {
		t.Fatalf("organizations = %d, error = %v", count, err)
	}
}
