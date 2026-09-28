<template>
  <div v-if="!current" class="rounded-xl border-2 border-dashed border-slate-200 py-16 text-center text-slate-400">
    <p class="mb-2 text-3xl">🛡️</p>
    <p>ابتدا یک مسابقهٔ تیمی انتخاب کنید</p>
  </div>

  <div v-else class="space-y-6">
    <!-- بازخورد -->
    <div
        v-if="feedback"
        role="status"
        class="flex items-start gap-3 rounded-xl border px-4 py-3 text-sm"
        :class="feedback.type === 'error'
        ? 'border-red-200 bg-red-50 text-red-700'
        : 'border-emerald-200 bg-emerald-50 text-emerald-700'"
    >
      <span class="shrink-0">{{ feedback.type === 'error' ? '⚠️' : '✅' }}</span>
      <p class="flex-1 whitespace-pre-line leading-6">{{ feedback.message }}</p>
      <button type="button" class="shrink-0 rounded p-1 opacity-60 hover:opacity-100" aria-label="بستن پیام" @click="feedback = null">✕</button>
    </div>

    <div v-if="!weightOptions.length" class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
      برای این رده سنی/جنسیت، جدول اوزان رسمی یافت نشد. ثبت ورزشکار بدون وزن انجام می‌شود.
    </div>

    <div v-if="!slotList.length" class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
      هنوز اسلات وزنی برای این مسابقه تعریف نشده است. ثبت تیم و ورزشکار آزاد است، اما برای ثبت ترکیب (Lineup) باید
      اسلات‌های وزنی (<code class="rounded bg-amber-100 px-1">weightSlots</code>) را از تب تنظیمات بسازید.
    </div>

    <!-- Toolbar -->
    <div class="flex flex-wrap items-center gap-3">
      <button type="button" :class="BTN.primary" :disabled="isLocked" @click="openTeam()">
        + افزودن تیم
      </button>

      <button type="button" :class="BTN.soft" :disabled="isLocked || !teamList.length" @click="openMember()">
        + افزودن ورزشکار به تیم
      </button>

      <label
          class="cursor-pointer rounded-lg border border-blue-200 bg-blue-50 px-4 py-2 text-sm font-medium text-blue-700 hover:bg-blue-100 focus-within:ring-2 focus-within:ring-blue-400"
          :class="isLocked ? 'pointer-events-none opacity-40' : ''"
      >
        <span>{{ importing ? '⏳ در حال خواندن…' : '📥 ایمپورت اکسل تیم‌ها' }}</span>
        <input ref="fileInput" type="file" accept=".xlsx,.xls,.csv" class="sr-only" :disabled="isLocked || importing" @change="readExcel" />
      </label>

      <div class="relative">
        <button type="button" class="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-600 hover:bg-slate-50" @click="showMenu = !showMenu">⋮</button>
        <div v-if="showMenu" class="absolute left-0 top-full z-10 mt-2 w-48 rounded-lg bg-white p-1 shadow-xl ring-1 ring-black/5" @mouseleave="showMenu = false">
          <button
              type="button"
              class="flex w-full px-3 py-2 text-right text-sm text-red-600 hover:bg-red-50 disabled:opacity-40"
              :disabled="isLocked || !teamList.length"
              @click="purge"
          >
            🗑️ پاک‌سازی کلی (Flush)
          </button>
        </div>
      </div>

      <div class="mr-auto text-xs text-slate-500">
        {{ toFa(teamList.length) }} تیم / {{ toFa(membersCount) }} ورزشکار
      </div>
    </div>

    <div v-if="!teamList.length" class="rounded-xl border border-slate-100 bg-slate-50/50 py-12 text-center text-slate-400">
      تیمی ثبت نشده است. از دکمه‌های بالا برای افزودن استفاده کنید.
    </div>

    <!-- Teams -->
    <div v-else class="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
      <div
          v-for="team in sortedTeams"
          :key="team.id"
          class="flex flex-col rounded-xl border border-slate-200 bg-white shadow-sm transition-shadow hover:shadow-md"
      >
        <div class="flex items-center justify-between border-b border-slate-100 bg-slate-50/40 px-3 py-2.5">
          <div class="flex min-w-0 items-center gap-2">
            <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-indigo-100 text-xs font-bold text-indigo-700">
              {{ toFa(rosterOf(team).length) }}
            </span>
            <div class="min-w-0">
              <h3 class="truncate text-sm font-bold text-slate-800">{{ team.name }}</h3>
              <p v-if="team.coach" class="truncate text-[10px] text-slate-500">مربی: {{ team.coach }}</p>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-0.5">
            <button type="button" :class="BTN.icon" title="ویرایش نام تیم" :disabled="isLocked" @click="openTeam(team)">✏️</button>
            <button type="button" :class="BTN.iconDanger" title="حذف کامل تیم" :disabled="isLocked" @click="removeTeam(team.id)">🗑️</button>
          </div>
        </div>

        <div class="flex-1 p-2.5">
          <p v-if="!rosterOf(team).length" class="py-4 text-center text-xs italic text-slate-400">عضوی ثبت نشده</p>
          <ul v-else class="space-y-1.5">
            <li
                v-for="(m, i) in sortedRoster(team)"
                :key="memberKey(m, i)"
                class="group flex items-center gap-2 rounded-lg border border-slate-100 px-2 py-1.5 text-xs hover:bg-slate-50"
            >
              <span class="w-4 shrink-0 text-center text-[10px] font-medium text-slate-400">{{ toFa(i + 1) }}</span>

              <div class="min-w-0 flex-1">
                <p class="truncate font-medium text-slate-700">{{ memberName(m) }}</p>
                <p v-if="memberNationalId(m)" class="truncate text-[9px] text-slate-400">{{ memberNationalId(m) }}</p>
              </div>

              <span
                  v-if="memberWeight(m)"
                  :title="weightTitle(memberWeight(m))"
                  class="shrink-0 rounded bg-indigo-50 px-1.5 py-0.5 text-[10px] font-medium text-indigo-600"
              >
                {{ weightBadge(memberWeight(m)) }}
              </span>
              <span
                  v-else-if="weightOptions.length"
                  class="shrink-0 rounded bg-amber-50 px-1.5 py-0.5 text-[10px] font-medium text-amber-600"
              >
                بدون وزن
              </span>

              <div class="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                <button type="button" :class="BTN.icon" title="ویرایش" :disabled="isLocked" @click="openMember(team.id, memberId(m))">✏️</button>
                <button type="button" :class="BTN.iconDanger" title="حذف از تیم" :disabled="isLocked" @click="removeMember(team.id, memberId(m))">✕</button>
              </div>
            </li>
          </ul>
        </div>

        <div class="border-t border-slate-50 p-1.5">
          <button
              type="button"
              class="w-full rounded-lg py-1.5 text-[11px] font-medium text-indigo-600 hover:bg-indigo-50 disabled:opacity-30"
              :disabled="isLocked"
              @click="openMember(team.id)"
          >
            + افزودن سریع
          </button>
        </div>
      </div>
    </div>

    <!-- مودال داخلی: تیم / ورزشکار / ایمپورت -->
    <Teleport to="body">
      <Transition
          enter-active-class="transition duration-150 ease-out"
          enter-from-class="opacity-0"
          leave-active-class="transition duration-100 ease-in"
          leave-to-class="opacity-0"
      >
        <div
            v-if="modal.type"
            dir="rtl"
            class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 p-4 backdrop-blur-sm"
            @click.self="closeModal"
        >
          <div
              role="dialog"
              aria-modal="true"
              :aria-label="TITLES[modal.type]"
              class="flex max-h-[90vh] w-full flex-col overflow-hidden rounded-2xl bg-white shadow-2xl"
              :class="modal.type === 'import' ? 'max-w-3xl' : 'max-w-md'"
          >
            <div class="flex items-center justify-between border-b border-slate-100 px-5 py-3">
              <h2 class="font-bold text-slate-800">{{ TITLES[modal.type] }}</h2>
              <button type="button" class="rounded p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600" aria-label="بستن" @click="closeModal">✕</button>
            </div>

            <form id="teamManagerForm" class="flex-1 overflow-y-auto p-5" @submit.prevent="submit">
              <!-- فرم تیم -->
              <div v-if="modal.type === 'team'" class="space-y-4">
                <label class="block">
                  <span class="mb-1 block text-sm font-medium text-slate-700">نام تیم *</span>
                  <input ref="firstField" v-model="modal.form.name" type="text" :class="BTN.input" placeholder="مثال: تهران الف" />
                </label>
                <label class="block">
                  <span class="mb-1 block text-sm font-medium text-slate-700">نام مربی / سرپرست</span>
                  <input v-model="modal.form.coach" type="text" :class="BTN.input" />
                </label>
              </div>

              <!-- فرم ورزشکار -->
              <div v-else-if="modal.type === 'member'" class="space-y-4">
                <label class="block">
                  <span class="mb-1 block text-sm font-medium text-slate-700">انتخاب تیم *</span>
                  <select ref="firstField" v-model="modal.form.teamId" :class="BTN.input" :disabled="modal.isEdit">
                    <option v-for="t in sortedTeams" :key="t.id" :value="t.id">{{ t.name }}</option>
                  </select>
                </label>

                <label class="block">
                  <span class="mb-1 block text-sm font-medium text-slate-700">نام کامل ورزشکار *</span>
                  <input v-model="modal.form.name" type="text" :class="BTN.input" placeholder="مثال: علی رضایی" />
                </label>

                <label class="block">
                  <span class="mb-1 block text-sm font-medium text-slate-700">شناسه سیستم / کد ملی (اختیاری)</span>
                  <input v-model="modal.form.nationalId" type="text" :class="BTN.input" />
                </label>

                <label v-if="weightItems.length" class="block">
                  <span class="mb-1 block text-sm font-medium text-slate-700">انتخاب وزن</span>
                  <select v-model="modal.form.weight" :class="BTN.input">
                    <option value="">نامشخص</option>
                    <option v-for="w in weightItems" :key="w.value" :value="w.value">{{ w.label }}</option>
                  </select>
                </label>

                <p class="rounded-lg bg-slate-50 px-3 py-2 text-[11px] leading-5 text-slate-500">
                  ℹ️ تعیین وزن در این مرحله اختیاری است؛ می‌توانید وزن نهایی را در تب Lineup مشخص کنید.
                </p>
              </div>

              <!-- گزارش ایمپورت -->
              <div v-else-if="modal.type === 'import'" class="space-y-4">
                <div class="max-h-72 overflow-y-auto rounded-lg bg-slate-50 p-4 font-mono text-xs leading-relaxed text-slate-600">
                  <div
                      v-for="(log, i) in modal.logs"
                      :key="i"
                      :class="log.startsWith('❌') ? 'text-red-600' : (log.startsWith('✅') ? 'text-emerald-600' : '')"
                  >
                    {{ log }}
                  </div>
                </div>
                <div v-if="importing" class="flex items-center gap-2 text-indigo-600">
                  <span class="animate-spin text-lg">⏳</span>
                  <span class="text-sm">در حال پردازش سطرها…</span>
                </div>
              </div>
            </form>

            <div class="flex items-center justify-end gap-2 border-t border-slate-100 bg-slate-50 p-3">
              <button type="button" class="rounded-lg px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-200" @click="closeModal">
                {{ modal.type === 'import' ? 'بستن' : 'انصراف' }}
              </button>
              <button
                  v-if="modal.type !== 'import'"
                  form="teamManagerForm"
                  type="submit"
                  :class="BTN.primary"
                  :disabled="!isValid"
              >
                {{ modal.isEdit ? 'بروزرسانی' : 'ثبت نهایی' }}
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { reactive, ref, computed, nextTick } from 'vue';
import { storeToRefs } from 'pinia';
import { useTeamTournamentStore } from '../../stores/teamTournament';
import * as XLSX from 'xlsx';
import { WEIGHT_CATEGORIES } from '../../data/categories.js';

/* ---------- استایل‌های ثابت ---------- */
const BTN = {
  primary:
      'bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 text-sm font-medium transition disabled:opacity-50 disabled:cursor-not-allowed',
  soft:
      'bg-indigo-50 text-indigo-700 px-4 py-2 rounded-lg hover:bg-indigo-100 text-sm font-medium transition disabled:opacity-50 disabled:cursor-not-allowed',
  icon:
      'w-6 h-6 flex items-center justify-center rounded-md text-slate-400 hover:text-indigo-600 hover:bg-indigo-50 transition disabled:opacity-30',
  iconDanger:
      'w-6 h-6 flex items-center justify-center rounded-md text-slate-400 hover:text-red-600 hover:bg-red-50 transition disabled:opacity-30',
  input:
      'w-full rounded-lg border border-slate-300 px-3 py-2 shadow-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-sm disabled:bg-slate-100',
};

const TITLES = {
  team: 'مدیریت تیم',
  member: 'مدیریت ورزشکار',
  import: 'گزارش ایمپورت اکسل',
};

const ORDINALS = [
  'اول', 'دوم', 'سوم', 'چهارم', 'پنجم', 'ششم', 'هفتم', 'هشتم', 'نهم', 'دهم',
  'یازدهم', 'دوازدهم', 'سیزدهم', 'چهاردهم', 'پانزدهم', 'شانزدهم', 'هفدهم', 'هجدهم', 'نوزدهم', 'بیستم',
];

/* ---------- استور ---------- */
const store = useTeamTournamentStore();
const { current } = storeToRefs(store);

/* ---------- استیت ---------- */
const feedback = ref(null);
const importing = ref(false);
const showMenu = ref(false);
const firstField = ref(null);
const fileInput = ref(null);

const modal = reactive({
  type: null,
  isEdit: false,
  form: {},
  logs: [],
});

/* ---------- ابزارهای پایه ---------- */
const toFa = (n) => String(n ?? 0).replace(/\d/g, (d) => '۰۱۲۳۴۵۶۷۸۹'[d]);

const toEnDigits = (s) =>
    String(s ?? '').replace(/[۰-۹]/g, (d) => '0123456789'['۰۱۲۳۴۵۶۷۸۹'.indexOf(d)]);

function weightNum(w) {
  const m = toEnDigits(w).match(/\d+(\.\d+)?/);
  return m ? parseFloat(m[0]) : Number.POSITIVE_INFINITY;
}

const ordinalOf = (i) => ORDINALS[i] ?? `${toFa(i + 1)}ام`;

/* ---------- کامپیوتدها ---------- */
const tid = computed(() => current.value?.id ?? null);
const teamList = computed(() => current.value?.teams ?? []);
const slotList = computed(() => current.value?.weightSlots ?? []);
const athleteList = computed(() => current.value?.athletes ?? []);

/* لیست اوزان مرتب‌شده از سبک به سنگین */
const weightOptions = computed(() => {
  const t = current.value;
  if (!t) return [];
  const raw = WEIGHT_CATEGORIES[t.ageCategory]?.[t.gender] ?? [];
  return [...raw].sort((a, b) => weightNum(a) - weightNum(b));
});

/* لیست غنی‌شده برای نمایش در سلکت */
const weightItems = computed(() =>
    weightOptions.value.map((w, i) => ({
      value: w,
      index: i + 1,
      ordinal: ordinalOf(i),
      label: `وزن ${ordinalOf(i)} — ${w}`,
    }))
);

/* نگاشت وزن → شماره ترتیبی */
const weightRank = computed(() => {
  const map = new Map();
  weightOptions.value.forEach((w, i) => map.set(String(w), i + 1));
  return map;
});

const isLocked = computed(() => {
  const t = current.value;
  if (!t) return false;
  return t.rosterLocked === true || t.teamsLocked === true || t.locked === true;
});

const membersCount = computed(() =>
    teamList.value.reduce((acc, t) => acc + rosterOf(t).length, 0)
);

const sortedTeams = computed(() =>
    [...teamList.value].sort((a, b) => String(a.name).localeCompare(String(b.name), 'fa'))
);

const isValid = computed(() => {
  if (modal.type === 'team') return !!String(modal.form.name || '').trim();
  if (modal.type === 'member')
    return !!modal.form.teamId && !!String(modal.form.name || '').trim();
  return false;
});

/* ---------- ابزارهای وزن ---------- */
function weightBadge(w) {
  if (!w) return '';
  const r = weightRank.value.get(String(w));
  return r ? `وزن ${toFa(r)}  ${w}` : String(w);
}

function weightTitle(w) {
  if (!w) return '';
  const r = weightRank.value.get(String(w));
  return r ? `وزن ${ordinalOf(r - 1)} (${w})` : String(w);
}

/* ---------- ابزارهای عضو ---------- */
function rosterOf(team) {
  return team?.roster ?? team?.members ?? [];
}

function memberId(m) {
  return typeof m === 'string' || typeof m === 'number' ? m : (m?.athleteId ?? m?.id ?? null);
}

function memberKey(m, i) {
  return memberId(m) ?? `idx-${i}`;
}

const athleteById = (athleteId) =>
    athleteList.value.find((a) => String(a.id) === String(athleteId)) || null;

function memberName(m) {
  const direct = typeof m === 'object' && m?.name ? String(m.name).trim() : '';
  if (direct) return direct;
  return athleteById(memberId(m))?.name || '—';
}

function memberWeight(m) {
  const direct = typeof m === 'object' ? (m?.weight || '') : '';
  if (direct) return direct;
  return athleteById(memberId(m))?.weight || '';
}

function memberNationalId(m) {
  const direct = typeof m === 'object' ? (m?.nationalId || m?.national_id || '') : '';
  if (direct) return direct;
  const a = athleteById(memberId(m));
  return a?.nationalId || a?.national_id || '';
}

/* روستر مرتب بر اساس ترتیب وزن، سپس نام */
function sortedRoster(team) {
  const rank = weightRank.value;
  return [...rosterOf(team)].sort((a, b) => {
    const wa = memberWeight(a);
    const wb = memberWeight(b);
    const ra = rank.get(String(wa)) ?? (wa ? 900 + weightNum(wa) : 999);
    const rb = rank.get(String(wb)) ?? (wb ? 900 + weightNum(wb) : 999);
    if (ra !== rb) return ra - rb;
    return String(memberName(a)).localeCompare(String(memberName(b)), 'fa');
  });
}

/* ---------- بازخورد ---------- */
function notify(type, message) {
  feedback.value = { type, message };
  if (type !== 'error') setTimeout(() => (feedback.value = null), 4000);
}

function call(res, okMsg) {
  if (res && typeof res === 'object' && res.error) {
    notify('error', res.error);
    return null;
  }
  if (okMsg) notify('success', okMsg);
  return res ?? true;
}

function guard() {
  if (!tid.value) {
    notify('error', 'ابتدا یک مسابقهٔ تیمی را انتخاب کنید.');
    return false;
  }
  if (isLocked.value) {
    notify('error', 'روستر قفل شده است.');
    return false;
  }
  return true;
}

/* ---------- مودال ---------- */
function closeModal() {
  modal.type = null;
  modal.isEdit = false;
  modal.form = {};
  modal.logs = [];
}

async function openTeam(existing = null) {
  if (!guard()) return;
  modal.type = 'team';
  modal.isEdit = !!existing;
  modal.form = existing
      ? { id: existing.id, name: existing.name, coach: existing.coach || '' }
      : { name: '', coach: '' };
  await nextTick();
  firstField.value?.focus();
}

async function openMember(teamId = null, existingAthleteId = null) {
  if (!guard()) return;
  const a = existingAthleteId ? athleteById(existingAthleteId) : null;

  modal.type = 'member';
  modal.isEdit = !!a;
  modal.form = a
      ? {
        athleteId: a.id,
        teamId: teamId || sortedTeams.value[0]?.id || null,
        name: a.name || '',
        weight: a.weight || '',
        nationalId: a.nationalId || a.national_id || '',
      }
      : {
        teamId: teamId || sortedTeams.value[0]?.id || null,
        name: '',
        weight: '',
        nationalId: '',
      };
  await nextTick();
  firstField.value?.focus();
}

/* ---------- ثبت ---------- */
function submit() {
  if (!isValid.value || !guard()) return;

  if (modal.type === 'team') {
    const payload = {
      name: String(modal.form.name).trim(),
      coach: String(modal.form.coach || '').trim(),
    };
    const res = modal.isEdit
        ? store.updateTeam(tid.value, modal.form.id, payload)
        : store.addTeam(tid.value, payload);
    if (call(res, 'تیم ذخیره شد.')) closeModal();
    return;
  }

  if (modal.type === 'member') {
    const name = String(modal.form.name).trim();
    const weight = String(modal.form.weight || '').trim();
    const nationalId = String(modal.form.nationalId || '').trim();

    if (modal.isEdit) {
      const res = store.updateAthlete(tid.value, modal.form.athleteId, { name, weight, nationalId });
      if (call(res, 'ورزشکار به‌روزرسانی شد.')) closeModal();
      return;
    }

    const athlete = resolveAthlete({ name, weight, nationalId });
    if (!athlete) return;

    const res = store.addMember(tid.value, modal.form.teamId, { athleteId: athlete.id });
    if (call(res, 'ورزشکار به تیم افزوده شد.')) closeModal();
  }
}

function resolveAthlete({ name, weight, nationalId }) {
  const found = athleteList.value.find((a) => {
    const aNid = a.nationalId || a.national_id || '';
    if (nationalId && aNid) return String(aNid) === String(nationalId);
    return String(a.name).trim() === name;
  });

  if (found) {
    if (weight && !found.weight) store.updateAthlete(tid.value, found.id, { weight });
    return found;
  }

  const res = store.addAthlete(tid.value, { name, weight, nationalId });
  if (res?.error) { notify('error', res.error); return null; }
  return res;
}

/* ---------- حذف ---------- */
function removeTeam(teamId) {
  if (!guard()) return;
  if (!confirm('حذف کامل این تیم و روستر آن؟')) return;
  call(store.removeTeam(tid.value, teamId), 'تیم حذف شد.');
}

function removeMember(teamId, athleteId) {
  if (!guard()) return;
  if (!confirm('این ورزشکار از تیم حذف شود؟')) return;
  call(store.removeMember(tid.value, teamId, athleteId));
}

function purge() {
  showMenu.value = false;
  if (!guard()) return;
  if (!confirm('هشدار: تمام تیم‌ها و روسترها پاک خواهند شد. مطمئن هستید؟')) return;
  call(store.clearTeams(tid.value), 'همهٔ تیم‌ها پاک شدند.');
}

/* ---------- ایمپورت اکسل ---------- */
function pick(row, keys) {
  for (const k of keys) {
    if (row[k] !== undefined && row[k] !== null && String(row[k]).trim() !== '') {
      return String(row[k]).trim();
    }
  }
  return '';
}

function normalizeWeight(val) {
  if (!val) return '';
  let s = toEnDigits(val).replace(/[^0-9.]/g, '');
  if (!s) return '';
  const num = parseFloat(s);
  if (isNaN(num)) return '';
  const official = weightOptions.value.find(
      (w) => String(w).replace(/[^0-9.]/g, '') === String(num)
  );
  return official || `-${num}`;
}

function readExcel(e) {
  const file = e.target?.files?.[0];
  if (!file) return;
  if (!guard()) { if (fileInput.value) fileInput.value.value = ''; return; }

  importing.value = true;
  modal.type = 'import';
  modal.logs = [`📄 فایل: ${file.name}`, 'در حال خواندن…'];

  const reader = new FileReader();
  reader.onerror = () => { modal.logs.push('❌ خطا در خواندن فایل.'); importing.value = false; };
  reader.onload = (ev) => {
    try {
      const data = new Uint8Array(ev.target.result);
      const wb = XLSX.read(data, { type: 'array' });
      const rows = XLSX.utils.sheet_to_json(wb.Sheets[wb.SheetNames[0]], { defval: '' });

      let created = 0, newTeams = 0, errors = 0;

      rows.forEach((row) => {
        const tName = pick(row, ['team', 'تیم', 'باشگاه']);
        const mName = pick(row, ['name', 'نام', 'ورزشکار']);
        const wVal = normalizeWeight(pick(row, ['weight', 'وزن']));
        const nid = pick(row, ['national_id', 'nationalId', 'کد ملی']);

        if (!tName || !mName) { errors++; return; }

        let team = teamList.value.find((t) => t.name === tName);
        if (!team) {
          store.addTeam(tid.value, { name: tName, coach: pick(row, ['coach', 'مربی']) });
          team = teamList.value.find((t) => t.name === tName);
          newTeams++;
        }

        const athlete = resolveAthlete({ name: mName, weight: wVal, nationalId: nid });
        if (athlete && team) {
          const res = store.addMember(tid.value, team.id, { athleteId: athlete.id });
          if (res?.error) errors++; else created++;
        } else {
          errors++;
        }
      });

      modal.logs.push(`✅ پایان: ${toFa(newTeams)} تیم جدید، ${toFa(created)} ورزشکار ثبت شد.`);
      if (errors) modal.logs.push(`❌ ${toFa(errors)} سطر نادیده گرفته شد.`);
    } catch (err) {
      modal.logs.push(`❌ خطا: ${err.message}`);
    } finally {
      importing.value = false;
      if (fileInput.value) fileInput.value.value = '';
    }
  };
  reader.readAsArrayBuffer(file);
}
</script>
