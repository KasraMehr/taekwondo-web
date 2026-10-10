package platform

import (
	"backend/internal/modules/tournaments"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The monitor exposes only the fields needed by the weigh-in display.
func (a *API) publicWeighIn(r *Request) (any, error) {
	id := r.C.Query("tournamentId")
	if id != "" {
		if _, err := uuid.Parse(id); err != nil {
			return nil, bad("invalid tournamentId")
		}
	} else if r.C.Query("slot") != "" {
		var err error
		id, err = a.tournamentForPublicSlot(r)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, bad("tournamentId or slot is required")
	}
	var name, gender, age string
	var revision int64
	if err := r.Tx.QueryRow(r.Context(), `SELECT name,gender,age_category,revision FROM tournaments WHERE id=$1`, id).Scan(&name, &gender, &age, &revision); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound()
		}
		return nil, err
	}
	rows, err := r.Tx.Query(r.Context(), `SELECT athlete_id::text,data FROM tournament_entries WHERE tournament_id=$1 ORDER BY athlete_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	athletes := []tournaments.TournamentAthlete{}
	for rows.Next() {
		var athleteID string
		var raw []byte
		var athlete tournaments.TournamentAthlete
		if err := rows.Scan(&athleteID, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &athlete); err != nil {
			return nil, err
		}
		athlete.ID = athleteID
		athletes = append(athletes, athlete)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	maxNumber := 0
	for _, athlete := range athletes {
		if athlete.Number > maxNumber {
			maxNumber = athlete.Number
		}
	}
	seen := map[int]bool{}
	result := []gin.H{}
	for _, athlete := range athletes {
		number := athlete.Number
		if number <= 0 || seen[number] {
			maxNumber++
			number = maxNumber
		}
		seen[number] = true
		status := "pending"
		var weight *float64
		if athlete.WeighIn != nil {
			status = string(athlete.WeighIn.Status)
			weight = athlete.WeighIn.WeightKg
		} else if athlete.WeighedIn {
			status = "passed"
		}
		if status != "passed" && status != "failed" {
			status = "pending"
		}
		result = append(result, gin.H{"id": athlete.ID, "number": number, "name": athlete.Name, "club": athlete.Club, "weightCategory": athlete.WeightCategory, "status": status, "weightKg": weight})
	}
	return gin.H{"tournament": gin.H{"id": id, "name": name, "gender": gender, "ageCategory": age, "revision": revision}, "athletes": result}, nil
}
