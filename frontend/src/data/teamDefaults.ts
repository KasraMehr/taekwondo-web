import type { Gender } from './categories'
import type { TeamDrawOptions, TeamScoringConfig, TeamWeightSlot } from '../types'

export function defaultTeamScoring(
    blindLineup = false,
): TeamScoringConfig {
    return {
        pointsWin: 3,
        pointsDraw: 1,
        pointsLoss: 0,
        walkoverRounds: 2,
        blindLineup,
        tieBreakers: [
            'points', 'boutDiff', 'roundDiff', 'headToHead',
            'boutsWon', 'roundsWon', 'fewestDsq', 'draw',
        ],
    }
}

export function defaultTeamDraw(): TeamDrawOptions {
    return {
        minRest: 1,
        matsPerEncounter: 1,
        parallel: true,
        drawType: 'random',
        iterations: 3000,
    }
}

/** اسلات‌های خالی؛ عنوان واقعی اوزان در صفحه مسابقه تیمی ویرایش می‌شود */
export function buildDefaultWeightSlots(
    slotCount: 8 | 10,
    gender: Gender,
): TeamWeightSlot[] {
    return Array.from({ length: slotCount }, (_, i) => ({
        slotNo: i + 1,
        title: `وزن ${i + 1}`,
        gender,
    }))
}
