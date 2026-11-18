package platform

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// athletePortal only reads entries explicitly linked to the signed-in profile.
func (a *API) athletePortal(r *Request) (any, error) {
	var profile []byte
	err := r.Tx.QueryRow(r.Context(), `SELECT jsonb_build_object('id',a.id,'name',a.name,'nationalCode',a.national_code,'birthDate',a.birth_date,'photoData',a.photo_data,'club',c.name) FROM athletes a LEFT JOIN clubs c ON c.id=a.club_id WHERE a.organization_id=$1 AND a.user_id=$2 AND a.is_active`, r.OrganizationID, r.UserID).Scan(&profile)
	if errors.Is(err, pgx.ErrNoRows) {
		return gin.H{"role": r.Role, "profile": nil, "events": []any{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var person any
	if err = json.Unmarshal(profile, &person); err != nil {
		return nil, err
	}
	rows, err := r.Tx.Query(r.Context(), `SELECT t.id::text,t.name,t.event_date::text,t.revision,t.weigh_in_at,e.data,t.settings FROM tournament_entries e JOIN tournaments t ON t.id=e.tournament_id WHERE t.organization_id=$1 AND e.profile_id=(SELECT id FROM athletes WHERE organization_id=$1 AND user_id=$2 AND is_active) ORDER BY t.event_date,t.id`, r.OrganizationID, r.UserID)
	if err != nil {
		return nil, err
	}
	events := []gin.H{}
	for rows.Next() {
		var id, name, date, entryID string
		var revision int64
		var weighInAt *time.Time
		var data, settings []byte
		if err = rows.Scan(&id, &name, &date, &revision, &weighInAt, &data, &settings); err != nil {
			return nil, err
		}
		var entry struct {
			ID             string `json:"id"`
			WeightCategory string `json:"weightCategory"`
			WeighIn        struct {
				WeightKg *float64 `json:"weightKg"`
			} `json:"weighIn"`
		}
		if err = json.Unmarshal(data, &entry); err != nil {
			return nil, err
		}
		entryID = entry.ID
		var config any
		if len(settings) > 0 {
			if err = json.Unmarshal(settings, &config); err != nil {
				return nil, err
			}
		}
		events = append(events, gin.H{"id": id, "name": name, "date": date, "revision": revision, "weighInAt": weighInAt, "entryId": entryID, "weightCategory": entry.WeightCategory, "weightKg": entry.WeighIn.WeightKg, "settings": config, "matches": []any{}})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for _, event := range events {
		matches, queryErr := r.Tx.Query(r.Context(), `SELECT data FROM tournament_matches WHERE tournament_id=$1 AND (data->>'athlete1Id'=$2 OR data->>'athlete2Id'=$2) AND COALESCE((data->>'isBye')::boolean,false)=false ORDER BY (data->>'day')::int,(data->>'court')::int,(data->>'order')::int`, event["id"], event["entryId"])
		if queryErr != nil {
			return nil, queryErr
		}
		bouts := []any{}
		for matches.Next() {
			var raw []byte
			if err = matches.Scan(&raw); err != nil {
				matches.Close()
				return nil, err
			}
			var match any
			if err = json.Unmarshal(raw, &match); err != nil {
				matches.Close()
				return nil, err
			}
			bouts = append(bouts, match)
		}
		if err = matches.Err(); err != nil {
			matches.Close()
			return nil, err
		}
		matches.Close()
		event["matches"] = bouts
	}
	return gin.H{"role": r.Role, "profile": person, "events": events}, nil
}

func (a *API) saveWeighInTime(r *Request) (any, error) {
	if err := r.authorizeTournament(permTournamentsManage, r.C.Param("tournament")); err != nil {
		return nil, err
	}
	in, err := bind[struct {
		At string `json:"at"`
	}](r)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.At) == "" {
		return nil, bad("weigh-in time is required")
	}
	var at string
	err = r.Tx.QueryRow(r.Context(), `UPDATE tournaments SET weigh_in_at=$3::timestamptz,revision=revision+1 WHERE id=$1 AND organization_id=$2 RETURNING weigh_in_at::text`, r.C.Param("tournament"), r.OrganizationID, in.At).Scan(&at)
	return gin.H{"weighInAt": at}, err
}

func (a *API) saveOwnPhoto(r *Request) (any, error) {
	in, err := bind[struct {
		PhotoData string `json:"photoData"`
	}](r)
	if err != nil {
		return nil, err
	}
	if len(in.PhotoData) > 1024*1024 || len(in.PhotoData) < 30 || (!strings.HasPrefix(in.PhotoData, "data:image/jpeg;base64,") && !strings.HasPrefix(in.PhotoData, "data:image/png;base64,")) {
		return nil, bad("photo must be a JPEG or PNG data URL under 1 MB")
	}
	tag, err := r.Tx.Exec(r.Context(), `UPDATE athletes SET photo_data=$3,updated_at=now() WHERE organization_id=$1 AND user_id=$2 AND is_active`, r.OrganizationID, r.UserID, in.PhotoData)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, notFound()
	}
	return gin.H{"photoData": in.PhotoData}, nil
}
