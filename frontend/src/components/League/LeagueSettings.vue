<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useLeagueStore } from "../../stores/league";
import { loadWebLeagues, webApi } from "../../webApi";

const leagueStore = useLeagueStore();
const currentLeague = computed(() => leagueStore.currentLeague);

// تنظیمات امتیاز
const goldPoints = ref(10);
const silverPoints = ref(6);
const bronzePoints = ref(3);
const weighInPoint = ref(1);
const winPoint = ref(2);
const countWeighIn = ref(true);
const countWin = ref(true);

// قوانین صعود
const teamsToPromote = ref(2);
const autoPromote = ref(true);

// لود تنظیمات فعلی
watch(
    () => currentLeague.value?.id,
    () => {
      if (!currentLeague.value) return;
      const config = currentLeague.value.season.scoringConfig;
      goldPoints.value = config.gold;
      silverPoints.value = config.silver;
      bronzePoints.value = config.bronze;
      weighInPoint.value = config.weighInPoint;
      winPoint.value = config.winPoint;
      countWeighIn.value = config.countWeighIn;
      countWin.value = config.countWin;
      teamsToPromote.value = currentLeague.value.promotionRules.teamsToPromote;
      autoPromote.value = currentLeague.value.promotionRules.autoPromote;
    },
    { immediate: true }
);

async function saveScoringConfig() {
  if (!currentLeague.value) return;
  try {
    await webApi().call(`/leagues/${currentLeague.value.id}/settings`, 'PUT', {
      scoring: { gold: goldPoints.value, silver: silverPoints.value, bronze: bronzePoints.value, weighInPoint: weighInPoint.value, winPoint: winPoint.value, countWeighIn: countWeighIn.value, countWin: countWin.value },
      teamsToPromote: teamsToPromote.value, autoPromote: autoPromote.value,
    })
    leagueStore.replaceFromBackend(await loadWebLeagues())
    alert("تنظیمات امتیاز ذخیره شد");
  } catch (e: any) { alert(e?.message || 'ذخیره تنظیمات ناموفق بود') }
}

async function promoteTeamsNow() {
  if (!currentLeague.value) return;
  const stage = currentLeague.value.season.stages.find(s => !s.completed)
  if (!stage) return alert('مرحله فعالی وجود ندارد')
  try { await webApi().call(`/leagues/${currentLeague.value.id}/stages/${stage.id}/complete`, 'POST'); leagueStore.replaceFromBackend(await loadWebLeagues()); alert(`مرحله ${stage.name} تکمیل شد`) }
  catch (e: any) { alert(e?.message || 'تکمیل مرحله ناموفق بود') }
}
</script>

<template>
  <div class="space-y-6">
    <!-- تنظیمات امتیاز -->
    <div class="rounded-2xl border border-slate-200 bg-white p-6">
      <h3 class="mb-4 text-lg font-bold">🎯 تنظیمات امتیازدهی</h3>
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div>
          <label class="mb-1 block text-sm font-medium text-slate-600">امتیاز طلا 🥇</label>
          <input v-model.number="goldPoints" type="number" min="0" class="w-full rounded-lg border border-slate-300 px-3 py-2" />
        </div>
        <div>
          <label class="mb-1 block text-sm font-medium text-slate-600">امتیاز نقره 🥈</label>
          <input v-model.number="silverPoints" type="number" min="0" class="w-full rounded-lg border border-slate-300 px-3 py-2" />
        </div>
        <div>
          <label class="mb-1 block text-sm font-medium text-slate-600">امتیاز برنز 🥉</label>
          <input v-model.number="bronzePoints" type="number" min="0" class="w-full rounded-lg border border-slate-300 px-3 py-2" />
        </div>
        <div>
          <label class="mb-1 block text-sm font-medium text-slate-600">امتیاز وزن‌کشی</label>
          <input v-model.number="weighInPoint" type="number" min="0" class="w-full rounded-lg border border-slate-300 px-3 py-2" />
        </div>
        <div>
          <label class="mb-1 block text-sm font-medium text-slate-600">امتیاز هر برد</label>
          <input v-model.number="winPoint" type="number" min="0" class="w-full rounded-lg border border-slate-300 px-3 py-2" />
        </div>
      </div>

      <div class="mt-4 space-y-2">
        <label class="flex items-center gap-2">
          <input v-model="countWeighIn" type="checkbox" class="rounded" />
          <span class="text-sm">محاسبه امتیاز وزن‌کشی</span>
        </label>
        <label class="flex items-center gap-2">
          <input v-model="countWin" type="checkbox" class="rounded" />
          <span class="text-sm">محاسبه امتیاز برد</span>
        </label>
      </div>

      <button
          @click="saveScoringConfig"
          class="mt-4 rounded-lg bg-emerald-600 px-5 py-2 font-medium text-white transition hover:bg-emerald-700"
      >
        ذخیره تنظیمات امتیاز
      </button>
    </div>

    <!-- قوانین صعود -->
    <div class="rounded-2xl border border-slate-200 bg-white p-6">
      <h3 class="mb-4 text-lg font-bold">⬆️ قوانین صعود</h3>
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="mb-1 block text-sm font-medium text-slate-600">تعداد تیم‌های صعودکننده</label>
          <input v-model.number="teamsToPromote" type="number" min="1" max="10" class="w-full rounded-lg border border-slate-300 px-3 py-2" />
        </div>
        <div>
          <label class="flex items-center gap-2 pt-6">
            <input v-model="autoPromote" type="checkbox" class="rounded" />
            <span class="text-sm">صعود خودکار پس از اتمام مرحله</span>
          </label>
        </div>
      </div>

      <button
          @click="promoteTeamsNow"
          class="mt-4 rounded-lg bg-blue-600 px-5 py-2 font-medium text-white transition hover:bg-blue-700"
      >
        ⬆️ صعود تیم‌ها به مرحله بعد
      </button>
    </div>
  </div>
</template>
