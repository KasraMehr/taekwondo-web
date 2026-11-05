<script setup lang="ts">
import { computed, ref, watch, onUnmounted } from 'vue'
import draggable from 'vuedraggable'
import { useTournamentStore } from '../stores/tournament'
import type { Match, Tournament } from '../types'
import { webApi } from '../webApi'

const store = useTournamentStore()
const t = computed(() => store.currentTournament)
const matches = computed(() => t.value?.matches ?? [])

const activeTab = ref<'categories' | 'schedule' | 'bracket'>('categories')
const sortMode = ref<'weight' | 'count'>('weight')
const numberingPreview = ref<any | null>(null)
const numberingBusy = ref(false)
const numberingError = ref('')
const numberingMessage = ref('')
const numberingReadiness = ref<any | null>(null)
const aiWorking = ref(false)
const aiStep = ref(0)
const aiWizardOpen = ref(false)
const aiWizardError = ref('')
const aiStage = ref(0)
const aiCourtPage = ref(0)
const aiCourtPageSize = 4
const selectedAiCourts = ref<Record<string, number[]>>({})
const aiOptions = ref({matchMinutes: 5, restMinutes: 10, startAt: 1, limitHogu: false, hoguPairs: {'1': 0, '2': 0, '3': 0, '4': 0} as Record<string, number>})
const neededHoguSizes = computed(() => [...new Set(playableWeights.value.map(aiHoguSize))].sort())
const playableWeights = computed(() => [...new Set(matches.value.filter(m => !m.isBye).map(m => m.weightCategory))].sort((a, b) => weightSortValue(a) - weightSortValue(b)))
const aiCourtPageCount = computed(() => Math.max(1, Math.ceil(playableWeights.value.length / aiCourtPageSize)))
const visibleAiWeights = computed(() => playableWeights.value.slice(aiCourtPage.value * aiCourtPageSize, (aiCourtPage.value + 1) * aiCourtPageSize))
const aiHogu = computed<any>(() => (t.value as any)?.settings?.hogu ?? {})
function aiHoguSize(category: string): string {
  const weight = Number(category.replace(/[+-]/g, '')) + (category.startsWith('+') ? 1 : 0)
  const old = weight <= 40 ? '1' : weight <= 55 ? '2' : weight <= 70 ? '3' : '4'
  const age = t.value?.ageCategory
  const proposed = age === 'خردسالان' ? (weight <= 36 ? '1' : weight <= 52 ? '2' : '3')
    : age === 'نونهالان' ? (weight <= 41 ? '1' : weight <= 59 ? '2' : '3')
    : age === 'نوجوانان' ? (weight <= 49 ? '2' : weight <= 68 ? '3' : '4') : old
  const saved = aiHogu.value.categorySizes?.[category]
  return saved && saved !== old ? saved : proposed
}
function courtAllowed(category: string, court: number): boolean {
  return Boolean(court >= 1 && court <= (t.value?.courts ?? 0) && (!aiOptions.value.limitHogu || aiOptions.value.hoguPairs[aiHoguSize(category)] > 0))
}
function openAiWizard() {
  if (!t.value) return
  aiWizardError.value = ''
  aiStage.value = 0
  aiCourtPage.value = 0
  numberingPreview.value = null
  const settings = (t.value as any).settings
  const pairs: Record<string, number> = {'1': 0, '2': 0, '3': 0, '4': 0}
  for (const size of Object.keys(pairs)) pairs[size] = settings?.hogu?.pairCounts?.[size] ?? (settings?.hogu?.enabled ? Object.values(settings.hogu.courtSizes ?? {}).filter((sizes: any) => sizes?.includes(size)).length : 0)
  aiOptions.value = {matchMinutes: settings?.schedule?.matchMinutes || 5, restMinutes: settings?.schedule?.restMinutes ?? 10, startAt: settings?.numbering?.startAt || 1, limitHogu: Boolean(settings?.hogu?.enabled), hoguPairs: pairs}
  const next: Record<string, number[]> = {}
  for (const category of playableWeights.value) {
    const previous = (t.value.courtAssignment?.[category] ?? []).filter(court => courtAllowed(category, court)).slice(0, 2)
    next[category] = previous.length ? previous : Array.from({length: t.value.courts}, (_, i) => i + 1).filter(court => courtAllowed(category, court)).slice(0, 1)
  }
  selectedAiCourts.value = next
  aiWizardOpen.value = true
}
function nextAiStage() {
  aiWizardError.value = ''
  if (aiStage.value === 1 && aiOptions.value.limitHogu) {
    if (Object.values(aiOptions.value.hoguPairs).some(count => !Number.isInteger(count) || count < 0 || count > 100)) {
      aiWizardError.value = 'تعداد هر سایز هوگو باید عددی بین صفر تا ۱۰۰ باشد.'
      return
    }
    if (neededHoguSizes.value.some(size => !Number.isInteger(aiOptions.value.hoguPairs[size]) || aiOptions.value.hoguPairs[size] < 1 || aiOptions.value.hoguPairs[size] > 100)) {
      aiWizardError.value = 'برای همهٔ سایزهای موردنیاز، تعداد جفت هوگوی موجود را وارد کنید.'
      return
    }
    for (const category of playableWeights.value) {
      selectedAiCourts.value[category] = (selectedAiCourts.value[category] ?? []).filter(court => courtAllowed(category, court))
      if (!selectedAiCourts.value[category].length) selectedAiCourts.value[category] = [1]
    }
  }
  aiStage.value++
}
function toggleAiCourt(category: string, court: number) {
  if (!courtAllowed(category, court)) return
  const current = selectedAiCourts.value[category] ?? []
  selectedAiCourts.value = {...selectedAiCourts.value, [category]: current.includes(court) ? current.filter(c => c !== court) : current.length < 2 ? [...current, court] : current}
}
async function confirmAiCourts() {
  if (!Number.isInteger(aiOptions.value.matchMinutes) || aiOptions.value.matchMinutes < 1 || aiOptions.value.matchMinutes > 60 || !Number.isInteger(aiOptions.value.restMinutes) || aiOptions.value.restMinutes < 0 || aiOptions.value.restMinutes > 120 || !Number.isInteger(aiOptions.value.startAt) || aiOptions.value.startAt < 1) {
    aiWizardError.value = 'مدت بازی باید ۱ تا ۶۰ دقیقه، استراحت ۰ تا ۱۲۰ دقیقه و شمارهٔ شروع بزرگ‌تر از صفر باشد.'
    return
  }
  if (playableWeights.value.some(category => !selectedAiCourts.value[category]?.length)) {
    aiWizardError.value = 'برای هر وزن دست‌کم یک زمین انتخاب کنید.'
    return
  }
  aiStage.value = 3
  await previewAiNumbering()
}
let aiTimer: ReturnType<typeof setInterval> | undefined
const aiSteps = ['بررسی قرعه‌ها و وزن‌کشی', 'بررسی ظرفیت زمین‌ها', 'طراحی اولویت وزن‌ها', 'کنترل استراحت و مسیر براکت', 'ساخت پیش‌نمایش نهایی']
const hasManualPositions = computed(() => matches.value.some(match => Boolean(match.movedAt)))
onUnmounted(() => { if (aiTimer) clearInterval(aiTimer) })

async function previewAiNumbering() {
  if (!t.value || aiWorking.value || numberingBusy.value) return
  numberingError.value = ''
  numberingPreview.value = null
  aiStep.value = 0
  aiWorking.value = true
  aiTimer = setInterval(() => { aiStep.value = Math.min(aiStep.value + 1, aiSteps.length - 1) }, 4200)
  try {
    const readiness = await webApi().call<any>(`/tournaments/${t.value.id}/draw-readiness`)
    numberingReadiness.value = readiness
    if (!readiness.ready) throw new Error('ابتدا وزن‌کشی و قرعهٔ همهٔ وزن‌ها را کامل کنید.')
    numberingPreview.value = await webApi().call(`/tournaments/${t.value.id}/numbering/ai/preview`, 'POST', {courts: selectedAiCourts.value, options: aiOptions.value}, {revision: t.value.revision})
    aiStage.value = 4
  } catch (error: any) {
    numberingError.value = error?.message || 'شماره‌گذاری هوشمند ناموفق بود'
    aiWizardError.value = numberingError.value
    aiStage.value = 2
  } finally {
    if (aiTimer) clearInterval(aiTimer)
    aiWorking.value = false
  }
}
const rebalanceBusy = ref(false)
const rebalanceMessage = ref('')
const rebalanceError = ref('')

const canRebalanceCurrentDraw = computed(() =>
  Boolean(t.value?.matches.length) &&
  !matches.value.some(match =>
    !match.isBye && Boolean(match.winnerId || match.result || match.status === 'ongoing')
  )
)

async function rebalanceCurrentDraw() {
  if (!t.value || !canRebalanceCurrentDraw.value || rebalanceBusy.value) return
  const accepted = window.confirm(
    'قرعه و حریف‌ها حفظ می‌شوند، اما تخصیص دستی زمین‌ها و شماره‌های فعلی بازی‌ها دوباره محاسبه خواهند شد. ادامه می‌دهید؟'
  )
  if (!accepted) return

  const tournamentId = t.value.id
  rebalanceBusy.value = true
  rebalanceMessage.value = ''
  rebalanceError.value = ''
  try {
    const updated = await webApi().call<Tournament>(`/tournaments/${tournamentId}/rebalance`, 'POST', undefined, {revision: t.value.revision})
    store.replaceFromServer(updated)
    numberingPreview.value = null
    selectedMatchId.value = null
    rebalanceMessage.value = updated.matches.filter(m => !m.isBye).every(m => m.matchNumber)
        ? 'توزیع زمین‌ها و شماره‌های جدید ذخیره شد.'
        : 'توزیع زمین‌ها ذخیره شد؛ شماره‌گذاری پس از تکمیل وزن‌کشی و قرعهٔ همهٔ وزن‌ها انجام می‌شود.'
  } catch (error: any) {
    rebalanceError.value = error?.message || 'توزیع مجدد بازی‌ها ناموفق بود'
  } finally {
    rebalanceBusy.value = false
  }
}

async function loadNumberingReadiness() {
  if (!t.value) return
  try {
    numberingReadiness.value = await webApi().call(`/tournaments/${t.value.id}/draw-readiness`)
  } catch (error: any) {
    numberingReadiness.value = null
    numberingError.value = error?.message || 'بررسی کامل بودن قرعه ناموفق بود'
  }
}

watch(() => (t.value as any)?.revision, () => { void loadNumberingReadiness() }, { immediate: true })

async function previewNumbering() {
  if (!t.value) return
  numberingBusy.value = true
  numberingError.value = ''
  try {
    const readiness = await webApi().call<any>(`/tournaments/${t.value.id}/draw-readiness`)
    numberingReadiness.value = readiness
    if (!readiness.ready) {
      const pending = [...(readiness.pendingWeighIns ?? []), ...(readiness.undrawnCategories ?? [])]
      numberingError.value = pending.length
          ? `وزن‌کشی یا قرعهٔ این وزن‌ها کامل نیست: ${pending.join('، ')}`
          : 'ابتدا وزن‌کشی و قرعهٔ همهٔ وزن‌ها را کامل کنید.'
      return
    }
    numberingPreview.value = await webApi().call(`/tournaments/${t.value.id}/numbering/preview`, 'POST')
  } catch (error: any) {
    numberingError.value = error?.message || 'پیش‌نمایش شماره‌گذاری ناموفق بود'
  } finally {
    numberingBusy.value = false
  }
}

async function applyNumbering() {
  if (!t.value || numberingBusy.value) return
  numberingBusy.value = true
  numberingError.value = ''
  numberingMessage.value = ''
  try {
    const updated = numberingPreview.value?.priority
      ? await webApi().call<Tournament>(`/tournaments/${t.value.id}/numbering/ai`, 'POST', {priority: numberingPreview.value.priority, courts: numberingPreview.value.selectedCourts, options: numberingPreview.value.selectedOptions}, {revision: numberingPreview.value.revision})
      : await store.applyMatchNumbering(t.value.id, numberingPreview.value?.revision)
    if (numberingPreview.value?.priority) store.replaceFromServer(updated)
    numberingPreview.value = null
    aiWizardOpen.value = false
    numberingMessage.value = `شماره‌گذاری ${updated.matches.filter(m => !m.isBye).length} بازی ذخیره شد.`
  } catch (error: any) {
    numberingError.value = error?.message || 'ثبت شماره‌های بازی ناموفق بود'
  } finally {
    numberingBusy.value = false
  }
}

// ─── ثابت‌ها ───────────────────────────────────────────────────────────────

const COURT_LABELS = ['A', 'B', 'C', 'D', 'E', 'F']
const courtLabel = (n: number) => COURT_LABELS[n - 1] ?? String(n)
const numberingCourts = computed(() => Array.from({ length: t.value?.courts ?? 0 }, (_, index) => {
  const court = index + 1
  const courtMatches = (numberingPreview.value?.matches ?? [])
      .filter((match: Match) => match.court === court && !match.isBye)
      .sort((a: Match, b: Match) => (a.day ?? 1) - (b.day ?? 1) || a.order - b.order)
  return { court, matches: courtMatches }
}))
function scheduleWait(match: Match): number { return match.scheduledWaitMinutes ?? 0 }
const scheduleBusy = ref(false)
const scheduleError = ref('')
function resetColumns() {
  for (const court of activeCourts.value) localColumns.value[court] = [...(matchesByCourt.value[court] ?? [])]
}
async function persistSwap(a: string, b: string): Promise<boolean> {
  if (!t.value || scheduleBusy.value) return false
  scheduleBusy.value = true
  scheduleError.value = ''
  try {
    await store.swapMatchPositions(t.value.id, a, b)
    numberingPreview.value = null
    return true
  } catch (error: any) {
    scheduleError.value = error?.message || 'ذخیره جابه‌جایی ناموفق بود'
    resetColumns()
    return false
  } finally { scheduleBusy.value = false }
}

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

function previewParticipantName(match: Match, slot: 1 | 2): string {
  const athleteId = slot === 1 ? match.athlete1Id : match.athlete2Id
  if (athleteId) return t.value?.athletes.find(a => a.id === athleteId)?.name ?? 'نام ورزشکار نامشخص'
  const feeder = (numberingPreview.value?.matches as Match[] | undefined)?.find(m =>
    m.nextMatchId === match.id && m.nextSlot === (slot === 1 ? 'athlete1' : 'athlete2')
  )
  return feeder?.matchNumber ? `برندهٔ بازی ${courtLabel(feeder.court)}${feeder.matchNumber}` : 'در انتظار تعیین حریف'
}

function matchLabel(match: Match): string {
  return match.matchNumber ? `${courtLabel(match.court)}${match.matchNumber}` : 'شماره‌گذاری نشده'
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
        .sort((a, b) => (a.day ?? 1) - (b.day ?? 1) || a.order - b.order)
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

async function onDragEnd(evt: any) {
  const matchId = evt.item?.getAttribute('data-id')
  const toCourt = Number(evt.to?.getAttribute('data-court'))
  if (!matchId || !toCourt || !t.value || scheduleBusy.value) { resetColumns(); return }
  const day = t.value.matches.find(m => m.id === matchId)?.day ?? 1
  const destination = (localColumns.value[toCourt] ?? []).filter(m => (m.day ?? 1) === day)
  const newOrder = destination.findIndex(m => m.id === matchId) + 1
  scheduleBusy.value = true
  scheduleError.value = ''
  try {
    await store.moveMatch(t.value.id, matchId, toCourt, newOrder)
    numberingPreview.value = null
  } catch (error: any) {
    scheduleError.value = error?.message || 'ذخیره جابه‌جایی ناموفق بود'
  } finally { resetColumns(); scheduleBusy.value = false }
}

// ─── جابجایی بازی‌ها (click-to-swap) ─────────────────────────────────────

const selectedMatchId = ref<string | null>(null)
const swapMatchResult = ref('')

async function onMatchClick(match: Match) {
  if (match.winnerId || scheduleBusy.value) return
  if (!selectedMatchId.value) {
    selectedMatchId.value = match.id
    return
  }
  if (selectedMatchId.value === match.id) {
    selectedMatchId.value = null
    return
  }
  const saved = await persistSwap(selectedMatchId.value, match.id)
  if (!saved) return
  swapMatchResult.value = 'جابجایی ذخیره شد'
  selectedMatchId.value = null
  setTimeout(() => (swapMatchResult.value = ''), 2000)
}

// ─── جابجایی متنی بازی‌ها (text swap) ──────────────────────────────────────

const textSwapCodeA = ref('')
const textSwapCodeB = ref('')
const textSwapMessage = ref('')

function findMatchByCode(code: string): Match | null {
  const upper = code.trim().toUpperCase()
  // فرمت: حرف زمین + شماره نمایشی (مثل A3 یا B12)
  const match = upper.match(/^([A-F])(\d+)$/)
  if (!match) return null
  const courtIdx = COURT_LABELS.indexOf(match[1])
  if (courtIdx === -1) return null
  const court = courtIdx + 1
  const number = parseInt(match[2])
  const col = localColumns.value[court] ?? []
  return col.find((m: Match) => m.matchNumber === number) ?? null
}

const previewA = computed(() => findMatchByCode(textSwapCodeA.value))
const previewB = computed(() => findMatchByCode(textSwapCodeB.value))

async function doTextSwap() {
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
  if (await persistSwap(mA.id, mB.id)) {
    textSwapMessage.value = `جابجایی ${textSwapCodeA.value.toUpperCase()} و ${textSwapCodeB.value.toUpperCase()} ذخیره شد`
  }

}
</script>

<template>
  <div v-if="!matches.length" class="flex items-center justify-center h-40 text-slate-400 text-sm">
    ابتدا قرعه‌کشی را انجام دهید.
  </div>

  <div v-else class="mx-auto flex w-full max-w-[1600px] flex-col gap-5 px-4 py-6 lg:px-8">

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

    <section class="overflow-hidden rounded-[1.75rem] border border-blue-200 bg-white shadow-lg shadow-blue-900/5" style="overflow-anchor: none">
      <div class="relative overflow-hidden bg-gradient-to-l from-slate-950 via-blue-950 to-sky-800 p-6 text-white sm:p-8">
        <div class="pointer-events-none absolute -left-10 -top-16 h-52 w-52 rounded-full bg-cyan-400/20 blur-3xl"></div>
        <div class="pointer-events-none absolute -bottom-24 right-1/3 h-48 w-48 rounded-full bg-blue-400/20 blur-3xl"></div>
        <div class="relative flex flex-wrap items-center justify-between gap-6">
          <div class="max-w-2xl">
            <div class="mb-3 inline-flex items-center gap-2 rounded-full border border-cyan-300/30 bg-cyan-300/10 px-3 py-1 text-xs font-black text-cyan-100"><span class="text-base">✦</span> دستیار شماره‌گذاری</div>
            <h2 class="text-2xl font-black leading-tight sm:text-3xl">برنامهٔ مسابقه را با هم می‌چینیم</h2>
            <p class="mt-3 text-sm leading-7 text-sky-100">موجودی هوگو و زمین هر وزن را مشخص کنید؛ پیش از ثبت، همهٔ شماره‌ها و حریف‌ها را می‌بینید.</p>
          </div>
          <button v-if="!aiWizardOpen" :disabled="numberingBusy||rebalanceBusy||scheduleBusy||hasManualPositions||!numberingReadiness?.ready" class="group inline-flex min-h-12 items-center gap-3 rounded-2xl bg-gradient-to-l from-cyan-400 to-sky-300 px-6 py-3 text-sm font-black text-slate-950 shadow-lg shadow-cyan-950/20 transition hover:-translate-y-0.5 hover:from-cyan-300 hover:to-white disabled:cursor-not-allowed disabled:opacity-50" @click="openAiWizard"><span>شروع شماره‌گذاری هوشمند</span><span class="text-lg transition group-hover:-translate-x-1">←</span></button>
        </div>
        <div v-if="!aiWizardOpen" class="relative mt-6 flex flex-wrap gap-2 text-xs font-bold text-sky-100">
          <span class="rounded-full border border-white/20 bg-white/10 px-3 py-1.5">{{playableWeights.length}} وزن</span>
          <span class="rounded-full border border-white/20 bg-white/10 px-3 py-1.5">{{matches.filter(m=>!m.isBye).length}} بازی</span>
          <span class="rounded-full border border-white/20 bg-white/10 px-3 py-1.5">{{t?.courts??0}} زمین</span>
        </div>
        <p v-if="hasManualPositions" class="relative mt-4 rounded-xl border border-amber-300/30 bg-amber-400/10 px-4 py-2 text-xs font-bold text-amber-100">برای شروع، ابتدا چیدمان دستی را با «توزیع مجدد بازی‌ها» بازسازی کنید.</p>
        <p v-else-if="!numberingReadiness?.ready" class="relative mt-4 rounded-xl border border-amber-300/30 bg-amber-400/10 px-4 py-2 text-xs font-bold text-amber-100">راهنما پس از تکمیل وزن‌کشی و قرعهٔ همهٔ وزن‌ها فعال می‌شود.</p>
      </div>
      <div v-if="aiWizardOpen" class="min-h-[38rem] bg-gradient-to-b from-sky-50/80 via-white to-slate-50 p-4 sm:p-7">
        <div class="grid grid-cols-4 gap-1.5 rounded-2xl border border-slate-200 bg-white p-2 shadow-sm sm:gap-2 sm:p-3">
          <div v-for="(label, index) in ['شروع','موجودی هوگو','زمین وزن‌ها','بازبینی']" :key="label" class="flex min-w-0 flex-col items-center gap-1 rounded-xl px-1 py-2 text-center text-[10px] font-black sm:flex-row sm:justify-center sm:gap-2 sm:text-xs" :class="index === Math.min(aiStage,3) ? 'bg-blue-600 text-white shadow-sm' : index < aiStage ? 'bg-sky-50 text-blue-700' : 'text-slate-400'">
            <span class="grid h-6 w-6 shrink-0 place-items-center rounded-full text-xs" :class="index === Math.min(aiStage,3) ? 'bg-white/20' : index < aiStage ? 'bg-blue-100 text-blue-700' : 'bg-slate-100 text-slate-400'">{{index < aiStage ? '✓' : index + 1}}</span><span class="truncate">{{label}}</span>
          </div>
        </div>

        <div v-if="aiStage === 0" class="mt-5 grid gap-5 lg:grid-cols-[1.35fr_1fr]">
          <div class="flex flex-col justify-between rounded-3xl border border-sky-200 bg-gradient-to-br from-white via-sky-50 to-cyan-50 p-6 shadow-sm sm:p-8">
            <div><div class="mb-4 grid h-14 w-14 place-items-center rounded-2xl bg-blue-600 text-2xl text-white shadow-lg shadow-blue-600/20">✦</div><h3 class="text-2xl font-black text-slate-900">چند انتخاب ساده، یک برنامهٔ آماده</h3><p class="mt-3 max-w-xl text-sm leading-8 text-slate-600">در گام بعد، موجودی هوگو را می‌پرسیم. سپس برای هر وزن زمین دلخواه را انتخاب می‌کنید؛ هر وزن می‌تواند روی دو زمین باشد. نتیجه را قبل از ذخیره بررسی خواهید کرد.</p></div>
            <button class="mt-8 inline-flex min-h-12 w-fit items-center gap-3 rounded-xl bg-blue-600 px-6 py-3 text-sm font-black text-white shadow-sm transition hover:bg-blue-700" @click="nextAiStage">شروع تنظیمات <span>←</span></button>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div class="rounded-2xl border border-blue-100 bg-white p-5 shadow-sm"><div class="text-3xl font-black text-blue-700">{{playableWeights.length}}</div><div class="mt-2 text-sm font-bold text-slate-600">وزن مسابقه</div></div>
            <div class="rounded-2xl border border-cyan-100 bg-white p-5 shadow-sm"><div class="text-3xl font-black text-cyan-700">{{matches.filter(m=>!m.isBye).length}}</div><div class="mt-2 text-sm font-bold text-slate-600">بازی قابل اجرا</div></div>
            <div class="col-span-2 rounded-2xl border border-indigo-100 bg-gradient-to-l from-indigo-50 to-white p-5 shadow-sm"><div class="text-3xl font-black text-indigo-700">{{t?.courts??0}}</div><div class="mt-2 text-sm font-bold text-slate-600">زمین برای برنامه‌ریزی</div><p class="mt-3 text-xs leading-6 text-slate-500">زمین‌های انتخابی شما در شماره‌گذاری رعایت می‌شوند.</p></div>
          </div>
        </div>

        <div v-else-if="aiStage === 1" class="mt-5">
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm sm:p-6"><div class="flex items-start gap-3"><div class="grid h-11 w-11 shrink-0 place-items-center rounded-xl bg-cyan-100 text-xl text-cyan-700">◈</div><div><h3 class="text-xl font-black text-slate-900">موجودی هوگو را مشخص کنید</h3><p class="mt-2 text-sm leading-7 text-slate-600">اگر تعداد هوگو محدود است، بازی‌های هم‌زمان متناسب با موجودی شما چیده می‌شوند.</p></div></div></div>
          <label class="mt-4 flex cursor-pointer items-center gap-4 rounded-2xl border p-4 shadow-sm transition sm:p-5" :class="aiOptions.limitHogu ? 'border-blue-400 bg-blue-50' : 'border-slate-200 bg-white hover:border-blue-200'"><input v-model="aiOptions.limitHogu" type="checkbox" class="h-5 w-5 shrink-0 accent-blue-600"><span class="flex-1"><b class="block text-sm text-slate-900">محدودیت تعداد هوگو دارم</b><span class="mt-1 block text-xs leading-6 text-slate-500">با فعال کردن این گزینه، تعداد جفت هوگوی هر سایز را وارد می‌کنید.</span></span><span class="rounded-full px-3 py-1 text-xs font-black" :class="aiOptions.limitHogu ? 'bg-blue-600 text-white' : 'bg-slate-100 text-slate-500'">{{aiOptions.limitHogu ? 'فعال' : 'اختیاری'}}</span></label>
          <div v-if="aiOptions.limitHogu" class="mt-4 rounded-2xl border border-blue-100 bg-white p-4 shadow-sm sm:p-5"><div class="mb-4 text-sm font-black text-slate-800">از هر سایز چند جفت دارید؟</div><div class="grid grid-cols-2 gap-3 sm:grid-cols-4"><label v-for="size in ['1','2','3','4']" :key="size" class="rounded-xl border p-4 text-sm font-bold text-slate-700" :class="neededHoguSizes.includes(size) ? 'border-blue-200 bg-blue-50/50' : 'border-slate-200 bg-slate-50'">سایز {{size}}<span v-if="neededHoguSizes.includes(size)" class="mr-1 text-[10px] text-blue-700">موردنیاز</span><input v-model.number="aiOptions.hoguPairs[size]" type="number" min="0" max="100" class="mt-3 block w-full rounded-lg border border-slate-300 bg-white p-2.5 text-lg font-black text-slate-900 outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"></label></div><p class="mt-3 text-xs leading-6 text-slate-500">سایز پیشنهادی هر وزن از ردهٔ سنی و وزن محاسبه می‌شود.</p></div>
          <div class="mt-6 flex flex-wrap items-center justify-between gap-3"><button class="rounded-xl border border-slate-200 bg-white px-5 py-2.5 text-sm font-bold text-slate-600 hover:bg-slate-50" @click="aiStage=0">بازگشت</button><button class="rounded-xl bg-blue-600 px-6 py-3 text-sm font-black text-white shadow-sm hover:bg-blue-700" @click="nextAiStage">ادامه به انتخاب زمین ←</button></div>
        </div>

        <div v-else-if="aiStage === 2" class="mt-5">
          <div class="flex flex-wrap items-start justify-between gap-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm sm:p-6"><div><div class="mb-2 text-xs font-black text-blue-700">گام ۳ از ۴</div><h3 class="text-xl font-black text-slate-900">زمین هر وزن را انتخاب کنید</h3><p class="mt-2 max-w-2xl text-sm leading-7 text-slate-600">برای هر وزن یک یا دو زمین بزنید. اولین زمین انتخابی محل فینال است؛ انتخاب‌هایتان هنگام ورق زدن حفظ می‌شود.</p></div><span class="rounded-full bg-blue-50 px-4 py-2 text-xs font-black text-blue-700">{{aiCourtPage + 1}} از {{aiCourtPageCount}} · {{playableWeights.length}} وزن</span></div>
          <div class="mt-4 grid gap-3 lg:grid-cols-2">
            <div v-for="(category, index) in visibleAiWeights" :key="category" class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm transition hover:border-blue-300 sm:p-5">
              <div class="mb-4 flex items-start justify-between gap-3"><div><div class="text-[10px] font-black tracking-wide text-blue-600">وزن {{aiCourtPage * aiCourtPageSize + index + 1}} از {{playableWeights.length}}</div><h4 class="mt-1 text-xl font-black text-slate-900">{{category}}</h4></div><div class="rounded-xl bg-sky-50 px-3 py-2 text-left text-xs font-bold text-sky-800">{{matches.filter(m=>!m.isBye&&m.weightCategory===category).length}} بازی<br>هوگو {{aiHoguSize(category)}}</div></div>
              <div class="mb-2 text-xs font-bold text-slate-500">زمین‌های این وزن</div>
              <div class="grid grid-cols-2 gap-2 sm:grid-cols-4"><button v-for="court in Array.from({length:t?.courts??0},(_,i)=>i+1)" :key="court" :disabled="!courtAllowed(category,court)" class="min-h-11 rounded-xl border px-2 py-2 text-xs font-black transition disabled:cursor-not-allowed disabled:opacity-30" :class="(selectedAiCourts[category]??[]).includes(court) ? 'border-blue-600 bg-blue-600 text-white shadow-sm' : 'border-slate-200 bg-slate-50 text-slate-700 hover:border-blue-400 hover:bg-blue-50'" @click="toggleAiCourt(category,court)">زمین {{courtLabel(court)}}<span v-if="selectedAiCourts[category]?.[0]===court" class="block text-[10px] opacity-80">فینال</span></button></div>
              <p class="mt-3 text-[11px] font-bold" :class="selectedAiCourts[category]?.length ? 'text-emerald-700' : 'text-rose-600'">{{selectedAiCourts[category]?.length ? `${selectedAiCourts[category].length} زمین انتخاب شده` : 'لطفاً یک زمین انتخاب کنید'}}</p>
            </div>
          </div>
          <div v-if="aiCourtPageCount > 1" class="mt-4 flex items-center justify-between rounded-2xl border border-blue-100 bg-blue-50 px-4 py-3"><button :disabled="aiCourtPage===0" class="rounded-lg bg-white px-4 py-2 text-xs font-black text-blue-700 shadow-sm disabled:opacity-40" @click="aiCourtPage--">صفحهٔ قبل</button><div class="flex gap-1.5"><span v-for="page in aiCourtPageCount" :key="page" class="h-2 rounded-full transition-all" :class="page-1===aiCourtPage ? 'w-6 bg-blue-600' : 'w-2 bg-blue-200'"></span></div><button :disabled="aiCourtPage>=aiCourtPageCount-1" class="rounded-lg bg-white px-4 py-2 text-xs font-black text-blue-700 shadow-sm disabled:opacity-40" @click="aiCourtPage++">صفحهٔ بعد</button></div>
          <details class="mt-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm"><summary class="cursor-pointer text-sm font-black text-slate-700">تنظیمات بیشتر <span class="text-xs font-medium text-slate-400">· مدت بازی، استراحت و شمارهٔ شروع</span></summary><div class="mt-4 grid gap-3 sm:grid-cols-3"><label class="text-xs font-bold text-slate-600">مدت بازی (دقیقه)<input v-model.number="aiOptions.matchMinutes" type="number" min="1" max="60" class="mt-1 block w-full rounded-lg border border-slate-300 p-2.5 text-base outline-none focus:border-blue-500"></label><label class="text-xs font-bold text-slate-600">استراحت (دقیقه)<input v-model.number="aiOptions.restMinutes" type="number" min="0" max="120" class="mt-1 block w-full rounded-lg border border-slate-300 p-2.5 text-base outline-none focus:border-blue-500"></label><label class="text-xs font-bold text-slate-600">شمارهٔ شروع<input v-model.number="aiOptions.startAt" type="number" min="1" class="mt-1 block w-full rounded-lg border border-slate-300 p-2.5 text-base outline-none focus:border-blue-500"></label></div></details>
          <div class="mt-6 flex flex-wrap items-center justify-between gap-3"><button class="rounded-xl border border-slate-200 bg-white px-5 py-2.5 text-sm font-bold text-slate-600 hover:bg-slate-50" @click="aiStage=1">بازگشت</button><button class="rounded-xl bg-gradient-to-l from-blue-700 to-sky-600 px-6 py-3 text-sm font-black text-white shadow-sm hover:from-blue-800 hover:to-sky-700" @click="confirmAiCourts">ساخت پیش‌نمایش ✦</button></div>
        </div>

        <div v-else-if="aiStage === 3" class="mt-5 flex min-h-[30rem] flex-col items-center justify-center rounded-3xl border border-sky-200 bg-gradient-to-b from-sky-100 via-white to-white p-8 text-center text-slate-900"><div class="grid h-20 w-20 animate-pulse place-items-center rounded-3xl bg-gradient-to-br from-blue-700 to-cyan-400 text-4xl text-white shadow-lg shadow-blue-600/20">✦</div><h3 class="mt-5 text-xl font-black">در حال طراحی برنامهٔ بازی‌ها</h3><p class="mt-2 text-sm text-blue-700">{{aiSteps[aiStep]}}…</p><div class="mt-6 flex flex-wrap justify-center gap-2"><span v-for="(_, index) in aiSteps" :key="index" class="h-2 rounded-full transition-all" :class="index<=aiStep ? 'w-9 bg-blue-600' : 'w-5 bg-slate-200'"></span></div><p class="mt-5 text-xs text-slate-500">پیش‌نمایش را پیش از ثبت نهایی خواهید دید.</p></div>
        <div v-else-if="aiStage === 4 && numberingPreview" class="mt-7">
          <div class="relative overflow-hidden rounded-2xl bg-gradient-to-l from-slate-900 via-blue-900 to-sky-800 p-6 text-white shadow-lg shadow-blue-900/15 sm:p-7">
            <div class="pointer-events-none absolute -left-8 -top-12 h-40 w-40 rounded-full bg-cyan-300/20 blur-2xl"></div>
            <div class="relative flex flex-wrap items-center justify-between gap-4">
              <div>
                <div class="text-xs font-black tracking-wider text-cyan-200">✦ پیش‌نمایش شماره‌گذاری</div>
                <h3 class="mt-2 text-xl font-black sm:text-2xl">برنامهٔ بازی‌ها آماده است</h3>
                <p class="mt-2 text-sm text-sky-100">{{numberingPreview.matches.filter((m: Match) => !m.isBye).length}} بازی در {{numberingCourts.length}} زمین؛ نام حریف‌ها را پیش از ثبت بررسی کنید.</p>
              </div>
              <div class="rounded-2xl border border-white/20 bg-white/10 px-4 py-3 text-center backdrop-blur-sm">
                <div class="text-2xl font-black">{{numberingPreview.matches.filter((m: Match) => !m.isBye).length}}</div>
                <div class="text-xs text-sky-100">بازی آماده</div>
              </div>
            </div>
          </div>
          <p v-if="numberingPreview.reason" class="mt-4 rounded-xl border border-sky-100 bg-sky-50 px-4 py-3 text-sm text-sky-900">{{numberingPreview.reason}}</p>
          <div class="mt-5 grid gap-4 lg:grid-cols-2 xl:grid-cols-4">
            <article v-for="court in numberingCourts" :key="court.court" class="min-w-0 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
              <div class="flex items-center justify-between bg-gradient-to-l px-4 py-4 text-white" :class="court.court % 4 === 1 ? 'from-blue-700 to-sky-500' : court.court % 4 === 2 ? 'from-cyan-700 to-teal-500' : court.court % 4 === 3 ? 'from-violet-700 to-indigo-500' : 'from-orange-600 to-amber-500'">
                <h4 class="text-lg font-black">زمین {{courtLabel(court.court)}}</h4>
                <span class="rounded-full bg-white/20 px-3 py-1 text-xs font-bold">{{court.matches.length}} بازی</span>
              </div>
              <ol class="space-y-2 bg-slate-50/70 p-3">
                <li v-for="match in court.matches" :key="match.id" class="rounded-xl border border-slate-200 bg-white p-3 shadow-sm">
                  <div class="mb-3 flex items-center justify-between gap-2">
                    <b class="rounded-lg bg-blue-50 px-2.5 py-1 text-sm text-blue-800">بازی {{courtLabel(court.court)}}{{match.matchNumber}}</b>
                    <span class="text-xs font-bold text-slate-500">وزن {{match.weightCategory}}</span>
                  </div>
                  <div class="space-y-2 text-sm font-bold text-slate-800">
                    <div class="flex min-w-0 items-center gap-2"><span class="h-2.5 w-2.5 shrink-0 rounded-full bg-blue-500"></span><span class="min-w-0 truncate" :title="previewParticipantName(match, 1)">{{previewParticipantName(match, 1)}}</span></div>
                    <div class="flex min-w-0 items-center gap-2"><span class="h-2.5 w-2.5 shrink-0 rounded-full bg-rose-500"></span><span class="min-w-0 truncate" :title="previewParticipantName(match, 2)">{{previewParticipantName(match, 2)}}</span></div>
                  </div>
                </li>
                <li v-if="!court.matches.length" class="rounded-xl border border-dashed border-slate-200 bg-white p-5 text-center text-sm text-slate-500">هنوز بازی‌ای برای این زمین نیست.</li>
              </ol>
            </article>
          </div>
          <div class="mt-6 flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-slate-200 bg-white p-4">
            <span class="text-xs font-bold text-slate-500">تا زمان تأیید، شماره‌ها ذخیره نمی‌شوند.</span>
            <div class="flex flex-wrap gap-2">
              <button class="rounded-xl border border-slate-200 px-5 py-2 text-sm font-bold text-slate-700 hover:bg-slate-50" @click="aiStage=2">اصلاح زمین‌ها</button>
              <button :disabled="numberingBusy" class="rounded-xl bg-emerald-600 px-7 py-3 text-sm font-black text-white shadow-sm transition hover:bg-emerald-700 disabled:opacity-50" @click="applyNumbering">{{numberingBusy?'در حال ذخیره…':'تأیید و ثبت شماره‌ها'}}</button>
            </div>
          </div>
        </div>
        <p v-if="aiWizardError" role="alert" class="mt-5 rounded-xl bg-rose-50 p-3 text-sm font-bold text-rose-700">{{aiWizardError}}</p>
        <button v-if="aiStage !== 3" class="mt-5 text-xs font-bold text-slate-400 hover:text-slate-700" @click="aiWizardOpen=false;numberingPreview=null">بستن راهنما</button>
      </div>
    </section>

    <div class="rounded-2xl border border-sky-200 bg-sky-50 p-4 shadow-sm">
      <div class="flex flex-wrap items-center gap-3">
        <div class="min-w-0 flex-1">
          <div class="text-sm font-bold text-sky-950">توزیع دوباره روی زمین‌ها</div>
          <div class="mt-1 text-xs leading-5 text-sky-700">
            قرعه و حریف‌ها ثابت می‌مانند؛ فقط زمین، ترتیب اجرا و شماره‌های بازی دوباره محاسبه می‌شوند.
          </div>
        </div>
        <button
            :disabled="rebalanceBusy || !canRebalanceCurrentDraw"
            class="rounded-xl bg-sky-600 px-4 py-2 text-sm font-bold text-white shadow-sm transition-colors hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-50"
            :title="canRebalanceCurrentDraw ? 'توزیع مجدد براساس قرعه فعلی' : 'پس از شروع یا ثبت نتیجهٔ بازی‌ها امکان توزیع مجدد وجود ندارد'"
            @click="rebalanceCurrentDraw"
        >
          {{ rebalanceBusy ? 'در حال توزیع…' : 'توزیع مجدد بازی‌ها' }}
        </button>
      </div>
      <p v-if="!canRebalanceCurrentDraw" class="mt-2 text-xs text-amber-700">
        چون حداقل یک بازی شروع شده یا نتیجه دارد، توزیع مجدد قفل است.
      </p>
      <p v-if="rebalanceMessage" class="mt-2 rounded-lg bg-emerald-100 px-3 py-2 text-xs text-emerald-800">
        ✓ {{ rebalanceMessage }}
      </p>
      <p v-if="rebalanceError" class="mt-2 rounded-lg bg-rose-100 px-3 py-2 text-xs text-rose-800">
        {{ rebalanceError }}
      </p>
    </div>

    <!-- ══════════════ تب دسته‌بندی‌ها ══════════════ -->
    <template v-if="activeTab === 'categories'">

      <!-- بار هر زمین -->
      <div class="grid grid-cols-2 gap-4 xl:grid-cols-4">
        <div
            v-for="cl in courtLoads"
            :key="cl.court"
            class="rounded-2xl bg-slate-800 p-5 text-center text-white shadow-lg"
        >
          <div class="text-xs text-slate-400 mb-1">زمین {{ courtLabel(cl.court) }}</div>
          <div class="text-4xl font-black">{{ cl.total }}</div>
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
      <p v-if="scheduleError" role="alert" class="rounded-xl bg-rose-50 p-3 text-sm text-rose-700">{{scheduleError}}</p>
      <p v-if="scheduleBusy" class="text-sm text-indigo-700">در حال ذخیره جابه‌جایی…</p>

      <div class="flex flex-wrap items-center gap-3 rounded-xl border border-indigo-100 bg-indigo-50 p-4">
        <div class="min-w-0 flex-1"><div class="text-sm font-bold text-indigo-950">تخصیص زمین و شماره‌گذاری</div><div class="mt-1 text-xs text-indigo-700">برنامهٔ زمین‌ها با احتساب استراحت بین دورها ساخته می‌شود. ترتیب دستی حفظ می‌شود؛ برای ساخت از صفر از «توزیع مجدد بازی‌ها» استفاده کنید. برای شماره‌های A1 و B1، محدودهٔ شماره‌گذاری را در تنظیمات روی «هر زمین در هر روز» بگذارید.</div></div>
        <button :disabled="numberingBusy||rebalanceBusy||scheduleBusy||!numberingReadiness?.ready" class="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-bold text-white disabled:opacity-50" @click="applyNumbering">{{numberingBusy?'در حال ثبت…':'شماره‌گذاری و ذخیره'}}</button>
        <button :disabled="numberingBusy||!numberingReadiness?.ready" class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-bold text-white disabled:opacity-50" @click="previewNumbering">{{numberingBusy?'در حال آماده‌سازی…':numberingReadiness?.ready?'پیش‌نمایش شماره‌گذاری':'پس از تکمیل قرعهٔ همهٔ وزن‌ها'}}</button>
        <p v-if="numberingMessage" role="status" class="w-full rounded-lg bg-emerald-100 px-3 py-2 text-xs text-emerald-800">{{numberingMessage}}</p>
        <p v-if="numberingError" class="w-full rounded-lg bg-rose-100 px-3 py-2 text-xs text-rose-800">{{numberingError}}</p>
      </div>

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
              :disabled="scheduleBusy || !previewA || !previewB"
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
              :disabled="scheduleBusy"
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

                    <p v-if="match.scheduledStartMinute != null" class="mb-1 text-[10px] text-slate-500">روز {{match.day ?? 1}} · دقیقه {{match.scheduledStartMinute}} تا {{match.scheduledEndMinute}} از شروع روز</p>
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

  <div v-if="numberingPreview && !aiWizardOpen" class="fixed inset-0 z-[70] grid place-items-center bg-slate-950/50 p-4" @click.self="numberingPreview=null">
    <section class="max-h-[90vh] w-full max-w-4xl overflow-auto rounded-2xl bg-white p-5 shadow-2xl" dir="rtl">
      <div class="flex items-start justify-between gap-3"><div><h3 class="text-lg font-black text-slate-900">پیش‌نمایش صف و شمارهٔ زمین‌ها</h3><p class="mt-1 text-sm text-slate-500">این پیش‌نمایش هنوز ذخیره نشده است؛ برای ثبت دکمهٔ «تأیید و ثبت شماره‌ها» را بزنید. زمان‌ها از شروع هر روز و به‌صورت تخمینی هستند. بازی‌های آماده فاصلهٔ استراحت را پر می‌کنند؛ انتظار باقی‌مانده مشخص می‌شود. ترتیب دستی حفظ می‌شود.</p><p v-if="numberingPreview.reason" class="mt-2 rounded-lg bg-violet-50 p-2 text-sm font-bold text-violet-800">✦ {{numberingPreview.reason}}</p></div><button class="rounded-lg px-3 py-1 text-slate-500 hover:bg-slate-100" @click="numberingPreview=null">بستن</button></div>
      <div class="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <article v-for="court in numberingCourts" :key="court.court" class="rounded-xl border border-slate-200 p-3"><h4 class="mb-2 font-bold">زمین {{courtLabel(court.court)}}</h4><ol class="space-y-1 text-xs"><li v-for="(match, index) in court.matches" :key="match.id" class="border-b border-slate-100 py-1"><div v-if="scheduleWait(match) > 0" class="mb-1 rounded bg-amber-50 px-2 py-1 font-bold text-amber-800">انتظار برنامه‌ریزی‌شده: {{scheduleWait(match)}} دقیقه</div><div v-if="index === 0 || court.matches[index - 1].weightCategory !== match.weightCategory" class="mb-1 rounded bg-indigo-50 px-2 py-1 font-bold text-indigo-800">وزن {{match.weightCategory}}</div><div class="flex gap-2"><b class="shrink-0">{{courtLabel(court.court)}}{{match.matchNumber}}</b><span class="text-slate-500">دقیقه {{match.scheduledStartMinute}} تا {{match.scheduledEndMinute}}</span><span>{{athleteDisplayName(match.athlete1Id,match.id,1)}} / {{athleteDisplayName(match.athlete2Id,match.id,2)}}</span></div></li></ol><p v-if="!court.matches.length" class="text-xs text-slate-400">بازی ندارد</p></article>
      </div>
      <p v-if="numberingError" class="mt-3 rounded-lg bg-rose-50 p-3 text-sm text-rose-700">{{numberingError}}</p>
      <div class="sticky bottom-0 mt-5 flex justify-end gap-2 border-t bg-white py-3"><button :disabled="numberingBusy" class="rounded-lg border px-4 py-2 text-sm" @click="numberingPreview=null">لغو</button><button :disabled="numberingBusy" class="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-bold text-white disabled:opacity-50" @click="applyNumbering">{{numberingBusy?'در حال ثبت…':'تأیید و ثبت شماره‌ها'}}</button></div>
    </section>
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
