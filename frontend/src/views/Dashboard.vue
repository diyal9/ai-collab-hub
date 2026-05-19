<template>
  <div>
    <h1 class="text-2xl font-bold mb-6">📊 协作平台概览</h1>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
      <div class="bg-white rounded-xl shadow p-5">
        <div class="text-3xl font-bold text-blue-600">{{ onlineAgents }}</div>
        <div class="text-sm text-gray-500 mt-1">在线 Agent</div>
      </div>
      <div class="bg-white rounded-xl shadow p-5">
        <div class="text-3xl font-bold text-green-600">{{ activeFlows }}</div>
        <div class="text-sm text-gray-500 mt-1">运行流程</div>
      </div>
      <div class="bg-white rounded-xl shadow p-5">
        <div class="text-3xl font-bold text-purple-600">{{ skillCount }}</div>
        <div class="text-sm text-gray-500 mt-1">可用技能</div>
      </div>
      <div class="bg-white rounded-xl shadow p-5">
        <div class="text-3xl font-bold text-yellow-600">{{ pendingKnowledge }}</div>
        <div class="text-sm text-gray-500 mt-1">待审知识</div>
      </div>
    </div>

    <!-- 快速入口 -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
      <router-link to="/terminals" class="bg-white rounded-xl shadow p-5 hover:shadow-md transition text-center cursor-pointer">
        <div class="text-4xl mb-2">💻</div>
        <div class="font-medium">Agent 终端</div>
        <div class="text-xs text-gray-500 mt-1">管理远程连接</div>
      </router-link>
      <router-link to="/skills" class="bg-white rounded-xl shadow p-5 hover:shadow-md transition text-center cursor-pointer">
        <div class="text-4xl mb-2">🧩</div>
        <div class="font-medium">技能管理</div>
        <div class="text-xs text-gray-500 mt-1">Skills 包</div>
      </router-link>
      <router-link to="/orchestrator" class="bg-white rounded-xl shadow p-5 hover:shadow-md transition text-center cursor-pointer">
        <div class="text-4xl mb-2">🔗</div>
        <div class="font-medium">Agent 编排</div>
        <div class="text-xs text-gray-500 mt-1">可视化流程</div>
      </router-link>
      <router-link to="/knowledge" class="bg-white rounded-xl shadow p-5 hover:shadow-md transition text-center cursor-pointer">
        <div class="text-4xl mb-2">🧠</div>
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
import { getAgents, getFlows, getSkills, getKnowledge, getTasks } from '../api'

const onlineAgents = ref(0)
const activeFlows = ref(0)
const skillCount = ref(0)
const pendingKnowledge = ref(0)
const recentTasks = ref([])

onMounted(async () => {
  try {
    const [agents, flows, skills, knowledge, tasks] = await Promise.all([
      getAgents().catch(() => ({ data: { online_count: 0 } })),
      getFlows().catch(() => ({ data: [] })),
      getSkills().catch(() => ({ data: [] })),
      getKnowledge({ status: 'pending' }).catch(() => ({ data: [] })),
      getTasks().catch(() => ({ data: [] }))
    ])
    onlineAgents.value = agents.data?.online_count || 0
    activeFlows.value = flows.data?.filter(f => f.status === 'active').length || 0
    skillCount.value = skills.data?.filter(s => s.enabled).length || 0
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
