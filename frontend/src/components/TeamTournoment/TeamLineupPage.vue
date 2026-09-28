<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useTeamTournamentStore } from '../../stores/teamTournament'
import LineupEditor from './LineupEditor.vue'

const store = useTeamTournamentStore()

type Bout = {
  matchId: string
  homeTeamId: string
  awayTeamId: string
  groupId: string | null
  round: number | null
  status?: string
}

const tournamentId = computed(() => store.current?.id ?? '')
const teams = computed(() => store.current?.teams ?? [])
const groups = computed(() => store.current?.groups ?? [])

/** فقط بازی‌هایی که هر دو تیمشان مشخص است، وگرنه ensureEncounter خطا می‌دهد */
const bouts = computed<Bout[]>(() => {
  const t: any = store.current
  if (!t) return []
  const encs: any[] = t.encounters ?? []

  return (t.matches ?? [])
      .filter((m: any) => m.teamAId && m.teamBId)
      .map((m: any) => ({
        matchId: String(m.id),
        homeTeamId: m.teamAId,
        awayTeamId: m.teamBId,
        groupId: m.groupId ?? null,
        round: m.round ?? null,
        status:
            encs.find((e) => e.id === m.id || String(e.id).endsWith(String(m.id)))?.status ??
            m.status,
      }))
})

const matchId = ref('')

const teamName = (id?: string | null) =>
    id ? (teams.value.find((t) => t.id === id)?.name ?? '—') : '—'

const groupName = (id?: string | null) =>
    id ? (groups.value.find((g) => g.id === id)?.name ?? '') : ''

const currentBout = computed<Bout | null>(
    () => bouts.value.find((b) => b.matchId === matchId.value) ?? null,
)

/** تیم راست = میزبان، تیم چپ = میهمان */
const rightTeamId = computed(() => currentBout.value?.homeTeamId ?? '')
const leftTeamId = computed(() => currentBout.value?.awayTeamId ?? '')

const boutLabel = (b: Bout) => {
  const g = groupName(b.groupId)
  const r = b.round != null ? ` — دور ${b.round}` : ''
  const gp = g ? `${g}${r} · ` : ''
  return `${gp}${teamName(b.homeTeamId)} × ${teamName(b.awayTeamId)}`
}

watch(
    bouts,
    (list) => {
      if (!list.length) { matchId.value = ''; return }
      if (!list.some((b) => b.matchId === matchId.value)) {
        matchId.value = (list.find((b) => b.status !== 'completed') ?? list[0]).matchId
      }
    },
    { immediate: true },
)
</script>

<template>
  <div dir="rtl">
    <div v-if="!tournamentId" class="rounded-xl border border-slate-200 bg-white p-8 text-center text-sm text-slate-600">
      مسابقه‌ای انتخاب نشده است.
    </div>

    <div v-else-if="!teams.length" class="rounded-xl border border-slate-200 bg-white p-8 text-center text-sm text-slate-600">
      هنوز تیمی ثبت نشده است. ابتدا تیم‌ها را اضافه کنید.
    </div>

    <div v-else-if="!bouts.length" class="rounded-xl border border-slate-200 bg-white p-8 text-center text-sm text-slate-600">
      بازیِ آماده‌ای وجود ندارد. ابتدا از بخش گروه‌ها بازی‌ها را تولید کنید و هر دو تیم هر بازی مشخص باشد.
    </div>

    <div v-else class="space-y-4">
      <!-- فقط انتخاب بازی؛ دیگر انتخاب تیم لازم نیست -->
      <div class="flex flex-wrap items-center gap-3 rounded-xl border border-slate-200 bg-white p-4">
        <label class="text-sm text-slate-600">بازی:</label>
        <select v-model="matchId" class="rounded-lg border border-slate-200 px-3 py-2 text-sm">
          <option v-for="b in bouts" :key="b.matchId" :value="b.matchId">
            {{ boutLabel(b) }}
          </option>
        </select>
      </div>

      <!-- دو تیم کنار هم: در RTL ستون اول سمت راست رندر می‌شود -->
      <div v-if="currentBout" class="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <!-- تیم راست (میزبان) -->
        <section class="min-w-0 rounded-xl border border-slate-200 bg-white p-4">
          <div class="mb-3 flex items-center justify-between border-b border-slate-100 pb-2">
            <span class="font-semibold text-rose-600">{{ teamName(rightTeamId) }}</span>
            <span class="text-xs text-slate-400">تیم قرمز</span>
          </div>
          <LineupEditor
              :key="`${matchId}::${rightTeamId}`"
              side="right"
              :tournament-id="tournamentId"
              :match-id="matchId"
              :team-id="rightTeamId"
          />
        </section>

        <!-- تیم چپ (میهمان) -->
        <section class="min-w-0 rounded-xl border border-slate-200 bg-white p-4">
          <div class="mb-3 flex items-center justify-between border-b border-slate-100 pb-2">
            <span class="font-semibold text-indigo-600">{{ teamName(leftTeamId) }}</span>
            <span class="text-xs text-slate-400">تیم آبی</span>
          </div>
          <LineupEditor
              :key="`${matchId}::${leftTeamId}`"
              side="left"
              :tournament-id="tournamentId"
              :match-id="matchId"
              :team-id="leftTeamId"
          />
        </section>
      </div>

      <div v-else class="rounded-xl border border-slate-200 bg-white p-8 text-center text-sm text-slate-600">
        یک بازی را انتخاب کنید.
      </div>
    </div>
  </div>
</template>
