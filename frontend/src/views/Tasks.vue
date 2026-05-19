<template>
  <div class="space-y-4">
    <div class="flex justify-between items-center">
      <h1 class="text-2xl font-bold flex items-center gap-2"><svg class="w-7 h-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg> 任务编排</h1>
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
            <svg class="w-3.5 h-3.5 inline" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="5" r="3"></circle><circle cx="6" cy="12" r="3"></circle><circle cx="18" cy="19" r="3"></circle><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line></svg> 引用流程 #{{ row.flow_id }}
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
            <div class="font-medium flex items-center gap-2"><svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="5" r="3"></circle><circle cx="6" cy="12" r="3"></circle><circle cx="18" cy="19" r="3"></circle><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line></svg> 引用编排流程</div>
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
    <el-dialog v-model="showSteps" :title="currentTaskDetail?.source === 'flow' ? `任务详情 (流程执行)` : '任务执行详情'" width="70%">
      <div class="mb-4">
        <el-descriptions :column="3" border v-if="currentTaskDetail">
          <el-descriptions-item label="任务 ID">{{ currentTaskDetail.id }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusTag(currentTaskDetail?.status)" size="small">{{ currentTaskDetail?.status }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="来源">
            <el-tag :type="currentTaskDetail?.source === 'flow' ? 'warning' : 'success'" size="small">
              {{ currentTaskDetail?.source === 'flow' ? '编排' : '自定义' }}
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
            <span class="text-2xl" v-html="stepTypeIcon(step.node_type)"></span>
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
  const icons = {
    agent: '<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="10" rx="2"></rect><circle cx="12" cy="5" r="2"></circle><path d="M12 7v4"></path></svg>',
    condition: '<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>',
    merge: '<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="5" r="3"></circle><circle cx="6" cy="12" r="3"></circle><circle cx="18" cy="19" r="3"></circle><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line></svg>',
    trigger: '<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>',
    webhook: '<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path></svg>',
    code: '<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg>'
  }
  return icons[t] || '<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path></svg>'
}
const stepStatusBadge = (s) => {
  return { running: 'bg-blue-100 text-blue-700', completed: 'bg-green-100 text-green-700', failed: 'bg-red-100 text-red-700', skipped: 'bg-gray-200 text-gray-600' }[s] || 'bg-gray-100'
}
const stepStatusColor = (s) => {
  return { running: 'border-blue-300 bg-blue-50', completed: 'border-green-300 bg-green-50', failed: 'border-red-300 bg-red-50', skipped: 'border-gray-200 bg-gray-50' }[s] || ''
}
</script>
