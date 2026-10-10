package tournaments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

var ErrTournamentNotFound = errors.New("tournament not found")
var ErrConflict = errors.New("resource changed; reload and retry")

type TournamentRepository interface {
	Create(context.Context, *Tournament) error
	GetByID(context.Context, string) (*Tournament, error)
	List(context.Context, TournamentListFilter) ([]Tournament, error)
	Update(context.Context, *Tournament) error
	Delete(context.Context, string) error
}
type TournamentListFilter struct {
	Gender           *Gender
	AgeCategory      *AgeCategory
	Format           *TournamentFormat
	LeagueID         *string
	WeekID           *string
	TeamTournamentID *string
	EventDateFrom    *time.Time
	EventDateTo      *time.Time
	Limit            int
	Offset           int
}

// The transaction is owned by the HTTP request. Every aggregate read locks its parent
// row; validation, normalized child writes and audit commit together.
type tournamentRepository struct {
	db             pgx.Tx
	organizationID string
}

func NewTournamentRepository(db pgx.Tx, organizationID string) TournamentRepository {
	return &tournamentRepository{db: db, organizationID: organizationID}
}
func (r *tournamentRepository) validateContext(ctx context.Context, t *Tournament) error {
	if t.LeagueID != nil {
		var ok bool
		err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM leagues l JOIN league_weeks w ON w.league_id=l.id WHERE l.id=$1 AND l.organization_id=$2 AND w.id=$3 AND ($4::uuid IS NULL OR EXISTS(SELECT 1 FROM league_groups g WHERE g.league_id=l.id AND g.id=$4)))`, t.LeagueID, r.organizationID, t.WeekID, t.GroupID).Scan(&ok)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidTournamentContext
		}
	}
	if t.TeamTournamentID != nil {
		return fmt.Errorf("%w: use league encounters for team competitions", ErrInvalidTournamentContext)
	}
	return nil
}
func (r *tournamentRepository) Create(ctx context.Context, t *Tournament) error {
	if t == nil {
		return ErrInvalidTournamentInput
	}
	if err := r.validateContext(ctx, t); err != nil {
		return err
	}
	normalizeTournamentCollections(t)
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	t.Revision = 1
	_, err := r.db.Exec(ctx, `INSERT INTO tournaments(id,organization_id,name,event_date,courts,gender,age_category,format,court_assignment,elimination_matches,league_id,stage_order,week_id,group_id,created_at,updated_at,settings)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		t.ID, r.organizationID, t.Name, t.Date, t.Courts, t.Gender, t.AgeCategory, t.Format, t.CourtAssignment, t.EliminationMatches, t.LeagueID, t.StageOrder, t.WeekID, t.GroupID, t.CreatedAt, t.UpdatedAt, t.Settings)
	if err != nil {
		return err
	}
	return r.saveChildren(ctx, t)
}
func (r *tournamentRepository) GetByID(ctx context.Context, id string) (*Tournament, error) {
	t := &Tournament{}
	err := r.db.QueryRow(ctx, `SELECT id::text,name,event_date,courts,gender,age_category,format,court_assignment,elimination_matches,league_id::text,stage_order,week_id::text,group_id::text,created_at,updated_at,revision,settings FROM tournaments WHERE id=$1 AND organization_id=$2 FOR UPDATE`, id, r.organizationID).Scan(
		&t.ID, &t.Name, &t.Date, &t.Courts, &t.Gender, &t.AgeCategory, &t.Format, &t.CourtAssignment, &t.EliminationMatches, &t.LeagueID, &t.StageOrder, &t.WeekID, &t.GroupID, &t.CreatedAt, &t.UpdatedAt, &t.Revision, &t.Settings)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTournamentNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT data FROM tournament_entries WHERE tournament_id=$1 ORDER BY athlete_id`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var b []byte
		var a TournamentAthlete
		if err = rows.Scan(&b); err == nil {
			err = json.Unmarshal(b, &a)
		}
		if err != nil {
			rows.Close()
			return nil, err
		}
		t.Athletes = append(t.Athletes, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	assignAthleteNumbers(t.Athletes)
	rows, err = r.db.Query(ctx, `SELECT data FROM tournament_matches WHERE tournament_id=$1 ORDER BY (data->>'round')::int,(data->>'bracketIndex')::int,id`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var b []byte
		var m Match
		if err = rows.Scan(&b); err == nil {
			err = json.Unmarshal(b, &m)
		}
		if err != nil {
			rows.Close()
			return nil, err
		}
		t.Matches = append(t.Matches, m)
	}
	err = rows.Err()
	rows.Close()
	normalizeTournamentCollections(t)
	return t, err
}
func (r *tournamentRepository) List(ctx context.Context, f TournamentListFilter) ([]Tournament, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	rows, err := r.db.Query(ctx, `SELECT id::text FROM tournaments WHERE organization_id=$1
 AND ($2::text IS NULL OR gender=$2) AND ($3::text IS NULL OR age_category=$3)
 AND ($4::text IS NULL OR format=$4) AND ($5::uuid IS NULL OR league_id=$5)
 AND ($6::uuid IS NULL OR week_id=$6) AND ($7::date IS NULL OR event_date >= $7)
 AND ($8::date IS NULL OR event_date <= $8)
 ORDER BY event_date DESC,id LIMIT $9 OFFSET $10`, r.organizationID, f.Gender, f.AgeCategory, f.Format, f.LeagueID, f.WeekID, f.EventDateFrom, f.EventDateTo, f.Limit, f.Offset)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []Tournament{}
	for _, id := range ids {
		t, e := r.GetByID(ctx, id)
		if e != nil {
			return nil, e
		}
		out = append(out, *t)
	}
	return out, nil
}
func (r *tournamentRepository) Update(ctx context.Context, t *Tournament) error {
	if t == nil {
		return ErrInvalidTournamentInput
	}
	if err := r.validateContext(ctx, t); err != nil {
		return err
	}
	normalizeTournamentCollections(t)
	t.UpdatedAt = time.Now().UTC()
	tag, err := r.db.Exec(ctx, `UPDATE tournaments SET name=$3,event_date=$4,courts=$5,gender=$6,age_category=$7,format=$8,court_assignment=$9,elimination_matches=$10,league_id=$11,stage_order=$12,week_id=$13,group_id=$14,updated_at=$15,revision=revision+1,settings=$17 WHERE id=$1 AND organization_id=$2 AND revision=$16`,
		t.ID, r.organizationID, t.Name, t.Date, t.Courts, t.Gender, t.AgeCategory, t.Format, t.CourtAssignment, t.EliminationMatches, t.LeagueID, t.StageOrder, t.WeekID, t.GroupID, t.UpdatedAt, t.Revision, t.Settings)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	if err = r.saveChildren(ctx, t); err != nil {
		return err
	}
	t.Revision++
	return nil
}
func (r *tournamentRepository) saveChildren(ctx context.Context, t *Tournament) error {
	// Parent lock serializes replacement and result propagation across all instances.
	ids := []string{}
	for _, a := range t.Athletes {
		ids = append(ids, a.ID)
		_, err := r.db.Exec(ctx, `INSERT INTO tournament_entries(tournament_id,athlete_id,data,profile_id) VALUES($1,$2,$3,$4) ON CONFLICT(tournament_id,athlete_id) DO UPDATE SET data=excluded.data,profile_id=excluded.profile_id`, t.ID, a.ID, a, a.ProfileID)
		if err != nil {
			return err
		}
	}
	if _, err := r.db.Exec(ctx, `DELETE FROM tournament_entries WHERE tournament_id=$1 AND NOT(athlete_id=ANY($2::uuid[]))`, t.ID, ids); err != nil {
		return err
	}
	ids = []string{}
	for _, m := range t.Matches {
		ids = append(ids, m.ID)
		_, err := r.db.Exec(ctx, `INSERT INTO tournament_matches(tournament_id,id,data) VALUES($1,$2,$3) ON CONFLICT(tournament_id,id) DO UPDATE SET data=excluded.data`, t.ID, m.ID, m)
		if err != nil {
			return err
		}
	}
	_, err := r.db.Exec(ctx, `DELETE FROM tournament_matches WHERE tournament_id=$1 AND NOT(id=ANY($2::uuid[]))`, t.ID, ids)
	return err
}
func (r *tournamentRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM tournaments WHERE id=$1 AND organization_id=$2`, id, r.organizationID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrTournamentNotFound
	}
	return nil
}
func normalizeTournamentCollections(t *Tournament) {
	if t.Settings == nil {
		defaults := DefaultSettings(t.Date)
		t.Settings = &defaults
	}
	for i := range t.Matches {
		t.Matches[i].Day = dayForCategory(t, t.Matches[i].WeightCategory)
	}
	if t.Athletes == nil {
		t.Athletes = []TournamentAthlete{}
	}
	for i := range t.Athletes {
		a := &t.Athletes[i]
		if a.WeighIn == nil {
			a.WeighIn = newPendingWeighIn()
			if a.WeighedIn {
				a.WeighIn.Status = WeighInPassed
			}
		}
		if a.WeighIn.Status == "" {
			a.WeighIn.Status = WeighInPending
		}
		if a.WeighIn.Attempts == nil {
			a.WeighIn.Attempts = []WeighInAttempt{}
		}
		a.WeighedIn = a.WeighIn.Status == WeighInPassed
	}
	if t.Matches == nil {
		t.Matches = []Match{}
	}
	if t.CourtAssignment == nil {
		t.CourtAssignment = map[string][]int{}
	}
	if t.EliminationMatches == nil {
		t.EliminationMatches = []EliminationMatch{}
	}
}

// Legacy entries are loaded in athlete-ID order; keep assigned numbers and
// give any unnumbered entry a new, unique number.
func assignAthleteNumbers(athletes []TournamentAthlete) {
	maxNumber := 0
	for _, a := range athletes {
		if a.Number > maxNumber {
			maxNumber = a.Number
		}
	}
	seen := make(map[int]bool, len(athletes))
	for i := range athletes {
		number := athletes[i].Number
		if number > 0 && !seen[number] {
			seen[number] = true
			continue
		}
		maxNumber++
		athletes[i].Number = maxNumber
		seen[maxNumber] = true
	}
}
