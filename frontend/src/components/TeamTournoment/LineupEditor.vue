<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import { useTeamTournamentStore } from '../../stores/teamTournament'

const props = withDefaults(
    defineProps<{
      tournamentId: string
      matchId: string
      teamId: string
      side?: 'right' | 'left'
    }>(),
    { side: 'right' },
)

const store = useTeamTournamentStore()

/* ---------- تایپ‌های کمکی ---------- */

type BenchItem = {
  athleteId: string
  name: string
  /** برچسب وزنی مثل «-۵۴» */
  weightLabel: string | null
  /** وزن‌کشی عددی */
  weighInKg: number | null
  slotNo: number | null
  usedInSlot: number | null
  isEligible: boolean
  reason: string | null
}

type Issue = {
  level: 'error' | 'warning'
  code: string
  message: string
  slotNo?: number
  athleteId?: string
}

type ValidationView = {
  isValid: boolean
  canSubmit: boolean
  issues: Issue[]
  filledCount: number
  totalSlots: number
  emptySlots: number[]
}

type Candidate = { item: BenchItem; ok: boolean; reason: string | null }

type ActionResult =
    | string
    | null
    | void
    | { error?: string | null; filled?: number; encounterId?: string | null }

function errOf(res: ActionResult): string | null {
  if (!res || typeof res === 'string') return null
  return res.error ?? null
}

/* ---------- UI state ---------- */

const activeSlot = ref<number | null>(null)

const toast = ref<string | null>(null)
let toastTimer = 0

function flash(msg: string) {
  toast.value = msg
  window.clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => (toast.value = null), 2500)
}

onBeforeUnmount(() => window.clearTimeout(toastTimer))

/* ---------- تضمین مواجهه از روی matchId ---------- */

const encounterId = ref<string | null>(null)
const encError = ref<string | null>(null)

watch(
    () => [props.tournamentId, props.matchId] as const,
    ([tid, mid]) => {
      encounterId.value = null
      encError.value = null
      activeSlot.value = null

      if (!tid || !mid) {
        encError.value = 'بازی انتخاب نشده است.'
        return
      }

      const res = store.ensureEncounter(tid, mid) as ActionResult

      if (typeof res === 'string') {
        encounterId.value = res || null
        if (!res) encError.value = 'مواجهه‌ای برای این بازی ساخته نشد.'
        return
      }

      const obj = (res ?? {}) as { encounterId?: string | null; error?: string | null }
      if (obj.error) {
        encError.value = obj.error
        return
      }

      encounterId.value = obj.encounterId ?? null
      if (!encounterId.value) encError.value = 'مواجهه‌ای برای این بازی ساخته نشد.'
    },
    { immediate: true },
)

const eid = computed(() => encounterId.value)
const ready = computed(() => eid.value !== null)

/* ---------- نرمال‌سازی داده‌های استور ---------- */

const fullName = (a: any): string => {
  if (!a) return '—'
  const n =
      a.name ??
      a.fullName ??
      [a.firstName, a.lastName].filter(Boolean).join(' ')
  return n?.trim() || '—'
}

const num = (v: any): number | null => {
  if (v === null || v === undefined || v === '') return null
  const n = Number(v)
  return Number.isFinite(n) ? n : null
}

/** ارقام فارسی/عربی ← لاتین */
function toLatinDigits(s: string): string {
  return s
      .replace(/[۰-۹]/g, (d) => String('۰۱۲۳۴۵۶۷۸۹'.indexOf(d)))
      .replace(/[٠-٩]/g, (d) => String('٠١٢٣٤٥٦٧٨٩'.indexOf(d)))
}

/** برچسب وزنی خوانا: «-54» / «+87» */
function weightLabelOf(...src: any[]): string | null {
  for (const o of src) {
    if (!o) continue
    const raw =
        o.weightCategory ?? o.weightClass ?? o.weightTitle ?? o.category ?? null
    if (!raw) continue

    const s = toLatinDigits(String(raw))
        .replace(/[−–—]/g, '-')
        .replace(/کیلوگرم|کیلو|kg/gi, '')
        .trim()

    const n = s.match(/\d+(?:\.\d+)?/)
    if (!n) continue

    const plus = /\+|بالای|بیش\s*از|over/i.test(s)
    return `${plus ? '+' : '-'}${n[0]}`
  }
  return null
}

/** وزن‌کشی عددی */
const weighInOf = (...src: any[]): number | null => {
  for (const o of src) {
    if (!o) continue
    const n = num(
        o.weighInKg ?? o.weightKg ?? o.weight ?? o.weighIn ?? o.bodyWeight ?? o.kg,
    )
    if (n != null) return n
  }
  return null
}

function normalizeRoster(raw: any): BenchItem[] {
  const list: any[] = Array.isArray(raw)
      ? raw
      : Array.isArray(raw?.entries)
          ? raw.entries
          : []

  return list
      .map((r) => {
        const a = r?.athlete ?? r
        const m = r?.member ?? {}
        const athleteId = String(r?.athleteId ?? a?.id ?? m?.athleteId ?? '')
        const full = athleteId ? store.athleteById?.(props.tournamentId, athleteId) : null

        const direct = r?.name ?? (fullName(a) !== '—' ? fullName(a) : null)

        return {
          athleteId,
          name: direct ?? fullName(full),
          weightLabel: weightLabelOf(a, r, m, full),
          weighInKg: weighInOf(r, m, a, full),
          slotNo: num(r?.slotNo ?? m?.slotNo),
          usedInSlot: r?.usedInSlot ?? null,
          isEligible: r?.isEligible !== false,
          reason: r?.ineligibleReason ?? r?.reason ?? null,
        } as BenchItem
      })
      .filter((x) => x.athleteId)
}

function normalizeEntries(raw: any): Record<number, string | null> {
  const src = raw?.entries ?? raw ?? {}
  const out: Record<number, string | null> = {}
  for (const [k, v] of Object.entries(src)) {
    const n = Number(k)
    if (!Number.isNaN(n)) out[n] = (v as string | null) ?? null
  }
  return out
}

const EMPTY_VALIDATION: ValidationView = {
  isValid: false,
  canSubmit: false,
  issues: [],
  filledCount: 0,
  totalSlots: 0,
  emptySlots: [],
}

/* ---------- state مشتق‌شده ---------- */

const slots = computed(() => store.weightSlotsOf(props.tournamentId) ?? [])

const editable = computed(() =>
    ready.value && store.isLineupEditable(props.tournamentId, eid.value!, props.teamId),
)

const canEdit = computed(() => ready.value && editable.value)

const entries = computed<Record<number, string | null>>(() =>
    ready.value
        ? normalizeEntries(store.getLineup(props.tournamentId, eid.value!, props.teamId))
        : {},
)

const validation = computed<ValidationView>(() => {
  if (!ready.value) return EMPTY_VALIDATION
  const v = store.validateLineup(
      props.tournamentId, eid.value!, props.teamId,
  ) as Partial<ValidationView> | null
  return { ...EMPTY_VALIDATION, ...(v ?? {}) }
})

const roster = computed<BenchItem[]>(() =>
    ready.value
        ? normalizeRoster(store.rosterFor(props.tournamentId, eid.value!, props.teamId))
        : [],
)

const byId = computed<Record<string, BenchItem>>(() => {
  const m: Record<string, BenchItem> = {}
  for (const r of roster.value) m[r.athleteId] = r
  return m
})

const slotOfAthlete = computed<Record<string, number>>(() => {
  const m: Record<string, number> = {}
  for (const [k, v] of Object.entries(entries.value)) if (v) m[v] = Number(k)
  for (const r of roster.value) if (r.usedInSlot != null) m[r.athleteId] = r.usedInSlot
  return m
})

const freeCount = computed(
    () => roster.value.filter((r) => slotOfAthlete.value[r.athleteId] == null).length,
)

/* ---------- نمایش وزن ---------- */

/** متن وزن یک بازیکن: برچسب وزنی و در صورت وجود وزن‌کشی */
function weightText(item: BenchItem | null): string {
  if (!item) return '—'
  const parts: string[] = []
  if (item.weightLabel) parts.push(`${item.weightLabel} kg`)
  return parts.length ? parts.join(' · ') : '—'
}

/* ---------- کاندیداها: همهٔ بازیکنان تیم ---------- */

function candidatesFor(slotNo: number): Candidate[] {
  const current = entries.value[slotNo]

  return roster.value.map((item): Candidate => {
    if (item.athleteId === current) return { item, ok: true, reason: 'انتخاب فعلی' }

    const busyIn = slotOfAthlete.value[item.athleteId]
    if (busyIn != null) return { item, ok: false, reason: `در اسلات ${busyIn}` }

    if (!item.isEligible) return { item, ok: false, reason: item.reason ?? 'غیرمجاز' }

    return { item, ok: true, reason: null }
  })
}

function visibleFor(slotNo: number): Candidate[] {
  return [...candidatesFor(slotNo)].sort((a, b) => {
    if (a.ok !== b.ok) return a.ok ? -1 : 1
    const wa = a.item.weighInKg ?? Number(a.item.weightLabel?.slice(1)) ?? 1e9
    const wb = b.item.weighInKg ?? Number(b.item.weightLabel?.slice(1)) ?? 1e9
    if (wa !== wb) return wa - wb
    return a.item.name.localeCompare(b.item.name, 'fa')
  })
}

const eligibleCount = (slotNo: number): number =>
    candidatesFor(slotNo)
        .filter((c) => c.ok && c.item.athleteId !== entries.value[slotNo]).length

function athleteInSlot(slotNo: number): BenchItem | null {
  const aid = entries.value[slotNo]
  if (!aid) return null

  const known = byId.value[aid]
  if (known) return known

  const a = store.athleteById?.(props.tournamentId, aid)
  return {
    athleteId: aid,
    name: fullName(a),
    weightLabel: weightLabelOf(a),
    weighInKg: weighInOf(a),
    slotNo,
    usedInSlot: slotNo,
    isEligible: true,
    reason: null,
  }
}

const issuesFor = (slotNo: number): Issue[] =>
    validation.value.issues.filter((i: Issue) => i.slotNo === slotNo)

const slotHasError = (slotNo: number): boolean =>
    issuesFor(slotNo).some((i: Issue) => i.level === 'error')

const globalIssues = computed<Issue[]>(() =>
    validation.value.issues.filter((i: Issue) => i.slotNo == null),
)

/* ---------- انتخاب ---------- */

function toggleSlot(slotNo: number) {
  if (!canEdit.value) return
  activeSlot.value = activeSlot.value === slotNo ? null : slotNo
}

function pick(slotNo: number, c: Candidate) {
  if (!canEdit.value) return

  if (entries.value[slotNo] === c.item.athleteId) {
    clear(slotNo)
    return
  }

  if (!c.ok) {
    flash(c.reason ?? 'این ورزشکار قابل انتخاب نیست')
    return
  }

  const e = errOf(
      store.assignSlot(
          props.tournamentId, eid.value!, props.teamId, slotNo, c.item.athleteId,
      ) as ActionResult,
  )
  if (e) flash(e)
  else activeSlot.value = null
}

/* ---------- اکشن‌ها ---------- */

function clear(slotNo: number) {
  if (!canEdit.value) return
  const e = errOf(
      store.clearSlot(props.tournamentId, eid.value!, props.teamId, slotNo) as ActionResult,
  )
  if (e) flash(e)
}

/**
 * پر کردن خودکار: اسلات‌های خالی به ترتیب، با بازیکنان آزادِ مجاز.
 * هر بازیکن حداکثر یک بار استفاده می‌شود.
 */
function autoFillLocal(): number {
  const taken = new Set(Object.values(entries.value).filter(Boolean) as string[])
  for (const [aid, s] of Object.entries(slotOfAthlete.value)) {
    if (s != null) taken.add(aid)
  }

  const pool = roster.value
      .filter((it) => it.isEligible && !taken.has(it.athleteId))
      .sort((a, b) => {
        const wa = a.weighInKg ?? Number(a.weightLabel?.slice(1)) ?? 1e9
        const wb = b.weighInKg ?? Number(b.weightLabel?.slice(1)) ?? 1e9
        if (wa !== wb) return wa - wb
        return a.name.localeCompare(b.name, 'fa')
      })

  const empty = slots.value
      .map((s: any) => Number(s.slotNo))
      .filter((n) => !entries.value[n])
      .sort((a, b) => a - b)

  let filled = 0
  for (const slotNo of empty) {
    const chosen = pool.find((it) => !taken.has(it.athleteId))
    if (!chosen) break

    const e = errOf(
        store.assignSlot(
            props.tournamentId, eid.value!, props.teamId, slotNo, chosen.athleteId,
        ) as ActionResult,
    )
    if (!e) {
      taken.add(chosen.athleteId)
      filled++
    } else {
      // این بازیکن را از استخر خارج کن تا در حلقه گیر نکنیم
      taken.add(chosen.athleteId)
    }
  }
  return filled
}

function autoFill() {
  if (!canEdit.value) return
  const filled = autoFillLocal()
  flash(filled ? `${filled} اسلات پر شد` : 'بازیکن آزادی برای اسلات‌های خالی نبود')
  activeSlot.value = null
}

function resetAll() {
  if (!canEdit.value) return
  const e = errOf(
      store.resetLineup(props.tournamentId, eid.value!, props.teamId) as ActionResult,
  )
  flash(e ?? 'ترکیب پاک شد')
  activeSlot.value = null
}

function submit() {
  if (!ready.value) return
  const e = errOf(
      store.submitDraftLineup(props.tournamentId, eid.value!, props.teamId) as ActionResult,
  )
  flash(e ?? 'ترکیب ثبت شد')
  if (!e) activeSlot.value = null
}

function unsubmit() {
  if (!ready.value) return
  const e = errOf(
      store.unsubmitLineup(props.tournamentId, eid.value!, props.teamId) as ActionResult,
  )
  flash(e ?? 'ترکیب به حالت پیش‌نویس برگشت')
}
</script>

<template>
  <div class="flex min-w-0 flex-col gap-3" dir="rtl">

    <!-- نوار ابزار -->
    <header class="flex flex-wrap items-center gap-2 rounded-xl border border-slate-200 bg-white p-3">
      <span class="text-sm text-slate-600">
        {{ validation.filledCount }} از {{ validation.totalSlots || slots.length }}
      </span>

      <span class="text-xs text-slate-400">آزاد: {{ freeCount }}</span>

      <span
          v-if="ready && !editable"
          class="rounded-md bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700"
      >
        ثبت‌شده
      </span>

      <div class="mr-auto flex flex-wrap items-center gap-2">
        <button
            type="button"
            class="rounded border border-slate-200 px-2 py-1 text-sm hover:bg-slate-50 disabled:opacity-50"
            :disabled="!canEdit"
            @click="autoFill"
        >
          پر کردن خودکار
        </button>

        <button
            type="button"
            class="rounded border border-rose-200 px-2 py-1 text-sm text-rose-600 hover:bg-rose-50 disabled:opacity-50"
            :disabled="!canEdit"
            @click="resetAll"
        >
          پاک کردن همه
        </button>

        <button
            v-if="editable"
            type="button"
            class="rounded bg-indigo-600 px-3 py-1 text-sm text-white hover:bg-indigo-700 disabled:opacity-50"
            :disabled="!validation.canSubmit"
            @click="submit"
        >
          تایید و ثبت نهایی
        </button>

        <button
            v-else-if="ready"
            type="button"
            class="rounded border border-slate-300 px-3 py-1 text-sm text-slate-700 hover:bg-slate-50"
            @click="unsubmit"
        >
          ویرایش مجدد
        </button>
      </div>
    </header>

    <p
        v-if="!ready"
        class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-700"
    >
      {{ encError ?? 'مواجهه‌ای برای این بازی وجود ندارد؛ ابتدا جدول گروهی را بسازید و هر دو تیم را مشخص کنید.' }}
    </p>

    <template v-else>
      <p
          v-if="!slots.length"
          class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-700"
      >
        اسلات وزنی تعریف نشده است؛ از تنظیمات مسابقه اوزان (۸ یا ۱۰ نفره) را مشخص کنید.
      </p>

      <p
          v-if="!roster.length"
          class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-700"
      >
        روستر این تیم خالی است؛ ابتدا اعضای تیم را ثبت کنید.
      </p>

      <ul
          v-if="globalIssues.length"
          class="space-y-1 rounded-lg border border-amber-200 bg-amber-50 p-3"
      >
        <li
            v-for="issue in globalIssues"
            :key="issue.code"
            class="text-xs"
            :class="issue.level === 'error' ? 'text-rose-700' : 'text-amber-700'"
        >
          ⚠ {{ issue.message }}
        </li>
      </ul>

      <!-- اسلات‌ها -->
      <ul class="space-y-2">
        <li
            v-for="slot in slots"
            :key="slot.slotNo"
            class="overflow-hidden rounded-lg border bg-white transition-colors"
            :class="[
              slotHasError(slot.slotNo)
                ? 'border-rose-300'
                : activeSlot === slot.slotNo
                  ? 'border-indigo-400 ring-1 ring-indigo-200'
                  : 'border-slate-200',
            ]"
        >
          <!-- سربرگ اسلات -->
          <div
              class="flex items-center gap-3 p-3"
              :class="canEdit ? 'cursor-pointer hover:bg-slate-50' : 'cursor-default'"
              role="button"
              tabindex="0"
              @click="toggleSlot(slot.slotNo)"
              @keydown.enter.prevent="toggleSlot(slot.slotNo)"
              @keydown.space.prevent="toggleSlot(slot.slotNo)"
          >
            <div class="w-28 shrink-0">
              <div class="text-xs font-medium text-slate-600">
                اسلات {{ slot.slotNo }} — {{ slot.title }}
              </div>
            </div>

            <div class="min-w-0 flex-1">
              <template v-if="athleteInSlot(slot.slotNo)">
                <span class="truncate text-sm font-semibold text-slate-800">
                  {{ athleteInSlot(slot.slotNo)?.name }}
                </span>
                <span class="mr-2 text-xs text-slate-500">
                  {{ weightText(athleteInSlot(slot.slotNo)) }}
                </span>
              </template>
              <span v-else class="text-sm text-slate-300">
                {{ canEdit ? 'برای انتخاب ورزشکار کلیک کنید' : 'خالی' }}
              </span>
            </div>

            <span
                v-if="canEdit && !entries[slot.slotNo]"
                class="shrink-0 text-[10px] text-slate-400"
            >
              {{ eligibleCount(slot.slotNo) }} گزینه
            </span>

            <button
                v-if="entries[slot.slotNo] && canEdit"
                type="button"
                class="shrink-0 rounded px-2 py-0.5 text-xs text-rose-600 hover:bg-rose-50"
                @click.stop="clear(slot.slotNo)"
            >
              حذف
            </button>

            <span v-if="canEdit" class="shrink-0 text-slate-400">
              {{ activeSlot === slot.slotNo ? '▴' : '▾' }}
            </span>
          </div>

          <!-- پنل انتخاب -->
          <div
              v-if="activeSlot === slot.slotNo && canEdit"
              class="border-t border-slate-100 bg-slate-50 p-3"
          >
            <div class="mb-2 flex items-center justify-between">
              <span class="text-[11px] text-slate-500">بازیکنان تیم</span>
              <span class="text-[11px] text-slate-400">
                {{ visibleFor(slot.slotNo).length }} نفر
              </span>
            </div>

            <p
                v-if="!visibleFor(slot.slotNo).length"
                class="py-4 text-center text-xs text-slate-400"
            >
              بازیکنی در روستر نیست.
            </p>

            <div v-else class="grid max-h-64 grid-cols-1 gap-1 overflow-y-auto">
              <button
                  v-for="c in visibleFor(slot.slotNo)"
                  :key="c.item.athleteId"
                  type="button"
                  class="flex items-center justify-between gap-2 rounded border px-2 py-1.5 text-right text-sm transition-colors"
                  :class="[
                    entries[slot.slotNo] === c.item.athleteId
                      ? 'border-indigo-600 bg-indigo-600 text-white'
                      : c.ok
                        ? 'border-slate-200 bg-white hover:border-indigo-400 hover:bg-indigo-50'
                        : 'cursor-not-allowed border-slate-100 bg-slate-100 text-slate-400',
                  ]"
                  :disabled="!c.ok && entries[slot.slotNo] !== c.item.athleteId"
                  :title="c.reason ?? ''"
                  @click="pick(slot.slotNo, c)"
              >
                <span class="truncate">{{ c.item.name }}</span>
                <span class="shrink-0 text-[10px] opacity-70">
                  {{ weightText(c.item) }}
                  <template v-if="!c.ok && c.reason"> · {{ c.reason }}</template>
                </span>
              </button>
            </div>
          </div>

          <!-- خطاهای اسلات -->
          <p
              v-for="issue in issuesFor(slot.slotNo)"
              :key="issue.code"
              class="px-3 pb-2 text-[10px]"
              :class="issue.level === 'error' ? 'font-bold text-rose-600' : 'text-amber-600'"
          >
            ⚠ {{ issue.message }}
          </p>
        </li>
      </ul>
    </template>

    <transition
        enter-active-class="transition duration-200"
        enter-from-class="translate-y-2 opacity-0"
        leave-active-class="transition duration-200"
        leave-to-class="translate-y-2 opacity-0"
    >
      <div
          v-if="toast"
          class="fixed bottom-4 z-50 rounded-lg bg-slate-800 px-4 py-2 text-sm text-white shadow-lg"
          :class="props.side === 'left' ? 'left-4' : 'right-4'"
      >
        {{ toast }}
      </div>
    </transition>
  </div>
</template>
