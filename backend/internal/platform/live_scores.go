package platform

import (
	"backend/internal/modules/tournaments"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type liveScore struct {
	Version     int64                    `json:"version"`
	BlueID      string                   `json:"blueId"`
	RedID       string                   `json:"redId"`
	ActiveRound int                      `json:"activeRound"`
	Rounds      []tournaments.MatchRound `json:"rounds"`
}

func (a *API) saveLiveScore(r *Request, in liveScore) (any, error) {
	tid, mid := r.C.Param("tournament"), r.C.Param("match")
	// The tournament lock also serializes this write with final result operations.
	var tidCheck string
	if err := r.Tx.QueryRow(r.Context(), `SELECT id::text FROM tournaments WHERE id=$1 AND organization_id=$2 FOR UPDATE`, tid, r.OrganizationID).Scan(&tidCheck); err != nil {
		return nil, err
	}
	var raw []byte
	if err := r.Tx.QueryRow(r.Context(), `SELECT data FROM tournament_matches WHERE tournament_id=$1 AND id=$2`, tid, mid).Scan(&raw); err != nil {
		return nil, notFound()
	}
	var m tournaments.Match
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m.Status == nil || *m.Status != tournaments.MatchStatusOngoing {
		return nil, httpError{409, "live scores require an ongoing match"}
	}
	if m.Athlete1ID == nil || m.Athlete2ID == nil || in.BlueID == in.RedID || !((in.BlueID == *m.Athlete1ID && in.RedID == *m.Athlete2ID) || (in.BlueID == *m.Athlete2ID && in.RedID == *m.Athlete1ID)) {
		return nil, bad("invalid corners")
	}
	if len(in.Rounds) < 1 || len(in.Rounds) > 5 || in.ActiveRound < 0 || in.ActiveRound >= len(in.Rounds) || in.Version < 0 {
		return nil, bad("invalid rounds")
	}
	for i, round := range in.Rounds {
		if round.Number != i+1 {
			return nil, bad("invalid round number")
		}
		for _, score := range []tournaments.RoundScore{round.Blue, round.Red} {
			for _, v := range []int{score.Punch, score.BodyKick, score.HeadKick, score.TurningBodyKick, score.TurningHeadKick, score.GamJeom, score.GamJeomLate} {
				if v < 0 || v > 1000 {
					return nil, bad("invalid score")
				}
			}
		}
	}
	var version int64
	err := r.Tx.QueryRow(r.Context(), `SELECT version FROM match_live_scores WHERE tournament_id=$1 AND match_id=$2`, tid, mid).Scan(&version)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if version != in.Version {
		return nil, httpError{409, "امتیاز زنده توسط اپراتور دیگری تغییر کرده؛ فرم را دوباره باز کنید"}
	}
	in.Version++
	_, err = r.Tx.Exec(r.Context(), `INSERT INTO match_live_scores(tournament_id,match_id,version,data) VALUES($1,$2,$3,$4) ON CONFLICT(tournament_id,match_id) DO UPDATE SET version=excluded.version,data=excluded.data,updated_at=now()`, tid, mid, in.Version, in)
	if err != nil {
		return nil, err
	}
	return in, nil
}

func (a *API) publicLiveScores(r *Request) (any, error) {
	tid := r.C.Param("tournament")
	var exists bool
	if err := r.Tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM tournaments WHERE id=$1)`, tid).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, notFound()
	}
	rows, err := r.Tx.Query(r.Context(), `SELECT s.match_id::text,s.data,s.updated_at FROM match_live_scores s JOIN tournament_matches m ON m.tournament_id=s.tournament_id AND m.id=s.match_id JOIN tournaments t ON t.id=s.tournament_id WHERE s.tournament_id=$1 AND m.data->>'status'='ongoing' AND COALESCE((m.data->>'day')::int,1)=COALESCE((SELECT (d->>'day')::int FROM jsonb_array_elements(COALESCE(t.settings->'schedule'->'days','[]'::jsonb)) d WHERE d->>'date'=((now() AT TIME ZONE 'Asia/Tehran')::date)::text LIMIT 1),CASE WHEN t.event_date=(now() AT TIME ZONE 'Asia/Tehran')::date THEN 1 ELSE 0 END)`, tid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]any{}
	for rows.Next() {
		var id string
		var raw []byte
		var updated any
		if err = rows.Scan(&id, &raw, &updated); err != nil {
			return nil, err
		}
		var score liveScore
		if err = json.Unmarshal(raw, &score); err != nil {
			return nil, err
		}
		out[id] = gin.H{"score": score, "updatedAt": updated}
	}
	r.C.Header("Cache-Control", "public, max-age=0, s-maxage=1")
	return out, rows.Err()
}
