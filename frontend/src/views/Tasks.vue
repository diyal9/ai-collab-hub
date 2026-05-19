<template>
  <div class="space-y-4">
    <div class="flex justify-between items-center">
      <h1 class="text-2xl font-bold">🚀 任务管理</h1>
      <button @click="openCreateDialog" class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 transition">
        + 新建任务
      </button>
    </div>

    <!-- 任务列表 -->
    <el-table :data="tasks" class="w-full" style="width: 100%">
      <el-table-column prop="title" label="任务名称" min-width="200">
        <template #default="{ row }">
          <div class="font-medium">{{ row.title }}</div>
          <div v-if="row.source === 'flow'" class="text-xs text-blue-500 mt-1">
            🧩 引用流程 #{{ row.flow_id }}
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="source" label="来源" width="100">
        <template #default="{ row }">
          <el-tag :type="row.source === 'flow' ? 'warning' : 'success'" size="small">
            {{ row.source === 'flow' ? '编排' : '自定义' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="statusTag(row.status)" size="small">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button size="small" @click="viewSteps(row.id)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 新建任务弹窗 -->
    <el-dialog v-model="showCreate" title="新建任务" width="600px">
      <el-form label-position="top">
        <el-form-item label="标题">
          <el-input v-model="newTask.title" placeholder="例如：5.21 版本资源构建" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="newTask.desc" type="textarea" rows="2" placeholder="任务描述（可选）" />
        </el-form-item>

        <el-divider>执行方式</el-divider>
        
        <div class="flex gap-4 mb-4">
          <div @click="newTask.source = 'manual'" 
               :class="['flex-1 p-4 border-2 rounded-lg cursor-pointer transition', newTask.source === 'manual' ? 'border-blue-500 bg-blue-50' : 'border-gray-200 hover:border-gray-300']">
            <div class="font-medium">🛠️ 自定义步骤</div>
            <div class="text-xs text-gray-500 mt-1">手动输入命令或 Prompt，适合简单/临时任务</div>
          </div>
          <div @click="newTask.source = 'flow'" 
               :class="['flex-1 p-4 border-2 rounded-lg cursor-pointer transition', newTask.source === 'flow' ? 'border-blue-500 bg-blue-50' : 'border-gray-200 hover:border-gray-300']">
            <div class="font-medium">🧩 引用编排流程</div>
            <div class="text-xs text-gray-500 mt-1">复用已有的可视化流水线，一键执行复杂管线</div>
          </div>
        </div>

        <!-- 自定义步骤表单 -->
        <div v-if="newTask.source === 'manual'">
          <el-form-item label="步骤 (每行一条)">
            <el-input v-model="newTask.steps" type="textarea" rows="4" placeholder="例如：&#10;1. 清理旧缓存&#10;2. cmake 编译&#10;3. 运行测试" />
          </el-form-item>
        </div>

        <!-- 流程选择表单 -->
        <div v-else-if="newTask.source === 'flow'">
          <el-form-item label="选择流程">
            <el-select v-model="newTask.flow_id" placeholder="请选择要执行的流程" class="w-full" @change="onFlowSelect">
              <el-option v-for="f in flows" :key="f.id" :label="f.name" :value="f.id">
                <div class="flex justify-between items-center">
                  <span>{{ f.name }}</span>
                  <span class="text-xs text-gray-400">状态: {{ f.status }}</span>
                </div>
              </el-option>
            </el-select>
          </el-form-item>
          <div v-if="selectedFlowPreview" class="bg-gray-50 p-3 rounded border text-sm text-gray-600 mt-2">
            <div class="font-medium mb-1 text-gray-500">👇 流程预览</div>
            <div class="flex gap-2 flex-wrap">
              <span v-for="(n, i) in selectedFlowPreview" :key="i" class="bg-white px-2 py-1 rounded border">
                {{ n.name }}
                <span v-if="i < selectedFlowPreview.length - 1" class="ml-1 text-gray-400">→</span>
              </span>
            </div>
          </div>
        </div>
      </el-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="showCreate = false">取消</el-button>
          <el-button type="primary" @click="createTask" :disabled="!isValid">创建并执行</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 任务详情弹窗 -->
    <el-dialog v-model="showSteps" :title="currentTaskDetail.source === 'flow' ? `任务详情 (流程执行)` : '任务执行详情'" width="70%">
      <div class="mb-4">
        <el-descriptions :column="3" border v-if="currentTaskDetail">
          <el-descriptions-item label="任务 ID">{{ currentTaskDetail.id }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusTag(currentTaskDetail.status)" size="small">{{ currentTaskDetail.status }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="来源">
            <el-tag :type="currentTaskDetail.source === 'flow' ? 'warning' : 'success'" size="small">
              {{ currentTaskDetail.source === 'flow' ? '编排' : '自定义' }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>
      </div>

      <!-- 流程执行详情 UI (复用编排界面) -->
      <div v-if="currentTaskDetail.source === 'flow'">
        <div v-if="flowSteps.length > 0" class="space-y-2 overflow-y-auto max-h-[60vh] pr-2">
          <div v-for="step in flowSteps" :key="step.id"
            class="flex items-center gap-3 p-3 rounded border bg-white transition hover:shadow-sm"
            :class="stepStatusColor(step.status)">
            <span class="text-2xl">{{ stepTypeIcon(step.node_type) }}</span>
            <div class="flex-1 min-w-0">
              <div class="font-medium text-sm truncate">{{ step.node_name }} <span class="text-xs text-gray-400 ml-1">({{ step.node_type }})</span></div>
              <div v-if="step.output" class="text-xs text-gray-600 mt-1 font-mono bg-gray-50 p-2 rounded max-h-24 overflow-y-auto whitespace-pre-wrap border border-gray-100">{{ step.output }}</div>
              <div v-if="step.error" class="text-xs text-red-500 mt-1 bg-red-50 p-1 rounded">❌ {{ step.error }}</div>
            </div>
            <span class="text-xs px-2 py-1 rounded font-medium whitespace-nowrap" :class="stepStatusBadge(step.status)">{{ step.status }}</span>
          </div>
        </div>
        <div v-else class="text-center text-gray-400 py-8">
          <div v-if="currentTaskDetail.status === 'running'" class="flex items-center justify-center gap-2">
            <el-icon class="is-loading"><Loading /></el-icon> 流程执行中...
          </div>
          <div v-else>暂无执行步骤</div>
        </div>
      </div>

      <!-- 自定义步骤 UI (旧版) -->
      <div v-else>
        <el-timeline v-if="steps.length">
          <el-timeline-item v-for="s in steps" :key="s.id" :type="s.status === 'success' ? 'success' : 'warning'">
            <p class="font-medium">{{ s.command }}</p>
            <small class="text-gray-500">{{ s.status }}</small>
            <p v-if="s.output" class="bg-gray-100 p-2 mt-1 rounded text-sm font-mono max-h-24 overflow-y-auto whitespace-pre-wrap">{{ s.output }}</p>
            <div v-if="s.input_req" class="mt-2 flex gap-2">
              <el-input v-model="s.user_reply" placeholder="回复 AI..." />
              <el-button size="small" @click="reply(s.id, s.task_id, s.user_reply)">发送</el-button>
            </div>
          </el-timeline-item>
        </el-timeline>
        <div v-else class="text-center text-gray-400 py-8">暂无步骤信息</div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Loading } from '@element-plus/icons-vue'
import { getTasks, createTask as apiCreate, getTaskSteps, replyTask, getFlows, getExecutionSteps } from '../api'

const tasks = ref([])
const steps = ref([])
const flowSteps = ref([]) // Store flow execution steps
const flows = ref([])
const currentTaskDetail = ref(null)
const pollingTimer = ref(null)

const showCreate = ref(false)
const showSteps = ref(false)

const newTask = ref({
  title: '',
  desc: '',
  source: 'manual', // manual | flow
  flow_id: null,
  steps: ''
})

const selectedFlowPreview = ref(null)

const isValid = computed(() => {
  if (newTask.value.source === 'manual') {
    return newTask.value.title && newTask.value.steps
  } else {
    return newTask.value.title && newTask.value.flow_id
  }
})

onMounted(async () => {
  tasks.value = (await getTasks()).data
  try {
    flows.value = (await getFlows()).data
  } catch {}
})

onUnmounted(() => {
  stopStatusPolling()
})

const openCreateDialog = () => {
  newTask.value = { title: '', desc: '', source: 'manual', flow_id: null, steps: '' }
  selectedFlowPreview.value = null
  showCreate.value = true
}

const onFlowSelect = async (flowId) => {
  selectedFlowPreview.value = null
  const flow = flows.value.find(f => f.id === flowId)
  if (flow) {
    try {
      // Fetch nodes to show preview
      const { data } = await import('../api').then(m => m.getFlowNodes(flowId))
      // Sort by position_y or just show all
      selectedFlowPreview.value = data.sort((a, b) => (a.position_y || 0) - (b.position_y || 0))
    } catch {}
  }
}

const statusTag = (s) => ({
  running: 'primary',
  completed: 'success',
  failed: 'danger',
  pending: 'info',
}[s] || '')

const createTask = async () => {
  let payload = {}
  if (newTask.value.source === 'manual') {
    payload = {
      title: newTask.value.title,
      desc: newTask.value.desc,
      type: 'serial',
      source: 'manual',
      steps: newTask.value.steps.split('\n').filter(s => s.trim())
    }
  } else {
    payload = {
      title: newTask.value.title,
      desc: newTask.value.desc,
      source: 'flow',
      flow_id: newTask.value.flow_id
    }
  }

  await apiCreate(payload)
  showCreate.value = false
  tasks.value = (await getTasks()).data
}

const viewSteps = async (id) => {
  showSteps.value = true
  steps.value = []
  flowSteps.value = []
  currentTaskDetail.value = tasks.value.find(t => t.id === id) || { id }
  
  stopStatusPolling()

  if (currentTaskDetail.value.source === 'flow') {
    // If it is a flow task, fetch flow execution details
    if (currentTaskDetail.value.flow_execution_id) {
      try {
        const res = await getExecutionSteps(currentTaskDetail.value.flow_execution_id)
        flowSteps.value = res.data || []
        
        // If running, start polling
        if (currentTaskDetail.value.status === 'running') {
          startStatusPolling(currentTaskDetail.value.flow_execution_id)
        }
      } catch {}
    } else {
      // Flow started but execution record not yet linked (race condition)
      // Try finding the latest execution for the flow
      try {
        const execRes = await import('../api').then(m => m.getFlowExecutions(currentTaskDetail.value.flow_id))
        const executions = execRes.data || []
        if (executions.length > 0) {
          const latest = executions[0]
          // Optimistic: assume latest is ours
          const res = await getExecutionSteps(latest.id)
          flowSteps.value = res.data || []
        }
      } catch {}
    }
  } else {
    // Custom task
    try {
      const res = await getTaskSteps(id)
      steps.value = res.data || []
    } catch {}
  }
}

const startStatusPolling = (execId) => {
  stopStatusPolling()
  pollingTimer.value = setInterval(async () => {
    try {
      const res = await getExecutionSteps(execId)
      flowSteps.value = res.data || []
      
      // Update task status in detail view
      const allDone = flowSteps.value.length > 0 && flowSteps.value.every(s => ['completed', 'failed', 'skipped'].includes(s.status))
      if (allDone) {
        // Refresh the task detail to reflect completion
        const task = currentTaskDetail.value
        if (task.status === 'running') {
          task.status = flowSteps.value.some(s => s.status === 'failed') ? 'failed' : 'completed'
        }
        stopStatusPolling()
      }
    } catch {}
  }, 1000)
}

const stopStatusPolling = () => {
  if (pollingTimer.value) { clearInterval(pollingTimer.value); pollingTimer.value = null }
}

const reply = async (sid, tid, txt) => {
  await replyTask(tid, sid, txt)
  viewSteps(tid)
}

// UI Helpers for Flow Steps
const stepTypeIcon = (t) => {
  return { agent: '🤖', condition: '🔀', merge: '🔗', trigger: '⚡', webhook: '🔌', code: '💻' }[t] || '📦'
}
const stepStatusBadge = (s) => {
  return { running: 'bg-blue-100 text-blue-700', completed: 'bg-green-100 text-green-700', failed: 'bg-red-100 text-red-700', skipped: 'bg-gray-200 text-gray-600' }[s] || 'bg-gray-100'
}
const stepStatusColor = (s) => {
  return { running: 'border-blue-300 bg-blue-50', completed: 'border-green-300 bg-green-50', failed: 'border-red-300 bg-red-50', skipped: 'border-gray-200 bg-gray-50' }[s] || ''
}
</script>
