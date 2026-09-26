package platform

import (
	"backend/internal/modules/leagues"
	"github.com/gin-gonic/gin"
)

func (r *Request) LeagueService() *leagues.Service {
	return leagues.NewService(r.Tx, r.OrganizationID, r.UserID)
}
func leagueBody[T any](a *API, fn func(*Request, T) (any, error)) gin.HandlerFunc {
	return a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permLeaguesManage); err != nil {
			return nil, err
		}
		in, err := bind[T](r)
		if err != nil {
			return nil, err
		}
		return fn(r, in)
	})
}
func (a *API) leagueRoutes(org *gin.RouterGroup) {
	g := org.Group("/leagues")
	g.POST("", leagueBody(a, func(r *Request, in leagues.CreateInput) (any, error) {
		return r.LeagueService().Create(r.Context(), in)
	}))
	g.GET("", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permLeaguesRead); err != nil {
			return nil, err
		}
		return r.LeagueService().List(r.Context())
	}))
	g.GET("/:league", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permLeaguesRead); err != nil {
			return nil, err
		}
		return r.LeagueService().Get(r.Context(), r.C.Param("league"))
	}))
	g.PUT("/:league/settings", leagueBody(a, func(r *Request, in leagues.Settings) (any, error) {
		return r.LeagueService().UpdateSettings(r.Context(), r.C.Param("league"), in)
	}))
	g.POST("/:league/groups", leagueBody(a, func(r *Request, in leagues.AddGroupInput) (any, error) {
		return r.LeagueService().AddGroup(r.Context(), r.C.Param("league"), in)
	}))
	g.PUT("/:league/groups/:group", leagueBody(a, func(r *Request, in leagues.UpdateGroupInput) (any, error) {
		return r.LeagueService().UpdateGroup(r.Context(), r.C.Param("league"), r.C.Param("group"), in)
	}))
	g.DELETE("/:league/groups/:group", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permLeaguesManage); err != nil {
			return nil, err
		}
		return r.LeagueService().RemoveGroup(r.Context(), r.C.Param("league"), r.C.Param("group"))
	}))
	g.POST("/:league/teams", leagueBody(a, func(r *Request, in leagues.AddTeamInput) (any, error) {
		return r.LeagueService().AddTeam(r.Context(), r.C.Param("league"), in)
	}))
	g.PUT("/:league/teams/:team", leagueBody(a, func(r *Request, in leagues.UpdateTeamInput) (any, error) {
		return r.LeagueService().UpdateTeam(r.Context(), r.C.Param("league"), r.C.Param("team"), in)
	}))
	g.POST("/:league/stages/:stage/weeks", leagueBody(a, func(r *Request, in leagues.AddWeekInput) (any, error) {
		return r.LeagueService().AddWeek(r.Context(), r.C.Param("league"), r.C.Param("stage"), in)
	}))
	g.PUT("/:league/weeks/:week/lock", leagueBody(a, func(r *Request, in leagues.LockWeekInput) (any, error) {
		return r.LeagueService().SetWeekLocked(r.Context(), r.C.Param("league"), r.C.Param("week"), in.Locked)
	}))
	g.POST("/:league/stages/:stage/complete", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permLeaguesManage); err != nil {
			return nil, err
		}
		return r.LeagueService().CompleteStage(r.Context(), r.C.Param("league"), r.C.Param("stage"))
	}))
	g.POST("/:league/athletes", leagueBody(a, func(r *Request, in leagues.AddAthleteInput) (any, error) {
		return r.LeagueService().AddAthlete(r.Context(), r.C.Param("league"), in)
	}))
	g.PUT("/:league/athletes/:leagueAthlete", leagueBody(a, func(r *Request, in leagues.UpdateAthleteInput) (any, error) {
		return r.LeagueService().UpdateAthlete(r.Context(), r.C.Param("league"), r.C.Param("leagueAthlete"), in)
	}))
	g.DELETE("/:league/athletes/:leagueAthlete", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permLeaguesManage); err != nil {
			return nil, err
		}
		return nil, r.LeagueService().RemoveAthlete(r.Context(), r.C.Param("league"), r.C.Param("leagueAthlete"))
	}))
	g.POST("/:league/tournaments", leagueBody(a, func(r *Request, in leagues.CreateTournamentInput) (any, error) {
		return r.LeagueService().CreateTournament(r.Context(), r.C.Param("league"), in)
	}))
	g.POST("/:league/tournaments/:tournament/sync-roster", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permLeaguesManage); err != nil {
			return nil, err
		}
		return r.LeagueService().SyncRoster(r.Context(), r.C.Param("league"), r.C.Param("tournament"))
	}))
	g.POST("/:league/tournaments/:tournament/publish", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permLeaguesManage); err != nil {
			return nil, err
		}
		return r.LeagueService().Publish(r.Context(), r.C.Param("league"), r.C.Param("tournament"))
	}))
	g.GET("/:league/standings", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permLeaguesRead); err != nil {
			return nil, err
		}
		var stage *string
		if v := r.C.Query("stageId"); v != "" {
			stage = &v
		}
		return r.LeagueService().Standings(r.Context(), r.C.Param("league"), stage)
	}))
}
