package platform

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"backend/internal/modules/leagues"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Public responses deliberately select competition data, never athlete profiles,
// weigh-in measurements/signatures, operator IDs or private result notes.
func (a *API) publicCompetitions(r *Request) (any, error) {
	offset := 0
	if raw := r.C.Query("offset"); raw != "" {
		var err error
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return nil, bad("invalid offset")
		}
	}
	rows, err := r.Tx.Query(r.Context(), `SELECT t.id::text,t.name,((now() AT TIME ZONE 'Asia/Tehran')::date)::text,t.gender,t.age_category,t.courts,COALESCE(l.name,''),o.name FROM tournaments t JOIN organizations o ON o.id=t.organization_id LEFT JOIN leagues l ON l.id=t.league_id WHERE (t.event_date=(now() AT TIME ZONE 'Asia/Tehran')::date OR EXISTS (SELECT 1 FROM jsonb_array_elements(COALESCE(t.settings->'schedule'->'days','[]'::jsonb)) d WHERE d->>'date'=((now() AT TIME ZONE 'Asia/Tehran')::date)::text)) AND ($1='' OR t.name ILIKE '%' || $1 || '%' OR l.name ILIKE '%' || $1 || '%') ORDER BY t.updated_at DESC,t.id LIMIT 51 OFFSET $2`, strings.TrimSpace(r.C.Query("q")), offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var id, name, date, gender, age, league, org string
		var courts int
		if err = rows.Scan(&id, &name, &date, &gender, &age, &courts, &league, &org); err != nil {
			return nil, err
		}
		items = append(items, gin.H{"id": id, "name": name, "date": date, "gender": gender, "ageCategory": age, "courts": courts, "leagueName": league, "organizer": org})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	more := len(items) > 50
	if more {
		items = items[:50]
	}
	r.C.Header("Cache-Control", "public, max-age=0, s-maxage=2")
	return gin.H{"items": items, "hasMore": more}, nil
}

func publicStandings(s *leagues.Standings) gin.H {
	convert := func(rows []leagues.Standing) []gin.H {
		out := []gin.H{}
		for _, p := range rows {
			out = append(out, gin.H{"rank": p.Rank, "name": p.Name, "clubName": p.ClubName, "coachName": p.CoachName, "weightCategory": p.WeightCategory, "groupName": p.GroupName, "totalPoints": p.TotalPoints, "gold": p.Gold, "silver": p.Silver, "bronze": p.Bronze, "wins": p.Wins, "losses": p.Losses})
		}
		return out
	}
	return gin.H{"players": convert(s.Players), "teams": convert(s.Teams)}
}

func (a *API) publicCompetition(r *Request) (any, error) {
	id := r.C.Param("tournament")
	err := r.Tx.QueryRow(r.Context(), `SELECT organization_id::text FROM tournaments WHERE id=$1`, id).Scan(&r.OrganizationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	t, err := r.Service().GetByID(r.Context(), id)
	if err != nil {
		return nil, err
	}
	location, locationErr := time.LoadLocation("Asia/Tehran")
	if locationErr != nil {
		location = time.FixedZone("Asia/Tehran", 3*60*60+30*60)
	}
	today := time.Now().In(location).Format("2006-01-02")
	activeDay := 0
	if t.Settings != nil {
		for _, day := range t.Settings.Schedule.Days {
			if day.Date == today {
				activeDay = day.Day
				break
			}
		}
	}
	if activeDay == 0 && t.Date.Format("2006-01-02") == today {
		activeDay = 1
	}
	responseDate := t.Date.Format("2006-01-02")
	if activeDay > 0 {
		responseDate = today
	}
	activeCategories := map[string]bool{}
	if t.Settings != nil {
		for category, day := range t.Settings.Schedule.CategoryDays {
			if day == activeDay {
				activeCategories[category] = true
			}
		}
	}
	athletes := []gin.H{}
	for _, p := range t.Athletes {
		if activeDay > 0 && len(activeCategories) > 0 && !activeCategories[p.WeightCategory] {
			continue
		}
		affiliation := p.Club
		if t.LeagueID != nil && affiliation == "" {
			affiliation = p.Coach
		}
		athletes = append(athletes, gin.H{"id": p.ID, "name": p.Name, "club": affiliation, "weightCategory": p.WeightCategory})
	}
	matches := []gin.H{}
	for _, m := range t.Matches {
		if activeDay > 0 && m.Day != activeDay {
			continue
		}
		var result any
		if m.Result != nil {
			result = gin.H{"winType": m.Result.WinType, "winnerCorner": m.Result.WinnerCorner, "blueId": m.Result.BlueID, "redId": m.Result.RedID, "rounds": m.Result.Rounds}
		}
		matches = append(matches, gin.H{"id": m.ID, "athlete1Id": m.Athlete1ID, "athlete2Id": m.Athlete2ID, "winnerId": m.WinnerID, "court": m.Court, "order": m.Order, "matchNumber": m.MatchNumber, "bracketIndex": m.BracketIndex, "status": m.Status, "round": m.Round, "side": m.Side, "weightCategory": m.WeightCategory, "nextMatchId": m.NextMatchID, "nextSlot": m.NextSlot, "isBye": m.IsBye, "result": result})
	}
	cfg := leagues.DefaultScoring()
	scope := "tournament"
	leagueName := ""
	var standings *leagues.Standings
	if t.LeagueID != nil {
		l, e := r.LeagueService().Get(r.Context(), *t.LeagueID)
		if e != nil {
			return nil, e
		}
		cfg = l.Settings.Scoring
		leagueName = l.Name
		scope = "league"
		standings, err = r.LeagueService().Standings(r.Context(), l.ID, nil)
		if err != nil {
			return nil, err
		}
	} else {
		events, e := leagues.ScoreTournament(t, cfg, nil)
		if e != nil {
			return nil, e
		}
		players := map[string]*leagues.Standing{}
		teams := map[string]*leagues.Standing{}
		for _, p := range t.Athletes {
			players[p.ID] = &leagues.Standing{Name: p.Name, ClubName: p.Club, WeightCategory: p.WeightCategory}
		}
		for _, event := range events {
			p := players[event.EntryID]
			if p == nil {
				continue
			}
			p.TotalPoints += event.Points
			p.Gold += event.Gold
			p.Silver += event.Silver
			p.Bronze += event.Bronze
			p.Wins += event.Wins
			p.Losses += event.Losses
			p.RoundDiff += event.RoundDiff
		}
		standings = &leagues.Standings{Players: []leagues.Standing{}, Teams: []leagues.Standing{}}
		for _, p := range players {
			standings.Players = append(standings.Players, *p)
			if p.ClubName == "" {
				continue
			}
			team := teams[p.ClubName]
			if team == nil {
				team = &leagues.Standing{Name: p.ClubName}
				teams[p.ClubName] = team
			}
			team.TotalPoints += p.TotalPoints
			team.Gold += p.Gold
			team.Silver += p.Silver
			team.Bronze += p.Bronze
			team.Wins += p.Wins
			team.Losses += p.Losses
			team.RoundDiff += p.RoundDiff
		}
		for _, team := range teams {
			standings.Teams = append(standings.Teams, *team)
		}
		leagues.Rank(standings.Players)
		leagues.Rank(standings.Teams)
	}
	r.C.Header("Cache-Control", "public, max-age=0, s-maxage=2")
	return gin.H{"id": t.ID, "name": t.Name, "date": responseDate, "activeDay": activeDay, "gender": t.Gender, "ageCategory": t.AgeCategory, "courts": t.Courts, "revision": t.Revision, "athletes": athletes, "matches": matches, "standings": publicStandings(standings), "rankingScope": scope, "leagueName": leagueName, "scoring": cfg}, nil
}
