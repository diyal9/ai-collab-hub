<template>
  <div>
    <h1 class="text-2xl font-bold mb-6">📊 协作平台概览</h1>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-2 md:grid-cols-3 gap-4 mb-8">
      <div class="bg-white rounded-xl shadow p-5 text-center">
        <div class="text-3xl font-bold text-blue-600">{{ onlineAgents }}</div>
        <div class="text-sm text-gray-500 mt-1">在线 Agent</div>
      </div>
      <div class="bg-white rounded-xl shadow p-5 text-center">
        <div class="text-3xl font-bold text-green-600">{{ activeFlows }}</div>
        <div class="text-sm text-gray-500 mt-1">运行流程</div>
      </div>
      <div class="bg-white rounded-xl shadow p-5 text-center">
        <div class="text-3xl font-bold text-yellow-600">{{ pendingKnowledge }}</div>
        <div class="text-sm text-gray-500 mt-1">待审知识</div>
      </div>
    </div>

    <!-- 快速入口 -->
    <div class="grid grid-cols-2 md:grid-cols-3 gap-4 mb-8">
      <router-link to="/terminals" class="bg-white rounded-xl shadow p-5 hover:shadow-md transition text-center cursor-pointer flex flex-col items-center justify-center">
        <svg class="w-12 h-12 mx-auto mb-2 text-gray-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
          <line x1="8" y1="21" x2="16" y2="21"></line>
          <line x1="12" y1="17" x2="12" y2="21"></line>
        </svg>
        <div class="font-medium">Agent 终端</div>
        <div class="text-xs text-gray-500 mt-1">管理远程连接</div>
      </router-link>
      <router-link to="/orchestrator" class="bg-white rounded-xl shadow p-5 hover:shadow-md transition text-center cursor-pointer flex flex-col items-center justify-center">
        <svg class="w-12 h-12 mx-auto mb-2 text-gray-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="18" cy="5" r="3"></circle>
          <circle cx="6" cy="12" r="3"></circle>
          <circle cx="18" cy="19" r="3"></circle>
          <line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line>
          <line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line>
        </svg>
        <div class="font-medium">Agent 编排</div>
        <div class="text-xs text-gray-500 mt-1">可视化流程</div>
      </router-link>
      <router-link to="/knowledge" class="bg-white rounded-xl shadow p-5 hover:shadow-md transition text-center cursor-pointer flex flex-col items-center justify-center">
        <svg class="w-12 h-12 mx-auto mb-2 text-gray-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path>
          <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path>
        </svg>
        <div class="font-medium">知识中枢</div>
        <div class="text-xs text-gray-500 mt-1">沉淀业务</div>
      </router-link>
    </div>

    <!-- 最近任务 -->
    <div class="bg-white rounded-xl shadow p-5">
      <h2 class="font-semibold mb-4">最近任务</h2>
      <div v-if="recentTasks.length === 0" class="text-gray-400 text-sm">暂无任务</div>
      <div v-for="t in recentTasks" :key="t.id" class="flex justify-between items-center py-2 border-b last:border-0">
        <div>
          <span class="font-medium">{{ t.title }}</span>
          <span class="text-xs text-gray-400 ml-2">{{ t.type }}</span>
        </div>
        <span class="text-xs px-2 py-0.5 rounded" :class="taskStatusClass(t.status)">{{ t.status }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getAgents, getFlows, getKnowledge, getTasks } from '../api'

const onlineAgents = ref(0)
const activeFlows = ref(0)
const pendingKnowledge = ref(0)
const recentTasks = ref([])

onMounted(async () => {
  try {
    const [agents, flows, knowledge, tasks] = await Promise.all([
      getAgents().catch(() => ({ data: { online_count: 0 } })),
      getFlows().catch(() => ({ data: [] })),
      getKnowledge({ status: 'pending' }).catch(() => ({ data: [] })),
      getTasks().catch(() => ({ data: [] }))
    ])
    onlineAgents.value = agents.data?.online_count || 0
    activeFlows.value = flows.data?.filter(f => f.status === 'active').length || 0
    pendingKnowledge.value = knowledge.data?.length || 0
    recentTasks.value = (tasks.data || []).slice(0, 5)
  } catch (e) {
    console.error('Failed to load dashboard data:', e)
  }
})

const taskStatusClass = (s) => {
  return {
    pending: 'bg-gray-100 text-gray-600',
    running: 'bg-blue-100 text-blue-600',
    completed: 'bg-green-100 text-green-600',
    failed: 'bg-red-100 text-red-600'
  }[s] || 'bg-gray-100 text-gray-600'
}
</script>
