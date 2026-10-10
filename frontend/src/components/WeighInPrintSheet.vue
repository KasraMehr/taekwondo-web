<template>
  <div class="weigh-print-preview" dir="rtl">
    <div class="weigh-print-toolbar">
      <strong>پیش‌نمایش فرم خام وزن‌کشی</strong>
      <div>
        <button type="button" @click="$emit('print')">🖨️ چاپ همهٔ وزن‌ها</button>
        <button type="button" @click="$emit('close')">بستن</button>
      </div>
    </div>
    <div class="weigh-print-sheet">
    <section v-for="group in groups" :key="group.weight" class="weigh-print-page">
      <header class="weigh-print-header">
        <div>
          <h1>فرم خام وزن‌کشی ورزشکاران</h1>
          <strong>{{ tournament?.name }}</strong>
        </div>
        <div class="weigh-print-meta">
          <strong>{{ group.label }}</strong>
          <span>تاریخ: {{ printDate }}</span>
          <span>تعداد ورزشکاران: {{ fa(group.athletes.length) }}</span>
        </div>
      </header>
      <p class="weigh-print-note">وزن‌های اندازه‌گیری‌شده و نتیجه را با خودکار در خانه‌های خالی بنویسید.</p>
      <table>
        <thead>
          <tr>
            <th class="number">شماره</th>
            <th class="name">نام ورزشکار</th>
            <th class="club">باشگاه / تیم</th>
            <th v-for="attempt in attemptColumns" :key="attempt" class="measure">وزن {{ fa(attempt) }}</th>
            <th class="result">نتیجه</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="athlete in group.athletes" :key="athlete.id">
            <td class="number">{{ fa(athlete.number) }}</td>
            <td class="name">{{ athlete.name }}</td>
            <td class="club">{{ athlete.club || '—' }}</td>
            <td v-for="attempt in attemptColumns" :key="attempt"></td>
            <td class="weigh-print-result"><span>□ تأیید</span><span>□ رد</span></td>
          </tr>
        </tbody>
      </table>
      <footer>نام داور: ........................................ <span>برگه وزن {{ group.label }}</span></footer>
    </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Athlete, Tournament } from '../types'
const props = defineProps<{
  tournament: Tournament | null
  groups: { weight: string; label: string; athletes: Athlete[] }[]
  maxAttempts: number
}>()
defineEmits<{ print: []; close: [] }>()
const attemptColumns = computed(() => Number.isInteger(props.maxAttempts) && props.maxAttempts > 0 ? props.maxAttempts : 2)
const fa = (value: number | undefined) => String(value ?? '').replace(/\d/g, digit => '۰۱۲۳۴۵۶۷۸۹'[Number(digit)])
const printDate = computed(() => {
  const date = props.tournament?.date
  return date ? new Date(date).toLocaleDateString('fa-IR') : '................'
})
</script>

<style>
.weigh-print-preview { font-family: Vazirmatn, sans-serif; }
.weigh-print-sheet { color: #111827; background: white; }
.weigh-print-page { box-sizing: border-box; width: 190mm; min-height: 277mm; margin: 0 auto; padding: 0; background: white; }
.weigh-print-header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #111827; padding-bottom: 4mm; gap: 8mm; }
.weigh-print-header h1 { margin: 0 0 2mm; font-size: 15pt; }
.weigh-print-header strong { font-size: 10pt; }
.weigh-print-meta { display: grid; gap: 1mm; text-align: left; font-size: 9pt; }
.weigh-print-note { font-size: 8pt; margin: 4mm 0; }
.weigh-print-sheet table { width: 100%; table-layout: fixed; border-collapse: collapse; font-size: 8.5pt; }
.weigh-print-sheet thead { display: table-header-group; }
.weigh-print-sheet tr { break-inside: avoid; }
.weigh-print-sheet th, .weigh-print-sheet td { border: .25mm solid #475569; padding: 1mm; text-align: center; }
.weigh-print-sheet th { background: #f1f5f9; font-weight: 800; }
.weigh-print-sheet td { height: 8mm; overflow-wrap: anywhere; }
.weigh-print-sheet .number { width: 12mm; font-weight: 800; }
.weigh-print-sheet .name { width: 39mm; text-align: right; }
.weigh-print-sheet .club { width: 37mm; text-align: right; }
.weigh-print-sheet .measure { width: 16mm; }
.weigh-print-sheet .result { width: 28mm; }
.weigh-print-result { white-space: nowrap; }
.weigh-print-result span + span { margin-right: 3mm; }
.weigh-print-sheet footer { display: flex; justify-content: space-between; margin-top: 5mm; font-size: 9pt; }
@media screen {
  .weigh-print-preview { position: fixed; inset: 0; z-index: 1000; overflow: auto; padding: 20px; background: rgba(15,23,42,.75); }
  .weigh-print-toolbar { position: sticky; top: 0; z-index: 1; display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 12px; max-width: 190mm; margin: 0 auto 12px; padding: 10px 14px; border-radius: 10px; background: white; }
  .weigh-print-toolbar div { display: flex; gap: 8px; }
  .weigh-print-toolbar button { padding: 6px 12px; border: 1px solid #cbd5e1; border-radius: 7px; font-size: 12px; font-weight: 700; }
  .weigh-print-toolbar button:first-child { background: #0f172a; color: white; }
  .weigh-print-page { padding: 10mm; width: 210mm; min-height: 297mm; margin-bottom: 18px; box-shadow: 0 8px 30px #02061755; }
}
@media print {
  @page { size: A4 portrait; margin: 10mm; }
  body > #app { display: none !important; }
  .weigh-print-preview { position: static; padding: 0; background: white; }
  .weigh-print-toolbar { display: none !important; }
  .weigh-print-page { break-after: page; page-break-after: always; }
  .weigh-print-page:last-child { break-after: auto; page-break-after: auto; }
}
</style>
