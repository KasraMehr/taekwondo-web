<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { roundTotals } from '../utils/scoring'
import type { MatchRound } from '../types'
const props=defineProps<{tournamentId:string;names?:Map<string,{name:string}>}>()
type Score={version:number;blueId:string;redId:string;activeRound:number;rounds:MatchRound[]}
const scores=ref<Record<string,{score:Score;updatedAt:string}>>({}),error=ref('')
let generation=0,timer:ReturnType<typeof setTimeout>|undefined,controller:AbortController|undefined
watch(()=>props.tournamentId,async id=>{
  const current=++generation;clearTimeout(timer);controller?.abort();scores.value={}
  if(!id)return
  async function load(){
    controller=new AbortController()
    try{const response=await fetch(`/api/v1/public/competitions/${id}/live`,{signal:controller.signal,cache:'no-store'});if(!response.ok)throw new Error('ارتباط امتیاز زنده قطع است');const data=await response.json();if(current===generation){scores.value=data;error.value=''}}
    catch(e:any){if(e.name!=='AbortError'&&current===generation)error.value='ارتباط امتیاز زنده قطع است؛ امتیاز نمایش‌داده‌شده ممکن است قدیمی باشد.'}
    finally{if(current===generation)timer=setTimeout(load,1000)}
  }
  await load()
},{immediate:true})
onUnmounted(()=>{generation++;clearTimeout(timer);controller?.abort()})
const cards=computed(()=>Object.entries(scores.value).map(([id,value])=>({...value,id,round:value.score.rounds[value.score.activeRound],totals:roundTotals(value.score.rounds[value.score.activeRound])})))
</script>
<template>
  <section v-if="cards.length||error" dir="rtl" class="space-y-3">
    <h2 class="text-lg font-bold">امتیاز زندهٔ داخل راند</h2>
    <p v-if="error" role="alert" class="rounded-lg bg-rose-100 p-3 text-sm text-rose-800">{{error}}</p>
    <div class="grid gap-3 md:grid-cols-2"><article v-for="card in cards" :key="card.id" class="overflow-hidden rounded-xl border border-slate-300 bg-white text-slate-800"><header class="bg-slate-100 p-2 text-center text-sm font-bold">راند {{card.round.number}} · ثبت اپراتور</header><div class="grid grid-cols-2" dir="ltr"><div class="bg-blue-700 p-4 text-center text-white"><p>{{names?.get(card.score.blueId)?.name||'آبی'}}</p><strong class="text-4xl">{{card.totals.blue}}</strong></div><div class="bg-rose-700 p-4 text-center text-white"><p>{{names?.get(card.score.redId)?.name||'قرمز'}}</p><strong class="text-4xl">{{card.totals.red}}</strong></div></div><p class="p-2 text-center text-xs text-slate-500">امتیاز جاری است؛ نتیجهٔ نهایی هنوز ثبت نشده.</p></article></div>
  </section>
</template>
