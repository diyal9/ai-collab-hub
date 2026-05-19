<template>
  <div class="flex h-screen bg-gray-50">
    <aside class="w-16 md:w-52 bg-white border-r flex flex-col">
      <div class="p-4 text-xl font-bold text-blue-600 border-b">
        <span class="hidden md:inline">AI Collab Hub</span>
        <span class="md:hidden">AI</span>
      </div>
      <nav class="flex-1 space-y-1 px-2 py-3 overflow-y-auto">
        <div class="text-xs text-gray-400 px-3 py-1 hidden md:block">工作台</div>
        <router-link v-for="i in workbench" :key="i.path" :to="i.path"
          class="flex items-center p-3 rounded-lg transition hover:bg-blue-50 text-gray-700"
          :class="{ 'bg-blue-50 text-blue-600 font-medium': $route.path === i.path }">
          <span class="text-xl w-7 text-center">{{ i.icon }}</span>
          <span class="ml-3 hidden md:inline text-sm">{{ i.label }}</span>
        </router-link>

        <div class="text-xs text-gray-400 px-3 py-1 mt-4 hidden md:block">AI 系统</div>
        <router-link v-for="i in aiSystem" :key="i.path" :to="i.path"
          class="flex items-center p-3 rounded-lg transition hover:bg-purple-50 text-gray-700"
          :class="{ 'bg-purple-50 text-purple-600 font-medium': $route.path === i.path }">
          <span class="text-xl w-7 text-center">{{ i.icon }}</span>
          <span class="ml-3 hidden md:inline text-sm">{{ i.label }}</span>
        </router-link>

        <div class="text-xs text-gray-400 px-3 py-1 mt-4 hidden md:block">管理</div>
        <router-link v-for="i in adminItems" :key="i.path" :to="i.path"
          class="flex items-center p-3 rounded-lg transition hover:bg-orange-50 text-gray-700"
          :class="{ 'bg-orange-50 text-orange-600 font-medium': $route.path === i.path }">
          <span class="text-xl w-7 text-center">{{ i.icon }}</span>
          <span class="ml-3 hidden md:inline text-sm">{{ i.label }}</span>
        </router-link>
      </nav>
      <div class="p-2 border-t">
        <button @click="logout" class="w-full flex items-center p-3 rounded-lg transition hover:bg-red-50 text-gray-600 hover:text-red-600">
          <span class="text-xl w-7 text-center">🚪</span>
          <span class="ml-3 hidden md:inline text-sm">退出登录</span>
        </button>
      </div>
    </aside>
    <main class="flex-1 p-4 md:p-8 overflow-auto">
      <router-view />
    </main>
  </div>
</template>
<script setup>
import { useRouter } from 'vue-router'
const router = useRouter()

const logout = () => {
  localStorage.removeItem('token')
  router.push('/login')
}

const workbench = [
  { path: '/', label: '概览', icon: '📊' },
  { path: '/tasks', label: '任务编排', icon: '🚀' },
  { path: '/files', label: '文件管理', icon: '📂' },
]
const aiSystem = [
  { path: '/terminals', label: '终端管理', icon: '💻' },
  { path: '/agents', label: 'Agent 管理', icon: '🤖' },
  { path: '/skills', label: '技能管理', icon: '🧩' },
  { path: '/orchestrator', label: 'Agent 编排', icon: '🔗' },
  { path: '/knowledge', label: '知识中枢', icon: '🧠' },
]
const adminItems = [
  { path: '/admin', label: '系统管理', icon: '⚙️' }
]
</script>
