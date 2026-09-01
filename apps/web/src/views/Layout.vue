<template>
  <div class="flex h-screen bg-gray-50">
    <aside
      class="bg-white border-r flex flex-col transition-all duration-200 shrink-0"
      :class="sidebarExpanded ? 'w-56' : 'w-16'"
    >
      <div class="p-4 border-b flex items-center gap-2">
        <div class="flex-1 min-w-0 text-lg font-bold text-blue-600 whitespace-nowrap">
          <span v-if="sidebarExpanded">AI Collab Hub</span>
          <span v-else>AI</span>
        </div>
        <button
          type="button"
          class="shrink-0 p-1.5 rounded-md hover:bg-gray-100 text-gray-500"
          :aria-label="sidebarExpanded ? '收起侧边栏' : '展开侧边栏'"
          @click="sidebarExpanded = !sidebarExpanded"
        >
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path v-if="sidebarExpanded" d="M15 18l-6-6 6-6" />
            <path v-else d="M9 18l6-6-6-6" />
          </svg>
        </button>
      </div>

      <nav class="flex-1 space-y-1 px-2 py-3 overflow-y-auto">
        <div v-if="sidebarExpanded" class="text-xs text-gray-400 px-3 py-1 text-center">工作台</div>
        <SidebarLink
          v-for="item in workbench"
          :key="item.path"
          :item="item"
          :expanded="sidebarExpanded"
          :active="isNavActive(item.path)"
          accent="blue"
        />

        <div v-if="sidebarExpanded" class="text-xs text-gray-400 px-3 py-1 mt-4 text-center">AI 系统</div>
        <SidebarLink
          v-for="item in aiSystem"
          :key="item.path"
          :item="item"
          :expanded="sidebarExpanded"
          :active="isNavActive(item.path)"
          accent="purple"
        />

        <div v-if="sidebarExpanded" class="text-xs text-gray-400 px-3 py-1 mt-4 text-center">中枢系统</div>
        <SidebarLink
          v-for="item in adminItems"
          :key="item.path"
          :item="item"
          :expanded="sidebarExpanded"
          :active="isNavActive(item.path)"
          accent="orange"
        />
      </nav>

      <div class="p-2 border-t space-y-1">
        <SidebarLink
          :item="{ path: '/admin', label: '系统设置', icon: 'settings' }"
          :expanded="sidebarExpanded"
          :active="isNavActive('/admin')"
          accent="orange"
        />

        <button
          type="button"
          class="w-full flex items-center p-3 rounded-lg transition hover:bg-red-50 text-gray-600 hover:text-red-600"
          :class="{ 'justify-center': !sidebarExpanded }"
          @click="logout"
        >
          <el-tooltip content="退出登录" :disabled="sidebarExpanded" placement="right">
            <svg class="w-6 h-6 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
              <polyline points="16 17 21 12 16 7"></polyline>
              <line x1="21" y1="12" x2="9" y2="12"></line>
            </svg>
          </el-tooltip>
          <span v-if="sidebarExpanded" class="ml-3 text-sm">退出登录</span>
        </button>
      </div>
    </aside>

    <main class="flex-1 p-4 md:p-8 overflow-auto">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { DataAnalysis, Promotion, Files, Monitor, Connection, Reading } from '@element-plus/icons-vue'
import SidebarLink from '../components/SidebarLink.vue'
import { clearAuth } from '../utils/auth'
import { notifySuccess } from '../utils/notify'

const router = useRouter()
const route = useRoute()
const sidebarExpanded = ref(window.innerWidth >= 768)

const logout = () => {
  clearAuth()
  notifySuccess('已退出登录')
  router.push('/login')
}

const isNavActive = (path) => {
  const current = route.path
  if (path === '/') return current === '/' || current === ''
  return current === path || current.startsWith(`${path}/`)
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
