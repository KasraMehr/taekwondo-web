<template>
  <!-- دکمه اصلی چاپ -->
  <button
      type="button"
      class="no-print fixed bottom-6 left-6 z-50 rounded-xl bg-slate-900 px-5 py-3 text-sm font-bold text-white shadow-xl transition hover:-translate-y-0.5 hover:bg-slate-800"
      @click="print"
  >
    🖨️
    {{
      selectedCategory === 'all'
          ? 'خروجی PDF همه وزن‌ها'
          : `خروجی PDF ${getWeightLabel(selectedCategory)}`
    }}
  </button>

  <div
      id="bracket-print-root"
      class="min-h-screen bg-slate-100 p-4 font-sans"
      dir="rtl"
  >
    <!-- فیلتر وزن؛ در چاپ نمایش داده نمی‌شود -->
    <div
        class="no-print sticky top-0 z-40 mb-4 flex flex-wrap items-center gap-2 rounded-2xl border border-slate-200 bg-white/95 px-3 py-2.5 shadow-sm backdrop-blur"
    >
      <span class="ml-1 text-[11px] font-extrabold text-slate-500">
        فیلتر وزن:
      </span>

      <button
          type="button"
          class="rounded-full px-3.5 py-2 text-[11px] font-extrabold leading-none transition"
          :class="
            selectedCategory === 'all'
              ? 'bg-slate-900 text-white shadow-sm ring-1 ring-slate-900'
              : 'bg-slate-100 text-slate-700 ring-1 ring-slate-200 hover:bg-slate-200'
          "
          @click="selectedCategory = 'all'"
      >
        همه وزن‌ها
        <span class="opacity-70">
          ({{ toFaNumber(allCategories.length) }})
        </span>
      </button>

      <button
          v-for="cat in allCategories"
          :key="cat"
          type="button"
          class="rounded-full px-3.5 py-2 text-[11px] font-extrabold leading-none transition"
          :class="
            selectedCategory === cat
              ? 'bg-slate-900 text-white shadow-sm ring-1 ring-slate-900'
              : 'bg-white text-slate-700 ring-1 ring-slate-200 hover:bg-slate-100'
          "
          @click="selectedCategory = cat"
      >
        {{ getWeightLabel(cat) }}
      </button>
    </div>

    <!-- پیام نبودن مسابقه -->
    <div
        v-if="pages.length === 0"
        class="rounded-2xl border border-dashed border-slate-300 bg-white p-10 text-center"
    >
      <div class="text-4xl">🥋</div>

      <div class="mt-3 text-base font-black text-slate-700">
        مسابقه‌ای برای چاپ وجود ندارد
      </div>

      <div class="mt-1 text-xs text-slate-400">
        ابتدا جدول مسابقات را ایجاد کنید.
      </div>
    </div>

    <!-- یک صفحه برای هر بخش یا وزن -->
    <section
        v-for="page in pages"
        :key="page.key"
        class="category-page mb-8 rounded-2xl border border-slate-200 bg-white shadow-md"
    >
      <!-- هدر صفحه؛ تمام اطلاعات کلیدی داخل همین نوار رنگی است -->
      <!-- هدر صفحه - چیدمان مرکزی -->
      <header class="pdf-header">
        <!-- سمت راست: تاریخ + وزن -->
        <div class="header-side header-side-start">
          <span v-if="t?.date" class="header-pill header-date">
            📅 {{ toJalaliDate(t.date) }}
          </span>
          <span class="header-pill header-weight">
            ⚖️ {{ getWeightLabel(page.cat) }}
          </span>
        </div>

        <!-- وسط: عنوان + سازمان -->
        <div class="header-center">
          <div class="header-title">
            {{ t?.name || 'مسابقات تکواندو' }}
            <span v-if="page.partTitle" class="header-part">
        ({{ page.partTitle }})
      </span>
          </div>
          <div class="header-org">
            فدراسیون تکواندو جمهوری اسلامی ایران
          </div>
        </div>

        <!-- سمت چپ: تعداد + زمین + چاپ -->
        <div class="header-side header-side-end">
    <span class="header-pill header-count">
      👥 {{ toFaNumber(page.athleteCount) }} نفر
    </span>
          <span v-if="page.courts.length" class="header-pill header-court">
      {{ page.courts.map(c => normalizedCourtLabel(c)).join(' • ') }}
    </span>
          <button
              type="button"
              class="no-print header-print-btn"
              @click="printCategory(page.cat)"
          >
            🖨️ چاپ
          </button>
        </div>
      </header>

      <!-- براکت اصلی -->
      <div class="bracket-scroll-container">
        <div
            class="bracket-stage"
            :class="`size-${page.displaySize}`"
            :style="bracketStyle(page.sideMatchCount)"
            dir="ltr"
        >
          <!-- سمت چپ -->
          <div class="bracket-side left-flow">
            <div
                v-for="col in sortedSideColumns(page.data.left, 'left')"
                :key="`left-${page.key}-${col.round}`"
                class="round-column"
            >
              <div class="round-heading" dir="rtl">
                <span>
                  {{ roundLabel(col.round + page.roundOffset, page.totalRounds) }}
                </span>

                <small>
                  {{ toFaNumber(col.matches.length) }} بازی
                </small>
              </div>

              <div class="round-grid">
                <div
                    v-for="(match, index) in col.matches"
                    :key="match.id"
                    class="match-node"
                    :class="{
                    'pair-top': index % 2 === 0,
                    'pair-bottom': index % 2 === 1,
                  }"
                    :style="{
                    gridRow: `span ${roundSpan(col.round)}`,
                  }"
                >
                  <MatchBox
                      class="bracket-match-card"
                      :match="match"
                      :athletes="athletes"
                      :show-meta="!match.isBye"
                      :placeholder1="page.slotLabels?.[match.id]?.athlete1"
                      :placeholder2="page.slotLabels?.[match.id]?.athlete2"
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- ستون فینال -->
          <div class="final-column">
            <div
                class="round-heading final-heading-spacer"
                aria-hidden="true"
            >
              <span>&nbsp;</span>
              <small>&nbsp;</small>
            </div>

            <div class="final-track">
              <div
                  class="final-match-anchor"
                  :class="{ 'is-placeholder': !page.showFinal }"
              >
                <template v-if="page.showFinal">
                  <div class="final-badge" dir="rtl">
                    <span class="final-badge-icon">🏆</span>

                    <div>
                      <strong>{{ page.finalLabel }}</strong>
                      <small>مسابقه قهرمانی</small>
                    </div>
                  </div>

                  <MatchBox
                      v-if="page.data.final"
                      class="bracket-match-card final-match-card"
                      :match="page.data.final"
                      :athletes="athletes"
                      :is-final="true"
                      :show-meta="!page.data.final.isBye"
                      :placeholder1="page.slotLabels?.[page.data.final.id]?.athlete1"
                      :placeholder2="page.slotLabels?.[page.data.final.id]?.athlete2"
                  />

                  <div
                      v-else
                      class="empty-final-card"
                      dir="rtl"
                  >
                    <span>🏆</span>
                    <strong>فینال هنوز مشخص نشده است</strong>
                  </div>

                  <div
                      v-if="page.data.final?.winnerId"
                      class="champion-card"
                      dir="rtl"
                  >
                    <div class="champion-crown">
                      ★
                    </div>

                    <div class="champion-content">
                      <span class="champion-label">
                        قهرمان {{ getWeightLabel(page.cat) }}
                      </span>

                      <strong>
                        {{ athleteInfo(page.data.final.winnerId).name }}
                      </strong>

                      <small
                          v-if="athleteInfo(page.data.final.winnerId).club"
                      >
                        {{ athleteInfo(page.data.final.winnerId).club }}
                      </small>
                    </div>
                  </div>

                  <!-- رده‌بندی نهایی؛ رتبه ۲ و دو نفر سوم مشترک -->
                  <div
                      v-if="page.podium.length > 1"
                      class="podium-card"
                      dir="rtl"
                  >
                    <span class="podium-title">رده‌بندی نهایی</span>

                    <div
                        v-for="row in page.podium"
                        :key="`${row.rank}-${row.name}`"
                        class="podium-row"
                        :class="`podium-rank-${row.rank}`"
                    >
                      <span class="podium-rank">
                        {{ toFaNumber(row.rank) }}
                      </span>

                      <div class="podium-info">
                        <strong>{{ row.name }}</strong>

                        <small v-if="row.club">
                          {{ row.club }}
                        </small>
                      </div>
                    </div>
                  </div>
                </template>

                <template v-else>
                  <div class="empty-final-card" dir="rtl">
                    <span>➡️</span>
                    <strong>ادامه براکت در صفحه بعد</strong>
                  </div>
                </template>
              </div>
            </div>
          </div>

          <!-- سمت راست -->
          <div class="bracket-side right-flow">
            <div
                v-for="col in sortedSideColumns(page.data.right, 'right')"
                :key="`right-${page.key}-${col.round}`"
                class="round-column"
            >
              <div class="round-heading" dir="rtl">
                <span>
                  {{ roundLabel(col.round + page.roundOffset, page.totalRounds) }}
                </span>

                <small>
                  {{ toFaNumber(col.matches.length) }} بازی
                </small>
              </div>

              <div class="round-grid">
                <div
                    v-for="(match, index) in col.matches"
                    :key="match.id"
                    class="match-node"
                    :class="{
                    'pair-top': index % 2 === 0,
                    'pair-bottom': index % 2 === 1,
                  }"
                    :style="{
                    gridRow: `span ${roundSpan(col.round)}`,
                  }"
                >
                  <MatchBox
                      class="bracket-match-card"
                      :match="match"
                      :athletes="athletes"
                      :show-meta="!match.isBye"
                      :placeholder1="page.slotLabels?.[match.id]?.athlete1"
                      :placeholder2="page.slotLabels?.[match.id]?.athlete2"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import {
  computed,
  defineComponent,
  h,
  nextTick,
  ref,
  type CSSProperties,
  type PropType, onUnmounted, onMounted,
} from 'vue'

import { useTournamentStore } from '../stores/tournament'
import {
  buildBracketColumns,
  courtLabel,
} from '../utils/printBracket'

import {
  type AgeCategory,
  type Gender,
  WEIGHT_CATEGORIES,
} from '../data/categories'

type MatchLike = {
  id: string
  athlete1Id: string | null
  athlete2Id: string | null
  winnerId?: string | null
  court: number
  order: number
  bracketIndex?: number
  weightCategory: string
  round: number
  side: 'left' | 'right' | 'final'
  nextMatchId?: string
  isBye?: boolean
  nextSlot?: 'athlete1' | 'athlete2'
}

type AthleteLike = {
  id: string
  name?: string | null
  fullName?: string | null
  club?: string | null
  coach?: string | null
}

type BracketColumn = {
  round: number
  matches: MatchLike[]
}

type BracketData = {
  left: BracketColumn[]
  right: BracketColumn[]
  final: MatchLike | null
}

type PodiumRow = {
  rank: number
  name: string
  club: string
}

type SlotLabelMap = Record<
    string,
    { athlete1?: string; athlete2?: string }
>

type BracketPage = {
  key: string
  cat: string
  courts: number[]
  matchCount: number
  totalRounds: number
  bracketSize: number
  displaySize: number
  sideMatchCount: number
  data: BracketData
  isSplitted: boolean
  partTitle?: string
  showFinal: boolean
  athleteCount: number
  weightIndex: number
  weightTotal: number
  podium: PodiumRow[]
  finalLabel: string
  roundOffset: number
  slotLabels: SlotLabelMap
  advanceHint?: string
  groupIndex?: number
  groupTotal?: number
}

const store = useTournamentStore()
const t = computed(() => store.currentTournament)

const athletes = computed<AthleteLike[]>(
    () => t.value?.athletes ?? []
)

function athleteInfo(
    id: string | null | undefined
): { name: string; club: string } {
  if (!id) {
    return {
      name: '-',
      club: '',
    }
  }

  const athlete = athletes.value.find(item => item.id === id)

  return {
    name:
        athlete?.name ??
        athlete?.fullName ??
        '-',
    club:
        athlete?.club ??
        '',
  }
}

function toFaNumber(
    value: number | string | null | undefined
): string {
  if (
      value === null ||
      value === undefined ||
      value === ''
  ) {
    return '۰'
  }

  const numericValue = Number(value)

  if (Number.isNaN(numericValue)) {
    return String(value)
        .replace(/0/g, '۰')
        .replace(/1/g, '۱')
        .replace(/2/g, '۲')
        .replace(/3/g, '۳')
        .replace(/4/g, '۴')
        .replace(/5/g, '۵')
        .replace(/6/g, '۶')
        .replace(/7/g, '۷')
        .replace(/8/g, '۸')
        .replace(/9/g, '۹')
  }

  return new Intl.NumberFormat('fa-IR', {
    useGrouping: false,
  }).format(numericValue)
}

function toJalaliDate(
    value: string | number | Date | null | undefined
): string {
  if (!value) return ''

  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)

  return new Intl.DateTimeFormat('fa-IR-u-ca-persian', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  }).format(d)
}


function normalizedCourtLabel(court: number): string {
  const label = courtLabel(court)

  if (
      String(label).includes('زمین') ||
      String(label).toLowerCase().includes('court')
  ) {
    return toFaNumber(String(label))
  }

  return `زمین ${toFaNumber(label)}`
}

/** حرف اختصاری زمین؛ اگر courtLabel عدد بدهد از روی شماره ساخته می‌شود */
function courtToken(court: number): string {
  const raw = String(courtLabel(court))
  const letter = raw.match(/[A-Za-z]/)

  if (letter) {
    return letter[0].toUpperCase()
  }

  return String.fromCharCode(
      64 + Math.min(26, Math.max(1, court))
  )
}

/** برچسب ترکیبی زمین و شماره بازی؛ مثل C۲۶ */
function matchTag(match: MatchLike): string {
  if (isTrueBye(match)) {
    return courtToken(match.court)
  }

  return `${courtToken(match.court)}${toFaNumber(match.order ?? 0)}`
}

function weightSortValue(category: string): number {
  const match = category.match(/(\d+(\.\d+)?)/)
  const number = match ? parseFloat(match[1]) : 9999

  return category.includes('+')
      ? number + 0.5
      : number
}

const allCategories = computed<string[]>(() => {
  const tournament = t.value
  if (!tournament) return []

  const age = tournament.ageCategory as AgeCategory
  const gender = tournament.gender as Gender
  const officialCategories = WEIGHT_CATEGORIES[age]?.[gender] ?? []

  const usedCategories = new Set(
      ((tournament.matches ?? []) as MatchLike[]).map(
          match => match.weightCategory
      )
  )

  return officialCategories.filter(category =>
      usedCategories.has(category)
  )
})

const officialCategories = computed<string[]>(() => {
  const tournament = t.value
  if (!tournament) return []

  const age = tournament.ageCategory as AgeCategory
  const gender = tournament.gender as Gender

  return WEIGHT_CATEGORIES[age]?.[gender] ?? []
})

const selectedCategory = ref<string>('all')

function isTrueBye(match: MatchLike): boolean {
  if (match.isBye === true) {
    return true
  }

  return (
      match.round === 1 &&
      ((match.athlete1Id === null) !==
          (match.athlete2Id === null))
  )
}

function isPrintableMatch(_match: MatchLike): boolean {
  return true
}

function sortedSideColumns(
    columns: BracketColumn[] | undefined,
    side: 'left' | 'right'
): BracketColumn[] {
  const result = [...(columns ?? [])]

  if (side === 'left') {
    return result.sort((a, b) => a.round - b.round)
  }

  return result.sort((a, b) => b.round - a.round)
}

function roundSpan(round: number): number {
  return Math.pow(2, Math.max(0, round - 1))
}

function bracketStyle(
    sideMatchCount: number
): CSSProperties {
  return {
    '--side-matches': String(Math.max(1, sideMatchCount)),
  } as CSSProperties
}

function getSideMatchCount(data: BracketData): number {
  const columns = [...(data.left ?? []), ...(data.right ?? [])]
  if (!columns.length) return 1

  const entryRound = Math.min(...columns.map(c => c.round))

  return Math.max(
      1,
      ...columns
          .filter(c => c.round === entryRound)
          .map(c => c.matches.length)
  )
}

const GROUP_SIZE = 32

/** ۳۲→۱۶→۸→۴→۲→۱ یعنی پنج دور داخل هر گروه */
const GROUP_ROUNDS = 5

function isPowerOfTwo(value: number): boolean {
  return value >= 1 && (value & (value - 1)) === 0
}

/** سهم یک گروه از یک سمت براکت، محدود به دورهای ۱..maxRound */
function sliceSide(
    columns: BracketColumn[] | undefined,
    part: number,
    parts: number,
    maxRound: number
): BracketColumn[] {
  return (columns ?? [])
      .filter(c => c.round <= maxRound)
      .map(c => {
        const size = c.matches.length / parts
        if (size < 1) return { round: c.round, matches: [] as MatchLike[] }

        return {
          round: c.round,
          matches: c.matches.slice(
              Math.round(part * size),
              Math.round((part + 1) * size)
          ),
        }
      })
      .filter(c => c.matches.length > 0)
}

/** زیرجدول ۳۲ نفره → چیدمان روبه‌رو: نیمه بالا چپ، نیمه پایین راست، تک‌بازی آخر مرکز */
function toFacingBracket(columns: BracketColumn[]): BracketData {
  if (!columns.length) return { left: [], right: [], final: null }

  const maxRound = Math.max(...columns.map(c => c.round))
  const lastColumn = columns.find(c => c.round === maxRound)
  const hasDecider = lastColumn?.matches.length === 1

  const left: BracketColumn[] = []
  const right: BracketColumn[] = []

  for (const c of columns) {
    if (hasDecider && c.round === maxRound) continue

    const half = Math.ceil(c.matches.length / 2)
    const top = c.matches.slice(0, half)
    const bottom = c.matches.slice(half)

    if (top.length) left.push({ round: c.round, matches: top })
    if (bottom.length) right.push({ round: c.round, matches: bottom })
  }

  return {
    left,
    right,
    final: hasDecider ? lastColumn!.matches[0] : null,
  }
}

/** دورهای بعد از مرحله گروهی، شماره‌گذاری بازنویسی‌شده از ۱ */
function finalStageColumns(
    columns: BracketColumn[] | undefined
): BracketColumn[] {
  return (columns ?? [])
      .filter(c => c.round > GROUP_ROUNDS)
      .map(c => ({ round: c.round - GROUP_ROUNDS, matches: c.matches }))
      .filter(c => c.matches.length > 0)
}

/** اسلات‌های ورودی صفحه نهایی: چپ بالا→پایین، سپس راست */
function buildEntrySlots(
    data: BracketData
): Array<{ matchId: string; slot: 'athlete1' | 'athlete2' }> {
  const columns = [...data.left, ...data.right]

  const source = columns.length
      ? (() => {
        const entryRound = Math.min(...columns.map(c => c.round))

        return [
          ...data.left.filter(c => c.round === entryRound).flatMap(c => c.matches),
          ...data.right.filter(c => c.round === entryRound).flatMap(c => c.matches),
        ]
      })()
      : data.final ? [data.final] : []

  return source.flatMap(m => [
    { matchId: m.id, slot: 'athlete1' as const },
    { matchId: m.id, slot: 'athlete2' as const },
  ])
}


/** اندیس ساختاری براکت؛ order برای مسابقات دارای استراحت صفر است و قابل اتکا نیست */
function bracketKey(match: MatchLike): number {
  return match.bracketIndex ?? match.order ?? 0
}

function normalizeColumns(data: BracketData): BracketData {
  const fix = (columns: BracketColumn[] | undefined): BracketColumn[] =>
      [...(columns ?? [])].map(column => ({
        round: column.round,
        matches: [...column.matches].sort(
            (a, b) => bracketKey(a) - bracketKey(b)
        ),
      }))

  return {
    left: fix(data.left),
    right: fix(data.right),
    final: data.final,
  }
}

/** تعداد شرکت‌کننده واقعی وزن */
function countAthletes(matches: MatchLike[]): number {
  const ids = new Set<string>()

  for (const match of matches) {
    if (match.athlete1Id) ids.add(match.athlete1Id)
    if (match.athlete2Id) ids.add(match.athlete2Id)
  }

  return ids.size
}

/** رتبه ۱ تا ۴؛ در تکواندو دو بازنده نیمه‌نهایی هر دو سوم مشترک هستند */
function buildPodium(
    matches: MatchLike[],
    totalRounds: number
): PodiumRow[] {
  const final = matches.find(m => m.round === totalRounds)

  if (!final?.winnerId) {
    return []
  }

  const rows: PodiumRow[] = []

  const goldId = final.winnerId

  const silverId =
      final.athlete1Id === goldId
          ? final.athlete2Id
          : final.athlete1Id

  const gold = athleteInfo(goldId)

  rows.push({
    rank: 1,
    name: gold.name,
    club: gold.club,
  })

  if (silverId) {
    const silver = athleteInfo(silverId)

    rows.push({
      rank: 2,
      name: silver.name,
      club: silver.club,
    })
  }

  const semis = matches
      .filter(m => m.round === totalRounds - 1)
      .sort((a, b) => bracketKey(a) - bracketKey(b))

  for (const semi of semis) {
    if (!semi.winnerId) continue
    if (isTrueBye(semi)) continue

    const loserId =
        semi.athlete1Id === semi.winnerId
            ? semi.athlete2Id
            : semi.athlete1Id

    if (!loserId) continue

    const bronze = athleteInfo(loserId)

    rows.push({
      rank: 3,
      name: bronze.name,
      club: bronze.club,
    })
  }

  return rows
}

const pages = computed<BracketPage[]>(() => {
  const matches = (t.value?.matches ?? []) as MatchLike[]
  const printableMatches = matches.filter(isPrintableMatch)
  const categoryMap = new Map<string, MatchLike[]>()

  for (const match of printableMatches) {
    const key = match.weightCategory
    if (!categoryMap.has(key)) categoryMap.set(key, [])
    categoryMap.get(key)!.push(match)
  }

  const sortedCategories = [...categoryMap.entries()].sort(([catA], [catB]) => {
    const indexA = allCategories.value.indexOf(catA)
    const indexB = allCategories.value.indexOf(catB)

    if (indexA === -1 && indexB === -1) {
      return weightSortValue(catA) - weightSortValue(catB)
    }

    if (indexA === -1) return 1
    if (indexB === -1) return -1

    return indexA - indexB
  })

  const allPages: BracketPage[] = []
  const weightTotal = sortedCategories.length

  sortedCategories.forEach(([category, categoryMatches], index) => {
    const sortedMatches = [...categoryMatches].sort((a, b) => {
      if (a.round !== b.round) return a.round - b.round
      return bracketKey(a) - bracketKey(b)
    })

    const fullData = normalizeColumns(
        buildBracketColumns(
            sortedMatches.map(m => ({
              ...m,
              winnerId: m.winnerId ?? undefined,
            }))
        ) as BracketData
    )

    const totalRounds = Math.max(...sortedMatches.map(m => m.round), 1)
    const firstRoundCount = sortedMatches.filter(m => m.round === 1).length
    const bracketSize = Math.max(2, firstRoundCount * 2)

    const courts = [
      ...new Set(
          sortedMatches
              .filter(m => !isTrueBye(m))
              .map(m => m.court)
      ),
    ].sort((a, b) => a - b)

    const printableMatchCount = sortedMatches.filter(m => !isTrueBye(m)).length

    const athleteCount = countAthletes(sortedMatches)
    const podium = buildPodium(sortedMatches, totalRounds)
    const weightIndex = index + 1

    // تفکیک ۳ صفحه‌ای صرفاً برای جداول بزرگ‌تر از ۳۲ نفر
    const basePage = {
      cat: category,
      courts,
      matchCount: printableMatchCount,
      totalRounds,
      bracketSize,
      athleteCount,
      weightIndex,
      weightTotal,
    }

    const groupCount = bracketSize / GROUP_SIZE

    // تا ۳۲ نفر و اندازه‌های غیرمتعارف: یک صفحه
    if (bracketSize <= GROUP_SIZE || groupCount < 2 || !isPowerOfTwo(groupCount)) {
      allPages.push({
        ...basePage,
        key: category,
        displaySize: bracketSize,
        sideMatchCount: getSideMatchCount(fullData),
        isSplitted: false,
        showFinal: true,
        data: fullData,
        podium,
        roundOffset: 0,
        finalLabel: 'فینال',
        slotLabels: {},
      })

      return
    }

    const finalStageData: BracketData = {
      left: finalStageColumns(fullData.left),
      right: finalStageColumns(fullData.right),
      final: fullData.final,
    }

    // گروه‌ها: اول سمت چپ از بالا به پایین، سپس سمت راست
    const partsPerSide = groupCount / 2
    const groups: BracketData[] = []

    for (const side of ['left', 'right'] as const) {
      for (let part = 0; part < partsPerSide; part++) {
        groups.push(
            toFacingBracket(
                sliceSide(
                    side === 'left' ? fullData.left : fullData.right,
                    part,
                    partsPerSide,
                    GROUP_ROUNDS
                )
            )
        )
      }
    }

    // نقشه «برنده گروه n» برای اسلات‌های خالی صفحه نهایی
    const entrySlots = buildEntrySlots(finalStageData)
    const entryRoundLabel = roundLabel(GROUP_ROUNDS + 1, totalRounds)
    const slotLabels: SlotLabelMap = {}

    groups.forEach((group, i) => {
      const decider = group.final

      const target =
          decider?.nextMatchId && decider.nextSlot
              ? { matchId: decider.nextMatchId, slot: decider.nextSlot }
              : entrySlots[i]

      if (!target) return

      const bucket = slotLabels[target.matchId] ?? {}
      bucket[target.slot] = `برنده گروه ${toFaNumber(i + 1)}`
      slotLabels[target.matchId] = bucket
    })

    // صفحات ۳۲ نفره
    groups.forEach((group, i) => {
      allPages.push({
        ...basePage,
        key: `${category}-g${i + 1}`,
        displaySize: GROUP_SIZE,
        sideMatchCount: getSideMatchCount(group),
        isSplitted: true,
        partTitle:
            `گروه ${toFaNumber(i + 1)} از ${toFaNumber(groupCount)}` +
            ` — جدول ${toFaNumber(GROUP_SIZE)} نفره`,
        advanceHint:
            `برنده گروه ${toFaNumber(i + 1)} → جایگاه ${toFaNumber(i + 1)}` +
            ` در ${entryRoundLabel}`,
        showFinal: Boolean(group.final),
        data: group,
        podium: [],
        roundOffset: 0,
        finalLabel: roundLabel(group.final?.round ?? GROUP_ROUNDS, totalRounds),
        slotLabels: {},
        groupIndex: i + 1,
        groupTotal: groupCount,
      })
    })

    // صفحه مرحله نهایی: از دور بعد از گروه‌ها تا فینال
    allPages.push({
      ...basePage,
      key: `${category}-final`,
      displaySize: groupCount,
      sideMatchCount: getSideMatchCount(finalStageData),
      isSplitted: true,
      partTitle:
          groupCount === 2
              ? 'مسابقه فینال و قهرمانی'
              : `مرحله نهایی — ${toFaNumber(groupCount)} نفر آخر` +
              ` (${entryRoundLabel} تا فینال)`,
      showFinal: true,
      data: finalStageData,
      podium,
      roundOffset: GROUP_ROUNDS,
      finalLabel: 'فینال',
      slotLabels,
      groupTotal: groupCount,
    })
  })

  return selectedCategory.value === 'all'
      ? allPages
      : allPages.filter(p => p.cat === selectedCategory.value)
})

function roundLabel(
    round: number,
    totalRounds: number
): string {
  const distanceFromFinal = totalRounds - round

  if (distanceFromFinal === 0) {
    return 'فینال'
  }

  if (distanceFromFinal === 1) {
    return 'نیمه‌نهایی'
  }

  if (distanceFromFinal === 2) {
    return 'یک‌چهارم نهایی'
  }

  if (distanceFromFinal === 3) {
    return 'یک‌هشتم نهایی'
  }

  if (distanceFromFinal === 4) {
    return 'یک‌شانزدهم نهایی'
  }

  return `دور ${toFaNumber(round)}`
}

const MatchBox = defineComponent({
  name: 'MatchBox',

  props: {
    match: {
      type: Object as PropType<MatchLike>,
      required: true,
    },

    athletes: {
      type: Array as PropType<AthleteLike[]>,
      required: true,
    },

    isFinal: {
      type: Boolean,
      default: false,
    },
    showMeta: {
      type: Boolean,
      default: true,
    },
    placeholder1: {
      type: String as PropType<string | undefined>,
      default: undefined,
    },
    placeholder2: {
      type: String as PropType<string | undefined>,
      default: undefined,
    },
  },

  setup(props) {
    return () => {
      const match = props.match
      const byeMatch = isTrueBye(match)

      const findAthlete = (
          id: string | null | undefined
      ): { name: string; club: string; coach?: string } => {
        if (!id) {
          return {
            name: '-',
            club: '',
          }
        }

        const athlete = props.athletes.find(
            item => item.id === id
        )

        return {
          name:
              athlete?.name ??
              athlete?.fullName ??
              '-',
          club:
              athlete?.club ??
              '',
          coach: athlete?.coach?.trim() ?? '',
        }
      }

      const athlete1 = findAthlete(match.athlete1Id)
      const athlete2 = findAthlete(match.athlete2Id)

      const athlete1IsWinner = Boolean(
          match.winnerId &&
          match.winnerId === match.athlete1Id
      )

      const athlete2IsWinner = Boolean(
          match.winnerId &&
          match.winnerId === match.athlete2Id
      )

      // اسلات خالی هیچ متنی ندارد؛ نه «استراحت» و نه «منتظر برنده»
      const slotText = (id: string | null, placeholder?: string): string => {
        if (id) return findAthlete(id).name
        return placeholder ?? ''
      }
      const slotClass = (
          isWinner: boolean,
          isEmpty: boolean,
          color: 'blue' | 'red'
      ): string => {
        const base = [
          'athlete-slot',
          'flex min-w-0 items-center gap-1.5',
          'rounded-md border px-2 py-1',
          'text-[11px] leading-tight',
          'flex-1',
        ].join(' ')

        if (isEmpty) {
          return [
            base,
            'empty-slot',
            'border-dashed',
            'border-slate-200',
            'bg-slate-50/60',
          ].join(' ')
        }

        if (isWinner && byeMatch) {
          return [
            base,
            'bye-win-slot',
            'border-slate-300',
            'bg-white',
            'font-bold',
            'text-slate-700',
          ].join(' ')
        }

        if (isWinner) {
          return [
            base,
            'winner-slot',
            'border-emerald-400',
            'bg-emerald-50',
            'font-black',
            'text-emerald-900',
          ].join(' ')
        }

        if (color === 'blue') {
          return [
            base,
            'border-blue-200',
            'bg-blue-50/70',
            'text-slate-800',
          ].join(' ')
        }

        return [
          base,
          'border-rose-200',
          'bg-rose-50/70',
          'text-slate-800',
        ].join(' ')
      }

      const colorMarker = (color: 'blue' | 'red') => {
        return h('span', {
          class: [
            'athlete-color-marker',
            color === 'blue'
                ? 'bg-blue-500'
                : 'bg-rose-500',
          ].join(' '),
        })
      }

      const renderSlot = (
          info: { name: string; club: string; coach?: string },
          id: string | null,
          isWinner: boolean,
          color: 'blue' | 'red',
          placeholder?: string
      ) => {
        const isEmpty = !id
        const byeWin = isWinner && byeMatch
        const affiliation = info.club

        return h(
            'div',
            { class: slotClass(isWinner, isEmpty, color) },
            [
              !isEmpty ? colorMarker(color) : null,

              h('div', { class: 'athlete-details min-w-0 flex-1' }, [
                h(
                    'div',
                    {
                      class: isEmpty
                          ? 'truncate text-[10px] font-bold opacity-60'
                          : 'athlete-name truncate font-bold',
                    },
                    slotText(id, placeholder)
                ),

                !isEmpty && affiliation
                    ? h(
                        'div',
                        { class: 'athlete-affiliation truncate', title: affiliation },
                        affiliation
                    )
                    : null,
              ]),

              isWinner && !byeWin
                  ? h(
                      'span',
                      {
                        class:
                            'winner-star mr-auto flex-shrink-0 text-[10px] font-black',
                      },
                      '★'
                  )
                  : null,
            ]
        )
      }


      return h(
          'article',
          {
            class: [
              'match-card',
              'relative rounded-lg border bg-white p-1.5 flex flex-col',
              props.isFinal
                  ? 'is-final-match border-amber-400'
                  : byeMatch
                      ? 'is-bye-win border-slate-200'
                      : match.winnerId
                          ? 'is-finished border-emerald-300'
                          : 'border-slate-300',
            ].join(' '),
            dir: 'rtl',
          },
          [
            h(
                'div',
                { class: 'match-meta' },
                [
                  props.showMeta
                      ? h('div', { class: byeMatch ? 'match-tag is-bye-tag' : 'match-tag' }, matchTag(match))
                      : null
                ]
            ),

            renderSlot(athlete1, match.athlete1Id, athlete1IsWinner, 'blue', props.placeholder1),


            h('div', { class: 'my-0.5 border-t border-slate-100' }),


            renderSlot(athlete2, match.athlete2Id, athlete2IsWinner, 'red', props.placeholder2),

          ]
      )
    }
  },
})

const weightLabels = ['اول', 'دوم', 'سوم', 'چهارم', 'پنجم', 'ششم', 'هفتم', 'هشتم', 'نهم', 'دهم']

function getWeightOrder(category: string): string {
  const index = officialCategories.value.indexOf(category)

  if (index === -1) {
    return ''
  }

  return weightLabels[index] ?? toFaNumber(index + 1)
}


function getWeightLabel(category: string): string {
  const order = getWeightOrder(category)

  if (!order) {
    return `وزن (${category})`
  }

  return `وزن ${order} (${category})`
}


async function print(): Promise<void> {
  await nextTick()

  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      window.print()
    })
  })
}

async function printCategory(
    category: string
): Promise<void> {
  const previousCategory = selectedCategory.value

  selectedCategory.value = category
  await nextTick()

  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      window.print()
    })
  })

  const restoreCategory = () => {
    selectedCategory.value = previousCategory

    window.removeEventListener(
        'afterprint',
        restoreCategory
    )
  }

  window.addEventListener(
      'afterprint',
      restoreCategory
  )
}

// A4 landscape با حاشیه 4mm => ‌فضای قابل چاپ ≈ 289mm × 202mm
const PAGE_W = (289 / 25.4) * 94;
const PAGE_H = (202 / 25.4) * 94;

// ضریب اطمینان: جلوی سرریز ناشی از گرد شدن zoom در رندر چاپ را می‌گیرد
const SAFETY = 0.95;

function fitBracketsToPage() {
  document.querySelectorAll<HTMLElement>('.category-page').forEach((page) => {
    const stage = page.querySelector<HTMLElement>('.bracket-stage');
    if (!stage) return;

    stage.style.removeProperty('--print-scale');

    const header = page.querySelector<HTMLElement>('.pdf-header');
    const headerH = header ? header.offsetHeight : 0;

    const scale = Math.min(
        (PAGE_W / stage.scrollWidth) * SAFETY,
        ((PAGE_H - headerH) / stage.scrollHeight) * SAFETY,
        1.6,
    );

    stage.style.setProperty('--print-scale', String(scale));
  });
}

onMounted(() => window.addEventListener('beforeprint', fitBracketsToPage));
onUnmounted(() => window.removeEventListener('beforeprint', fitBracketsToPage));

</script>

<style>
/* =========================================================
   Page header
   ========================================================= */

/* =========================================================
   PDF Header - New centered layout
   ========================================================= */

.pdf-header {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 12px;
  padding: 12px 18px;
  background:
      radial-gradient(circle at 20% 10%, rgba(59,130,246,.28), transparent 30%),
      linear-gradient(120deg, #020617 0%, #172554 52%, #312e81 100%);
  border-bottom: 1px solid #1e3a8a;
}

.header-side {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: nowrap;
  min-width: 0;
}

.header-side-start { justify-content: flex-start; }
.header-side-end   { justify-content: flex-end; }

.header-center {
  text-align: center;
  min-width: 0;
}

.header-title {
  font-size: 17px;
  font-weight: 900;
  color: #fff;
  line-height: 1.25;
  white-space: nowrap;
}

.header-part { color: #fbbf24; }

.header-org {
  margin-top: 2px;
  font-size: 11px;
  font-weight: 800;
  color: #fbbf24;
  white-space: nowrap;
}

.header-date {
  border: 1px solid rgba(96,165,250,.4);
  background: rgba(59,130,246,.2);
  color: #dbeafe;
}

.header-print-btn {
  border-radius: 999px;
  border: 1px solid rgba(255,255,255,.2);
  background: rgba(255,255,255,.1);
  padding: 6px 12px;
  font-size: 11px;
  font-weight: 700;
  color: #fff;
}

@media print {
  .pdf-header { padding: 10px 14px; }
  .header-title { font-size: 15px; }
  .header-org { font-size: 10px; }
  .header-pill { font-size: 10px !important; padding: 4px 10px !important; }
}

.header-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  padding: 6px 12px;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 900;
}

.header-index {
  border: 1px solid rgba(255, 255, 255, 0.25);
  background: rgba(255, 255, 255, 0.18);
  color: #ffffff;
}

.header-weight {
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.13);
  color: #ffffff;
}

.header-court {
  border: 1px solid #fbbf24;
  background: #fbbf24;
  color: #172033;
}

.header-count {
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.08);
  color: #e2e8f0;
}

/* =========================================================
   Main bracket
   ========================================================= */

.bracket-scroll-container {
  overflow-x: auto;
  overflow-y: visible;
  background: #ffffff;
}

.bracket-stage {
  --column-width: 190px;
  --column-gap: 26px;
  --connector-size: calc(var(--column-gap) / 2);
  --row-height: 118px;
  --row-gap: 12px;
  --final-column-width: 210px;
  --card-height: 100px;
  --slot-height: 30px;
  --meta-height: 18px;
  --print-scale: 1;

  --track-height: calc(
      var(--side-matches) * var(--row-height) +
      (var(--side-matches) - 1) * var(--row-gap)
  );

  display: grid;
  grid-template-columns:
    minmax(0, 1fr)
    var(--final-column-width)
    minmax(0, 1fr);

  align-items: start;
  column-gap: var(--column-gap);
  width: max-content;
  min-width: 100%;
  padding: 18px 22px 180px;
  background:
      radial-gradient(
          circle at 50% 50%,
          rgba(245, 158, 11, 0.08),
          transparent 22%
      ),
      linear-gradient(
          180deg,
          #ffffff 0%,
          #f8fafc 100%
      );
  direction: ltr;
  overflow: visible;
}

.bracket-stage .match-card,
.bracket-stage .round-heading,
.bracket-stage .final-badge,
.bracket-stage .champion-card,
.bracket-stage .podium-card,
.bracket-stage .empty-final-card {
  direction: rtl;
}

/* =========================================================
   Bracket sides
   ========================================================= */

.bracket-side {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: var(--column-gap);
}

.left-flow {
  justify-content: flex-end;
}

.right-flow {
  justify-content: flex-start;
}

.round-column {
  display: flex;
  width: var(--column-width);
  min-width: var(--column-width);
  flex: 0 0 var(--column-width);
  flex-direction: column;
}

.round-heading {
  position: relative;
  z-index: 5;
  display: flex;
  height: 42px;
  min-height: 42px;
  max-height: 42px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  margin-bottom: 10px;
  border: 1px solid #dbe4ef;
  border-radius: 10px;
  background:
      linear-gradient(
          180deg,
          #ffffff 0%,
          #f1f5f9 100%
      );
  box-shadow:
      0 1px 2px rgba(15, 23, 42, 0.06),
      inset 0 1px 0 rgba(255, 255, 255, 0.9);
  color: #334155;
  text-align: center;
}

.round-heading span {
  font-size: 11px;
  font-weight: 950;
}

.round-heading small {
  margin-top: 1px;
  color: #94a3b8;
  font-size: 8px;
  font-weight: 700;
}

.round-grid {
  display: grid;
  grid-auto-flow: row dense;
  grid-template-rows: repeat(var(--side-matches), var(--row-height));
  row-gap: var(--row-gap);
  min-height: var(--track-height);
  align-items: start;
}

.match-node {
  position: relative;
  display: flex;
  min-width: 0;
  min-height: 0;
  align-items: center;
  align-self: stretch;
  justify-content: center;
  overflow: visible;
}

.match-node .bracket-match-card {
  position: relative;
  z-index: 3;
  display: block;
  width: 100%;
  height: var(--card-height);
  min-height: var(--card-height);
  max-height: var(--card-height);
}

/* =========================================================
   Connectors - Left
   ========================================================= */

.left-flow .match-node::after {
  content: "";
  position: absolute;
  z-index: 1;
  top: 50%;
  right: calc(var(--connector-size) * -1);
  width: var(--connector-size);
  height: 2px;
  background: #94a3b8;
  transform: translateY(-50%);
}

.left-flow
.round-column:not(:last-child)
.match-node.pair-top::before {
  content: "";
  position: absolute;
  z-index: 1;
  top: 50%;
  right: calc(var(--connector-size) * -1);
  width: 2px;
  height: calc(100% + var(--row-gap));
  background: #94a3b8;
}

.left-flow
.round-column:not(:last-child)
.match-node.pair-bottom::before {
  content: "";
  position: absolute;
  z-index: 1;
  bottom: 50%;
  right: calc(var(--connector-size) * -1);
  width: 2px;
  height: calc(100% + var(--row-gap));
  background: #94a3b8;
}

.left-flow
.round-column:not(:first-child)
.match-node
.bracket-match-card::before {
  content: "";
  position: absolute;
  z-index: 1;
  top: 50%;
  left: calc(var(--connector-size) * -1);
  width: var(--connector-size);
  height: 2px;
  background: #94a3b8;
  transform: translateY(-50%);
}

/* =========================================================
   Connectors - Right
   ========================================================= */

.right-flow .match-node::after {
  content: "";
  position: absolute;
  z-index: 1;
  top: 50%;
  left: calc(var(--connector-size) * -1);
  width: var(--connector-size);
  height: 2px;
  background: #94a3b8;
  transform: translateY(-50%);
}

.right-flow
.round-column:not(:first-child)
.match-node.pair-top::before {
  content: "";
  position: absolute;
  z-index: 1;
  top: 50%;
  left: calc(var(--connector-size) * -1);
  width: 2px;
  height: calc(100% + var(--row-gap));
  background: #94a3b8;
}

.right-flow
.round-column:not(:first-child)
.match-node.pair-bottom::before {
  content: "";
  position: absolute;
  z-index: 1;
  left: calc(var(--connector-size) * -1);
  bottom: 50%;
  width: 2px;
  height: calc(100% + var(--row-gap));
  background: #94a3b8;
}

.right-flow
.round-column:not(:last-child)
.match-node
.bracket-match-card::after {
  content: "";
  position: absolute;
  z-index: 1;
  top: 50%;
  right: calc(var(--connector-size) * -1);
  width: var(--connector-size);
  height: 2px;
  background: #94a3b8;
  transform: translateY(-50%);
}

/* =========================================================
   Match card
   ========================================================= */

.match-card {
  position: relative;
  display: flex;
  height: var(--card-height);
  min-height: var(--card-height);
  max-height: var(--card-height);
  flex-direction: column;
  justify-content: flex-start;
  box-sizing: border-box;
  overflow: hidden;
  border-color: #cbd5e1;
  background: #ffffff;
  box-shadow:
      0 2px 5px rgba(15, 23, 42, 0.08),
      0 1px 1px rgba(15, 23, 42, 0.05);
  transition:
      border-color 150ms ease,
      box-shadow 150ms ease,
      transform 150ms ease;
}

.match-card:hover {
  border-color: #94a3b8;
  box-shadow:
      0 5px 12px rgba(15, 23, 42, 0.12),
      0 2px 3px rgba(15, 23, 42, 0.06);
}

.match-card.is-finished {
  border-color: #86efac;
  background:
      linear-gradient(
          180deg,
          #ffffff 0%,
          #f0fdf4 100%
      );
}

/* برد با استراحت؛ عمداً از برد واقعی کم‌رنگ‌تر است */
.match-card.is-bye-win {
  background: #fcfdfe;
  box-shadow: none;
}

.match-meta {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  min-height: var(--meta-height);
  margin-bottom: 4px;
}

.match-tag {
  display: inline-flex;
  min-height: 8px;
  align-items: center;
  justify-content: center;
  border: 1px solid #1e293b;
  border-radius: 4px;
  background: #0f172a;
  padding: 2px 6px;
  white-space: nowrap;
  color: #ffffff;
  font-size: 8px;
  font-weight: 950;
  line-height: 1;
  letter-spacing: 0.03em;
}

.match-tag.is-bye-tag {
  border-color: #e2e8f0;
  background: #f8fafc;
  color: #94a3b8;
}

.athlete-slot {
  display: flex;
  align-items: center;
  min-height: var(--slot-height);
}

.athlete-affiliation {
  margin-top: 2px;
  font-size: 11px;
  font-weight: 600;
  line-height: 1.35;
  color: #334155;
}

.empty-slot {
  min-height: var(--slot-height);
}

.athlete-color-marker {
  display: inline-block;
  width: 8px;
  height: 8px;
  flex: 0 0 8px;
  border-radius: 999px;
}

.winner-slot {
  position: relative;
  box-shadow:
      inset 3px 0 0 #22c55e,
      0 1px 2px rgba(34, 197, 94, 0.08);
}

.bye-win-slot {
  box-shadow: inset 3px 0 0 #cbd5e1;
}

.winner-star {
  color: #059669;
}

/* =========================================================
   Final column
   ========================================================= */

.final-column {
  display: flex;
  width: var(--final-column-width);
  min-width: var(--final-column-width);
  flex-direction: column;
}

.final-heading-spacer {
  visibility: hidden;
}

.final-track {
  position: relative;
  display: grid;
  min-height: var(--track-height);
  align-items: center;
}

.final-match-anchor {
  position: relative;
  z-index: 6;
  width: 100%;
  align-self: center;
}

.final-match-anchor::before {
  content: "";
  position: absolute;
  z-index: 1;
  top: 50%;
  left: calc(var(--connector-size) * -1);
  width: var(--connector-size);
  height: 2px;
  background:
      linear-gradient(
          90deg,
          #94a3b8 0%,
          #f59e0b 100%
      );
  transform: translateY(-50%);
}

.final-match-anchor::after {
  content: "";
  position: absolute;
  z-index: 1;
  top: 50%;
  right: calc(var(--connector-size) * -1);
  width: var(--connector-size);
  height: 2px;
  background:
      linear-gradient(
          90deg,
          #f59e0b 0%,
          #94a3b8 100%
      );
  transform: translateY(-50%);
}

.final-match-anchor.is-placeholder::before,
.final-match-anchor.is-placeholder::after {
  background: #cbd5e1;
}

.final-badge {
  position: absolute;
  right: 0;
  bottom: calc(100% + 10px);
  left: 0;
  display: flex;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  gap: 7px;
  border: 1px solid #f59e0b;
  border-radius: 12px;
  background:
      linear-gradient(
          135deg,
          #fffbeb 0%,
          #fef3c7 100%
      );
  box-shadow:
      0 4px 12px rgba(245, 158, 11, 0.15);
  color: #92400e;
}

.final-badge-icon {
  font-size: 21px;
}

.final-badge > div {
  display: flex;
  flex-direction: column;
}

.final-badge strong {
  font-size: 12px;
  font-weight: 950;
}

.final-badge small {
  margin-top: 1px;
  color: #b45309;
  font-size: 8px;
  font-weight: 700;
}

.final-match-card {
  position: relative;
  z-index: 3;
  border: 2px solid #f59e0b !important;
  background:
      linear-gradient(
          180deg,
          #ffffff 0%,
          #fffbeb 100%
      ) !important;
  box-shadow:
      0 7px 20px rgba(245, 158, 11, 0.2),
      0 2px 5px rgba(15, 23, 42, 0.1) !important;
}

.empty-final-card {
  display: flex;
  height: var(--card-height);
  min-height: var(--card-height);
  max-height: var(--card-height);
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 5px;
  border: 2px dashed #fbbf24;
  border-radius: 10px;
  background: #fffbeb;
  color: #92400e;
  text-align: center;
  box-sizing: border-box;
}

.empty-final-card span {
  font-size: 20px;
}

.empty-final-card strong {
  font-size: 9px;
  font-weight: 900;
}

/* =========================================================
   Champion
   ========================================================= */

.champion-card {
  position: absolute;
  top: calc(100% + 10px);
  right: 0;
  left: 0;
  display: flex;
  min-height: 58px;
  align-items: center;
  gap: 8px;
  padding: 7px 9px;
  border: 1px solid #10b981;
  border-radius: 12px;
  background:
      linear-gradient(
          135deg,
          #ecfdf5 0%,
          #d1fae5 100%
      );
  box-shadow:
      0 5px 15px rgba(16, 185, 129, 0.14);
}

.champion-crown {
  display: flex;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background:
      linear-gradient(
          135deg,
          #fbbf24 0%,
          #f59e0b 100%
      );
  box-shadow:
      0 3px 8px rgba(245, 158, 11, 0.25);
  color: #ffffff;
  font-size: 14px;
}

.champion-content {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.champion-label {
  color: #059669;
  font-size: 8px;
  font-weight: 900;
}

.champion-content strong {
  overflow: hidden;
  margin-top: 1px;
  color: #065f46;
  font-size: 11px;
  font-weight: 950;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.champion-content small {
  overflow: hidden;
  color: #047857;
  font-size: 8px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* =========================================================
   Podium - رتبه ۲ و ۳ مشترک
   ========================================================= */

.podium-card {
  position: absolute;
  top: calc(100% + 78px);
  right: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  background: #ffffff;
  box-shadow: 0 3px 10px rgba(15, 23, 42, 0.06);
}

.podium-title {
  margin-bottom: 1px;
  color: #94a3b8;
  font-size: 8px;
  font-weight: 950;
}

.podium-row {
  display: flex;
  align-items: center;
  gap: 7px;
  border-radius: 8px;
  padding: 4px 6px;
  background: #f8fafc;
}

.podium-rank {
  display: flex;
  width: 18px;
  height: 18px;
  flex: 0 0 18px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: #ffffff;
  font-size: 9px;
  font-weight: 950;
}

.podium-rank-1 .podium-rank {
  background: #f59e0b;
}

.podium-rank-2 .podium-rank {
  background: #94a3b8;
}

.podium-rank-3 .podium-rank {
  background: #b45309;
}

.podium-info {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.podium-info strong {
  overflow: hidden;
  color: #1e293b;
  font-size: 10px;
  font-weight: 900;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.podium-info small {
  overflow: hidden;
  color: #94a3b8;
  font-size: 8px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* =========================================================
   Dynamic bracket sizes
   ========================================================= */

.bracket-stage.size-2 {
  grid-template-columns: 1fr !important;
  justify-items: center;
  align-content: center;
  min-height: 460px;
}

.bracket-stage.size-2 .bracket-side {
  display: none;
}

.bracket-stage.size-2 .final-column {
  width: 350px;
  --final-column-width: 350px;
}

.bracket-stage.size-4 {
  --column-width: 196px;
  --final-column-width: 214px;
  --row-height: 126px;
  --row-gap: 14px;
  --card-height: 120px;
  --slot-height: 32px;
  --print-scale: 1;
}

.bracket-stage.size-8 {
  --column-width: 188px;
  --final-column-width: 208px;
  --row-height: 122px;
  --row-gap: 13px;
  --card-height: 120px;
  --slot-height: 31px;
  --print-scale: 0.94;
}

.bracket-stage.size-16 {
  --column-width: 178px;
  --column-gap: 22px;
  --final-column-width: 198px;
  --row-height: 114px;
  --row-gap: 11px;
  --card-height: 120px;
  --slot-height: 28px;
  --print-scale: 0.75;
}

.bracket-stage.size-32 {
  --column-width: 162px;
  --column-gap: 18px;
  --final-column-width: 184px;
  --row-height: 106px;
  --row-gap: 9px;
  --card-height: 100px;
  --slot-height: 26px;
  --meta-height: 16px;
  --print-scale: 0.62;
}

.bracket-stage.size-32 .match-card {
  height: var(--card-height) !important;
  max-height: var(--card-height) !important;
  overflow: hidden !important;
}

.bracket-stage.size-32 .athlete-slot,
.bracket-stage.size-32 .athlete-slot * {
  white-space: nowrap !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
}

.bracket-stage.size-32 .round-heading {
  height: 38px;
  min-height: 38px;
  max-height: 38px;
  margin-bottom: 8px;
}

.bracket-stage.size-32 .round-heading span {
  font-size: 9px;
}

.bracket-stage.size-32 .round-heading small {
  font-size: 7px;
}

.bracket-stage.size-64 {
  --column-width: 142px;
  --column-gap: 14px;
  --final-column-width: 168px;
  --row-height: 92px;
  --row-gap: 7px;
  --card-height: 78px;
  --slot-height: 22px;
  --meta-height: 14px;
  --print-scale: 0.46;
}

.bracket-stage.size-64 .match-card {
  padding: 4px;
}

.bracket-stage.size-64 .match-tag {
  padding: 2px 4px;
  font-size: 7px;
}

.bracket-stage.size-64 .athlete-slot {
  padding: 3px 5px;
  font-size: 9px;
}

.bracket-stage.size-64 .round-heading {
  height: 34px;
  min-height: 34px;
  max-height: 34px;
  margin-bottom: 7px;
}

.bracket-stage.size-64 .round-heading span {
  font-size: 8px;
}

.bracket-stage.size-64 .round-heading small {
  font-size: 6px;
}

.bracket-stage.size-64 .final-badge strong {
  font-size: 11px;
}

.bracket-stage.size-64 .champion-content strong {
  font-size: 10px;
}

/* =========================================================
   Responsive preview
   ========================================================= */

@media screen and (max-width: 1200px) {
  .category-page {
    overflow: hidden;
  }

  .bracket-scroll-container {
    overflow-x: auto;
  }
}
/* =========================================================
   Print — سبک خطی (بدون کادر)
   ========================================================= */

@media print {
  .no-print {
    display: none !important;
  }

  html,
  body {
    width: 100% !important;
    margin: 0 !important;
    padding: 0 !important;
    background: #ffffff !important;
  }

  body * {
    visibility: hidden;
  }

  #bracket-print-root,
  #bracket-print-root * {
    visibility: visible;
  }

  #bracket-print-root {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    min-height: 0;
    padding: 0 !important;
    background: #ffffff !important;
  }

  /* ---------- هدر ---------- */
  .pdf-header {
    background: #ffffff !important;
    border: 0 !important;
    border-bottom: 1.5px solid #000 !important;
    padding: 6px 8px !important;
  }

  .header-title,
  .header-org,
  .header-part {
    color: #000 !important;
  }

  .header-title { font-size: 14pt !important; }
  .header-org { font-size: 9pt !important; }

  .header-pill {
    background: #fff !important;
    color: #000 !important;
    border: 1px solid #000 !important;
    border-radius: 4px !important;
    font-size: 9pt !important;
    padding: 3px 8px !important;
  }

  .header-court {
    background: #fff !important;
    border: 1.5px solid #000 !important;
  }

  /* ---------- صفحه‌بندی ---------- */
  .category-page {
    overflow: visible !important;
    margin: 0 !important;
    border: 0 !important;
    border-radius: 0 !important;
    box-shadow: none !important;
    break-after: page;
    break-inside: avoid !important;
    page-break-after: always;
    page-break-inside: avoid !important;
  }

  .category-page:last-child {
    break-after: auto;
    page-break-after: auto;
  }

  .bracket-scroll-container {
    overflow: visible !important;
    padding: 0 !important;
    background: #ffffff !important;
  }

  /* ---------- استیج ----------
     ارتفاع کارت از روی اسلات‌ها حساب می‌شود:
     ۲ اسلات + ۲۲px فضای شماره مسابقه.
     پس اسلات دوم دیگر هیچ‌وقت بیرون کارت نمی‌افتد. */
  .bracket-stage {
    overflow: visible !important;
    width: max-content !important;
    min-width: 0 !important;
    margin: 0 auto !important;
    padding: 6px 4px 40px !important;
    background: #ffffff !important;
    --slot-height: 56px !important;
    --card-height: calc(var(--slot-height) * 2 + 22px) !important;
    --row-height: calc(var(--card-height) + 12px) !important;
    --row-gap: 12px !important;
    /* ضریب اطمینان روی مقیاسِ JS تا گِرد شدن zoom باعث پرش به صفحه بعد نشود */
    zoom: calc(var(--print-scale, var(--print-scale-fallback, 1)) * var(--print-safety, 1));
  }

  .bracket-stage.size-32 {
    --slot-height: 50px !important;
    --row-gap: 8px !important;
    --print-safety: 0.96;
    --print-scale-fallback: 0.34;
  }

  .bracket-stage.size-64 {
    --slot-height: 50px !important;
    --row-gap: 6px !important;
    --print-safety: 0.96;
    --print-scale-fallback: 0.30;
  }

  /* همین هندسه هنگام اندازه‌گیری JS هم فعال باشد */
  body.printing .bracket-stage {
    --slot-height: 54px !important;
    --card-height: calc(var(--slot-height) * 2 + 22px) !important;
    --row-height: calc(var(--card-height) + 12px) !important;
    --row-gap: 12px !important;
    padding: 6px 4px 40px !important;
  }
  body.printing .bracket-stage.size-32 {
    --slot-height: 60px !important;
    --row-gap: 8px !important;
  }
  body.printing .bracket-stage.size-64 {
    --slot-height: 60px !important;
    --row-gap: 6px !important;
  }

  /* ---------- کارت مسابقه: بدون کادر، فقط فضا ---------- */
  .bracket-stage .bracket-match-card {
    width: 400px !important;
  }

  .bracket-stage .match-card,
  .bracket-stage .final-match-card {
    height: var(--card-height) !important;
    min-height: 0 !important;
    max-height: none !important;
    overflow: visible !important;
    display: flex !important;
    flex-direction: column !important;
    justify-content: flex-end !important;
    border: 0 !important;
    border-radius: 0 !important;
    background: transparent !important;
    box-shadow: none !important;
    padding: 0 4px !important;
  }

  /* ---------- اسلات بازیکن: اسم روی خط ---------- */
  .bracket-stage .athlete-slot {
    height: var(--slot-height) !important;
    min-height: 0 !important;
    max-height: var(--slot-height) !important;
    display: flex !important;
    flex-direction: column !important;
    justify-content: flex-end !important;
    font-size: 12pt !important;
    line-height: 1.15 !important;
    color: #000 !important;
    border-bottom: 1px solid #000 !important;
    padding: 0 4px 2px !important;
    overflow: visible !important;  /* برش عمودی نده */
    text-align: center !important;
  }

  .bracket-stage.size-32 .athlete-slot { font-size: 11.5pt !important; }
  .bracket-stage.size-64 .athlete-slot { font-size: 10.5pt !important; }

  /* nowrap/ellipsis فقط روی متنِ داخل اسلات، نه خود اسلات —
     تا فقط از پهنا کوتاه شود، نه از ارتفاع */
  .bracket-stage .athlete-slot > * {
    white-space: nowrap !important;
    overflow: hidden !important;
    text-overflow: ellipsis !important;
    max-width: 100% !important;
  }

  .bracket-stage .athlete-details {
    flex: 0 0 auto !important;
    width: 100% !important;
    min-width: 0 !important;
  }

  .bracket-stage .athlete-name {
    line-height: 1.3 !important;
  }

  .bracket-stage .athlete-affiliation {
    margin-top: 2px !important;
    display: block !important;
    font-size: 10pt !important;
    font-weight: 600 !important;
    line-height: 1.3 !important;
    color: #222 !important;
    opacity: 1 !important;
    text-align: center !important;
    width: 100% !important;
  }

  .bracket-stage .athlete-slot:last-child {
    border-bottom: 1.5px solid #000 !important;
  }

  .bracket-stage .empty-slot {
    height: var(--slot-height) !important;
    min-height: 0 !important;
    border-bottom: 1.5px solid #000 !important;
  }

  /* ---------- برنده: فقط بولد + خط ضخیم‌تر ---------- */
  .bracket-stage .winner-slot {
    box-shadow: none !important;
    background: transparent !important;
    font-weight: 900 !important;
    border-bottom-width: 2.5px !important;
  }

  .bracket-stage .bye-win-slot {
    box-shadow: none !important;
    background: transparent !important;
    border-bottom-color: #999 !important;
  }

  .bracket-stage .winner-star { color: #000 !important; }

  .bracket-stage .athlete-color-marker {
    border: 1px solid #000 !important;
  }

  /* ---------- شماره مسابقه: عدد ساده کنار خط ---------- */
  .bracket-stage .match-tag {
    background: transparent !important;
    color: #000 !important;
    border: 0 !important;
    border-radius: 0 !important;
    font-size: 16pt !important;
    font-weight: 900 !important;
    line-height: 1 !important;
    padding: 0 4px !important;
    letter-spacing: 0 !important;
  }

  .bracket-stage .match-tag.is-bye-tag {
    color: #888 !important;
  }

  .bracket-stage .match-card.is-finished,
  .bracket-stage .match-card.is-bye-win {
    background: transparent !important;
    box-shadow: none !important;
    border: 0 !important;
  }

  /* ---------- خطوط اتصال ---------- */
  .bracket-stage .match-node::before,
  .bracket-stage .match-node::after,
  .bracket-stage .bracket-match-card::before,
  .bracket-stage .bracket-match-card::after,
  .bracket-stage .final-match-anchor::before,
  .bracket-stage .final-match-anchor::after {
    background: #000 !important;
  }

  /* ---------- سرستون دورها ---------- */
  .round-heading {
    background: transparent !important;
    border: 0 !important;
    box-shadow: none !important;
    color: #000 !important;
  }

  /* ---------- فینال ---------- */
  .bracket-stage .final-badge {
    background: #fff !important;
    border: 0 !important;
    border-bottom: 1.5px solid #000 !important;
    border-radius: 0 !important;
    box-shadow: none !important;
    min-height: 0 !important;
    color: #000 !important;
    padding-bottom: 3px !important;
  }

  .bracket-stage .final-badge-icon { display: none !important; }

  .bracket-stage .final-badge strong {
    font-size: 11pt !important;
    color: #000 !important;
  }

  .bracket-stage .final-badge small {
    font-size: 8pt !important;
    color: #333 !important;
  }

  .bracket-stage .final-match-card {
    border: 0 !important;
    background: transparent !important;
    box-shadow: none !important;
  }

  .bracket-stage .final-match-card .athlete-slot {
    border-bottom-width: 2px !important;
    font-size: 12pt !important;
  }

  .bracket-stage .empty-final-card {
    border: 0 !important;
    border-bottom: 1.5px dashed #000 !important;
    border-radius: 0 !important;
    background: transparent !important;
    color: #000 !important;
    box-shadow: none !important;
  }

  .bracket-stage .empty-final-card span { display: none !important; }
  .bracket-stage .empty-final-card strong { font-size: 10.5pt !important; }

  /* ---------- قهرمان ---------- */
  .bracket-stage .champion-card {
    background: transparent !important;
    border: 0 !important;
    border-top: 1.5px solid #000 !important;
    border-radius: 0 !important;
    box-shadow: none !important;
    padding: 6px 2px !important;
  }

  .bracket-stage .champion-crown {
    background: #fff !important;
    border: 1.5px solid #000 !important;
    color: #000 !important;
    box-shadow: none !important;
  }

  .bracket-stage .champion-label { color: #000 !important; font-size: 8pt !important; }
  .bracket-stage .champion-content strong { color: #000 !important; font-size: 12pt !important; }
  .bracket-stage .champion-content small { color: #333 !important; font-size: 8pt !important; }

  /* ---------- رده‌بندی نهایی ---------- */
  .bracket-stage .podium-card {
    background: transparent !important;
    border: 0 !important;
    border-top: 1px solid #000 !important;
    border-radius: 0 !important;
    box-shadow: none !important;
    padding: 6px 0 0 !important;
    gap: 0 !important;
  }

  .bracket-stage .podium-title {
    color: #000 !important;
    font-size: 8.5pt !important;
    margin-bottom: 4px !important;
  }

  .bracket-stage .podium-row {
    background: transparent !important;
    border-radius: 0 !important;
    border-bottom: 1px solid #ccc !important;
    padding: 4px 2px !important;
    gap: 8px !important;
  }

  .bracket-stage .podium-rank {
    background: #fff !important;
    border: 1.5px solid #000 !important;
    color: #000 !important;
    font-size: 9pt !important;
  }

  .bracket-stage .podium-info strong { color: #000 !important; font-size: 10.5pt !important; }
  .bracket-stage .podium-info small { color: #333 !important; font-size: 8pt !important; }

  /* ---------- پاک‌سازی عمومی ---------- */
  .bracket-stage,
  .bracket-stage * {
    background-image: none !important;
    text-shadow: none !important;
  }

  .match-card,
  .round-heading,
  .final-badge,
  .champion-card,
  .podium-card,
  .final-match-card {
    box-shadow: none !important;
  }



  .match-card {
    break-inside: avoid;
    page-break-inside: avoid;
  }

  * {
    -webkit-print-color-adjust: exact !important;
    print-color-adjust: exact !important;
  }

  @page {
    size: A4 landscape;
    margin: 4mm;
  }

  .bracket-stage .final-column,
  .bracket-stage .final-track,
  .bracket-stage .final-match-anchor {
    width: 200px !important;
    min-width: 0 !important;
    max-width: 400px !important;
    box-sizing: border-box !important;
  }

  .bracket-stage .final-heading-spacer,
  .bracket-stage .final-badge,
  .bracket-stage .final-match-card,
  .bracket-stage .empty-final-card,
  .bracket-stage .champion-card,
  .bracket-stage .podium-card {
    width: 100% !important;
    min-width: 0 !important;
    max-width: 100% !important;
    box-sizing: border-box !important;
  }
}
</style>
