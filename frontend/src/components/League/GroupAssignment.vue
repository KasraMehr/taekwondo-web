<!-- components/league/GroupAssignment.vue -->
<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import type { Athlete, AthleteDraft, LeagueAthlete, LeagueGroup } from '../../types'
import { useLeagueStore } from '../../stores/league'
import { useLeagueAthletePool } from '../../composables/useLeagueAthletePool'
import { buildGroupResolver } from '../../utils/groupResolver'
import AthleteFormModal from './AthleteFormModal.vue'
import { loadWebLeagues, webApi } from '../../webApi'

const props = defineProps<{ leagueId: string }>()

const leagueStore = useLeagueStore()
const pool = useLeagueAthletePool(toRef(props, 'leagueId'))

const league = computed(() => leagueStore.leagues.find((l) => l.id === props.leagueId) ?? null)
const groups = computed<LeagueGroup[]>(() =>
    [...(league.value?.season.groups ?? [])].sort((a, b) => a.order - b.order)
)
const resolver = computed(() => buildGroupResolver(groups.value))

/** رکوردهای خام ورزشکار برای ویرایش (استخر شکل نمایشی دارد، نه شکل ذخیره‌سازی) */
const athleteRecords = computed(
    () => new Map<string, LeagueAthlete>((league.value?.athletes ?? []).map((a) => [a.id, a]))
)
/** منبع باشگاه‌ها؛ اگر جای دیگری نگه‌داری می‌شود همین خط را عوض کن */
const clubs = computed(() => league.value?.clubs ?? [])

const isFree = (a: Athlete) => !a.club || a.club === 'آزاد'
const errorMessage = (e: unknown) => (e instanceof Error ? e.message : 'خطای نامشخص رخ داد.')

/** groupId → ورزشکاران (یک‌بار محاسبه، نه صدا زدن تابع در تمپلیت) */
const membersByGroup = computed(() => {
  const map = new Map<string, Athlete[]>(groups.value.map((g) => [g.id, [] as Athlete[]]))
  for (const a of pool.value) {
    const gid = resolver.value.groupIdOf(a)
    if (gid) map.get(gid)?.push(a)
  }
  return map
})

const unassignedCount = computed(
    () => pool.value.filter((a) => !resolver.value.groupIdOf(a)).length
)

/* ── فیلترها ── */
const search = ref('')
const weightFilter = ref('')
const clubFilter = ref('')
const onlyUnassigned = ref(true)

const weightOptions = computed(() =>
    [...new Set(pool.value.map((a) => a.weightCategory).filter(Boolean))].sort((a, b) =>
        a.localeCompare(b, 'fa')
    )
)
const clubOptions = computed(() =>
    [...new Set(pool.value.filter((a) => !isFree(a)).map((a) => a.club))].sort((a, b) =>
        a.localeCompare(b, 'fa')
    )
)

const filteredPool = computed(() => {
  const q = search.value.trim().toLowerCase()
  return pool.value.filter((a) => {
    if (q && !a.name.toLowerCase().includes(q)) return false
    if (weightFilter.value && a.weightCategory !== weightFilter.value) return false
    if (clubFilter.value === '__free__' && !isFree(a)) return false
    if (clubFilter.value && clubFilter.value !== '__free__' && a.club !== clubFilter.value) return false
    if (onlyUnassigned.value && resolver.value.groupIdOf(a)) return false
    return true
  })
})

/* ── انتخاب و انتقال ── */
const selected = ref<Set<string>>(new Set())
const byId = computed(() => new Map(pool.value.map((a) => [a.id, a])))

const toRefs = (ids: string[]) =>
    ids.map((id) => byId.value.get(id)).filter((a): a is Athlete => !!a)

function toggle(id: string) {
  const next = new Set(selected.value)
  next.has(id) ? next.delete(id) : next.add(id)
  selected.value = next
}

function deselect(id: string) {
  if (!selected.value.has(id)) return
  const next = new Set(selected.value)
  next.delete(id)
  selected.value = next
}

async function refreshLeague() {
  leagueStore.replaceFromBackend(await loadWebLeagues())
}

function athletePayload(record: LeagueAthlete, patch: Partial<{ groupId: string | null; teamId: string | null }> = {}) {
  const raw = record as LeagueAthlete & { teamId?: string | null; coachName?: string; ranking?: number | null }
  return {
    teamId: patch.teamId !== undefined ? patch.teamId : raw.teamId ?? null,
    groupId: patch.groupId !== undefined ? patch.groupId : record.groupId,
    weightCategory: record.weightCategory,
    coachName: raw.coachName ?? '', ranking: raw.ranking ?? null, active: record.isActive,
  }
}

async function move(groupId: string | null, ids: string[] = [...selected.value]) {
  const refs = toRefs(ids)
  if (!refs.length) return
  try {
    await Promise.all(refs.map(a => {
      const record = athleteRecords.value.get(a.id)
      return record ? webApi().call(`/leagues/${props.leagueId}/athletes/${record.id}`, 'PUT', athletePayload(record, { groupId })) : Promise.resolve()
    }))
    await refreshLeague()
    selected.value = new Set()
    listError.value = ''
  } catch (e) { listError.value = errorMessage(e) }
}

const overCapacity = (g: LeagueGroup) =>
    !!g.capacity && (membersByGroup.value.get(g.id)?.length ?? 0) > g.capacity

/* ── درگ ── */
const dragOver = ref<string | null>(null)

function onDragStart(e: DragEvent, id: string) {
  const ids = selected.value.has(id) ? [...selected.value] : [id]
  e.dataTransfer?.setData('text/plain', JSON.stringify(ids))
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
}

function onDrop(e: DragEvent, groupId: string | null) {
  e.preventDefault()
  dragOver.value = null
  const raw = e.dataTransfer?.getData('text/plain')
  if (!raw) return
  try {
    move(groupId, JSON.parse(raw) as string[])
  } catch { /* payload نامعتبر */ }
}

/* ── CRUD ورزشکار ── */
const formOpen = ref(false)
const editing = ref<LeagueAthlete | null>(null)
const formError = ref('')
const listError = ref('')

function openCreate() {
  editing.value = null
  formError.value = ''
  formOpen.value = true
}

function openEdit(id: string) {
  const record = athleteRecords.value.get(id)
  if (!record) {
    listError.value = 'رکورد این ورزشکار یافت نشد.'
    return
  }
  editing.value = record
  formError.value = ''
  formOpen.value = true
}

function closeForm() {
  formOpen.value = false
  editing.value = null
  formError.value = ''
}

async function onSubmit(draft: AthleteDraft) {
  try {
    const club = clubs.value.find(c => c.id === draft.clubId) as (typeof clubs.value[number] & { teamId?: string }) | undefined
    const teamId = club?.teamId ?? null
    const groupId = club?.groupId ?? null
    if (editing.value) {
      const raw = editing.value as LeagueAthlete & { profileId?: string }
      if (!raw.profileId) throw new Error('پروفایل ورزشکار پیدا نشد')
      await webApi().call(`/athletes/${raw.profileId}`, 'PUT', { name: draft.name, gender: draft.gender, clubId: draft.clubId })
      await webApi().call(`/leagues/${props.leagueId}/athletes/${editing.value.id}`, 'PUT', {
        ...athletePayload(editing.value, { teamId, groupId }), weightCategory: draft.weightCategory,
      })
    } else {
      await webApi().call(`/leagues/${props.leagueId}/athletes`, 'POST', {
        name: draft.name, gender: draft.gender, clubId: draft.clubId,
        teamId, groupId, weightCategory: draft.weightCategory, coachName: '', ranking: null,
      })
    }
    await refreshLeague()
    listError.value = ''
    closeForm()
  } catch (e) {
    formError.value = errorMessage(e)
  }
}

/** حذف نرم؛ اول از تخصیص گروه خارج می‌شود تا ارجاع بی‌صاحب نماند */
async function remove(a: Athlete) {
  if (!window.confirm(`«${a.name}» حذف شود؟ سوابق تورنمنت‌ها حفظ می‌شود.`)) return
  try {
    await webApi().call(`/leagues/${props.leagueId}/athletes/${a.id}`, 'DELETE')
    await refreshLeague()
    deselect(a.id)
    listError.value = ''
  } catch (e) {
    listError.value = errorMessage(e)
  }
}
</script>

<template>
  <div v-if="league" dir="rtl" class="grid gap-4 lg:grid-cols-[340px_1fr]">
    <!-- استخر -->
    <section
        class="rounded-xl border-2 bg-white p-4 transition"
        :class="dragOver === '__pool__' ? 'border-amber-400 bg-amber-50/40' : 'border-slate-200'"
        @dragover.prevent="dragOver = '__pool__'"
        @dragleave="dragOver = null"
        @drop="onDrop($event, null)"
    >
      <header class="mb-3 flex items-center justify-between gap-2">
        <h3 class="font-semibold text-slate-800">ورزشکاران</h3>
        <div class="flex items-center gap-2">
          <span class="text-xs text-slate-500">{{ unassignedCount }} تخصیص‌نیافته</span>
          <button
              type="button"
              class="rounded-lg bg-emerald-600 px-2 py-1 text-xs font-medium text-white hover:bg-emerald-700"
              @click="openCreate"
          >
            + افزودن
          </button>
        </div>
      </header>

      <p v-if="listError" role="alert"
         class="mb-3 rounded-lg bg-rose-50 px-3 py-2 text-xs text-rose-700">
        {{ listError }}
      </p>

      <div class="space-y-2">
        <input
            v-model="search"
            type="search"
            aria-label="جستجوی نام ورزشکار"
            placeholder="جستجوی نام…"
            class="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
        />
        <div class="grid grid-cols-2 gap-2">
          <select v-model="weightFilter" aria-label="فیلتر وزن"
                  class="rounded-lg border border-slate-300 px-2 py-2 text-sm">
            <option value="">همه وزن‌ها</option>
            <option v-for="w in weightOptions" :key="w" :value="w">{{ w }}</option>
          </select>
          <select v-model="clubFilter" aria-label="فیلتر باشگاه"
                  class="rounded-lg border border-slate-300 px-2 py-2 text-sm">
            <option value="">همه باشگاه‌ها</option>
            <option value="__free__">آزاد</option>
            <option v-for="c in clubOptions" :key="c" :value="c">{{ c }}</option>
          </select>
        </div>
        <label class="flex items-center gap-2 text-sm text-slate-600">
          <input v-model="onlyUnassigned" type="checkbox" class="rounded border-slate-300" />
          فقط تخصیص‌نیافته‌ها
        </label>
      </div>

      <div class="mt-3 flex items-center gap-2 border-t border-slate-100 pt-3">
        <button type="button" class="rounded-lg bg-slate-100 px-2 py-1 text-xs hover:bg-slate-200"
                @click="selected = new Set(filteredPool.map((a) => a.id))">
          انتخاب همه
        </button>
        <button type="button" :disabled="!selected.size"
                class="rounded-lg bg-slate-100 px-2 py-1 text-xs hover:bg-slate-200 disabled:opacity-40"
                @click="selected = new Set()">
          پاک‌کردن
        </button>
        <span v-if="selected.size" class="ms-auto text-xs font-medium text-emerald-700">
          {{ selected.size }} انتخاب‌شده
        </span>
      </div>

      <ul class="mt-3 max-h-[28rem] space-y-1 overflow-y-auto">
        <li v-for="a in filteredPool" :key="a.id">
          <div
              draggable="true"
              class="flex items-center gap-1 rounded-lg border px-2 py-1.5"
              :class="selected.has(a.id)
                ? 'border-emerald-400 bg-emerald-50'
                : 'border-slate-200 hover:bg-slate-50'"
              @dragstart="onDragStart($event, a.id)"
          >
            <button
                type="button"
                :aria-pressed="selected.has(a.id)"
                class="flex min-w-0 flex-1 items-center gap-2 rounded px-1 py-0.5 text-right text-sm"
                @click="toggle(a.id)"
            >
              <span class="flex-1 truncate font-medium text-slate-800">{{ a.name }}</span>
              <span class="shrink-0 text-xs text-slate-500">{{ a.weightCategory }}</span>
              <span class="shrink-0 rounded px-1.5 py-0.5 text-[11px]"
                    :class="isFree(a) ? 'bg-amber-100 text-amber-700' : 'bg-slate-100 text-slate-600'">
                {{ isFree(a) ? 'آزاد' : a.club }}
              </span>
            </button>

            <button type="button" :aria-label="`ویرایش ${a.name}`" title="ویرایش"
                    class="shrink-0 rounded px-1.5 py-0.5 text-xs text-slate-500 hover:bg-slate-100 hover:text-slate-700"
                    @click="openEdit(a.id)">✎</button>
            <button type="button" :aria-label="`حذف ${a.name}`" title="حذف"
                    class="shrink-0 rounded px-1.5 py-0.5 text-xs text-rose-600 hover:bg-rose-50"
                    @click="remove(a)">🗑</button>
          </div>
        </li>
        <li v-if="!filteredPool.length"
            class="rounded-lg bg-slate-50 px-3 py-6 text-center text-sm text-slate-500">
          موردی مطابق فیلترها نیست.
        </li>
      </ul>
    </section>

    <!-- گروه‌ها -->
    <section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <article
          v-for="g in groups"
          :key="g.id"
          class="rounded-xl border-2 bg-white p-4 transition"
          :class="dragOver === g.id ? 'border-emerald-400 bg-emerald-50/40' : 'border-slate-200'"
          @dragover.prevent="dragOver = g.id"
          @dragleave="dragOver = null"
          @drop="onDrop($event, g.id)"
      >
        <header class="mb-2 flex items-center justify-between">
          <h4 class="font-semibold text-slate-800">{{ g.name }}</h4>
          <span class="text-xs" :class="overCapacity(g) ? 'font-medium text-rose-600' : 'text-slate-500'">
            {{ membersByGroup.get(g.id)?.length ?? 0 }}<template v-if="g.capacity"> / {{ g.capacity }}</template>
          </span>
        </header>

        <p class="mb-3 truncate text-[11px] text-slate-400" :title="g.clubs.join('، ')">
          {{ g.clubs.length ? g.clubs.join('، ') : 'بدون باشگاه' }}
        </p>

        <button
            type="button"
            :disabled="!selected.size"
            class="mb-3 w-full rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-emerald-700 disabled:opacity-40"
            @click="move(g.id)"
        >
          انتقال {{ selected.size || '' }} انتخاب‌شده
        </button>

        <ul class="max-h-64 space-y-1 overflow-y-auto">
          <li v-for="a in membersByGroup.get(g.id) ?? []" :key="a.id"
              draggable="true"
              class="flex items-center gap-1 rounded-lg bg-slate-50 px-2 py-1.5 text-sm"
              @dragstart="onDragStart($event, a.id)">
            <span class="flex-1 truncate text-slate-800">{{ a.name }}</span>
            <span class="shrink-0 text-xs text-slate-500">{{ a.weightCategory }}</span>
            <span v-if="resolver.isExplicit(a)"
                  class="shrink-0 rounded bg-emerald-100 px-1 text-[10px] text-emerald-700">دستی</span>
            <button type="button" :aria-label="`ویرایش ${a.name}`" title="ویرایش"
                    class="shrink-0 rounded px-1 text-slate-500 hover:bg-slate-200 hover:text-slate-700"
                    @click="openEdit(a.id)">✎</button>
            <button type="button" :aria-label="`خارج‌کردن ${a.name} از ${g.name}`" title="خارج‌کردن از گروه"
                    class="shrink-0 rounded px-1.5 text-rose-600 hover:bg-rose-50"
                    @click="move(null, [a.id])">✕</button>
          </li>
          <li v-if="!(membersByGroup.get(g.id) ?? []).length"
              class="py-6 text-center text-xs text-slate-400">خالی — اینجا رها کن</li>
        </ul>
      </article>

      <p v-if="!groups.length" class="text-sm text-slate-500">
        برای این فصل گروهی تعریف نشده است.
      </p>
    </section>

    <AthleteFormModal
        :open="formOpen"
        :gender="league.gender"
        :age-category="league.ageCategory"
        :athlete="editing"
        :clubs="clubs"
        :error="formError"
        @submit="onSubmit"
        @close="closeForm"
    />
  </div>
</template>
