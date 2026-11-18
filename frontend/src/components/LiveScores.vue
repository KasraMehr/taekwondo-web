<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { roundTotals } from '../utils/scoring'
import type { MatchRound } from '../types'
type CourtMatch={id:string;court:number;status:string;matchNumber:number|null;order:number}
const props=defineProps<{tournamentId:string;names?:Map<string,{name:string}>;matches?:CourtMatch[];courts?:number}>()
const selectedCourt=ref(0)
const courtLabel=(court:number)=>String.fromCharCode(64+court)
type Score={version:number;blueId:string;redId:string;activeRound:number;rounds:MatchRound[]}
const scores=ref<Record<string,{score:Score;updatedAt:string}>>({}),error=ref('')
let generation=0,timer:ReturnType<typeof setTimeout>|undefined,controller:AbortController|undefined
watch(()=>props.tournamentId,async id=>{
  const current=++generation;clearTimeout(timer);controller?.abort();scores.value={};error.value='';selectedCourt.value=0
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
const cards=computed(()=>Object.entries(scores.value).filter(([,value])=>value.score.rounds[value.score.activeRound]).map(([id,value])=>({...value,id,round:value.score.rounds[value.score.activeRound],totals:roundTotals(value.score.rounds[value.score.activeRound])})))

const courtPanels=computed(()=>Array.from({length:Math.max(4,props.courts??4)},(_,i)=>{
 const court=i+1
 const entries=cards.value.filter(card=>props.matches?.some(match=>match.id===card.id&&match.court===court&&match.status!=='completed'))
 return {court,enabled:court<=(props.courts??4),entries}
}).filter(panel=>!selectedCourt.value||panel.court===selectedCourt.value))
const matchNumber=(id:string)=>{const match=props.matches?.find(m=>m.id===id);return match?Math.max(1,match.matchNumber||match.order):''}
</script>
<template>
  <section v-if="matches" class="court-board" dir="rtl" aria-label="نتایج زنده زمین‌ها">
<h2>نتایج لحظه‌ای <span>امتیاز جاری راند</span></h2>
<div class="court-filters" aria-label="انتخاب زمین"><button :aria-pressed="selectedCourt===0" @click="selectedCourt=0">همه زمین‌ها</button><button v-for="court in Math.max(4,courts??4)" :key="court" :aria-pressed="selectedCourt===court" @click="selectedCourt=court">زمین {{courtLabel(court)}}</button></div>
<p v-if="error" role="alert" class="live-error">{{error}}</p>
<div class="court-grid" :class="{single:selectedCourt}" dir="ltr"><article v-for="panel in courtPanels" :key="panel.court" class="court-panel" dir="rtl"><header><b>زمین {{courtLabel(panel.court)}}</b><span>{{panel.entries.length?'● زنده':'آماده'}}</span></header><div v-for="card in panel.entries" :key="card.id" class="court-game"><p class="round-label">بازی {{courtLabel(panel.court)}}{{matchNumber(card.id)}} · راند {{card.round.number}}</p><div class="score-row blue"><span>{{names?.get(card.score.blueId)?.name||'آبی'}}</span><strong>{{card.totals.blue}}</strong></div><div class="score-row red"><span>{{names?.get(card.score.redId)?.name||'قرمز'}}</span><strong>{{card.totals.red}}</strong></div></div><p v-if="!panel.entries.length" class="court-empty">{{panel.enabled?'هنوز امتیاز زنده‌ای ثبت نشده':'این زمین در مسابقه فعال نیست'}}</p></article></div>
<p class="live-note">امتیازها خودکار به‌روز می‌شوند؛ نتیجهٔ نهایی در فهرست بازی‌ها ثبت می‌شود.</p>
</section>
<section v-else-if="cards.length||error" dir="rtl" class="space-y-3">
    <h2 class="text-lg font-bold">امتیاز زندهٔ بازی‌ها</h2>
    <p v-if="error" role="alert" class="rounded-lg bg-rose-100 p-3 text-sm text-rose-800">{{error}}</p>
    <div class="grid gap-3 md:grid-cols-2"><article v-for="card in cards" :key="card.id" class="overflow-hidden rounded-2xl border border-slate-200 bg-white text-slate-800"><header class="bg-white p-3 text-center text-sm font-bold">راند {{card.round.number}}</header><div class="grid grid-cols-2" dir="ltr"><div class="bg-blue-50 p-5 text-center text-blue-800"><p>{{names?.get(card.score.blueId)?.name||'آبی'}}</p><strong class="text-4xl">{{card.totals.blue}}</strong></div><div class="bg-rose-50 p-5 text-center text-rose-800"><p>{{names?.get(card.score.redId)?.name||'قرمز'}}</p><strong class="text-4xl">{{card.totals.red}}</strong></div></div><p class="p-2 text-center text-xs text-slate-500">امتیاز جاری است؛ نتیجهٔ نهایی هنوز ثبت نشده.</p></article></div>
  </section>
</template>

<style scoped>
.court-board{min-width:0;padding:16px;background:#fff;border:1px solid #d9e7e3;border-radius:20px}
.court-board h2{font-size:20px;font-weight:800;color:#164e43;display:flex;justify-content:space-between;align-items:center;gap:8px}
.court-board h2 span{font-size:12px;color:#64748b;font-weight:400}
.court-filters{display:flex;gap:6px;flex-wrap:wrap;margin:12px 0}
.court-filters button{min-height:44px;padding:8px 12px;border:1px solid #dce6e3;border-radius:10px;font-size:13px;color:#475569}
.court-filters button[aria-pressed=true]{background:#136e60;color:white;border-color:#136e60}
.court-filters button:focus-visible{outline:3px solid #55ad98;outline-offset:2px}
.court-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px;aspect-ratio:1}
.court-grid.single{grid-template-columns:1fr;aspect-ratio:auto}
.court-panel{min-width:0;border:1px solid #e1e8ee;border-radius:12px;overflow:hidden;background:#fafcfd}
.court-panel header{display:flex;justify-content:space-between;gap:4px;padding:10px;font-size:14px;border-bottom:1px solid #e4ece9;background:#f0f8f5}
.court-panel header span{font-size:11px;color:#277861}
.round-label{text-align:center;font-size:12px;color:#64748b;padding:6px}
.score-row{display:flex;align-items:center;justify-content:space-between;gap:6px;padding:8px 10px;margin:4px 6px;border-radius:8px}
.score-row span{font-size:14px;overflow-wrap:anywhere}
.score-row strong{font-size:28px;line-height:1.2}
.blue{background:#edf4ff;color:#244e9e}.red{background:#fff0f2;color:#ab304b}
.court-empty{font-size:13px;color:#64748b;padding:24px 12px;text-align:center}
.live-note{font-size:12px;color:#64748b;margin-top:12px}
.live-error{background:#fff1f2;color:#a12543;padding:10px;margin-bottom:10px;border-radius:10px;font-size:13px}
@media(min-width:700px){.court-grid{aspect-ratio:auto;min-height:380px}}
@media(max-width:400px){.court-board{padding:10px}.court-grid{gap:6px}.court-filters button{padding:6px 8px;font-size:12px}.score-row{padding:7px 6px;margin:4px}.score-row span{font-size:12px}.score-row strong{font-size:24px}.court-panel header{padding:8px;font-size:13px}.court-empty{padding:20px 8px}}
@media print{.court-board{display:none}}
</style>
