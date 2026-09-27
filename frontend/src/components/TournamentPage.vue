<script setup lang="ts">
import { computed, ref, onUnmounted } from "vue";

import { useUiStore } from "../stores/ui";
import type { TournamentTab } from "../stores/ui";
import { useTournamentStore } from "../stores/tournament";
import { useLeagueStore } from "../stores/league";

import AthleteManager from "./AthleteManager.vue";
import DrawPage from "./DrawPage.vue";
import MatchesPage from "./MatchesPage.vue";
import StandingsPage from "./StandingsPage.vue";
import BracketPdfExport from "./BracketPdfExport.vue";

import { GENDER_LABELS } from "../data/categories";
import WeighInView from "./WeighInView.vue";

const ui = useUiStore();
const store = useTournamentStore();
const leagueStore = useLeagueStore();

interface TournamentTabItem {
  key: TournamentTab;
  label: string;
}

const tabs: TournamentTabItem[] = [
  { key: "athletes", label: "👤 ورزشکاران" },
  { key: "weighIn", label: "⚖️ وزن‌کشی" },
  { key: "draw", label: "🎲 قرعکشی" },
  { key: "matches", label: "⚔️ بازی‌ها" },
  { key: "standings", label: "🏆 رده‌بندی" },
  { key: "export", label: "🧷 جداول نهایی" },
];

// ─────────────── تشخیص تورنمنت لیگی ───────────────
const isLeagueTournament = computed(() => {
  const tournament = store.currentTournament;
  if (!tournament) return false;

  if (tournament.leagueId) return true;

  const league = leagueStore.currentLeague;
  if (!league) return false;

  return league.season.stages.some((stage) =>
      stage.weeks.some((week) => week.tournamentId === tournament.id)
  );
});

function goBackToLeague() {
  const tournament = store.currentTournament;
  if (!tournament) return;

  const league = leagueStore.leagues.find((l) =>
      l.season.stages.some((stage) =>
          stage.weeks.some((week) => week.tournamentId === tournament.id)
      )
  );

  if (league) {
    leagueStore.selectLeague(league.id);
    ui.page = "league";
  }
}

// ─────────────── Toast ───────────────
type Toast = { msg: string; type: "success" | "error" };

const toast = ref<Toast | null>(null);
let toastTimer: ReturnType<typeof setTimeout> | undefined;

function notify(msg: string, type: Toast["type"] = "success") {
  toast.value = { msg, type };
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (toast.value = null), 3500);
}

onUnmounted(() => clearTimeout(toastTimer));

// ─────────────── اکسل ───────────────
const importInput = ref<HTMLInputElement | null>(null);
const exporting = ref(false);
const importing = ref(false);

function errMsg(err: unknown) {
  return err instanceof Error ? err.message : String(err);
}

async function onExportExcel() {
  const t = store.currentTournament;
  if (!t) return notify("ابتدا یک مسابقه انتخاب کنید", "error");
  if (!t.athletes?.length) return notify("هنوز ورزشکاری ثبت نشده", "error");

  exporting.value = true;
  try {
    const result = store.exportTournamentToExcel(t.id);
    if (result.error) {
      notify(`خطا در خروجی اکسل: ${result.error}`, "error");
    } else {
      notify("خروجی اکسل با موفقیت دانلود شد ✅");
    }
  } catch (err) {
    notify(`خطا در خروجی اکسل: ${errMsg(err)}`, "error");
  } finally {
    exporting.value = false;
  }
}

function triggerImport() {
  importInput.value?.click();
}

async function onImportExcel(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;

  importing.value = true;
  try {
    // 👇 اسم واقعی تابع خودت را بگذار
    await store.importTournamentFromExcel(file);
    notify("ایمپورت با موفقیت انجام شد");
  } catch (err) {
    notify(`خطا در ایمپورت: ${errMsg(err)}`, "error");
  } finally {
    importing.value = false;
    input.value = "";
  }
}
</script>

<template>
  <div>
    <!-- برگشت به لیگ (فقط برای تورنمنت‌های لیگی) -->
    <button
        v-if="isLeagueTournament"
        type="button"
        @click="goBackToLeague"
        class="group mb-3 inline-flex items-center gap-1.5 px-1 py-1 text-[13px] text-slate-400 transition-colors hover:text-blue-600"
    >
      <svg
          class="h-4 w-4 transition-transform group-hover:-translate-x-0.5"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7 7-7m-7 7h18" />
      </svg>
      برگشت به لیگ
    </button>

    <!-- کارت هدر -->
    <section class="mb-6 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <div class="px-5 py-4 sm:px-6 sm:py-5">
        <div class="flex flex-wrap items-start justify-between gap-x-8 gap-y-4">
          <!-- عنوان و مشخصات -->
          <div class="min-w-0 flex-1">
            <h2 class="truncate text-lg font-bold leading-7 text-slate-900 sm:text-xl">
              {{ store.currentTournament?.name }}
            </h2>

            <div class="mt-2.5 flex flex-wrap items-center gap-x-2 gap-y-1.5">
              <span
                  v-if="store.currentTournament?.ageCategory"
                  class="inline-flex items-center rounded-md bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600"
              >
                {{ store.currentTournament.ageCategory }}
              </span>
              <span
                  v-if="store.currentTournament"
                  class="inline-flex items-center rounded-md bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600"
              >
                {{ GENDER_LABELS[store.currentTournament.gender] }}
              </span>
              <span
                  v-if="store.currentTournament"
                  class="inline-flex items-center rounded-md bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600"
              >
                {{ store.currentTournament.courts }} زمین
              </span>
              <span
                  class="inline-flex items-center rounded-md bg-blue-50 px-2.5 py-1 text-xs font-medium text-blue-600"
              >
                {{ store.currentTournament?.athletes?.length ?? 0 }} ورزشکار
              </span>
            </div>
          </div>

          <!-- اکشن‌ها -->
          <div class="flex shrink-0 items-center gap-2">
            <button
                type="button"
                :disabled="exporting"
                @click="onExportExcel"
                class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-3.5 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-wait disabled:opacity-60 disabled:hover:bg-blue-600"
            >
              <svg
                  class="h-4 w-4 shrink-0"
                  :class="{ 'animate-spin': exporting }"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="2"
              >
                <path
                    v-if="!exporting"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M12 3v12m0 0l-4-4m4 4l4-4M4 17v2a2 2 0 002 2h12a2 2 0 002-2v-2"
                />
                <path v-else stroke-linecap="round" d="M12 3a9 9 0 109 9" />
              </svg>
              <span class="whitespace-nowrap">
                {{ exporting ? "در حال ساخت…" : "خروجی اکسل" }}
              </span>
            </button>

            <button
                type="button"
                :disabled="importing"
                @click="triggerImport"
                class="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3.5 py-2 text-sm font-medium text-slate-600 transition-colors hover:border-slate-300 hover:bg-slate-50 hover:text-slate-800 disabled:cursor-wait disabled:opacity-60"
            >
              <svg
                  class="h-4 w-4 shrink-0 text-blue-500"
                  :class="{ 'animate-spin': importing }"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="2"
              >
                <path
                    v-if="!importing"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M124M4 170 12l-4-4m4 4l4-4M4 17v2a2 2 0 002 2h12a2 2 0 002-2v-2"
                />
                <path v-else stroke-linecap="round" d="M12 3a9 9 0 109 9" />
              </svg>
              <span class="whitespace-nowrap">
                {{ importing ? "در حال ایمپورت…" : "ایمپورت اکسل" }}
              </span>
            </button>

            <input
                ref="importInput"
                type="file"
                accept=".xlsx"
                class="hidden"
                @change="onImportExcel"
            />
          </div>
        </div>
      </div>

      <!-- تب‌ها -->
      <nav class="flex items-center gap-1 overflow-x-auto border-t border-slate-100 px-3 py-2">
        <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            :class="[
              'flex-shrink-0 whitespace-nowrap rounded-lg px-3.5 py-2 text-sm transition-colors',
              ui.tab === tab.key
                ? 'bg-blue-50 font-semibold text-blue-700'
                : 'text-slate-500 hover:bg-slate-50 hover:text-slate-800',
            ]"
            @click="ui.tab = tab.key"
        >
          {{ tab.label }}
        </button>

      </nav>
    </section>

    <AthleteManager v-if="ui.tab === 'athletes'" />
    <DrawPage v-else-if="ui.tab === 'draw'" />
    <MatchesPage v-else-if="ui.tab === 'matches'" />
    <StandingsPage v-else-if="ui.tab === 'standings'" />
    <BracketPdfExport v-else-if="ui.tab === 'export'" />
    <WeighInView v-else-if="ui.tab === 'weighIn'" />

    <!-- Toast -->
    <Transition
        enter-active-class="transition duration-200 ease-out"
        enter-from-class="translate-y-3 opacity-0"
        leave-active-class="transition duration-150 ease-in"
        leave-to-class="translate-y-3 opacity-0"
    >
      <div
          v-if="toast"
          class="fixed bottom-6 left-1/2 z-[60] -translate-x-1/2 rounded-lg px-5 py-3 text-sm text-white shadow-lg"
          :class="toast.type === 'error' ? 'bg-red-600' : 'bg-slate-800'"
      >
        {{ toast.msg }}
      </div>
    </Transition>
  </div>
</template>
