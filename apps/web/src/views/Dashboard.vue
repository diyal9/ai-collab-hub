<template>
  <div>
    <h1 class="text-2xl font-bold mb-6">协作平台概览</h1>

    <el-alert
      v-if="loadError"
      type="error"
      :title="loadError"
      show-icon
      class="mb-6"
      :closable="false"
    />

    <!-- 统计卡片 -->
    <div class="grid grid-cols-2 md:grid-cols-3 gap-4 mb-8">
      <template v-if="loading">
        <div v-for="n in 3" :key="n" class="bg-white rounded-xl shadow p-5">
          <el-skeleton animated>
            <template #template>
              <el-skeleton-item variant="h1" style="width: 40%; margin: 0 auto 8px" />
              <el-skeleton-item variant="text" style="width: 60%; margin: 0 auto" />
            </template>
          </el-skeleton>
        </div>
      </template>
      <template v-else>
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
      </template>
    </div>

    <!-- 快速入口 -->
    <div class="grid grid-cols-2 md:grid-cols-3 gap-4 mb-8">
      <router-link
        v-for="entry in quickEntries"
        :key="entry.to"
        :to="entry.to"
        class="bg-white rounded-xl shadow p-5 hover:shadow-md transition text-center cursor-pointer flex flex-col items-center justify-center"
      >
        <component :is="entry.icon" class="w-12 h-12 mb-2 text-gray-600" />
        <div class="font-medium">{{ entry.title }}</div>
        <div class="text-xs text-gray-500 mt-1">{{ entry.desc }}</div>
      </router-link>
    </div>

    <!-- 最近任务 -->
    <div class="bg-white rounded-xl shadow p-5">
      <h2 class="font-semibold mb-4">最近任务</h2>
      <div v-if="loading" class="space-y-3">
        <el-skeleton v-for="n in 3" :key="n" animated />
      </div>
      <div v-else-if="recentTasks.length === 0" class="text-gray-400 text-sm py-4 text-center">
        暂无任务
      </div>
      <div
        v-else
        v-for="t in recentTasks"
        :key="t.id"
        class="flex justify-between items-center py-2 border-b last:border-0"
      >
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
import { ref, onMounted, h } from 'vue'
import { getAgents, getFlows, getKnowledge, getTasks } from '../api'
import { showApiError } from '../utils/notify'

const loading = ref(true)
const loadError = ref('')
const onlineAgents = ref(0)
const activeFlows = ref(0)
const pendingKnowledge = ref(0)
const recentTasks = ref([])

const IconTerminal = {
  render: () =>
    h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '1.5' }, [
      h('rect', { x: '2', y: '3', width: '20', height: '14', rx: '2' }),
      h('line', { x1: '8', y1: '21', x2: '16', y2: '21' }),
      h('line', { x1: '12', y1: '17', x2: '12', y2: '21' }),
    ]),
}

const IconOrchestrator = {
  render: () =>
    h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '1.5' }, [
      h('circle', { cx: '18', cy: '5', r: '3' }),
      h('circle', { cx: '6', cy: '12', r: '3' }),
      h('circle', { cx: '18', cy: '19', r: '3' }),
      h('line', { x1: '8.59', y1: '13.51', x2: '15.42', y2: '17.49' }),
      h('line', { x1: '15.41', y1: '6.51', x2: '8.59', y2: '10.49' }),
    ]),
}

const IconAgent = {
  render: () =>
    h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '1.5' }, [
      h('rect', { x: '3', y: '11', width: '18', height: '10', rx: '2' }),
      h('circle', { cx: '12', cy: '5', r: '2' }),
      h('path', { d: 'M12 7v4' }),
    ]),
}

const IconKnowledge = {
  render: () =>
    h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '1.5' }, [
      h('path', { d: 'M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z' }),
      h('path', { d: 'M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z' }),
    ]),
}

const quickEntries = [
  { to: '/terminals', title: 'Agent 终端', desc: '管理远程连接', icon: IconTerminal },
  { to: '/orchestrator', title: 'Agent 编排', desc: '可视化流程', icon: IconOrchestrator },
  { to: '/agents', title: 'Agent 管理', desc: '实例与任务', icon: IconAgent },
  { to: '/knowledge', title: '知识中枢', desc: '沉淀业务', icon: IconKnowledge },
]

onMounted(async () => {
  loading.value = true
  loadError.value = ''
  const errors = []

  try {
    const results = await Promise.allSettled([
      getAgents(),
      getFlows(),
      getKnowledge({ status: 'pending' }),
      getTasks(),
    ])

    if (results[0].status === 'fulfilled') {
      onlineAgents.value = results[0].value.data?.online_count || 0
    } else {
      errors.push('Agent 数据')
    }

    if (results[1].status === 'fulfilled') {
      const flows = results[1].value.data || []
      activeFlows.value = flows.filter((f) => f.status === 'active').length
    } else {
      errors.push('流程数据')
    }

    if (results[2].status === 'fulfilled') {
      pendingKnowledge.value = results[2].value.data?.length || 0
    } else {
      errors.push('知识数据')
    }

    if (results[3].status === 'fulfilled') {
      recentTasks.value = (results[3].value.data || []).slice(0, 5)
    } else {
      errors.push('任务数据')
    }

    if (errors.length === 4) {
      loadError.value = '概览数据加载失败，请检查网络或稍后重试'
      showApiError(results[0].reason, loadError.value)
    } else if (errors.length > 0) {
      loadError.value = `部分数据未能加载：${errors.join('、')}`
    }
  } finally {
    loading.value = false
  }
})

const taskStatusClass = (s) =>
  ({
    pending: 'bg-gray-100 text-gray-600',
    running: 'bg-blue-100 text-blue-600',
    completed: 'bg-green-100 text-green-600',
    failed: 'bg-red-100 text-red-600',
  })[s] || 'bg-gray-100 text-gray-600'
</script>
