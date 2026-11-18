<script setup lang="ts">
import {computed} from 'vue'
const props=defineProps<{players:{name:string;weightCategory?:string;totalPoints:number;gold:number;silver:number;bronze:number}[]}>()
const groups=computed(()=>{
  const grouped=new Map<string,typeof props.players>()
  for(const player of props.players){const weight=player.weightCategory||'نامشخص';if(!grouped.has(weight))grouped.set(weight,[]);grouped.get(weight)!.push(player)}
  return [...grouped].map(([weight,players])=>({weight,players:players.slice(0,4)}))
})
</script>
<template>
  <section v-if="groups.length" class="mb-5 space-y-3" dir="rtl"><h2 class="font-bold">نفرات برتر هر دستهٔ وزنی</h2><div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3"><article v-for="group in groups" :key="group.weight" class="rounded-xl border border-slate-200 bg-white p-4"><h3 class="mb-3 font-bold text-teal-700">وزن <span dir="ltr">{{group.weight}}</span></h3><ol class="space-y-2"><li v-for="(p,i) in group.players" :key="i" class="flex items-center gap-2 text-sm"><span class="rounded bg-slate-100 px-2 py-1 font-bold">{{i+1}}</span><span class="flex-1">{{p.name}}</span><b>{{p.totalPoints}} امتیاز</b><span class="text-xs" :title="`طلا ${p.gold}، نقره ${p.silver}، برنز ${p.bronze}`">{{p.gold?'🥇':p.silver?'🥈':p.bronze?'🥉':''}}</span></li></ol></article></div></section>
</template>
