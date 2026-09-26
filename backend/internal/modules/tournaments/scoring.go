package tournaments

import "fmt"

func techniquePoints(s RoundScore) int {
	return s.Punch + 2*s.BodyKick + 3*s.HeadKick + 4*s.TurningBodyKick + 6*s.TurningHeadKick
}
func penaltyCount(s RoundScore) int { return s.GamJeom + s.GamJeomLate }
func roundTotals(r MatchRound) (int, int) {
	return techniquePoints(r.Blue) + r.Red.GamJeom + 2*r.Red.GamJeomLate, techniquePoints(r.Red) + r.Blue.GamJeom + 2*r.Blue.GamJeomLate
}

// The values here deliberately match the desktop rules, not a claim about current WT rules.
func resolveRound(r MatchRound) (Corner, RoundEndReason) {
	return resolveRoundWithRules(r, defaultScoring())
}
func resolveRoundWithRules(r MatchRound, rules ScoringSettings) (Corner, RoundEndReason) {
	if r.ManualWinner && r.Winner != nil {
		reason := RoundEndSUP
		if r.EndedBy != nil {
			reason = *r.EndedBy
		}
		return *r.Winner, reason
	}
	gb, gr := penaltyCount(r.Blue), penaltyCount(r.Red)
	if gb >= rules.GamJeomLimitPerRound && gr < rules.GamJeomLimitPerRound {
		return CornerRed, RoundEndPUN
	}
	if gr >= rules.GamJeomLimitPerRound && gb < rules.GamJeomLimitPerRound {
		return CornerBlue, RoundEndPUN
	}
	blue, red := roundTotals(r)
	winner := CornerBlue
	if red > blue {
		winner = CornerRed
	}
	if r.IsGoldenPoint {
		if blue == red {
			return "", ""
		}
		return winner, RoundEndGDP
	}
	gap := blue - red
	if gap < 0 {
		gap = -gap
	}
	if r.Number >= rules.PointGapFromRound && gap >= rules.PointGap {
		return winner, RoundEndPTG
	}
	if blue != red {
		return winner, RoundEndPTF
	}
	bs := []int{r.Blue.TurningHeadKick, r.Blue.TurningBodyKick, r.Blue.HeadKick, r.Blue.BodyKick, r.Blue.Punch, -gb}
	rs := []int{r.Red.TurningHeadKick, r.Red.TurningBodyKick, r.Red.HeadKick, r.Red.BodyKick, r.Red.Punch, -gr}
	for i := range bs {
		if bs[i] > rs[i] {
			return CornerBlue, RoundEndPTF
		}
		if rs[i] > bs[i] {
			return CornerRed, RoundEndPTF
		}
	}
	return "", RoundEndSUP
}
func nonScoreWin(w WinType) bool {
	switch w {
	case WinTypeWDR, WinTypeDSQ, WinTypeRSC, WinTypePUN, WinTypeRSCInj, WinTypeWO:
		return true
	}
	return false
}
func normalizeScoredResult(input *MatchResultInput, options ...ScoringSettings) error {
	rules := defaultScoring()
	if len(options) > 0 {
		rules = options[0]
	}
	if len(input.Rounds) > rules.MaxRounds {
		return fmt.Errorf("%w: too many rounds for the tournament rules", ErrInvalidTournamentInput)
	}
	rounds := append([]MatchRound(nil), input.Rounds...)
	wins := map[Corner]int{}
	var final Corner
	var reason RoundEndReason
	for i := range rounds {
		if final != "" {
			return fmt.Errorf("%w: rounds after a decided match are not allowed", ErrInvalidTournamentInput)
		}
		winner, end := resolveRoundWithRules(rounds[i], rules)
		reason = end
		if winner != "" {
			w := winner
			rounds[i].Winner = &w
			wins[w]++
		} else {
			rounds[i].Winner = nil
		}
		e := end
		rounds[i].EndedBy = &e
		if wins[winner] >= rules.RoundsToWin {
			final = winner
		}
	}
	input.Rounds = rounds
	if nonScoreWin(input.WinType) {
		return nil
	}
	if final == "" {
		return fmt.Errorf("%w: rounds do not decide a winner", ErrInvalidTournamentInput)
	}
	winnerID := input.BlueID
	if final == CornerRed {
		winnerID = input.RedID
	}
	if input.WinnerID != "" && input.WinnerID != winnerID {
		return fmt.Errorf("%w: winner contradicts round scores", ErrInvalidTournamentInput)
	}
	input.WinnerID = winnerID
	input.WinnerCorner = &final
	input.WinType = WinType(reason)
	return nil
}
