package platform

import (
	p "backend/internal/modules/poomsae"
	"github.com/gin-gonic/gin"
)

func (a *API) poomsaeLeagueRoutes(org *gin.RouterGroup) {
	g := org.Group("/poomsae-leagues")
	allowed := func(r *Request, permission string) error {
		if err := r.permit(permission); err != nil {
			return err
		}
		if r.poomsaeLegacyRestricted() || len(r.Scope.PoomsaeEventIDs) > 0 || len(r.Scope.PoomsaeDivisionIDs) > 0 {
			return forbidden()
		}
		return nil
	}
	g.GET("", a.route(true, true, func(r *Request) (any, error) {
		if err := allowed(r, permPoomsaeRead); err != nil {
			return nil, err
		}
		return r.poomsaeRepo().PoomsaeLeagues(r.Context())
	}))
	g.POST("", a.route(true, true, func(r *Request) (any, error) {
		if err := allowed(r, permPoomsaeManage); err != nil {
			return nil, err
		}
		in, err := bind[p.CreatePoomsaeLeagueInput](r)
		if err != nil {
			return nil, err
		}
		return r.poomsaeRepo().CreatePoomsaeLeague(r.Context(), in)
	}))
	g.GET("/:poomsaeLeague", a.route(true, true, func(r *Request) (any, error) {
		if err := allowed(r, permPoomsaeRead); err != nil {
			return nil, err
		}
		repo := r.poomsaeRepo()
		league, err := repo.PoomsaeLeague(r.Context(), r.C.Param("poomsaeLeague"))
		if err != nil {
			return nil, err
		}
		standings, err := repo.PoomsaeLeagueStandings(r.Context(), league)
		if err != nil {
			return nil, err
		}
		return gin.H{"league": league, "standings": standings}, nil
	}))
	g.PUT("/:poomsaeLeague/weeks/:week", a.route(true, true, func(r *Request) (any, error) {
		if err := allowed(r, permPoomsaeManage); err != nil {
			return nil, err
		}
		in, err := bind[p.LinkPoomsaeLeagueWeekInput](r)
		if err != nil {
			return nil, err
		}
		return r.poomsaeRepo().LinkPoomsaeLeagueWeek(r.Context(), r.C.Param("poomsaeLeague"), r.C.Param("week"), in.EventID)
	}))
}
