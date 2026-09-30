// Package provision creates owners through trusted database access, never HTTP.
package provision

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func CreateAdmin(ctx context.Context, pool *pgxpool.Pool, name, email, password, organization string) (string, error) {
	name, email, organization = strings.TrimSpace(name), strings.ToLower(strings.TrimSpace(email)), strings.TrimSpace(organization)
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 || name == "" || len(name) > 100 || organization == "" || len(organization) > 150 || len(password) < 10 || len(password) > 72 {
		return "", errors.New("provide a valid email, name (1-100 bytes), organization (1-150 bytes), and password (10-72 bytes)")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(context.Background())
	var userID, orgID string
	// Existing accounts are never silently promoted or assigned a new password.
	err = tx.QueryRow(ctx, `INSERT INTO users(email,name,password_hash) VALUES($1,$2,$3) RETURNING id::text`, email, name, string(hash)).Scan(&userID)
	if err != nil {
		return "", err
	}
	err = tx.QueryRow(ctx, `INSERT INTO organizations(name) VALUES($1) RETURNING id::text`, organization).Scan(&orgID)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO memberships(organization_id,user_id,role) VALUES($1,$2,'owner')`, orgID, userID)
	if err != nil {
		return "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return orgID, nil
}
