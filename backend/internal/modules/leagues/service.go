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

var (
	ErrNotFound     = errors.New("league resource not found")
	ErrInvalid      = errors.New("invalid league input")
	ErrRosterLocked = errors.New("league roster is locked after weigh-in or draw")
	ErrIncomplete   = errors.New("tournament results are incomplete")
)

type Service struct {
	db                     pgx.Tx
	organizationID, userID string
}

func NewService(db pgx.Tx, organizationID, userID string) *Service {
	return &Service{db: db, organizationID: organizationID, userID: userID}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*League, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.SeasonName = strings.TrimSpace(in.SeasonName)
	gender := tournaments.Gender(in.Gender)
	age := tournaments.AgeCategory(in.AgeCategory)
	if in.Name == "" || !gender.IsValid() || !age.IsValid() || in.StageCount < 1 || in.StageCount > 10 || in.WeeksPerStage < 1 || in.WeeksPerStage > 100 {
		return nil, ErrInvalid
	}
	settings := Settings{Scoring: DefaultScoring(), TeamsToPromote: 2, AutoPromote: true}
	if in.Settings != nil {
		settings = *in.Settings
	}
	settings.SeasonName = in.SeasonName
	if err := validateSettings(settings); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(settings)
	id := uuid.NewString()
	now := time.Now().UTC()
	_, err := s.db.Exec(ctx, `INSERT INTO leagues(id,organization_id,name,config,gender,age_category,status,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,'draft',$7,$7)`, id, s.organizationID, in.Name, b, in.Gender, in.AgeCategory, now)
	if err != nil {
		return nil, err
	}
	for i := 1; i <= in.StageCount; i++ {
		sid := uuid.NewString()
		status := "pending"
		if i == 1 {
			status = "active"
		}
		_, err = s.db.Exec(ctx, `INSERT INTO league_stages(id,league_id,name,position,status) VALUES($1,$2,$3,$4,$5)`, sid, id, fmt.Sprintf("مرحله %d", i), i, status)
		if err != nil {
			return nil, err
		}
		for j := 1; j <= in.WeeksPerStage; j++ {
			_, err = s.db.Exec(ctx, `INSERT INTO league_weeks(league_id,stage_id,name,position) VALUES($1,$2,$3,$4)`, id, sid, fmt.Sprintf("هفته %d", j), j)
			if err != nil {
				return nil, err
			}
		}
	}
	for i, name := range in.GroupNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		_, err = s.db.Exec(ctx, `INSERT INTO league_groups(league_id,name,position) VALUES($1,$2,$3)`, id, name, i+1)
		if err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, id)
}

func validateSettings(v Settings) error {
	c := v.Scoring
	if c.Gold < 0 || c.Silver < 0 || c.Bronze < 0 || c.WeighInPoint < 0 || c.WinPoint < 0 || v.TeamsToPromote < 0 {
		return ErrInvalid
	}
	return nil
}

func (s *Service) UpdateSettings(ctx context.Context, id string, v Settings) (*League, error) {
	locked, err := s.structureLocked(ctx, id)
	if err != nil {
		return nil, err
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if locked && ((v.SeasonName != "" && v.SeasonName != current.Settings.SeasonName) || v.TeamsToPromote != current.Settings.TeamsToPromote || v.AutoPromote != current.Settings.AutoPromote) {
		return nil, ErrStructureLocked
	}
	if v.SeasonName == "" {
		current, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		v.SeasonName = current.Settings.SeasonName
	}
	if err := validateSettings(v); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(v)
	tag, err := s.db.Exec(ctx, `UPDATE leagues SET config=$3,revision=revision+1,updated_at=now() WHERE id=$1 AND organization_id=$2`, id, s.organizationID, b)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	// The request transaction keeps settings and published ledgers atomic.
	rows, err := s.db.Query(ctx, `SELECT tournament_id::text FROM league_tournament_publications WHERE league_id=$1 ORDER BY tournament_id`, id)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var tournamentID string
		if err = rows.Scan(&tournamentID); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, tournamentID)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, tournamentID := range ids {
		if _, err = s.Publish(ctx, id, tournamentID); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id string) (*League, error) {
	l := &League{}
	var cfg []byte
	err := s.db.QueryRow(ctx, `SELECT id::text,name,gender,age_category,status,config,revision,created_at,updated_at,structure_locked FROM leagues WHERE id=$1 AND organization_id=$2`, id, s.organizationID).Scan(&l.ID, &l.Name, &l.Gender, &l.AgeCategory, &l.Status, &cfg, &l.Revision, &l.CreatedAt, &l.UpdatedAt, &l.StructureLocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(cfg, &l.Settings); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `SELECT id::text,name,position,status,promoted_team_ids::text[] FROM league_stages WHERE league_id=$1 ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var x Stage
		if err = rows.Scan(&x.ID, &x.Name, &x.Position, &x.Status, &x.PromotedTeamIDs); err != nil {
			rows.Close()
			return nil, err
		}
		l.Stages = append(l.Stages, x)
	}
	rows.Close()
	for i := range l.Stages {
		rows, err = s.db.Query(ctx, `SELECT w.id::text,w.name,w.position,w.date_range,w.locked,t.id::text FROM league_weeks w LEFT JOIN tournaments t ON t.league_id=w.league_id AND t.week_id=w.id AND t.organization_id=$2 AND t.group_id IS NULL WHERE w.league_id=$1 AND w.stage_id=$3 ORDER BY w.position`, id, s.organizationID, l.Stages[i].ID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var x Week
			if err = rows.Scan(&x.ID, &x.Name, &x.Position, &x.DateRange, &x.Locked, &x.TournamentID); err != nil {
				rows.Close()
				return nil, err
			}
			l.Stages[i].Weeks = append(l.Stages[i].Weeks, x)
		}
		rows.Close()
	}
	rows, err = s.db.Query(ctx, `SELECT id::text,name,position FROM league_groups WHERE league_id=$1 ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var x Group
		if err = rows.Scan(&x.ID, &x.Name, &x.Position); err != nil {
			rows.Close()
			return nil, err
		}
		l.Groups = append(l.Groups, x)
	}
	rows.Close()
	rows, err = s.db.Query(ctx, `SELECT lt.id::text,lt.club_id::text,c.name,lt.group_id::text,lt.active FROM league_teams lt JOIN clubs c ON c.id=lt.club_id WHERE lt.league_id=$1 ORDER BY c.name`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var x Team
		if err = rows.Scan(&x.ID, &x.ClubID, &x.ClubName, &x.GroupID, &x.Active); err != nil {
			rows.Close()
			return nil, err
		}
		l.Teams = append(l.Teams, x)
	}
	rows.Close()
	rows, err = s.db.Query(ctx, `SELECT la.id::text,la.profile_id::text,a.name,la.team_id::text,la.group_id::text,la.club_name,la.coach_name,la.weight_category,la.ranking,la.active,la.created_at,la.updated_at FROM league_athletes la JOIN athletes a ON a.id=la.profile_id WHERE la.league_id=$1 ORDER BY a.name`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var x Athlete
		if err = rows.Scan(&x.ID, &x.ProfileID, &x.Name, &x.TeamID, &x.GroupID, &x.ClubName, &x.CoachName, &x.WeightCategory, &x.Ranking, &x.Active, &x.CreatedAt, &x.UpdatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		l.Athletes = append(l.Athletes, x)
	}
	rows.Close()
	if l.Stages == nil {
		l.Stages = []Stage{}
	}
	if l.Groups == nil {
		l.Groups = []Group{}
	}
	if l.Teams == nil {
		l.Teams = []Team{}
	}
	if l.Athletes == nil {
		l.Athletes = []Athlete{}
	}
	return l, nil
}

func (s *Service) List(ctx context.Context) ([]League, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text FROM leagues WHERE organization_id=$1 ORDER BY created_at DESC`, s.organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	out := []League{}
	for _, id := range ids {
		l, e := s.Get(ctx, id)
		if e != nil {
			return nil, e
		}
		out = append(out, *l)
	}
	return out, nil
}

func (s *Service) AddTeam(ctx context.Context, leagueID string, in AddTeamInput) (*League, error) {
	if _, err := uuid.Parse(in.ClubID); err != nil {
		return nil, ErrInvalid
	}
	var name string
	err := s.db.QueryRow(ctx, `SELECT name FROM clubs WHERE id=$1 AND organization_id=$2`, in.ClubID, s.organizationID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec(ctx, `INSERT INTO league_teams(league_id,club_id,group_id) VALUES($1,$2,$3)`, leagueID, in.ClubID, in.GroupID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, leagueID)
}

func (s *Service) UpdateTeam(ctx context.Context, leagueID, teamID string, in UpdateTeamInput) (*League, error) {
	locked, err := s.structureLocked(ctx, leagueID)
	if err != nil {
		return nil, err
	}
	if locked {
		var allowed bool
		if err = s.db.QueryRow(ctx, `SELECT active=$3 AND (group_id IS NOT DISTINCT FROM $4::uuid OR (group_id IS NULL AND $4::uuid IS NOT NULL)) FROM league_teams WHERE league_id=$1 AND id=$2`, leagueID, teamID, in.Active, in.GroupID).Scan(&allowed); err != nil {
			return nil, err
		}
		if !allowed {
			return nil, ErrStructureLocked
		}
	}
	tag, err := s.db.Exec(ctx, `UPDATE league_teams SET group_id=$3,active=$4 WHERE id=$1 AND league_id=$2`, teamID, leagueID, in.GroupID, in.Active)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, leagueID)
}
func (s *Service) AddGroup(ctx context.Context, leagueID string, in AddGroupInput) (*League, error) {
	if err := s.requireEditable(ctx, leagueID); err != nil {
		return nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, ErrInvalid
	}
	var position int
	err := s.db.QueryRow(ctx, `SELECT COALESCE(max(position),0)+1 FROM league_groups WHERE league_id=$1`, leagueID).Scan(&position)
	if err != nil {
		return nil, err
	}
	tag, err := s.db.Exec(ctx, `INSERT INTO league_groups(league_id,name,position) SELECT id,$2,$3 FROM leagues WHERE id=$1 AND organization_id=$4`, leagueID, in.Name, position, s.organizationID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, leagueID)
}
func (s *Service) UpdateGroup(ctx context.Context, leagueID, groupID string, in UpdateGroupInput) (*League, error) {
	if err := s.requireEditable(ctx, leagueID); err != nil {
		return nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, ErrInvalid
	}
	tag, err := s.db.Exec(ctx, `UPDATE league_groups SET name=$3 WHERE id=$1 AND league_id=$2`, groupID, leagueID, in.Name)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, leagueID)
}
func (s *Service) RemoveGroup(ctx context.Context, leagueID, groupID string) (*League, error) {
	if err := s.requireEditable(ctx, leagueID); err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(ctx, `UPDATE league_athletes SET group_id=NULL,updated_at=now() WHERE league_id=$1 AND group_id=$2`, leagueID, groupID); err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(ctx, `UPDATE league_teams SET group_id=NULL WHERE league_id=$1 AND group_id=$2`, leagueID, groupID); err != nil {
		return nil, err
	}
	tag, err := s.db.Exec(ctx, `DELETE FROM league_groups WHERE id=$1 AND league_id=$2`, groupID, leagueID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, leagueID)
}
func (s *Service) AddWeek(ctx context.Context, leagueID, stageID string, in AddWeekInput) (*League, error) {
	if err := s.requireEditable(ctx, leagueID); err != nil {
		return nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, ErrInvalid
	}
	var position int
	err := s.db.QueryRow(ctx, `SELECT COALESCE(max(position),0)+1 FROM league_weeks WHERE league_id=$1 AND stage_id=$2`, leagueID, stageID).Scan(&position)
	if err != nil {
		return nil, err
	}
	tag, err := s.db.Exec(ctx, `INSERT INTO league_weeks(league_id,stage_id,name,position,date_range) SELECT $1,id,$3,$4,$5 FROM league_stages WHERE id=$2 AND league_id=$1`, leagueID, stageID, in.Name, position, in.DateRange)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, leagueID)
}
func (s *Service) SetWeekLocked(ctx context.Context, leagueID, weekID string, locked bool) (*League, error) {
	tag, err := s.db.Exec(ctx, `UPDATE league_weeks SET locked=$3 WHERE id=$1 AND league_id=$2`, weekID, leagueID, locked)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, leagueID)
}
func (s *Service) CompleteStage(ctx context.Context, leagueID, stageID string) (*League, error) {
	l, err := s.Get(ctx, leagueID)
	if err != nil {
		return nil, err
	}
	var stage *Stage
	for i := range l.Stages {
		if l.Stages[i].ID == stageID {
			stage = &l.Stages[i]
			break
		}
	}
	if stage == nil {
		return nil, ErrNotFound
	}
	standings, err := s.Standings(ctx, leagueID, &stageID)
	if err != nil {
		return nil, err
	}
	teamGroup := map[string]string{}
	for _, team := range l.Teams {
		if team.GroupID != nil {
			teamGroup[team.ID] = *team.GroupID
		}
	}
	promoted := []string{}
	counts := map[string]int{}
	for _, row := range standings.Teams {
		group := teamGroup[row.ID]
		if len(l.Groups) == 0 {
			group = "all"
		}
		if counts[group] < l.Settings.TeamsToPromote {
			promoted = append(promoted, row.ID)
			counts[group]++
		}
	}
	_, err = s.db.Exec(ctx, `UPDATE league_stages SET status='completed',promoted_team_ids=$3 WHERE id=$1 AND league_id=$2`, stageID, leagueID, promoted)
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec(ctx, `UPDATE league_stages SET status='active' WHERE league_id=$1 AND position=$2 AND status='pending'`, leagueID, stage.Position+1)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, leagueID)
}

func (s *Service) AddAthlete(ctx context.Context, leagueID string, in AddAthleteInput) (*Athlete, error) {
	if in.TeamID == nil && strings.TrimSpace(in.CoachName) == "" {
		return nil, fmt.Errorf("%w: نام مربی برای بازیکن آزاد الزامی است", ErrInvalid)
	}
	if strings.TrimSpace(in.ProfileID) == "" {
		in.Name = strings.TrimSpace(in.Name)
		if in.Name == "" || (in.Gender != "male" && in.Gender != "female") {
			return nil, ErrInvalid
		}
		if in.ClubID != nil {
			var exists bool
			if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clubs WHERE id=$1 AND organization_id=$2)`, *in.ClubID, s.organizationID).Scan(&exists); err != nil || !exists {
				return nil, ErrInvalid
			}
		}
		if err := s.db.QueryRow(ctx, `INSERT INTO athletes(id,organization_id,name,gender,club_id) VALUES(gen_random_uuid(),$1,$2,$3,$4) RETURNING id::text`, s.organizationID, in.Name, in.Gender, in.ClubID).Scan(&in.ProfileID); err != nil {
			return nil, err
		}
	}
	var name, gender string
	err := s.db.QueryRow(ctx, `SELECT name,gender FROM athletes WHERE id=$1 AND organization_id=$2 AND is_active`, in.ProfileID, s.organizationID).Scan(&name, &gender)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var leagueGender, age, club string
	err = s.db.QueryRow(ctx, `SELECT gender,age_category FROM leagues WHERE id=$1 AND organization_id=$2`, leagueID, s.organizationID).Scan(&leagueGender, &age)
	if err != nil {
		return nil, ErrNotFound
	}
	if gender != leagueGender || !tournaments.IsValidWeightCategory(tournaments.AgeCategory(age), tournaments.Gender(leagueGender), strings.TrimSpace(in.WeightCategory)) {
		return nil, ErrInvalid
	}
	if in.TeamID != nil {
		err = s.db.QueryRow(ctx, `SELECT c.name FROM league_teams lt JOIN clubs c ON c.id=lt.club_id WHERE lt.id=$1 AND lt.league_id=$2`, *in.TeamID, leagueID).Scan(&club)
		if err != nil {
			return nil, ErrInvalid
		}
	}
	id := uuid.NewString()
	err = s.db.QueryRow(ctx, `INSERT INTO league_athletes(id,league_id,profile_id,team_id,group_id,club_name,coach_name,weight_category,ranking) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING created_at,updated_at`, id, leagueID, in.ProfileID, in.TeamID, in.GroupID, club, strings.TrimSpace(in.CoachName), strings.TrimSpace(in.WeightCategory), in.Ranking).Scan(new(time.Time), new(time.Time))
	if err != nil {
		return nil, err
	}
	l, err := s.Get(ctx, leagueID)
	if err != nil {
		return nil, err
	}
	for i := range l.Athletes {
		if l.Athletes[i].ID == id {
			return &l.Athletes[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) UpdateAthlete(ctx context.Context, leagueID, athleteID string, in UpdateAthleteInput) (*Athlete, error) {
	if in.TeamID == nil && strings.TrimSpace(in.CoachName) == "" {
		return nil, fmt.Errorf("%w: نام مربی برای بازیکن آزاد الزامی است", ErrInvalid)
	}
	locked, lockErr := s.structureLocked(ctx, leagueID)
	if lockErr != nil {
		return nil, lockErr
	}
	if locked {
		var unchanged bool
		if err := s.db.QueryRow(ctx, `SELECT team_id IS NOT DISTINCT FROM $3::uuid AND (group_id IS NOT DISTINCT FROM $4::uuid OR (team_id IS NULL AND group_id IS NULL AND $4::uuid IS NOT NULL)) AND ranking IS NOT DISTINCT FROM $5::int AND active=$6 FROM league_athletes WHERE id=$1 AND league_id=$2`, athleteID, leagueID, in.TeamID, in.GroupID, in.Ranking, in.Active).Scan(&unchanged); err != nil {
			return nil, err
		}
		if !unchanged {
			return nil, ErrStructureLocked
		}
	}
	var gender, age, club string
	err := s.db.QueryRow(ctx, `SELECT gender,age_category FROM leagues WHERE id=$1 AND organization_id=$2`, leagueID, s.organizationID).Scan(&gender, &age)
	if err != nil {
		return nil, ErrNotFound
	}
	if !tournaments.IsValidWeightCategory(tournaments.AgeCategory(age), tournaments.Gender(gender), in.WeightCategory) {
		return nil, ErrInvalid
	}
	if in.TeamID != nil {
		err = s.db.QueryRow(ctx, `SELECT c.name FROM league_teams lt JOIN clubs c ON c.id=lt.club_id WHERE lt.id=$1 AND lt.league_id=$2`, *in.TeamID, leagueID).Scan(&club)
		if err != nil {
			return nil, ErrInvalid
		}
	}
	tag, err := s.db.Exec(ctx, `UPDATE league_athletes SET team_id=$3,group_id=$4,club_name=$5,coach_name=$6,weight_category=$7,ranking=$8,active=$9,updated_at=now() WHERE id=$1 AND league_id=$2`, athleteID, leagueID, in.TeamID, in.GroupID, club, strings.TrimSpace(in.CoachName), in.WeightCategory, in.Ranking, in.Active)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	if err = s.syncAthleteEntries(ctx, leagueID, athleteID, club, strings.TrimSpace(in.CoachName), in.WeightCategory, in.Ranking, ""); err != nil {
		return nil, err
	}
	l, err := s.Get(ctx, leagueID)
	if err != nil {
		return nil, err
	}
	for i := range l.Athletes {
		if l.Athletes[i].ID == athleteID {
			return &l.Athletes[i], nil
		}
	}
	return nil, ErrNotFound
}

// SyncTournamentAthlete persists a linked tournament edit in the league and all
// other tournament entries. The caller's transaction makes the operation atomic.
func (s *Service) SyncTournamentAthlete(ctx context.Context, leagueID, athleteID string, in tournaments.UpdateAthleteInput) error {
	var teamID *string
	var club string
	err := s.db.QueryRow(ctx, `SELECT team_id::text,club_name FROM league_athletes WHERE id=$1 AND league_id=$2`, athleteID, leagueID).Scan(&teamID, &club)
	if err != nil {
		return err
	}
	if teamID == nil && strings.TrimSpace(in.Coach) == "" {
		return fmt.Errorf("%w: نام مربی برای بازیکن آزاد الزامی است", ErrInvalid)
	}
	if club != strings.TrimSpace(in.Club) {
		return ErrStructureLocked
	}
	_, err = s.db.Exec(ctx, `UPDATE league_athletes SET coach_name=$3,weight_category=$4,ranking=$5,updated_at=now() WHERE id=$1 AND league_id=$2`, athleteID, leagueID, strings.TrimSpace(in.Coach), in.WeightCategory, in.Ranking)
	if err != nil {
		return err
	}
	var profileID string
	if err = s.db.QueryRow(ctx, `SELECT profile_id::text FROM league_athletes WHERE id=$1 AND league_id=$2`, athleteID, leagueID).Scan(&profileID); err != nil {
		return err
	}
	if _, err = s.db.Exec(ctx, `UPDATE athletes SET name=$2,updated_at=now() WHERE id=$1 AND organization_id=$3`, profileID, strings.TrimSpace(in.Name), s.organizationID); err != nil {
		return err
	}
	return s.syncAthleteEntries(ctx, leagueID, athleteID, club, strings.TrimSpace(in.Coach), in.WeightCategory, in.Ranking, "")
}

func (s *Service) syncAthleteEntries(ctx context.Context, leagueID, athleteID, club, coach, weight string, ranking *int, skipTournament string) error {
	rows, err := s.db.Query(ctx, `SELECT e.tournament_id::text,e.athlete_id::text FROM tournament_entries e JOIN tournaments t ON t.id=e.tournament_id WHERE t.league_id=$1 AND e.athlete_id=$2`, leagueID, athleteID)
	if err != nil {
		return err
	}
	type entry struct{ tournamentID, athleteID string }
	entries := []entry{}
	for rows.Next() {
		var e entry
		if err = rows.Scan(&e.tournamentID, &e.athleteID); err != nil {
			rows.Close()
			return err
		}
		entries = append(entries, e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	ts := tournaments.NewTournamentService(tournaments.NewTournamentRepository(s.db, s.organizationID))
	for _, e := range entries {
		if e.tournamentID == skipTournament {
			continue
		}
		t, getErr := ts.GetByID(ctx, e.tournamentID)
		if getErr != nil {
			return getErr
		}
		for _, a := range t.Athletes {
			if a.ID == e.athleteID {
				var name string
				if err = s.db.QueryRow(ctx, `SELECT a.name FROM league_athletes la JOIN athletes a ON a.id=la.profile_id WHERE la.id=$1`, athleteID).Scan(&name); err != nil {
					return err
				}
				_, err = ts.UpdateAthlete(ctx, e.tournamentID, e.athleteID, tournaments.UpdateAthleteInput{Name: name, Club: club, Coach: coach, WeightCategory: weight, Ranking: ranking})
				if err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func (s *Service) RemoveAthlete(ctx context.Context, leagueID, athleteID string) error {
	if err := s.requireEditable(ctx, leagueID); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `UPDATE league_athletes SET active=false,updated_at=now() WHERE id=$1 AND league_id=$2`, athleteID, leagueID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Service) CreateTournament(ctx context.Context, leagueID string, in CreateTournamentInput) (*tournaments.Tournament, error) {
	var gender, age string
	var position int
	var locked bool
	err := s.db.QueryRow(ctx, `SELECT l.gender,l.age_category,st.position,w.locked FROM leagues l JOIN league_stages st ON st.league_id=l.id JOIN league_weeks w ON w.stage_id=st.id AND w.league_id=l.id WHERE l.id=$1 AND l.organization_id=$2 AND st.id=$3 AND w.id=$4`, leagueID, s.organizationID, in.StageID, in.WeekID).Scan(&gender, &age, &position, &locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if locked {
		return nil, ErrRosterLocked
	}
	ts := tournaments.NewTournamentService(tournaments.NewTournamentRepository(s.db, s.organizationID))
	t, err := ts.Create(ctx, tournaments.CreateTournamentInput{Name: in.Name, Date: in.Date, Courts: in.Courts, Gender: tournaments.Gender(gender), AgeCategory: tournaments.AgeCategory(age), Format: tournaments.TournamentFormatGrandPrix, LeagueID: &leagueID, StageOrder: &position, WeekID: &in.WeekID, GroupID: in.GroupID})
	if err != nil {
		return nil, err
	}
	if _, err = s.SyncRoster(ctx, leagueID, t.ID); err != nil {
		return nil, err
	}
	return ts.GetByID(ctx, t.ID)
}

func (s *Service) SyncRoster(ctx context.Context, leagueID, tournamentID string) (*tournaments.Tournament, error) {
	ts := tournaments.NewTournamentService(tournaments.NewTournamentRepository(s.db, s.organizationID))
	t, err := ts.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	if t.LeagueID == nil || *t.LeagueID != leagueID {
		return nil, ErrNotFound
	}
	if len(t.Matches) > 0 {
		return nil, ErrRosterLocked
	}
	for _, a := range t.Athletes {
		if a.WeighIn != nil && (a.WeighIn.Status != tournaments.WeighInPending || len(a.WeighIn.Attempts) > 0) {
			return nil, ErrRosterLocked
		}
	}
	rows, err := s.db.Query(ctx, `SELECT la.id::text,la.profile_id::text,a.name,la.club_name,la.coach_name,la.weight_category,la.ranking FROM league_athletes la JOIN athletes a ON a.id=la.profile_id LEFT JOIN league_teams lt ON lt.id=la.team_id WHERE la.league_id=$1 AND la.active AND (la.group_id=$2 OR (la.group_id IS NULL AND lt.group_id=$2) OR $2::uuid IS NULL) ORDER BY a.name`, leagueID, t.GroupID)
	if err != nil {
		return nil, err
	}
	type seed struct {
		id, profile, name, club, coach, weight string
		ranking                                *int
	}
	seeds := []seed{}
	for rows.Next() {
		var x seed
		if err = rows.Scan(&x.id, &x.profile, &x.name, &x.club, &x.coach, &x.weight, &x.ranking); err != nil {
			rows.Close()
			return nil, err
		}
		seeds = append(seeds, x)
	}
	rows.Close()
	previous, err := s.previousWeekSeeds(ctx, t)
	if err != nil {
		return nil, err
	}
	for i := range seeds {
		if ranks, ok := previous[seeds[i].weight]; ok {
			seeds[i].ranking = nil
			if rank, found := ranks[seeds[i].profile]; found {
				value := rank
				seeds[i].ranking = &value
			}
		}
	}
	existing := map[string]tournaments.TournamentAthlete{}
	for _, a := range t.Athletes {
		if a.ProfileID != nil {
			existing[*a.ProfileID] = a
		}
	}
	for _, x := range seeds {
		if old, ok := existing[x.profile]; ok {
			// Refresh already-created, unlocked weeks as well as new snapshots.
			if _, err = s.db.Exec(ctx, `UPDATE tournament_entries SET data=jsonb_set(jsonb_set(jsonb_set(data,'{ranking}',COALESCE(to_jsonb($3::int),'null'::jsonb)),'{coach}',to_jsonb($4::text)),'{club}',to_jsonb($5::text)) WHERE tournament_id=$1 AND athlete_id=$2`, t.ID, old.ID, x.ranking, x.coach, x.club); err != nil {
				return nil, err
			}
			continue
		}
		pid := x.profile
		created, addErr := ts.AddAthlete(ctx, t.ID, tournaments.AddAthleteInput{ID: x.id, ProfileID: &pid, Name: x.name, Club: x.club, Coach: x.coach, WeightCategory: x.weight, Ranking: x.ranking})
		if addErr != nil {
			return nil, addErr
		}
		// League membership, not the global profile's club, determines this entry's affiliation.
		_, err = s.db.Exec(ctx, `UPDATE tournament_entries SET data=jsonb_set(jsonb_set(data,'{club}',to_jsonb($3::text)),'{coach}',to_jsonb($4::text)) WHERE tournament_id=$1 AND athlete_id=$2`, t.ID, created.ID, x.club, x.coach)
		if err != nil {
			return nil, err
		}
	}
	// Removed or deactivated league athletes disappear only while the tournament is still unlocked.
	allowed := map[string]bool{}
	for _, x := range seeds {
		allowed[x.profile] = true
	}
	t, err = ts.GetByID(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	for _, a := range t.Athletes {
		if a.ProfileID != nil && !allowed[*a.ProfileID] {
			if err = ts.RemoveAthlete(ctx, t.ID, a.ID); err != nil {
				return nil, err
			}
		}
	}
	return ts.GetByID(ctx, t.ID)
}

func (s *Service) Publish(ctx context.Context, leagueID, tournamentID string) (*Standings, error) {
	ts := tournaments.NewTournamentService(tournaments.NewTournamentRepository(s.db, s.organizationID))
	t, err := ts.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	if t.LeagueID == nil || *t.LeagueID != leagueID || t.WeekID == nil {
		return nil, ErrNotFound
	}
	var stageID string
	var cfgBytes []byte
	err = s.db.QueryRow(ctx, `SELECT st.id::text,l.config FROM leagues l JOIN league_stages st ON st.league_id=l.id JOIN league_weeks w ON w.stage_id=st.id WHERE l.id=$1 AND l.organization_id=$2 AND w.id=$3`, leagueID, s.organizationID, *t.WeekID).Scan(&stageID, &cfgBytes)
	if err != nil {
		return nil, ErrNotFound
	}
	var settings Settings
	if err = json.Unmarshal(cfgBytes, &settings); err != nil {
		return nil, err
	}
	teams := map[string]*string{}
	rows, err := s.db.Query(ctx, `SELECT la.id::text,la.team_id::text FROM league_athletes la WHERE la.league_id=$1`, leagueID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		var team *string
		if err = rows.Scan(&id, &team); err != nil {
			rows.Close()
			return nil, err
		}
		teams[id] = team
	}
	rows.Close()
	events, err := ScoreTournament(t, settings.Scoring, teams)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIncomplete, err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM league_point_events WHERE tournament_id=$1`, t.ID); err != nil {
		return nil, err
	}
	for _, e := range events {
		var profile any
		if e.ProfileID != "" {
			profile = e.ProfileID
		}
		_, err = tx.Exec(ctx, `INSERT INTO league_point_events(league_id,stage_id,tournament_id,tournament_entry_id,profile_id,team_id,athlete_name,club_name,weight_category,event_type,event_key,points,gold,silver,bronze,round_diff,wins_2_0,wins_2_1,losses_0_2,losses_1_2,wins,losses) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`, leagueID, stageID, t.ID, e.EntryID, profile, e.TeamID, e.AthleteName, e.ClubName, e.WeightCategory, e.EventType, e.EventKey, e.Points, e.Gold, e.Silver, e.Bronze, e.RoundDiff, e.Wins20, e.Wins21, e.Losses02, e.Losses12, e.Wins, e.Losses)
		if err != nil {
			return nil, err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO league_tournament_publications(tournament_id,league_id,tournament_revision,published_by) VALUES($1,$2,$3,$4) ON CONFLICT(tournament_id) DO UPDATE SET tournament_revision=excluded.tournament_revision,published_at=now(),published_by=excluded.published_by`, t.ID, leagueID, t.Revision, s.userID)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Standings(ctx, leagueID, &stageID)
}

func (s *Service) Standings(ctx context.Context, leagueID string, stageID *string) (*Standings, error) {
	// A tournament can contain a guest/legacy entry without a global athlete
	// profile. Keep those entries visible in live standings by using the stable
	// tournament entry id as their identity instead of scanning a NULL UUID.
	query := `SELECT COALESCE(profile_id::text,tournament_entry_id::text),athlete_name,club_name,weight_category,sum(points),sum(gold),sum(silver),sum(bronze),sum(CASE WHEN event_type='weigh_in' THEN points ELSE 0 END),sum(CASE WHEN event_type='match_stats' THEN points ELSE 0 END),sum(round_diff),sum(wins_2_0),sum(wins_2_1),sum(losses_0_2),sum(losses_1_2),sum(wins),sum(losses) FROM league_point_events WHERE league_id=$1 AND ($2::uuid IS NULL OR stage_id=$2) GROUP BY profile_id,tournament_entry_id,athlete_name,club_name,weight_category`
	rows, err := s.db.Query(ctx, query, leagueID, stageID)
	if err != nil {
		return nil, err
	}
	out := &Standings{Players: []Standing{}, Teams: []Standing{}}
	for rows.Next() {
		var x Standing
		if err = rows.Scan(&x.ID, &x.Name, &x.ClubName, &x.WeightCategory, &x.TotalPoints, &x.Gold, &x.Silver, &x.Bronze, &x.WeighInPoints, &x.WinPoints, &x.RoundDiff, &x.Wins20, &x.Wins21, &x.Losses02, &x.Losses12, &x.Wins, &x.Losses); err != nil {
			rows.Close()
			return nil, err
		}
		out.Players = append(out.Players, x)
	}
	rows.Close()
	playerGroups := map[string][2]string{}
	playerCoaches := map[string]string{}
	groupRows, groupErr := s.db.Query(ctx, `SELECT la.profile_id::text,COALESCE(la.group_id,lt.group_id)::text,COALESCE(g.name,''),la.coach_name FROM league_athletes la LEFT JOIN league_teams lt ON lt.id=la.team_id LEFT JOIN league_groups g ON g.id=COALESCE(la.group_id,lt.group_id) AND g.league_id=la.league_id WHERE la.league_id=$1`, leagueID)
	if groupErr != nil {
		return nil, groupErr
	}
	for groupRows.Next() {
		var profile, coach string
		var groupID *string
		var name string
		if err = groupRows.Scan(&profile, &groupID, &name, &coach); err != nil {
			groupRows.Close()
			return nil, err
		}
		playerCoaches[profile] = coach
		if groupID != nil {
			playerGroups[profile] = [2]string{*groupID, name}
		}
	}
	groupRows.Close()
	for i := range out.Players {
		out.Players[i].CoachName = playerCoaches[out.Players[i].ID]
		if group, ok := playerGroups[out.Players[i].ID]; ok {
			out.Players[i].GroupID = group[0]
			out.Players[i].GroupName = group[1]
		}
	}
	Rank(out.Players)
	// Keep each tournament's best medal per team and weight, including its points.
	// Individual ledgers and non-medal points remain unchanged.
	rows, err = s.db.Query(ctx, `WITH ranked_events AS (
		SELECT *, row_number() OVER (
			PARTITION BY tournament_id,team_id,weight_category,(event_type IN ('gold','silver','bronze'))
			ORDER BY CASE event_type WHEN 'gold' THEN 1 WHEN 'silver' THEN 2 WHEN 'bronze' THEN 3 ELSE 4 END,tournament_id,event_key
		) AS medal_rank
		FROM league_point_events
		WHERE league_id=$1 AND team_id IS NOT NULL AND ($2::uuid IS NULL OR stage_id=$2)
	) SELECT team_id::text,max(club_name),sum(points),sum(gold),sum(silver),sum(bronze),sum(CASE WHEN event_type='weigh_in' THEN points ELSE 0 END),sum(CASE WHEN event_type='match_stats' THEN points ELSE 0 END),sum(round_diff),sum(wins_2_0),sum(wins_2_1),sum(losses_0_2),sum(losses_1_2),sum(wins),sum(losses)
	FROM ranked_events WHERE event_type NOT IN ('gold','silver','bronze') OR medal_rank=1
	GROUP BY team_id`, leagueID, stageID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var x Standing
		if err = rows.Scan(&x.ID, &x.Name, &x.TotalPoints, &x.Gold, &x.Silver, &x.Bronze, &x.WeighInPoints, &x.WinPoints, &x.RoundDiff, &x.Wins20, &x.Wins21, &x.Losses02, &x.Losses12, &x.Wins, &x.Losses); err != nil {
			rows.Close()
			return nil, err
		}
		out.Teams = append(out.Teams, x)
	}
	rows.Close()
	teamGroups := map[string][2]string{}
	groupRows, groupErr = s.db.Query(ctx, `SELECT lt.id::text,COALESCE(lt.group_id::text,''),COALESCE(g.name,'') FROM league_teams lt LEFT JOIN league_groups g ON g.id=lt.group_id AND g.league_id=lt.league_id WHERE lt.league_id=$1`, leagueID)
	if groupErr != nil {
		return nil, groupErr
	}
	for groupRows.Next() {
		var teamID, groupID, name string
		if err = groupRows.Scan(&teamID, &groupID, &name); err != nil {
			groupRows.Close()
			return nil, err
		}
		teamGroups[teamID] = [2]string{groupID, name}
	}
	groupRows.Close()
	for i := range out.Teams {
		if group, ok := teamGroups[out.Teams[i].ID]; ok {
			out.Teams[i].GroupID = group[0]
			out.Teams[i].GroupName = group[1]
		}
	}
	Rank(out.Teams)
	return out, nil
}
