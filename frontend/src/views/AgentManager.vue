<template>
  <div class="space-y-6">
    <h1 class="text-2xl font-bold text-gray-800">Agent 管理</h1>

    <!-- Tabs -->
    <div class="flex space-x-4 border-b">
      <button v-for="tab in tabs" :key="tab.key"
        @click="activeTab = tab.key"
        class="px-4 py-2 text-sm font-medium transition"
        :class="activeTab === tab.key
          ? 'border-b-2 border-blue-600 text-blue-600'
          : 'text-gray-500 hover:text-gray-700'">
        {{ tab.label }}
      </button>
    </div>

    <!-- Agent Instances -->
    <div v-if="activeTab === 'instances'" class="space-y-4">
      <div class="flex justify-between">
        <button @click="showAddAgent = true"
          class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm">
          + 添加 Agent
        </button>
      </div>

      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <div v-for="a in agents" :key="a.id"
          class="bg-white rounded-lg shadow p-4 border-l-4"
          :class="statusColor(a.status)">
          <div class="flex items-center justify-between mb-2">
            <h3 class="font-semibold text-gray-800">{{ a.name }}</h3>
            <span class="text-xs px-2 py-1 rounded-full"
              :class="statusBadge(a.status)">{{ a.status }}</span>
          </div>
          <div class="text-sm text-gray-500 space-y-1">
            <div>类型: <span class="capitalize">{{ a.type }}</span></div>
            <div>平台: {{ a.platform || '-' }}</div>
            <div>主机: {{ a.host || '-' }}</div>
            <!-- Capabilities -->
            <div v-if="a.capabilities" class="flex flex-wrap gap-1 mt-1">
              <span v-for="cap in parseCapabilities(a.capabilities)" :key="cap"
                class="text-xs px-1.5 py-0.5 rounded bg-blue-50 text-blue-600 border border-blue-200">
                {{ cap }}
              </span>
            </div>
            <div v-if="a.current_task_id" class="text-blue-600">
              当前任务: {{ a.current_task_id.slice(-8) }}
            </div>
            <div>最后心跳: {{ formatTime(a.last_heartbeat) }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Tasks -->
    <div v-if="activeTab === 'tasks'" class="space-y-4">
      <div class="flex space-x-2">
        <button @click="showAddTask = true"
          class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 text-sm">
          + 下发任务
        </button>
        <button @click="autoDispatch"
          class="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 text-sm">
          <svg class="w-4 h-4 inline" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg> 自动匹配分发
        </button>
      </div>

      <div class="bg-white rounded-lg shadow overflow-hidden">
        <table class="min-w-full divide-y">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-4 py-2 text-left text-xs text-gray-500">任务 ID</th>
              <th class="px-4 py-2 text-left text-xs text-gray-500">标题</th>
              <th class="px-4 py-2 text-left text-xs text-gray-500">技能要求</th>
              <th class="px-4 py-2 text-left text-xs text-gray-500">状态</th>
              <th class="px-4 py-2 text-left text-xs text-gray-500">时间</th>
              <th class="px-4 py-2 text-left text-xs text-gray-500">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-for="t in tasks" :key="t.id" class="hover:bg-gray-50">
              <td class="px-4 py-2 text-sm font-mono">{{ t.id.slice(-8) }}</td>
              <td class="px-4 py-2 text-sm">
                {{ t.title }}
                <span v-if="t.parent_task_id" class="text-xs text-gray-400 ml-1">↩ 子任务</span>
              </td>
              <td class="px-4 py-2">
                <span v-if="t.required_skills" class="text-xs px-1.5 py-0.5 rounded bg-gray-100 text-gray-600">
                  {{ parseSkillsShort(t.required_skills) }}
                </span>
                <span v-else class="text-xs text-gray-400">不限</span>
              </td>
              <td class="px-4 py-2">
                <span class="text-xs px-2 py-1 rounded-full"
                  :class="taskBadge(t.status)">{{ t.status }}</span>
              </td>
              <td class="px-4 py-2 text-sm text-gray-500">{{ formatTime(t.created_at) }}</td>
              <td class="px-4 py-2 space-x-2">
                <button @click="viewTaskLogs(t)"
                  class="text-blue-600 text-sm hover:underline">📄 日志</button>
                <button v-if="t.status === 'completed'"
                  @click="openSubtaskModal(t)"
                  class="text-purple-600 text-sm hover:underline">+ 子任务</button>
                <button v-if="t.status === 'running' || t.status === 'queued'"
                  @click="cancelTask(t.id)"
                  class="text-red-600 text-sm hover:underline">取消</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Approvals -->
    <div v-if="activeTab === 'approvals'" class="space-y-4">
      <div v-if="approvals.length === 0" class="text-gray-500 text-center py-8">
        暂无待审批项
      </div>
      <div v-for="a in approvals" :key="a.id"
        class="bg-white rounded-lg shadow p-4 border-l-4 border-yellow-400">
        <div class="flex items-start justify-between">
          <div>
            <h3 class="font-semibold">{{ a.question }}</h3>
            <p class="text-sm text-gray-500 mt-1">任务: {{ a.task_id.slice(-8) }}</p>
            <pre v-if="a.context" class="text-xs bg-gray-100 p-2 mt-2 rounded max-h-40 overflow-auto">{{ a.context }}</pre>
          </div>
          <div class="flex space-x-2">
            <button @click="replyApproval(a.id, true, '')"
              class="px-3 py-1 bg-green-600 text-white rounded text-sm hover:bg-green-700">
              批准
            </button>
            <button @click="showReplyModal = a"
              class="px-3 py-1 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">
              回复
            </button>
            <button @click="replyApproval(a.id, false, '')"
              class="px-3 py-1 bg-red-600 text-white rounded text-sm hover:bg-red-700">
              拒绝
            </button>
          </div>
        </div>
        <div v-if="a.suggested" class="mt-2 text-sm text-blue-600">
          建议: {{ a.suggested }}
        </div>
      </div>
    </div>

    <!-- Add Agent Modal -->
    <div v-if="showAddAgent" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg p-6 w-full max-w-md">
        <h3 class="text-lg font-semibold mb-4">添加 Agent</h3>
        <div v-if="!createdAgent" class="space-y-3">
          <input v-model="newAgent.name" placeholder="名称 (如 hermes-main)" class="w-full border rounded px-3 py-2 text-sm" />
          <select v-model="newAgent.type" class="w-full border rounded px-3 py-2 text-sm">
            <option value="hermes">Hermes Agent</option>
            <option value="codex">Codex CLI</option>
            <option value="cursor">Cursor</option>
          </select>
          <div class="mt-4 flex justify-end space-x-2">
            <button @click="showAddAgent = false" class="px-4 py-2 text-gray-600">取消</button>
            <button @click="addAgent" class="px-4 py-2 bg-blue-600 text-white rounded">创建</button>
          </div>
        </div>
        <div v-else class="space-y-3">
          <div class="bg-green-50 border border-green-200 rounded p-3 text-sm">
            <p class="font-semibold text-green-800">✅ Agent 创建成功</p>
            <p class="mt-1 text-green-700">Token: <code class="bg-green-100 px-1 rounded text-xs">{{ createdAgent.token }}</code></p>
          </div>
          <div class="bg-gray-50 p-3 rounded">
            <p class="text-xs font-semibold text-gray-600 mb-1">连接命令:</p>
            <code class="block text-xs bg-gray-900 text-green-400 p-2 rounded overflow-x-auto whitespace-pre">{{ createdAgent.connect_cmd }}</code>
          </div>
          <button @click="copyCmd" class="w-full px-4 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">
            📋 复制命令
          </button>
          <button @click="closeAgentModal" class="w-full px-4 py-2 text-gray-600 text-sm">关闭</button>
        </div>
      </div>
    </div>

    <!-- Add Task Modal -->
    <div v-if="showAddTask" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <h3 class="text-lg font-semibold mb-4">下发任务</h3>
        <div class="space-y-3">
          <select v-model="newTask.agentId" class="w-full border rounded px-3 py-2 text-sm">
            <option value="">自动匹配 (按能力路由)</option>
            <option v-for="a in onlineAgents" :key="a.id" :value="a.id">
              {{ a.name }} ({{ a.type }}, {{ a.status }})
            </option>
          </select>
          <input v-model="newTask.title" placeholder="任务标题" class="w-full border rounded px-3 py-2 text-sm" />
          <textarea v-model="newTask.prompt" placeholder="任务指令 (Prompt)" rows="4" class="w-full border rounded px-3 py-2 text-sm"></textarea>
          
          <!-- 技能选择器 -->
          <div>
            <label class="text-sm font-medium text-gray-600 mb-1 block">所需技能 (留空=不限制)</label>
            <div class="flex flex-wrap gap-2">
              <label v-for="skill in allSkills" :key="skill"
                class="flex items-center gap-1 text-xs px-2 py-1 rounded border cursor-pointer select-none"
                :class="newTask.skills.includes(skill) ? 'bg-blue-100 border-blue-400 text-blue-700' : 'bg-gray-50 border-gray-200 text-gray-500'">
                <input type="checkbox" :value="skill" v-model="newTask.skills" class="hidden" />
                {{ skill }}
              </label>
            </div>
          </div>

          <!-- 超时设置 -->
          <div class="flex items-center gap-2">
            <label class="text-sm text-gray-600">超时:</label>
            <input v-model.number="newTask.timeoutMinutes" type="number" min="1" max="120"
              class="border rounded px-2 py-1 text-sm w-20" />
            <span class="text-sm text-gray-500">分钟</span>
          </div>
        </div>
        <div class="flex justify-end space-x-2 mt-4">
          <button @click="showAddTask = false" class="px-4 py-2 text-gray-600">取消</button>
          <button @click="createTask" class="px-4 py-2 bg-green-600 text-white rounded">下发</button>
        </div>
      </div>
    </div>

    <!-- Subtask Modal -->
    <div v-if="showSubtask" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg p-6 w-full max-w-lg">
        <h3 class="text-lg font-semibold mb-2">创建子任务</h3>
        <p class="text-sm text-gray-500 mb-4">
          父任务: <span class="font-mono">{{ parentTask?.id?.slice(-8) }}</span> — {{ parentTask?.title }}
          <br><span class="text-xs text-gray-400">子任务将自动继承父任务的输出上下文</span>
        </p>
        <div class="space-y-3">
          <input v-model="subtask.title" placeholder="子任务标题" class="w-full border rounded px-3 py-2 text-sm" />
          <textarea v-model="subtask.prompt" placeholder="任务指令 (将附加父任务上下文)" rows="4" class="w-full border rounded px-3 py-2 text-sm"></textarea>
          <div class="flex items-center gap-2">
            <label class="text-sm text-gray-600">超时:</label>
            <input v-model.number="subtask.timeoutMinutes" type="number" min="1" max="120"
              class="border rounded px-2 py-1 text-sm w-20" />
            <span class="text-sm text-gray-500">分钟</span>
          </div>
        </div>
        <div class="flex justify-end space-x-2 mt-4">
          <button @click="showSubtask = false" class="px-4 py-2 text-gray-600">取消</button>
          <button @click="createSubtask" class="px-4 py-2 bg-purple-600 text-white rounded">创建子任务</button>
        </div>
      </div>
    </div>

    <!-- Task Logs Modal -->
    <div v-if="showLogs" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg p-6 w-full max-w-3xl max-h-[80vh] flex flex-col">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold">任务日志: {{ currentLogTask?.title }} <span class="text-sm text-gray-400 font-mono">#{{ currentLogTask?.id?.slice(-8) }}</span></h3>
          <button @click="showLogs = false" class="text-gray-500 hover:text-gray-700 text-2xl">&times;</button>
        </div>
        <div class="flex-1 bg-gray-900 text-green-400 p-4 rounded font-mono text-xs overflow-y-auto whitespace-pre-wrap" v-html="taskLogs"></div>
        <div class="mt-4 flex justify-end">
          <button @click="refreshLogs" class="px-3 py-1 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">🔄 刷新</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { api } from '../api/index'

const tabs = [
  { key: 'instances', label: 'Agent 实例' },
  { key: 'tasks', label: '任务' },
  { key: 'approvals', label: '审批' },
]
const activeTab = ref('instances')
const agents = ref([])
const tasks = ref([])
const approvals = ref([])
const showAddAgent = ref(false)
const showAddTask = ref(false)
const showSubtask = ref(false)
const showReplyModal = ref(null)
const showLogs = ref(false)
const currentLogTask = ref(null)
const taskLogs = ref('')
const createdAgent = ref(null)
const parentTask = ref(null)
const newAgent = ref({ name: '', type: 'hermes' })
const newTask = ref({ agentId: '', title: '', prompt: '', skills: [], timeoutMinutes: 30 })
const subtask = ref({ title: '', prompt: '', timeoutMinutes: 30 })

const allSkills = ['terminal', 'browser', 'file', 'web', 'code-execution', 'editor', 'mcp', 'git']

const onlineAgents = computed(() => agents.value.filter(a => a.status === 'online' || a.status === 'busy'))

const hubWsUrl = computed(() => {
  const loc = window.location
  return `${loc.protocol === 'https:' ? 'wss:' : 'ws:'}//${loc.host}/ws/agent`
})

const statusColor = (s) => ({
  online: 'border-green-500',
  offline: 'border-gray-300',
  busy: 'border-yellow-500',
  error: 'border-red-500',
}[s] || 'border-gray-300')

const statusBadge = (s) => ({
  online: 'bg-green-100 text-green-800',
  offline: 'bg-gray-100 text-gray-600',
  busy: 'bg-yellow-100 text-yellow-800',
  error: 'bg-red-100 text-red-800',
}[s] || 'bg-gray-100 text-gray-600')

const taskBadge = (s) => ({
  queued: 'bg-gray-100 text-gray-600',
  running: 'bg-blue-100 text-blue-800',
  waiting_input: 'bg-yellow-100 text-yellow-800',
  completed: 'bg-green-100 text-green-800',
  failed: 'bg-red-100 text-red-800',
  cancelled: 'bg-gray-200 text-gray-600',
}[s] || 'bg-gray-100')

const formatTime = (t) => {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

const parseCapabilities = (caps) => {
  try { return JSON.parse(caps) } catch { return [] }
}

const parseSkillsShort = (skills) => {
  try { return JSON.parse(skills).join(', ') } catch { return skills }
}

const loadAgents = async () => {
  const { data } = await api.get('/agent-instances')
  agents.value = data
}

const loadTasks = async () => {
  const { data } = await api.get('/agent-tasks')
  tasks.value = data
}

const loadApprovals = async () => {
  const { data } = await api.get('/approvals')
  approvals.value = data
}

const cancelTask = async (id) => {
  await api.post(`/agent-tasks/${id}/cancel`)
  loadTasks()
}

const autoDispatch = async () => {
  await api.post('/agent-tasks/auto-dispatch')
  loadTasks()
}

const replyApproval = async (id, approved, reply) => {
  await api.post(`/approvals/${id}/reply`, { approved, reply })
  loadApprovals()
}

const addAgent = async () => {
  try {
    const { data } = await api.post('/agent-instances', {
      name: newAgent.value.name,
      type: newAgent.value.type,
    })
    createdAgent.value = data
  } catch (e) {
    alert('创建失败: ' + (e.response?.data?.error || e.message))
  }
}

const copyCmd = () => {
  if (createdAgent.value?.connect_cmd) {
    navigator.clipboard.writeText(createdAgent.value.connect_cmd)
    alert('命令已复制')
  }
}

const closeAgentModal = () => {
  showAddAgent.value = false
  createdAgent.value = null
  newAgent.value = { name: '', type: 'hermes' }
  loadAgents()
}

const createTask = async () => {
  const skillsJson = newTask.value.skills.length > 0
    ? JSON.stringify(newTask.value.skills)
    : ''
  await api.post('/agent-tasks', {
    agent_id: newTask.value.agentId || undefined,
    title: newTask.value.title,
    prompt: newTask.value.prompt,
    priority: 1,
    required_skills: skillsJson,
    timeout_minutes: newTask.value.timeoutMinutes || 30,
  })
  showAddTask.value = false
  newTask.value = { agentId: '', title: '', prompt: '', skills: [], timeoutMinutes: 30 }
  loadTasks()
}

const openSubtaskModal = (task) => {
  parentTask.value = task
  subtask.value = { title: '', prompt: '', timeoutMinutes: 30 }
  showSubtask.value = true
}

const createSubtask = async () => {
  await api.post(`/agent-tasks/subtask/${parentTask.value.id}`, {
    title: subtask.value.title,
    prompt: subtask.value.prompt,
    timeout_minutes: subtask.value.timeoutMinutes || 30,
  })
  showSubtask.value = false
  parentTask.value = null
  loadTasks()
}

const viewTaskLogs = async (task) => {
  currentLogTask.value = task
  showLogs.value = true
  await refreshLogs()
}

const refreshLogs = async () => {
  if (!currentLogTask.value) return
  try {
    const { data } = await api.get(`/agent-tasks/${currentLogTask.value.id}/logs`)
    if (data.length === 0) {
      taskLogs.value = '<span class="text-gray-500">暂无日志输出</span>'
    } else {
      taskLogs.value = data.map(l => `<div class="mb-1"><span class="text-gray-500">[${formatTime(l.timestamp)}]</span> ${l.message || l.output || l.text}</div>`).join('')
    }
  } catch (e) {
    taskLogs.value = `<span class="text-red-400">加载失败: ${e.message}</span>`
  }
}

onMounted(() => {
  loadAgents()
  loadTasks()
  loadApprovals()
  setInterval(() => {
    loadAgents()
    loadApprovals()
  }, 5000)
})
</script>
