<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import LiveScores from './LiveScores.vue'
import WeightLeaders from './WeightLeaders.vue'
import BracketPdfExport from './BracketPdfExport.vue'
import type { Match as BracketMatch, Tournament } from '../types'
import { competitionPeriod, tehranDate } from '../utils/publicCompetition'
import { normalizeText } from '../utils/athleteImport'

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
const today=ref(tehranDate()), personQuery=ref('')
const submittedQuery=ref('')
let dateTimer:ReturnType<typeof setInterval>|undefined
const period=(date:string)=>competitionPeriod(date,today.value)
const todayItems=computed(()=>items.value.filter(item=>period(item.date)==='today'))
const otherItems=computed(()=>items.value.filter(item=>period(item.date)!=='today'))
const listSections=computed(()=>[
  {key:'today',title:'مسابقات امروز',hint:'جدول و نتایج مسابقه‌های امروز را از اینجا دنبال کنید.',items:todayItems.value},
  {key:'other',title:'دیگر مسابقات',hint:'مسابقات پیش رو و نتایج روزهای گذشته',items:otherItems.value},
].filter(section=>section.items.length))
const periodLabel=(date:string)=>({today:'امروز',upcoming:'پیش رو',past:'روزهای گذشته'}[period(date)])
const competitionLink=(value:string,section='bracket')=>`/competitions/${encodeURIComponent(value)}?tab=${section}`
const contains=(values:(string|undefined)[])=>normalizeText(values.filter(Boolean).join(' ')).includes(normalizeText(personQuery.value))
let timer:number|undefined, controller:AbortController|undefined, stopped=false, sequence=0
async function load(append=false){
  if(!append)submittedQuery.value=query.value.trim()
  const current=++sequence
  controller?.abort();controller=new AbortController()
  loading.value=true
  try{
    const url=id?`/api/v1/public/competitions/${encodeURIComponent(id)}`:`/api/v1/public/competitions?q=${encodeURIComponent(submittedQuery.value)}&offset=${append?items.value.length:0}`
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
onMounted(()=>{if(id)void poll();else void load();window.addEventListener('popstate',onPop);dateTimer=setInterval(()=>{const day=tehranDate();if(day!==today.value){today.value=day;void load()}},60000)})
onUnmounted(()=>{stopped=true;controller?.abort();clearTimeout(timer);clearInterval(dateTimer);window.removeEventListener('popstate',onPop)})
const names=computed(()=>new Map(detail.value?.athletes.map(a=>[a.id,a])??[]))
const weights=computed(()=>[...new Set(detail.value?.matches.map(m=>m.weightCategory)??[])])
const matches=computed(()=>(detail.value?.matches??[]).filter(m=>!weight.value||m.weightCategory===weight.value))
const publicTournament=computed(()=>detail.value?{...detail.value,matches:detail.value.matches.map(m=>({ ...m, matchNumber:m.matchNumber&&m.matchNumber>0?m.matchNumber:Math.max(1,m.order), order:Math.max(1,m.order) })) as unknown as BracketMatch[],leagueId:detail.value.rankingScope==='league'?'league':null} as unknown as Tournament:undefined)
const results=computed(()=>matches.value.filter(m=>!m.isBye&&contains([names.value.get(m.athlete1Id||'')?.name,names.value.get(m.athlete2Id||'')?.name,names.value.get(m.athlete1Id||'')?.club,names.value.get(m.athlete2Id||'')?.club])).sort((a,b)=>({ongoing:0,completed:1,pending:2}[a.status]??2)-({ongoing:0,completed:1,pending:2}[b.status]??2)||(a.matchNumber??a.order)-(b.matchNumber??b.order)))
const ranking=computed(()=>(detail.value?.standings[tab.value==='teams'?'teams':'players']??[]).filter(p=>(tab.value==='teams'||!weight.value||p.weightCategory===weight.value)&&contains([p.name,p.clubName,p.coachName])))
const completed=computed(()=>detail.value?.matches.filter(m=>!m.isBye&&m.status==='completed').length??0)
const number=(m:Match)=>m.matchNumber&&m.matchNumber>0?m.matchNumber:Math.max(1,m.order)
const status=(m:Match)=>m.isBye?'استراحت':m.status==='completed'?'پایان‌یافته':m.status==='ongoing'?'در حال برگزاری':'در انتظار'
function participant(m:Match,slot:1|2){const athlete=names.value.get((slot===1?m.athlete1Id:m.athlete2Id)||'');if(athlete)return athlete.name;return m.isBye?'استراحت':'در انتظار صعود'}
function score(m:Match){if(!m.result)return '—';const rounds=m.result.rounds??[];const b=rounds.filter(r=>r.winner==='blue').length,r=rounds.filter(r=>r.winner==='red').length;return `${b} – ${r} · ${m.result.winType}`}
const dateLabel=(value:string)=>new Date(value.slice(0,10)+'T12:00:00+03:30').toLocaleDateString('fa-IR',{timeZone:'Asia/Tehran',day:'numeric',month:'long',year:'numeric'})
</script>

<template>
  <div dir="rtl" class="public-hub">
    <header class="site-header">
      <div class="page-width header-inner">
        <a href="/competitions" class="brand"><span class="brand-mark" aria-hidden="true">TKD</span><span>مسابقات تکواندو<small>همراه شما، کنار زمین</small></span></a>
        <a v-if="id" href="/competitions" class="back-link">همه مسابقات ←</a>
        <span v-else class="header-note">جدول‌ها · نتایج · رتبه‌بندی</span>
      </div>
    </header>
    <main class="page-width main-content">
      <template v-if="!id">
        <section class="welcome">
          <div><span class="eyebrow">برای بازیکن‌ها، مربی‌ها و خانواده‌ها</span><h1>مسابقه را از نزدیک<br><span>دنبال کنید.</span></h1><p>مسابقه‌تان را پیدا کنید؛ جدول بازی‌ها، امتیازهای زنده و جایگاه تیم و بازیکن‌ها همین‌جاست.</p></div>
          <div class="welcome-aside"><span class="calendar-mark" aria-hidden="true">امروز</span><strong>{{dateLabel(today)}}</strong><span>هر امتیاز، یک قدم تا قهرمانی</span><a v-if="todayItems.length" href="#today-competitions" class="text-link">مشاهده مسابقات امروز ↓</a></div>
        </section>
        <form class="search-panel" role="search" @submit.prevent="load()"><label for="competition-search">دنبال کدام مسابقه هستید؟</label><div class="search-row"><input id="competition-search" v-model="query" type="search" placeholder="نام مسابقه یا لیگ را بنویسید…"><button class="primary-button" :disabled="loading">جستجو</button></div></form>
        <section v-for="section in listSections" :id="section.key==='today'?'today-competitions':undefined" :key="section.key" class="competition-section">
          <div class="section-heading"><div><h2><span v-if="section.key==='today'" class="today-dot" aria-hidden="true"></span>{{section.title}}</h2><p>{{section.hint}}</p></div><span class="count-badge">{{section.items.length.toLocaleString('fa-IR')}} مسابقه</span></div>
          <div class="competition-grid"><article v-for="item in section.items" :key="item.id" class="competition-card" :class="{'today-card':section.key==='today'}">
            <div class="card-top"><span class="event-kind">{{item.leagueName||'تورنومنت'}}</span><span class="date-badge" :class="{'today-badge':section.key==='today'}">{{periodLabel(item.date)}}</span></div>
            <h3><a :href="competitionLink(item.id)">{{item.name}}</a></h3>
            <p class="event-date">{{dateLabel(item.date)}}</p>
            <div class="event-tags"><span>{{item.ageCategory}}</span><span>{{item.gender==='male'?'پسران / آقایان':'دختران / بانوان'}}</span><span>{{item.courts.toLocaleString('fa-IR')}} زمین</span></div>
            <p v-if="item.organizer" class="organizer">برگزارکننده: {{item.organizer}}</p>
            <div class="card-actions"><a :href="competitionLink(item.id)" class="primary-button">جدول مسابقه ←</a><a :href="competitionLink(item.id,'results')" class="secondary-button">نتایج بازی‌ها</a></div>
          </article></div>
        </section>
        <div v-if="!loading&&!error&&!items.length" class="empty-state"><strong>مسابقه‌ای پیدا نشد</strong><p>نام کوتاه‌تر یا نام لیگ را جستجو کنید.</p><button v-if="query" class="secondary-button" @click="query='';load()">نمایش همه مسابقات</button></div>
        <div v-if="hasMore" class="load-more"><button :disabled="loading" class="secondary-button" @click="load(true)">نمایش مسابقات بیشتر ↓</button></div>
      </template>
      <template v-else-if="detail">
        <section class="event-heading" :class="{'event-today':period(detail.date)==='today'}">
          <div class="card-top"><span class="eyebrow">{{detail.leagueName||'تورنومنت تکواندو'}}</span><span class="date-badge" :class="{'today-badge':period(detail.date)==='today'}">{{periodLabel(detail.date)}}</span></div>
          <h1>{{detail.name}}</h1><p>{{dateLabel(detail.date)}} · {{detail.ageCategory}} · {{detail.gender==='male'?'پسران / آقایان':'دختران / بانوان'}}</p>
          <div class="event-summary"><div class="event-stats"><span><b>{{detail.athletes.length.toLocaleString('fa-IR')}}</b> بازیکن</span><span><b>{{detail.courts.toLocaleString('fa-IR')}}</b> زمین</span><span><b>{{completed.toLocaleString('fa-IR')}}</b> بازی پایان‌یافته</span></div><a :href="'/ovr?tournamentId='+encodeURIComponent(detail.id)" target="_blank" rel="noopener" class="secondary-button">نمایشگر زمین‌ها (OVR) ↗</a></div>
          <p class="update-status" :class="{'connection-error':error}"><span class="status-dot" aria-hidden="true"></span>{{error?'ارتباط قطع شده؛ اطلاعات ممکن است قدیمی باشد':'به‌روزرسانی خودکار · آخرین دریافت '+updated}}</p>
        </section>
        <LiveScores :tournament-id="detail.id" :names="names" />
        <nav class="competition-tabs" aria-label="بخش‌های مسابقه"><button v-for="item in tabs" :key="item.id" :aria-pressed="tab===item.id" :class="{'selected':tab===item.id}" @click="selectTab(item.id)">{{item.label}}<span>{{item.id==='bracket'?'مسیر رسیدن به فینال':item.id==='results'?'بازی‌ها و برنده‌ها':item.id==='players'?'امتیاز و مدال بازیکن‌ها':'جایگاه تیم‌ها و باشگاه‌ها'}}</span></button></nav>
        <div v-if="tab!=='bracket'" class="filter-panel"><label class="person-search" for="person-search">{{tab==='teams'?'تیم خود را پیدا کنید':'بازیکن خود را پیدا کنید'}}<input id="person-search" v-model="personQuery" type="search" :placeholder="tab==='teams'?'نام تیم یا باشگاه…':'نام بازیکن، تیم یا مربی…'"></label><label v-if="tab!=='teams'" for="weight-filter">دستهٔ وزنی<select id="weight-filter" v-model="weight"><option value="">همه وزن‌ها</option><option v-for="w in weights" :key="w" :value="w">{{w}}</option></select></label><button v-if="personQuery||weight" class="text-link" @click="personQuery='';weight=''">پاک کردن فیلترها</button></div>
        <template v-if="tab==='bracket'"><BracketPdfExport :public-tournament="publicTournament" /></template>
        <section v-else-if="tab==='results'" class="results-section">
          <div class="section-heading"><div><h2>نتایج بازی‌ها</h2><p>بازی‌های در حال برگزاری در ابتدای فهرست قرار دارند.</p></div><span class="count-badge">{{results.length.toLocaleString('fa-IR')}} بازی</span></div>
          <div class="results-grid"><article v-for="m in results" :key="m.id" class="result-card" :class="{'ongoing-card':m.status==='ongoing'}"><header><strong>بازی {{number(m).toLocaleString('fa-IR')}}</strong><span class="match-status" :class="m.status">{{status(m)}}</span></header><p class="match-meta">زمین {{m.court>0?m.court.toLocaleString('fa-IR'):'نامشخص'}} · وزن <b dir="ltr">{{m.weightCategory}}</b></p><div class="athlete-row blue"><span class="corner-dot" aria-hidden="true"></span><div><strong>{{participant(m,1)}}</strong><small>{{names.get(m.athlete1Id||'')?.club}}</small></div><span v-if="m.winnerId&&m.winnerId===m.athlete1Id" class="winner-label">✓ برنده</span></div><div class="athlete-row red"><span class="corner-dot" aria-hidden="true"></span><div><strong>{{participant(m,2)}}</strong><small>{{names.get(m.athlete2Id||'')?.club}}</small></div><span v-if="m.winnerId&&m.winnerId===m.athlete2Id" class="winner-label">✓ برنده</span></div><footer><span>نتیجهٔ راندها <small>(آبی – قرمز)</small></span><strong dir="ltr">{{score(m)}}</strong></footer></article></div>
          <div v-if="!results.length" class="empty-state"><strong>{{personQuery||weight?'بازی‌ای با این مشخصات پیدا نشد':'هنوز بازی‌ای برای نمایش وجود ندارد'}}</strong><p>{{personQuery||weight?'نام یا دستهٔ وزنی را تغییر دهید.':'پس از آماده‌شدن برنامه، بازی‌ها اینجا نمایش داده می‌شوند.'}}</p></div>
        </section>
        <template v-else>
          <WeightLeaders v-if="tab==='players'&&!personQuery&&!weight" :players="detail.standings.players" />
          <section class="ranking-section"><div class="section-heading"><div><h2>{{tab==='teams'?'رتبه‌بندی تیم‌ها':'رتبه‌بندی بازیکن‌ها'}}</h2><p>{{detail.rankingScope==='league'?'امتیازهای کل لیگ '+detail.leagueName:'امتیازهای همین تورنومنت'}}</p></div></div>
            <div class="table-scroll" tabindex="0" role="region" aria-label="جدول رتبه‌بندی؛ برای ستون‌های بیشتر به طرفین حرکت کنید"><table><thead><tr><th>رتبه</th><th>{{tab==='teams'?'تیم / باشگاه':'بازیکن'}}</th><th v-if="tab==='players'">تیم یا مربی / وزن</th><th>گروه</th><th>امتیاز</th><th>طلا / نقره / برنز</th><th>برد / باخت</th></tr></thead><tbody><tr v-for="(p,i) in ranking" :key="i"><td><span class="rank-number" :class="{'top-rank':p.rank<=3}">{{p.rank.toLocaleString('fa-IR')}}</span></td><td class="ranking-name">{{p.name||'آزاد'}}</td><td v-if="tab==='players'">{{p.clubName||p.coachName||'آزاد'}} <small class="block" dir="ltr">{{p.weightCategory}}</small></td><td>{{p.groupName||'—'}}</td><td class="points">{{p.totalPoints.toLocaleString('fa-IR')}}</td><td>{{p.gold}} / {{p.silver}} / {{p.bronze}}</td><td>{{p.wins}} / {{p.losses}}</td></tr><tr v-if="!ranking.length"><td :colspan="tab==='players'?7:6" class="empty-state">{{personQuery||weight?'نتیجه‌ای با این مشخصات پیدا نشد.':'هنوز امتیازی ثبت نشده است.'}}</td></tr></tbody></table></div>
            <details class="scoring-help"><summary>امتیازها چطور محاسبه می‌شوند؟</summary><p>طلا {{detail.scoring.gold}}، نقره {{detail.scoring.silver}}، برنز {{detail.scoring.bronze}}، هر برد {{detail.scoring.countWin?detail.scoring.winPoint:0}} و وزن‌کشی {{detail.scoring.countWeighIn?detail.scoring.weighInPoint:0}} امتیاز. مدال‌ها پس از تکمیل جدول هر وزن ثبت می‌شوند؛ دو بازندهٔ نیمه‌نهایی برنز می‌گیرند.</p></details>
          </section>
        </template>
      </template>
      <p v-if="loading&&!detail" role="status" class="loading-message">در حال دریافت مسابقات…</p>
      <div v-if="error" role="alert" class="error-message">{{error}} <button class="text-link" @click="load()">تلاش دوباره</button></div>
    </main>
    <footer class="site-footer page-width"><span>مسابقات تکواندو</span><span>کنار بازیکن‌ها، مربی‌ها و خانواده‌ها</span></footer>
  </div>
</template>

<style scoped>
#today-competitions .competition-grid{grid-template-columns:repeat(auto-fit,minmax(min(100%,360px),1fr))}
#today-competitions .card-actions{max-width:440px;width:100%}
.public-hub{min-height:100vh;background:#fafbfc;color:#1e293b;font-family:Vazirmatn,sans-serif;font-size:14px;line-height:1.7}
.page-width{width:min(1240px,100%);margin:auto;padding-inline:24px}
.site-header{background:#fff;border-bottom:1px solid #e8edf0}
.header-inner{display:flex;align-items:center;justify-content:space-between;gap:16px;min-height:92px}
.brand{display:flex;align-items:center;gap:12px;font-size:17px;font-weight:900;color:#173c39}
.brand small{display:block;font-size:11px;font-weight:400;color:#64748b;margin-top:2px}
.brand-mark{display:grid;place-items:center;width:46px;height:46px;border-radius:15px;background:#e9f5f1;color:#0f766e;font-size:14px;letter-spacing:1px}
.header-note,.back-link{font-size:12px;color:#64748b}
.back-link{padding:10px;border:1px solid #e2e8f0;border-radius:12px}
.main-content{padding-block:36px 70px;display:flex;flex-direction:column;gap:28px}
.welcome{display:flex;justify-content:space-between;gap:40px;padding:24px 4px 18px}
.eyebrow{font-size:12px;font-weight:700;color:#0f766e}
.welcome h1{font-size:clamp(30px,4vw,46px);font-weight:900;line-height:1.65;margin:12px 0}
.welcome h1 span{color:#0f766e}
.welcome p{max-width:570px;color:#64748b;line-height:2.1;font-size:15px}
.welcome-aside{min-width:250px;display:flex;flex-direction:column;justify-content:center;align-items:center;gap:14px;border-right:1px solid #e2e8f0;padding:16px 40px;color:#64748b;font-size:12px}
.welcome-aside strong{font-size:19px;color:#334155}
.calendar-mark{background:#fff;border:1px solid #dbe8e3;border-top:4px solid #0f766e;border-radius:15px;padding:12px 26px;color:#0f766e;font-weight:900;font-size:18px;box-shadow:0 8px 24px #163d3408}
.search-panel{border:1px solid #e2e8f0;background:#fff;border-radius:20px;padding:20px}
.search-panel label,.filter-panel label{display:block;font-size:12px;font-weight:700;color:#475569}
.search-row{display:flex;gap:10px;margin-top:10px}
input,select{border:1px solid #dce4e9;border-radius:12px;background:#fff;color:#1e293b;min-height:46px;padding:10px 14px;font:inherit;min-width:0}
input::placeholder{color:#8492a2}
.search-row input{flex:1}
.primary-button,.secondary-button{display:inline-flex;justify-content:center;align-items:center;gap:8px;min-height:44px;padding:10px 18px;border-radius:12px;font-size:13px;font-weight:700;text-align:center;transition:background .15s,border-color .15s}
.primary-button{background:#0f766e;color:#fff;border:1px solid #0f766e}
.primary-button:hover{background:#115e59}
.secondary-button{background:#fff;border:1px solid #dce5e8;color:#475569}
.secondary-button:hover{background:#f0f7f5;border-color:#8abcb1}
button{cursor:pointer}
button:disabled{opacity:.55;cursor:wait}
.text-link{color:#0f766e;font-size:12px;font-weight:700;padding:6px}
.section-heading{display:flex;align-items:center;justify-content:space-between;gap:14px;margin-bottom:18px}
.section-heading h2{font-size:20px;font-weight:800;display:flex;align-items:center;gap:10px}
.section-heading p{color:#64748b;font-size:12px;margin-top:5px}
.count-badge{padding:4px 11px;border:1px solid #e2e8f0;border-radius:20px;color:#64748b;font-size:11px;white-space:nowrap;background:#fff}
.today-dot,.status-dot{display:inline-block;width:8px;height:8px;background:#15936f;border-radius:50%}
.today-dot{box-shadow:0 0 0 5px #e3f4ec}
.competition-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:18px}
.competition-card{display:flex;flex-direction:column;background:#fff;border:1px solid #e2e8f0;border-radius:20px;padding:24px;box-shadow:0 3px 12px #182c3903;transition:box-shadow .2s,border-color .2s}
.competition-card:hover{border-color:#a8cfc4;box-shadow:0 8px 24px #182c3909}
.today-card{border:2px solid #69b39e;background:linear-gradient(160deg,#eef9f3 0%,#fff 60%);box-shadow:0 6px 22px #16624b09}
.card-top{display:flex;align-items:center;justify-content:space-between;gap:12px}
.event-kind{font-size:11px;color:#64748b}
.date-badge{flex-shrink:0;background:#f1f5f9;color:#64748b;padding:4px 11px;border-radius:8px;font-size:11px;font-weight:700}
.today-badge{background:#d8f0e3;color:#146447}
.competition-card h3{font-size:19px;font-weight:800;line-height:1.9;margin:18px 0 10px}
.today-card h3{font-size:22px;color:#154c3f}
.event-date{font-size:13px;color:#475569}
.event-tags{display:flex;flex-wrap:wrap;gap:6px;margin-top:14px}
.event-tags span{font-size:11px;border-radius:7px;background:#f6f8fa;padding:4px 8px;color:#64748b}
.organizer{font-size:11px;color:#64748b;margin:14px 0 0}
.card-actions{display:flex;gap:8px;padding-top:22px;margin-top:auto}
.card-actions a{flex:1;padding-inline:8px;font-size:12px}
.empty-state{text-align:center;padding:36px 20px;border:1px dashed #d7e0e6;border-radius:16px;background:white;color:#64748b}
.empty-state strong{color:#334155}
.empty-state p{margin:8px 0 16px;font-size:13px}
.load-more{text-align:center}
.event-heading{background:#fff;border:1px solid #e2e8f0;border-radius:24px;padding:28px}
.event-today{border-top:4px solid #27856b}
.event-heading h1{font-size:clamp(23px,3vw,32px);font-weight:900;margin:12px 0}
.event-heading>p{color:#64748b;font-size:13px}
.event-summary{display:flex;justify-content:space-between;align-items:center;gap:18px;flex-wrap:wrap;margin-top:24px;padding-top:20px;border-top:1px solid #edf1f4}
.event-stats{display:flex;gap:24px;color:#64748b;font-size:12px}
.event-stats b{font-size:20px;color:#334155;margin-left:5px}
.event-heading .update-status{margin-top:18px;font-size:11px;color:#52726a;display:flex;align-items:center;gap:7px}
.event-heading .connection-error{color:#be123c}
.connection-error .status-dot{background:#be123c}
.competition-tabs{display:grid;grid-template-columns:repeat(4,1fr);gap:8px;background:#fff;border:1px solid #e2e8f0;padding:8px;border-radius:18px}
.competition-tabs button{text-align:right;padding:13px 18px;border-radius:12px;color:#64748b;font-weight:700;font-size:14px;min-height:64px}
.competition-tabs button span{display:block;font-size:10px;font-weight:400;margin-top:4px}
.competition-tabs .selected{color:#0f655d;background:#eaf5f1;box-shadow:inset 0 -3px #188574}
.competition-tabs button:hover{background:#f1f7f5}
.filter-panel{display:flex;align-items:end;flex-wrap:wrap;gap:14px;padding:20px;background:#fff;border:1px solid #e2e8f0;border-radius:18px}
.filter-panel label{display:flex;flex-direction:column;gap:7px}
.person-search{flex:1;min-width:200px}
.filter-panel select{min-width:150px}
.results-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}
.result-card{background:#fff;border:1px solid #e2e8f0;border-radius:18px;padding:18px}
.ongoing-card{border-color:#6db79f}
.result-card header{display:flex;justify-content:space-between;align-items:center}
.result-card header strong{font-size:13px}
.match-status{font-size:10px;padding:4px 9px;border-radius:7px;background:#f1f5f9;color:#64748b}
.match-status.ongoing{background:#e3f6ec;color:#16654a}
.match-status.completed{background:#eff4f7;color:#475569}
.match-meta{font-size:11px;color:#64748b;margin:9px 0 18px}
.athlete-row{display:flex;align-items:center;gap:10px;margin-top:8px;padding:12px 9px;background:#f7faff;border-radius:10px}
.athlete-row.red{background:#fff7f8}
.athlete-row>div{flex:1;min-width:0}
.athlete-row strong{font-size:13px;overflow-wrap:anywhere}
.athlete-row small{display:block;font-size:10px;color:#64748b;margin-top:3px}
.corner-dot{width:4px;align-self:stretch;border-radius:4px;background:#3873d2}
.red .corner-dot{background:#dc526d}
.winner-label{font-size:10px;color:#167351;white-space:nowrap}
.result-card footer{display:flex;justify-content:space-between;gap:10px;border-top:1px solid #eef2f5;margin-top:16px;padding-top:12px;font-size:11px;color:#64748b}
.result-card footer strong{color:#334155}
.ranking-section{background:white;border:1px solid #e2e8f0;border-radius:20px;overflow:hidden}
.ranking-section>.section-heading{padding:22px 22px 0}
.table-scroll{overflow-x:auto}
table{width:100%;border-collapse:collapse;white-space:nowrap;text-align:right}
th{font-size:11px;color:#64748b;background:#f8fafb;font-weight:500}
th,td{padding:16px 18px;border-bottom:1px solid #edf1f4}
td{font-size:13px}
.ranking-name{font-weight:700}
.points{font-weight:900;color:#0f766e;font-size:16px}
.rank-number{display:inline-grid;place-items:center;width:32px;height:32px;border-radius:10px;background:#f4f6f8;font-weight:700}
.top-rank{color:#986d18;background:#fff6df}
.scoring-help{padding:18px 22px;font-size:12px;color:#64748b}
.scoring-help summary{cursor:pointer;color:#475569;font-weight:700}
.scoring-help p{padding-top:12px;line-height:2}
.error-message{background:#fff1f2;border:1px solid #fecdd3;border-radius:14px;padding:18px;color:#be123c}
.loading-message{text-align:center;color:#64748b;padding:24px}
.site-footer{display:flex;justify-content:space-between;gap:16px;border-top:1px solid #e2e8f0;padding-block:24px;color:#94a3b8;font-size:11px}
.public-hub :is(a,button,input,select,summary,[tabindex]):focus-visible{outline:3px solid #40a990;outline-offset:3px}
@media(max-width:1000px){.competition-grid,.results-grid{grid-template-columns:repeat(2,minmax(0,1fr))}
.welcome-aside{min-width:210px;padding-inline:24px}}
@media(max-width:640px){.page-width{padding-inline:16px}
.header-inner{min-height:76px}
.header-note{display:none}
.brand{font-size:15px}
.main-content{padding-top:20px;gap:24px}
.welcome{padding-top:4px;gap:12px;flex-direction:column}
.welcome h1{font-size:32px}
.welcome p{font-size:13px}
.welcome-aside{border:0;border-top:1px solid #e2e8f0;min-width:0;flex-direction:row;flex-wrap:wrap;padding:16px 0 0;justify-content:start;gap:10px}
.welcome-aside>span:not(.calendar-mark){display:none}
.welcome-aside strong{font-size:13px}
.calendar-mark{font-size:11px;padding:5px 12px;border-radius:8px;border-top-width:2px;box-shadow:none}
.welcome-aside .text-link{margin-right:auto;font-size:10px}
.search-panel{padding:16px}
.search-row .primary-button{padding-inline:16px}
.search-row input{font-size:12px}
.competition-grid,.results-grid{grid-template-columns:1fr}
.competition-card{padding:20px}
.competition-card h3{font-size:19px}
.today-card h3{font-size:21px}
.section-heading h2{font-size:18px}
.section-heading p{font-size:11px}
.count-badge{font-size:10px}
.event-heading{padding:20px}
.event-stats{gap:14px;flex-wrap:wrap}
.event-stats b{font-size:17px}
.event-summary>.secondary-button{width:100%}
.competition-tabs{grid-template-columns:repeat(2,1fr)}
.competition-tabs button{font-size:12px;padding:12px;min-height:62px}
.filter-panel{padding:16px;gap:12px}
.person-search{flex-basis:100%}
.filter-panel select{min-width:160px}
.site-footer{font-size:10px;flex-wrap:wrap}
.result-card footer{font-size:12px}}
@media print{.site-header,.site-footer,.welcome,.competition-tabs,.filter-panel,.event-heading{display:none}
.public-hub{background:white}
.main-content{padding:0;width:100%}}
</style>
