<script setup lang="ts">
import { onMounted, ref } from 'vue'
defineProps<{ title: string; busy?: boolean }>()
const emit = defineEmits<{ close: [] }>()
const dialog = ref<HTMLDialogElement>()
onMounted(() => dialog.value?.showModal())
</script>
<template>
  <Teleport to="body"><dialog ref="dialog" class="ps ps-dialog" dir="rtl" aria-labelledby="ps-dialog-title" @cancel.prevent="!busy && emit('close')">
    <header class="ps-dialog-header"><h3 id="ps-dialog-title" class="ps-title">{{ title }}</h3><button type="button" class="ps-btn secondary small" aria-label="بستن" :disabled="busy" @click="emit('close')">✕</button></header>
    <div class="ps-dialog-body"><slot /></div>
  </dialog></Teleport>
</template>
