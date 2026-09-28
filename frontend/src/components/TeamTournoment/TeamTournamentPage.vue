<script setup lang="ts">
import { computed } from "vue";

import { useUiStore } from "../../stores/ui";
import type { TeamTournamentTab } from "../../stores/ui";
import { useTeamTournamentStore } from "../../stores/teamTournament";
import { useLeagueStore } from "../../stores/league";

import TeamManager from "./TeamManager.vue";
import TeamLineupPage from "./TeamLineupPage.vue";
import TeamMatchesPage from "./TeamMatchesPage.vue";
// import TeamExportPage from "./TeamExportPage.vue";

import { GENDER_LABELS } from "../../data/categories";

const ui = useUiStore();
const teamStore = useTeamTournamentStore();
const leagueStore = useLeagueStore();

// ─────────────── مسابقهٔ جاری ───────────────
const current = computed(() => teamStore.current);

const teamCount = computed(() => current.value?.teams?.length ?? 0);
const encounters = computed(() => current.value?.encounters ?? []);
const doneEncounters = computed(
    () => encounters.value.filter((e) => e.status === "completed").length,
);

const STATUS_LABELS: Record<string, string> = {
  draft: "پیش‌نویس",
  teamsRegistered: "تیم‌ها ثبت شد",
  rosterLocked: "روستر قفل",
  drawn: "قرعه‌کشی شد",
  inProgress: "در حال اجرا",
  finished: "پایان‌یافته",
};

const STATUS_CLASSES: Record<string, string> = {
  draft: "bg-slate-100 text-slate-600",
  teamsRegistered: "bg-blue-50 text-blue-700",
  rosterLocked: "bg-amber-50 text-amber-700",
  drawn: "bg-violet-50 text-violet-700",
  inProgress: "bg-emerald-50 text-emerald-700",
  finished: "bg-slate-800 text-white",
};

function formatDate(iso?: string | null): string {
  if (!iso) return "—";
  const [y, m, d] = iso.slice(0, 10).split("-").map(Number);
  if (!y || !m || !d) return String(iso);
  return new Date(y, m - 1, d).toLocaleDateString("fa-IR");
}

// ─────────────── تب‌ها ───────────────
interface TeamTabItem {
  key: TeamTournamentTab;
  label: string;
  badge?: string;
  danger?: boolean;
}

const tabs = computed<TeamTabItem[]>(() => [
  {
    key: "teams",
    label: "🛡️ تیم‌ها",
    badge: teamCount.value ? String(teamCount.value) : undefined,
  },
  {
    key: "matches",
    label: "⚔️ بازی‌ها",
    badge: encounters.value.length
        ? `${doneEncounters.value}/${encounters.value.length}`
        : undefined,
  },
  { key: "lineups", label: "📋 ترکیب" },
  // { key: "export", label: "🧷 خروجی" },
]);

// ─────────────── جابجایی بین مسابقات تیمی ───────────────
const otherTournaments = computed(() =>
    teamStore.items.filter((t) => t.id !== current.value?.id),
);

function switchTo(id: string) {
  if (!id || id === current.value?.id) return;
  teamStore.select(id);
  ui.teamTab = "teams";
}

// ─────────────── لیگ والد ───────────────
const parentLeague = computed(() => {
  const leagueId = current.value?.leagueId;
  if (!leagueId) return null;
  return leagueStore.leagues.find((l) => l.id === leagueId) ?? null;
});

const stageLabel = computed(() =>
    current.value?.stageOrder != null ? `مرحلهٔ ${current.value.stageOrder}` : null,
);

function goBackToLeague() {
  const league = parentLeague.value;
  if (!league) return;
  leagueStore.selectLeague(league.id);
  ui.page = "league";
}

function goHome() {
  teamStore.clearSelection();
  ui.goHome();
}
</script>

<template>
  <div>
    <!-- حالت بدون مسابقهٔ فعال -->
    <div v-if="!current" class="rounded-xl border border-slate-200 bg-white p-8">
      <p class="text-center text-sm text-slate-600">
        {{
          teamStore.items.length
              ? "یک مسابقهٔ تیمی را انتخاب کنید:"
              : "هنوز مسابقهٔ تیمی ثبت نشده است."
        }}
      </p>

      <ul v-if="teamStore.items.length" class="mx-auto mt-5 max-w-lg space-y-2">
        <li v-for="t in teamStore.items" :key="t.id">
          <button
              type="button"
              @click="switchTo(t.id)"
              class="flex w-full items-center justify-between gap-3 rounded-lg border border-slate-200 px-4 py-3 text-right text-sm transition-colors hover:border-indigo-300 hover:bg-indigo-50/40"
          >
            <span class="truncate font-medium text-slate-800">{{ t.name }}</span>
            <span class="shrink-0 text-xs text-slate-500">
              {{ formatDate(t.date ?? t.createdAt) }}
            </span>
          </button>
        </li>
      </ul>

      <div class="mt-6 text-center">
        <button
            type="button"
            @click="goHome"
            class="rounded-lg border border-slate-200 px-4 py-2 text-sm text-slate-700 transition-colors hover:bg-slate-50"
        >
          برگشت به صفحهٔ اصلی
        </button>
      </div>
    </div>

    <template v-else>
      <!-- برگشت -->
      <div class="mb-4 flex items-center gap-4">
        <button
            type="button"
            @click="parentLeague ? goBackToLeague() : goHome()"
            class="inline-flex items-center gap-1.5 text-sm text-slate-500 transition-colors hover:text-blue-600"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7 7-7m-7 7h18" />
          </svg>
          {{ parentLeague ? `برگشت به لیگ: ${parentLeague.name}` : "برگشت به فهرست" }}
        </button>
      </div>

      <!-- هدر -->
      <div class="mb-6 rounded-xl border border-slate-200 bg-white p-5">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="rounded-md bg-indigo-50 px-2 py-0.5 text-xs font-semibold text-indigo-700">
                تیمی
              </span>
              <h2 class="truncate text-xl font-bold text-slate-800">
                {{ current.name }}
              </h2>
              <span
                  class="rounded-full px-2.5 py-0.5 text-xs font-semibold"
                  :class="STATUS_CLASSES[current.status] ?? 'bg-slate-100 text-slate-600'"
              >
                {{ STATUS_LABELS[current.status] ?? current.status }}
              </span>
            </div>

            <div class="mt-2 flex flex-wrap items-center gap-1.5 text-xs text-slate-600">
              <span v-if="stageLabel" class="rounded-md bg-slate-100 px-2 py-1">
                {{ stageLabel }}
              </span>
              <span class="rounded-md bg-slate-100 px-2 py-1">
                {{ current.ageCategory }}
              </span>
              <span class="rounded-md bg-slate-100 px-2 py-1">
                {{ GENDER_LABELS[current.gender] }}
              </span>
              <span class="rounded-md bg-slate-100 px-2 py-1">
                {{ formatDate(current.date ?? current.createdAt) }}
              </span>
              <span class="rounded-md bg-slate-100 px-2 py-1">
                {{ teamCount }} تیم
              </span>
            </div>
          </div>

          <div class="flex shrink-0 flex-wrap items-center gap-2">
            <!-- جابجایی بین مسابقات تیمی -->
            <select
                v-if="otherTournaments.length"
                class="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700 transition-colors hover:border-slate-300"
                :value="current.id"
                @change="switchTo(($event.target as HTMLSelectElement).value)"
                aria-label="انتخاب مسابقهٔ تیمی"
            >
              <option :value="current.id">{{ current.name }} (فعلی)</option>
              <option v-for="t in otherTournaments" :key="t.id" :value="t.id">
                {{ t.name }}
              </option>
            </select>
          </div>
        </div>
      </div>

      <!-- تب‌ها -->
      <div class="mb-6 border-b border-slate-200">
        <div class="flex gap-1 overflow-x-auto" role="tablist">
          <button
              v-for="tab in tabs"
              :key="tab.key"
              type="button"
              role="tab"
              :aria-selected="ui.teamTab === tab.key"
              :class="[
                'relative flex items-center gap-2 whitespace-nowrap rounded-t-lg px-4 py-2.5 text-sm transition-colors',
                'after:absolute after:inset-x-2 after:-bottom-px after:h-0.5 after:rounded-full after:transition-colors',
                ui.teamTab === tab.key
                  ? 'font-semibold text-indigo-600 after:bg-indigo-600'
                  : 'text-slate-500 hover:bg-slate-50 hover:text-slate-800 after:bg-transparent',
              ]"
              @click="ui.teamTab = tab.key"
          >
            {{ tab.label }}
            <span
                v-if="tab.badge"
                class="flex h-5 min-w-[20px] items-center justify-center rounded-full px-1 text-[10px] font-bold"
                :class="[
                  tab.danger
                    ? 'bg-rose-100 text-rose-700'
                    : ui.teamTab === tab.key
                      ? 'bg-indigo-600 text-white'
                      : 'bg-slate-200 text-slate-600',
                ]"
            >
              {{ tab.badge }}
            </span>
          </button>
        </div>
      </div>

      <div class="min-h-[400px]">
        <TeamManager v-if="ui.teamTab === 'teams'" />
        <TeamLineupPage v-else-if="ui.teamTab === 'lineups'" />
        <TeamMatchesPage v-else-if="ui.teamTab === 'matches'" />
<!--        <TeamExportPage v-else-if="ui.teamTab === 'export'" />-->
      </div>
    </template>
  </div>
</template>
