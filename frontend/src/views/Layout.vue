<template>
  <div class="flex h-screen bg-gray-50">
    <aside class="w-16 md:w-52 bg-white border-r flex flex-col">
      <div class="p-4 text-xl font-bold text-blue-600 border-b text-center">
        <span class="hidden md:inline">AI Collab Hub</span>
        <span class="md:hidden">AI</span>
      </div>
      <nav class="flex-1 space-y-1 px-2 py-3 overflow-y-auto">
        <div class="text-xs text-gray-400 px-3 py-1 hidden md:block text-center">工作台</div>
        <router-link v-for="i in workbench" :key="i.path" :to="i.path"
          class="flex items-center justify-center md:justify-start p-3 rounded-lg transition hover:bg-blue-50 text-gray-700"
          :class="{ 'bg-blue-50 text-blue-600 font-medium': $route.path === i.path }">
          <component :is="i.icon" class="w-6 h-6 flex-shrink-0" />
          <span class="ml-3 hidden md:inline text-sm">{{ i.label }}</span>
        </router-link>

        <div class="text-xs text-gray-400 px-3 py-1 mt-4 hidden md:block text-center">AI 系统</div>
        <router-link v-for="i in aiSystem" :key="i.path" :to="i.path"
          class="flex items-center justify-center md:justify-start p-3 rounded-lg transition hover:bg-purple-50 text-gray-700"
          :class="{ 'bg-purple-50 text-purple-600 font-medium': $route.path === i.path }">
          <template v-if="i.icon === 'agent'">
            <svg class="w-6 h-6 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="3" y="11" width="18" height="10" rx="2"></rect>
              <circle cx="12" cy="5" r="2"></circle>
              <path d="M12 7v4"></path>
              <line x1="8" y1="16" x2="8" y2="16"></line>
              <line x1="16" y1="16" x2="16" y2="16"></line>
            </svg>
          </template>
          <component v-else :is="i.icon" class="w-6 h-6 flex-shrink-0" />
          <span class="ml-3 hidden md:inline text-sm">{{ i.label }}</span>
        </router-link>

        <div class="text-xs text-gray-400 px-3 py-1 mt-4 hidden md:block text-center">中枢系统</div>
        <router-link v-for="i in adminItems" :key="i.path" :to="i.path"
          class="flex items-center justify-center md:justify-start p-3 rounded-lg transition hover:bg-orange-50 text-gray-700"
          :class="{ 'bg-orange-50 text-orange-600 font-medium': $route.path === i.path }">
          <template v-if="i.icon === 'memory'">
            <svg class="w-6 h-6 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <ellipse cx="12" cy="5" rx="9" ry="3"></ellipse>
              <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"></path>
              <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path>
            </svg>
          </template>
          <component v-else :is="i.icon" class="w-6 h-6 flex-shrink-0" />
          <span class="ml-3 hidden md:inline text-sm">{{ i.label }}</span>
        </router-link>
      </nav>
      <div class="p-2 border-t">
        <router-link to="/admin"
          class="flex items-center justify-center md:justify-start p-3 rounded-lg transition hover:bg-orange-50 text-gray-600 hover:text-orange-600">
          <svg class="w-6 h-6 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="3"></circle>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
          </svg>
          <span class="ml-3 hidden md:inline text-sm">系统设置</span>
        </router-link>
      </div>
    </aside>
    <main class="flex-1 p-4 md:p-8 overflow-auto">
      <router-view />
    </main>
  </div>
</template>
<script setup>
import { useRouter } from 'vue-router'
import { DataAnalysis, Promotion, Files, Monitor, Connection, Reading } from '@element-plus/icons-vue'
const router = useRouter()

const logout = () => {
  localStorage.removeItem('token')
  router.push('/login')
}

const workbench = [
  { path: '/', label: '概览', icon: DataAnalysis },
  { path: '/tasks', label: '任务编排', icon: Promotion },
  { path: '/files', label: '文件管理', icon: Files },
]
const aiSystem = [
  { path: '/terminals', label: '终端管理', icon: Monitor },
  { path: '/agents', label: 'Agent 管理', icon: 'agent' },
  { path: '/orchestrator', label: 'Agent 编排', icon: Connection },
]
const adminItems = [
  { path: '/knowledge', label: '知识中枢', icon: Reading },
  { path: '/memory', label: '记忆系统', icon: 'memory' },
]
</script>
