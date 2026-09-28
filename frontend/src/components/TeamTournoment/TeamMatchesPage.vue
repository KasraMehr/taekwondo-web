<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import draggable from 'vuedraggable'
import { useTeamTournamentStore } from '../../stores/teamTournament'
import { COURTS, TIME_SLOTS, type TimeSlot, MAX_COURTS_PER_GROUP } from '../../types'
import type { TeamMatch } from '../../types'

const store = useTeamTournamentStore()
const t = computed(() => store.current)
const teams = computed(() => t.value?.teams ?? [])
const groups = computed(() => t.value?.groups ?? [])
const matches = computed(() => t.value?.matches ?? [])

const activeTab = ref<'groups' | 'schedule'>('groups')

/* ─── پیام‌ها ─── */
const notice = ref<{ text: string; kind: 'error' | 'ok' } | null>(null)
let noticeTimer: number | undefined
function notify(text: string, kind: 'error' | 'ok' = 'error') {
  notice.value = { text, kind }
  clearTimeout(noticeTimer)
  noticeTimer = window.setTimeout(() => (notice.value = null), 3500)
}

/* ─── کمکی‌ها ─── */
const COURT_LABELS = ['A', 'B', 'C', 'D']
const courtLabel = (n: number) => COURT_LABELS[n - 1] ?? String(n)
const teamName = (id: string) => teams.value.find(x => x.id === id)?.name ?? '؟'
// const slotLabel = (id: number) => TIME_SLOTS.find(s => s.id === id)?.label ?? `سانس ${id}`

const GROUP_STYLES = [
  { chip: 'bg-indigo-100 text-indigo-700', bar: 'bg-indigo-400', ring: 'border-indigo-200' },
  { chip: 'bg-emerald-100 text-emerald-700', bar: 'bg-emerald-400', ring: 'border-emerald-200' },
  { chip: 'bg-amber-100 text-amber-700', bar: 'bg-amber-400', ring: 'border-amber-200' },
  { chip: 'bg-sky-100 text-sky-700', bar: 'bg-sky-400', ring: 'border-sky-200' },
  { chip: 'bg-rose-100 text-rose-700', bar: 'bg-rose-400', ring: 'border-rose-200' },
]
const groupIndex = (gid: string) => Math.max(0, groups.value.findIndex(g => g.id === gid))
const groupStyle = (gid: string) => GROUP_STYLES[groupIndex(gid) % GROUP_STYLES.length]!
const groupName = (gid: string) => groups.value.find(g => g.id === gid)?.name ?? '—'

const isFinished = (m: TeamMatch) => m.scoreA !== null && m.scoreB !== null

/* ─── تب گروه‌ها ─── */
const newGroupName = ref('')

const unassignedTeams = computed(() => {
  const taken = new Set(groups.value.flatMap(g => g.teamIds))
  return teams.value.filter(x => !taken.has(x.id))
})

const groupRows = computed(() =>
    groups.value.map(g => {
      const ms = matches.value.filter(m => m.groupId === g.id)
      return {
        ...g,
        matchCount: ms.length,
        scheduled: ms.filter(m => m.courtId !== null && m.slotId !== null).length,
        done: ms.filter(isFinished).length,
        courts: t.value?.courtsPerGroup[g.id] ?? [],
      }
    })
)

function addGroup() {
  if (!t.value) return
  const { error } = store.addGroup(t.value.id, newGroupName.value)
  if (error) return notify(error)
  newGroupName.value = ''
}

function toggleCourt(groupId: string, court: number) {
  if (!t.value) return
  const cur = [...(t.value.courtsPerGroup[groupId] ?? [])]
  const i = cur.indexOf(court)
  if (i >= 0) cur.splice(i, 1)
  else {
    if (cur.length >= MAX_COURTS_PER_GROUP)
      return notify(`هر گروه حداکثر ${MAX_COURTS_PER_GROUP} زمین می‌تواند داشته باشد`)
    cur.push(court)
  }
  const { error } = store.setGroupCourts(t.value.id, groupId, cur.sort((a, b) => a - b))
  if (error) notify(error)
}

function assignTeam(groupId: string, e: Event) {
  const el = e.target as HTMLSelectElement
  const teamId = el.value
  if (!t.value || !teamId) return
  const { error } = store.assignTeamToGroup(t.value.id, groupId, teamId)
  el.value = ''
  if (error) notify(error)
}

function generate(groupId: string) {
  if (!t.value) return
  const { error } = store.generateGroupMatches(t.value.id, groupId)
  if (error) return notify(error)
  notify('بازی‌های دوره‌ای ساخته شد', 'ok')
}

function generateAll() {
  if (!t.value) return
  let ok = 0
  for (const g of groups.value) {
    if (!store.generateGroupMatches(t.value.id, g.id).error) ok++
  }
  notify(ok ? `بازی‌های ${ok} گروه ساخته شد` : 'گروه معتبری برای قرعه‌کشی نبود', ok ? 'ok' : 'error')
}

/* ─── تب زمان‌بندی ─── */
const poolFilter = ref<string>('')

const localGrid = ref<Record<number, Record<number, TeamMatch[]>>>({})
const localPool = ref<TeamMatch[]>([])

function rebuild() {
  const grid: Record<number, Record<number, TeamMatch[]>> = {}
  for (const s of TIME_SLOTS) {
    grid[s.id] = {}
    for (const c of COURTS) {
      const m = store.scheduleGrid[s.id]?.[c] ?? null
      grid[s.id]![c] = m ? [m] : []
    }
  }
  localGrid.value = grid
  localPool.value = store.matchPool.filter(
      m => !poolFilter.value || m.groupId === poolFilter.value
  )
}

watch([() => store.scheduleGrid, () => store.matchPool, poolFilter], rebuild, {
  immediate: true,
  deep: true,
})

function onDragEnd(evt: any) {
  const matchId: string | null = evt.item?.getAttribute('data-id') ?? null
  const to = evt.to as HTMLElement | null
  if (!matchId || !to || !t.value) return rebuild()

  const slot = to.getAttribute('data-slot')
  const court = to.getAttribute('data-court')

  const res = slot && court
      ? store.placeMatch(t.value.id, matchId, Number(court), Number(slot))
      : store.placeMatch(t.value.id, matchId, null, null)

  if (res.error) notify(res.error)
  rebuild()
}

function unplace(matchId: string) {
  if (!t.value) return
  store.placeMatch(t.value.id, matchId, null, null)
}

function runAuto() {
  if (!t.value) return
  const res = store.autoSchedule(t.value.id)
  if (res.error) return notify(res.error)
  notify(
      res.unplaced
          ? `${res.placed} بازی چیده شد، ${res.unplaced} بازی جا نشد`
          : `همهٔ ${res.placed} بازی چیده شد`,
      res.unplaced ? 'error' : 'ok'
  )
}

function onScore(m: TeamMatch, side: 'A' | 'B', e: Event) {
  if (!t.value) return
  const raw = (e.target as HTMLInputElement).value
  const val = raw === '' ? null : Number(raw)
  store.setMatchScore(
      t.value.id, m.id,
      side === 'A' ? val : m.scoreA,
      side === 'B' ? val : m.scoreB,
  )
}

/* ─── آمار ─── */
const courtLoads = computed(() =>
    COURTS.map(c => {
      const ms = matches.value.filter(m => m.courtId === c)
      return { court: c, total: ms.length, done: ms.filter(isFinished).length }
    })
)
const unscheduledCount = computed(() => store.matchPool.length)

const slots = TIME_SLOTS as readonly TimeSlot[]
</script>

<template>
  <div v-if="!t" class="flex items-center justify-center h-40 text-slate-400 text-sm">
    تورنمنتی انتخاب نشده است.
  </div>

  <div v-else class="p-4 flex flex-col gap-4 max-w-6xl mx-auto">
    <!-- تب‌ها -->
    <div class="flex bg-slate-100 rounded-full p-0.5 w-fit self-center">
      <button
          class="text-xs rounded-full px-4 py-1.5 transition-colors"
          :class="activeTab === 'groups' ? 'bg-white shadow text-slate-800' : 'text-slate-500'"
          @click="activeTab = 'groups'"
      >گروه‌بندی</button>
      <button
          class="text-xs rounded-full px-4 py-1.5 transition-colors"
          :class="activeTab === 'schedule' ? 'bg-white shadow text-slate-800' : 'text-slate-500'"
          @click="activeTab = 'schedule'"
      >زمان‌بندی زمین‌ها</button>
    </div>

    <!-- پیام -->
    <Transition enter-active-class="transition-all duration-200" enter-from-class="opacity-0 -translate-y-1" leave-to-class="opacity-0">
      <div
          v-if="notice"
          class="text-sm rounded-xl px-4 py-2.5 border flex items-center gap-2"
          :class="notice.kind === 'error'
            ? 'bg-amber-50 border-amber-300 text-amber-800'
            : 'bg-emerald-50 border-emerald-300 text-emerald-800'"
      >
        {{ notice.kind === 'error' ? '⚠️' : '✓' }} {{ notice.text }}
      </div>
    </Transition>

    <!-- ══════════ گروه‌بندی ══════════ -->
    <template v-if="activeTab === 'groups'">
      <div class="flex flex-wrap items-center gap-2">
        <input
            v-model="newGroupName"
            placeholder="نام گروه (مثلاً گروه الف)"
            class="text-xs border border-slate-200 rounded-lg px-3 py-2 bg-white text-slate-700 focus:outline-none focus:border-indigo-400 w-52"
            @keyup.enter="addGroup"
        />
        <button
            class="text-xs rounded-lg px-3 py-2 bg-slate-800 text-white hover:bg-slate-700 transition-colors"
            @click="addGroup"
        >+ گروه جدید</button>
        <button
            v-if="groups.length"
            class="text-xs rounded-lg px-3 py-2 border border-indigo-300 text-indigo-600 hover:bg-indigo-50 transition-colors"
            @click="generateAll"
        >⚙️ قرعه‌کشی همهٔ گروه‌ها</button>

        <span class="text-[11px] text-slate-400 ms-auto">
          تیم بدون گروه: {{ unassignedTeams.length }}
        </span>
      </div>

      <div v-if="!groups.length" class="flex items-center justify-center h-32 text-slate-400 text-sm bg-slate-50 rounded-2xl border border-dashed border-slate-200">
        هنوز گروهی ساخته نشده است.
      </div>

      <div v-else class="grid gap-3 md:grid-cols-2">
        <div
            v-for="row in groupRows"
            :key="row.id"
            class="bg-white rounded-2xl border shadow-sm overflow-hidden"
            :class="groupStyle(row.id).ring"
        >
          <div class="flex items-center justify-between px-4 py-2.5 border-b border-slate-100 bg-slate-50/70">
            <div class="flex items-center gap-2">
              <span class="text-xs rounded-full px-2.5 py-0.5 font-medium" :class="groupStyle(row.id).chip">
                {{ row.name }}
              </span>
              <span class="text-[11px] text-slate-400">{{ row.teamIds.length }} تیم</span>
            </div>
            <button
                class="text-[11px] text-slate-400 hover:text-rose-600 transition-colors"
                @click="store.removeGroup(t.id, row.id)"
            >حذف گروه</button>
          </div>

          <div class="p-4 flex flex-col gap-3">
            <!-- تیم‌ها -->
            <div class="flex flex-wrap gap-1.5">
              <span
                  v-for="tid in row.teamIds"
                  :key="tid"
                  class="inline-flex items-center gap-1 text-xs rounded-full px-2.5 py-1 bg-slate-100 text-slate-600"
              >
                {{ teamName(tid) }}
              </span>
              <span v-if="!row.teamIds.length" class="text-xs text-slate-300">تیمی اضافه نشده</span>
            </div>

            <select
                class="text-xs border border-slate-200 rounded-lg px-2 py-1.5 bg-white text-slate-600 focus:outline-none focus:border-indigo-400 w-full"
                :disabled="!unassignedTeams.length"
                @change="assignTeam(row.id, $event)"
            >
              <option value="">
                {{ unassignedTeams.length ? '+ افزودن تیم به این گروه' : 'همهٔ تیم‌ها گروه‌بندی شده‌اند' }}
              </option>
              <option v-for="tm in unassignedTeams" :key="tm.id" :value="tm.id">{{ tm.name }}</option>
            </select>

            <!-- زمین‌های مجاز -->
            <div>
              <div class="text-[11px] text-slate-400 mb-1.5">
                زمین‌های مجاز (حداکثر {{ MAX_COURTS_PER_GROUP }})
              </div>
              <div class="flex gap-1.5">
                <button
                    v-for="c in COURTS"
                    :key="c"
                    class="text-xs rounded-lg px-3 py-1.5 border transition-all"
                    :class="row.courts.includes(c)
                      ? 'bg-indigo-500 border-indigo-500 text-white'
                      : 'border-slate-200 text-slate-500 hover:border-slate-300'"
                    @click="toggleCourt(row.id, c)"
                >🏟 {{ courtLabel(c) }}</button>
              </div>
              <div v-if="!row.courts.length" class="text-[10px] text-slate-400 mt-1">
                خالی = همهٔ زمین‌ها مجاز
              </div>
            </div>

            <!-- آمار بازی‌ها -->
            <div class="flex items-center justify-between gap-3 pt-1">
              <div class="flex-1">
                <div class="flex items-center justify-between text-[11px] text-slate-500 mb-1">
                  <span>{{ row.scheduled }}/{{ row.matchCount }} چیده‌شده</span>
                  <span>{{ row.done }} انجام‌شده</span>
                </div>
                <div class="h-1.5 bg-slate-100 rounded-full overflow-hidden">
                  <div
                      class="h-full rounded-full transition-all"
                      :class="groupStyle(row.id).bar"
                      :style="`width:${row.matchCount ? (row.scheduled / row.matchCount * 100) : 0}%`"
                  />
                </div>
              </div>
              <button
                  class="text-xs rounded-lg px-3 py-1.5 border border-slate-200 text-slate-600 hover:border-indigo-300 hover:text-indigo-600 transition-all whitespace-nowrap"
                  @click="generate(row.id)"
              >
                {{ row.matchCount ? '↻ قرعه‌کشی مجدد' : '⚙️ قرعه‌کشی' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- ══════════ زمان‌بندی ══════════ -->
    <template v-else>
      <div v-if="!matches.length" class="flex items-center justify-center h-32 text-slate-400 text-sm bg-slate-50 rounded-2xl border border-dashed border-slate-200">
        ابتدا در تب گروه‌بندی قرعه‌کشی را انجام دهید.
      </div>

      <template v-else>
        <!-- آمار زمین‌ها -->
        <div class="grid gap-3" :style="`grid-template-columns: repeat(${courtLoads.length}, minmax(0,1fr))`">
          <div v-for="cl in courtLoads" :key="cl.court" class="bg-slate-800 text-white rounded-xl p-3 text-center">
            <div class="text-xs text-slate-400 mb-1">🏟 زمین {{ courtLabel(cl.court) }}</div>
            <div class="text-2xl font-bold">{{ cl.total }}</div>
            <div class="text-xs text-slate-400">بازی ({{ cl.done }} انجام شده)</div>
            <div class="mt-2 h-1.5 bg-slate-600 rounded-full overflow-hidden">
              <div class="h-full bg-indigo-400 rounded-full transition-all"
                   :style="`width:${cl.total ? (cl.done / cl.total * 100) : 0}%`" />
            </div>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <button
              class="text-xs rounded-lg px-3 py-2 bg-slate-800 text-white hover:bg-slate-700 transition-colors"
              @click="runAuto"
          >✨ چیدمان خودکار</button>
          <span class="text-[11px] text-slate-400">
            بازی‌های چیده‌نشده: {{ unscheduledCount }}
          </span>
        </div>

        <div class="grid gap-4 lg:grid-cols-[1fr_260px] items-start">
          <!-- جدول سانس × زمین -->
          <div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
            <div class="grid" :style="`grid-template-columns: 90px repeat(${COURTS.length}, minmax(0,1fr))`">
              <div class="bg-slate-50 border-b border-slate-200 px-3 py-2.5 text-xs text-slate-500 font-medium">سانس</div>
              <div
                  v-for="c in COURTS"
                  :key="`h-${c}`"
                  class="bg-slate-50 border-b border-s border-slate-200 px-3 py-2.5 text-xs text-slate-600 font-medium text-center"
              >🏟 زمین {{ courtLabel(c) }}</div>

              <template v-for="s in slots" :key="s.id">
                <div class="border-b border-slate-100 px-3 py-3 text-xs text-slate-500 flex items-center bg-slate-50/50">
                  {{ s.label || `سانس ${s.id}` }}
                </div>
                <draggable
                    v-for="c in COURTS"
                    :key="`${s.id}-${c}`"
                    :list="localGrid[s.id]![c]"
                    :group="{ name: 'matches' }"
                    item-key="id"
                    tag="div"
                    :data-slot="s.id"
                    :data-court="c"
                    :animation="180"
                    :force-fallback="true"
                    filter=".is-locked"
                    ghost-class="drag-ghost"
                    chosen-class="drag-chosen"
                    fallback-class="drag-fallback"
                    class="border-b border-s border-slate-100 p-2 min-h-[104px]"
                    @end="onDragEnd"
                >
                  <template #item="{ element: m }">
                    <div
                        :data-id="m.id"
                        class="rounded-xl border bg-white shadow-sm select-none transition-all duration-150 p-2"
                        :class="isFinished(m)
                          ? 'is-locked opacity-70 cursor-not-allowed border-slate-100'
                          : 'cursor-grab hover:shadow-md hover:border-indigo-200 border-slate-100'"
                    >
                      <div class="flex items-center justify-between mb-1.5">
                        <span class="text-[10px] rounded-full px-2 py-0.5" :class="groupStyle(m.groupId).chip">
                          {{ groupName(m.groupId) }}
                        </span>
                        <span class="font-mono text-[10px] bg-slate-100 text-slate-500 rounded-full px-2 py-0.5">
                          دور {{ m.round }}
                        </span>
                      </div>

                      <div class="space-y-1">
                        <div class="flex items-center gap-1.5">
                          <input
                              type="number" min="0" inputmode="numeric"
                              class="w-9 text-center text-[11px] border border-slate-200 rounded-md py-0.5 focus:outline-none focus:border-indigo-400"
                              :value="m.scoreA ?? ''"
                              @change="onScore(m, 'A', $event)"
                          />
                          <span class="text-xs text-slate-700 truncate">{{ teamName(m.teamAId) }}</span>
                        </div>
                        <div class="flex items-center gap-1.5">
                          <input
                              type="number" min="0" inputmode="numeric"
                              class="w-9 text-center text-[11px] border border-slate-200 rounded-md py-0.5 focus:outline-none focus:border-indigo-400"
                              :value="m.scoreB ?? ''"
                              @change="onScore(m, 'B', $event)"
                          />
                          <span class="text-xs text-slate-700 truncate">{{ teamName(m.teamBId) }}</span>
                        </div>
                      </div>

                      <button
                          class="mt-1.5 text-[10px] text-slate-400 hover:text-rose-600 transition-colors"
                          @click="unplace(m.id)"
                      >↩️ بازگشت به استخر</button>
                    </div>
                  </template>
                </draggable>
              </template>
            </div>
          </div>

          <!-- استخر بازی‌ها -->
          <div class="rounded-2xl border border-slate-200 bg-slate-50/80 shadow-sm overflow-hidden flex flex-col lg:sticky lg:top-4">
            <div class="flex items-center justify-between px-3 py-2.5 border-b border-slate-200 bg-white/80">
              <span class="font-semibold text-slate-700 text-sm">📋 چیده‌نشده</span>
              <span class="text-[11px] text-slate-500 bg-slate-100 rounded-full px-2 py-0.5">
                {{ localPool.length }}
              </span>
            </div>

            <div class="px-2.5 pt-2.5">
              <select
                  v-model="poolFilter"
                  class="w-full text-xs border border-slate-200 rounded-lg px-2 py-1.5 bg-white text-slate-600 focus:outline-none focus:border-indigo-400"
              >
                <option value="">همهٔ گروه‌ها</option>
                <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
              </select>
            </div>

            <draggable
                :list="localPool"
                :group="{ name: 'matches' }"
                item-key="id"
                tag="div"
                :animation="180"
                :force-fallback="true"
                ghost-class="drag-ghost"
                chosen-class="drag-chosen"
                fallback-class="drag-fallback"
                class="flex flex-col gap-2 p-2.5 min-h-[420px] flex-1 overflow-y-auto max-h-[70vh]"
                @end="onDragEnd"
            >
              <template #item="{ element: m }">
                <div
                    :data-id="m.id"
                    class="bg-white rounded-xl border border-slate-100 shadow-sm select-none cursor-grab hover:shadow-md hover:border-indigo-200 transition-all duration-150 p-2.5"
                >
                  <div class="flex items-center justify-between mb-1.5">
                    <span class="text-[10px] rounded-full px-2 py-0.5" :class="groupStyle(m.groupId).chip">
                      {{ groupName(m.groupId) }}
                    </span>
                    <span class="font-mono text-[10px] bg-slate-100 text-slate-500 rounded-full px-2 py-0.5">
                      دور {{ m.round }}
                    </span>
                  </div>
                  <div class="text-xs text-slate-700 truncate">{{ teamName(m.teamAId) }}</div>
                  <div class="text-[10px] text-slate-300 my-0.5">در برابر</div>
                  <div class="text-xs text-slate-700 truncate">{{ teamName(m.teamBId) }}</div>
                </div>
              </template>
            </draggable>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>

<style>
.drag-ghost {
  opacity: 0.35;
  background: #eef2ff;
  border: 2px dashed #818cf8 !important;
}

.drag-fallback {
  transform: rotate(2deg);
  box-shadow: 0 14px 26px rgba(15, 23, 42, 0.18);
  cursor: grabbing;
}

.drag-chosen {
  cursor: grabbing;
}
</style>
