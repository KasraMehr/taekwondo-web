<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useLeagueStore } from "../../stores/league";
import { useTournamentStore } from "../../stores/tournament";
import { useUiStore } from "../../stores/ui";
import type {Athlete, LeagueGroup, Tournament, Week} from "../../types";
import { tournamentSeedsForGroup } from "../../utils/leagueRoster";
import { loadWebLeagues, loadWebTournaments, webApi } from "../../webApi";

const leagueStore = useLeagueStore();
const tournamentStore = useTournamentStore();
const ui = useUiStore();

const currentLeague = computed(() => leagueStore.currentLeague);

/* ---------------- گروه فعال ---------------- */

const groups = computed<LeagueGroup[]>(() =>
    [...(currentLeague.value?.season.groups ?? [])].sort((a, b) => a.order - b.order)
);

const activeGroupId = ref<string | null>(null);

const activeGroup = computed<LeagueGroup | null>(
    () => groups.value.find((g) => g.id === activeGroupId.value) ?? null
);

watch(
    groups,
    (list) => {
      if (!list.some((g) => g.id === activeGroupId.value)) {
        activeGroupId.value = list[0]?.id ?? null;
      }
    },
    { immediate: true }
);

/* ---------------- هفته‌ها ---------------- */

const stages = computed(() =>
    [...(currentLeague.value?.season.stages ?? [])].sort((a, b) => a.order - b.order)
);

// مرحله ۱ منبع حقیقت است؛ مراحل بعدی فقط تیم‌های صعودکننده را دارند
const activeStage = computed(() => stages.value[0] ?? null);

type StageWeek = Week & { stageOrder: number };

const weeks = computed<StageWeek[]>(() => {
  const stage = activeStage.value;
  if (!stage) return [];
  return stage.weeks.map((w) => ({ ...w, stageOrder: stage.order }));
});

/* ---------------- بازیکنان گروه ---------------- */

function athletesOfGroup(groupId: string | null): Omit<Athlete, "id">[] {
  const league = currentLeague.value;
  return league ? tournamentSeedsForGroup(league, groupId) : [];
}

const activeGroupAthletes = computed(() => athletesOfGroup(activeGroupId.value));

/* ---------------- نقشه‌ی تورنمنت‌ها (یک‌بار محاسبه) ---------------- */

const tournamentsByWeekGroup = computed<Map<string, Tournament>>(() => {
  const leagueId = currentLeague.value?.id;
  const stageOrder = activeStage.value?.order;
  const map = new Map<string, Tournament>();
  if (!leagueId || stageOrder == null) return map;

  for (const t of tournamentStore.tournaments) {
    if (t.leagueId !== leagueId || !t.weekId || !t.groupId) continue;
    // تورنمنت‌های قدیمی ممکن است stageOrder نداشته باشند
    if (t.stageOrder != null && t.stageOrder !== stageOrder) continue;
    map.set(`${t.weekId}::${t.groupId}`, t);
  }
  return map;
});


function tournamentOf(weekId: string, groupId: string | null | undefined): Tournament | null {
  if (!groupId) return null;
  return tournamentsByWeekGroup.value.get(`${weekId}::${groupId}`) ?? null;
}

/* ---------------- ساخت تورنمنت ---------------- */

const showCreateTournament = ref(false);
const submitting = ref(false);
const newTournamentName = ref("");
const newTournamentCourts = ref(2);

function todayLocal(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
      d.getDate()
  ).padStart(2, "0")}`;
}
const newTournamentDate = ref(todayLocal());

// هفته و گروه در لحظه‌ی باز شدن فرم قفل می‌شوند
const draftWeek = ref<StageWeek | null>(null);
const draftGroupId = ref<string | null>(null);

const draftGroup = computed<LeagueGroup | null>(
    () => groups.value.find((g) => g.id === draftGroupId.value) ?? null
);

const draftAthletes = computed(() => athletesOfGroup(draftGroupId.value));

function createTournamentForWeek(week: StageWeek) {
  const group = activeGroup.value;
  if (!group) {
    alert("اول یک گروه بساز");
    return;
  }
  draftWeek.value = week;
  draftGroupId.value = group.id;
  newTournamentName.value = `${week.name} — ${group.name}`;
  newTournamentDate.value = todayLocal();
  showCreateTournament.value = true;
}

function closeForm() {
  showCreateTournament.value = false;
  draftWeek.value = null;
  draftGroupId.value = null;
}

async function submitTournament() {
  if (submitting.value) return;

  const league = currentLeague.value;
  const week = draftWeek.value;
  const group = draftGroup.value;
  if (!league || !week || !group) return;

  if (tournamentOf(week.id, group.id)) {
    alert("برای این هفته و گروه از قبل تورنمنت ساخته شده");
    closeForm();
    return;
  }

  const seeds = draftAthletes.value;
  if (
      !seeds.length &&
      !confirm("هیچ بازیکنی از روستر این گروه پیدا نشد. تورنمنت خالی ساخته شود؟")
  ) {
    return;
  }

  submitting.value = true;
  try {
    const stage = league.season.stages.find(item => item.order === week.stageOrder)
    if (!stage) throw new Error('مرحله لیگ پیدا نشد')
    const tournament = await webApi().call<any>(`/leagues/${league.id}/tournaments`, 'POST', {
      name: newTournamentName.value.trim() || `${week.name} — ${group.name}`,
      date: newTournamentDate.value, courts: newTournamentCourts.value,
      stageId: stage.id, weekId: week.id, groupId: group.id,
    })
    if (!tournament) {
      alert("ساخت تورنمنت ناموفق بود");
      return;
    }

    const tournaments = await loadWebTournaments()
    tournamentStore.tournaments.splice(0, tournamentStore.tournaments.length, ...tournaments as any)
    tournamentStore.save()
    closeForm();
  } catch (e: any) {
    alert(e?.message || 'ساخت تورنومنت لیگ ناموفق بود')
  } finally {
    submitting.value = false;
  }
}

/* ---------------- قفل و ناوبری ---------------- */

async function toggleLockWeek(week: StageWeek) {
  const league = currentLeague.value;
  if (!league) return;
  try { await webApi().call(`/leagues/${league.id}/weeks/${week.id}/lock`, 'PUT', { locked: !week.locked }); leagueStore.replaceFromBackend(await loadWebLeagues()) }
  catch (e: any) { alert(e?.message || 'تغییر وضعیت هفته ناموفق بود') }
}

function openTournament(tournamentId: string) {
  tournamentStore.selectTournament(tournamentId);
  ui.goTournament();
}

async function recalculatePoints() {
  const league = currentLeague.value;
  const stage = activeStage.value;
  if (!league || !stage) return;
  try {
    const leagueTournaments = tournamentStore.tournaments.filter(t => t.leagueId === league.id)
    await Promise.all(leagueTournaments.map(t => webApi().call(`/leagues/${league.id}/tournaments/${t.id}/publish`, 'POST')))
    leagueStore.replaceFromBackend(await loadWebLeagues())
    alert("امتیازات محاسبه شد");
  } catch (e: any) { alert(e?.message || 'محاسبه امتیازات ناموفق بود') }
}
</script>

<template>
  <div>
    <!-- انتخاب گروه -->
    <div class="mb-6 flex flex-wrap items-center gap-3">
      <label class="text-sm font-medium text-slate-600">گروه:</label>

      <div class="flex flex-wrap gap-2">
        <button
            v-for="group in groups"
            :key="group.id"
            @click="activeGroupId = group.id"
            :class="[
            'rounded-lg px-4 py-2 text-sm font-medium transition',
            activeGroup?.id === group.id
              ? 'bg-emerald-600 text-white'
              : 'bg-slate-100 text-slate-600 hover:bg-slate-200',
          ]"
        >
          {{ group.name }}
          <span class="text-xs opacity-70">({{ group.clubs?.length ?? 0 }})</span>
        </button>
      </div>

      <p v-if="!groups.length" class="text-sm text-amber-600">
        هنوز گروهی ساخته نشده — از تب گروه‌ها شروع کن.
      </p>
    </div>

    <!-- خلاصه گروه فعال -->
    <div
        v-if="activeGroup"
        class="mb-6 rounded-2xl border border-slate-200 bg-slate-50 p-4"
    >
      <div class="flex items-center justify-between gap-4">
        <div>
          <h3 class="font-bold">{{ activeGroup.name }}</h3>
          <p class="mt-1 text-sm text-slate-500">
            {{ activeGroup.clubs?.length ?? 0 }} باشگاه ·
            {{ activeGroupAthletes.length }} بازیکن آماده ثبت
          </p>
        </div>

        <button
            @click="recalculatePoints"
            class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-blue-700"
        >
          🔄 محاسبه امتیاز
        </button>
      </div>

      <div class="mt-3 flex flex-wrap gap-2">
        <span
            v-for="club in activeGroup.clubs"
            :key="club"
            class="rounded-full bg-white px-3 py-1 text-xs text-slate-600 ring-1 ring-slate-200"
        >
          {{ club }}
        </span>
      </div>
    </div>

    <!-- هفته‌های گروه فعال -->
    <div v-if="activeGroup" class="space-y-4">
      <div
          v-for="week in weeks"
          :key="week.id"
          class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
      >
        <div class="flex items-center justify-between gap-4">
          <div>
            <h3 class="text-lg font-bold">
              {{ week.name }} — {{ activeGroup.name }}
            </h3>
            <p class="mt-1 text-sm text-slate-500">
              <span v-if="tournamentOf(week.id, activeGroup.id)">
                🏟 تورنمنت متصل است
              </span>
              <span v-else>
                ⚠️ هنوز تورنمنتی برای این گروه ساخته نشده
              </span>
            </p>
          </div>

          <div class="flex gap-2">
            <button
                @click="toggleLockWeek(week)"
                :class="[
                'rounded-lg px-3 py-1.5 text-sm font-medium transition',
                week.locked
                  ? 'bg-amber-100 text-amber-700 hover:bg-amber-200'
                  : 'bg-slate-100 text-slate-600 hover:bg-slate-200',
              ]"
            >
              {{ week.locked ? "🔒 قفل شده" : "🔓 باز" }}
            </button>

            <button
                v-if="tournamentOf(week.id, activeGroup.id)"
                @click="openTournament(tournamentOf(week.id, activeGroup.id)!.id)"
                class="rounded-lg bg-emerald-600 px-3 py-1.5 text-sm font-medium text-white transition hover:bg-emerald-700"
            >
              ورود به تورنمنت
            </button>

            <button
                v-else
                @click="createTournamentForWeek(week)"
                class="rounded-lg bg-blue-600 px-3 py-1.5 text-sm font-medium text-white transition hover:bg-blue-700"
            >
              + ساخت تورنمنت
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- فرم ساخت تورنمنت -->
    <div
        v-if="showCreateTournament"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    >
      <div class="w-full max-w-md rounded-2xl bg-white p-6">
        <h3 class="mb-1 text-lg font-bold">ایجاد تورنمنت</h3>
        <p class="mb-4 text-sm text-slate-500">
          {{ draftWeek?.name }} · {{ draftGroup?.name }} ·
          {{ draftAthletes.length }} بازیکن منتقل می‌شود
        </p>

        <p v-if="!draftAthletes.length" class="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700">
          هیچ بازیکنی از روستر باشگاه‌های این گروه پیدا نشد.
        </p>

        <div class="space-y-4">
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">
              نام تورنمنت
            </label>
            <input
                v-model="newTournamentName"
                type="text"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
            />
          </div>

          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">
              تاریخ
            </label>
            <input
                v-model="newTournamentDate"
                type="date"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
            />
          </div>

          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">
              تعداد زمین
            </label>
            <input
                v-model.number="newTournamentCourts"
                type="number"
                min="1"
                max="12"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
            />
          </div>
        </div>

        <div class="mt-6 flex gap-3">
          <button
              @click="submitTournament"
              :disabled="submitting || !draftWeek || !draftGroup"
              class="flex-1 rounded-lg bg-emerald-600 px-4 py-2 font-medium text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {{ submitting ? "..." : "ایجاد" }}
          </button>
          <button @click="closeForm" class="rounded-lg bg-slate-200 px-4 py-2 text-slate-700 transition hover:bg-slate-300">
            انصراف
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
