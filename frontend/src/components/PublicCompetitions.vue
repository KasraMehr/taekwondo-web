<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import LiveScores from './LiveScores.vue'
import WeightLeaders from './WeightLeaders.vue'

type Athlete={id:string;name:string;club:string;weightCategory:string}
type Match={id:string;athlete1Id:string|null;athlete2Id:string|null;winnerId:string|null;court:number;order:number;matchNumber:number|null;status:string;round:number;side:string;weightCategory:string;nextMatchId:string|null;isBye:boolean;result:null|{winType:string;rounds:{winner?:string}[]}}
type Standing={rank:number;name:string;clubName?:string;coachName?:string;weightCategory?:string;groupName?:string;totalPoints:number;gold:number;silver:number;bronze:number;wins:number;losses:number}
type Competition={id:string;name:string;date:string;gender:string;ageCategory:string;leagueName:string;organizer?:string;courts:number}
type Detail=Competition&{athletes:Athlete[];matches:Match[];standings:{players:Standing[];teams:Standing[]};rankingScope:string;scoring:{gold:number;silver:number;bronze:number;winPoint:number;weighInPoint:number;countWin:boolean;countWeighIn:boolean}}
const id=location.pathname.split('/').filter(Boolean)[1]||''
const tabs=[{id:'bracket',label:'جدول و شماره‌ها'},{id:'results',label:'نتایج لحظه‌ای'},{id:'players',label:'رنکینگ بازیکنان'},{id:'teams',label:'رنکینگ تیم‌ها'}]
const initial=new URLSearchParams(location.search).get('tab')||'bracket'
const tab=ref(tabs.some(t=>t.id===initial)?initial:'bracket')
const items=ref<Competition[]>([]), detail=ref<Detail|null>(null), loading=ref(true), error=ref(''), query=ref(''), hasMore=ref(false), weight=ref(''), updated=ref('')
let timer:number|undefined, controller:AbortController|undefined, stopped=false, sequence=0
async function load(append=false){
  const current=++sequence
  controller?.abort();controller=new AbortController()
  loading.value=true
  try{
    const url=id?`/api/v1/public/competitions/${encodeURIComponent(id)}`:`/api/v1/public/competitions?q=${encodeURIComponent(query.value)}&offset=${append?items.value.length:0}`
    const response=await fetch(url,{signal:controller.signal,cache:'no-store'})
    if(!response.ok)throw new Error(response.status===404?'مسابقه پیدا نشد.':'دریافت اطلاعات ناموفق بود؛ دوباره تلاش کنید.')
    const data=await response.json()
    if(stopped||current!==sequence)return
    if(id)detail.value=data;else{items.value=append?[...items.value,...data.items]:data.items;hasMore.value=data.hasMore}
    updated.value=new Date().toLocaleTimeString('fa-IR');error.value=''
  }catch(e:any){if(e.name!=='AbortError'&&current===sequence)error.value=e.message}
  finally{if(current===sequence)loading.value=false}
}
async function poll(){await load();if(!stopped)timer=window.setTimeout(poll,5000)}
function selectTab(value:string){tab.value=value;history.replaceState({},'',`${location.pathname}?tab=${value}`)}
const onPop=()=>{const value=new URLSearchParams(location.search).get('tab');tab.value=tabs.some(t=>t.id===value)?value!:'bracket'}
onMounted(()=>{if(id)void poll();else void load();window.addEventListener('popstate',onPop)})
onUnmounted(()=>{stopped=true;controller?.abort();clearTimeout(timer);window.removeEventListener('popstate',onPop)})
const names=computed(()=>new Map(detail.value?.athletes.map(a=>[a.id,a])??[]))
const weights=computed(()=>[...new Set(detail.value?.matches.map(m=>m.weightCategory)??[])])
const matches=computed(()=>(detail.value?.matches??[]).filter(m=>!weight.value||m.weightCategory===weight.value))
const rounds=computed(()=>[...new Set(matches.value.map(m=>m.round))].sort((a,b)=>a-b))
const results=computed(()=>matches.value.filter(m=>!m.isBye).sort((a,b)=>({ongoing:0,completed:1,pending:2}[a.status]??2)-({ongoing:0,completed:1,pending:2}[b.status]??2)||(a.matchNumber??a.order)-(b.matchNumber??b.order)))
const ranking=computed(()=>detail.value?.standings[tab.value==='teams'?'teams':'players']??[])
const number=(m:Match)=>m.matchNumber??m.order
const status=(m:Match)=>m.isBye?'استراحت':m.status==='completed'?'پایان‌یافته':m.status==='ongoing'?'در حال برگزاری':'در انتظار'
function participant(m:Match,slot:1|2){const athlete=names.value.get((slot===1?m.athlete1Id:m.athlete2Id)||'');if(athlete)return athlete.name;return m.isBye?'استراحت':'در انتظار صعود'}
function nextLabel(m:Match){const next=detail.value?.matches.find(n=>n.id===m.nextMatchId);return next?`صعود به مبارزه ${number(next)}`:m.side==='final'?'فینال':''}
function score(m:Match){if(!m.result)return '—';const rounds=m.result.rounds??[];const b=rounds.filter(r=>r.winner==='blue').length,r=rounds.filter(r=>r.winner==='red').length;return `${b} – ${r} · ${m.result.winType}`}
const dateLabel=(value:string)=>new Date(value).toLocaleDateString('fa-IR')
</script>

<template>
  <div dir="rtl" class="min-h-screen bg-slate-50 text-slate-800">
    <header class="border-b border-slate-800 bg-slate-950 text-white">
      <div class="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-4 px-5 py-5">
        <a href="/competitions" class="text-lg font-black">🥋 مسابقات تکواندو</a>
        <nav class="flex gap-4 text-sm"><a href="/competitions" class="text-slate-200">همه مسابقات</a><a href="/" class="text-slate-300">ورود برگزارکنندگان</a></nav>
      </div>
    </header>
    <main class="mx-auto max-w-7xl space-y-6 px-4 py-7 sm:px-6">
      <template v-if="!id">
        <section class="rounded-3xl bg-slate-900 p-7 text-white sm:p-10"><p class="text-sm text-teal-300">تماشای عمومی مسابقات</p><h1 class="mt-3 text-3xl font-black">از قرعه‌کشی تا سکوی قهرمانی</h1><p class="mt-4 leading-8 text-slate-300">جدول مبارزه‌ها، برنامه زمین‌ها، نتایج و رتبه‌بندی بازیکنان و تیم‌ها را بدون ورود به حساب دنبال کنید.</p></section>
        <form class="flex gap-2" @submit.prevent="load()"><input v-model="query" aria-label="نام مسابقه یا لیگ" placeholder="جستجوی مسابقه یا لیگ…" class="min-w-0 flex-1 rounded-xl border border-slate-300 bg-white px-4 py-3"><button class="rounded-xl bg-teal-700 px-6 text-white">جستجو</button></form>
        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"><a v-for="item in items" :key="item.id" :href="`/competitions/${item.id}`" class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm transition hover:border-teal-500 hover:shadow-md"><p class="text-xs text-teal-700">{{item.leagueName||'تورنومنت'}}</p><h2 class="mt-2 text-xl font-bold">{{item.name}}</h2><p class="mt-3 text-sm text-slate-500">{{dateLabel(item.date)}} · {{item.ageCategory}} · {{item.gender==='male'?'مردان':'زنان'}}</p><p class="mt-2 text-sm text-slate-500">{{item.organizer}}</p><span class="mt-5 inline-block text-sm font-bold text-teal-700">مشاهده جدول و نتایج ←</span></a></div>
        <p v-if="!loading&&!error&&!items.length" class="rounded-xl border border-dashed p-10 text-center text-slate-500">مسابقه‌ای پیدا نشد.</p>
        <button v-if="hasMore" :disabled="loading" class="rounded-xl border bg-white px-6 py-3" @click="load(true)">نمایش مسابقات بیشتر</button>
      </template>
      <template v-else-if="detail">
        <LiveScores :tournament-id="detail.id" :names="names" />
        <section class="rounded-2xl border border-slate-200 bg-white p-6"><p class="text-sm text-teal-700">{{detail.leagueName||'تورنومنت'}}</p><h1 class="mt-2 text-2xl font-black">{{detail.name}}</h1><p class="mt-3 text-sm text-slate-500">{{dateLabel(detail.date)}} · {{detail.ageCategory}} · {{detail.gender==='male'?'مردان':'زنان'}} · {{detail.courts}} زمین</p><div class="mt-5 flex flex-wrap items-center gap-3"><a :href="`/ovr?tournamentId=${detail.id}`" target="_blank" rel="noopener" class="rounded-xl bg-slate-900 px-5 py-2.5 text-sm text-white">نمایش OVR زمین‌ها ↗</a><span class="text-xs" :class="error?'text-rose-600':'text-teal-700'">{{error?'ارتباط قطع شده؛ اطلاعات ممکن است قدیمی باشد':`به‌روزرسانی خودکار هر ۵ ثانیه · ${updated}`}}</span></div></section>
        <nav class="flex gap-2 overflow-x-auto pb-1" aria-label="بخش‌های مسابقه"><button v-for="item in tabs" :key="item.id" :aria-pressed="tab===item.id" class="whitespace-nowrap rounded-xl px-5 py-3 text-sm font-bold" :class="tab===item.id?'bg-teal-700 text-white':'border border-slate-200 bg-white text-slate-600'" @click="selectTab(item.id)">{{item.label}}</button></nav>
        <label v-if="tab==='bracket'||tab==='results'" class="flex items-center gap-3 text-sm">وزن<select v-model="weight" class="rounded-lg border bg-white px-4 py-2"><option value="">همه وزن‌ها</option><option v-for="w in weights" :key="w" :value="w">{{w}}</option></select></label>
        <template v-if="tab==='bracket'">
          <p class="text-xs text-slate-500">شمارهٔ هر مبارزه و شمارهٔ مبارزهٔ بعدی روی کارت درج شده است.</p>
          <div v-if="matches.length" class="flex gap-5 overflow-x-auto pb-4">
            <section v-for="round in rounds" :key="round" class="w-72 shrink-0 space-y-4"><h2 class="font-bold">دور {{round}}</h2><article v-for="m in matches.filter(m=>m.round===round).sort((a,b)=>number(a)-number(b))" :key="m.id" class="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm"><header class="flex items-center justify-between bg-slate-100 px-4 py-2 text-xs"><strong>مبارزه {{number(m)}}</strong><span>زمین {{m.court}} · {{m.weightCategory}}</span></header><div class="border-r-4 border-blue-600 p-3 text-sm" :class="m.winnerId===m.athlete1Id&&m.winnerId?'bg-emerald-50 font-bold':''">{{participant(m,1)}} <small class="block text-slate-500">{{names.get(m.athlete1Id||'')?.club}}</small> <span v-if="m.winnerId&&m.winnerId===m.athlete1Id">✓</span></div><div class="border-r-4 border-rose-600 p-3 text-sm" :class="m.winnerId===m.athlete2Id&&m.winnerId?'bg-emerald-50 font-bold':''">{{participant(m,2)}} <small class="block text-slate-500">{{names.get(m.athlete2Id||'')?.club}}</small> <span v-if="m.winnerId&&m.winnerId===m.athlete2Id">✓</span></div><footer class="space-y-1 border-t px-4 py-2 text-xs text-slate-500"><p>{{status(m)}} <span v-if="m.result" dir="ltr">{{score(m)}}</span></p><p>{{nextLabel(m)}}</p></footer></article></section>
          </div>
          <p v-else class="rounded-xl border border-dashed p-8 text-center">جدول این مسابقه هنوز آماده نشده است.</p>
        </template>
        <div v-else-if="tab==='results'" class="overflow-x-auto rounded-xl border bg-white"><table class="w-full whitespace-nowrap text-right text-sm"><thead class="bg-slate-100"><tr><th class="p-4">شماره / زمین</th><th class="p-4">آبی</th><th class="p-4">قرمز</th><th class="p-4">وزن</th><th class="p-4">نتیجه راندها (آبی–قرمز)</th><th class="p-4">برنده / وضعیت</th></tr></thead><tbody><tr v-for="m in results" :key="m.id" class="border-t"><td class="p-4">{{number(m)}} / {{m.court}}</td><td class="p-4 text-blue-700">{{participant(m,1)}}</td><td class="p-4 text-rose-700">{{participant(m,2)}}</td><td class="p-4" dir="ltr">{{m.weightCategory}}</td><td class="p-4" dir="ltr">{{score(m)}}</td><td class="p-4"><b v-if="m.winnerId">{{names.get(m.winnerId)?.name}}</b><div class="text-xs text-slate-500">{{status(m)}}</div></td></tr><tr v-if="!results.length"><td colspan="6" class="p-8 text-center text-slate-500">مبارزه‌ای برای نمایش وجود ندارد.</td></tr></tbody></table></div>
        <template v-else>
          <WeightLeaders v-if="tab==='players'" :players="detail.standings.players" />
          <p class="text-sm leading-7 text-slate-500">{{detail.rankingScope==='league'?`رتبه‌بندی کل لیگ ${detail.leagueName}`:'رتبه‌بندی همین تورنومنت بر اساس امتیازهای پیش‌فرض'}} · طلا {{detail.scoring.gold}}، نقره {{detail.scoring.silver}}، برنز {{detail.scoring.bronze}}، هر برد {{detail.scoring.countWin?detail.scoring.winPoint:0}}، وزن‌کشی {{detail.scoring.countWeighIn?detail.scoring.weighInPoint:0}} امتیاز. مدال‌ها پس از تکمیل جدول هر وزن ثبت می‌شوند؛ دو بازندهٔ نیمه‌نهایی برنز می‌گیرند.</p>
          <div class="overflow-x-auto rounded-xl border bg-white"><table class="w-full whitespace-nowrap text-right text-sm"><thead class="bg-slate-100"><tr><th class="p-4">رتبه</th><th class="p-4">{{tab==='teams'?'تیم / باشگاه':'بازیکن'}}</th><th v-if="tab==='players'" class="p-4">تیم یا مربی / وزن</th><th class="p-4">گروه</th><th class="p-4">امتیاز</th><th class="p-4">طلا / نقره / برنز</th><th class="p-4">برد / باخت</th></tr></thead><tbody><tr v-for="(p,i) in ranking" :key="i" class="border-t"><td class="p-4 font-bold">{{p.rank}}</td><td class="p-4 font-bold">{{p.name||'آزاد'}}</td><td v-if="tab==='players'" class="p-4">{{p.clubName||p.coachName||'آزاد'}} / {{p.weightCategory}}</td><td class="p-4">{{p.groupName||'—'}}</td><td class="p-4 font-black text-teal-700">{{p.totalPoints}}</td><td class="p-4">{{p.gold}} / {{p.silver}} / {{p.bronze}}</td><td class="p-4">{{p.wins}} / {{p.losses}}</td></tr><tr v-if="!ranking.length"><td :colspan="tab==='players'?7:6" class="p-8 text-center text-slate-500">هنوز امتیازی ثبت نشده است.</td></tr></tbody></table></div>
        </template>
      </template>
      <p v-if="loading&&!detail" role="status" class="text-center text-slate-500">در حال دریافت اطلاعات…</p>
      <div v-if="error" role="alert" class="rounded-xl bg-rose-50 p-4 text-sm text-rose-700">{{error}} <button class="mr-4 underline" @click="load()">تلاش دوباره</button></div>
    </main>
  </div>
</template>
