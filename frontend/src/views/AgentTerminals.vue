<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold flex items-center gap-2"><svg class="w-7 h-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect><line x1="8" y1="21" x2="16" y2="21"></line><line x1="12" y1="17" x2="12" y2="21"></line></svg> 终端管理</h1>
      <button @click="showAdd = true" class="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700">
        + 添加终端
      </button>
    </div>

    <!-- 终端列表 -->
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <div v-for="t in terminals" :key="t.id" class="bg-white rounded-xl shadow p-5 border border-gray-100">
        <div class="flex items-center justify-between mb-1">
          <h3 class="font-semibold text-lg">{{ t.name }}</h3>
          <span :class="statusClass(t.status)" class="px-2 py-1 rounded-full text-xs font-medium">
            {{ t.status }}
          </span>
        </div>
        <p class="text-sm text-gray-500 mb-4">{{ t.endpoint }}</p>

        <div class="space-y-2 text-sm text-gray-600 border-t border-b py-3 mb-4">
          <div class="flex justify-between">
            <span class="text-gray-400">最后连接</span>
            <span>{{ formatTime(t.last_connect) }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-400">创建时间</span>
            <span>{{ formatTime(t.created_at) }}</span>
          </div>
        </div>

        <div class="flex gap-2">
          <button @click="connectTerminal(t.id)" :disabled="t.status === 'connected'"
            class="flex-1 text-sm py-2 rounded-lg bg-green-50 text-green-600 hover:bg-green-100 disabled:opacity-50 flex items-center justify-center gap-1.5">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14"></path><path d="M12 5l7 7-7 7"></path></svg>
            {{ t.status === 'connected' ? '已连接' : '连接' }}
          </button>
          <button @click="editTerminal(t)" class="px-3 py-2 text-sm rounded-lg bg-gray-50 hover:bg-gray-100 flex items-center justify-center">
            <svg class="w-4 h-4 text-gray-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
          </button>
          <button @click="deleteTerminal(t.id)" class="px-3 py-2 text-sm rounded-lg bg-red-50 text-red-600 hover:bg-red-100 flex items-center justify-center">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
          </button>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="terminals.length === 0" class="text-center py-16 text-gray-400">
      <svg class="w-20 h-20 mx-auto mb-4 text-gray-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect><line x1="8" y1="21" x2="16" y2="21"></line><line x1="12" y1="17" x2="12" y2="21"></line></svg>
      <p>暂无终端，点击右上角添加</p>
    </div>

    <!-- 添加/编辑弹窗 -->
    <div v-if="showAdd" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md">
        <h2 class="text-xl font-bold mb-4">{{ editing ? '编辑终端' : '添加终端' }}</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-1">终端名称</label>
            <input v-model="form.name" class="w-full border rounded-lg px-3 py-2" placeholder="如: Agent-001" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">WebSocket 地址</label>
            <input v-model="form.endpoint" class="w-full border rounded-lg px-3 py-2" placeholder="wss://host:port/ws" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">认证 Token</label>
            <input v-model="form.token" class="w-full border rounded-lg px-3 py-2" placeholder="Bearer token" />
          </div>
        </div>
        <div class="flex gap-3 mt-6">
          <button @click="showAdd = false" class="flex-1 py-2 rounded-lg border">取消</button>
          <button @click="saveTerminal" class="flex-1 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getTerminals, createTerminal, updateTerminal, deleteTerminal as delTerminal, connectTerminal as connTerminal } from '../api'

const terminals = ref([])
const showAdd = ref(false)
const editing = ref(null)
const form = ref({ name: '', endpoint: '', token: '' })

const load = async () => {
  const { data } = await getTerminals()
  terminals.value = data
}

const saveTerminal = async () => {
  if (editing.value) {
    await updateTerminal(editing.value.id, form.value)
  } else {
    await createTerminal(form.value)
  }
  showAdd.value = false
  form.value = { name: '', endpoint: '', token: '' }
  editing.value = null
  load()
}

const editTerminal = (t) => {
  editing.value = t
  form.value = { name: t.name, endpoint: t.endpoint, token: t.token }
  showAdd.value = true
}

const deleteTerminal = async (id) => {
  if (!confirm('确定删除此终端？')) return
  await delTerminal(id)
  load()
}

const connectTerminal = async (id) => {
  await connTerminal(id)
  load()
}

const statusClass = (s) => {
  return {
    connected: 'bg-green-100 text-green-700',
    disconnected: 'bg-gray-100 text-gray-600',
    error: 'bg-red-100 text-red-700',
  }[s] || 'bg-gray-100 text-gray-600'
}

const formatTime = (t) => {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

onMounted(load)
</script>
