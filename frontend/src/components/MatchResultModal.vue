<!-- MatchResultModal.vue -->
<template>
  <div
      v-if="match"
      class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-6"
      role="dialog"
      aria-modal="true"
      :aria-label="`ثبت نتیجه بازی وزن ${match.weightCategory}`"
  >
    <div class="absolute inset-0 bg-slate-900/50 backdrop-blur-[2px]" @click="requestClose" />

    <div
        ref="panel"
        tabindex="-1"
        class="relative w-full max-w-3xl max-h-[92vh] flex flex-col rounded-2xl bg-white shadow-2xl outline-none"
        @keydown.esc.stop="requestClose"
    >
      <!-- هدر -->
      <div class="flex items-center gap-3 px-5 py-3.5 border-b border-slate-200 bg-gradient-to-l from-slate-50 to-white rounded-t-2xl">
        <span class="bg-blue-600 text-white text-[12px] font-bold px-3 py-1 rounded-full">
          وزن {{ match.weightCategory }}
        </span>
        <span class="text-[12px] text-slate-500">{{ roundLabel }}</span>
        <span
            v-if="match.status === 'completed'"
            class="text-[11px] text-emerald-700 bg-emerald-50 px-2.5 py-1 rounded-full font-semibold"
        >ثبت‌شده</span>
        <span
            v-else-if="match.status === 'ongoing'"
            class="text-[11px] text-amber-700 bg-amber-50 px-2.5 py-1 rounded-full font-semibold"
        >در جریان</span>

        <button
            class="mr-auto w-8 h-8 rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition"
            aria-label="بستن"
            @click="requestClose"
        >✕</button>
      </div>

      <div class="flex-1 overflow-y-auto px-5 py-4">
        <!-- کرنرها -->
        <div class="flex items-stretch gap-2 mb-4">
          <div class="flex-1 rounded-xl border-2 border-blue-200 bg-blue-50/60 px-3 py-2.5 text-center">
            <div class="text-[10px] font-bold text-blue-600 mb-0.5">آبی</div>
            <div class="text-[13.5px] font-bold text-slate-800 truncate">{{ blueName }}</div>
            <div class="text-[10.5px] text-slate-400 truncate">{{ blueClub || '—' }}</div>
          </div>

          <button
              class="px-2.5 rounded-xl border border-slate-200 text-slate-400 hover:text-slate-700 hover:border-slate-300 transition text-lg"
              aria-label="جابه‌جایی کرنر آبی و قرمز"
              title="جابه‌جایی کرنرها"
              @click="swapCorners = !swapCorners"
          >⇄</button>

          <div class="flex-1 rounded-xl border-2 border-rose-200 bg-rose-50/60 px-3 py-2.5 text-center">
            <div class="text-[10px] font-bold text-rose-600 mb-0.5">قرمز</div>
            <div class="text-[13.5px] font-bold text-slate-800 truncate">{{ redName }}</div>
            <div class="text-[10.5px] text-slate-400 truncate">{{ redClub || '—' }}</div>
          </div>
        </div>

        <!-- حالت ثبت -->
        <div class="flex gap-2 mb-4">
          <button
              v-for="opt in modeOptions"
              :key="opt.value"
              class="flex-1 py-2 rounded-xl text-[12.5px] border transition"
              :class="mode === opt.value
                ? 'bg-slate-800 text-white border-slate-800'
                : 'bg-white text-slate-600 border-slate-200 hover:bg-slate-50'"
              @click="mode = opt.value"
          >{{ opt.label }}</button>
        </div>

        <!-- ── حالت امتیازی ── -->
        <template v-if="mode === 'score'">
          <div
              v-for="(r, ri) in rounds"
              :key="r.number"
              class="rounded-xl border border-slate-200 mb-3 overflow-hidden"
          >
            <div class="flex items-center gap-2 px-3 py-2 bg-slate-50 border-b border-slate-200">
              <span class="text-[12.5px] font-bold text-slate-700">
                {{ r.isGoldenPoint ? 'راند سوم' : `راند ${r.number}` }}
              </span>
              <span class="text-[12px] font-mono text-slate-500">
                {{ totals[ri].blue }} : {{ totals[ri].red }}
              </span>
              <span
                  v-if="outcomes[ri].winner"
                  class="text-[11px] px-2 py-0.5 rounded-full font-semibold"
                  :class="outcomes[ri].winner === 'blue'
                    ? 'bg-blue-100 text-blue-700' : 'bg-rose-100 text-rose-700'"
              >
                برنده {{ outcomes[ri].winner === 'blue' ? 'آبی' : 'قرمز' }}
                <template v-if="outcomes[ri].endedBy"> · {{ outcomes[ri].endedBy }}</template>
              </span>
              <span v-else class="text-[11px] px-2 py-0.5 rounded-full bg-slate-200 text-slate-600">
                تساوی کامل
              </span>

              <button
                  v-if="rounds.length > 2 && ri === rounds.length - 1"
                  class="mr-auto text-[11px] text-rose-500 hover:text-rose-700"
                  @click="removeLastRound"
              >حذف راند</button>
            </div>

            <table class="w-full text-[12px]">
              <thead>
              <tr class="text-slate-400 text-[10.5px]">
                <th class="w-24 py-1.5 font-medium text-center">آبی</th>
                <th class="py-1.5 font-medium">تکنیک</th>
                <th class="w-24 py-1.5 font-medium text-center">قرمز</th>
              </tr>
              </thead>
              <tbody>
              <tr v-for="t in TECHNIQUES" :key="t.key" class="border-t border-slate-100">
                <td class="py-1">
                  <Counter
                      :value="r.blue[t.key]"
                      tone="blue"
                      :label="`${t.label} آبی، راند ${r.number}`"
                      @change="v => (r.blue[t.key] = v)"
                  />
                </td>
                <td class="text-center text-slate-600">
                  {{ t.label }}
                  <span class="text-slate-400 text-[10.5px]">({{ t.pts }})</span>
                </td>
                <td class="py-1">
                  <Counter
                      :value="r.red[t.key]"
                      tone="red"
                      :label="`${t.label} قرمز، راند ${r.number}`"
                      @change="v => (r.red[t.key] = v)"
                  />
                </td>
              </tr>

              <tr v-for="p in PENALTIES" :key="p.key" class="border-t border-slate-100 bg-amber-50/40">
                <td class="py-1">
                  <Counter
                      :value="r.blue[p.key]"
                      tone="blue"
                      :label="`${p.label} آبی، راند ${r.number}`"
                      @change="v => (r.blue[p.key] = v)"
                  />
                </td>
                <td class="text-center text-amber-800">
                  {{ p.label }}
                  <span class="text-amber-600/70 text-[10.5px]">({{ p.pts }}+ حریف)</span>
                </td>
                <td class="py-1">
                  <Counter
                      :value="r.red[p.key]"
                      tone="red"
                      :label="`${p.label} قرمز، راند ${r.number}`"
                      @change="v => (r.red[p.key] = v)"
                  />
                </td>
              </tr>
              </tbody>
            </table>

            <!-- برنده دستی وقتی تساوی کامل است -->
            <div v-if="!autoOutcomes[ri].winner" class="flex items-center gap-2 px-3 py-2 border-t border-slate-200 bg-slate-50">
              <span class="text-[11.5px] text-slate-500">تصمیم برتری (SUP):</span>
              <button
                  class="text-[11.5px] px-2.5 py-1 rounded-lg border transition"
                  :class="r.winner === 'blue'
                    ? 'bg-blue-600 text-white border-blue-600'
                    : 'bg-white border-slate-200 text-slate-600 hover:bg-blue-50'"
                  @click="setManual(ri, 'blue')"
              >آبی</button>
              <button
                  class="text-[11.5px] px-2.5 py-1 rounded-lg border transition"
                  :class="r.winner === 'red'
                    ? 'bg-rose-600 text-white border-rose-600'
                    : 'bg-white border-slate-200 text-slate-600 hover:bg-rose-50'"
                  @click="setManual(ri, 'red')"
              >قرمز</button>
              <button
                  v-if="r.manualWinner"
                  class="mr-auto text-[11px] text-slate-400 hover:text-slate-600"
                  @click="setManual(ri, null)"
              >پاک کردن</button>
            </div>
          </div>

          <button
              v-if="canAddRound"
              class="w-full py-2 rounded-xl border border-dashed border-slate-300 text-[12.5px] text-slate-500 hover:bg-slate-50 hover:border-slate-400 transition"
              @click="addRound(matchOutcome.blueRoundsWon === matchOutcome.redRoundsWon && rounds.length >= 2)"
          >
            + افزودن {{ rounds.length >= 2 ? 'راند سوم' : 'راند' }}
          </button>

          <!-- خلاصه -->
          <div class="mt-4 rounded-xl bg-slate-50 border border-slate-200 px-3.5 py-2.5 flex items-center gap-3 flex-wrap">
            <span class="text-[12px] text-slate-500">راندهای برده:</span>
            <span class="text-[12.5px] font-bold text-blue-700">آبی {{ matchOutcome.blueRoundsWon }}</span>
            <span class="text-slate-300">/</span>
            <span class="text-[12.5px] font-bold text-rose-700">قرمز {{ matchOutcome.redRoundsWon }}</span>
            <span
                v-if="matchOutcome.decided"
                class="mr-auto text-[12.5px] font-bold text-emerald-700 bg-emerald-50 px-2.5 py-1 rounded-full"
            >
              برنده: {{ matchOutcome.winnerCorner === 'blue' ? blueName : redName }}
              ({{ matchOutcome.winType }})
            </span>
            <span v-else class="mr-auto text-[12px] text-amber-700">برنده هنوز مشخص نیست</span>
          </div>
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
        <div class="mt-4">
          <label class="block text-[11.5px] text-slate-500 mb-1" for="match-note">یادداشت داور (اختیاری)</label>
          <textarea
              id="match-note"
              v-model="note"
              rows="2"
              class="w-full rounded-xl border border-slate-200 px-3 py-2 text-[12.5px] focus:outline-none focus:ring-2 focus:ring-blue-200 focus:border-blue-300"
              placeholder="مثلاً: توقف پزشکی در ثانیه ۴۵ راند دوم"
          />
        </div>

        <p v-if="error" class="mt-3 text-[12.5px] text-rose-600 bg-rose-50 border border-rose-200 rounded-xl px-3 py-2" role="alert">
          {{ error }}
        </p>
      </div>

      <!-- فوتر -->
      <div class="flex items-center gap-2 px-5 py-3 border-t border-slate-200 bg-slate-50/70 rounded-b-2xl">
        <button
            v-if="match.status === 'completed'"
            class="text-[12.5px] text-rose-600 hover:text-rose-800 px-3 py-2 rounded-lg hover:bg-rose-50 transition"
            @click="onClear"
        >حذف نتیجه</button>

        <button
            v-if="match.status === 'pending'"
            class="text-[12.5px] text-amber-700 hover:text-amber-900 px-3 py-2 rounded-lg hover:bg-amber-50 transition"
            @click="onStart"
        >شروع بازی</button>

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

/* شمارنده‌ی کوچک، inline تا فایل اضافه نشود */
const Counter = defineComponent({
  props: {
    value: { type: Number, required: true },
    tone: { type: String, default: 'blue' },
    label: { type: String, default: '' },
  },
  emits: ['change'],
  setup(p, { emit }) {
    const btn = 'w-6 h-6 rounded-md text-[13px] leading-none font-bold transition select-none disabled:opacity-30'
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
        class: 'w-6 text-center font-mono text-[13px] text-slate-800',
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
}
function removeLastRound() {
  if (rounds.value.length > 2) rounds.value.pop()
}
function setManual(i: number, corner: 'blue' | 'red' | null) {
  const r = rounds.value[i]
  r.winner = corner
  r.manualWinner = !!corner
  r.endedBy = corner ? 'SUP' : null
}

function onStart() {
  store.startMatch(props.tournamentId, props.match.id)
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
