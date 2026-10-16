package poomsae

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type LeagueWeek struct {
	ID       string  `json:"id"`
	Stage    int     `json:"stage"`
	Position int     `json:"position"`
	Name     string  `json:"name"`
	EventID  *string `json:"eventId"`
}

type PoomsaeLeague struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	SeasonName   string       `json:"seasonName"`
	Weeks        []LeagueWeek `json:"weeks"`
	PointsByRank []int        `json:"pointsByRank"`
	Revision     int64        `json:"revision"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type CreatePoomsaeLeagueInput struct {
	Name          string `json:"name"`
	SeasonName    string `json:"seasonName"`
	StageCount    int    `json:"stageCount"`
	WeeksPerStage int    `json:"weeksPerStage"`
}

type LinkPoomsaeLeagueWeekInput struct {
	EventID *string `json:"eventId"`
}

type PoomsaeLeagueStanding struct {
	Rank        int    `json:"rank"`
	AthleteID   string `json:"athleteId,omitempty"`
	Name        string `json:"name"`
	TeamName    string `json:"teamName,omitempty"`
	Points      int    `json:"points"`
	Gold        int    `json:"gold"`
	Silver      int    `json:"silver"`
	Bronze      int    `json:"bronze"`
	Appearances int    `json:"appearances"`
}

type PoomsaeLeagueStandings struct {
	Players []PoomsaeLeagueStanding `json:"players"`
	Teams   []PoomsaeLeagueStanding `json:"teams"`
}

func (r Repository) CreatePoomsaeLeague(ctx context.Context, input CreatePoomsaeLeagueInput) (*PoomsaeLeague, error) {
	input.Name, input.SeasonName = strings.TrimSpace(input.Name), strings.TrimSpace(input.SeasonName)
	if input.Name == "" || input.SeasonName == "" || input.StageCount < 1 || input.StageCount > 10 || input.WeeksPerStage < 1 || input.WeeksPerStage > 52 {
		return nil, invalid("invalid league name, season, or number of stages/weeks")
	}
	now := time.Now().UTC()
	l := &PoomsaeLeague{ID: uuid.NewString(), Name: input.Name, SeasonName: input.SeasonName, Weeks: []LeagueWeek{}, PointsByRank: []int{10, 6, 3, 1}, Revision: 1, CreatedAt: now, UpdatedAt: now}
	for stage := 1; stage <= input.StageCount; stage++ {
		for week := 1; week <= input.WeeksPerStage; week++ {
			l.Weeks = append(l.Weeks, LeagueWeek{ID: uuid.NewString(), Stage: stage, Position: week, Name: fmt.Sprintf("هفته %d", week)})
		}
	}
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	_, err = r.Tx.Exec(ctx, `INSERT INTO poomsae_leagues(id,organization_id,data,revision,created_at,updated_at) VALUES($1,$2,$3,1,$4,$4)`, l.ID, r.OrganizationID, b, now)
	return l, err
}

func (r Repository) PoomsaeLeague(ctx context.Context, id string) (*PoomsaeLeague, error) {
	var b []byte
	var revision int64
	var created, updated time.Time
	err := r.Tx.QueryRow(ctx, `SELECT data,revision,created_at,updated_at FROM poomsae_leagues WHERE organization_id=$1 AND id=$2 FOR UPDATE`, r.OrganizationID, id).Scan(&b, &revision, &created, &updated)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var l PoomsaeLeague
	if err = json.Unmarshal(b, &l); err != nil {
		return nil, err
	}
	l.Revision, l.CreatedAt, l.UpdatedAt = revision, created, updated
	return &l, nil
}

func (r Repository) PoomsaeLeagues(ctx context.Context) ([]PoomsaeLeague, error) {
	rows, err := r.Tx.Query(ctx, `SELECT data,revision,created_at,updated_at FROM poomsae_leagues WHERE organization_id=$1 ORDER BY created_at DESC`, r.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PoomsaeLeague{}
	for rows.Next() {
		var b []byte
		var l PoomsaeLeague
		if err = rows.Scan(&b, &l.Revision, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &l); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r Repository) LinkPoomsaeLeagueWeek(ctx context.Context, id, weekID string, eventID *string) (*PoomsaeLeague, error) {
	l, err := r.PoomsaeLeague(ctx, id)
	if err != nil {
		return nil, err
	}
	index := -1
	for i, week := range l.Weeks {
		if week.ID == weekID {
			index = i
		}
	}
	if index < 0 {
		return nil, ErrNotFound
	}
	if eventID != nil {
		if _, err = uuid.Parse(*eventID); err != nil {
			return nil, invalid("invalid event id")
		}
		if _, err = r.Event(ctx, *eventID); err != nil {
			return nil, err
		}
		for i, week := range l.Weeks {
			if i != index && week.EventID != nil && *week.EventID == *eventID {
				return nil, conflict("event is already assigned to another week")
			}
		}
	}
	l.Weeks[index].EventID = eventID
	l.Revision++
	l.UpdatedAt = time.Now().UTC()
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	_, err = r.Tx.Exec(ctx, `UPDATE poomsae_leagues SET data=$3,revision=$4,updated_at=$5 WHERE organization_id=$1 AND id=$2`, r.OrganizationID, id, b, l.Revision, l.UpdatedAt)
	return l, err
}

func (r Repository) PoomsaeLeagueStandings(ctx context.Context, l *PoomsaeLeague) (PoomsaeLeagueStandings, error) {
	players, teams := map[string]*PoomsaeLeagueStanding{}, map[string]*PoomsaeLeagueStanding{}
	for _, week := range l.Weeks {
		if week.EventID == nil {
			continue
		}
		rows, err := r.Tx.Query(ctx, `SELECT id::text FROM poomsae_divisions WHERE organization_id=$1 AND event_id=$2 AND data->>'status'='finalized'`, r.OrganizationID, *week.EventID)
		if err != nil {
			return PoomsaeLeagueStandings{}, err
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				break
			}
			ids = append(ids, id)
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		if err != nil {
			return PoomsaeLeagueStandings{}, err
		}
		for _, id := range ids {
			d, err := r.Division(ctx, *week.EventID, id)
			if err != nil {
				return PoomsaeLeagueStandings{}, err
			}
			for _, result := range d.Standings().Ranked {
				if result.Rank == nil {
					continue
				}
				points := 0
				if *result.Rank > 0 && *result.Rank <= len(l.PointsByRank) {
					points = l.PointsByRank[*result.Rank-1]
				}
				entry := result.Entry
				key := entry.AthleteID
				if key == "" {
					key = entry.ID
				}
				player := players[key]
				if player == nil {
					player = &PoomsaeLeagueStanding{AthleteID: key}
					players[key] = player
				}
				player.Name = strings.TrimSpace(entry.FirstName + " " + entry.LastName)
				player.TeamName = strings.TrimSpace(entry.TeamName)
				addPoomsaeLeaguePoints(player, *result.Rank, points)
				if player.TeamName != "" {
					team := teams[player.TeamName]
					if team == nil {
						team = &PoomsaeLeagueStanding{Name: player.TeamName}
						teams[player.TeamName] = team
					}
					addPoomsaeLeaguePoints(team, *result.Rank, points)
				}
			}
		}
	}
	out := PoomsaeLeagueStandings{Players: []PoomsaeLeagueStanding{}, Teams: []PoomsaeLeagueStanding{}}
	for _, v := range players {
		out.Players = append(out.Players, *v)
	}
	for _, v := range teams {
		out.Teams = append(out.Teams, *v)
	}
	sortPoomsaeLeagueStandings(out.Players)
	sortPoomsaeLeagueStandings(out.Teams)
	return out, nil
}

func addPoomsaeLeaguePoints(row *PoomsaeLeagueStanding, rank, points int) {
	row.Points += points
	row.Appearances++
	switch rank {
	case 1:
		row.Gold++
	case 2:
		row.Silver++
	case 3:
		row.Bronze++
	}
}

func sortPoomsaeLeagueStandings(rows []PoomsaeLeagueStanding) {
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if a.Gold != b.Gold {
			return a.Gold > b.Gold
		}
		if a.Silver != b.Silver {
			return a.Silver > b.Silver
		}
		if a.Bronze != b.Bronze {
			return a.Bronze > b.Bronze
		}
		return a.Name < b.Name
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
}
