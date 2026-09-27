<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import draggable from 'vuedraggable'
import { useTournamentStore } from '../stores/tournament'
import type { Match } from '../types'

const store = useTournamentStore()
const t = computed(() => store.currentTournament)
const matches = computed(() => t.value?.matches ?? [])

const activeTab = ref<'categories' | 'schedule' | 'bracket'>('categories')
const sortMode = ref<'weight' | 'count'>('weight')

// ─── ثابت‌ها ───────────────────────────────────────────────────────────────

const COURT_LABELS = ['A', 'B', 'C', 'D', 'E', 'F']
const courtLabel = (n: number) => COURT_LABELS[n - 1] ?? String(n)

const ROUND_NAMES: Record<number, string> = {
  1: 'فینال',
  2: 'نیمه‌نهایی',
  4: 'یک‌چهارم نهایی',
  8: 'یک‌هشتم نهایی',
  16: 'یک‌شانزدهم نهایی',
  32: 'یک‌سی‌ودوم نهایی',
}

// ─── توابع کمکی ────────────────────────────────────────────────────────────

function roundLabel(match: Match): string {
  if (!t.value) return `دور ${match.round}`
  const catRounds = t.value.matches
      .filter(m => m.weightCategory === match.weightCategory)
      .map(m => m.round)
  if (!catRounds.length) return `دور ${match.round}`
  const maxRound = Math.max(...catRounds)
  const matchesInRound = 2 ** (maxRound - match.round)
  return ROUND_NAMES[matchesInRound] ?? `دور ${match.round}`
}

function weightSortValue(cat: string) {
  const m = cat.match(/(\d+(\.\d+)?)/)
  const num = m ? parseFloat(m[1]) : 0
  return cat.includes('+') ? num + 0.5 : num
}

function findFeederMatch(matchId: string, slot: 1 | 2): Match | null {
  if (!t.value) return null
  return (
      t.value.matches.find(
          m =>
              m.nextMatchId === matchId &&
              m.nextSlot !== undefined &&
              m.nextSlot === (slot === 1 ? 'athlete1' : 'athlete2')
      ) ?? null
  )
}

function athleteDisplayName(id: string | null, matchId: string, slot: 1 | 2): string {
  if (!id) {
    const feeder = findFeederMatch(matchId, slot)
    if (feeder) {
      if (feeder.court === 0) return `برندهٔ ${roundLabel(feeder)}`
      return `برندهٔ ${roundLabel(feeder)} (${courtLabel(feeder.court)})`
    }
    return 'در انتظار حریف'
  }
  return t.value?.athletes.find(a => a.id === id)?.name ?? '؟'
}

function matchLabel(match: Match): string {
  return `${courtLabel(match.court)}${match.order}`
}

// ─── دسته‌بندی‌ها ─────────────────────────────────────────────────────────

const categories = computed(() => {
  if (!t.value) return []
  const assignment = t.value.courtAssignment ?? {}
  const rows = Object.entries(
      t.value.athletes.reduce<Record<string, number>>((acc, a) => {
        acc[a.weightCategory] = (acc[a.weightCategory] ?? 0) + 1
        return acc
      }, {})
  ).map(([cat, athleteCount]) => {
    const catMatches = matches.value.filter(m => m.weightCategory === cat && m.court !== 0)
    const done = catMatches.filter(m => m.winnerId).length
    const courts = assignment[cat] ?? []
    const courtBreakdown = courts.map((c: number) => ({
      court: c,
      count: catMatches.filter(m => m.court === c).length,
    }))
    return { cat, athleteCount, total: catMatches.length, done, courts, courtBreakdown }
  })
  return sortMode.value === 'weight'
      ? rows.sort((a, b) => weightSortValue(a.cat) - weightSortValue(b.cat))
      : rows.sort((a, b) => b.athleteCount - a.athleteCount)
})

const courtLoads = computed(() => {
  if (!t.value) return []
  return Array.from({ length: t.value.courts }, (_, i) => {
    const c = i + 1
    const total = matches.value.filter(m => m.court === c).length
    const done = matches.value.filter(m => m.court === c && m.winnerId).length
    return { court: c, total, done }
  })
})

// ─── جابجایی دسته‌بندی (swap) ─────────────────────────────────────────────

const swapA = ref<string | null>(null)
const swapWarning = ref(false)

function startSwap(cat: string) {
  if (swapA.value === cat) { swapA.value = null; return }
  if (!swapA.value) { swapA.value = cat; return }
  const result = store.swapCourtCategories(t.value!.id, swapA.value, cat)
  swapWarning.value = result.warning
  swapA.value = null
  if (result.warning) setTimeout(() => (swapWarning.value = false), 4000)
}

function onReassign(cat: string, e: Event) {
  const court = Number((e.target as HTMLSelectElement).value)
  if (!t.value || !court) return
  store.reassignCourt(t.value.id, cat, court)
}

// ─── زمان‌بندی زمین‌ها ────────────────────────────────────────────────────

const activeCourts = computed(() => {
  if (!t.value) return []
  return Array.from({ length: t.value.courts }, (_, i) => i + 1)
})

const matchesByCourt = computed(() => {
  const result: Record<number, Match[]> = {}
  if (!t.value) return result
  for (const c of activeCourts.value) {
    result[c] = t.value.matches
        .filter(m => m.court === c && !m.isBye)
        .sort((a, b) => a.order - b.order)
  }
  return result
})

const localColumns = ref<Record<number, Match[]>>({})

watch(
    matchesByCourt,
    (val) => {
      const next: Record<number, Match[]> = {}
      for (const court of activeCourts.value) {
        next[court] = [...(val[court] ?? [])]
      }
      localColumns.value = next
    },
    { immediate: true, deep: true }
)

function onDragEnd(evt: any) {
  const matchId = evt.item?.getAttribute('data-id')
  const toCourt = Number(evt.to?.getAttribute('data-court'))
  const newIndex = evt.newIndex
  if (!matchId || Number.isNaN(toCourt) || !t.value) return
  store.moveMatch(t.value.id, matchId, toCourt, newIndex + 1)
}

// ─── جابجایی بازی‌ها (click-to-swap) ─────────────────────────────────────

const selectedMatchId = ref<string | null>(null)
const swapMatchResult = ref('')

function onMatchClick(match: Match) {
  if (match.winnerId) return
  if (!selectedMatchId.value) {
    selectedMatchId.value = match.id
    return
  }
  if (selectedMatchId.value === match.id) {
    selectedMatchId.value = null
    return
  }
  store.swapMatchPositions(t.value!.id, selectedMatchId.value, match.id)
  swapMatchResult.value = 'جابجایی انجام شد'
  selectedMatchId.value = null
  setTimeout(() => (swapMatchResult.value = ''), 2000)
}

// ─── جابجایی متنی بازی‌ها (text swap) ──────────────────────────────────────

const textSwapCodeA = ref('')
const textSwapCodeB = ref('')
const textSwapMessage = ref('')

function findMatchByCode(code: string): Match | null {
  const upper = code.trim().toUpperCase()
  // فرمت: حرف زمین + شماره ترتیب (مثل A3 یا B12)
  const match = upper.match(/^([A-F])(\d+)$/)
  if (!match) return null
  const courtIdx = COURT_LABELS.indexOf(match[1])
  if (courtIdx === -1) return null
  const court = courtIdx + 1
  const order = parseInt(match[2])
  const col = localColumns.value[court] ?? []
  return col.find((m: Match) => m.order === order) ?? null
}

const previewA = computed(() => findMatchByCode(textSwapCodeA.value))
const previewB = computed(() => findMatchByCode(textSwapCodeB.value))

function doTextSwap() {
  textSwapMessage.value = ''
  const mA = findMatchByCode(textSwapCodeA.value)
  const mB = findMatchByCode(textSwapCodeB.value)
  if (!mA || !mB) {
    textSwapMessage.value = 'کد وارد شده معتبر نیست'
    return
  }
  if (mA.id === mB.id) {
    textSwapMessage.value = 'دو بازی یکسان انتخاب شده'
    return
  }
  // جابجایی در localColumns
  let colKeyA = -1, colKeyB = -1, idxA = -1, idxB = -1
  for (const colKey of Object.keys(localColumns.value)) {
    const key = Number(colKey)
    const col = localColumns.value[key] as Match[]
    const i = col.findIndex(m => m.id === mA.id)
    const j = col.findIndex(m => m.id === mB.id)
    if (i !== -1) { colKeyA = key; idxA = i }
    if (j !== -1) { colKeyB = key; idxB = j }
  }
  if (colKeyA !== -1 && colKeyB !== -1) {
    const tmp = localColumns.value[colKeyA][idxA]
    localColumns.value[colKeyA][idxA] = localColumns.value[colKeyB][idxB]
    localColumns.value[colKeyB][idxB] = tmp
    textSwapMessage.value = `بازی ${textSwapCodeA.value.toUpperCase()} و ${textSwapCodeB.value.toUpperCase()} جابجا شد`
  }
}
</script>

<template>
  <div v-if="!matches.length" class="flex items-center justify-center h-40 text-slate-400 text-sm">
    ابتدا قرعه‌کشی را انجام دهید.
  </div>

  <div v-else class="p-4 flex flex-col gap-4 max-w-5xl mx-auto">

    <!-- تب‌ها -->
    <div class="flex bg-slate-100 rounded-full p-0.5 w-fit self-center gap-0.5">
      <button
          class="text-xs rounded-full px-4 py-1.5 transition-colors"
          :class="activeTab === 'categories' ? 'bg-white shadow text-slate-800' : 'text-slate-500 hover:text-slate-700'"
          @click="activeTab = 'categories'"
      >دسته‌بندی‌ها</button>
      <button
          class="text-xs rounded-full px-4 py-1.5 transition-colors"
          :class="activeTab === 'schedule' ? 'bg-white shadow text-slate-800' : 'text-slate-500 hover:text-slate-700'"
          @click="activeTab = 'schedule'"
      >زمان‌بندی زمین‌ها</button>
    </div>

    <!-- ══════════════ تب دسته‌بندی‌ها ══════════════ -->
    <template v-if="activeTab === 'categories'">

      <!-- بار هر زمین -->
      <div
          class="grid gap-3"
          :style="`grid-template-columns: repeat(${courtLoads.length}, minmax(0,1fr))`"
      >
        <div
            v-for="cl in courtLoads"
            :key="cl.court"
            class="bg-slate-800 text-white rounded-xl p-3 text-center"
        >
          <div class="text-xs text-slate-400 mb-1">زمین {{ courtLabel(cl.court) }}</div>
          <div class="text-2xl font-bold">{{ cl.total }}</div>
          <div class="text-xs text-slate-400">بازی · {{ cl.done }} انجام شده</div>
          <div class="mt-2 h-1.5 bg-slate-600 rounded-full overflow-hidden">
            <div
                class="h-full bg-indigo-400 rounded-full transition-all"
                :style="`width:${cl.total ? (cl.done / cl.total * 100) : 0}%`"
            />
          </div>
        </div>
      </div>

      <!-- هشدار جابجایی دسته -->
      <Transition
          enter-active-class="transition-all duration-200"
          enter-from-class="opacity-0 -translate-y-1"
          leave-to-class="opacity-0"
      >
        <div
            v-if="swapWarning"
            class="bg-amber-50 border border-amber-300 text-amber-800 text-sm rounded-xl px-4 py-2.5"
        >
          ⚠️ جابه‌جایی انجام شد، اما تعداد شرکت‌کنندگان این دو دسته اختلاف زیادی دارند.
        </div>
      </Transition>

      <!-- نوار انتخاب swap دسته -->
      <Transition
          enter-active-class="transition-all duration-200"
          enter-from-class="opacity-0 translate-y-1"
          leave-to-class="opacity-0"
      >
        <div
            v-if="swapA"
            class="fixed bottom-5 left-1/2 -translate-x-1/2 bg-slate-800 text-white text-sm px-5 py-2.5 rounded-full shadow-xl z-50 flex items-center gap-3"
        >
          <span class="w-2 h-2 rounded-full bg-blue-400 animate-pulse inline-block"></span>
          دسته «{{ swapA }}» انتخاب شد — روی دسته دیگری کلیک کنید
          <button
              class="text-slate-400 hover:text-white text-xs border border-slate-600 rounded-full px-2 py-0.5"
              @click="swapA = null"
          >لغو</button>
        </div>
      </Transition>

      <!-- مرتب‌سازی -->
      <div class="flex items-center gap-2">
        <span class="text-xs text-slate-400">مرتب‌سازی:</span>
        <div class="flex bg-slate-100 rounded-full p-0.5">
          <button
              class="text-xs rounded-full px-3 py-1 transition-colors"
              :class="sortMode === 'weight' ? 'bg-white shadow text-slate-800' : 'text-slate-500'"
              @click="sortMode = 'weight'"
          >بر اساس وزن</button>
          <button
              class="text-xs rounded-full px-3 py-1 transition-colors"
              :class="sortMode === 'count' ? 'bg-white shadow text-slate-800' : 'text-slate-500'"
              @click="sortMode = 'count'"
          >بر اساس تعداد</button>
        </div>
      </div>

      <!-- جدول دسته‌ها -->
      <div class="bg-white rounded-2xl border border-slate-200 overflow-hidden shadow-sm">
        <table class="w-full text-sm">
          <thead>
          <tr class="bg-slate-50 border-b border-slate-200 text-slate-500 text-xs">
            <th class="text-right px-4 py-3 font-medium">دسته وزنی</th>
            <th class="text-center px-3 py-3 font-medium">ورزشکاران</th>
            <th class="text-center px-3 py-3 font-medium">بازی‌ها</th>
            <th class="text-center px-3 py-3 font-medium">زمین(ها)</th>
            <th class="text-center px-3 py-3 font-medium">تغییر زمین</th>
            <th class="text-center px-3 py-3 font-medium">جابه‌جایی</th>
          </tr>
          </thead>
          <tbody>
          <tr
              v-for="row in categories"
              :key="row.cat"
              :class="[
                'border-b border-slate-100 last:border-0 transition-colors',
                swapA === row.cat ? 'bg-blue-50' : 'hover:bg-slate-50',
              ]"
          >
            <td class="px-4 py-3 font-medium text-slate-700">{{ row.cat }}</td>
            <td class="px-3 py-3 text-center text-slate-600">{{ row.athleteCount }}</td>
            <td class="px-3 py-3 text-center">
              <span class="text-slate-600">{{ row.done }}/{{ row.total }}</span>
              <div class="mt-1 h-1 bg-slate-100 rounded-full w-16 mx-auto overflow-hidden">
                <div
                    class="h-full bg-green-400 rounded-full"
                    :style="`width:${row.total ? (row.done / row.total * 100) : 0}%`"
                />
              </div>
            </td>
            <td class="px-3 py-3 text-center">
              <div class="flex gap-1 justify-center flex-wrap">
                  <span
                      v-for="cb in row.courtBreakdown"
                      :key="cb.court"
                      class="inline-flex items-center gap-1 bg-indigo-50 text-indigo-600 text-xs rounded-full px-2 py-0.5"
                  >
                    {{ courtLabel(cb.court) }}<span class="text-indigo-400">({{ cb.count }})</span>
                  </span>
                <span v-if="!row.courts.length" class="text-slate-300 text-xs">—</span>
              </div>
            </td>
            <td class="px-3 py-3 text-center">
              <select
                  v-if="t && row.courts.length <= 1"
                  class="text-xs border border-slate-200 rounded-lg px-2 py-1 bg-white text-slate-600 focus:outline-none focus:border-indigo-400"
                  :value="row.courts[0] ?? ''"
                  @change="onReassign(row.cat, $event)"
              >
                <option v-for="n in t.courts" :key="n" :value="n">زمین {{ courtLabel(n) }}</option>
              </select>

              <span v-else class="text-slate-300 text-xs">چند زمین</span>
            </td>
            <td class="px-3 py-3 text-center">
              <button
                  :class="[
                    'text-xs rounded-lg px-3 py-1.5 border transition-all',
                    swapA === row.cat
                      ? 'bg-blue-500 border-blue-500 text-white'
                      : swapA
                        ? 'bg-blue-50 border-blue-300 text-blue-600 hover:bg-blue-100'
                        : 'border-slate-200 text-slate-500 hover:border-slate-300 hover:text-slate-700',
                  ]"
                  @click="startSwap(row.cat)"
              >
                {{ swapA === row.cat ? '✓ انتخاب شد' : swapA ? 'انتخاب این' : '⇄ جابه‌جایی' }}
              </button>
            </td>
          </tr>
          </tbody>
        </table>
      </div>
    </template>

    <!-- ══════════════ تب زمان‌بندی زمین‌ها ══════════════ -->
    <template v-else-if="activeTab === 'schedule'">

      <!-- راهنمای click-to-swap -->
      <div class="bg-white border border-slate-200 rounded-2xl px-4 py-3 shadow-sm flex items-center gap-3">
        <span class="text-slate-400 text-lg leading-none">⇄</span>
        <div>
          <div class="text-xs font-semibold text-slate-600">جابجایی بازی‌ها</div>
          <div class="text-[11px] text-slate-400 mt-0.5">
            روی یک بازی کلیک کنید تا انتخاب شود، سپس روی بازی دیگری کلیک کنید تا جابجا شوند.
          </div>
        </div>
        <Transition
            enter-active-class="transition-all duration-150"
            enter-from-class="opacity-0 scale-95"
            leave-to-class="opacity-0"
        >
          <span
              v-if="swapMatchResult"
              class="mr-auto text-xs text-green-700 bg-green-50 border border-green-200 rounded-full px-3 py-1"
          >✓ {{ swapMatchResult }}</span>
        </Transition>
      </div>

      <!-- جابجایی متنی بازی‌ها -->
      <div class="bg-white border border-slate-200 rounded-2xl px-4 py-3 shadow-sm">
        <div class="text-xs font-semibold text-slate-600 mb-2">جابجایی با کد (مثال: A3 ، B12)</div>
        <div class="flex gap-2 items-start">
          <div class="flex flex-col gap-1 flex-1">
            <input
                v-model="textSwapCodeA"
                placeholder="کد بازی اول"
                class="text-xs border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:border-indigo-400 uppercase"
            />
            <span v-if="previewA" class="text-[10px] text-indigo-600 px-1">
        ✓ {{ previewA.weightCategory }} — {{ athleteDisplayName(previewA.athlete1Id, previewA.id, 1) }} vs {{ athleteDisplayName(previewA.athlete2Id, previewA.id, 2) }}
      </span>
            <span v-else-if="textSwapCodeA" class="text-[10px] text-red-400 px-1">بازی یافت نشد</span>
          </div>
          <span class="text-slate-400 text-sm pt-1.5">⇄</span>
          <div class="flex flex-col gap-1 flex-1">
            <input
                v-model="textSwapCodeB"
                placeholder="کد بازی دوم"
                class="text-xs border border-slate-200 rounded-lg px-3 py-1.5 focus:outline-none focus:border-indigo-400 uppercase"
            />
            <span v-if="previewB" class="text-[10px] text-indigo-600 px-1">
        ✓ {{ previewB.weightCategory }} — {{ athleteDisplayName(previewB.athlete1Id, previewB.id, 1) }} vs {{ athleteDisplayName(previewB.athlete2Id, previewB.id, 2) }}
      </span>
            <span v-else-if="textSwapCodeB" class="text-[10px] text-red-400 px-1">بازی یافت نشد</span>
          </div>
          <button
              @click="doTextSwap"
              :disabled="!previewA || !previewB"
              class="text-xs bg-indigo-600 text-white rounded-lg px-3 py-1.5 disabled:opacity-40 disabled:cursor-not-allowed hover:bg-indigo-700 transition-colors mt-0.5"
          >اجرا</button>
        </div>
        <Transition enter-active-class="transition-all duration-150" enter-from-class="opacity-0" leave-to-class="opacity-0">
          <div v-if="textSwapMessage" class="mt-2 text-[11px] text-green-700 bg-green-50 border border-green-200 rounded-lg px-3 py-1.5">
            ✓ {{ textSwapMessage }}
          </div>
        </Transition>
      </div>

      <!-- ستون‌های زمین -->
      <div
          class="grid gap-3"
          :style="`grid-template-columns: repeat(${activeCourts.length}, minmax(0,1fr))`"
      >
        <div
            v-for="court in activeCourts"
            :key="court"
            class="flex flex-col rounded-2xl border border-slate-200 bg-slate-50/80 shadow-sm overflow-hidden"
        >
          <!-- هدر زمین -->
          <div class="flex items-center justify-between px-3 py-2 border-b border-slate-200 bg-white">
            <span class="text-xs font-semibold text-slate-600">زمین {{ courtLabel(court) }}</span>
            <span class="text-[11px] text-slate-500 bg-slate-100 rounded-full px-2 py-0.5">
              {{ localColumns[court]?.length ?? 0 }} بازی
            </span>
          </div>

          <!-- لیست draggable -->
          <draggable
              :list="localColumns[court]"
              group="matches"
              item-key="id"
              tag="div"
              :data-court="court"
              :animation="180"
              :force-fallback="true"
              filter=".is-locked"
              ghost-class="drag-ghost"
              chosen-class="drag-chosen"
              fallback-class="drag-fallback"
              class="flex flex-col gap-2 p-2.5 min-h-[460px] flex-1"
              @end="onDragEnd"
          >
            <template #item="{ element: match }">
              <div
                  :data-id="match.id"
                  :class="[
                  'rounded-xl border bg-white transition-all select-none h-[136px]',
                  match.winnerId
                    ? 'is-locked opacity-60 cursor-not-allowed border-slate-100'
                    : selectedMatchId === match.id
                      ? 'ring-2 ring-indigo-500 border-indigo-300 cursor-pointer shadow-md'
                      : selectedMatchId
                        ? 'cursor-pointer border-indigo-100 hover:ring-2 hover:ring-indigo-300 hover:shadow-md'
                        : 'cursor-pointer hover:shadow-md hover:border-indigo-200 border-slate-100',
                ]"
                  @click="onMatchClick(match)"
              >
                <div class="h-full flex flex-col justify-between p-2.5">
                  <!-- هدر کارت -->
                  <div>
                    <div class="flex items-center justify-between mb-1.5">
                      <span class="font-mono text-[10px] bg-slate-100 text-slate-500 rounded-full px-2 py-0.5">
                        {{ matchLabel(match) }}
                      </span>
                      <span class="text-[10px] text-indigo-600 bg-indigo-50 rounded-full px-2 py-0.5">
                        {{ roundLabel(match) }}
                      </span>
                    </div>
                    <div class="text-[10px] text-slate-400 truncate mb-1.5">{{ match.weightCategory }}</div>

                    <!-- طرفین -->
                    <div class="space-y-1">
                      <div
                          v-for="(slot, idx) in ([1, 2] as const)"
                          :key="slot"
                          class="flex items-center gap-1.5"
                      >
                        <span
                            class="w-5 h-5 rounded-full flex items-center justify-center text-[9px] font-bold shrink-0"
                            :class="match.winnerId === (idx === 0 ? match.athlete1Id : match.athlete2Id)
                            ? 'bg-green-100 text-green-700'
                            : 'bg-slate-100 text-slate-500'"
                        >{{ idx === 0 ? '۱' : '۲' }}</span>
                        <span class="text-slate-700 text-xs truncate">
                          {{ athleteDisplayName(idx === 0 ? match.athlete1Id : match.athlete2Id, match.id, slot) }}
                        </span>
                      </div>
                    </div>
                  </div>

                  <!-- فوتر کارت -->
                  <div class="flex items-center justify-between mt-1">
                    <span
                        v-if="match.winnerId"
                        class="text-[10px] bg-green-50 text-green-600 rounded-full px-2 py-0.5"
                    >✓ پایان یافته</span>
                    <span
                        v-else-if="match.sourceLabel"
                        class="text-[10px] bg-amber-50 text-amber-600 border border-amber-200 rounded-full px-2 py-0.5 truncate"
                    >↩ {{ match.sourceLabel }}</span>
                    <span
                        v-else-if="selectedMatchId === match.id"
                        class="text-[10px] text-indigo-500 font-medium"
                    >انتخاب شد</span>
                    <span v-else class="text-[10px] text-slate-300">—</span>
                  </div>
                </div>
              </div>
            </template>
          </draggable>
        </div>
      </div>
    </template>

  </div>

  <!-- نوار شناور انتخاب بازی برای جابجایی -->
  <Transition
      enter-active-class="transition-all duration-150"
      enter-from-class="opacity-0 translate-y-2"
      leave-to-class="opacity-0 translate-y-2"
  >
    <div
        v-if="selectedMatchId"
        class="fixed bottom-5 left-1/2 -translate-x-1/2 bg-slate-800 text-white text-sm px-5 py-2.5 rounded-full shadow-xl z-50 flex items-center gap-3"
    >
      <span class="w-2 h-2 rounded-full bg-indigo-400 animate-pulse inline-block"></span>
      یک بازی انتخاب شد — روی بازی دیگری کلیک کنید
      <button
          class="text-slate-400 hover:text-white text-xs border border-slate-600 rounded-full px-2 py-0.5 transition-colors"
          @click.stop="selectedMatchId = null"
      >لغو</button>
    </div>
  </Transition>
</template>

<style scoped>
.drag-ghost {
  opacity: 0.3;
  background: #eef2ff;
  border: 2px dashed #818cf8 !important;
}
.drag-fallback {
  transform: rotate(2deg);
  box-shadow: 0 14px 28px rgba(15, 23, 42, 0.2);
  cursor: grabbing;
}
.drag-chosen {
  cursor: grabbing;
}
</style>
