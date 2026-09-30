// Package platform owns the HTTP boundary, identity and transaction lifetime.
package platform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"strconv"
	"strings"
	"sync"
	"time"

	"backend/internal/modules/leagues"
	"backend/internal/modules/tournaments"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type API struct {
	DB       *pgxpool.Pool
	mu       sync.Mutex
	attempts map[string]attempt
}
type attempt struct {
	Count int
	Until time.Time
}
type Request struct {
	C                            *gin.Context
	Tx                           pgx.Tx
	UserID, OrganizationID, Role string
	Permissions                  map[string]bool
	Scope                        AccessScope
}
type endpoint func(*Request) (any, error)
type httpError struct {
	Status  int
	Message string
}

func (e httpError) Error() string           { return e.Message }
func bad(message string) error              { return httpError{400, message} }
func forbidden() error                      { return httpError{403, "permission denied"} }
func notFound() error                       { return httpError{404, "resource not found"} }
func (r *Request) Context() context.Context { return r.C.Request.Context() }
func (r *Request) Service() tournaments.TournamentService {
	return tournaments.NewTournamentService(tournaments.NewTournamentRepository(r.Tx, r.OrganizationID))
}
func (r *Request) require(roles ...string) error {
	for _, role := range roles {
		if r.Role == role {
			return nil
		}
	}
	return forbidden()
}
func (r *Request) manager() error  { return r.permit(permTournamentsManage) }
func (r *Request) official() error { return r.permit(permTournamentsRead) }
func bind[T any](r *Request) (T, error) {
	var v T
	d := json.NewDecoder(r.C.Request.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		return v, bad("invalid JSON request: " + err.Error())
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return v, bad("request must contain one JSON value")
	}
	return v, nil
}
func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func (a *API) route(auth, scoped bool, fn endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<20)
		for _, p := range c.Params {
			if _, err := uuid.Parse(p.Value); err != nil {
				writeError(c, bad("invalid "+p.Key))
				return
			}
		}
		tx, err := a.DB.Begin(c.Request.Context())
		if err != nil {
			writeError(c, err)
			return
		}
		defer tx.Rollback(context.Background())
		r := &Request{C: c, Tx: tx}
		if auth {
			value := c.GetHeader("Authorization")
			if !strings.HasPrefix(value, "Bearer ") {
				writeError(c, httpError{401, "authentication required"})
				return
			}
			err = tx.QueryRow(r.Context(), `SELECT user_id::text FROM sessions WHERE token_hash=$1 AND expires_at>now()`, tokenHash(strings.TrimPrefix(value, "Bearer "))).Scan(&r.UserID)
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(c, httpError{401, "session expired or invalid"})
				return
			}
			if err != nil {
				writeError(c, err)
				return
			}
		}
		if scoped {
			r.OrganizationID = c.Param("org")
			var permissionBytes, scopeBytes []byte
			var custom bool
			err = tx.QueryRow(r.Context(), `SELECT role,permissions,scope,permissions_custom FROM memberships WHERE organization_id=$1 AND user_id=$2 AND active FOR SHARE`, r.OrganizationID, r.UserID).Scan(&r.Role, &permissionBytes, &scopeBytes, &custom)
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(c, forbidden())
				return
			}
			if err != nil {
				writeError(c, err)
				return
			}
			if err = r.loadAccess(permissionBytes, scopeBytes, custom); err != nil {
				writeError(c, err)
				return
			}
		}
		if id := c.Param("tournament"); id != "" && c.Request.Method != "GET" && c.GetHeader("If-Match") != "" {
			want, e := strconv.ParseInt(strings.Trim(c.GetHeader("If-Match"), "\" "), 10, 64)
			if e != nil || want < 1 {
				writeError(c, bad("If-Match must contain the tournament revision"))
				return
			}
			var current int64
			if e = tx.QueryRow(r.Context(), `SELECT revision FROM tournaments WHERE id=$1 AND organization_id=$2 FOR UPDATE`, id, r.OrganizationID).Scan(&current); e != nil {
				writeError(c, e)
				return
			}
			if current != want {
				writeError(c, tournaments.ErrConflict)
				return
			}
		}
		result, err := fn(r)
		if err != nil {
			writeError(c, err)
			return
		}
		if auth && c.Request.Method != "GET" {
			_, err = tx.Exec(r.Context(), `INSERT INTO audit_events(organization_id,actor_id,action,resource) VALUES($1,$2,$3,$4)`, nullable(r.OrganizationID), r.UserID, c.Request.Method, c.Request.URL.Path)
			if err != nil {
				writeError(c, err)
				return
			}
		}
		if err = tx.Commit(r.Context()); err != nil {
			writeError(c, err)
			return
		}
		if result == nil {
			c.Status(http.StatusNoContent)
		} else {
			c.JSON(http.StatusOK, result)
		}
	}
}
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func writeError(c *gin.Context, err error) {
	status, message := 500, "internal server error"
	var h httpError
	var pg *pgconn.PgError
	switch {
	case errors.As(err, &h):
		status, message = h.Status, h.Message
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, tournaments.ErrTournamentNotFound), errors.Is(err, tournaments.ErrAthleteNotFound), errors.Is(err, tournaments.ErrMatchNotFound), errors.Is(err, leagues.ErrNotFound):
		status, message = 404, "resource not found"
	case errors.Is(err, tournaments.ErrConflict):
		status, message = 409, err.Error()
	case errors.As(err, &pg):
		switch pg.Code {
		case "23505":
			status, message = 409, "resource already exists"
		case "23503", "23514", "22P02", "22007", "22008":
			status, message = 400, "invalid resource reference or field"
		case "40001", "40P01":
			status, message = 409, "concurrent update; retry request"
		}
	default:
		if errors.Is(err, leagues.ErrInvalid) {
			status, message = 400, err.Error()
		}
		if errors.Is(err, leagues.ErrRosterLocked) || errors.Is(err, leagues.ErrIncomplete) {
			status, message = 409, err.Error()
		}
		invalid := []error{tournaments.ErrInvalidTournamentInput, tournaments.ErrInvalidTournamentContext, tournaments.ErrInvalidTournamentDate, tournaments.ErrInvalidTournamentEnum, tournaments.ErrInvalidWeight, tournaments.ErrInvalidSignature, tournaments.ErrInvalidCourt, tournaments.ErrInvalidMatchPosition}
		conflicts := []error{tournaments.ErrDuplicateAthlete, tournaments.ErrAthleteLocked, tournaments.ErrMatchLocked, tournaments.ErrMaxWeighInAttempts, tournaments.ErrWeighInAlreadyPassed, tournaments.ErrBracketAlreadyDrawn, tournaments.ErrNoEligibleAthletes, tournaments.ErrResultAlreadyExists, tournaments.ErrDownstreamResult, tournaments.ErrByeMatch, tournaments.ErrWeighInNotPassed, tournaments.ErrWeighInAlreadySigned}
		for _, e := range invalid {
			if errors.Is(err, e) {
				status, message = 400, err.Error()
				break
			}
		}
		for _, e := range conflicts {
			if errors.Is(err, e) {
				status, message = 409, err.Error()
				break
			}
		}
	}
	if status == 500 {
		log.Printf("request %s %s failed: %v", c.Request.Method, c.FullPath(), err)
	}
	c.AbortWithStatusJSON(status, gin.H{"error": http.StatusText(status), "message": message})
}
func (a *API) authLimit(c *gin.Context) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	if a.attempts == nil {
		a.attempts = map[string]attempt{}
	}
	for key, v := range a.attempts {
		if now.After(v.Until) {
			delete(a.attempts, key)
		}
	}
	key := c.ClientIP()
	v := a.attempts[key]
	if v.Count >= 20 || len(a.attempts) > 10000 {
		return false
	}
	if v.Count == 0 {
		v.Until = now.Add(15 * time.Minute)
	}
	v.Count++
	a.attempts[key] = v
	return true
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("constant-dummy-password"), bcrypt.DefaultCost)

func (a *API) login(r *Request) (any, error) {
	if !a.authLimit(r.C) {
		return nil, httpError{429, "too many authentication attempts"}
	}
	in, err := bind[credentials](r)
	if err != nil {
		return nil, err
	}
	var id, hash string
	err = r.Tx.QueryRow(r.Context(), `SELECT id::text,password_hash FROM users WHERE email=$1`, strings.ToLower(strings.TrimSpace(in.Email))).Scan(&id, &hash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if hash == "" {
		hash = string(dummyHash)
	}
	if e := bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)); e != nil || id == "" {
		return nil, httpError{401, "invalid email or password"}
	}
	return newSession(r, id)
}
func newSession(r *Request, userID string) (any, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(b)
	expires := time.Now().UTC().Add(24 * time.Hour)
	_, err := r.Tx.Exec(r.Context(), `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)`, tokenHash(token), userID, expires)
	return gin.H{"token": token, "expiresAt": expires, "userId": userID}, err
}
func (a *API) Routes(router *gin.Engine) {
	api := router.Group("/api/v1")
	// Account provisioning is a server-side operation, never an HTTP operation.
	api.POST("/auth/register", func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "public registration is disabled; contact the administrator"})
	})
	api.POST("/auth/login", a.route(false, false, a.login))
	api.GET("/public/ovr", a.route(false, false, a.publicOVR))
	api.POST("/auth/logout", a.route(true, false, func(r *Request) (any, error) {
		_, err := r.Tx.Exec(r.Context(), `DELETE FROM sessions WHERE token_hash=$1`, tokenHash(strings.TrimPrefix(r.C.GetHeader("Authorization"), "Bearer ")))
		return nil, err
	}))
	api.GET("/me", a.route(true, false, func(r *Request) (any, error) {
		var id, name, email string
		err := r.Tx.QueryRow(r.Context(), `SELECT id::text,name,email FROM users WHERE id=$1`, r.UserID).Scan(&id, &name, &email)
		return gin.H{"id": id, "name": name, "email": email}, err
	}))
	a.organizationRoutes(api)
	org := api.Group("/organizations/:org")
	a.profileRoutes(org)
	a.tournamentRoutes(org)
	a.leagueRoutes(org)
}
