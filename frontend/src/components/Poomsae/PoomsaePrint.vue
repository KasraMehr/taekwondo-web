<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { usePoomsaeStore } from '../../stores/poomsae'
import { webApi } from '../../webApi'
import type { Sheet, SheetMode } from '../../types/poomsae'
import { attendanceLabels, fa, genderLabels, jalaliFromIso, message } from '../../utils/poomsae'
const store = usePoomsaeStore(), mode = ref<SheetMode>('results'), sheet = ref<Sheet | null>(null), error = ref(''), loading = ref(false), exporting = ref(false), exportProgress = ref('')
let generation = 0
const labels = { results: 'برگه ثبت نتایج', blank: 'برگه خام داوری', standings: 'جدول رتبه‌بندی' }
const pages = computed(() => { if (!sheet.value) return []; const rows = sheet.value.rows; return Array.from({ length: Math.max(1, Math.ceil(rows.length / 12)) }, (_, i) => rows.slice(i * 12, (i + 1) * 12)) })
async function load() { const ticket = ++generation; loading.value = true; error.value = ''; sheet.value = null; try { const data = await webApi().call<Sheet>(store.divisionPath() + `/sheet?mode=${mode.value}`); if (ticket === generation) sheet.value = data } catch (e) { if (ticket === generation) error.value = message(e) } finally { if (ticket === generation) loading.value = false } }
async function print() { if (!sheet.value || loading.value) return; await document.fonts.ready; window.print() }
async function exportPDF() {
  if (!sheet.value || exporting.value) return
  exporting.value = true; error.value = ''
  const snapshot = sheet.value
  const host = document.createElement('div'); host.className = 'ps-pdf-capture'; host.dir = 'rtl'; host.setAttribute('aria-hidden', 'true')
  const sourcePages = Array.from(document.querySelectorAll<HTMLElement>('#poomsae-print-root .ps-paper'), page => page.cloneNode(true) as HTMLElement)
  try {
    await document.fonts.ready
    const [{ jsPDF }, { domToPng }] = await Promise.all([import('jspdf'), import('modern-screenshot')])
    const pdf = new jsPDF({ orientation: 'landscape', unit: 'mm', format: 'a4', compress: true })
    pdf.setProperties({ title: `${snapshot.event.name} - ${snapshot.division.name}`, subject: labels[snapshot.mode] })
    document.body.appendChild(host)
    for (let i = 0; i < sourcePages.length; i++) {
      exportProgress.value = `آماده‌سازی صفحه ${fa(i + 1)} از ${fa(sourcePages.length)}`
      host.replaceChildren(sourcePages[i])
      await new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve())))
      const page = sourcePages[i]
      const width = Math.ceil(Math.max(page.getBoundingClientRect().width, page.scrollWidth))
      const height = Math.ceil(Math.max(page.getBoundingClientRect().height, page.scrollHeight))
      if (width < 800 || height < 200) throw new Error('ابعاد برگه برای ساخت PDF معتبر نیست.')
      const png = await domToPng(page, { width, height, scale: 3, backgroundColor: '#ffffff' })
      const image = pdf.getImageProperties(png)
      if (image.width < width * 2 || image.height < height * 2) throw new Error('تصویر کامل برگه برای PDF ساخته نشد؛ دوباره تلاش کنید.')
      const scale = Math.min(277 / image.width, 190 / image.height)
      if (i) pdf.addPage()
      pdf.addImage(png, 'PNG', (297 - image.width * scale) / 2, 10, image.width * scale, image.height * scale, undefined, 'FAST')
    }
    pdf.save(`poomsae-${snapshot.divisionId.slice(0, 8)}-${snapshot.mode}.pdf`)
  } catch (e) { error.value = `ساخت PDF انجام نشد. ${message(e)}` } finally { host.remove(); exporting.value = false; exportProgress.value = '' }
}
onMounted(() => { document.body.classList.add('poomsae-print-mode'); load() })
onUnmounted(() => { generation++; document.body.classList.remove('poomsae-print-mode') })
watch(mode, load)
</script>
<template>
  <div class="ps-card space-y-4"><div class="ps-toolbar"><div><h3 class="ps-title">چاپ و خروجی PDF</h3><p class="ps-muted mt-1">A4 افقی · {{ fa(pages.length) }} صفحه · اطلاعات ذخیره‌شدهٔ سرور</p></div><div class="ps-actions"><button class="ps-btn secondary" :disabled="!sheet || loading || exporting" @click="print">چاپ برگه</button><button class="ps-btn" :disabled="!sheet || loading || exporting" @click="exportPDF">{{ exporting ? 'در حال ساخت PDF…' : 'دانلود PDF' }}</button></div></div><div class="ps-actions"><label class="ps-field min-w-56">نوع برگه<select v-model="mode" :disabled="exporting"><option value="results">نتایج به ترتیب اجرا</option><option value="blank">برگه خام برای ثبت دستی</option><option value="standings">جدول رتبه‌بندی</option></select></label><button class="ps-btn secondary mt-6" :disabled="loading || exporting" @click="load">دریافت آخرین نتایج</button></div><p v-if="store.dirtyCount" class="ps-note">نمره‌های ذخیره‌نشده در برگه چاپ نمی‌شوند.</p><p class="ps-muted">فایل PDF با کاغذ A4 افقی آماده می‌شود. برای چاپ مستقیم نیز در پنجرهٔ چاپ، حالت افقی را انتخاب کنید.</p><p v-if="exporting" class="ps-note" role="status">{{ exportProgress }}</p><p v-if="loading" class="ps-muted" role="status">در حال آماده‌سازی برگه…</p><p v-if="error" class="ps-error" role="alert">{{ error }}</p></div>
  <Teleport to="body"><div v-if="sheet" id="poomsae-print-root" dir="rtl">
    <article v-for="(rows, page) in pages" :key="page" class="ps-paper">
      <header class="ps-paper-header"><div class="ps-paper-brand"><span class="ps-paper-symbol">◎</span><div><span class="ps-paper-eyebrow">مسابقات پومسه · انفرادی</span><h1>{{ sheet.event.name }}</h1><p>{{ labels[sheet.mode] }}</p></div></div><div class="ps-paper-meta"><strong>{{ sheet.final ? 'نتایج نهایی' : sheet.mode === 'blank' ? 'فرم داوری' : 'نتایج موقت' }}</strong><span>تاریخ مسابقه: {{ jalaliFromIso(sheet.event.date) }}</span><span>صفحه {{ fa(page + 1) }} از {{ fa(pages.length) }}</span></div></header>
      <div class="ps-paper-info"><strong>{{ sheet.division.name }}</strong><span>{{ genderLabels[sheet.division.gender] }}</span><span v-if="sheet.division.belt">کمربند: {{ sheet.division.belt }}</span><span v-if="sheet.event.rules.formSelection === 'division'">فرم اول: {{ fa(sheet.division.form1Code) }} · فرم دوم: {{ fa(sheet.division.form2Code) }}</span><span>قرعه {{ fa(sheet.draw?.version) }}</span></div>
      <table class="ps-paper-table"><colgroup><col style="width:4%"><col style="width:9%"><col style="width:12%"><col style="width:13%"><col style="width:8%"><col style="width:8%"><col style="width:8%"><col style="width:8%"><col style="width:8%"><col style="width:8%"><col style="width:8%"><col style="width:6%"></colgroup><thead><tr><th rowspan="2">ترتیب</th><th rowspan="2">نام</th><th rowspan="2">نام خانوادگی</th><th rowspan="2">تیم</th><th colspan="3">فرم اول</th><th colspan="3">فرم دوم</th><th rowspan="2">امتیاز کل</th><th rowspan="2">رتبه</th></tr><tr><th>دقت</th><th>اجرا</th><th>نهایی</th><th>دقت</th><th>اجرا</th><th>نهایی</th></tr></thead><tbody><tr v-for="row in rows" :key="row.entry.id"><td>{{ fa(row.position) }}</td><td>{{ row.entry.firstName }}</td><td>{{ row.entry.lastName }}<small v-if="sheet.event.rules.formSelection === 'entry'">فرم‌ها: {{ fa(row.entry.scores.form1.code) }} / {{ fa(row.entry.scores.form2.code) }}</small></td><td>{{ row.entry.teamName }}<small v-if="row.entry.status !== 'active'">{{ attendanceLabels[row.entry.status] }}</small></td><td>{{ row.entry.scores.form1.accuracy === null && sheet.mode === 'blank' ? '' : fa(row.entry.scores.form1.accuracy) }}</td><td>{{ row.entry.scores.form1.presentation === null && sheet.mode === 'blank' ? '' : fa(row.entry.scores.form1.presentation) }}</td><td class="ps-paper-subtotal">{{ sheet.mode === 'blank' ? '' : fa(row.form1Total) }}</td><td>{{ sheet.mode === 'blank' ? '' : fa(row.entry.scores.form2.accuracy) }}</td><td>{{ sheet.mode === 'blank' ? '' : fa(row.entry.scores.form2.presentation) }}</td><td class="ps-paper-subtotal">{{ sheet.mode === 'blank' ? '' : fa(row.form2Total) }}</td><td class="ps-paper-total">{{ sheet.mode === 'blank' ? '' : fa(row.total) }}</td><td>{{ sheet.mode === 'blank' ? '' : fa(row.rank) }}</td></tr><tr v-if="!rows.length"><td colspan="12">هنوز ورزشکاری در این رده ثبت نشده است.</td></tr></tbody></table>
      <footer class="ps-paper-footer"><div class="ps-signatures"><span>نام و امضای سرداور: ..................................</span><span>نام و امضای مسئول برگزاری: ..................................</span></div><div class="ps-paper-footnote"><span>تهیه‌شده در {{ new Date(sheet.generatedAt).toLocaleString('fa-IR') }} · نسخه {{ fa(sheet.revision) }}</span><span>نمرهٔ هر فرم = دقت + اجرا · امتیاز کل = جمع دو فرم</span></div></footer>
    </article>
  </div></Teleport>
</template>
<style>
#poomsae-print-root { max-width:1120px; margin:20px auto 50px; padding:0 16px; font-family:Vazirmatn,sans-serif; color:#172b35; }
.ps-paper { background:white; border:1px solid #dbe3e8; box-shadow:0 10px 25px #0f172a0d; padding:28px; margin-bottom:24px; }
.ps-paper-header { display:flex; justify-content:space-between; gap:20px; padding-bottom:18px; border-bottom:3px solid #173e43; }
.ps-paper-brand { display:flex; align-items:center; gap:15px; }.ps-paper-symbol { font-size:44px; color:#235e62; }.ps-paper-eyebrow { font-size:10px; letter-spacing:.6px; color:#597379; }.ps-paper-header h1 { font-size:23px; font-weight:900; margin:5px 0; }.ps-paper-header p { font-size:12px; }
.ps-paper-meta { display:flex; flex-direction:column; justify-content:center; gap:7px; font-size:10px; text-align:left; }.ps-paper-meta strong { border:1px solid #64748b; padding:5px 10px; text-align:center; }
.ps-paper-info { display:flex; gap:18px; flex-wrap:wrap; align-items:center; padding:14px 0; font-size:11px; }
.ps-paper-table { border-collapse:collapse; width:100%; table-layout:fixed; text-align:center; font-size:10px; }.ps-paper-table th { padding:9px 3px; background:#eef4f4; font-weight:800; border:1px solid #8c9da3; }.ps-paper-table td { border:1px solid #a8b6bb; padding:10px 4px; overflow-wrap:anywhere; height:40px; line-height:1.8; }.ps-paper-table small { display:block; font-size:8px; color:#475569; }.ps-paper-subtotal { font-weight:700; background:#f8fafb; }.ps-paper-total { font-weight:900; border-inline:2px solid #627e82!important; }
.ps-signatures { display:flex; justify-content:space-between; gap:20px; padding:28px 0; font-size:11px; }.ps-paper-footnote { display:flex; justify-content:space-between; gap:12px; border-top:1px solid #cbd5e1; padding-top:10px; color:#526671; font-size:8px; }
.ps-pdf-capture { position:fixed; left:0; top:0; z-index:-1; width:1120px; opacity:0; pointer-events:none; font-family:Vazirmatn,sans-serif; color:#172b35; }
.ps-pdf-capture .ps-paper { box-sizing:border-box; width:1120px!important; padding:0; margin:0; border:0; box-shadow:none; }
.ps-pdf-capture .ps-paper-table { font-size:11px; }.ps-pdf-capture .ps-paper-table td { height:32px; padding:6px 4px; line-height:1.6; }
@media screen and (max-width:900px) { #poomsae-print-root { overflow-x:auto; }.ps-paper { width:1050px; } }
@media print {
  @page { size:A4 landscape; margin:10mm; }
  body.poomsae-print-mode > *:not(#poomsae-print-root) { display:none!important; }
  body.poomsae-print-mode { background:white!important; margin:0!important; }
  #poomsae-print-root { display:block!important; width:277mm; max-width:none; margin:0; padding:0; color:black; }
  .ps-paper { padding:0; margin:0; border:0; box-shadow:none; break-after:page; }
  .ps-paper:last-child { break-after:auto; }
  .ps-paper-header { padding-bottom:4mm; }.ps-paper-header h1 { font-size:18px; }.ps-paper-header p { font-size:10px; }.ps-paper-symbol { font-size:32px; }.ps-paper-meta { font-size:9px; gap:3px; }
  .ps-paper-info { padding:3mm 0; font-size:10px; }.ps-paper-table { font-size:10px; }.ps-paper-table th { padding:2mm 1mm; }.ps-paper-table td { padding:1.3mm 1mm; height:8mm; }
  .ps-paper-table thead { display:table-header-group; }.ps-paper-table tr { break-inside:avoid; }.ps-paper-footer { break-inside:avoid; }.ps-signatures { padding:7mm 0 5mm; font-size:10px; }.ps-paper-footnote { padding-top:2mm; }
}
</style>
