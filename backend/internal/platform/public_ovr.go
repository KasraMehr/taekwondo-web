package platform

import (
	"encoding/json"
	"errors"
	"sort"

	"backend/internal/modules/tournaments"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ovrMatch struct {
	ID             string `json:"id"`
	BlueName       string `json:"blueName"`
	RedName        string `json:"redName"`
	WeightCategory string `json:"weightCategory"`
	Side           string `json:"side"`
	Status         string `json:"status"`
	MatchNumber    int    `json:"matchNumber"`
	Round          int    `json:"round"`
}

func matchForOVR(m tournaments.Match, names map[string]string) ovrMatch {
	number := m.Order
	if m.MatchNumber != nil {
		number = *m.MatchNumber
	}
	blue, red := "در انتظار", "در انتظار"
	if m.Athlete1ID != nil {
		blue = names[*m.Athlete1ID]
	}
	if m.Athlete2ID != nil {
		red = names[*m.Athlete2ID]
	}
	status := "pending"
	if m.Status != nil {
		status = string(*m.Status)
	}
	return ovrMatch{ID: m.ID, BlueName: blue, RedName: red, WeightCategory: m.WeightCategory, Side: string(m.Side), Status: status, MatchNumber: number, Round: m.Round}
}

func (a *API) publicOVR(r *Request) (any, error) {
	id := r.C.Query("tournamentId")
	if id != "" {
		if _, err := uuid.Parse(id); err != nil {
			return nil, bad("invalid tournamentId")
		}
	} else {
		err := r.Tx.QueryRow(r.Context(), `SELECT t.id::text FROM tournaments t WHERE EXISTS(SELECT 1 FROM tournament_matches m WHERE m.tournament_id=t.id AND COALESCE((m.data->>'isBye')::boolean,false)=false AND m.data->>'status' IN ('ongoing','pending')) ORDER BY t.event_date DESC,t.updated_at DESC LIMIT 1`).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return gin.H{"tournament": nil, "courts": []any{}}, nil
		}
		if err != nil {
			return nil, err
		}
	}
	var name string
	var courts int
	var revision int64
	if err := r.Tx.QueryRow(r.Context(), `SELECT name,courts,revision FROM tournaments WHERE id=$1`, id).Scan(&name, &courts, &revision); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound()
		}
		return nil, err
	}
	names := map[string]string{}
	rows, err := r.Tx.Query(r.Context(), `SELECT athlete_id::text,data FROM tournament_entries WHERE tournament_id=$1`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var athleteID string
		var raw []byte
		if err = rows.Scan(&athleteID, &raw); err != nil {
			rows.Close()
			return nil, err
		}
		var a tournaments.TournamentAthlete
		if err = json.Unmarshal(raw, &a); err != nil {
			rows.Close()
			return nil, err
		}
		names[athleteID] = a.Name
	}
	rows.Close()
	matches := []tournaments.Match{}
	rows, err = r.Tx.Query(r.Context(), `SELECT data FROM tournament_matches WHERE tournament_id=$1`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, err
		}
		var m tournaments.Match
		if err = json.Unmarshal(raw, &m); err != nil {
			rows.Close()
			return nil, err
		}
		if m.IsBye != nil && *m.IsBye {
			continue
		}
		matches = append(matches, m)
	}
	rows.Close()
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Court != matches[j].Court {
			return matches[i].Court < matches[j].Court
		}
		ni, nj := matches[i].Order, matches[j].Order
		if matches[i].MatchNumber != nil {
			ni = *matches[i].MatchNumber
		}
		if matches[j].MatchNumber != nil {
			nj = *matches[j].MatchNumber
		}
		return ni < nj
	})
	resultCourts := make([]gin.H, 0, courts)
	for court := 1; court <= courts; court++ {
		remaining := []tournaments.Match{}
		for _, m := range matches {
			status := "pending"
			if m.Status != nil {
				status = string(*m.Status)
			}
			if m.Court == court && status != "completed" {
				remaining = append(remaining, m)
			}
		}
		var current *tournaments.Match
		for i := range remaining {
			if remaining[i].Status != nil && *remaining[i].Status == tournaments.MatchStatusOngoing {
				current = &remaining[i]
				break
			}
		}
		if current == nil && len(remaining) > 0 {
			current = &remaining[0]
		}
		upcoming := []ovrMatch{}
		for i := range remaining {
			if current != nil && remaining[i].ID == current.ID {
				continue
			}
			upcoming = append(upcoming, matchForOVR(remaining[i], names))
			if len(upcoming) == 4 {
				break
			}
		}
		var active any
		if current != nil {
			active = matchForOVR(*current, names)
		}
		resultCourts = append(resultCourts, gin.H{"court": court, "current": active, "upcoming": upcoming})
	}
	return gin.H{"tournament": gin.H{"id": id, "name": name, "courts": courts, "revision": revision}, "courts": resultCourts}, nil
}
