<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 p-4"
       @click.self="emit('close')">
    <div class="w-full max-w-md max-h-[85vh] overflow-y-auto rounded-2xl bg-white shadow-xl"
         role="dialog" aria-modal="true" :aria-label="`جزئیات نتیجه ${n1} مقابل ${n2}`">

      <!-- هدر -->
      <div class="px-4 py-3 border-b border-slate-200 flex items-center gap-2">
        <div class="min-w-0">
          <div class="text-[13px] font-bold text-slate-800 truncate">{{ n1 }} <span class="text-slate-400">/</span> {{ n2 }}</div>
          <div class="text-[11px] text-slate-400">وزن {{ match.weightCategory }} · زمین {{ match.court }}</div>
        </div>
        <button type="button" class="mr-auto text-slate-400 hover:text-slate-700 text-lg leading-none px-1"
                aria-label="بستن" @click="emit('close')">×</button>
      </div>

      <div v-if="summary" class="p-4 space-y-3">
        <!-- نوع برد -->
        <div class="flex items-center gap-2 flex-wrap">
          <span class="bg-slate-800 text-white text-[11px] font-bold px-2 py-1 rounded-lg">{{ summary.short }}</span>
          <span class="text-[12.5px] text-slate-700">{{ summary.label }}</span>
          <span v-if="summary.tally" class="mr-auto tabular-nums text-[12.5px] font-bold text-slate-800">
            راندها {{ summary.tally }}
          </span>
        </div>

        <!-- برنده -->
        <div v-if="winnerName" class="flex items-center gap-2 px-3 py-2 rounded-xl bg-emerald-50 border border-emerald-200">
          <span>🏆</span>
          <span class="text-[12.5px] font-bold text-emerald-900">{{ winnerName }}</span>
        </div>

        <!-- راندها -->
        <div v-for="r in summary.rounds" :key="r.index" class="rounded-xl border border-slate-200 overflow-hidden">
          <div class="flex items-center gap-2 px-3 py-1.5 bg-slate-50 border-b border-slate-200">
            <span class="text-[11.5px] font-bold text-slate-600">راند {{ r.index }}</span>
            <span class="mr-auto tabular-nums text-[13px] font-extrabold text-slate-800">{{ r.p1 }} : {{ r.p2 }}</span>
            <span v-if="r.winner" class="text-[10.5px] text-emerald-700">
              برندهٔ راند: {{ r.winner === 1 ? n1 : n2 }}
            </span>
          </div>
          <div class="grid grid-cols-2 divide-x divide-x-reverse divide-slate-100 text-[11px]">
            <div v-for="slot in [1, 2] as const" :key="slot" class="p-2 space-y-1">
              <div class="font-bold" :class="slot === 1 ? 'text-blue-700' : 'text-red-700'">
                {{ slot === 1 ? n1 : n2 }}
              </div>
              <div v-for="row in breakdown(r.index - 1, slot)" :key="row.key"
                   class="flex items-center justify-between text-slate-600">
                <span>{{ row.label }}</span>
                <span class="tabular-nums font-semibold">{{ row.value }}</span>
              </div>
              <div v-if="!breakdown(r.index - 1, slot).length" class="text-slate-300">—</div>
            </div>
          </div>
        </div>

        <!-- یادداشت داور -->
        <div v-if="summary.note" class="text-[11.5px] text-slate-600 bg-slate-50 border border-slate-200 rounded-xl px-3 py-2">
          <span class="font-bold text-slate-500">یادداشت داور: </span>{{ summary.note }}
        </div>
      </div>

      <div v-else class="p-6 text-center text-sm text-slate-400">نتیجه‌ای ثبت نشده است.</div>

      <!-- اکشن‌ها -->
      <div class="px-4 py-3 border-t border-slate-200 flex gap-2">
        <button type="button"
                class="px-3.5 py-1.5 rounded-xl bg-blue-600 text-white text-[12.5px] font-semibold hover:bg-blue-700"
                @click="emit('edit', match.id)">ویرایش نتیجه</button>
        <button type="button"
                class="px-3.5 py-1.5 rounded-xl border border-slate-200 text-slate-600 text-[12.5px] hover:bg-slate-50"
                @click="emit('close')">بستن</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import type { Match, MatchRound, RoundScore, WinType } from '../types'

const props = defineProps<{ match: Match; athleteMap: Map<string, any> }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'edit', matchId: string): void }>()

/** ارزش امتیازی هر تکنیک (اخطارها اینجا صفرند؛ جداگانه به حریف امتیاز می‌دهند) */
const POINTS: Record<keyof RoundScore, number> = {
  punch: 1,
  bodyKick: 2,
  headKick: 3,
  turningBodyKick: 4,
  turningHeadKick: 5,
  gamJeom: 0,
  gamJeomLate: 0,
}

/** برچسب دسته‌ها برای ریز امتیاز هر راند (شامل اخطارها) */
const CATEGORY_LABELS: Record<keyof RoundScore, string> = {
  punch: 'مشت',
  bodyKick: 'لگد بدن',
  headKick: 'لگد سر',
  turningBodyKick: 'چرخشی بدن',
  turningHeadKick: 'چرخشی سر',
  gamJeom: 'گام‌جوم',
  gamJeomLate: 'گام‌جوم دیرهنگام',
}

const WIN_TYPE_LABELS: Record<WinType, { short: string; label: string }> = {
  PTF:     { short: 'PTF', label: 'برتری امتیازی' },
  PTG:     { short: 'PTG', label: 'اختلاف ۱۲ امتیاز' },
  RSC:     { short: 'RSC', label: 'توقف مسابقه' },
  SUP:     { short: 'SUP', label: 'برتری فنی (تصمیم داوران)' },
  GDP:     { short: 'GDP', label: 'امتیاز طلایی' },
  WDR:     { short: 'WDR', label: 'انصراف' },
  DSQ:     { short: 'DSQ', label: 'اخراج' },
  PUN:     { short: 'PUN', label: 'اخراج (۵ گام‌جوم)' },
  RSC_INJ: { short: 'RSC', label: 'مصدومیت' },
  WO:      { short: 'WO', label: 'عدم حضور' },
}


const n1 = computed(() => props.athleteMap.get(props.match.athlete1Id ?? '')?.name ?? '—')
const n2 = computed(() => props.athleteMap.get(props.match.athlete2Id ?? '')?.name ?? '—')

/** ورزشکار اسلات ۱/۲ در کدام گوشه است؟ */
function cornerOfSlot(slot: 1 | 2): 'blue' | 'red' {
  const r = props.match.result
  const id = slot === 1 ? props.match.athlete1Id : props.match.athlete2Id
  if (r && id) {
    if (r.blueId === id) return 'blue'
    if (r.redId === id) return 'red'
  }
  // پیش‌فرض: اسلات ۱ آبی، اسلات ۲ قرمز
  return slot === 1 ? 'blue' : 'red'
}

function techPoints(s: RoundScore): number {
  return (Object.keys(POINTS) as (keyof RoundScore)[])
      .reduce((sum, k) => sum + Number(s?.[k] ?? 0) * POINTS[k], 0)
}

/** جمع امتیاز یک طرف = امتیاز تکنیک خودش + اخطارهای حریف */
function sideTotal(own: RoundScore, opp: RoundScore): number {
  const oppGamJeom = Number(opp?.gamJeom ?? 0) + Number(opp?.gamJeomLate ?? 0)
  return techPoints(own) + oppGamJeom
}

/** RoundScore طرفِ اسلات مشخص در یک راند */
function roundSide(raw: MatchRound | undefined, slot: 1 | 2): RoundScore | undefined {
  if (!raw) return undefined
  return cornerOfSlot(slot) === 'blue' ? raw.blue : raw.red
}

const summary = computed(() => {
  const r = props.match.result
  if (!r) return null

  const rounds = (r.rounds ?? []).map((rd, i) => {
    const p1Corner = cornerOfSlot(1)
    const blueTotal = sideTotal(rd.blue, rd.red)
    const redTotal = sideTotal(rd.red, rd.blue)
    const p1 = p1Corner === 'blue' ? blueTotal : redTotal
    const p2 = p1Corner === 'blue' ? redTotal : blueTotal

    let winner: 1 | 2 | null = null
    if (rd.winner === 'blue') winner = p1Corner === 'blue' ? 1 : 2
    else if (rd.winner === 'red') winner = p1Corner === 'blue' ? 2 : 1
    else if (p1 > p2) winner = 1
    else if (p2 > p1) winner = 2

    return { index: i + 1, p1, p2, winner }
  })

  const w1 = rounds.filter(x => x.winner === 1).length
  const w2 = rounds.filter(x => x.winner === 2).length
  const tally = rounds.length ? `${w1} : ${w2}` : ''

  let winnerSlot: 1 | 2 | null = null
  const wid = r.winnerId ?? props.match.winnerId
  if (wid && wid === props.match.athlete1Id) winnerSlot = 1
  else if (wid && wid === props.match.athlete2Id) winnerSlot = 2
  else if (r.winnerCorner) winnerSlot = r.winnerCorner === cornerOfSlot(1) ? 1 : 2

  const wt = WIN_TYPE_LABELS[r.winType] ?? { short: String(r.winType ?? ''), label: '' }

  return { short: wt.short, label: wt.label, tally, rounds, winnerSlot, note: r.note }
})

const winnerName = computed(() => {
  const s = summary.value?.winnerSlot
  return s === 1 ? n1.value : s === 2 ? n2.value : ''
})

/** ریز امتیازهای یک طرف در یک راند؛ فقط مقادیر غیرصفر (شامل اخطارها) */
function breakdown(roundIndex: number, slot: 1 | 2) {
  const raw = props.match.result?.rounds?.[roundIndex]
  const side = roundSide(raw, slot)
  return (Object.keys(CATEGORY_LABELS) as (keyof RoundScore)[])
      .map(key => ({ key, label: CATEGORY_LABELS[key], value: Number(side?.[key] ?? 0) }))
      .filter(row => row.value > 0)
}

function onKey(e: KeyboardEvent) { if (e.key === 'Escape') emit('close') }
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>
