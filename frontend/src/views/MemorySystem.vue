<template>
  <div>
<h1 class="text-2xl font-bold flex items-center gap-2">
      <svg class="w-7 h-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <ellipse cx="12" cy="5" rx="9" ry="3"></ellipse>
        <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"></path>
        <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path>
      </svg>
      记忆系统
    </h1>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
      <div class="bg-white rounded-xl shadow p-5 text-center">
        <div class="text-3xl font-bold text-blue-600">{{ sessions.length }}</div>
        <div class="text-sm text-gray-500 mt-1">会话总数</div>
      </div>
      <div class="bg-white rounded-xl shadow p-5 text-center">
        <div class="text-3xl font-bold text-green-600">{{ activeSessions }}</div>
        <div class="text-sm text-gray-500 mt-1">活跃会话</div>
      </div>
      <div class="bg-white rounded-xl shadow p-5 text-center">
        <div class="text-3xl font-bold text-purple-600">{{ totalMemories }}</div>
        <div class="text-sm text-gray-500 mt-1">记忆条目</div>
      </div>
      <div class="bg-white rounded-xl shadow p-5 text-center">
        <div class="text-3xl font-bold text-yellow-600">{{ memorySize }}</div>
        <div class="text-sm text-gray-500 mt-1">占用空间</div>
      </div>
    </div>

    <!-- 会话列表 -->
    <div class="bg-white rounded-xl shadow border border-gray-100">
      <div class="p-4 border-b flex justify-between items-center">
        <h2 class="font-semibold">最近会话</h2>
        <div class="flex gap-2">
          <input v-model="search" placeholder="搜索会话..." class="border rounded px-3 py-1 text-sm" />
          <button @click="loadSessions" class="text-sm text-blue-600 hover:text-blue-700">刷新</button>
        </div>
      </div>
      <div class="divide-y divide-gray-100">
        <div v-for="s in filteredSessions" :key="s.id" class="p-4 hover:bg-gray-50 transition cursor-pointer" @click="viewSession(s)">
          <div class="flex justify-between items-start">
            <div>
              <h3 class="font-medium">{{ s.title || '未命名会话' }}</h3>
              <p class="text-sm text-gray-500 mt-1">{{ s.summary || '暂无摘要' }}</p>
            </div>
            <span class="text-xs text-gray-400">{{ formatTime(s.created_at) }}</span>
          </div>
          <div class="flex gap-2 mt-2">
            <span class="text-xs px-2 py-0.5 rounded bg-blue-50 text-blue-600">{{ s.message_count || 0 }} 条消息</span>
            <span v-if="s.model" class="text-xs px-2 py-0.5 rounded bg-gray-100 text-gray-600">{{ s.model }}</span>
          </div>
        </div>
      </div>
      <div v-if="filteredSessions.length === 0" class="text-center py-12 text-gray-400">
        <svg class="w-16 h-16 mx-auto mb-3 text-gray-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>
        </svg>
        <p>暂无记忆会话记录</p>
      </div>
    </div>

    <!-- 会话详情弹窗 -->
    <div v-if="showDetail" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showDetail = false">
      <div class="bg-white rounded-xl p-6 w-full max-w-2xl max-h-[80vh] flex flex-col">
        <div class="flex justify-between items-center mb-4 border-b pb-3">
          <h3 class="text-lg font-semibold">{{ currentSession?.title || '会话详情' }}</h3>
          <button @click="showDetail = false" class="text-gray-400 hover:text-gray-600 text-2xl">&times;</button>
        </div>
        <div class="flex-1 overflow-y-auto space-y-3">
          <div v-for="(msg, i) in sessionMessages" :key="i" class="flex" :class="msg.role === 'user' ? 'justify-end' : 'justify-start'">
            <div class="max-w-[80%] rounded-lg px-4 py-2 text-sm" :class="msg.role === 'user' ? 'bg-blue-500 text-white' : 'bg-gray-100 text-gray-800'">
              <div class="text-xs opacity-60 mb-1">{{ msg.role === 'user' ? '你' : 'AI' }}</div>
              <div class="whitespace-pre-wrap">{{ msg.content }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getMemorySessions } from '../api'

const sessions = ref([])
const search = ref('')
const showDetail = ref(false)
const currentSession = ref(null)
const sessionMessages = ref([])

const activeSessions = computed(() => sessions.value.filter(s => s.status === 'active').length)
const totalMemories = computed(() => sessions.value.reduce((sum, s) => sum + (s.message_count || 0), 0))
const memorySize = computed(() => {
  const bytes = sessions.value.reduce((sum, s) => sum + (s.content_length || 0), 0)
  if (bytes > 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  if (bytes > 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return bytes + ' B'
})

const filteredSessions = computed(() => {
  if (!search.value) return sessions.value
  const q = search.value.toLowerCase()
  return sessions.value.filter(s =>
    (s.title || '').toLowerCase().includes(q) ||
    (s.summary || '').toLowerCase().includes(q)
  )
})

const loadSessions = async () => {
  try {
    const { data } = await getMemorySessions()
    sessions.value = data || []
  } catch {
    sessions.value = []
  }
}

const viewSession = async (s) => {
  currentSession.value = s
  // Parse messages from graph/summary if available
  try {
    const msgs = typeof s.messages === 'string' ? JSON.parse(s.messages) : (s.messages || [])
    sessionMessages.value = msgs
  } catch {
    sessionMessages.value = []
  }
  showDetail.value = true
}

const formatTime = (t) => t ? new Date(t).toLocaleString('zh-CN') : '-'

onMounted(loadSessions)
</script>
