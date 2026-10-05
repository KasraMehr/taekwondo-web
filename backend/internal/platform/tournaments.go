package platform

import (
	"backend/internal/modules/tournaments"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

func refreshLeagueStandings(r *Request, tournamentID string) error {
	tournament, err := r.Service().GetByID(r.Context(), tournamentID)
	if err != nil {
		return err
	}
	if tournament.LeagueID == nil || *tournament.LeagueID == "" {
		return nil
	}
	_, err = r.LeagueService().Publish(r.Context(), *tournament.LeagueID, tournamentID)
	return err
}

func tournamentBody[T any](a *API, permission string, fn func(*Request, T) (any, error)) gin.HandlerFunc {
	return a.route(true, true, func(r *Request) (any, error) {
		var err error
		if id := r.C.Param("tournament"); id != "" {
			if matchID := r.C.Param("match"); matchID != "" && permission == permMatchesOperate {
				err = r.authorizeMatch(permission, id, matchID)
			} else {
				err = r.authorizeTournament(permission, id)
			}
		} else {
			err = r.permit(permission)
		}
		if err != nil {
			return nil, err
		}
		in, err := bind[T](r)
		if err != nil {
			return nil, err
		}
		return fn(r, in)
	})
}
func (a *API) tournamentRoutes(org *gin.RouterGroup) {
	org.GET("/tournament-settings-options", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permTournamentsRead); err != nil {
			return nil, err
		}
		return gin.H{
			"scheduleModes":      []gin.H{{"value": "single_court", "label": "همه بازی‌ها روی یک زمین"}, {"value": "split_halves", "label": "دو نیمه جدول روی دو زمین تا فینال"}, {"value": "balanced", "label": "توزیع متوازن بازی‌ها بین زمین‌ها"}},
			"numberingScopes":    []gin.H{{"value": "tournament", "label": "شماره یکتا در کل تورنومنت"}, {"value": "day", "label": "شروع مجدد در هر روز"}, {"value": "court_day", "label": "شروع مجدد برای هر زمین در هر روز"}},
			"numberingOrders":    []gin.H{{"value": "rounds", "label": "هماهنگی مراحل؛ فینال‌ها در پایان"}, {"value": "category", "label": "تکمیل هر وزن به ترتیب فهرست اوزان"}},
			"dayAssignments":     []gin.H{{"value": "manual", "label": "انتخاب دستی روز هر وزن"}, {"value": "alternating", "label": "جایگاه‌های فرد روز اول، زوج روز دوم"}},
			"finalCourtPolicies": []gin.H{{"value": "primary", "label": "زمین اصلی همان وزن"}, {"value": "fixed", "label": "زمین منتخب برای فینال‌ها"}},
			"ageCategories":      tournaments.AgeCategories(),
		}, nil
	}))
	group := org.Group("/tournaments")
	group.POST("", tournamentBody(a, permTournamentsManage, func(r *Request, in tournaments.CreateTournamentInput) (any, error) {
		return r.Service().Create(r.Context(), in)
	}))
	group.GET("", a.route(true, true, func(r *Request) (any, error) {
		if err := r.official(); err != nil {
			return nil, err
		}
		limit, _ := strconv.Atoi(r.C.Query("limit"))
		offset, _ := strconv.Atoi(r.C.Query("offset"))
		items, err := r.Service().List(r.Context(), tournaments.TournamentListFilter{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		if len(r.Scope.TournamentIDs) == 0 {
			return items, nil
		}
		filtered := []tournaments.Tournament{}
		for _, item := range items {
			if r.permitsTournament(item.ID) {
				filtered = append(filtered, item)
			}
		}
		return filtered, nil
	}))
	group.GET("/:tournament", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeTournament(permTournamentsRead, r.C.Param("tournament")); err != nil {
			return nil, err
		}
		return r.Service().GetByID(r.Context(), r.C.Param("tournament"))
	}))
	group.PUT("/:tournament", tournamentBody(a, permTournamentsManage, func(r *Request, in tournaments.UpdateTournamentInput) (any, error) {
		return r.Service().Update(r.Context(), r.C.Param("tournament"), in)
	}))
	group.DELETE("/:tournament", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeTournament(permTournamentsManage, r.C.Param("tournament")); err != nil {
			return nil, err
		}
		return nil, r.Service().Delete(r.Context(), r.C.Param("tournament"))
	}))
	group.GET("/:tournament/settings", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeTournament(permTournamentsRead, r.C.Param("tournament")); err != nil {
			return nil, err
		}
		t, err := r.Service().GetByID(r.Context(), r.C.Param("tournament"))
		if err != nil {
			return nil, err
		}
		return t.Settings, nil
	}))
	group.PUT("/:tournament/settings", tournamentBody(a, permTournamentsManage, func(r *Request, in tournaments.TournamentSettings) (any, error) {
		return r.Service().UpdateSettings(r.Context(), r.C.Param("tournament"), in)
	}))
	group.POST("/:tournament/athletes", tournamentBody(a, permAthletesManage, func(r *Request, in tournaments.AddAthleteInput) (any, error) {
		return r.Service().AddAthlete(r.Context(), r.C.Param("tournament"), in)
	}))
	group.PUT("/:tournament/athletes/:entry", tournamentBody(a, permAthletesManage, func(r *Request, in tournaments.UpdateAthleteInput) (any, error) {
		return r.Service().UpdateAthlete(r.Context(), r.C.Param("tournament"), r.C.Param("entry"), in)
	}))
	group.DELETE("/:tournament/athletes/:entry", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeTournament(permAthletesManage, r.C.Param("tournament")); err != nil {
			return nil, err
		}
		return nil, r.Service().RemoveAthlete(r.Context(), r.C.Param("tournament"), r.C.Param("entry"))
	}))
	group.POST("/:tournament/athletes/:entry/weigh-in/preview", tournamentBody(a, permWeighInManage, func(r *Request, in tournaments.PreviewWeighInInput) (any, error) {
		return r.Service().PreviewWeighIn(r.Context(), r.C.Param("tournament"), r.C.Param("entry"), in)
	}))
	group.POST("/:tournament/athletes/:entry/weigh-in", tournamentBody(a, permWeighInManage, func(r *Request, in tournaments.RecordWeighInInput) (any, error) {
		return r.Service().RecordWeighIn(r.Context(), r.C.Param("tournament"), r.C.Param("entry"), in)
	}))
	group.POST("/:tournament/athletes/:entry/weigh-in/signature", tournamentBody(a, permWeighInManage, func(r *Request, in tournaments.SignWeighInInput) (any, error) {
		return r.Service().SignWeighIn(r.Context(), r.C.Param("tournament"), r.C.Param("entry"), in)
	}))
	group.DELETE("/:tournament/athletes/:entry/weigh-in", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeTournament(permWeighInManage, r.C.Param("tournament")); err != nil {
			return nil, err
		}
		return r.Service().ResetWeighIn(r.Context(), r.C.Param("tournament"), r.C.Param("entry"))
	}))
	group.DELETE("/:tournament/athletes/:entry/weigh-in/signature", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeTournament(permWeighInManage, r.C.Param("tournament")); err != nil {
			return nil, err
		}
		return r.Service().ClearWeighInSignature(r.Context(), r.C.Param("tournament"), r.C.Param("entry"))
	}))
	type drawInput struct {
		Type           string `json:"type"`
		WeightCategory string `json:"weightCategory"`
	}
	group.POST("/:tournament/draw", tournamentBody(a, permDrawManage, func(r *Request, in drawInput) (any, error) {
		if in.Type == "" {
			t, err := r.Service().GetByID(r.Context(), r.C.Param("tournament"))
			if err != nil {
				return nil, err
			}
			in.Type = t.Settings.Draw.Type
		}
		if in.WeightCategory != "" {
			return r.Service().DrawBracketForCategory(r.Context(), r.C.Param("tournament"), in.WeightCategory, in.Type)
		}
		return r.Service().DrawBracket(r.Context(), r.C.Param("tournament"), in.Type)
	}))
	group.POST("/:tournament/reset-category", tournamentBody(a, permDrawManage, func(r *Request, in struct {
		WeightCategory string `json:"weightCategory"`
	}) (any, error) {
		return r.Service().ResetCategoryBracketAndWeighIns(r.Context(), r.C.Param("tournament"), in.WeightCategory)
	}))
	group.GET("/:tournament/draw-readiness", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeTournament(permTournamentsRead, r.C.Param("tournament")); err != nil {
			return nil, err
		}
		t, err := r.Service().GetByID(r.Context(), r.C.Param("tournament"))
		if err != nil {
			return nil, err
		}
		return tournaments.CheckDrawReadiness(t), nil
	}))
	group.POST("/:tournament/numbering/preview", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeTournament(permTournamentsManage, r.C.Param("tournament")); err != nil {
			return nil, err
		}
		return r.Service().NumberMatches(r.Context(), r.C.Param("tournament"), true)
	}))
	group.POST("/:tournament/numbering", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeTournament(permTournamentsManage, r.C.Param("tournament")); err != nil {
			return nil, err
		}
		return r.Service().NumberMatches(r.Context(), r.C.Param("tournament"), false)
	}))
	group.POST("/:tournament/rebalance", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeTournament(permTournamentsManage, r.C.Param("tournament")); err != nil {
			return nil, err
		}
		return r.Service().RebalanceCourts(r.Context(), r.C.Param("tournament"))
	}))
	group.POST("/:tournament/matches/:match/start", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeMatch(permMatchesOperate, r.C.Param("tournament"), r.C.Param("match")); err != nil {
			return nil, err
		}
		return r.Service().StartMatch(r.Context(), r.C.Param("tournament"), r.C.Param("match"))
	}))
	group.POST("/:tournament/matches/:match/result", tournamentBody(a, permMatchesOperate, func(r *Request, in tournaments.MatchResultInput) (any, error) {
		tournamentID := r.C.Param("tournament")
		match, err := r.Service().RecordMatchResult(r.Context(), tournamentID, r.C.Param("match"), in)
		if err != nil {
			return nil, err
		}
		if err = refreshLeagueStandings(r, tournamentID); err != nil {
			return nil, err
		}
		return match, nil
	}))
	group.DELETE("/:tournament/matches/:match/result", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeMatch(permMatchesOverride, r.C.Param("tournament"), r.C.Param("match")); err != nil {
			return nil, err
		}
		tournamentID := r.C.Param("tournament")
		match, err := r.Service().ClearMatchResult(r.Context(), tournamentID, r.C.Param("match"))
		if err != nil {
			return nil, err
		}
		if err = refreshLeagueStandings(r, tournamentID); err != nil {
			return nil, err
		}
		return match, nil
	}))
	group.PUT("/:tournament/matches/:match/number", tournamentBody(a, permTournamentsManage, func(r *Request, in tournaments.SetMatchNumberInput) (any, error) {
		return r.Service().SetMatchNumber(r.Context(), r.C.Param("tournament"), r.C.Param("match"), in)
	}))
	group.POST("/:tournament/matches/:match/move", tournamentBody(a, permTournamentsManage, func(r *Request, in tournaments.MoveMatchInput) (any, error) {
		return r.Service().MoveMatch(r.Context(), r.C.Param("tournament"), r.C.Param("match"), in)
	}))
	group.POST("/:tournament/matches/:match/reassign-court", tournamentBody(a, permTournamentsManage, func(r *Request, in tournaments.ReassignMatchCourtInput) (any, error) {
		return r.Service().ReassignMatchCourt(r.Context(), r.C.Param("tournament"), r.C.Param("match"), in)
	}))
	group.POST("/:tournament/matches/:match/swap-position", tournamentBody(a, permTournamentsManage, func(r *Request, in tournaments.SwapMatchPositionsInput) (any, error) {
		return r.Service().SwapMatchPositions(r.Context(), r.C.Param("tournament"), r.C.Param("match"), in)
	}))
	group.POST("/:tournament/rankings/reset", tournamentBody(a, permTournamentsManage, func(r *Request, in struct {
		WeightCategory string `json:"weightCategory"`
	}) (any, error) {
		return r.Service().ResetRankings(r.Context(), r.C.Param("tournament"), in.WeightCategory)
	}))
	group.GET("/:tournament/access", a.route(true, true, func(r *Request) (any, error) {
		if err := r.authorizeTournament(permTournamentsRead, r.C.Param("tournament")); err != nil {
			return nil, err
		}
		return gin.H{"role": r.Role, "permissions": r.Permissions, "scope": r.Scope}, nil
	}))
	group.GET("/:tournament/sheets/ta", a.route(true, true, func(r *Request) (any, error) {
		id := r.C.Param("tournament")
		if err := r.authorizeTournament(permSheetsTA, id); err != nil {
			return nil, err
		}
		if err := r.authorizeStation("ta"); err != nil {
			return nil, err
		}
		courtText := r.C.Query("court")
		if courtText == "" {
			return nil, bad("court is required")
		}
		court := 0
		if n, e := strconv.Atoi(courtText); e == nil {
			court = n
		} else if len([]rune(courtText)) == 1 {
			court = int([]rune(strings.ToUpper(courtText))[0]-'A') + 1
		}
		if court < 1 || !r.permitsCourt(court) {
			return nil, forbidden()
		}
		t, err := r.Service().GetByID(r.Context(), id)
		if err != nil {
			return nil, err
		}
		matches := []tournaments.Match{}
		for _, m := range t.Matches {
			if m.Court == court {
				matches = append(matches, m)
			}
		}
		return gin.H{"tournamentId": id, "court": courtLabel(court), "matches": matches}, nil
	}))
}
