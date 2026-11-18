<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useLeagueStore } from '../../stores/league'
import { loadWebLeagues, webApi } from '../../webApi'
import { AGE_CATEGORIES, GENDER_LABELS } from '../../data/categories'
const store=useLeagueStore()
const league=computed(()=>store.currentLeague)
const busy=ref(false), error=ref(''), saved=ref(false)
const form=reactive({name:'',gender:'male',ageCategory:'بزرگسالان',seasonName:'',stageCount:1})
watch(league,l=>{if(l) Object.assign(form,{name:l.name,gender:l.gender,ageCategory:l.ageCategory,seasonName:l.season.name,stageCount:l.season.stages.length})},{immediate:true})
async function save(){
 if(!league.value || league.value.structureLocked || busy.value)return
 busy.value=true;error.value='';saved.value=false
 try {await webApi().call(`/leagues/${league.value.id}`,'PUT',form);store.replaceFromBackend(await loadWebLeagues());saved.value=true}
 catch(e:any){error.value=e?.message||'ویرایش لیگ ناموفق بود'}finally{busy.value=false}
}
</script>
<template>
 <form v-if="league" @submit.prevent="save" class="rounded-2xl border bg-white p-6">
  <h3 class="mb-3 text-lg font-bold">مشخصات و ساختار لیگ</h3>
  <p class="mb-4 text-sm text-amber-800">{{league.structureLocked?'اولین مبارزه شروع شده؛ مشخصات، مراحل و گروه‌بندی قفل هستند. افزودن تیم و ورزشکار همچنان مجاز است.':'این تنظیمات تا شروع اولین مبارزه قابل ویرایش هستند. مرحله‌ای که مسابقه دارد حذف نمی‌شود؛ ردهٔ جدید باید با ورزشکاران ثبت‌شده سازگار باشد.'}}</p>
  <fieldset :disabled="league.structureLocked || busy" class="grid gap-4 sm:grid-cols-2 disabled:opacity-60">
   <label>نام لیگ<input v-model="form.name" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
   <label>نام فصل<input v-model="form.seasonName" class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
   <label>ردهٔ سنی<select v-model="form.ageCategory" class="mt-1 w-full rounded-lg border px-3 py-2"><option v-for="age in AGE_CATEGORIES" :key="age">{{age}}</option></select></label>
   <label>جنسیت<select v-model="form.gender" class="mt-1 w-full rounded-lg border px-3 py-2"><option v-for="(label,key) in GENDER_LABELS" :key="key" :value="key">{{label}}</option></select></label>
   <label>تعداد مراحل<input v-model.number="form.stageCount" type="number" min="1" max="10" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
   <button class="self-end rounded-lg bg-emerald-700 px-4 py-2 text-white">{{busy?'در حال ذخیره…':'ذخیره مشخصات لیگ'}}</button>
  </fieldset>
  <p v-if="error" role="alert" class="mt-3 text-rose-700">{{error}}</p>
  <p v-if="saved" role="status" class="mt-3 text-emerald-700">مشخصات لیگ ذخیره شد.</p>
 </form>
</template>
