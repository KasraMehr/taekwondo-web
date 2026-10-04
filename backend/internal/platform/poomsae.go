package platform

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"

	p "backend/internal/modules/poomsae"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	permPoomsaeRead     = "poomsae.read"
	permPoomsaeManage   = "poomsae.manage"
	permPoomsaeDraw     = "poomsae.draw"
	permPoomsaeScore    = "poomsae.score"
	permPoomsaeOverride = "poomsae.override"
)

func init() {
	all := []string{permPoomsaeRead, permPoomsaeManage, permPoomsaeDraw, permPoomsaeScore, permPoomsaeOverride}
	for _, v := range all {
		validPermissions[v] = true
	}
	for _, role := range []string{"admin", "organizer"} {
		rolePermissions[role] = append(rolePermissions[role], all...)
	}
	rolePermissions["referee"] = append(rolePermissions["referee"], permPoomsaeRead, permPoomsaeScore)
}
func (r *Request) poomsaeRepo() p.Repository {
	return p.Repository{Tx: r.Tx, OrganizationID: r.OrganizationID}
}
func (r *Request) poomsaeLegacyRestricted() bool {
	s := r.Scope
	return len(s.PoomsaeEventIDs) == 0 && len(s.PoomsaeDivisionIDs) == 0 && (len(s.TournamentIDs) > 0 || len(s.Courts) > 0 || len(s.Stations) > 0)
}
func (r *Request) poomsaeScope(event, division string) error {
	if r.poomsaeLegacyRestricted() {
		return forbidden()
	}
	if len(r.Scope.PoomsaeEventIDs) > 0 && !containsFold(r.Scope.PoomsaeEventIDs, event) {
		return forbidden()
	}
	if division != "" && len(r.Scope.PoomsaeDivisionIDs) > 0 && !containsFold(r.Scope.PoomsaeDivisionIDs, division) {
		return forbidden()
	}
	return nil
}
func requirePoomsaeVersion(r *Request, rev int64) error {
	raw := r.C.GetHeader("If-Match")
	if raw == "" {
		return httpError{428, "If-Match revision is required"}
	}
	n, err := strconv.ParseInt(strings.Trim(raw, "\""), 10, 64)
	if err != nil || n < 1 {
		return bad("invalid If-Match revision")
	}
	if n != rev {
		return httpError{409, "resource changed; reload and retry"}
	}
	return nil
}
func poomsaePage(r *Request) (int, int, error) {
	limit, offset := 50, 0
	var err error
	if v := r.C.Query("limit"); v != "" {
		limit, err = strconv.Atoi(v)
		if err != nil || limit < 1 || limit > 200 {
			return 0, 0, bad("limit must be 1..200")
		}
	}
	if v := r.C.Query("offset"); v != "" {
		offset, err = strconv.Atoi(v)
		if err != nil || offset < 0 {
			return 0, 0, bad("invalid offset")
		}
	}
	return limit, offset, nil
}
func (r *Request) loadPoomsaeEvent(permission string) (*p.Event, error) {
	if err := r.permit(permission); err != nil {
		return nil, err
	}
	id := r.C.Param("event")
	if err := r.poomsaeScope(id, r.C.Param("division")); err != nil {
		return nil, err
	}
	e, err := r.poomsaeRepo().Event(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if len(r.Scope.PoomsaeDivisionIDs) > 0 {
		var ok bool
		err = r.Tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM poomsae_divisions WHERE organization_id=$1 AND event_id=$2 AND id=ANY($3::uuid[]))`, r.OrganizationID, id, r.Scope.PoomsaeDivisionIDs).Scan(&ok)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, forbidden()
		}
	}
	return e, nil
}

type poomsaeOperation func(*Request, *p.Event, *p.Division) (any, string, error)

func poomsaeDivisionResponse(d *p.Division) any {
	return struct {
		*p.Division
		Results   []p.Result  `json:"results"`
		Standings p.Standings `json:"standings"`
	}{d, d.Results(), d.Standings()}
}

func (a *API) poomsaeDivisionRoute(permission string, write bool, fn poomsaeOperation) gin.HandlerFunc {
	return a.route(true, true, func(r *Request) (any, error) {
		e, err := r.loadPoomsaeEvent(permission)
		if err != nil {
			return nil, err
		}
		repo := r.poomsaeRepo()
		d, err := repo.Division(r.Context(), e.ID, r.C.Param("division"))
		if err != nil {
			return nil, err
		}
		var before json.RawMessage
		if write {
			if err = requirePoomsaeVersion(r, d.Revision); err != nil {
				return nil, err
			}
			before, err = json.Marshal(d)
			if err != nil {
				return nil, err
			}
		}
		result, reason, err := fn(r, e, d)
		if err != nil {
			return nil, err
		}
		if write {
			if err = repo.SaveDivision(r.Context(), d); err != nil {
				return nil, err
			}
			if err = repo.Audit(r.Context(), e.ID, d.ID, r.UserID, r.C.Request.Method+" "+r.C.FullPath(), reason, before, d); err != nil {
				return nil, err
			}
			return poomsaeDivisionResponse(d), nil
		}
		return result, nil
	})
}
func (a *API) poomsaeRoutes(org *gin.RouterGroup) {
	org.GET("/poomsae-settings-options", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permPoomsaeRead); err != nil {
			return nil, err
		}
		return gin.H{"competitionTypes": []string{"individual"}, "categories": p.CategoryTemplates(), "categorySource": "user-supplied image; verify before use", "requiresExplicitBirthBounds": true, "precision": 2, "tiePolicies": []string{"shared"}, "formSelections": []string{"division", "entry"}, "permissions": gin.H{
			"manage": r.hasPermission(permPoomsaeManage), "draw": r.hasPermission(permPoomsaeDraw), "score": r.hasPermission(permPoomsaeScore), "override": r.hasPermission(permPoomsaeOverride), "audit": r.hasPermission(permAuditRead),
			"createEvent":  r.hasPermission(permPoomsaeManage) && !r.poomsaeLegacyRestricted() && len(r.Scope.PoomsaeEventIDs) == 0 && len(r.Scope.PoomsaeDivisionIDs) == 0,
			"manageEvent":  r.hasPermission(permPoomsaeManage) && len(r.Scope.PoomsaeDivisionIDs) == 0,
			"athletesRead": r.hasPermission(permAthletesRead), "athletesManage": r.hasPermission(permAthletesManage),
		}}, nil
	}))
	group := org.Group("/poomsae-events")
	group.POST("", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permPoomsaeManage); err != nil {
			return nil, err
		}
		if r.poomsaeLegacyRestricted() || len(r.Scope.PoomsaeEventIDs) > 0 || len(r.Scope.PoomsaeDivisionIDs) > 0 {
			return nil, forbidden()
		}
		in, err := bind[p.EventInput](r)
		if err != nil {
			return nil, err
		}
		if err = p.ValidateEvent(in); err != nil {
			return nil, err
		}
		e := p.Event{ID: uuid.NewString(), EventInput: in, Revision: 1, CreatedAt: time.Now().UTC()}
		repo := r.poomsaeRepo()
		if err = repo.CreateEvent(r.Context(), &e); err != nil {
			return nil, err
		}
		return e, repo.Audit(r.Context(), e.ID, "", r.UserID, "event.create", "", nil, e)
	}))
	group.GET("", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permPoomsaeRead); err != nil {
			return nil, err
		}
		if r.poomsaeLegacyRestricted() {
			return []any{}, nil
		}
		limit, offset, err := poomsaePage(r)
		if err != nil {
			return nil, err
		}
		return listJSON(r, `SELECT e.data || jsonb_build_object('revision',e.revision) FROM poomsae_events e WHERE e.organization_id=$1
   AND ($2::uuid[] IS NULL OR e.id=ANY($2)) AND ($3::uuid[] IS NULL OR EXISTS(SELECT 1 FROM poomsae_divisions d WHERE d.event_id=e.id AND d.id=ANY($3)))
   ORDER BY e.created_at DESC,e.id LIMIT $4 OFFSET $5`, r.OrganizationID, nonemptyIDs(r.Scope.PoomsaeEventIDs), nonemptyIDs(r.Scope.PoomsaeDivisionIDs), limit, offset)
	}))
	group.GET("/:event", a.route(true, true, func(r *Request) (any, error) { return r.loadPoomsaeEvent(permPoomsaeRead) }))
	updateEvent := a.route(true, true, func(r *Request) (any, error) {
		e, err := r.loadPoomsaeEvent(permPoomsaeManage)
		if err != nil {
			return nil, err
		}
		if len(r.Scope.PoomsaeDivisionIDs) > 0 {
			return nil, forbidden()
		}
		if err = requirePoomsaeVersion(r, e.Revision); err != nil {
			return nil, err
		}
		in, err := bind[p.EventInput](r)
		if err != nil {
			return nil, err
		}
		if err = p.ValidateEvent(in); err != nil {
			return nil, err
		}
		// Rules/date affect every participant: once any division exists, preserve them.
		if in.Date != e.Date || !reflect.DeepEqual(in.Rules, e.Rules) {
			var exists bool
			err = r.Tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM poomsae_divisions WHERE event_id=$1)`, e.ID).Scan(&exists)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, httpError{409, "date and rules are locked after creating a division"}
			}
		}
		before := *e
		e.EventInput = in
		repo := r.poomsaeRepo()
		if err = repo.SaveEvent(r.Context(), e); err != nil {
			return nil, err
		}
		return e, repo.Audit(r.Context(), e.ID, "", r.UserID, "event.update", "", before, e)
	})
	group.PUT("/:event", updateEvent)
	group.GET("/:event/divisions", a.route(true, true, func(r *Request) (any, error) {
		e, err := r.loadPoomsaeEvent(permPoomsaeRead)
		if err != nil {
			return nil, err
		}
		limit, offset, err := poomsaePage(r)
		if err != nil {
			return nil, err
		}
		return listJSON(r, `SELECT (data-'entries'-'draws') || jsonb_build_object('revision',revision) FROM poomsae_divisions WHERE organization_id=$1 AND event_id=$2 AND ($3::uuid[] IS NULL OR id=ANY($3)) ORDER BY id LIMIT $4 OFFSET $5`, r.OrganizationID, e.ID, nonemptyIDs(r.Scope.PoomsaeDivisionIDs), limit, offset)
	}))
	group.POST("/:event/divisions", a.route(true, true, func(r *Request) (any, error) {
		e, err := r.loadPoomsaeEvent(permPoomsaeManage)
		if err != nil {
			return nil, err
		}
		if len(r.Scope.PoomsaeDivisionIDs) > 0 {
			return nil, forbidden()
		}
		if err = requirePoomsaeVersion(r, e.Revision); err != nil {
			return nil, err
		}
		in, err := bind[p.DivisionInput](r)
		if err != nil {
			return nil, err
		}
		if err = p.ValidateDivision(*e, in); err != nil {
			return nil, err
		}
		d := p.Division{ID: uuid.NewString(), EventID: e.ID, DivisionInput: in, Status: "draft", Revision: 1, Entries: []p.Entry{}, Draws: []p.Draw{}}
		repo := r.poomsaeRepo()
		if err = repo.CreateDivision(r.Context(), &d); err != nil {
			return nil, err
		}
		if err = repo.SaveEvent(r.Context(), e); err != nil {
			return nil, err
		}
		return gin.H{"division": d, "eventRevision": e.Revision}, repo.Audit(r.Context(), e.ID, d.ID, r.UserID, "division.create", "", nil, d)
	}))
	path := "/:event/divisions/:division"
	group.GET(path, a.poomsaeDivisionRoute(permPoomsaeRead, false, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		return poomsaeDivisionResponse(d), "", nil
	}))
	group.PUT(path, a.poomsaeDivisionRoute(permPoomsaeManage, true, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		if d.Status != "draft" || d.EverScored {
			return nil, "", httpError{409, "division settings are locked after draw"}
		}
		in, err := bind[p.DivisionInput](r)
		if err != nil {
			return nil, "", err
		}
		if err = p.ValidateDivision(*e, in); err != nil {
			return nil, "", err
		}
		d.DivisionInput = in
		for _, entry := range d.Entries {
			if err = d.ValidateEntry(*e, entry.EntryInput); err != nil {
				return nil, "", err
			}
		}
		return nil, "", nil
	}))
	group.GET(path+"/entries", a.poomsaeDivisionRoute(permPoomsaeRead, false, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		limit, offset, err := poomsaePage(r)
		if err != nil {
			return nil, "", err
		}
		items := []p.Entry{}
		search := strings.ToLower(strings.TrimSpace(r.C.Query("search")))
		team := r.C.Query("team")
		for _, entry := range d.Entries {
			if (search == "" || strings.Contains(strings.ToLower(entry.FirstName+" "+entry.LastName+" "+entry.TeamName), search)) && (team == "" || entry.TeamName == team) {
				items = append(items, entry)
			}
		}
		total := len(items)
		if offset > total {
			offset = total
		}
		end := offset + limit
		if end > total {
			end = total
		}
		return gin.H{"items": items[offset:end], "total": total, "revision": d.Revision}, "", nil
	}))
	group.POST(path+"/entries", a.poomsaeDivisionRoute(permPoomsaeManage, true, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		in, err := bind[p.EntryInput](r)
		if err != nil {
			return nil, "", err
		}
		if _, err = d.AddEntry(*e, in); err != nil {
			return nil, "", err
		}
		return nil, "", r.poomsaeRepo().ValidateProfile(r.Context(), in)
	}))
	group.PUT(path+"/entries/:entry", a.poomsaeDivisionRoute(permPoomsaeManage, true, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		in, err := bind[p.EntryInput](r)
		if err != nil {
			return nil, "", err
		}
		if err = d.UpdateEntry(*e, r.C.Param("entry"), in); err != nil {
			return nil, "", err
		}
		return nil, in.Reason, r.poomsaeRepo().ValidateProfile(r.Context(), in)
	}))
	group.DELETE(path+"/entries/:entry", a.poomsaeDivisionRoute(permPoomsaeManage, true, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		return nil, "", d.DeleteEntry(r.C.Param("entry"))
	}))
	group.GET(path+"/draw", a.poomsaeDivisionRoute(permPoomsaeRead, false, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		return gin.H{"draw": d.ActiveDraw(), "revision": d.Revision}, "", nil
	}))
	group.POST(path+"/draw", a.poomsaeDivisionRoute(permPoomsaeDraw, true, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		in, err := bind[p.ReasonInput](r)
		if err != nil {
			return nil, "", err
		}
		return nil, in.Reason, d.Draw(r.UserID, in.Reason)
	}))
	group.POST(path+"/invalidate-draw", a.poomsaeDivisionRoute(permPoomsaeDraw, true, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		in, err := bind[p.ReasonInput](r)
		if err != nil {
			return nil, "", err
		}
		return nil, in.Reason, d.InvalidateDraw(in.Reason)
	}))
	group.PUT(path+"/entries/:entry/scores", a.poomsaeDivisionRoute(permPoomsaeScore, true, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		in, err := bind[p.ScoreInput](r)
		if err != nil {
			return nil, "", err
		}
		return nil, in.Reason, d.SetScores(*e, r.C.Param("entry"), r.UserID, in)
	}))
	group.GET(path+"/standings", a.poomsaeDivisionRoute(permPoomsaeRead, false, func(r *Request, e *p.Event, d *p.Division) (any, string, error) { return d.Standings(), "", nil }))
	group.GET(path+"/sheet", a.poomsaeDivisionRoute(permPoomsaeRead, false, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		mode := r.C.DefaultQuery("mode", "results")
		if mode != "results" && mode != "blank" && mode != "standings" {
			return nil, "", bad("mode must be results, blank or standings")
		}
		standings := d.Standings()
		rows := d.Results()
		ranks := map[string]*int{}
		for _, row := range standings.Ranked {
			ranks[row.Entry.ID] = row.Rank
		}
		for i := range rows {
			rows[i].Rank = ranks[rows[i].Entry.ID]
		}
		if mode == "standings" {
			rows = append(standings.Ranked, standings.Unranked...)
		}
		if mode == "blank" {
			for i := range rows {
				rows[i].Entry.Scores = p.ScoreInput{Form1: p.FormScore{Code: rows[i].Entry.Scores.Form1.Code}, Form2: p.FormScore{Code: rows[i].Entry.Scores.Form2.Code}}
				rows[i].Form1Total = nil
				rows[i].Form2Total = nil
				rows[i].Total = nil
				rows[i].Rank = nil
			}
		}
		return gin.H{"event": e, "division": d.DivisionInput, "divisionId": d.ID, "revision": d.Revision, "draw": d.ActiveDraw(), "final": standings.Final, "mode": mode, "generatedAt": time.Now().UTC(), "rows": rows}, "", nil
	}))
	group.POST(path+"/finalize", a.poomsaeDivisionRoute(permPoomsaeManage, true, func(r *Request, e *p.Event, d *p.Division) (any, string, error) { return nil, "", d.Finalize() }))
	group.POST(path+"/reopen", a.poomsaeDivisionRoute(permPoomsaeOverride, true, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		in, err := bind[p.ReasonInput](r)
		if err != nil {
			return nil, "", err
		}
		return nil, in.Reason, d.Reopen(in.Reason)
	}))
	group.GET(path+"/history", a.poomsaeDivisionRoute(permAuditRead, false, func(r *Request, e *p.Event, d *p.Division) (any, string, error) {
		limit, offset, err := poomsaePage(r)
		if err != nil {
			return nil, "", err
		}
		result, err := listJSON(r, `SELECT jsonb_build_object('id',id,'actorId',actor_id,'action',action,'reason',reason,'before',before_data,'after',after_data,'createdAt',created_at) FROM poomsae_change_log WHERE organization_id=$1 AND event_id=$2 AND division_id=$3 ORDER BY id DESC LIMIT $4 OFFSET $5`, r.OrganizationID, e.ID, d.ID, limit, offset)
		return result, "", err
	}))
}
func nonemptyIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	return ids
}
