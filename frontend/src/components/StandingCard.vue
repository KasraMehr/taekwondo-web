<script setup lang="ts">
import { computed } from 'vue'
import type { Match, RoundScore, WinType } from '../types'
import { useTournamentStore } from '../stores/tournament'

const props = defineProps<{ match: Match }>()

function displayOrder(value: number | string | null | undefined) {
  const num = Number(value)
  return Number.isFinite(num) && num > 0 ? num : 'استراحت'
}

function displayCourt(value: number | string | null | undefined) {
  const num = Number(value)
  return Number.isFinite(num) && num > 0 ? num : 'استراحت'
}

const store = useTournamentStore()
const athletes = computed(() => store.currentTournament?.athletes ?? [])
const allMatches = computed(() => store.currentTournament?.matches ?? [])
const isFinal = computed(() => props.match.side === 'final')

function athleteId(s: number) {
  return s === 0 ? props.match.athlete1Id : props.match.athlete2Id
}

function athlete(s: number) {
  const id = athleteId(s)
  return id ? athletes.value.find(a => a.id === id) : null
}

function playerName(s: number) { return athlete(s)?.name ?? '—' }
function playerClub(s: number) { return athlete(s)?.club ?? '' }

function isWinner(s: number) {
  const id = athleteId(s)
  return !!id && props.match.winnerId === id
}

function isLoser(s: number) {
  return !!props.match.winnerId && !isWinner(s) && !!athleteId(s)
}

const nextInfo = computed(() => {
  if (!props.match.nextMatchId) return null
  return allMatches.value.find(m => m.id === props.match.nextMatchId) ?? null
})

function slotClass(s: number) {
  const empty = !athleteId(s)
  return {
    'text-slate-300': empty,
    'bg-green-100': isWinner(s),
    'bg-red-50 text-slate-400': isLoser(s),
  }
}

/** ارزش امتیازی هر تکنیک */
const POINTS: Record<keyof RoundScore, number> = {
  punch: 1, bodyKick: 2, headKick: 3,
  turningBodyKick: 4, turningHeadKick: 5,
  gamJeom: 0, gamJeomLate: 0,
}

const WIN_TYPE_SHORT: Record<WinType, string> = {
  PTF: 'PTF', PTG: 'PTG', RSC: 'RSC', SUP: 'SUP',
  GDP: 'GDP', WDR: 'WDR', DSQ: 'DSQ', PUN: 'PUN', RSC_INJ: 'RSC', WO: 'WO',
}

/** ورزشکار اسلات در کدام گوشه؟ */
function cornerOfSlot(slot: 0 | 1): 'blue' | 'red' {
  const r = props.match.result
  const id = athleteId(slot)
  if (r && id) {
    if (r.blueId === id) return 'blue'
    if (r.redId === id) return 'red'
  }
  return slot === 0 ? 'blue' : 'red'
}

function techPoints(s: RoundScore): number {
  return (Object.keys(POINTS) as (keyof RoundScore)[])
      .reduce((sum, k) => sum + Number(s?.[k] ?? 0) * POINTS[k], 0)
}

function sideTotal(own: RoundScore, opp: RoundScore): number {
  const oppGamJeom = Number(opp?.gamJeom ?? 0) + Number(opp?.gamJeomLate ?? 0)
  return techPoints(own) + oppGamJeom
}

/** محاسبه آمار بازی: نوع برد و تعداد راندهای برنده */
const matchSummary = computed(() => {
  const r = props.match.result
  if (!r || !props.match.winnerId) return null

  const rounds = (r.rounds ?? []).map(rd => {
    const p0Corner = cornerOfSlot(0)
    const blueTotal = sideTotal(rd.blue, rd.red)
    const redTotal = sideTotal(rd.red, rd.blue)
    const p0Score = p0Corner === 'blue' ? blueTotal : redTotal
    const p1Score = p0Corner === 'blue' ? redTotal : blueTotal

    let winner: 0 | 1 | null = null
    if (rd.winner === 'blue') winner = p0Corner === 'blue' ? 0 : 1
    else if (rd.winner === 'red') winner = p0Corner === 'blue' ? 1 : 0
    else if (p0Score > p1Score) winner = 0
    else if (p1Score > p0Score) winner = 1

    return { winner }
  })

  const w0 = rounds.filter(x => x.winner === 0).length
  const w1 = rounds.filter(x => x.winner === 1).length
  const tally = rounds.length ? `${w0}:${w1}` : ''
  const winType = WIN_TYPE_SHORT[r.winType] ?? String(r.winType ?? '')

  return { tally, winType }
})
</script>

<template>
  <div class="flex flex-col items-center gap-0.5">
    <div class="text-[8px] font-bold text-white bg-blue-600 rounded-md px-1.5 py-px self-start mr-1">
      بازی {{ displayOrder(match.order) }}
    </div>

    <div class="w-[120px] border border-slate-200 rounded-md overflow-hidden bg-white shadow-sm relative"
         :class="isFinal ? 'border-amber-400 shadow-amber-100 shadow-md' : ''">
      <div class="text-center text-[8px] font-semibold text-slate-500 bg-slate-50 border-b border-slate-200 py-0.5">
        زمین {{ displayCourt(match.court) }}
      </div>

      <div
          v-for="s in [0, 1]"
          :key="s"
          class="flex items-center gap-1 px-1.5 py-[5px] border-b border-slate-100 last:border-b-0 select-none"
          :class="slotClass(s)"
      >
        <div class="w-[3px] h-3.5 rounded-sm flex-shrink-0" :class="s === 0 ? 'bg-blue-600' : 'bg-red-600'" />
        <div class="flex-1 min-w-0 text-right">
          <div class="text-[10px] whitespace-nowrap overflow-hidden text-ellipsis">{{ playerName(s) }}</div>
          <div v-if="playerClub(s)" class="text-[8px] text-slate-400 whitespace-nowrap overflow-hidden text-ellipsis">{{ playerClub(s) }}</div>
        </div>
        <span v-if="isWinner(s)" class="text-[10px] font-extrabold text-green-600 flex-shrink-0">✓</span>
      </div>

      <!-- نوع برد و آمار راندها -->
      <div v-if="matchSummary" class="text-center text-[8px] font-semibold text-slate-600 bg-slate-50 border-t border-slate-200 py-0.5">
        {{ matchSummary.winType }} · {{ matchSummary.tally }}
      </div>
    </div>

    <div v-if="nextInfo && match.winnerId"
         class="text-[8px] text-blue-600 bg-blue-50 border border-blue-200 rounded px-1.5 py-px text-center w-[120px]">
      ← زمین {{ displayCourt(nextInfo.court) }} | بازی {{ displayOrder(nextInfo.order) }}
    </div>
  </div>
</template>
