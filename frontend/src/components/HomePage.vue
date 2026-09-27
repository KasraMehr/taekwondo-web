<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useTournamentStore } from "../stores/tournament";
import { useTeamTournamentStore } from "../stores/teamTournament";
import { useLeagueStore } from "../stores/league";
import { useUiStore } from "../stores/ui";
import { AGE_CATEGORIES, GENDER_LABELS, WEIGHT_CATEGORIES } from "../data/categories";
import type { Gender, AgeCategory } from "../data/categories";
import type { TournamentFormat, TeamTournamentStatus } from "../types";
import { loadWebLeagues, webApi } from "../webApi";
import {
  isValidJalaaliDate,
  toGregorian,
  toJalaali,
} from "jalaali-js";


const store = useTournamentStore();
const teamStore = useTeamTournamentStore();
const leagueStore = useLeagueStore();
const ui = useUiStore();

const FORMAT_LABELS: Record<TournamentFormat, string> = {
  grandPrix: "انفرادی",
  team: "تیمی",
};

const TEAM_STATUS_LABELS: Record<TeamTournamentStatus, string> = {
  draft: "پیش‌نویس",
  teamsRegistered: "تیم‌ها ثبت شد",
  rosterLocked: "روستر قفل شد",
  drawn: "قرعه‌کشی شده",
  inProgress: "در جریان",
  finished: "پایان‌یافته",
};

const FORMAT_FILTERS = [
  { key: "all", label: "همه" },
  { key: "grandPrix", label: "انفرادی" },
  { key: "team", label: "تیمی" },
] as const;

type FormatFilter = (typeof FORMAT_FILTERS)[number]["key"];

type JalaliParts = {
  jy: number;
  jm: number;
  jd: number;
};

const PERSIAN_DIGITS = "۰۱۲۳۴۵۶۷۸۹";
const ARABIC_DIGITS = "٠١٢٣٤٥٦٧٨٩";

function normalizeDigits(value: string): string {
  return value
      .replace(/[۰-۹]/g, (digit) =>
          String(PERSIAN_DIGITS.indexOf(digit)),
      )
      .replace(/[٠-٩]/g, (digit) =>
          String(ARABIC_DIGITS.indexOf(digit)),
      );
}

function toPersianDigits(value: string | number): string {
  return String(value).replace(/[0-9]/g, (digit) => {
    return PERSIAN_DIGITS[Number(digit)];
  });
}

function pad2(value: number): string {
  return String(value).padStart(2, "0");
}

function parseJalaliDate(value: string): JalaliParts | null {
  const normalized = normalizeDigits(value)
      .trim()
      .replace(/[.-]/g, "/")
      .replace(/\s+/g, "");

  const match =
      /^(\d{4})\/(\d{1,2})\/(\d{1,2})$/.exec(normalized);

  if (!match) {
    return null;
  }

  const jy = Number(match[1]);
  const jm = Number(match[2]);
  const jd = Number(match[3]);

  if (!isValidJalaaliDate(jy, jm, jd)) {
    return null;
  }

  return { jy, jm, jd };
}

function formatJalaliDate({
                            jy,
                            jm,
                            jd,
                          }: JalaliParts): string {
  return toPersianDigits(
      `${String(jy).padStart(4, "0")}/${pad2(jm)}/${pad2(jd)}`,
  );
}

/**
 * تاریخ امروز برای مقدار اولیه فرم
 */
function todayJalali(): string {
  const now = new Date();

  const { jy, jm, jd } = toJalaali(
      now.getFullYear(),
      now.getMonth() + 1,
      now.getDate(),
  );

  return formatJalaliDate({ jy, jm, jd });
}

/**
 * تبدیل مقدار فرم شمسی به مقدار قابل ذخیره در Store
 *
 * ۱۴۰۵/۰۶/۱۸ -> 2026-09-09
 */
function jalaliInputToIso(value: string): string {
  const jalali = parseJalaliDate(value);

  if (!jalali) {
    return "";
  }

  const { gy, gm, gd } = toGregorian(
      jalali.jy,
      jalali.jm,
      jalali.jd,
  );

  return `${String(gy).padStart(4, "0")}-${pad2(gm)}-${pad2(gd)}`;
}


/**
 * تبدیل مقدار ذخیره‌شده برای نمایش در کارت
 *
 * 202-09 -> ۱۴۰۵۱۴۰۵/۰۶/۱۸
 */
function formatDate(iso?: string | null): string {
  if (!iso) {
    return "—";
  }

  const match =
      /^(\d{4})-(\d{1,2})-(\d{1,2})/.exec(iso.trim());

  if (!match) {
    return String(iso);
  }

  const gy = Number(match[1]);
  const gm = Number(match[2]);
  const gd = Number(match[3]);

  const date = new Date(Date.UTC(gy, gm - 1, gd));

  if (
      Number.isNaN(date.getTime()) ||
      date.getUTCFullYear() !== gy ||
      date.getUTCMonth() + 1 !== gm ||
      date.getUTCDate() !== gd
  ) {
    return String(iso);
  }

  const { jy, jm, jd } = toJalaali(gy, gm, gd);

  return formatJalaliDate({ jy, jm, jd });
}

/**
 * استانداردسازی مقدار input بعد از خروج کاربر از فیلد
 */
function normalizeDateInput(): void {
  const parsed = parseJalaliDate(dateJalali.value);

  if (parsed) {
    dateJalali.value = formatJalaliDate(parsed);
  }
}

/** وضعیت مسابقه: در جریان، برگزار شده، آینده */
type EventStatus = "ongoing" | "past" | "upcoming";

function getEventStatus(isoDate?: string | null): EventStatus {
  if (!isoDate) return "upcoming";
  const d = new Date();
  const todayStr = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
  const dateOnly = isoDate.slice(0, 10);
  if (dateOnly === todayStr) return "ongoing";
  if (dateOnly < todayStr) return "past";
  return "upcoming";
}

const EVENT_STATUS_CONFIG: Record<EventStatus, { label: string; classes: string }> = {
  ongoing:  { label: "در حال برگزاری", classes: "bg-green-100 text-green-700 ring-1 ring-green-300" },
  past:     { label: "برگزار شده",     classes: "bg-slate-100 text-slate-500" },
  upcoming: { label: "آینده",          classes: "bg-amber-100 text-amber-700" },
};

function goTeamTournament(): boolean {
  const fn = (ui as unknown as Record<string, unknown>).goTeamTournament;
  if (typeof fn === "function") {
    (fn as () => void).call(ui);
    return true;
  }
  error.value = "نمای مسابقه تیمی هنوز در ui store ثبت نشده است";
  return false;
}

// ─── فرم مسابقه ───
const showForm = ref(false);
const format = ref<TournamentFormat>("grandPrix");
const name = ref("");
// تاریخ در فرم به شمسی نگه داشته می‌شود
const dateJalali = ref(todayJalali());
const courts = ref(2);
const gender = ref<Gender>("male");
const ageCategory = ref<AgeCategory>("نونهالان");
const error = ref("");

const blindLineup = ref(false);

const activeTab = ref<"tournaments" | "leagues">("tournaments");
const formatFilter = ref<FormatFilter>("all");

function resetForm() {
  name.value = "";
  dateJalali.value = todayJalali();
  courts.value = 2;
  blindLineup.value = false;
  error.value = "";
  showForm.value = false;
}

async function submit() {
  if (!name.value.trim()) {
    error.value = "نام مسابقه را وارد کنید";
    return;
  }

  if (!dateJalali.value.trim()) {
    error.value = "تاریخ را وارد کنید";
    return;
  }

  const isoDate = jalaliInputToIso(dateJalali.value);

  if (!isoDate) {
    error.value =
        "تاریخ شمسی معتبر نیست؛ مثال: ۱۴۰۵/۰۶/۱۵";
    return;
  }

  if (courts.value < 1 || courts.value > 12) {
    error.value = "تعداد زمین باید بین ۱ تا ۱۲ باشد";
    return;
  }

  // مقدار input را هم به شکل استاندارد فارسی نگه می‌داریم
  normalizeDateInput();

  if (format.value === "team") {
    const created = teamStore.create({
      name: name.value.trim(),

      // فقط همین مقدار باید ذخیره شود:
      // YYYY-MM-DD
      date: isoDate,

      courts: courts.value,
      gender: gender.value,
      ageCategory: ageCategory.value,
      blindLineup: blindLineup.value,
    });

    teamStore.select(created.id);

    if (!goTeamTournament()) {
      return;
    }

    resetForm();
    return;
  }

  // فقط ISO به Store ارسال می‌شود، نه تاریخ شمسی فرم
  try {
    const created = await webApi().call<any>('/tournaments', 'POST', {
      name: name.value.trim(), date: isoDate, courts: courts.value,
      gender: gender.value, ageCategory: ageCategory.value, format: 'grandPrix',
    })
    store.tournaments.push(created)
    store.save()
  } catch (e: any) {
    error.value = e?.message || 'ساخت مسابقه ناموفق بود'
    return
  }

  resetForm();
  ui.goTournament();
}

// ─── لیست یکپارچه ───
interface Row {
  key: string;
  id: string;
  format: TournamentFormat;
  name: string;
  date: string;
  courts: number;
  gender: Gender;
  ageCategory: AgeCategory;
  leagueId?: string | null;
  count: number;
  statusLabel?: string;
}

const rows = computed<Row[]>(() => {
  const individual: Row[] = store.tournaments.map((t) => ({
    key: `grandPrix:${t.id}`,
    id: t.id,
    format: (t.format ?? "grandPrix") as TournamentFormat,
    name: t.name,
    date: t.date,
    courts: t.courts,
    gender: t.gender,
    ageCategory: t.ageCategory,
    leagueId: t.leagueId ?? null,
    count: t.athletes.length,
  }));

  const teams: Row[] = teamStore.items.map((t) => ({
    key: `team:${t.id}`,
    id: t.id,
    format: "team" as TournamentFormat,
    name: t.name,
    date: t.date,
    courts: t.courts,
    gender: t.gender,
    ageCategory: t.ageCategory,
    leagueId: t.leagueId ?? null,
    count: t.teams.length,
    statusLabel: TEAM_STATUS_LABELS[t.status],
  }));

  return [...individual, ...teams].sort((a, b) => b.date.localeCompare(a.date));
});

const filteredRows = computed(() =>
    formatFilter.value === "all"
        ? rows.value
        : rows.value.filter((r) => r.format === formatFilter.value),
);

function open(row: Row) {
  if (row.format === "team") {
    teamStore.select(row.id);
    goTeamTournament();
    return;
  }
  store.selectTournament(row.id);
  ui.goTournament();
}

function remove(row: Row) {
  if (!confirm(`مسابقه «${row.name}» و تمام اطلاعات آن حذف شود؟`)) return;
  if (row.format === "team") teamStore.remove(row.id);
  else store.deleteTournament(row.id);
}

// ─── لیگ‌ها ───
const showLeagueForm = ref(false);
const leagueName = ref("");
const leagueFormat = ref<TournamentFormat>("grandPrix");
const leagueGender = ref<Gender>("male");
const leagueAgeCategory = ref<AgeCategory>("نونهالان");
const leagueSlotCount = ref<8 | 10>(10);
const leagueBlindLineup = ref(false);
const seasonName = ref("۱۴۰۵");
const stageCount = ref(2);
const weeksPerStage = ref(3);
const leagueError = ref("");

const leagueSuggestedSlotCount = computed<8 | 10>(() =>
    WEIGHT_CATEGORIES[leagueAgeCategory.value][leagueGender.value].length === 10 ? 10 : 8,
);
watch(leagueSuggestedSlotCount, (v) => { leagueSlotCount.value = v; }, { immediate: true });

function resetLeagueForm() {
  leagueName.value = "";
  seasonName.value = "۱۴۰۵";
  stageCount.value = 2;
  weeksPerStage.value = 3;
  leagueFormat.value = "grandPrix";
  leagueGender.value = "male";
  leagueAgeCategory.value = "نونهالان";
  leagueSlotCount.value = leagueSuggestedSlotCount.value;
  leagueBlindLineup.value = false;
  leagueError.value = "";
  showLeagueForm.value = false;
}

async function submitLeague() {
  if (!leagueName.value.trim()) { leagueError.value = "نام لیگ را وارد کنید"; return; }
  if (!seasonName.value.trim()) { leagueError.value = "نام فصل را وارد کنید"; return; }
  if (stageCount.value < 1 || stageCount.value > 6) { leagueError.value = "تعداد مراحل باید بین ۱ تا ۶ باشد"; return; }
  if (weeksPerStage.value < 1 || weeksPerStage.value > 12) { leagueError.value = "تعداد هفته در هر مرحله باید بین ۱ تا ۱۲ باشد"; return; }

  try {
    const created = await webApi().call<any>('/leagues', 'POST', {
      name: leagueName.value.trim(), seasonName: seasonName.value.trim(),
      gender: leagueGender.value, ageCategory: leagueAgeCategory.value,
      stageCount: stageCount.value, weeksPerStage: weeksPerStage.value,
      groupNames: ['گروه A'],
    })

    if (!created) {
      leagueError.value = "ایجاد لیگ ناموفق بود (نام تکراری یا ورودی نامعتبر)";
      return;
    }

    leagueStore.replaceFromBackend(await loadWebLeagues())
    leagueStore.selectLeague(created.id);
    resetLeagueForm();
    activeTab.value = "leagues";
  } catch (e) {
    console.error("[createLeague]", e);
    leagueError.value = e instanceof Error ? e.message : "ایجاد لیگ ناموفق بود";
  }
}

function openLeague(id: string) {
  leagueStore.selectLeague(id);
  ui.goLeagueDetail();
}

function removeLeague(id: string, lName: string) {
  if (confirm(`لیگ «${lName}» و تمام اطلاعات آن حذف شود؟`)) leagueStore.deleteLeague(id);
}

const totalTournaments = computed(() => rows.value.length);
const totalTeamTournaments = computed(() => teamStore.items.length);
const totalLeagues = computed(() => leagueStore.leagues.length);
</script>

<template>
  <div>
    <!-- تیتر -->
    <div class="mb-6">
      <h2 class="text-2xl font-bold">خانه</h2>
      <p class="mt-1 text-sm text-slate-500">
        {{ totalTournaments }} مسابقه ({{ totalTeamTournaments }} تیمی) و {{ totalLeagues }} لیگ
      </p>
    </div>

    <!-- تب‌ها -->
    <div class="mb-6 flex gap-2 border-b border-slate-200">
      <button
          @click="activeTab = 'tournaments'"
          :class="[
          'rounded-t-lg px-5 py-2.5 font-medium transition',
          activeTab === 'tournaments'
            ? 'border-b-2 border-blue-600 bg-white text-blue-600'
            : 'text-slate-500 hover:bg-slate-100',
        ]"
      >
        🥋 مسابقات
      </button>
      <button
          @click="activeTab = 'leagues'"
          :class="[
          'rounded-t-lg px-5 py-2.5 font-medium transition',
          activeTab === 'leagues'
            ? 'border-b-2 border-blue-600 bg-white text-blue-600'
            : 'text-slate-500 hover:bg-slate-100',
        ]"
      >
        🏆 لیگ‌ها
      </button>
    </div>

    <!-- ═══════════════ تب مسابقات ═══════════════ -->
    <div v-if="activeTab === 'tournaments'">
      <div class="mb-6 flex flex-wrap items-center justify-between gap-3">
        <!-- فیلتر نوع -->
        <div class="flex gap-1 rounded-xl bg-slate-100 p-1">
          <button
              v-for="opt in FORMAT_FILTERS"
              :key="opt.key"
              @click="formatFilter = opt.key"
              :class="[
              'rounded-lg px-3.5 py-1.5 text-sm font-medium transition',
              formatFilter === opt.key
                ? 'bg-white text-blue-600 shadow-sm'
                : 'text-slate-500 hover:text-slate-700',
            ]"
          >
            {{ opt.label }}
          </button>
        </div>

        <button
            @click="showForm = !showForm"
            class="rounded-xl bg-blue-600 px-5 py-2.5 font-medium text-white shadow transition hover:bg-blue-700"
        >
          + مسابقه جدید
        </button>
      </div>

      <!-- فرم ایجاد مسابقه -->
      <div v-if="showForm" class="mb-8 rounded-2xl border border-blue-200 bg-white p-6 shadow-sm">
        <h3 class="mb-4 text-lg font-bold">ایجاد مسابقه جدید</h3>

        <!-- نوع مسابقه -->
        <div class="mb-5">
          <label class="mb-2 block text-sm font-medium text-slate-600">نوع مسابقه</label>
          <div class="grid gap-3 sm:grid-cols-2">
            <button
                v-for="(label, key) in FORMAT_LABELS"
                :key="key"
                type="button"
                @click="format = key as TournamentFormat"
                :class="[
                'rounded-xl border-2 px-4 py-3 text-right transition',
                format === key
                  ? 'border-blue-600 bg-blue-50'
                  : 'border-slate-200 hover:border-slate-300',
              ]"
            >
              <span class="block font-medium">{{ label }}</span>
              <span class="mt-0.5 block text-xs text-slate-500">
                {{ key === "team" ? "مواجهه تیم‌ها با ارنج" : "جدول حذفی در هر وزن" }}
              </span>
            </button>
          </div>
        </div>

        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">نام مسابقه</label>
            <input
                v-model="name"
                @keyup.enter="submit"
                type="text"
                :placeholder="
                format === 'team'
                  ? 'مثلاً: لیگ تیمی استان — نونهالان'
                  : 'مثلاً: قهرمانی استان — نونهالان'
              "
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">
              تاریخ برگزاری (شمسی)
            </label>
            <!-- اینپوت تاریخ شمسی: کاربر به‌صورت YYYY/MM/DD وارد می‌کند -->
            <input
                v-model="dateJalali"
                @blur="normalizeDateInput"
                type="text"
                inputmode="numeric"
                autocomplete="off"
                placeholder="1405/06/15"
                dir="ltr"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            />

            <p class="mt-1 textlate-400late-400">
              فرمت: سال/ماه/روز — مثال: ۱۴۰۵/۰۶/۱۵
            </p>
            <p class="mt-1 text-xs text-slate-400">فرمت: YYYY/MM/DD — مثال: 1405/06/15</p>
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">تعداد زمین</label>
            <input
                v-model.number="courts"
                type="number"
                min="1"
                max="12"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">جنسیت</label>
            <select
                v-model="gender"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            >
              <option v-for="(label, key) in GENDER_LABELS" :key="key" :value="key">
                {{ label }}
              </option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">رده سنی</label>
            <select
                v-model="ageCategory"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            >
              <option v-for="c in AGE_CATEGORIES" :key="c" :value="c">{{ c }}</option>
            </select>
          </div>
        </div>

        <label
            v-if="format === 'team'"
            class="mt-4 flex cursor-pointer items-center gap-2 text-sm text-slate-700"
        >
          <input v-model="blindLineup" type="checkbox" class="h-4 w-4 rounded border-slate-300" />
          ارنج پنهان (ترکیب حریف تا ارسال هر دو مربی نمایش داده نشود)
        </label>

        <p v-if="error" class="mt-3 text-sm text-red-600">{{ error }}</p>
        <div class="mt-4 flex gap-3">
          <button
              @click="submit"
              class="rounded-lg bg-green-600 px-5 py-2 font-medium text-white transition hover:bg-green-700"
          >
            ایجاد و ورود
          </button>
          <button
              @click="resetForm"
              class="rounded-lg bg-slate-200 px-5 py-2 text-slate-700 transition hover:bg-slate-300"
          >
            انصراف
          </button>
        </div>
      </div>

      <!-- حالت خالی: هیچ مسابقه‌ای وجود ندارد -->
      <div
          v-if="rows.length === 0"
          class="rounded-2xl border-2 border-dashed border-slate-300 bg-white py-20 text-center"
      >
        <div class="mb-3 text-5xl">🥋</div>
        <p class="text-lg text-slate-500">هنوز مسابقه‌ای ایجاد نشده است</p>
        <p class="mt-1 text-sm text-slate-400">با دکمه «مسابقه جدید» شروع کنید</p>
      </div>

      <!-- حالت خالی: فیلتر نتیجه‌ای ندارد -->
      <div
          v-else-if="filteredRows.length === 0"
          class="rounded-2xl border-2 border-dashed border-slate-300 bg-white py-20 text-center"
      >
        <div class="mb-3 text-5xl">🔍</div>
        <p class="text-lg text-slate-500">در این فیلتر مسابقه‌ای نیست</p>
        <button @click="formatFilter = 'all'" class="mt-2 text-sm text-blue-600 hover:underline">
          نمایش همه
        </button>
      </div>

      <!-- لیست مسابقات -->
      <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div
            v-for="row in filteredRows"
            :key="row.key"
            class="group relative rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-blue-300 hover:shadow-md"
        >
          <!-- برچسب‌ها — ردیف اول: نوع + لیگی/مستقل -->
          <div class="absolute right-4 top-4 flex flex-wrap gap-1.5">
            <!-- نوع مسابقه: انفرادی یا تیمی -->
            <span
                :class="[
                'rounded-full px-2.5 py-1 text-xs font-medium',
                row.format === 'team'
                  ? 'bg-purple-100 text-purple-700'
                  : 'bg-emerald-100 text-emerald-700',
              ]"
            >
              {{ FORMAT_LABELS[row.format] }}
            </span>
            <!-- لیگی یا مستقل -->
            <span
                v-if="row.leagueId"
                class="rounded-full bg-blue-100 px-2.5 py-1 text-xs font-medium text-blue-700"
            >
              لیگی
            </span>
            <span
                v-else
                class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-500"
            >
              مستقل
            </span>
            <!-- وضعیت زمانی مسابقه -->
            <span
                :class="[
                'rounded-full px-2.5 py-1 text-xs font-medium',
                EVENT_STATUS_CONFIG[getEventStatus(row.date)].classes,
              ]"
            >
              {{ EVENT_STATUS_CONFIG[getEventStatus(row.date)].label }}
            </span>
          </div>

          <div class="mb-3 flex items-start justify-between">
            <!-- فضای بالا برای برچسب‌های سه‌گانه -->
            <h3 class="pr-2 pt-16 text-lg font-bold">{{ row.name }}</h3>
            <button
                @click="remove(row)"
                class="rounded-lg p-1.5 text-slate-400 opacity-0 transition hover:bg-red-50 hover:text-red-600 group-hover:opacity-100"
                title="حذف"
            >
              ✕
            </button>
          </div>

          <div class="space-y-1 text-sm text-slate-500">
            <p>📅 {{ formatDate(row.date) }}</p>
            <p>🏟️ {{ row.courts }} زمین</p>
            <p>👥 {{ GENDER_LABELS[row.gender] }} — {{ row.ageCategory }}</p>
            <p>
              {{ row.format === "team" ? "🛡️" : "🥋" }}
              {{ row.count }} {{ row.format === "team" ? "تیم" : "ورزشکار" }}
            </p>
            <p v-if="row.statusLabel">⚙️ {{ row.statusLabel }}</p>
          </div>

          <button
              @click="open(row)"
              class="mt-4 w-full rounded-lg bg-blue-50 py-2 text-sm font-medium text-blue-700 transition hover:bg-blue-100"
          >
            ورود به مسابقه
          </button>
        </div>
      </div>
    </div>

    <!-- ═══════════════ تب لیگ‌ها ═══════════════ -->
    <div v-else>
      <div class="mb-6 flex justify-end">
        <button
            @click="showLeagueForm = !showLeagueForm"
            class="rounded-xl bg-blue-600 px-5 py-2.5 font-medium text-white shadow transition hover:bg-blue-700"
        >
          + لیگ جدید
        </button>
      </div>

      <div
          v-if="showLeagueForm"
          class="mb-8 rounded-2xl border border-blue-200 bg-white p-6 shadow-sm"
      >
        <h3 class="mb-4 text-lg font-bold">ایجاد لیگ جدید</h3>

        <div class="mb-5">
          <label class="mb-2 block text-sm font-medium text-slate-600">نوع لیگ</label>
          <div class="grid gap-3 sm:grid-cols-2">
            <button
                v-for="(label, key) in FORMAT_LABELS"
                :key="key"
                type="button"
                @click="leagueFormat = key as TournamentFormat"
                :class="[
                'rounded-xl border-2 px-4 py-3 text-right transition',
                leagueFormat === key
                  ? 'border-blue-600 bg-blue-50'
                  : 'border-slate-200 hover:border-slate-300',
              ]"
            >
              <span class="block font-medium">{{ label }}</span>
            </button>
          </div>
        </div>

        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">نام لیگ</label>
            <input
                v-model="leagueName"
                type="text"
                placeholder="مثلاً: لیگ استان تهران"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">نام فصل</label>
            <input
                v-model="seasonName"
                type="text"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">جنسیت</label>
            <select
                v-model="leagueGender"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            >
              <option v-for="(label, key) in GENDER_LABELS" :key="key" :value="key">
                {{ label }}
              </option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">رده سنی</label>
            <select
                v-model="leagueAgeCategory"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            >
              <option v-for="c in AGE_CATEGORIES" :key="c" :value="c">{{ c }}</option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">تعداد مراحل</label>
            <input
                v-model.number="stageCount"
                type="number"
                min="1"
                max="6"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-slate-600">هفته در هر مرحله</label>
            <input
                v-model.number="weeksPerStage"
                type="number"
                min="1"
                max="12"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
          </div>
          <div v-if="leagueFormat === 'team'">
            <label class="mb-1 block text-sm font-medium text-slate-600">
              تعداد اوزان هر مواجهه
            </label>
            <div class="w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-slate-700">
              {{ leagueSlotCount }} وزن
            </div>
            <p class="mt-1 text-xs text-slate-500">بر اساس رده «{{ leagueAgeCategory }}»</p>
          </div>
        </div>

        <label
            v-if="leagueFormat === 'team'"
            class="mt-4 flex cursor-pointer items-center gap-2 text-sm text-slate-700"
        >
          <input
              v-model="leagueBlindLineup"
              type="checkbox"
              class="h-4 w-4 rounded border-slate-300"
          />
          ارنج پنهان به‌صورت پیش‌فرض برای هفته‌های این لیگ
        </label>

        <p v-if="leagueError" class="mt-3 text-sm text-red-600">{{ leagueError }}</p>
        <div class="mt-4 flex gap-3">
          <button
              @click="submitLeague"
              class="rounded-lg bg-green-600 px-5 py-2 font-medium text-white transition hover:bg-green-700"
          >
            ایجاد لیگ
          </button>
          <button
              @click="resetLeagueForm"
              class="rounded-lg bg-slate-200 px-5 py-2 text-slate-700 transition hover:bg-slate-300"
          >
            انصراف
          </button>
        </div>
      </div>

      <div
          v-if="leagueStore.leagues.length === 0"
          class="rounded-2xl border-2 border-dashed border-slate-300 bg-white py-20 text-center"
      >
        <div class="mb-3 text-5xl">🏆</div>
        <p class="text-lg text-slate-500">هنوز لیگی ایجاد نشده است</p>
      </div>

      <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div
            v-for="l in leagueStore.leagues"
            :key="l.id"
            class="group relative rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-blue-300 hover:shadow-md"
        >
          <span
              :class="[
              'absolute right-4 top-4 rounded-full px-2.5 py-1 text-xs font-medium',
              l.format === 'team'
                ? 'bg-purple-100 text-purple-700'
                : 'bg-emerald-100 text-emerald-700',
            ]"
          >
            {{ FORMAT_LABELS[l.format ?? "grandPrix"] }}
          </span>

          <div class="mb-3 flex items-start justify-between">
            <h3 class="pr-20 text-lg font-bold">{{ l.name }}</h3>
            <button
                @click="removeLeague(l.id, l.name)"
                class="rounded-lg p-1.5 text-slate-400 opacity-0 transition hover:bg-red-50 hover:text-red-600 group-hover:opacity-100"
                title="حذف"
            >
              ✕
            </button>
          </div>

          <div class="space-y-1 text-sm text-slate-500">
            <p>🗓️ فصل {{ l.season.name }}</p>
            <p>📚 {{ l.season.stages.length }} مرحله</p>
            <p>👥 {{ GENDER_LABELS[l.gender] }} — {{ l.ageCategory }}</p>
          </div>

          <button
              @click="openLeague(l.id)"
              class="mt-4 w-full rounded-lg bg-blue-50 py-2 text-sm font-medium text-blue-700 transition hover:bg-blue-100"
          >
            مدیریت لیگ
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
