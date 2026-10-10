package platform

import (
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// A slot is the one-based creation order of tournaments on an event day.
// The date nearest to today is used so these short URLs also work while
// preparing or reviewing an event outside its scheduled day.
func (a *API) tournamentForPublicSlot(r *Request) (string, error) {
	slot, err := strconv.Atoi(r.C.Query("slot"))
	if err != nil || slot < 1 || slot > 100 {
		return "", bad("invalid slot")
	}
	iran, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		return "", err
	}
	today := time.Now().In(iran).Format("2006-01-02")
	if requested := r.C.Query("date"); requested != "" {
		if _, err := time.Parse("2006-01-02", requested); err != nil {
			return "", bad("invalid date")
		}
		today = requested
	}
	var id string
	if r.C.Query("date") != "" {
		err = r.Tx.QueryRow(r.Context(), `
			SELECT id::text FROM tournaments WHERE event_date = $1::date
			ORDER BY created_at, id OFFSET $2 LIMIT 1`, today, slot-1).Scan(&id)
	} else {
		err = r.Tx.QueryRow(r.Context(), `
		SELECT t.id::text FROM tournaments t
		WHERE t.event_date = (
			SELECT event_date FROM tournaments
			ORDER BY abs(event_date - $1::date), event_date DESC LIMIT 1
		)
		ORDER BY t.created_at, t.id
		OFFSET $2 LIMIT 1`, today, slot-1).Scan(&id)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "", notFound()
	}
	return id, err
}
