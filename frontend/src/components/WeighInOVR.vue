<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { weightOptions, type AgeCategory, type Gender } from '../data/categories'
import { weighInNumbers } from '../utils/weighInNumbers'

type Entry = { id: string; number?: number; name: string; club: string; weightCategory: string; status: string; weightKg?: number | null }
type Display = { tournament: { name: string; ageCategory: AgeCategory; gender: Gender }; athletes: Entry[] }
const data = ref<Display | null>(null)
const error = ref(''), loading = ref(true), query = ref(''), weight = ref('')
const updated = ref('')
const pendingOnly = ref(false)
const viewport = ref({ width: window.innerWidth, height: window.innerHeight })
const page = ref(0)
const rotating = ref(true)
const id = new URLSearchParams(location.search).get('tournamentId') || ''
const slot = location.pathname.match(/^\/ovr\/weigh-in\/([1-9]\d*)\/?$/i)?.[1] || ''
const eventDate = new URLSearchParams(location.search).get('date')
let refreshTimer: number | undefined, pageTimer: number | undefined
let controller: AbortController | undefined
let disposed = false
const fa = (n: number) => n.toLocaleString('fa-IR')
const categories = computed(() => data.value ? weightOptions(data.value.tournament.ageCategory, data.value.tournament.gender) : [])
const numbers = computed(() => weighInNumbers(data.value?.athletes ?? [], categories.value))
const normalize = (s: string) => s.replace(/ي/g, 'ی').replace(/ك/g, 'ک').replace(/[۰-۹]/g, c => String('۰۱۲۳۴۵۶۷۸۹'.indexOf(c))).trim().toLowerCase()
const queue = computed(() => (data.value?.athletes ?? []).filter(a => a.status === 'pending' && (!weight.value || a.weightCategory === weight.value)).sort((a,b) => (numbers.value.get(a.id) ?? 0) - (numbers.value.get(b.id) ?? 0)))
const next = computed(() => queue.value[0])
const positions = computed(() => new Map(queue.value.map((a,i) => [a.id,i])))
const filtered = computed(() => (data.value?.athletes ?? []).filter(a =>
  (!weight.value || a.weightCategory === weight.value) && (!pendingOnly.value || a.status === 'pending') &&
  (!query.value.trim() || normalize(`${a.name} ${a.club} ${numbers.value.get(a.id)}`).includes(normalize(query.value))))
  .sort((a,b) => (numbers.value.get(a.id) ?? 0) - (numbers.value.get(b.id) ?? 0)))
const columns = computed(() => viewport.value.width >= 1200 ? 5 : viewport.value.width >= 900 ? 4 : viewport.value.width >= 650 ? 2 : 1)
const rows = computed(() => Math.max(4, Math.floor((viewport.value.height - 185) / 42)))
const pageSize = computed(() => columns.value * rows.value)
const pageCount = computed(() => Math.max(1, Math.ceil(filtered.value.length / pageSize.value)))
const visible = computed(() => filtered.value.slice(page.value * pageSize.value, (page.value + 1) * pageSize.value))
watch([query, weight, pendingOnly], () => { page.value = 0 })
watch(pageCount, count => { page.value = Math.min(page.value, count - 1) })
function onResize() { viewport.value = { width: window.innerWidth, height: window.innerHeight } }
function turnText(a: Entry) {
  if (a.status === 'passed') return 'تأیید شده'
  if (a.status === 'failed') return 'رد شده'
  const ahead = positions.value.get(a.id) ?? 0
  return ahead === 0 ? 'نفر بعدی صف' : `${fa(ahead)} نفر تا نوبت`
}
async function load() {
  if (!id && !slot) { error.value = 'نمایشگر را از بخش وزن‌کشی مسابقه باز کنید.'; loading.value = false; return }
  controller = new AbortController()
  const timeout = window.setTimeout(() => controller?.abort(), 8000)
  try {
    const selector = id ? `tournamentId=${encodeURIComponent(id)}` : `slot=${slot}${eventDate ? `&date=${encodeURIComponent(eventDate)}` : ''}`
    const response = await fetch(`/api/v1/public/weigh-in?${selector}`, { signal: controller.signal, cache: 'no-store' })
    if (!response.ok) throw new Error('دریافت اطلاعات وزن‌کشی ناموفق بود')
    data.value = await response.json(); error.value = ''; updated.value = new Date().toLocaleTimeString('fa-IR')
  } catch {
    if (!disposed) error.value = 'ارتباط با سرور برقرار نیست؛ اطلاعات نمایش‌داده‌شده ممکن است قدیمی باشد.'
  } finally {
    clearTimeout(timeout); loading.value = false
    if (!disposed) refreshTimer = window.setTimeout(load, 3000)
  }
}
async function fullscreen() {
  try { if (document.fullscreenElement) await document.exitFullscreen(); else await document.documentElement.requestFullscreen() }
  catch { error.value = 'مرورگر اجازهٔ نمایش تمام‌صفحه را نداد.' }
}
onMounted(() => { load(); window.addEventListener('resize', onResize); pageTimer = window.setInterval(() => { if (rotating.value && !query.value && pageCount.value > 1) page.value = (page.value + 1) % pageCount.value }, 10000) })
onUnmounted(() => { disposed = true; controller?.abort(); clearTimeout(refreshTimer); clearInterval(pageTimer); window.removeEventListener('resize', onResize) })
</script>

<template>
  <main class="queue-screen" dir="rtl">
    <header><div class="brand"><span class="tkd-mark">TKD</span><div><p class="eyebrow">WEIGH-IN · صف وزن‌کشی ورزشکاران</p><h1>{{ data?.tournament.name || 'نوبت وزن‌کشی' }}</h1></div></div><div class="tools"><span>{{ error ? 'ارتباط قطع است' : loading ? 'در حال اتصال…' : '● به‌روزرسانی زنده' }}</span><button @click="fullscreen">تمام‌صفحه</button></div></header>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <section v-if="next" class="next-strip"><span>نفر بعدی صف</span><strong class="next-number">{{ fa(numbers.get(next.id) ?? 0) }}</strong><strong>{{ next.name }}</strong><span>{{ next.club }}</span><span dir="ltr">{{ next.weightCategory }} kg</span><small>برای اعلام نوبت توسط داور آماده باشید</small></section>
    <section class="controls"><input v-model="query" aria-label="جست‌وجوی ورزشکار" placeholder="شماره، نام ورزشکار یا باشگاه…"><select v-model="weight" aria-label="دسته وزنی"><option value="">همهٔ وزن‌ها</option><option v-for="(w,i) in categories" :key="w" :value="w">وزن {{ fa(i+1) }} · {{ w }} kg</option></select><label><input v-model="pendingOnly" type="checkbox"> فقط افراد در انتظار</label><span>{{ fa(queue.length) }} نفر در انتظار · {{ fa(filtered.length) }} نفر در فهرست</span></section>
    <p v-if="loading || !data" class="empty">{{ loading ? 'در حال دریافت فهرست ورزشکاران…' : 'در انتظار ارتباط با سرور' }}</p>
    <template v-else>
      <p v-if="!queue.length && data.athletes.length" class="finished-note">همهٔ ورزشکاران این بخش وزن‌کشی شده‌اند.</p>
      <section class="athlete-list" :style="{ '--columns': columns, '--rows': rows }" aria-label="فهرست نوبت ورزشکاران">
        <article v-for="a in visible" :key="a.id" :class="{next:a.id===next?.id, completed:a.status!=='pending', failed:a.status==='failed'}">
          <strong class="number">{{ fa(numbers.get(a.id) ?? 0) }}</strong>
          <div class="person"><h2>{{ a.name }}</h2><p>{{ a.club || 'بدون باشگاه' }} <span dir="ltr">{{ a.weightCategory }} kg</span></p></div>
          <strong class="turn">{{ turnText(a) }}</strong>
        </article>
      </section>
      <p v-if="!filtered.length" class="empty">ورزشکاری با این فیلتر پیدا نشد.</p>
    </template>
    <footer><span>شماره‌ها مطابق فهرست و فرم وزن‌کشی هستند · آخرین دریافت {{ updated || '—' }}</span><nav v-if="pageCount > 1" class="page-controls" aria-label="صفحه‌های فهرست"><button @click="page = (page + pageCount - 1) % pageCount; rotating = false">قبلی</button><span>{{ fa(page + 1) }} / {{ fa(pageCount) }}</span><button @click="page = (page + 1) % pageCount; rotating = false">بعدی</button><button @click="rotating = !rotating">{{ rotating ? 'توقف گردش' : 'گردش خودکار' }}</button></nav></footer>
  </main>
</template>
<style scoped>
.queue-screen{min-height:100vh;background:#0a1220;color:#edf4ff;padding:clamp(14px,2vw,28px);font-family:Vazirmatn,sans-serif}header{display:flex;align-items:center;justify-content:space-between;gap:16px;margin-bottom:18px}.eyebrow{color:#78d8b9;font-size:12px}h1{font-size:clamp(20px,2.4vw,30px);font-weight:900}.tools{display:flex;align-items:center;gap:12px;font-size:12px;color:#78d8b9}button,select,input:not([type=checkbox]){border:1px solid #35445b;background:#152237;color:inherit;border-radius:8px;padding:10px 12px;font:inherit}button{font-size:12px}.next-strip{display:flex;flex-wrap:wrap;align-items:center;gap:18px;background:#153b34;border:1px solid #428a73;border-radius:12px;padding:12px 20px;margin-bottom:16px}.next-strip>strong{font-size:24px}.next-strip .next-number{font-size:36px;color:#9dffcf}.next-strip small{margin-right:auto;color:#a6cfc0}.controls{display:flex;flex-wrap:wrap;align-items:center;gap:12px;margin-bottom:18px;font-size:13px}.controls>input{flex:1;min-width:200px}.controls label{display:flex;align-items:center;gap:8px}.controls>span{color:#a6b9d2}.athlete-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px 16px}.athlete-list article{display:flex;align-items:center;gap:14px;background:#142035;border:1px solid #28384e;border-radius:10px;padding:13px 16px;min-height:82px}.athlete-list article.next{background:#153b34;border-color:#55b391}.number{font-size:28px;font-weight:900;color:#8bbdff;min-width:50px;text-align:center}.person{flex:1;min-width:0}.person h2{font-size:clamp(16px,1.5vw,22px);font-weight:800;overflow-wrap:anywhere}.person p{color:#9aaec8;font-size:12px;margin-top:4px;display:flex;flex-wrap:wrap;gap:10px}.turn{font-size:13px;color:#ffdc8c;white-space:nowrap}.completed .turn{color:#91b7a8;font-size:12px}.completed .number{color:#a6b9d2}.finished-note{padding:12px 16px;margin-bottom:14px;color:#ade8cc;background:#13332d;border-radius:8px;font-size:14px}.error{padding:12px 16px;background:#492532;color:#fecdd3;border-radius:8px;margin-bottom:14px}.empty{padding:50px;text-align:center;color:#a6b9d2}footer{display:flex;justify-content:space-between;gap:16px;margin-top:20px;color:#879cb7;font-size:11px}@media(min-width:1700px){.athlete-list{grid-template-columns:repeat(3,minmax(0,1fr))}}@media(max-width:900px){.athlete-list{grid-template-columns:1fr}.next-strip small{margin-right:0}}@media(max-width:520px){header,footer{flex-direction:column;align-items:flex-start}.athlete-list article{padding:10px;gap:8px}.number{font-size:24px;min-width:40px}.turn{font-size:11px}.person h2{font-size:16px}.controls>input{min-width:100%;box-sizing:border-box}}
.queue-screen{height:100vh;min-height:0;overflow:hidden;display:flex;flex-direction:column;padding:clamp(10px,1.2vw,18px)}header{margin-bottom:8px}.eyebrow{font-size:10px}h1{font-size:clamp(18px,1.8vw,25px)}.next-strip{min-height:42px;padding:5px 12px;margin-bottom:8px;gap:12px;border-radius:7px;font-size:12px}.next-strip>strong{font-size:16px}.next-strip .next-number{font-size:22px}.next-strip small{font-size:11px}.controls{margin-bottom:8px;gap:8px;font-size:11px}.controls input:not([type=checkbox]),.controls select{padding:5px 9px;font-size:11px}.controls>input{max-width:300px}.finished-note{padding:6px 10px;margin-bottom:7px;font-size:11px}.athlete-list{grid-template-columns:repeat(3,minmax(0,1fr));gap:4px 7px;min-height:0;align-content:start}.athlete-list article{min-height:0;height:58px;padding:4px 9px;border-radius:6px;gap:7px}.number{font-size:21px;min-width:36px}.person h2{font-size:clamp(12px,1.05vw,16px);line-height:1.25;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.person p{font-size:10px;margin-top:2px;gap:5px;white-space:nowrap;overflow:hidden}.turn{font-size:10px}.completed .turn{font-size:10px}footer{margin-top:auto;padding-top:7px;align-items:center}.page-controls{display:flex;align-items:center;gap:8px}.page-controls button{padding:5px 8px;font-size:10px}@media(min-width:1700px){.athlete-list{grid-template-columns:repeat(4,minmax(0,1fr))}}@media(max-width:1099px){.athlete-list{grid-template-columns:repeat(2,minmax(0,1fr))}}@media(max-width:649px){.athlete-list{grid-template-columns:1fr}.queue-screen{height:auto;min-height:100vh;overflow:visible}.athlete-list article{height:58px}.controls>input{max-width:none}.next-strip{flex-wrap:wrap}.next-strip small{display:none}footer{flex-direction:column;align-items:flex-start}}@media(max-height:700px){.next-strip{display:none}.athlete-list article{height:54px}.queue-screen{padding:8px}.controls{margin-bottom:5px}}
.athlete-list{gap:3px 5px}.athlete-list article{height:52px;padding:3px 6px;gap:5px}.number{font-size:17px;min-width:29px}.person h2{font-size:clamp(10px,.82vw,13px)}.person p{font-size:8px;gap:3px}.turn,.completed .turn{font-size:8px}.page-controls{gap:5px}@media(min-width:1200px){.athlete-list{grid-template-columns:repeat(5,minmax(0,1fr))}}@media(min-width:900px) and (max-width:1199px){.athlete-list{grid-template-columns:repeat(4,minmax(0,1fr))}}@media(min-width:650px) and (max-width:899px){.athlete-list{grid-template-columns:repeat(2,minmax(0,1fr))}}
.queue-screen{position:relative;background:radial-gradient(ellipse at 50% -30%,#183253 0%,#091321 58%,#070c16 100%);border-top:5px solid #e8f0fb;box-shadow:inset 6px 0 #1268d4,inset -6px 0 #cf244a;padding-inline:20px}
.queue-screen::before{content:'';position:absolute;z-index:0;top:0;left:0;right:0;height:3px;background:linear-gradient(90deg,#d81f4f 0 46%,#f3f6fc 46% 54%,#1573df 54% 100%)}
.queue-screen>*{position:relative;z-index:1}
header{background:linear-gradient(90deg,#112b4b,#132036 55%,#3b1527);border:1px solid #37506b;border-radius:9px;padding:7px 13px;margin-bottom:7px;box-shadow:0 4px 16px #0004}
.brand{display:flex;align-items:center;gap:10px;min-width:0}.tkd-mark{display:grid;place-items:center;flex:none;width:37px;height:37px;border:2px solid #d9e9ff;border-radius:50%;color:white;font-size:11px;font-weight:900;letter-spacing:-1px;box-shadow:inset 5px 0 #d92850,inset -5px 0 #1682ed}.eyebrow{color:#92bbef;letter-spacing:.04em;font-weight:700}.tools{color:#a0e5c8}.tools button{border-color:#66809e;background:#12243e;font-weight:700}
.controls{padding:5px 8px;border:1px solid #293b52;border-radius:7px;background:#101d30}.controls>span{font-weight:700;color:#d4e3f5}.controls input:not([type=checkbox]),.controls select{background:#172b45;border-color:#496584}
.next-strip{background:linear-gradient(90deg,#0c513c,#10433e 72%,#222b32);border-color:#c59b3c;box-shadow:inset 4px 0 #e8bc56,inset -4px 0 #e8bc56}.next-strip .next-number{color:#ffdc75}.next-strip small{color:#ffdc75}
.athlete-list{display:grid;grid-auto-flow:column;grid-template-columns:repeat(var(--columns),minmax(0,1fr))!important;grid-template-rows:repeat(var(--rows),minmax(0,1fr));flex:1;min-height:0;gap:3px 5px;align-content:stretch}
.athlete-list article{height:auto;min-height:0;overflow:hidden;border:1px solid #314358;border-radius:4px;background:linear-gradient(95deg,#162b48,#111c2d 58%,#231824);box-shadow:inset 3px 0 #2878ce,inset -3px 0 #b12748;padding:2px 7px}
.athlete-list article.next{background:linear-gradient(95deg,#544022,#24352f);border-color:#e4bc61;box-shadow:inset 3px 0 #e4bc61,inset -3px 0 #e4bc61}
.athlete-list article.failed{border-color:#653146;background:#261b2b}.athlete-list article.failed .turn{color:#ff9aab}
.athlete-list .number{font-size:clamp(15px,1.1vw,20px);font-variant-numeric:tabular-nums;color:#9cc9ff}.athlete-list .person h2{font-size:clamp(10px,.84vw,14px);color:#f4f7fc}.athlete-list .person p{color:#a7bad2}.athlete-list .turn{color:#efc778;font-weight:800}.athlete-list .completed .turn{color:#8acfb1}
footer{border-top:1px solid #31445d;margin-top:5px;padding-top:4px}.finished-note{background:#183b34;color:#b9ead0;border-right:3px solid #62caa3}
@media(max-width:649px){.queue-screen{height:auto;min-height:100vh;overflow:visible;padding-inline:10px}.athlete-list{display:flex;flex-direction:column;flex:none}.athlete-list article{min-height:43px}.brand{gap:6px}.tkd-mark{width:29px;height:29px;font-size:9px}}
</style>
