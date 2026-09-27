<template>
  <div class="w-28 rounded-lg border bg-white shadow-sm overflow-hidden text-xs"
       :class="[
         isFinal ? 'border-amber-400 border-2 shadow-md' : 'border-slate-200',
         byeMatch ? 'opacity-75' : ''
       ]">
    <div v-for="slot in [1, 2] as const" :key="slot"
         class="flex items-center justify-between px-1.5 py-1.5 transition-all gap-1"
         :class="slotClass(slot)"
         @click="onSlotClick(slot)">
      <div class="flex items-center gap-1 min-w-0">
        <span v-if="ranking(slot)" class="shrink-0 text-[9px] font-bold rounded px-0.5"
              :class="rankBadgeClass(ranking(slot))">{{ ranking(slot) }}</span>
        <div class="min-w-0 flex flex-col">
          <span class="truncate leading-tight text-[11px] font-medium">{{ name(athleteId(slot)) }}</span>
          <span v-if="club(slot)" class="truncate text-[9px] opacity-70 leading-tight">{{ club(slot) }}</span>
        </div>
      </div>

      <div class="flex items-center gap-0.5 shrink-0">
        <!-- بج استراحت به‌جای جام -->
        <span v-if="isByeWin(slot)"
              class="text-[8px] text-slate-500 border border-slate-300 rounded px-1 leading-[14px]">
          استراحت
        </span>

        <span v-if="summary?.tally && !byeMatch"
              class="tabular-nums text-[10px] font-extrabold leading-none px-1 rounded bg-white/60">
          {{ slot === 1 ? summary.roundWins[0] : summary.roundWins[1] }}
        </span>
        <span v-if="isRealWinner(slot)" class="text-emerald-700 text-[10px]">🏆</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Match } from '../types'
import { useTournamentStore } from '../stores/tournament'
import { matchSummary } from '../utils/matchSummary'

const props = defineProps<{
  match: Match
  swapMode?: boolean
  selectedSlot?: { matchId: string; slot: 1 | 2 } | null
}>()

const emit = defineEmits<{
  (e: 'select-slot', payload: { matchId: string; slot: 1 | 2 }): void
  (e: 'open-details', matchId: string): void
  (e: 'record', matchId: string): void
}>()

const store = useTournamentStore()
const athletes = computed(() => store.currentTournament?.athletes ?? [])
const isFinal = computed(() => props.match.side === 'final')
const summary = computed(() => matchSummary(props.match))

/* --- تشخیص استراحت (هم‌راستا با BracketPdfExport) --- */
const byeMatch = computed(() => {
  const m: any = props.match
  if (m.isBye === true) return true
  return m.round === 1 && ((m.athlete1Id == null) !== (m.athlete2Id == null))
})

function athleteId(slot: 1 | 2) { return slot === 1 ? props.match.athlete1Id : props.match.athlete2Id }
function athlete(id: string | null | undefined) { return id ? (athletes.value.find(x => x.id === id) ?? null) : null }
function name(id: string | null | undefined) { return athlete(id)?.name ?? '—' }
function club(slot: 1 | 2) { return athlete(athleteId(slot))?.club ?? '' }
function ranking(slot: 1 | 2) { return athlete(athleteId(slot))?.ranking ?? null }

function isWinner(slot: 1 | 2) { const id = athleteId(slot); return !!id && props.match.winnerId === id }
function isByeWin(slot: 1 | 2) { return isWinner(slot) && byeMatch.value }      // برنده‌ی بای
function isRealWinner(slot: 1 | 2) { return isWinner(slot) && !byeMatch.value } // برنده‌ی واقعی

const canSwap = computed(() => props.swapMode && props.match.round === 1 && !props.match.winnerId)
function isSelected(slot: 1 | 2) {
  return props.selectedSlot?.matchId === props.match.id && props.selectedSlot?.slot === slot
}

function slotClass(slot: 1 | 2) {
  const isEmpty = !athleteId(slot)
  const winner = isRealWinner(slot)
  const byeWin = isByeWin(slot)

  const blue  = 'bg-blue-100 text-blue-950 border-r-4 border-blue-500'
  const red   = 'bg-red-100 text-red-950 border-r-4 border-red-500'
  const empty = 'bg-slate-50 text-slate-400 border-r-4 border-slate-200'
  const bye   = 'bg-white text-slate-500 border-r-4 border-slate-300 border-dashed'

  const neutral = winner || byeWin

  return {
    'border-b border-white/70': slot === 1,
    [empty]: isEmpty && !neutral,
    [bye]:   byeWin,
    [blue]:  !isEmpty && slot === 1 && !neutral,
    [red]:   !isEmpty && slot === 2 && !neutral,
    'bg-emerald-100 text-emerald-950 border-r-4 border-emerald-500 font-bold': winner,
    'cursor-pointer hover:brightness-95': canSwap.value && !isEmpty,
    'cursor-default': !canSwap.value,
    'ring-2 ring-inset ring-indigo-500 relative z-10': isSelected(slot),
    'opacity-50': props.swapMode && (!canSwap.value || isEmpty),
  }
}

function onSlotClick(slot: 1 | 2) {
  if (!canSwap.value || !athleteId(slot)) return
  emit('select-slot', { matchId: props.match.id, slot })
}

function rankBadgeClass(rank: number | null) {
  if (rank === 1) return 'bg-yellow-400 text-white'
  if (rank === 2) return 'bg-gray-800 text-gray-200'
  if (rank === 3 || rank === 4) return 'bg-amber-700 text-white'
  return 'bg-white/70 text-slate-600'
}
</script>
