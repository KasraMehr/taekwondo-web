<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useLeagueStore } from "../../stores/league";
import { loadWebLeagues } from "../../webApi";

const leagueStore = useLeagueStore();
const currentLeague = computed(() => leagueStore.currentLeague);
onMounted(async()=>leagueStore.replaceFromBackend(await loadWebLeagues()))

const playerRankings = computed(() => {
  if (!currentLeague.value) return [];
  return leagueStore.getPlayerRankings(currentLeague.value.id);
});
</script>

<template>
  <div>
    <h3 class="mb-4 text-lg font-bold">👤 رده‌بندی ورزشکاران</h3>

    <div v-if="playerRankings.length === 0" class="rounded-2xl border-2 border-dashed border-slate-300 bg-white py-16 text-center">
      <p class="text-slate-500">هنوز امتیازی ثبت نشده است</p>
    </div>

    <div v-else class="overflow-x-auto rounded-2xl border border-slate-200 bg-white">
      <table class="w-full text-sm">
        <thead class="bg-slate-50">
        <tr>
          <th class="px-4 py-3 text-right font-medium text-slate-600">رتبه</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">نام</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">باشگاه</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">وزن</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">امتیاز کل</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">🥇</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">🥈</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">🥉</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">برد</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">باخت</th>
        </tr>
        </thead>
        <tbody>
        <tr
            v-for="player in playerRankings"
            :key="player.athleteId"
            class="border-t border-slate-100 hover:bg-slate-50"
        >
          <td class="px-4 py-3 font-bold">{{ player.rank }}</td>
          <td class="px-4 py-3 font-medium">{{ player.name }}</td>
          <td class="px-4 py-3">{{ player.club }}</td>
          <td class="px-4 py-3">{{ player.weightCategory }}</td>
          <td class="px-4 py-3 font-bold text-emerald-600">{{ player.totalPoints }}</td>
          <td class="px-4 py-3">{{ player.gold }}</td>
          <td class="px-4 py-3">{{ player.silver }}</td>
          <td class="px-4 py-3">{{ player.bronze }}</td>
          <td class="px-4 py-3">{{ player.totalWins }}</td>
          <td class="px-4 py-3">{{ player.totalLosses }}</td>
        </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
