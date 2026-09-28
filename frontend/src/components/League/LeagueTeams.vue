<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useLeagueStore } from "../../stores/league";
import { loadWebLeagues } from "../../webApi";

const leagueStore = useLeagueStore();
const currentLeague = computed(() => leagueStore.currentLeague);
onMounted(async()=>leagueStore.replaceFromBackend(await loadWebLeagues()))

const teamRankings = computed(() => {
  if (!currentLeague.value) return [];
  return leagueStore.getTeamRankings(currentLeague.value.id);
});
</script>

<template>
  <div>
    <h3 class="mb-4 text-lg font-bold">🏆 رده‌بندی تیم‌ها</h3>

    <div v-if="teamRankings.length === 0" class="rounded-2xl border-2 border-dashed border-slate-300 bg-white py-16 text-center">
      <p class="text-slate-500">هنوز امتیازی ثبت نشده است</p>
    </div>

    <div v-else class="overflow-x-auto rounded-2xl border border-slate-200 bg-white">
      <table class="w-full text-sm">
        <thead class="bg-slate-50">
        <tr>
          <th class="px-4 py-3 text-right font-medium text-slate-600">رتبه</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">باشگاه</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">امتیاز کل</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">🥇 طلا</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">🥈 نقره</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">🥉 برنز</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">تفاضل راند</th>
          <th class="px-4 py-3 text-right font-medium text-slate-600">ورزشکار</th>
        </tr>
        </thead>
        <tbody>
        <tr
            v-for="team in teamRankings"
            :key="team.club"
            class="border-t border-slate-100 hover:bg-slate-50"
        >
          <td class="px-4 py-3 font-bold">{{ team.rank }}</td>
          <td class="px-4 py-3 font-medium">{{ team.club }}</td>
          <td class="px-4 py-3 font-bold text-emerald-600">{{ team.totalPoints }}</td>
          <td class="px-4 py-3">{{ team.gold }}</td>
          <td class="px-4 py-3">{{ team.silver }}</td>
          <td class="px-4 py-3">{{ team.bronze }}</td>
          <td class="px-4 py-3">{{ team.roundDiff }}</td>
          <td class="px-4 py-3">{{ team.athleteCount }}</td>
        </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
