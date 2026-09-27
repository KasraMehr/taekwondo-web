<!-- StandingsPage.vue -->
<template>
  <div ref="rootEl">
    <!-- فیلتر وزن‌ها -->
    <div class="flex items-center justify-center gap-2 mb-6 flex-wrap print:hidden">
      <button
          @click="selectedCategory = null"
          class="px-3.5 py-1.5 rounded-full text-sm border transition-all"
          :class="selectedCategory === null
          ? 'bg-slate-800 text-white border-slate-800 shadow-sm'
          : 'bg-white text-slate-600 border-slate-200 hover:border-slate-300 hover:bg-slate-50'"
      >همه وزن‌ها</button>

      <button
          v-for="b in brackets"
          :key="b.cat"
          @click="selectedCategory = b.cat"
          class="px-3.5 py-1.5 rounded-full text-sm border transition-all"
          :class="selectedCategory === b.cat
          ? 'bg-blue-600 text-white border-blue-600 shadow-sm'
          : 'bg-white text-slate-600 border-slate-200 hover:border-blue-300 hover:bg-blue-50/50'"
      >{{ b.cat }}</button>
    </div>

    <div v-if="!hasMatches" class="text-center text-slate-400 py-16">
      ابتدا قرعه‌کشی انجام دهید.
    </div>

    <div
        v-for="b in visibleBrackets"
        :key="b.cat"
        class="border border-slate-200/80 rounded-2xl overflow-hidden bg-white shadow-sm mb-6 print-cat"
        :style="{ '--print-zoom': String(b.printZoom) }"
    >
      <!-- هدر -->
      <div class="flex items-center gap-2.5 px-4 py-3 bg-gradient-to-l from-slate-50 to-white border-b border-slate-200">
        <span class="bg-gradient-to-l from-blue-600 to-blue-500 text-white font-bold text-[13px] px-3.5 py-1 rounded-full shadow-sm">
          وزن {{ b.cat }}
        </span>
        <span class="text-[11px] text-slate-400">{{ b.athleteCount }} شرکت‌کننده</span>
        <span v-if="b.finished" class="mr-auto text-xs text-emerald-600 font-semibold bg-emerald-50 px-2.5 py-1 rounded-full">
          ✓ پایان یافته
        </span>
      </div>

      <!-- براکت -->
      <div class="bracket-zoom p-4" :style="{ zoom: String(b.zoom) }">
        <div class="flex items-stretch justify-center">
          <!-- بلوک چپ -->
          <div class="flex gap-6">
            <div v-for="col in b.columns" :key="'l' + col.round" class="flex flex-col w-[132px]">
              <div class="text-center text-[10.5px] font-bold text-slate-500 mb-2.5 bg-slate-100/70 rounded-md py-1">
                {{ col.label }}
              </div>
              <div class="flex flex-col gap-[5px] flex-1 justify-around">
                <div
                    v-for="m in col.left"
                    :key="m.id"
                    :role="isPlayable(m) ? 'button' : undefined"
                    :tabindex="isPlayable(m) ? 0 : undefined"
                    :aria-label="isPlayable(m) ? 'ثبت نتیجه بازی' : undefined"
                    class="rounded-lg transition print:cursor-auto"
                    :class="isPlayable(m)
        ? 'cursor-pointer hover:ring-2 hover:ring-blue-300/70 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500'
        : 'cursor-default'"
                    @click="openMatch(m)"
                    @keydown.enter.prevent="openMatch(m)"
                    @keydown.space.prevent="openMatch(m)"
                >
                  <StandingCard
                      :match="m"
                      :athlete-map="athleteMap"
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- فینال -->
          <div class="flex flex-col items-center w-[150px] mx-1 px-3 border-r border-l border-dashed border-amber-300/70">
            <div class="text-center text-[10.5px] font-bold text-amber-700 mb-2.5 bg-amber-50 rounded-md py-1 w-full">
              فینال
            </div>

            <div class="flex flex-col gap-[5px] flex-1 justify-around">
              <div
                  v-for="m in b.finals"
                  :key="m.id"
                  :role="isPlayable(m) ? 'button' : undefined"
                  :tabindex="isPlayable(m) ? 0 : undefined"
                  :aria-label="isPlayable(m) ? 'ثبت نتیجه بازی' : undefined"
                  class="rounded-lg transition print:cursor-auto"
                  :class="isPlayable(m)
        ? 'cursor-pointer hover:ring-2 hover:ring-blue-300/70 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500'
        : 'cursor-default'"
                  @click="openMatch(m)"
                  @keydown.enter.prevent="openMatch(m)"
                  @keydown.space.prevent="openMatch(m)"
              >
                <StandingCard
                    :match="m"
                    :athlete-map="athleteMap"
                />
              </div>
            </div>
            <div class="hidden print:block mt-4 text-[11px] text-center leading-relaxed">
              <div v-for="e in b.standings" :key="e.key" class="flex items-center gap-1.5 justify-center">
                <span>{{ e.medal }}</span><span>{{ e.name }}</span>
              </div>
            </div>
          </div>

          <!-- بلوک راست -->
          <div class="flex flex-row-reverse gap-6">
            <div v-for="col in b.columns" :key="'r' + col.round" class="flex flex-col w-[132px]">
              <div class="text-center text-[10.5px] font-bold text-slate-500 mb-2.5 bg-slate-100/70 rounded-md py-1">
                {{ col.label }}
              </div>
              <div class="flex flex-col gap-[5px] flex-1 justify-around">
                <div
                    v-for="m in col.right"
                    :key="m.id"
                    :role="isPlayable(m) ? 'button' : undefined"
                    :tabindex="isPlayable(m) ? 0 : undefined"
                    :aria-label="isPlayable(m) ? 'ثبت نتیجه بازی' : undefined"
                    class="rounded-lg transition print:cursor-auto"
                    :class="isPlayable(m)
        ? 'cursor-pointer hover:ring-2 hover:ring-blue-300/70 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500'
        : 'cursor-default'"
                    @click="openMatch(m)"
                    @keydown.enter.prevent="openMatch(m)"
                    @keydown.space.prevent="openMatch(m)"
                >
                  <StandingCard
                      :match="m"
                      :athlete-map="athleteMap"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- رده‌بندی -->
      <div class="border-t border-slate-200 px-4 py-3 bg-slate-50/70 print:hidden">
        <div class="text-[13px] font-bold text-slate-700 mb-2.5">رده‌بندی</div>
        <div class="flex gap-2.5 flex-wrap">
          <div
              v-for="e in b.standings"
              :key="e.key"
              class="flex items-center gap-2.5 px-3.5 py-2 rounded-xl border bg-gradient-to-l"
              :class="{
                'from-amber-50 to-white border-amber-200': e.bgClass === 'gold',
                'from-slate-100 to-white border-slate-300': e.bgClass === 'silver',
                'from-orange-50 to-white border-orange-200': e.bgClass === 'bronze',
              }"
          >
            <span class="text-lg leading-none">{{ e.medal }}</span>
            <div class="min-w-0">
              <div class="text-[12.5px] font-bold text-slate-800 whitespace-nowrap">{{ e.name }}</div>
              <div v-if="e.club" class="text-[10.5px] text-slate-400">{{ e.club }}</div>
            </div>
          </div>
          <div v-if="b.standings.length === 0" class="text-xs text-slate-400 px-4 py-2 border border-dashed border-slate-200 rounded-xl">
            هنوز نتیجه‌ای ثبت نشده
          </div>
        </div>
      </div>
    </div>
    <MatchResultModal
        v-if="activeMatch && store.currentTournament"
        :match="activeMatch"
        :tournament-id="store.currentTournament.id"
        :athlete-map="athleteMap"
        @close="activeMatchId = null"
    />

    <MatchResultDetails
        v-if="detailsMatch && store.currentTournament"
        :match="detailsMatch"
        :athlete-map="athleteMap"
        @close="detailsMatchId = null"
        @edit="editFromDetails"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import StandingCard from './StandingCard.vue'
import { useTournamentStore } from '../stores/tournament'
import MatchResultModal from './MatchResultModal.vue'
import MatchResultDetails from './MatchResultDetails.vue'

const activeMatch = computed(() =>
    activeMatchId.value
        ? allMatches.value.find(m => m.id === activeMatchId.value) ?? null
        : null)

const detailsMatch = computed(() =>
    detailsMatchId.value
        ? allMatches.value.find(m => m.id === detailsMatchId.value) ?? null
        : null
)

/** بازی قابل ثبت است؟ */
function isPlayable(m: any) {
  return !m.isBye && !!m.athlete1Id && !!m.athlete2Id
}

function openMatch(m: any) {
  if (!isPlayable(m)) return
  if (m.winnerId) detailsMatchId.value = m.id
  else activeMatchId.value = m.id
}

function editFromDetails(matchId: string) {
  detailsMatchId.value = null
  activeMatchId.value = matchId
}

const store = useTournamentStore()
const activeMatchId = ref<string | null>(null)
const detailsMatchId = ref<string | null>(null)

/** true = نیمه‌ی چپ و راست جابه‌جا شوند */
const SWAP_HALVES = true

const COL_W = 132
const COL_GAP = 24
const FINAL_W = 158
const EXTRA = 48
const MIN_ZOOM = 0.32
const PRINT_W = 1010 // A4 landscape @96dpi با حاشیه ۱۰mm

const allMatches = computed(() => store.currentTournament?.matches ?? [])
const athletes = computed(() => store.currentTournament?.athletes ?? [])
const hasMatches = computed(() => allMatches.value.length > 0)
const selectedCategory = ref<string | null>(null)

/* ---------- اندازه‌گیری عرض: یک observer، throttle با rAF ---------- */
const rootEl = ref<HTMLElement | null>(null)
const containerWidth = ref(1200)
let ro: ResizeObserver | null = null
let rafId = 0

onMounted(() => {
  if (!rootEl.value) return
  containerWidth.value = rootEl.value.clientWidth || 1200
  if (typeof ResizeObserver === 'undefined') return
  ro = new ResizeObserver(entries => {
    const w = entries[0].contentRect.width
    if (rafId) return
    rafId = requestAnimationFrame(() => {
      rafId = 0
      if (Math.abs(w - containerWidth.value) > 4) containerWidth.value = w
    })
  })
  ro.observe(rootEl.value)
})

onBeforeUnmount(() => {
  ro?.disconnect()
  if (rafId) cancelAnimationFrame(rafId)
})

/* ---------- کمکی‌ها ---------- */
const athleteMap = computed(() => {
  const m = new Map<string, any>()
  for (const a of athletes.value) m.set(a.id, a)
  return m
})

const ROUND_LABELS = [
  'فینال', 'نیمه‌نهایی', 'یک‌چهارم نهایی', 'یک‌هشتم نهایی',
  'یک‌شانزدهم نهایی', 'یک‌سی‌ودوم نهایی',
]

function bracketKey(m: any) {
  return m.bracketIndex ?? m.order ?? 0
}

function weightSortValue(cat: string) {
  const hit = cat.match(/(\d+(\.\d+)?)/)
  const num = hit ? parseFloat(hit[1]) : 0
  return cat.includes('+') ? num + 0.5 : num
}

/* ---------- محاسبه‌ی یک‌باره‌ی همه‌ی داده‌ها ---------- */
const brackets = computed(() => {
  const groups = new Map<string, any[]>()
  for (const m of allMatches.value) {
    const arr = groups.get(m.weightCategory)
    if (arr) arr.push(m)
    else groups.set(m.weightCategory, [m])
  }

  const am = athleteMap.value
  const out: any[] = []

  for (const [cat, ms] of groups) {
    const byRound = new Map<number, any[]>()
    for (const m of ms) {
      const arr = byRound.get(m.round)
      if (arr) arr.push(m)
      else byRound.set(m.round, [m])
    }
    const rounds = [...byRound.keys()].sort((a, b) => a - b)
    for (const arr of byRound.values()) arr.sort((a, b) => bracketKey(a) - bracketKey(b))

    const finalRound = rounds[rounds.length - 1]
    const semiRound = rounds[rounds.length - 2]

    // ستون‌ها: هر دور به دو نیمه تقسیم و در صورت نیاز جابه‌جا می‌شود
    const columns = rounds.slice(0, -1).map((r, i) => {
      const list = byRound.get(r) ?? []
      const mid = Math.ceil(list.length / 2)
      let left = list.slice(0, mid)
      let right = list.slice(mid)
      if (SWAP_HALVES) [left, right] = [right, left]
      const fromEnd = rounds.length - 1 - i
      return {
        round: r,
        label: ROUND_LABELS[fromEnd] ?? `دور ${r}`,
        left,
        right,
      }
    })

    const finals = byRound.get(finalRound) ?? []

    // رده‌بندی
    const standings: any[] = []
    const fm = finals[0]
    if (fm?.winnerId) {
      const gold = am.get(fm.winnerId)
      const silverId = fm.athlete1Id === fm.winnerId ? fm.athlete2Id : fm.athlete1Id
      const silver = silverId ? am.get(silverId) : null
      if (gold) standings.push({ key: 'g' + gold.id, medal: '🥇', name: gold.name, club: gold.club ?? '', bgClass: 'gold' })
      if (silver) standings.push({ key: 's' + silver.id, medal: '🥈', name: silver.name, club: silver.club ?? '', bgClass: 'silver' })
    }
    if (semiRound !== undefined) {
      for (const semi of byRound.get(semiRound) ?? []) {
        if (!semi.winnerId) continue
        const loserId = semi.athlete1Id === semi.winnerId ? semi.athlete2Id : semi.athlete1Id
        const bronze = loserId ? am.get(loserId) : null
        if (bronze) standings.push({ key: 'b' + bronze.id, medal: '🥉', name: bronze.name, club: bronze.club ?? '', bgClass: 'bronze' })
      }
    }

    // تعداد شرکت‌کننده‌های واقعی
    const ids = new Set<string>()
    for (const m of ms) {
      if (m.athlete1Id) ids.add(m.athlete1Id)
      if (m.athlete2Id) ids.add(m.athlete2Id)
    }

    const needed = columns.length * 2 * (COL_W + COL_GAP) + FINAL_W + EXTRA

    out.push({
      cat,
      columns,
      finals,
      standings,
      needed,
      athleteCount: ids.size,
      finished: !!fm?.winnerId,
    })
  }

  return out.sort((a, b) => weightSortValue(a.cat) - weightSortValue(b.cat))
})

/* zoom جدا نگه داشته شده تا تغییر عرض، محاسبات سنگین را دوباره اجرا نکند */
const visibleBrackets = computed(() => {
  const w = containerWidth.value - 8
  const list = selectedCategory.value
      ? brackets.value.filter(b => b.cat === selectedCategory.value)
      : brackets.value
  return list.map(b => ({
    ...b,
    zoom: Math.max(MIN_ZOOM, Math.min(1, w / b.needed)),
    printZoom: Math.min(1, PRINT_W / b.needed),
  }))
})
</script>

<style scoped>
.bracket-zoom { transform-origin: top center; }
</style>
