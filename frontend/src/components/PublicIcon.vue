<script setup lang="ts">
defineProps<{ name: string }>()
const paths: Record<string, string> = {
  bracket: 'M4 3v5h5m-5 8v5h5M9 8v8H4m5-4h6m0-8v16m0-8h5',
  results: 'M3 12h4l3-7 4 14 3-7h4',
  players: 'M8 3h8l-1 6-3 3-3-3-1-6Zm1 6-4 5 3 1 1 5 3-4 3 4 1-5 3-1-4-5',
  teams: 'M9 9a3 3 0 1 0 0-6 3 3 0 0 0 0 6Zm-6 11v-3a6 6 0 0 1 12 0v3m1-16a3 3 0 0 1 0 6m2 3a5 5 0 0 1 3 5v2',
  calendar: 'M7 3v4m10-4v4M4 10h16M5 5h14a1 1 0 0 1 1 1v14H4V6a1 1 0 0 1 1-1Zm3 9h2m4 0h2m-8 3h2',
  search: 'M10 17a7 7 0 1 0 0-14 7 7 0 0 0 0 14Zm5-2 6 6',
}
</script>
<template>
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path :d="paths[name] || paths.bracket" /></svg>
</template>
