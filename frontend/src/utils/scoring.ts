import type { RoundScore, MatchRound, MatchResult, WinType } from '../types'

/** ارزش هر تکنیک بر اساس آخرین قوانین WT */
export const TECHNIQUE_POINTS = {
    punch: 1,
    bodyKick: 2,
    headKick: 3,
    turningBodyKick: 4,
    turningHeadKick: 6,
} as const

export const GAMJEOM_POINTS = { normal: 1, late: 2 } as const

export interface ScoringRules {
    /** اختلاف امتیاز برای پایان راند (PTG) */
    pointGap: number
    /** PTG از کدام راند به بعد اعمال شود */
    pointGapFromRound: number
    /** تعداد گام‌جوم در یک راند که باعث باخت راند می‌شود (PUN) */
    gamJeomLimitPerRound: number
    /** تعداد راند لازم برای برد مسابقه */
    roundsToWin: number
    maxRounds: number
}

export const DEFAULT_RULES: ScoringRules = {
    pointGap: 15,
    pointGapFromRound: 1,
    gamJeomLimitPerRound: 5,
    roundsToWin: 2,
    maxRounds: 3,
}

export const emptyScore = (): RoundScore => ({
    punch: 0, bodyKick: 0, headKick: 0,
    turningBodyKick: 0, turningHeadKick: 0,
    gamJeom: 0, gamJeomLate: 0,
})

export const emptyRound = (number: number, isGoldenPoint = false): MatchRound => ({
    number,
    isGoldenPoint,
    blue: emptyScore(),
    red: emptyScore(),
    winner: null,
    endedBy: null,
})

/** امتیاز حاصل از تکنیک‌های خودِ ورزشکار */
export function techniquePoints(s: RoundScore): number {
    return s.punch * TECHNIQUE_POINTS.punch
        + s.bodyKick * TECHNIQUE_POINTS.bodyKick
        + s.headKick * TECHNIQUE_POINTS.headKick
        + s.turningBodyKick * TECHNIQUE_POINTS.turningBodyKick
        + s.turningHeadKick * TECHNIQUE_POINTS.turningHeadKick
}

/** تعداد کل گام‌جوم (برای آستانه PUN) */
export function gamJeomCount(s: RoundScore): number {
    return s.gamJeom + s.gamJeomLate
}

/** امتیازی که خطاهای این ورزشکار به حریف می‌دهد */
export function penaltyPointsConceded(s: RoundScore): number {
    return s.gamJeom * GAMJEOM_POINTS.normal + s.gamJeomLate * GAMJEOM_POINTS.late
}

/** امتیاز نهایی هر کرنر در یک راند */
export function roundTotals(r: MatchRound): { blue: number; red: number } {
    return {
        blue: techniquePoints(r.blue) + penaltyPointsConceded(r.red),
        red: techniquePoints(r.red) + penaltyPointsConceded(r.blue),
    }
}

/** سلسله‌مراتب تساوی: از گران‌ترین تکنیک به ارزان‌ترین، سپس گام‌جوم کمتر */
function breakTie(r: MatchRound): 'blue' | 'red' | null {
    const order: (keyof typeof TECHNIQUE_POINTS)[] =
        ['turningHeadKick', 'turningBodyKick', 'headKick', 'bodyKick', 'punch']

    for (const k of order) {
        if (r.blue[k] !== r.red[k]) return r.blue[k] > r.red[k] ? 'blue' : 'red'
    }
    const gb = gamJeomCount(r.blue)
    const gr = gamJeomCount(r.red)
    if (gb !== gr) return gb < gr ? 'blue' : 'red'
    return null // تصمیم برتری (SUP) → دستی
}

export interface RoundOutcome {
    winner: 'blue' | 'red' | null
    endedBy: MatchRound['endedBy']
}

/** برنده یک راند بر اساس قوانین؛ برنده دستی اولویت دارد */
export function resolveRound(r: MatchRound, rules = DEFAULT_RULES): RoundOutcome {
    if (r.manualWinner && r.winner) return { winner: r.winner, endedBy: r.endedBy ?? 'SUP' }

    // PUN: عبور از سقف گام‌جوم
    const gb = gamJeomCount(r.blue)
    const gr = gamJeomCount(r.red)
    if (gb >= rules.gamJeomLimitPerRound && gr < rules.gamJeomLimitPerRound)
        return { winner: 'red', endedBy: 'PUN' }
    if (gr >= rules.gamJeomLimitPerRound && gb < rules.gamJeomLimitPerRound)
        return { winner: 'blue', endedBy: 'PUN' }

    const { blue, red } = roundTotals(r)

    if (r.isGoldenPoint) {
        if (blue === red) return { winner: null, endedBy: null }
        return { winner: blue > red ? 'blue' : 'red', endedBy: 'GDP' }
    }

    // PTG: اختلاف امتیاز
    if (r.number >= rules.pointGapFromRound && Math.abs(blue - red) >= rules.pointGap)
        return { winner: blue > red ? 'blue' : 'red', endedBy: 'PTG' }

    if (blue !== red) return { winner: blue > red ? 'blue' : 'red', endedBy: 'PTF' }

    const tie = breakTie(r)
    return { winner: tie, endedBy: tie ? 'PTF' : 'SUP' }
}

export interface MatchOutcome {
    winnerCorner: 'blue' | 'red' | null
    blueRoundsWon: number
    redRoundsWon: number
    /** آیا با شمارش راندها مسابقه تمام شده است */
    decided: boolean
    winType: WinType | null
}

/** نتیجه کل مسابقه از روی راندها (فقط برای winType های امتیازی) */
export function resolveMatch(rounds: MatchRound[], rules = DEFAULT_RULES): MatchOutcome {
    let blueRoundsWon = 0
    let redRoundsWon = 0
    let lastReason: MatchRound['endedBy'] = null

    for (const r of rounds.slice(0, rules.maxRounds)) {
        const { winner, endedBy } = resolveRound(r, rules)
        if (!winner) continue
        if (winner === 'blue') blueRoundsWon++
        else redRoundsWon++
        lastReason = endedBy

        if (blueRoundsWon >= rules.roundsToWin || redRoundsWon >= rules.roundsToWin) break
    }

    const decided = blueRoundsWon >= rules.roundsToWin || redRoundsWon >= rules.roundsToWin
    const winnerCorner = decided ? (blueRoundsWon > redRoundsWon ? 'blue' : 'red') : null

    const winType: WinType | null = !decided
        ? null
        : lastReason === 'PTG' ? 'PTG'
            : lastReason === 'PUN' ? 'PUN'
                : lastReason === 'GDP' ? 'GDP'
                    : lastReason === 'SUP' ? 'SUP'
                        : 'PTF'

    return { winnerCorner, blueRoundsWon, redRoundsWon, decided, winType }
}

/** برنده‌های راندها را بازنویسی می‌کند (برای ذخیره‌سازی) */
export function normalizeRounds(rounds: MatchRound[], rules = DEFAULT_RULES): MatchRound[] {
    return rounds.map((r) => {
        if (r.manualWinner) return r
        const { winner, endedBy } = resolveRound(r, rules)
        return { ...r, winner, endedBy }
    })
}

/** winType هایی که به راند و امتیاز وابسته نیستند */
export const NON_SCORE_WIN_TYPES: WinType[] = ['WDR', 'DSQ', 'RSC', 'PUN', 'RSC_INJ', 'WO']

export function isNonScoreWin(w: WinType): boolean {
    return NON_SCORE_WIN_TYPES.includes(w)
}

/** خلاصه متنی برای PDF و لیست نتایج */
export function resultSummary(res: MatchResult): string {
    if (isNonScoreWin(res.winType)) return res.winType
    const rounds = res.rounds
        .filter((r) => r.winner)
        .map((r) => {
            const { blue, red } = roundTotals(r)
            return `${blue}:${red}`
        })
    return `${res.winType} (${rounds.join(' , ')})`
}
