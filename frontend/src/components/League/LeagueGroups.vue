<script setup lang="ts">
import { computed, ref } from "vue";
import { useLeagueStore } from "../../stores/league.ts";
import { useTournamentStore } from "../../stores/tournament";
import type { LeagueClub } from "../../types";
import { loadWebLeagues, webApi } from "../../webApi";

const leagueStore = useLeagueStore();
const tournamentStore = useTournamentStore();

const league = computed(() => leagueStore.currentLeague);
const leagueId = computed(() => league.value?.id ?? "");

const groups = computed(() =>
    [...(league.value?.season.groups ?? [])].sort((a, b) => a.order - b.order)
);
const clubs = computed<LeagueClub[]>(() => league.value?.clubs ?? []);

/** نام باشگاه → شناسهٔ گروه */
const groupOfClub = computed<Record<string, string>>(() => {
  const map: Record<string, string> = {};
  for (const g of groups.value) for (const c of g.clubs) map[c] = g.id;
  return map;
});

const unassigned = computed(() =>
    clubs.value.filter((c) => !groupOfClub.value[c.name])
);

/* ─────────────── پیام وضعیت ─────────────── */

type Notice = { type: "ok" | "error"; text: string };
const notice = ref<Notice | null>(null);
let noticeTimer: number | undefined;

function showNotice(type: Notice["type"], text: string) {
  notice.value = { type, text };
  if (noticeTimer) window.clearTimeout(noticeTimer);
  noticeTimer = window.setTimeout(() => (notice.value = null), 4000);
}

/* ─────────────── گروه‌ها ─────────────── */

const editingGroupId = ref<string | null>(null);
const editingGroupName = ref("");

async function refreshLeague() { leagueStore.replaceFromBackend(await loadWebLeagues()) }

async function onAddGroup() {
  try {
    await webApi().call(`/leagues/${leagueId.value}/groups`, 'POST', { name: `گروه ${groups.value.length + 1}` })
    await refreshLeague(); showNotice('ok', 'گروه اضافه شد')
  } catch (e: any) { showNotice('error', e?.message || 'امکان افزودن گروه جدید نیست') }
}

function startRenameGroup(id: string, name: string) {
  editingGroupId.value = id;
  editingGroupName.value = name;
}

async function commitRenameGroup() {
  const id = editingGroupId.value;
  const name = editingGroupName.value.trim();
  editingGroupId.value = null;
  if (!id || !name) return;
  try { await webApi().call(`/leagues/${leagueId.value}/groups/${id}`, 'PUT', { name }); await refreshLeague(); showNotice('ok', 'نام گروه تغییر کرد') }
  catch (e: any) { showNotice('error', e?.message || 'تغییر نام گروه انجام نشد') }
}

async function onRemoveGroup(id: string, name: string) {
  if (!window.confirm(`گروه «${name}» حذف شود؟ باشگاه‌هایش بی‌گروه می‌شوند.`)) return;
  try { await webApi().call(`/leagues/${leagueId.value}/groups/${id}`, 'DELETE'); await refreshLeague(); showNotice('ok', 'گروه حذف شد') }
  catch (e: any) { showNotice('error', e?.message || 'حذف گروه انجام نشد') }
}

async function onAssign(clubId: string, groupId: string) {
  const club = clubs.value.find(c => c.id === clubId) as (LeagueClub & { teamId?: string }) | undefined
  if (!club?.teamId) return showNotice('error', 'تیم لیگ پیدا نشد')
  try { await webApi().call(`/leagues/${leagueId.value}/teams/${club.teamId}`, 'PUT', { groupId: groupId || null, active: club.active !== false }); await refreshLeague(); showNotice('ok', groupId ? 'باشگاه جابه‌جا شد' : 'باشگاه از گروه خارج شد') }
  catch (e: any) { showNotice('error', e?.message || 'جابه‌جایی انجام نشد') }
}

const distributeAll = ref(false);

async function onDistribute() {
  if (groups.value.length === 0) {
    showNotice("error", "اول باید حداقل یک گروه بسازی");
    return;
  }
  if (
      distributeAll.value &&
      !window.confirm("همهٔ باشگاه‌ها از نو بین گروه‌ها پخش می‌شوند. ادامه؟")
  )
    return;

  try {
    const targets = clubs.value.filter(c => c.active !== false && (distributeAll.value || !c.groupId)) as Array<LeagueClub & { teamId?: string }>
    await Promise.all(targets.map((c, i) => c.teamId ? webApi().call(`/leagues/${leagueId.value}/teams/${c.teamId}`, 'PUT', { groupId: groups.value[i % groups.value.length].id, active: true }) : Promise.resolve()))
    await refreshLeague(); showNotice('ok', 'توزیع انجام شد')
  } catch (e: any) { showNotice('error', e?.message || 'توزیع انجام نشد') }
}

/* ─────────────── باشگاه‌ها ─────────────── */

const newClubName = ref("");
const clubSearch = ref("");
const showInactive = ref(true);

const filteredClubs = computed(() => {
  const q = clubSearch.value.trim();
  return clubs.value.filter((c) => {
    if (!showInactive.value && c.active === false) return false;
    return q ? c.name.includes(q) : true;
  });
});

async function onAddClub() {
  const name = newClubName.value.trim();
  if (!name) return;

  try {
    const club = await webApi().call<any>('/clubs', 'POST', { name })
    await webApi().call(`/leagues/${leagueId.value}/teams`, 'POST', { clubId: club.id, groupId: null })
    await refreshLeague(); newClubName.value = ''; showNotice('ok', `باشگاه «${name}» اضافه شد`)
  } catch (e: any) { showNotice('error', e?.message || `باشگاه «${name}» اضافه نشد`) }
}


const editingClub = ref<string | null>(null);
const editingClubName = ref("");

function startRenameClub(name: string) {
  editingClub.value = name;
  editingClubName.value = name;
}

async function commitRenameClub() {
  const oldName = editingClub.value;
  const newName = editingClubName.value.trim();
  editingClub.value = null;
  if (!oldName || !newName || oldName === newName) return;
  const club = clubs.value.find(c => c.name === oldName)
  if (!club) return
  try { await webApi().call(`/clubs/${club.id}`, 'PUT', { name: newName }); await refreshLeague(); showNotice('ok', 'نام باشگاه در همهٔ رکوردها به‌روز شد') }
  catch (e: any) { showNotice('error', e?.message || 'تغییر نام باشگاه انجام نشد') }
}

async function onToggleActive(name: string, active: boolean) {
  const club = clubs.value.find(c => c.name === name) as (LeagueClub & { teamId?: string }) | undefined
  if (!club?.teamId) return
  try { await webApi().call(`/leagues/${leagueId.value}/teams/${club.teamId}`, 'PUT', { groupId: club.groupId ?? null, active }); await refreshLeague(); showNotice('ok', active ? `«${name}» فعال شد` : `«${name}» غیرفعال شد`) }
  catch (e: any) { showNotice('error', e?.message || 'تغییر وضعیت انجام نشد') }
}

async function onRemoveClub(name: string) {
  if (!window.confirm(`باشگاه «${name}» حذف شود؟`)) return;

  await onToggleActive(name, false)
}

/* ─────────────── ورود گروهی باشگاه ─────────────── */

const bulkOpen = ref(false);
const bulkText = ref("");

async function onBulkImport() {
  const text = bulkText.value.trim();
  if (!text) return;
  const names = [...new Set(text.split(/[\n,،;]/).map(v => v.trim()).filter(Boolean))]
  const existing = new Set(clubs.value.map(c => c.name))
  let added = 0, skipped = 0
  try {
    for (const name of names) {
      if (existing.has(name)) { skipped++; continue }
      const club = await webApi().call<any>('/clubs', 'POST', { name })
      await webApi().call(`/leagues/${leagueId.value}/teams`, 'POST', { clubId: club.id, groupId: null })
      added++
    }
    await refreshLeague(); bulkOpen.value = false; bulkText.value = ''
    showNotice(added ? 'ok' : 'error', added ? `${added} باشگاه اضافه شد، ${skipped} تکراری رد شد` : 'همهٔ نام‌ها تکراری بودند')
  } catch (e: any) { await refreshLeague(); showNotice('error', e?.message || 'ورود گروهی کامل نشد') }
}

async function onSyncFromTournaments() {
  const existing = new Set(clubs.value.map(c => c.name))
  const names = [...new Set(tournamentStore.tournaments.filter(t => t.leagueId === leagueId.value).flatMap(t => t.athletes.map(a => a.club?.trim()).filter((v): v is string => !!v && v !== 'آزاد')))].filter(name => !existing.has(name))
  try {
    for (const name of names) {
      const club = await webApi().call<any>('/clubs', 'POST', { name })
      await webApi().call(`/leagues/${leagueId.value}/teams`, 'POST', { clubId: club.id, groupId: null })
    }
    await refreshLeague(); showNotice('ok', names.length ? `${names.length} باشگاه از تورنمنت‌ها اضافه شد` : 'باشگاه تازه‌ای در تورنمنت‌ها نبود')
  } catch (e: any) { await refreshLeague(); showNotice('error', e?.message || 'همگام‌سازی باشگاه‌ها کامل نشد') }
}


/* ─────────────── رتبه‌بندی و صعود ─────────────── */

const selectedStage = ref<number | null>(null);
const perGroup = ref(league.value?.promotionRules.teamsToPromote ?? 1);

const stageOrder = computed(() => selectedStage.value ?? undefined);

const rankings = computed(() =>
    leagueStore.getGroupTeamRankings(leagueId.value, stageOrder.value)
);

const promoted = computed(
    () =>
        new Set(
            leagueStore.getPromotionCandidates(
                leagueId.value,
                stageOrder.value,
                perGroup.value
            )
        )
);
</script>

<template>
  <div v-if="league" class="space-y-6">
    <!-- نوار اکشن -->
    <div class="flex flex-wrap items-center gap-2 rounded-xl border border-slate-200 bg-white p-4">
      <button
          type="button"
          class="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-emerald-700"
          @click="onAddGroup"
      >
        ➕ گروه جدید
      </button>

      <button
          type="button"
          class="rounded-lg bg-slate-200 px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-300"
          @click="onDistribute"
      >
        🔀 توزیع خودکار
      </button>

      <label class="flex cursor-pointer items-center gap-2 text-sm text-slate-600">
        <input v-model="distributeAll" type="checkbox" />
        بازتوزیع همهٔ باشگاه‌ها
      </label>

      <span class="mx-1 h-6 w-px bg-slate-200"></span>

      <button
          type="button"
          class="rounded-lg bg-slate-200 px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-300"
          @click="onSyncFromTournaments"
      >
        🔄 خواندن باشگاه‌ها از تورنمنت‌ها
      </button>

      <button
          type="button"
          class="rounded-lg bg-slate-200 px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-300"
          @click="bulkOpen = true"
      >
        📋 ورود گروهی
      </button>

      <span class="ms-auto text-sm text-slate-500">
        {{ clubs.length }} باشگاه • {{ groups.length }} گروه • {{ unassigned.length }} بی‌گروه
      </span>
    </div>

    <p
        v-if="notice"
        aria-live="polite"
        :class="[
          'rounded-lg px-4 py-2 text-sm',
          notice.type === 'ok' ? 'bg-emerald-50 text-emerald-700' : 'bg-rose-50 text-rose-700',
        ]"
    >
      {{ notice.text }}
    </p>

    <div class="grid gap-6 lg:grid-cols-[22rem_1fr]">
      <!-- ستون باشگاه‌ها -->
      <section class="rounded-xl border border-slate-200 bg-white p-4">
        <h3 class="mb-3 font-bold">🏛️ باشگاه‌ها</h3>

        <form class="flex gap-2" @submit.prevent="onAddClub">
          <label class="sr-only" for="new-club">نام باشگاه</label>
          <input
              id="new-club"
              v-model="newClubName"
              type="text"
              placeholder="نام باشگاه جدید"
              class="flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm"
          />
          <button
              type="submit"
              class="rounded-lg bg-emerald-600 px-3 py-2 text-sm font-medium text-white transition hover:bg-emerald-700"
          >
            افزودن
          </button>
        </form>

        <div class="mt-3 flex items-center gap-2">
          <label class="sr-only" for="club-search">جست‌وجو</label>
          <input
              id="club-search"
              v-model="clubSearch"
              type="search"
              placeholder="🔍 جست‌وجو"
              class="flex-1 rounded-lg border border-slate-300 px-3 py-1.5 text-sm"
          />
          <label class="flex cursor-pointer items-center gap-1 text-xs text-slate-600">
            <input v-model="showInactive" type="checkbox" />
            غیرفعال‌ها
          </label>
        </div>

        <ul class="mt-3 max-h-[32rem] divide-y divide-slate-100 overflow-y-auto">
          <li v-for="club in filteredClubs" :key="club.name" class="py-2">
            <div class="flex items-center gap-2">
              <template v-if="editingClub === club.name">
                <input
                    v-model="editingClubName"
                    type="text"
                    class="flex-1 rounded border border-emerald-400 px-2 py-1 text-sm"
                    @keydown.enter="commitRenameClub"
                    @keydown.esc="editingClub = null"
                    @blur="commitRenameClub"
                />
              </template>
              <template v-else>
                <span
                    :class="[
                      'flex-1 truncate text-sm',
                      club.active === false ? 'text-slate-400 line-through' : 'text-slate-800',
                    ]"
                >
                  {{ club.name }}
                </span>
                <button
                    type="button"
                    :title="`ویرایش نام ${club.name}`"
                    class="text-xs text-slate-400 transition hover:text-emerald-600"
                    @click="startRenameClub(club.name)"
                >
                  ✏️
                </button>
                <button
                    type="button"
                    :title="club.active === false ? 'فعال کردن' : 'غیرفعال کردن'"
                    class="text-xs text-slate-400 transition hover:text-amber-600"
                    @click="onToggleActive(club.name, club.active === false)"
                >
                  {{ club.active === false ? "☑️" : "🚫" }}
                </button>
                <button
                    type="button"
                    :title="`حذف ${club.name}`"
                    class="text-xs text-slate-400 transition hover:text-rose-600"
                    @click="onRemoveClub(club.name)"
                >
                  🗑️
                </button>
              </template>
            </div>

            <label class="mt-1 block">
              <span class="sr-only">گروه {{ club.name }}</span>
              <select
                  :value="groupOfClub[club.name] ?? ''"
                  class="w-full rounded border border-slate-200 bg-slate-50 px-2 py-1 text-xs text-slate-600"
                  @change="onAssign(club.id, ($event.target as HTMLSelectElement).value)"
              >
                <option value="">— بی‌گروه —</option>
                <option v-for="g in groups" :key="g.id" :value="g.id">
                  {{ g.name }}
                </option>
              </select>
            </label>
          </li>

          <li v-if="filteredClubs.length === 0" class="py-6 text-center text-sm text-slate-400">
            باشگاهی نیست
          </li>
        </ul>
      </section>

      <!-- ستون گروه‌ها -->
      <section class="space-y-4">
        <div class="flex flex-wrap items-center gap-3 rounded-xl border border-slate-200 bg-white p-4">
          <label class="text-sm text-slate-600">
            مبنای جدول
            <select
                v-model="selectedStage"
                class="ms-2 rounded-lg border border-slate-300 px-2 py-1 text-sm"
            >
              <option :value="null">کل فصل</option>
              <option v-for="stage in league.season.stages" :key="stage.order" :value="stage.order">
                {{ stage.name }}
              </option>
            </select>
          </label>

          <label class="text-sm text-slate-600">
            صعود از هر گروه
            <input
                v-model.number="perGroup"
                type="number"
                min="0"
                class="ms-2 w-16 rounded-lg border border-slate-300 px-2 py-1 text-sm"
            />
          </label>

          <span class="ms-auto text-sm text-slate-500">
            {{ promoted.size }} تیم صعودکننده
          </span>
        </div>

        <p
            v-if="groups.length === 0"
            class="rounded-xl border border-dashed border-slate-300 p-10 text-center text-sm text-slate-500"
        >
          هنوز گروهی ساخته نشده. با «گروه جدید» شروع کن، بعد «توزیع خودکار» بزن.
        </p>

        <div v-else class="grid gap-4 xl:grid-cols-2">
          <article
              v-for="g in rankings"
              :key="g.groupId"
              class="overflow-hidden rounded-xl border border-slate-200 bg-white"
          >
            <header class="flex items-center gap-2 border-b border-slate-200 bg-slate-50 px-3 py-2">
              <template v-if="editingGroupId === g.groupId">
                <input
                    v-model="editingGroupName"
                    type="text"
                    class="flex-1 rounded border border-emerald-400 px-2 py-1 text-sm"
                    @keydown.enter="commitRenameGroup"
                    @keydown.esc="editingGroupId = null"
                    @blur="commitRenameGroup"
                />
              </template>
              <template v-else>
                <h4 class="flex-1 truncate text-sm font-bold text-slate-800">
                  {{ g.groupName }}
                  <span class="ms-1 text-xs font-normal text-slate-500">({{ g.rows.length }} باشگاه)</span>
                </h4>
                <button type="button" :title="`ویرایش نام ${g.groupName}`"
                        class="text-xs text-slate-400 hover:text-emerald-600"
                        @click="startRenameGroup(g.groupId, g.groupName)">✏️</button>
                <button type="button" :title="`حذف ${g.groupName}`"
                        class="text-xs text-slate-400 hover:text-rose-600"
                        @click="onRemoveGroup(g.groupId, g.groupName)">🗑️</button>
              </template>
            </header>

            <table class="w-full text-sm">
              <caption class="sr-only">جدول رتبه‌بندی {{ g.groupName }}</caption>
              <thead class="bg-white text-xs text-slate-500">
              <tr>
                <th scope="col" class="px-2 py-2 text-center">#</th>
                <th scope="col" class="px-3 py-2 text-right">باشگاه</th>
                <th scope="col" class="px-2 py-2 text-center">امتیاز</th>
                <th scope="col" class="px-2 py-2 text-center">🥇</th>
                <th scope="col" class="px-2 py-2 text-center">🥈</th>
                <th scope="col" class="px-2 py-2 text-center">🥉</th>
                <th scope="col" class="px-2 py-2 text-center">برد/باخت</th>
                <th scope="col" class="w-8"></th>
              </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
              <tr
                  v-for="(row, index) in g.rows"
                  :key="row.club"
                  :class="promoted.has(row.club) ? 'bg-emerald-50/60' : ''"
              >
                <td class="px-2 py-2 text-center text-slate-400">{{ index + 1 }}</td>
                <td class="px-3 py-2 font-medium">
                  {{ row.club }}
                  <span v-if="promoted.has(row.club)" class="ms-1 text-xs text-emerald-600">▲</span>
                </td>
                <td class="px-2 py-2 text-center font-bold text-emerald-700">
                  {{ row.totalPoints }}
                </td>
                <td class="px-2 py-2 text-center">{{ row.gold }}</td>
                <td class="px-2 py-2 text-center">{{ row.silver }}</td>
                <td class="px-2 py-2 text-center">{{ row.bronze }}</td>
                <td class="px-2 py-2 text-center text-xs text-slate-500">
                  {{ row.totalWins }}/{{ row.totalLosses }}
                </td>
                <td class="pe-2 text-center">
                  <button
                      type="button"
                      :title="`خارج کردن ${row.club} از ${g.groupName}`"
                      class="text-xs text-slate-300 transition hover:text-rose-600"
                      @click="onAssign(row.club, '')"
                  >
                    ✕
                  </button>
                </td>
              </tr>
              <tr v-if="g.rows.length === 0">
                <td colspan="8" class="py-6 text-center text-sm text-slate-400">
                  باشگاهی در این گروه نیست
                </td>
              </tr>
              </tbody>
            </table>
          </article>
        </div>

        <!-- بی‌گروه‌ها -->
        <div
            v-if="unassigned.length > 0"
            class="rounded-xl border border-amber-200 bg-amber-50 p-4"
        >
          <h4 class="text-sm font-bold text-amber-800">
            بی‌گروه ({{ unassigned.length }})
          </h4>
          <div class="mt-2 flex flex-wrap gap-2">
            <span
                v-for="club in unassigned"
                :key="club.name"
                class="rounded-full bg-white px-3 py-1 text-xs text-slate-600"
            >
              {{ club.name }}
            </span>
          </div>
        </div>
      </section>
    </div>

    <!-- مودال ورود گروهی -->
    <div
        v-if="bulkOpen"
        class="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/50 p-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="bulk-title"
        @click.self="bulkOpen = false"
    >
      <div class="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
        <h3 id="bulk-title" class="text-lg font-bold">📋 ورود گروهی باشگاه‌ها</h3>
        <p class="mt-1 text-sm text-slate-500">
          هر خط یک نام. نام‌های تکراری خودکار رد می‌شوند.
        </p>

        <label class="sr-only" for="bulk-text">فهرست باشگاه‌ها</label>
        <textarea
            id="bulk-text"
            v-model="bulkText"
            rows="8"
            class="mt-4 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
            placeholder="باشگاه الف&#10;باشگاه ب"
        ></textarea>

        <div class="mt-5 flex justify-end gap-2">
          <button
              type="button"
              class="rounded-lg bg-slate-200 px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-300"
              @click="bulkOpen = false"
          >
            انصراف
          </button>
          <button
              type="button"
              class="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-emerald-700"
              @click="onBulkImport"
          >
            افزودن
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
