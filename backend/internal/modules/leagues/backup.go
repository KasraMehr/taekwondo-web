package leagues

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/modules/tournaments"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Backup struct {
	Kind        string                   `json:"kind"`
	Version     int                      `json:"version"`
	League      League                   `json:"league"`
	Tournaments []tournaments.Tournament `json:"tournaments"`
	Encounters  []BackupEncounter        `json:"encounters"`
}

type BackupEncounter struct {
	ID      string          `json:"id"`
	WeekID  string          `json:"weekId"`
	Team1ID string          `json:"team1Id"`
	Team2ID string          `json:"team2Id"`
	Status  string          `json:"status"`
	Bouts   json.RawMessage `json:"bouts"`
	Lineup  json.RawMessage `json:"lineup"`
}

type ImportBackupInput struct {
	Backup Backup `json:"backup"`
	Mode   string `json:"mode"`
}

func (s *Service) ExportBackup(ctx context.Context, id string) (*Backup, error) {
	l, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `SELECT id::text FROM tournaments WHERE organization_id=$1 AND league_id=$2 ORDER BY event_date,id`, s.organizationID, id)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	ts := tournaments.NewTournamentService(tournaments.NewTournamentRepository(s.db, s.organizationID))
	b := &Backup{Kind: "tkd-league-server-bundle", Version: 1, League: *l, Tournaments: []tournaments.Tournament{}, Encounters: []BackupEncounter{}}
	for _, tid := range ids {
		t, e := ts.GetByID(ctx, tid)
		if e != nil {
			return nil, e
		}
		b.Tournaments = append(b.Tournaments, *t)
	}
	encounterRows, err := s.db.Query(ctx, `SELECT id::text,week_id::text,team1_id::text,team2_id::text,status,bouts,lineup FROM encounters WHERE league_id=$1 ORDER BY week_id,id`, id)
	if err != nil {
		return nil, err
	}
	for encounterRows.Next() {
		var e BackupEncounter
		if err = encounterRows.Scan(&e.ID, &e.WeekID, &e.Team1ID, &e.Team2ID, &e.Status, &e.Bouts, &e.Lineup); err != nil {
			break
		}
		b.Encounters = append(b.Encounters, e)
	}
	if err == nil {
		err = encounterRows.Err()
	}
	encounterRows.Close()
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) ImportBackup(ctx context.Context, in ImportBackupInput) (*League, error) {
	b := in.Backup
	if b.Kind != "tkd-league-server-bundle" || b.Version != 1 || (in.Mode != "add" && in.Mode != "replace") {
		return nil, fmt.Errorf("%w: unsupported league backup or import mode", ErrInvalid)
	}
	if strings.TrimSpace(b.League.Name) == "" || !tournaments.Gender(b.League.Gender).IsValid() || !tournaments.AgeCategory(b.League.AgeCategory).IsValid() || len(b.League.Stages) == 0 {
		return nil, ErrInvalid
	}
	if err := validateSettings(b.League.Settings); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(b.League.ID); err != nil {
		return nil, ErrInvalid
	}
	id := b.League.ID
	now := time.Now().UTC()
	if in.Mode == "add" {
		id = uuid.NewString()
	} else {
		if _, err := s.Get(ctx, id); err != nil {
			return nil, err
		}
		for _, table := range []string{"encounters", "tournaments", "league_athletes", "league_teams", "league_groups", "league_weeks", "league_stages"} {
			if _, err := s.db.Exec(ctx, "DELETE FROM "+table+" WHERE league_id=$1", id); err != nil {
				return nil, err
			}
		}
	}
	config, err := json.Marshal(b.League.Settings)
	if err != nil {
		return nil, err
	}
	status := b.League.Status
	if status != "draft" && status != "active" && status != "completed" && status != "archived" {
		status = "draft"
	}
	if in.Mode == "add" {
		_, err = s.db.Exec(ctx, `INSERT INTO leagues(id,organization_id,name,config,gender,age_category,status,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8)`, id, s.organizationID, strings.TrimSpace(b.League.Name), config, b.League.Gender, b.League.AgeCategory, status, now)
	} else {
		_, err = s.db.Exec(ctx, `UPDATE leagues SET name=$3,config=$4,gender=$5,age_category=$6,status=$7,revision=revision+1,updated_at=$8 WHERE id=$1 AND organization_id=$2`, id, s.organizationID, strings.TrimSpace(b.League.Name), config, b.League.Gender, b.League.AgeCategory, status, now)
	}
	if err != nil {
		return nil, err
	}
	stageIDs, weekIDs, groupIDs, teamIDs, profileIDs, leagueAthleteIDs := map[string]string{}, map[string]string{}, map[string]string{}, map[string]string{}, map[string]string{}, map[string]string{}
	for _, stage := range b.League.Stages {
		if _, e := uuid.Parse(stage.ID); e != nil {
			return nil, ErrInvalid
		}
		stageID := stage.ID
		if in.Mode == "add" {
			stageID = uuid.NewString()
		}
		stageIDs[stage.ID] = stageID
		state := stage.Status
		if state != "pending" && state != "active" && state != "completed" {
			state = "pending"
		}
		_, err = s.db.Exec(ctx, `INSERT INTO league_stages(id,league_id,name,position,status) VALUES($1,$2,$3,$4,$5)`, stageID, id, stage.Name, stage.Position, state)
		if err != nil {
			return nil, err
		}
		for _, week := range stage.Weeks {
			if _, e := uuid.Parse(week.ID); e != nil {
				return nil, ErrInvalid
			}
			weekID := week.ID
			if in.Mode == "add" {
				weekID = uuid.NewString()
			}
			weekIDs[week.ID] = weekID
			_, err = s.db.Exec(ctx, `INSERT INTO league_weeks(id,league_id,stage_id,name,position,date_range,locked) VALUES($1,$2,$3,$4,$5,$6,$7)`, weekID, id, stageID, week.Name, week.Position, week.DateRange, week.Locked)
			if err != nil {
				return nil, err
			}
		}
	}
	for _, group := range b.League.Groups {
		if _, e := uuid.Parse(group.ID); e != nil {
			return nil, ErrInvalid
		}
		groupID := group.ID
		if in.Mode == "add" {
			groupID = uuid.NewString()
		}
		groupIDs[group.ID] = groupID
		_, err = s.db.Exec(ctx, `INSERT INTO league_groups(id,league_id,name,position) VALUES($1,$2,$3,$4)`, groupID, id, group.Name, group.Position)
		if err != nil {
			return nil, err
		}
	}
	for _, team := range b.League.Teams {
		if _, e := uuid.Parse(team.ID); e != nil {
			return nil, ErrInvalid
		}
		clubID, e := s.resolveBackupClub(ctx, team.ClubID, team.ClubName)
		if e != nil {
			return nil, e
		}
		teamID := team.ID
		if in.Mode == "add" {
			teamID = uuid.NewString()
		}
		teamIDs[team.ID] = teamID
		var groupID *string
		if team.GroupID != nil {
			mapped, ok := groupIDs[*team.GroupID]
			if !ok {
				return nil, ErrInvalid
			}
			groupID = &mapped
		}
		_, err = s.db.Exec(ctx, `INSERT INTO league_teams(id,league_id,club_id,group_id,active) VALUES($1,$2,$3,$4,$5)`, teamID, id, clubID, groupID, team.Active)
		if err != nil {
			return nil, err
		}
	}
	for _, athlete := range b.League.Athletes {
		if _, e := uuid.Parse(athlete.ID); e != nil {
			return nil, ErrInvalid
		}
		var teamID, groupID *string
		if athlete.TeamID != nil {
			mapped, ok := teamIDs[*athlete.TeamID]
			if !ok {
				return nil, ErrInvalid
			}
			teamID = &mapped
		}
		if athlete.GroupID != nil {
			mapped, ok := groupIDs[*athlete.GroupID]
			if !ok {
				return nil, ErrInvalid
			}
			groupID = &mapped
		}
		profileID, e := s.resolveBackupProfile(ctx, athlete.ProfileID, athlete.Name, b.League.Gender)
		if e != nil {
			return nil, e
		}
		profileIDs[athlete.ProfileID] = profileID
		athleteID := athlete.ID
		if in.Mode == "add" {
			athleteID = uuid.NewString()
		}
		leagueAthleteIDs[athlete.ID] = athleteID
		created := athlete.CreatedAt
		if created.IsZero() {
			created = now
		}
		_, err = s.db.Exec(ctx, `INSERT INTO league_athletes(id,league_id,profile_id,team_id,group_id,club_name,coach_name,weight_category,ranking,active,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, athleteID, id, profileID, teamID, groupID, athlete.ClubName, athlete.CoachName, athlete.WeightCategory, athlete.Ranking, athlete.Active, created, now)
		if err != nil {
			return nil, err
		}
	}
	for _, stage := range b.League.Stages {
		promoted := []string{}
		for _, sourceID := range stage.PromotedTeamIDs {
			mapped, ok := teamIDs[sourceID]
			if !ok {
				return nil, ErrInvalid
			}
			promoted = append(promoted, mapped)
		}
		_, err = s.db.Exec(ctx, `UPDATE league_stages SET promoted_team_ids=$2 WHERE id=$1`, stageIDs[stage.ID], promoted)
		if err != nil {
			return nil, err
		}
	}
	for _, source := range b.Encounters {
		weekID, weekOK := weekIDs[source.WeekID]
		team1ID, team1OK := teamIDs[source.Team1ID]
		team2ID, team2OK := teamIDs[source.Team2ID]
		if !weekOK || !team1OK || !team2OK || team1ID == team2ID {
			return nil, ErrInvalid
		}
		eid := source.ID
		if _, e := uuid.Parse(eid); e != nil {
			return nil, ErrInvalid
		}
		if in.Mode == "add" {
			eid = uuid.NewString()
		}
		status := source.Status
		if status != "pending" && status != "completed" {
			return nil, ErrInvalid
		}
		bouts := source.Bouts
		if len(bouts) == 0 {
			bouts = json.RawMessage(`[]`)
		}
		lineup := source.Lineup
		if len(lineup) == 0 {
			lineup = json.RawMessage(`{}`)
		}
		_, err = s.db.Exec(ctx, `INSERT INTO encounters(id,league_id,week_id,team1_id,team2_id,status,bouts,lineup) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, eid, id, weekID, team1ID, team2ID, status, bouts, lineup)
		if err != nil {
			return nil, err
		}
	}
	repo := tournaments.NewTournamentRepository(s.db, s.organizationID)
	ts := tournaments.NewTournamentService(repo)
	for _, source := range b.Tournaments {
		if source.LeagueID == nil || *source.LeagueID != b.League.ID || source.WeekID == nil {
			return nil, ErrInvalid
		}
		weekID, ok := weekIDs[*source.WeekID]
		if !ok {
			return nil, ErrInvalid
		}
		var groupID *string
		if source.GroupID != nil {
			mapped, yes := groupIDs[*source.GroupID]
			if !yes {
				return nil, ErrInvalid
			}
			groupID = &mapped
		}
		tid := source.ID
		if _, e := uuid.Parse(tid); e != nil {
			return nil, ErrInvalid
		}
		if in.Mode == "add" {
			tid = uuid.NewString()
		}
		stageOrder := source.StageOrder
		if stageOrder == nil {
			for _, stage := range b.League.Stages {
				for _, week := range stage.Weeks {
					if week.ID == *source.WeekID {
						order := stage.Position
						stageOrder = &order
					}
				}
			}
		}
		created := source.CreatedAt
		if created.IsZero() {
			created = now
		}
		blank := &tournaments.Tournament{ID: tid, Name: source.Name, Date: source.Date, Courts: source.Courts, Gender: source.Gender, AgeCategory: source.AgeCategory, Format: source.Format, LeagueID: &id, WeekID: &weekID, GroupID: groupID, StageOrder: stageOrder, CreatedAt: created, UpdatedAt: now}
		if err = repo.Create(ctx, blank); err != nil {
			return nil, err
		}
		athletes := append([]tournaments.TournamentAthlete(nil), source.Athletes...)
		athleteIDs := map[string]string{}
		for i := range athletes {
			originalID := athletes[i].ID
			if in.Mode == "add" {
				athletes[i].ID = uuid.NewString()
			}
			athleteIDs[originalID] = athletes[i].ID
			if athletes[i].ProfileID != nil {
				if mapped, ok := profileIDs[*athletes[i].ProfileID]; ok {
					athletes[i].ProfileID = &mapped
				} else {
					athletes[i].ProfileID = nil
				}
			}
		}
		matches := append([]tournaments.Match(nil), source.Matches...)
		matchIDs := map[string]string{}
		for i := range matches {
			originalID := matches[i].ID
			if in.Mode == "add" {
				matches[i].ID = uuid.NewString()
			}
			matchIDs[originalID] = matches[i].ID
		}
		for i := range matches {
			remapLeagueAthleteID(matches[i].Athlete1ID, athleteIDs)
			remapLeagueAthleteID(matches[i].Athlete2ID, athleteIDs)
			remapLeagueAthleteID(matches[i].WinnerID, athleteIDs)
			remapLeagueAthleteID(matches[i].NextMatchID, matchIDs)
			if matches[i].Result != nil {
				result := *matches[i].Result
				remapLeagueAthleteID(result.BlueID, athleteIDs)
				remapLeagueAthleteID(result.RedID, athleteIDs)
				remapLeagueAthleteID(result.WinnerID, athleteIDs)
				matches[i].Result = &result
			}
		}
		eliminationMatches := append([]tournaments.EliminationMatch(nil), source.EliminationMatches...)
		for i := range eliminationMatches {
			if in.Mode == "add" {
				eliminationMatches[i].ID = uuid.NewString()
			}
			remapLeagueAthleteID(eliminationMatches[i].WinnerID, athleteIDs)
		}
		_, err = ts.ImportSnapshot(ctx, tid, tournaments.ImportTournamentInput{Name: source.Name, Date: source.Date.Format(time.RFC3339), Courts: source.Courts, Gender: source.Gender, AgeCategory: source.AgeCategory, Format: source.Format, Athletes: athletes, Matches: matches, CourtAssignment: source.CourtAssignment, EliminationMatches: eliminationMatches, Settings: source.Settings})
		if err != nil {
			return nil, err
		}
		if _, err = s.Publish(ctx, id, tid); err != nil && !errors.Is(err, ErrIncomplete) {
			return nil, err
		}
	}
	return s.Get(ctx, id)
}

func remapLeagueAthleteID(value *string, ids map[string]string) {
	if value == nil {
		return
	}
	if mapped, ok := ids[*value]; ok {
		*value = mapped
	}
}

func (s *Service) resolveBackupClub(ctx context.Context, sourceID, name string) (string, error) {
	var id string
	if _, err := uuid.Parse(sourceID); err == nil {
		err = s.db.QueryRow(ctx, `SELECT id::text FROM clubs WHERE organization_id=$1 AND id=$2`, s.organizationID, sourceID).Scan(&id)
		if err == nil {
			return id, nil
		}
		if err != nil && err != pgx.ErrNoRows {
			return "", err
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalid
	}
	err := s.db.QueryRow(ctx, `INSERT INTO clubs(organization_id,name) VALUES($1,$2) ON CONFLICT(organization_id,name) DO UPDATE SET name=excluded.name RETURNING id::text`, s.organizationID, name).Scan(&id)
	return id, err
}

func (s *Service) resolveBackupProfile(ctx context.Context, sourceID, name, gender string) (string, error) {
	var id string
	if _, err := uuid.Parse(sourceID); err == nil {
		err = s.db.QueryRow(ctx, `SELECT id::text FROM athletes WHERE organization_id=$1 AND id=$2`, s.organizationID, sourceID).Scan(&id)
		if err == nil {
			return id, nil
		}
		if err != nil && err != pgx.ErrNoRows {
			return "", err
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalid
	}
	id = uuid.NewString()
	_, err := s.db.Exec(ctx, `INSERT INTO athletes(id,organization_id,name,gender) VALUES($1,$2,$3,$4)`, id, s.organizationID, name, gender)
	return id, err
}
