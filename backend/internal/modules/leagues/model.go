package leagues

import "time"

type ScoringConfig struct {
	Gold         int  `json:"gold"`
	Silver       int  `json:"silver"`
	Bronze       int  `json:"bronze"`
	WeighInPoint int  `json:"weighInPoint"`
	WinPoint     int  `json:"winPoint"`
	CountWeighIn bool `json:"countWeighIn"`
	CountWin     bool `json:"countWin"`
}

func DefaultScoring() ScoringConfig {
	return ScoringConfig{Gold: 10, Silver: 6, Bronze: 3, WeighInPoint: 1, WinPoint: 2, CountWeighIn: true, CountWin: true}
}

type Settings struct {
	Scoring        ScoringConfig `json:"scoring"`
	TeamsToPromote int           `json:"teamsToPromote"`
	AutoPromote    bool          `json:"autoPromote"`
}

type League struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Gender      string    `json:"gender"`
	AgeCategory string    `json:"ageCategory"`
	Status      string    `json:"status"`
	Settings    Settings  `json:"settings"`
	Revision    int64     `json:"revision"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Stages      []Stage   `json:"stages"`
	Groups      []Group   `json:"groups"`
	Teams       []Team    `json:"teams"`
	Athletes    []Athlete `json:"athletes"`
}

type Stage struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	Position        int      `json:"position"`
	Weeks           []Week   `json:"weeks"`
	PromotedTeamIDs []string `json:"promotedTeamIds"`
}
type Week struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Position  int     `json:"position"`
	DateRange *string `json:"dateRange,omitempty"`
	Locked    bool    `json:"locked"`
}
type Group struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}
type Team struct {
	ID       string  `json:"id"`
	ClubID   string  `json:"clubId"`
	ClubName string  `json:"clubName"`
	GroupID  *string `json:"groupId,omitempty"`
	Active   bool    `json:"active"`
}
type Athlete struct {
	ID             string    `json:"id"`
	ProfileID      string    `json:"profileId"`
	Name           string    `json:"name"`
	WeightCategory string    `json:"weightCategory"`
	TeamID         *string   `json:"teamId,omitempty"`
	GroupID        *string   `json:"groupId,omitempty"`
	ClubName       string    `json:"clubName"`
	CoachName      string    `json:"coachName"`
	Ranking        *int      `json:"ranking,omitempty"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Name          string    `json:"name"`
	Gender        string    `json:"gender"`
	AgeCategory   string    `json:"ageCategory"`
	SeasonName    string    `json:"seasonName"`
	StageCount    int       `json:"stageCount"`
	WeeksPerStage int       `json:"weeksPerStage"`
	GroupNames    []string  `json:"groupNames"`
	Settings      *Settings `json:"settings"`
}

type AddTeamInput struct {
	ClubID  string  `json:"clubId"`
	GroupID *string `json:"groupId"`
}
type UpdateTeamInput struct {
	GroupID *string `json:"groupId"`
	Active  bool    `json:"active"`
}
type AddGroupInput struct {
	Name string `json:"name"`
}
type UpdateGroupInput struct {
	Name string `json:"name"`
}
type AddWeekInput struct {
	Name      string  `json:"name"`
	DateRange *string `json:"dateRange"`
}
type LockWeekInput struct {
	Locked bool `json:"locked"`
}
type AddAthleteInput struct {
	ProfileID      string  `json:"profileId"`
	Name           string  `json:"name"`
	Gender         string  `json:"gender"`
	ClubID         *string `json:"clubId"`
	TeamID         *string `json:"teamId"`
	GroupID        *string `json:"groupId"`
	WeightCategory string  `json:"weightCategory"`
	CoachName      string  `json:"coachName"`
	Ranking        *int    `json:"ranking"`
}
type UpdateAthleteInput struct {
	TeamID         *string `json:"teamId"`
	GroupID        *string `json:"groupId"`
	WeightCategory string  `json:"weightCategory"`
	CoachName      string  `json:"coachName"`
	Ranking        *int    `json:"ranking"`
	Active         bool    `json:"active"`
}

type CreateTournamentInput struct {
	Name    string  `json:"name"`
	Date    string  `json:"date"`
	Courts  int     `json:"courts"`
	StageID string  `json:"stageId"`
	WeekID  string  `json:"weekId"`
	GroupID *string `json:"groupId"`
}

type PointEvent struct {
	EntryID, ProfileID, AthleteName, ClubName, WeightCategory, EventType, EventKey            string
	TeamID                                                                                    *string
	Points, Gold, Silver, Bronze, RoundDiff, Wins20, Wins21, Losses02, Losses12, Wins, Losses int
}

type Standing struct {
	Rank           int    `json:"rank"`
	ID             string `json:"id"`
	Name           string `json:"name"`
	ClubName       string `json:"clubName,omitempty"`
	WeightCategory string `json:"weightCategory,omitempty"`
	GroupID        string `json:"groupId,omitempty"`
	GroupName      string `json:"groupName,omitempty"`
	TotalPoints    int    `json:"totalPoints"`
	Gold           int    `json:"gold"`
	Silver         int    `json:"silver"`
	Bronze         int    `json:"bronze"`
	WeighInPoints  int    `json:"weighInPoints"`
	WinPoints      int    `json:"winPoints"`
	RoundDiff      int    `json:"roundDiff"`
	Wins20         int    `json:"wins2_0"`
	Wins21         int    `json:"wins2_1"`
	Losses02       int    `json:"losses0_2"`
	Losses12       int    `json:"losses1_2"`
	Wins           int    `json:"totalWins"`
	Losses         int    `json:"totalLosses"`
}

type Standings struct {
	Players []Standing `json:"players"`
	Teams   []Standing `json:"teams"`
}
