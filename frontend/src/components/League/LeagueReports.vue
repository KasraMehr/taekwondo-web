<script setup lang="ts">
import { computed } from "vue";
import { useLeagueStore } from "../../stores/league";

const leagueStore = useLeagueStore();
const currentLeague = computed(() => leagueStore.currentLeague);

const seasonStats = computed(() => {
  if (!currentLeague.value) return null;
  const league = currentLeague.value;
  const teamPoints = league.season.seasonTeamPoints;
  const playerPoints = league.season.seasonPlayerPoints;

  return {
    totalTeams: Object.keys(teamPoints).length,
    totalPlayers: Object.keys(playerPoints).length,
    totalMatches: league.season.stages.reduce((sum, stage) => {
      return sum + stage.weeks.length;
    }, 0),
    totalGold: Object.values(playerPoints).reduce((sum, p) => sum + p.gold, 0),
    totalSilver: Object.values(playerPoints).reduce((sum, p) => sum + p.silver, 0),
    totalBronze: Object.values(playerPoints).reduce((sum, p) => sum + p.bronze, 0),
  };
});
</script>

<template>
  <div>
    <h3 class="mb-4 text-lg font-bold">📊 گزارش کلی لیگ</h3>

    <div v-if="seasonStats" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <div class="rounded-2xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">تیم‌ها</p>
        <p class="mt-1 text-3xl font-bold">{{ seasonStats.totalTeams }}</p>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">ورزشکاران</p>
        <p class="mt-1 text-3xl font-bold">{{ seasonStats.totalPlayers }}</p>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">هفته‌های برگزار شده</p>
        <p class="mt-1 text-3xl font-bold">{{ seasonStats.totalMatches }}</p>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">🥇 مجموع طلا</p>
        <p class="mt-1 text-3xl font-bold text-amber-500">{{ seasonStats.totalGold }}</p>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">🥈 مجموع نقره</p>
        <p class="mt-1 text-3xl font-bold text-slate-400">{{ seasonStats.totalSilver }}</p>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">🥉 مجموع برنز</p>
        <p class="mt-1 text-3xl font-bold text-orange-700">{{ seasonStats.totalBronze }}</p>
      </div>
    </div>
  </div>
</template>
