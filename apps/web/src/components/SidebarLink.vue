<template>
  <el-tooltip :content="item.label" :disabled="expanded" placement="right">
    <router-link
      :to="item.path"
      class="flex items-center p-3 rounded-lg transition text-gray-700"
      :class="[
        accentClasses.hover,
        active ? `${accentClasses.active} font-medium` : '',
        expanded ? '' : 'justify-center',
      ]"
    >
      <template v-if="item.icon === 'agent'">
        <svg class="w-6 h-6 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="11" width="18" height="10" rx="2"></rect>
          <circle cx="12" cy="5" r="2"></circle>
          <path d="M12 7v4"></path>
        </svg>
      </template>
      <template v-else-if="item.icon === 'memory'">
        <svg class="w-6 h-6 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <ellipse cx="12" cy="5" rx="9" ry="3"></ellipse>
          <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"></path>
          <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path>
        </svg>
      </template>
      <template v-else-if="item.icon === 'settings'">
        <svg class="w-6 h-6 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"></circle>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
        </svg>
      </template>
      <component v-else :is="item.icon" class="w-6 h-6 flex-shrink-0" />
      <span v-if="expanded" class="ml-3 text-sm">{{ item.label }}</span>
    </router-link>
  </el-tooltip>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  item: { type: Object, required: true },
  expanded: { type: Boolean, default: true },
  active: { type: Boolean, default: false },
  accent: { type: String, default: 'blue' },
})

const accentClasses = computed(() => {
  const map = {
    blue: { hover: 'hover:bg-blue-50', active: 'bg-blue-50 text-blue-600' },
    purple: { hover: 'hover:bg-purple-50', active: 'bg-purple-50 text-purple-600' },
    orange: { hover: 'hover:bg-orange-50', active: 'bg-orange-50 text-orange-600' },
  }
  return map[props.accent] || map.blue
})
</script>

<style scoped>
a {
  display: flex;
}
</style>
