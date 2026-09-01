<template>
  <span class="inline-flex items-center justify-center w-5 h-5 text-xs font-medium rounded bg-gray-100 text-gray-600 uppercase">
    <template v-if="isDir">DIR</template>
    <template v-else>{{ label }}</template>
  </span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  ext: { type: String, default: '' },
  isDir: { type: Boolean, default: false },
})

const label = computed(() => {
  const ext = (props.ext || '').replace('.', '').slice(0, 3)
  return ext || 'file'
})
</script>
