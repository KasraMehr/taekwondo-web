import type { Match } from '../types'
import { resolveRound } from './scoring'

export type RoundView = { index: number; p1: number; p2: number; winner: 0 | 1 | 2 }

export type MatchSummary = {
    short: string          // PTF, WDR, ...
    label: string          // متن فارسی نوع برد
    tally: string | null   // «2-0»
    rounds: RoundView[]
    roundWins: [number, number]
    winnerSlot: 0 | 1 | 2
    note: string
}

const WIN_TYPES: Record<string, [string, string]> = {
    PTF: ['PTF', 'برد با امتیاز نهایی'],
    PTG: ['PTG', 'برد با اختلاف امتیاز'],
    GDP: ['GDP', 'برد با امتیاز طلایی'],
    SUP: ['SUP', 'برد با برتری فنی'],
    PUN: ['PUN', 'برد با گام‌جوم حریف'],
    RSC: ['RSC', 'توقف مسابقه توسط داور'],
    WDR: ['WDR', 'انصراف حریف'],
    DSQ: ['DSQ', 'اخراج حریف'],
    DQB: ['DQB', 'اخراج به‌دلیل رفتار'],
}

export const CATEGORY_LABELS: Record<string, string> = {
    punch: 'مشت',
    bodyKick: 'ضربه بدن',
    headKick: 'ضربه سر',
    turnBodyKick: 'چرخشی بدن',
    turnHeadKick: 'چرخشی سر',
    gamjeom: 'گام‌جوم',
    gamjeomLast10: 'گام‌جوم ۱۰ ثانیه آخر',
}

function num(v: unknown) { return typeof v === 'number' && isFinite(v) ? v : 0 }

function normalizeSlot(v: unknown): 0 | 1 | 2 {
    if (v === 1 || v === '1' || v === 'a1' || v === 'athlete1') return 1
    if (v === 2 || v === '2' || v === 'a2' || v === 'athlete2') return 2
    return 0
}

/** آبجکت امتیازهای یک طرف در یک راند (نام کلیدها در types.ts تطبیق داده می‌شود) */
export function roundSide(round: any, slot: 1 | 2): Record<string, number> {
    const keys = slot === 1 ? ['a1', 'athlete1', 'blue'] : ['a2', 'athlete2', 'red']
    for (const k of keys) if (round && round[k] && typeof round[k] === 'object') return round[k]
    return {}
}

/** یک راند را با قواعد scoring.ts به نمای قابل نمایش تبدیل می‌کند */
export function roundView(round: any, index: number): RoundView {
    let p1 = 0, p2 = 0, winner: 0 | 1 | 2 = 0
    try {
        const r: any = resolveRound(round)
        p1 = num(r?.p1 ?? r?.points1 ?? r?.score1 ?? r?.total1)
        p2 = num(r?.p2 ?? r?.points2 ?? r?.score2 ?? r?.total2)
        winner = normalizeSlot(r?.winner ?? r?.winnerSlot)
    } catch {
        // اگر ساختار راند ناقص بود، فقط بدون امتیاز نمایش داده می‌شود
    }
    if (!winner) winner = p1 > p2 ? 1 : p2 > p1 ? 2 : 0
    return { index: index + 1, p1, p2, winner }
}

/** خلاصهٔ نتیجه؛ اگر نتیجه ثبت نشده باشد null */
export function matchSummary(match: Match | any): MatchSummary | null {
    const result: any = match?.result
    if (!result) return null

    const rounds: RoundView[] = Array.isArray(result.rounds)
        ? result.rounds.map((r: any, i: number) => roundView(r, i))
        : []

    const roundWins: [number, number] = [0, 0]
    for (const r of rounds) {
        if (r.winner === 1) roundWins[0]++
        else if (r.winner === 2) roundWins[1]++
    }

    const type = String(result.winType ?? '').toUpperCase()
    const [short, label] = WIN_TYPES[type] ?? [type || '—', 'نتیجهٔ ثبت‌شده']

    const winnerId = result.winnerId ?? match.winnerId ?? null
    const winnerSlot: 0 | 1 | 2 = winnerId
        ? (winnerId === match.athlete1Id ? 1 : winnerId === match.athlete2Id ? 2 : 0)
        : 0

    return {
        short,
        label,
        tally: rounds.length ? `${roundWins[0]}-${roundWins[1]}` : null,
        rounds,
        roundWins,
        winnerSlot,
        note: String(result.note ?? result.refereeNote ?? ''),
    }
}
