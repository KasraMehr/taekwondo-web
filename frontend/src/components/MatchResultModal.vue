<!-- MatchResultModal.vue -->
<template>
  <div
      v-if="match"
      class="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-3"
      role="dialog"
      aria-modal="true"
      :aria-label="`ثبت نتیجه بازی وزن ${match.weightCategory}`"
  >
    <div class="absolute inset-0 bg-slate-900/50 backdrop-blur-[2px]" @click="requestClose" />

    <div
        ref="panel"
        tabindex="-1"
        class="relative flex max-h-[98dvh] w-full max-w-5xl flex-col overflow-hidden rounded-2xl border border-slate-300 bg-white shadow-2xl outline-none"
        @keydown.esc.stop="requestClose"
    >
      <div class="flex items-center gap-3 border-b border-slate-200 bg-white px-3 py-2 sm:px-4">
        <div class="grid h-8 w-8 place-items-center rounded-lg border border-blue-200 bg-blue-50 text-lg text-blue-700">✦</div>
        <div class="min-w-0">
          <div class="text-sm font-black text-slate-900">برگه ثبت امتیازات</div>
          <div class="text-[11px] text-slate-500">{{ store.currentTournament?.name || 'مسابقات تکواندو' }}</div>
        </div>
        <span class="rounded-full bg-slate-100 px-2.5 py-1 text-[11px] font-bold text-slate-600">{{ roundLabel }}</span>
        <span
            v-if="match.status === 'completed'"
            class="rounded-full bg-emerald-50 px-2.5 py-1 text-[11px] font-semibold text-emerald-700"
        >ثبت‌شده</span>
        <span
            v-else-if="match.status === 'ongoing'"
            class="rounded-full bg-amber-50 px-2.5 py-1 text-[11px] font-semibold text-amber-700"
        >در جریان</span>

        <button
            class="mr-auto grid h-9 w-9 place-items-center rounded-lg text-slate-500 transition hover:bg-slate-100 hover:text-slate-900"
            aria-label="بستن"
            @click="requestClose"
        >✕</button>
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain p-2 sm:p-3">
        <div class="mx-auto max-w-[1050px] bg-white">
          <div class="mb-2 grid grid-cols-3 gap-px overflow-hidden rounded-lg border border-slate-200 bg-slate-200 text-[11px] sm:grid-cols-6">
            <div class="bg-slate-50 px-2 py-1.5"><span class="text-slate-500">تاریخ: </span><b>{{ matchDate }}</b></div>
            <div class="bg-slate-50 px-2 py-1.5"><span class="text-slate-500">روز: </span><b>{{ match.day ?? 1 }}</b></div>
            <div class="bg-slate-50 px-2 py-1.5"><span class="text-slate-500">زمین: </span><b>{{ match.court ?? '—' }}</b></div>
            <div class="bg-slate-50 px-2 py-1.5"><span class="text-slate-500">بازی: </span><b>{{ match.matchNumber ?? '—' }}</b></div>
            <div class="bg-slate-50 px-2 py-1.5"><span class="text-slate-500">وزن: </span><b>{{ match.weightCategory }}</b></div>
            <div class="bg-slate-50 px-2 py-1.5"><span class="text-slate-500">مرحله: </span><b>{{ roundLabel }}</b></div>
          </div>

          <div class="mb-2 grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] gap-2" dir="ltr">
            <div class="min-w-0 overflow-hidden border border-blue-300" dir="rtl">
              <div class="bg-blue-700 px-2 py-1 text-center text-xs font-black tracking-wide text-white">CHUNG · آبی</div>
              <div class="min-w-0 bg-blue-50/40 px-3 py-1.5 text-center">
                <div class="truncate text-sm font-black text-slate-900" :title="blueName">{{ blueName }}</div>
                <div class="truncate text-[11px] text-slate-500" :title="blueClub">باشگاه: {{ blueClub || '—' }}</div>
              </div>
            </div>
            <button class="self-center rounded-lg border border-slate-300 bg-white px-2 py-1 text-sm font-bold text-slate-600 transition hover:border-blue-400 hover:text-blue-700" aria-label="جابه‌جایی کرنر آبی و قرمز" title="جابه‌جایی کرنرها" @click="swapCorners = !swapCorners">⇄</button>
            <div class="min-w-0 overflow-hidden border border-rose-300" dir="rtl">
              <div class="bg-rose-700 px-2 py-1 text-center text-xs font-black tracking-wide text-white">HONG · قرمز</div>
              <div class="min-w-0 bg-rose-50/40 px-3 py-1.5 text-center">
                <div class="truncate text-sm font-black text-slate-900" :title="redName">{{ redName }}</div>
                <div class="truncate text-[11px] text-slate-500" :title="redClub">باشگاه: {{ redClub || '—' }}</div>
              </div>
            </div>
          </div>

        <div class="mb-2 flex gap-2 border-b border-slate-200 pb-2">
          <button
              v-for="opt in modeOptions"
              :key="opt.value"
              class="flex-1 rounded-lg border py-1.5 text-xs transition"
              :class="mode === opt.value
                ? 'bg-slate-800 text-white border-slate-800'
                : 'bg-white text-slate-600 border-slate-200 hover:bg-slate-50'"
              @click="mode = opt.value"
          >{{ opt.label }}</button>
        </div>

        <!-- جدول امتیاز راندها، با چینش مشابه برگهٔ رسمی -->
        <template v-if="mode === 'score'">
          <div class="mb-2 flex flex-wrap items-center gap-1.5">
            <button v-for="(r, ri) in rounds" :key="r.number" class="rounded-lg border px-3 py-1.5 text-xs font-bold transition" :class="activeRoundIndex === ri ? 'border-slate-800 bg-slate-800 text-white' : 'border-slate-200 bg-white text-slate-600 hover:bg-slate-50'" :aria-pressed="activeRoundIndex === ri" @click="activeRoundIndex = ri">راند {{ r.number }} <span class="mr-1 font-mono">{{ totals[ri].blue }}–{{ totals[ri].red }}</span></button>
            <button v-if="canAddRound" class="rounded-lg border border-dashed border-blue-300 px-2.5 py-1.5 text-xs font-bold text-blue-700 hover:bg-blue-50" @click="addRound(matchOutcome.blueRoundsWon === matchOutcome.redRoundsWon && rounds.length >= 2)">+ راند</button>
          </div>
          <section v-for="(r, ri) in rounds" v-show="activeRoundIndex === ri" :key="r.number" class="mb-2 overflow-hidden rounded-lg border border-slate-300 bg-white">
            <div class="flex flex-wrap items-center gap-2 border-b border-slate-300 bg-slate-100 px-2 py-1">
              <span class="rounded-md bg-slate-900 px-2.5 py-1 text-xs font-black text-white">R{{ r.number }}</span>
              <b class="text-sm text-slate-800">{{ r.isGoldenPoint ? 'راند سوم / امتیاز طلایی' : `راند ${r.number}` }}</b>
              <span class="mr-auto font-mono text-sm font-black tabular-nums text-slate-700">{{ totals[ri].blue }} : {{ totals[ri].red }}</span>
              <button v-if="rounds.length > 2 && ri === rounds.length - 1" class="text-xs font-bold text-rose-600 hover:text-rose-800" @click="removeLastRound">حذف راند</button>
            </div>
            <div class="grid md:grid-cols-[minmax(0,1fr)_7rem_minmax(0,1fr)]" dir="ltr">
              <div class="min-w-0 border-b border-blue-200 lg:border-b-0 lg:border-r" dir="rtl">
                <div class="flex items-center justify-between bg-blue-50 px-3 py-2 text-xs font-black text-blue-800"><span>CHUNG · امتیاز آبی</span><span class="rounded-md bg-blue-700 px-2 py-0.5 font-mono text-base text-white">{{ totals[ri].blue }}</span></div>
                <div class="px-3 py-2"><div class="mb-1 text-[11px] font-black text-slate-500">Deuk-Jeom · امتیاز تکنیک‌ها</div><div class="grid gap-x-3 gap-y-1 sm:grid-cols-2 lg:grid-cols-2"><div v-for="tech in TECHNIQUES" :key="tech.key" class="flex min-h-10 items-center justify-between gap-2 border-b border-slate-100 text-xs"><span class="min-w-0 text-slate-700">{{ tech.label }} <small class="text-slate-400">({{ tech.pts }})</small></span><Counter :value="r.blue[tech.key]" tone="blue" :label="`${tech.label} آبی، راند ${r.number}`" @change="v => (r.blue[tech.key] = v)" /></div></div></div>
                <div class="border-t border-blue-100 bg-amber-50/40 px-3 py-2"><div class="mb-1 text-[11px] font-black text-amber-800">Gam-Jeom · اخطارها</div><div class="grid gap-x-3 sm:grid-cols-2 lg:grid-cols-2"><div v-for="penalty in PENALTIES" :key="penalty.key" class="flex min-h-10 items-center justify-between gap-2 text-xs"><span class="min-w-0 text-slate-700">{{ penalty.label }}</span><Counter :value="r.blue[penalty.key]" tone="blue" :label="`${penalty.label} آبی، راند ${r.number}`" @change="v => (r.blue[penalty.key] = v)" /></div></div></div>
              </div>
              <div class="flex min-h-28 flex-col items-center justify-center gap-2 border-y border-slate-200 bg-slate-50 px-2 py-4 text-center lg:border-x lg:border-y-0" dir="rtl">
                <span class="text-[11px] font-black text-slate-500">برندهٔ راند {{ r.number }}</span>
                <span v-if="outcomes[ri].winner" class="rounded-full px-3 py-1 text-xs font-black" :class="outcomes[ri].winner === 'blue' ? 'bg-blue-100 text-blue-800' : 'bg-rose-100 text-rose-800'">{{ outcomes[ri].winner === 'blue' ? 'آبی' : 'قرمز' }}</span>
                <span v-else class="rounded-full bg-amber-100 px-3 py-1 text-xs font-bold text-amber-800">نیاز به تصمیم برتری</span>
                <span v-if="outcomes[ri].endedBy" class="text-[10px] font-bold text-slate-500">{{ outcomes[ri].endedBy }}</span>
              </div>
              <div class="min-w-0 border-t border-rose-200 lg:border-t-0 lg:border-l" dir="rtl">
                <div class="flex items-center justify-between bg-rose-50 px-3 py-2 text-xs font-black text-rose-800"><span>HONG · امتیاز قرمز</span><span class="rounded-md bg-rose-700 px-2 py-0.5 font-mono text-base text-white">{{ totals[ri].red }}</span></div>
                <div class="px-3 py-2"><div class="mb-1 text-[11px] font-black text-slate-500">Deuk-Jeom · امتیاز تکنیک‌ها</div><div class="grid gap-x-3 gap-y-1 sm:grid-cols-2 lg:grid-cols-2"><div v-for="tech in TECHNIQUES" :key="tech.key" class="flex min-h-10 items-center justify-between gap-2 border-b border-slate-100 text-xs"><span class="min-w-0 text-slate-700">{{ tech.label }} <small class="text-slate-400">({{ tech.pts }})</small></span><Counter :value="r.red[tech.key]" tone="red" :label="`${tech.label} قرمز، راند ${r.number}`" @change="v => (r.red[tech.key] = v)" /></div></div></div>
                <div class="border-t border-rose-100 bg-amber-50/40 px-3 py-2"><div class="mb-1 text-[11px] font-black text-amber-800">Gam-Jeom · اخطارها</div><div class="grid gap-x-3 sm:grid-cols-2 lg:grid-cols-2"><div v-for="penalty in PENALTIES" :key="penalty.key" class="flex min-h-10 items-center justify-between gap-2 text-xs"><span class="min-w-0 text-slate-700">{{ penalty.label }}</span><Counter :value="r.red[penalty.key]" tone="red" :label="`${penalty.label} قرمز، راند ${r.number}`" @change="v => (r.red[penalty.key] = v)" /></div></div></div>
              </div>
            </div>
            <div v-if="!autoOutcomes[ri].winner" class="flex flex-wrap items-center gap-2 border-t border-slate-200 bg-amber-50 px-3 py-2"><span class="text-xs font-bold text-amber-900">تصمیم برتری راند:</span><button class="rounded-lg border px-3 py-1.5 text-xs font-bold" :class="r.winner === 'blue' ? 'border-blue-600 bg-blue-600 text-white' : 'border-blue-200 bg-white text-blue-700'" @click="setManual(ri, 'blue')">آبی</button><button class="rounded-lg border px-3 py-1.5 text-xs font-bold" :class="r.winner === 'red' ? 'border-rose-600 bg-rose-600 text-white' : 'border-rose-200 bg-white text-rose-700'" @click="setManual(ri, 'red')">قرمز</button><button v-if="r.manualWinner" class="mr-auto text-xs text-slate-500 hover:text-slate-700" @click="setManual(ri, null)">پاک کردن</button></div>
          </section>
          <div class="grid gap-px overflow-hidden rounded-lg border border-slate-300 bg-slate-300 text-center text-xs sm:grid-cols-3"><div class="bg-blue-50 p-2 font-bold text-blue-800">راندهای آبی <b class="mr-1 text-base">{{ matchOutcome.blueRoundsWon }}</b></div><div class="bg-white p-2 font-black text-slate-800">{{ matchOutcome.decided ? `برنده: ${matchOutcome.winnerCorner === 'blue' ? blueName : redName}` : 'برنده هنوز مشخص نیست' }}<span v-if="matchOutcome.winType" class="mr-1 text-slate-500">({{ matchOutcome.winType }})</span></div><div class="bg-rose-50 p-2 font-bold text-rose-800">راندهای قرمز <b class="mr-1 text-base">{{ matchOutcome.redRoundsWon }}</b></div></div>
        </template>
        <!-- ── حالت غیرامتیازی ── -->
        <template v-else>
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 mb-4">
            <button
                v-for="w in NON_SCORE_OPTIONS"
                :key="w.value"
                class="px-3 py-2.5 rounded-xl border text-right transition"
                :class="manualWinType === w.value
                  ? 'bg-slate-800 text-white border-slate-800'
                  : 'bg-white border-slate-200 hover:bg-slate-50'"
                @click="manualWinType = w.value"
            >
              <div class="text-[12.5px] font-bold">{{ w.value }}</div>
              <div class="text-[10.5px]" :class="manualWinType === w.value ? 'text-slate-300' : 'text-slate-400'">
                {{ w.label }}
              </div>
            </button>
          </div>

          <div class="text-[12px] text-slate-500 mb-2">برنده را انتخاب کنید:</div>
          <div class="flex gap-2">
            <button
                class="flex-1 py-2.5 rounded-xl border-2 text-[13px] font-bold transition"
                :class="manualWinnerId === blueId
                  ? 'bg-blue-600 text-white border-blue-600'
                  : 'bg-white text-slate-700 border-blue-200 hover:bg-blue-50'"
                @click="manualWinnerId = blueId"
            >{{ blueName }}</button>
            <button
                class="flex-1 py-2.5 rounded-xl border-2 text-[13px] font-bold transition"
                :class="manualWinnerId === redId
                  ? 'bg-rose-600 text-white border-rose-600'
                  : 'bg-white text-slate-700 border-rose-200 hover:bg-rose-50'"
                @click="manualWinnerId = redId"
            >{{ redName }}</button>
          </div>
        </template>

        <!-- یادداشت -->
        <div class="mt-2">
          <label class="sr-only" for="match-note">یادداشت داور (اختیاری)</label>
          <input
              id="match-note"
              v-model="note"
              type="text"
              class="w-full rounded-lg border border-slate-200 px-3 py-1.5 text-xs focus:border-blue-300 focus:outline-none focus:ring-2 focus:ring-blue-200"
              placeholder="یادداشت داور (اختیاری)"
          />
        </div>

        <p v-if="error" class="mt-3 text-[12.5px] text-rose-600 bg-rose-50 border border-rose-200 rounded-xl px-3 py-2" role="alert">
          {{ error }}
        </p>
        </div>
      </div>

      <!-- فوتر -->
      <div class="flex items-center gap-2 rounded-b-2xl border-t border-slate-200 bg-slate-50/70 px-4 py-2">
        <button
            v-if="match.status === 'completed'"
            class="text-[12.5px] text-rose-600 hover:text-rose-800 px-3 py-2 rounded-lg hover:bg-rose-50 transition"
            @click="onClear"
        >حذف نتیجه</button>

        <div class="mr-auto flex gap-2">
          <button
              class="px-4 py-2 rounded-xl border border-slate-200 bg-white text-[12.5px] text-slate-600 hover:bg-slate-100 transition"
              @click="requestClose"
          >انصراف</button>
          <button
              class="px-5 py-2 rounded-xl bg-blue-600 text-white text-[12.5px] font-bold hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed transition"
              :disabled="!canSave || saving"
              @click="onSave"
          >{{ saving ? 'در حال ثبت…' : 'ذخیره نتیجه' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch, h, defineComponent } from 'vue'
import type { PropType } from 'vue'
import { useTournamentStore } from '../stores/tournament'
import {
  emptyRound, roundTotals, resolveRound, resolveMatch, DEFAULT_RULES,
} from '../utils/scoring'
import type { MatchRound, WinType } from '../types'

const props = defineProps({
  match: { type: Object as PropType<any | null>, default: null },
  tournamentId: { type: String, required: true },
  athleteMap: { type: Object as PropType<Map<string, any>>, required: true },
})
const emit = defineEmits<{ (e: 'close'): void; (e: 'saved', id: string): void }>()

const store = useTournamentStore()
const matchDate = computed(() => {
  const value = store.currentTournament?.date
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('fa-IR-u-ca-persian', {
    year: 'numeric', month: '2-digit', day: '2-digit', timeZone: 'Asia/Tehran',
  }).format(date)
})

/* شمارنده‌ی کوچک، inline تا فایل اضافه نشود */
const Counter = defineComponent({
  props: {
    value: { type: Number, required: true },
    tone: { type: String, default: 'blue' },
    label: { type: String, default: '' },
  },
  emits: ['change'],
  setup(p, { emit }) {
    const btn = 'h-8 w-8 rounded-md text-base leading-none font-bold transition select-none disabled:opacity-30'
    const tint = () => (p.tone === 'red'
        ? 'bg-rose-100 text-rose-700 hover:bg-rose-200'
        : 'bg-blue-100 text-blue-700 hover:bg-blue-200')
    return () => h('div', { class: 'flex items-center justify-center gap-1.5' }, [
      h('button', {
        class: [btn, tint()],
        'aria-label': `کاهش ${p.label}`,
        disabled: p.value <= 0,
        onClick: () => emit('change', Math.max(0, p.value - 1)),
      }, '−'),
      h('span', {
        class: 'w-6 text-center font-mono text-sm font-bold tabular-nums text-slate-800',
        'aria-label': p.label,
      }, String(p.value)),
      h('button', {
        class: [btn, tint()],
        'aria-label': `افزایش ${p.label}`,
        onClick: () => emit('change', p.value + 1),
      }, '+'),
    ])
  },
})

const TECHNIQUES = [
  { key: 'punch', label: 'مشت', pts: 1 },
  { key: 'bodyKick', label: 'هوگو', pts: 2 },
  { key: 'headKick', label: 'سر', pts: 3 },
  { key: 'turningBodyKick', label: 'چرخشی هوگو', pts: 4 },
  { key: 'turningHeadKick', label: 'چرخشی سر', pts: 6 },
] as const

const PENALTIES = [
  { key: 'gamJeom', label: 'گام‌جوم', pts: 1 },
  { key: 'gamJeomLate', label: 'گام‌جوم ۱۰ ثانیه آخر', pts: 2 },
] as const

const NON_SCORE_OPTIONS: { value: WinType; label: string }[] = [
  { value: 'WDR', label: 'انصراف' },
  { value: 'RSC', label: 'توقف داور / پزشک' },
  { value: 'DSQ', label: 'اخراج' },
  { value: 'SUP', label: 'برتری فنی (تصمیم داوران)' },
  { value: 'PUN', label: 'حد گام‌جوم' },
]

const modeOptions = [
  { value: 'score' as const, label: 'ورود امتیاز راندها' },
  { value: 'manual' as const, label: 'انصراف / اخراج / پزشکی' },
]

const panel = ref<HTMLElement | null>(null)
const mode = ref<'score' | 'manual'>('score')
const rounds = ref<MatchRound[]>([])
const activeRoundIndex = ref(0)
const swapCorners = ref(false)
const manualWinType = ref<WinType>('WDR')
const manualWinnerId = ref<string | null>(null)
const note = ref('')
const error = ref('')
const dirty = ref(false)

const blueId = computed(() =>
    swapCorners.value ? props.match?.athlete2Id : props.match?.athlete1Id)
const redId = computed(() =>
    swapCorners.value ? props.match?.athlete1Id : props.match?.athlete2Id)

const nameOf = (id: string | null) => {
  if (!id) return '—'
  const a = props.athleteMap.get(id)
  return a?.name ?? a?.fullName ?? '—'
}
const clubOf = (id: string | null) => (id ? props.athleteMap.get(id)?.club ?? '' : '')

const blueName = computed(() => nameOf(blueId.value))
const redName = computed(() => nameOf(redId.value))
const blueClub = computed(() => clubOf(blueId.value))
const redClub = computed(() => clubOf(redId.value))

const ROUND_LABELS = ['فینال', 'نیمه‌نهایی', 'یک‌چهارم نهایی', 'یک‌هشتم نهایی', 'یک‌شانزدهم نهایی']
const roundLabel = computed(() => {
  const m = props.match
  if (!m) return ''
  const cat = (store.currentTournament?.matches ?? []).filter(x => x.weightCategory === m.weightCategory)
  const last = Math.max(...cat.map(x => x.round))
  return ROUND_LABELS[last - m.round] ?? `دور ${m.round}`
})

/* محاسبات زنده */
const totals = computed(() => rounds.value.map(r => roundTotals(r)))
const autoOutcomes = computed(() =>
    rounds.value.map(r => resolveRound({ ...r, manualWinner: false, winner: null }, DEFAULT_RULES)))
const outcomes = computed(() => rounds.value.map(r => resolveRound(r, DEFAULT_RULES)))
const matchOutcome = computed(() => resolveMatch(rounds.value, DEFAULT_RULES))

const canAddRound = computed(() =>
    rounds.value.length < DEFAULT_RULES.maxRounds && !matchOutcome.value.decided)

const canSave = computed(() => {
  if (!props.match?.athlete1Id || !props.match?.athlete2Id) return false
  return mode.value === 'manual' ? !!manualWinnerId.value : matchOutcome.value.decided
})

/* مقدار اولیه */
function hydrate() {
  const m = props.match
  error.value = ''
  dirty.value = false
  activeRoundIndex.value = 0
  if (!m) return

  const res = m.result
  if (res) {
    swapCorners.value = res.blueId === m.athlete2Id
    note.value = res.note ?? ''
    mode.value = ['WDR', 'DSQ', 'RSC', 'DQB', 'PUN'].includes(res.winType) && !res.rounds?.length
        ? 'manual' : 'score'
    manualWinType.value = res.winType
    manualWinnerId.value = res.winnerId ?? null
    rounds.value = res.rounds?.length
        ? JSON.parse(JSON.stringify(res.rounds))
        : [emptyRound(1), emptyRound(2)]
  } else {
    swapCorners.value = false
    note.value = ''
    mode.value = 'score'
    manualWinType.value = 'WDR'
    manualWinnerId.value = null
    rounds.value = [emptyRound(1), emptyRound(2)]
  }
  nextTick(() => panel.value?.focus())
}

watch(() => props.match?.id, hydrate, { immediate: true })
watch([rounds, mode, manualWinnerId, manualWinType, note, swapCorners],
    () => { dirty.value = true; error.value = '' }, { deep: true })

/* اکشن‌ها */
function addRound(golden = false) {
  rounds.value.push(emptyRound(rounds.value.length + 1, golden))
  activeRoundIndex.value = rounds.value.length - 1
}
function removeLastRound() {
  if (rounds.value.length > 2) {
    rounds.value.pop()
    activeRoundIndex.value = rounds.value.length - 1
  }
}
function setManual(i: number, corner: 'blue' | 'red' | null) {
  const r = rounds.value[i]
  r.winner = corner
  r.manualWinner = !!corner
  r.endedBy = corner ? 'SUP' : null
}

const saving = ref(false)
async function onSave() {
  if (saving.value) return
  saving.value = true
  const res = await store.submitMatchResult(props.tournamentId, props.match.id, {
    winType: mode.value === 'manual' ? manualWinType.value : 'PTF',
    rounds: mode.value === 'manual' ? [] : rounds.value,
    winnerId: mode.value === 'manual' ? manualWinnerId.value : undefined,
    note: note.value.trim() || undefined,
    swapCorners: swapCorners.value,
  })
  saving.value = false
  if (res.error) { error.value = res.error; return }
  emit('saved', props.match.id)
  emit('close')
}

function onClear() {
  if (!confirm('نتیجه این بازی و نتایج مراحل بعدی وابسته به آن پاک می‌شود. مطمئنی؟')) return
  const res = store.clearMatchResult(props.tournamentId, props.match.id)
  if (res.error) { error.value = res.error; return }
  emit('close')
}

function requestClose() {
  if (dirty.value && !confirm('تغییرات ذخیره نشده است. بستن؟')) return
  emit('close')
}
</script>
