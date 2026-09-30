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
import TournamentSettings from "./TournamentSettings.vue";

const ui = useUiStore();
const store = useTournamentStore();
const leagueStore = useLeagueStore();

interface TournamentTabItem {
  key: TournamentTab;
  label: string;
}

const tabs: TournamentTabItem[] = [
  { key: "athletes", label: "ورزشکاران" },
  { key: "weighIn", label: "وزن‌کشی" },
  { key: "draw", label: "قرعه‌کشی" },
  { key: "matches", label: "برگزاری بازی‌ها" },
  { key: "standings", label: "رده‌بندی" },
  { key: "export", label: "جداول نهایی" },
];
const matchCount=computed(()=>store.currentTournament?.matches?.length??0)
const completedCount=computed(()=>store.currentTournament?.matches?.filter((m:any)=>m.winnerId||m.status==='completed').length??0)

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
    <section class="relative mb-6 overflow-hidden rounded-3xl bg-slate-950 text-white shadow-xl">
      <div class="pointer-events-none absolute -left-16 -top-20 h-56 w-56 rounded-full bg-blue-500/20 blur-3xl"></div><div class="pointer-events-none absolute -bottom-24 right-1/3 h-48 w-48 rounded-full bg-rose-500/15 blur-3xl"></div>
      <div class="relative px-5 py-5 sm:px-7 sm:py-7">
        <div class="flex flex-wrap items-start justify-between gap-x-8 gap-y-4">
          <!-- عنوان و مشخصات -->
          <div class="min-w-0 flex-1">
            <div class="mb-2 inline-flex items-center gap-2 rounded-full border border-emerald-400/20 bg-emerald-400/10 px-3 py-1 text-[11px] font-bold text-emerald-300"><span class="h-2 w-2 animate-pulse rounded-full bg-emerald-400"></span>پنل برگزاری مسابقه</div>
            <h2 class="max-w-2xl text-xl font-black leading-8 text-white sm:text-3xl">
              {{ store.currentTournament?.name }}
            </h2>

            <div class="mt-2.5 flex flex-wrap items-center gap-x-2 gap-y-1.5">
              <span
                  v-if="store.currentTournament?.ageCategory"
                  class="inline-flex items-center rounded-lg border border-white/10 bg-white/[.07] px-2.5 py-1 text-xs font-medium text-slate-300"
              >
                {{ store.currentTournament.ageCategory }}
              </span>
              <span
                  v-if="store.currentTournament"
                  class="inline-flex items-center rounded-lg border border-white/10 bg-white/[.07] px-2.5 py-1 text-xs font-medium text-slate-300"
              >
                {{ GENDER_LABELS[store.currentTournament.gender] }}
              </span>
              <span
                  v-if="store.currentTournament"
                  class="inline-flex items-center rounded-lg border border-white/10 bg-white/[.07] px-2.5 py-1 text-xs font-medium text-slate-300"
              >
                {{ store.currentTournament.courts }} زمین
              </span>
              <span
                  class="inline-flex items-center rounded-lg border border-blue-400/20 bg-blue-400/10 px-2.5 py-1 text-xs font-medium text-blue-200"
              >
                {{ store.currentTournament?.athletes?.length ?? 0 }} ورزشکار
              </span>
            </div>
            <div class="mt-5 grid max-w-lg grid-cols-3 gap-2"><div class="rounded-xl border border-white/10 bg-white/[.05] p-3"><div class="text-xl font-black">{{store.currentTournament?.athletes?.length??0}}</div><div class="text-[10px] text-slate-400">ورزشکار</div></div><div class="rounded-xl border border-white/10 bg-white/[.05] p-3"><div class="text-xl font-black text-blue-300">{{matchCount}}</div><div class="text-[10px] text-slate-400">کل بازی‌ها</div></div><div class="rounded-xl border border-white/10 bg-white/[.05] p-3"><div class="text-xl font-black text-emerald-300">{{completedCount}}</div><div class="text-[10px] text-slate-400">نتیجه ثبت‌شده</div></div></div>
          </div>

          <!-- اکشن‌ها -->
          <div class="flex shrink-0 items-center gap-2">
            <button type="button" @click="ui.tab='settings'" class="inline-flex items-center gap-2 rounded-xl border px-3.5 py-2 text-sm font-bold transition" :class="ui.tab==='settings'?'border-white bg-white text-slate-950 shadow':'border-white/15 bg-white/10 text-white hover:bg-white/20'"><span class="text-base">⚙</span>تنظیمات</button>
            <button
                type="button"
                :disabled="exporting"
                @click="onExportExcel"
                class="inline-flex items-center gap-2 rounded-xl bg-blue-600 px-3.5 py-2 text-sm font-bold text-white transition-colors hover:bg-blue-500 disabled:cursor-wait disabled:opacity-60"
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
                class="inline-flex items-center gap-2 rounded-xl border border-white/15 bg-white/10 px-3.5 py-2 text-sm font-medium text-slate-200 transition-colors hover:bg-white/20 disabled:cursor-wait disabled:opacity-60"
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
      <nav class="relative flex items-center gap-1 overflow-x-auto border-t border-white/10 bg-white/[.04] px-3 py-2.5">
        <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            :class="[
              'flex-shrink-0 whitespace-nowrap rounded-lg px-3.5 py-2 text-sm transition-colors',
              ui.tab === tab.key
                ? 'bg-white font-bold text-slate-950 shadow'
                : 'text-slate-400 hover:bg-white/10 hover:text-white',
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
    <TournamentSettings v-else-if="ui.tab === 'settings'" />

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
