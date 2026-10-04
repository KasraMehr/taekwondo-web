package poomsae

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Repository uses the request-owned transaction. Lock event before division on
// every operation, including snapshots; configuration and scores stay consistent.
type Repository struct {
	Tx             pgx.Tx
	OrganizationID string
}

func (r Repository) Event(ctx context.Context, id string) (*Event, error) {
	var b []byte
	var rev int64
	err := r.Tx.QueryRow(ctx, `SELECT data,revision FROM poomsae_events WHERE organization_id=$1 AND id=$2 FOR UPDATE`, r.OrganizationID, id).Scan(&b, &rev)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var e Event
	if err = json.Unmarshal(b, &e); err != nil {
		return nil, err
	}
	e.Revision = rev
	return &e, nil
}
func (r Repository) CreateEvent(ctx context.Context, e *Event) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = r.Tx.Exec(ctx, `INSERT INTO poomsae_events(id,organization_id,data,revision) VALUES($1,$2,$3,$4)`, e.ID, r.OrganizationID, b, e.Revision)
	return err
}
func (r Repository) SaveEvent(ctx context.Context, e *Event) error {
	e.Revision++
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = r.Tx.Exec(ctx, `UPDATE poomsae_events SET data=$3,revision=$4 WHERE organization_id=$1 AND id=$2`, r.OrganizationID, e.ID, b, e.Revision)
	return err
}
func (r Repository) EventHasScores(ctx context.Context, id string) (bool, error) {
	var yes bool
	err := r.Tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM poomsae_divisions WHERE organization_id=$1 AND event_id=$2 AND (data->>'everScored')::boolean)`, r.OrganizationID, id).Scan(&yes)
	return yes, err
}
func (r Repository) Division(ctx context.Context, eventID, id string) (*Division, error) {
	var b []byte
	var rev int64
	err := r.Tx.QueryRow(ctx, `SELECT data,revision FROM poomsae_divisions WHERE organization_id=$1 AND event_id=$2 AND id=$3 FOR UPDATE`, r.OrganizationID, eventID, id).Scan(&b, &rev)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var d Division
	if err = json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	d.Revision = rev
	d.Entries = []Entry{}
	d.Draws = []Draw{}
	rows, err := r.Tx.Query(ctx, `SELECT data FROM poomsae_entries WHERE organization_id=$1 AND division_id=$2 ORDER BY id`, r.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var raw []byte
		var a Entry
		if err = rows.Scan(&raw); err == nil {
			err = json.Unmarshal(raw, &a)
		}
		if err != nil {
			rows.Close()
			return nil, err
		}
		d.Entries = append(d.Entries, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = r.Tx.Query(ctx, `SELECT data FROM poomsae_draws WHERE division_id=$1 ORDER BY version`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var raw []byte
		var draw Draw
		if err = rows.Scan(&raw); err == nil {
			err = json.Unmarshal(raw, &draw)
		}
		if err != nil {
			rows.Close()
			return nil, err
		}
		d.Draws = append(d.Draws, draw)
	}
	err = rows.Err()
	rows.Close()
	return &d, err
}
func (r Repository) CreateDivision(ctx context.Context, d *Division) error {
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	_, err = r.Tx.Exec(ctx, `INSERT INTO poomsae_divisions(id,organization_id,event_id,data,revision) VALUES($1,$2,$3,$4,$5)`, d.ID, r.OrganizationID, d.EventID, b, d.Revision)
	return err
}
func (r Repository) SaveDivision(ctx context.Context, d *Division) error {
	d.Revision++
	metadata := *d
	metadata.Entries = nil
	metadata.Draws = nil
	b, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = r.Tx.Exec(ctx, `UPDATE poomsae_divisions SET data=$4,revision=$5 WHERE organization_id=$1 AND event_id=$2 AND id=$3`, r.OrganizationID, d.EventID, d.ID, b, d.Revision)
	if err != nil {
		return err
	}
	ids := []string{}
	for _, a := range d.Entries {
		ids = append(ids, a.ID)
		raw, e := json.Marshal(a)
		if e != nil {
			return e
		}
		_, err = r.Tx.Exec(ctx, `INSERT INTO poomsae_entries(id,organization_id,division_id,athlete_id,data) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET data=excluded.data`, a.ID, r.OrganizationID, d.ID, a.AthleteID, raw)
		if err != nil {
			return err
		}
	}
	_, err = r.Tx.Exec(ctx, `DELETE FROM poomsae_entries WHERE organization_id=$1 AND division_id=$2 AND NOT(id=ANY($3::uuid[]))`, r.OrganizationID, d.ID, ids)
	if err != nil {
		return err
	}
	// Deactivate first to preserve the partial unique index during a redraw.
	_, err = r.Tx.Exec(ctx, `UPDATE poomsae_draws SET active=false WHERE division_id=$1`, d.ID)
	if err != nil {
		return err
	}
	for _, draw := range d.Draws {
		raw, e := json.Marshal(draw)
		if e != nil {
			return e
		}
		_, err = r.Tx.Exec(ctx, `INSERT INTO poomsae_draws(id,division_id,version,active,data) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET active=excluded.active,data=excluded.data`, draw.ID, d.ID, draw.Version, draw.Active, raw)
		if err != nil {
			return err
		}
	}
	return nil
}
func (r Repository) ValidateProfile(ctx context.Context, in EntryInput) error {
	var active bool
	var birth, gender *string
	err := r.Tx.QueryRow(ctx, `SELECT is_active,to_char(birth_date,'YYYY-MM-DD'),gender FROM athletes WHERE organization_id=$1 AND id=$2 FOR SHARE`, r.OrganizationID, in.AthleteID).Scan(&active, &birth, &gender)
	if errors.Is(err, pgx.ErrNoRows) {
		return invalid("athlete profile does not belong to organization")
	}
	if err != nil {
		return err
	}
	if !active {
		return invalid("athlete profile is inactive")
	}
	if birth != nil && *birth != in.BirthDate || gender != nil && *gender != "" && *gender != in.Gender {
		return invalid("birth date or gender differs from athlete profile")
	}
	return nil
}
func (r Repository) Audit(ctx context.Context, eventID, divisionID, actor, action, reason string, before, after any) error {
	b, err := json.Marshal(before)
	if err != nil {
		return err
	}
	a, err := json.Marshal(after)
	if err != nil {
		return err
	}
	var division any
	if divisionID != "" {
		division = divisionID
	}
	_, err = r.Tx.Exec(ctx, `INSERT INTO poomsae_change_log(organization_id,event_id,division_id,actor_id,action,reason,before_data,after_data) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, r.OrganizationID, eventID, division, actor, action, reason, b, a)
	return err
}
