<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold flex items-center gap-2"><svg class="w-7 h-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path></svg> 知识中枢</h1>
      <button @click="showAdd = true" class="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700">
        + 录入知识
      </button>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-4 gap-4 mb-6">
      <div class="bg-white rounded-xl shadow p-4 text-center">
        <div class="text-3xl font-bold text-blue-600">{{ stats.total }}</div>
        <div class="text-sm text-gray-500">总知识</div>
      </div>
      <div class="bg-white rounded-xl shadow p-4 text-center">
        <div class="text-3xl font-bold text-yellow-600">{{ stats.pending }}</div>
        <div class="text-sm text-gray-500">待审核</div>
      </div>
      <div class="bg-white rounded-xl shadow p-4 text-center">
        <div class="text-3xl font-bold text-green-600">{{ stats.approved }}</div>
        <div class="text-sm text-gray-500">已通过</div>
      </div>
      <div class="bg-white rounded-xl shadow p-4 text-center">
        <div class="text-3xl font-bold text-red-600">{{ stats.rejected }}</div>
        <div class="text-sm text-gray-500">已驳回</div>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="flex gap-3 mb-4">
      <select v-model="filter.status" @change="load" class="border rounded-lg px-3 py-2 text-sm">
        <option value="">全部状态</option>
        <option value="pending">待审核</option>
        <option value="approved">已通过</option>
        <option value="rejected">已驳回</option>
        <option value="archived">已归档</option>
      </select>
      <select v-model="filter.category" @change="load" class="border rounded-lg px-3 py-2 text-sm">
        <option value="">全部分类</option>
        <option value="concept">概念</option>
        <option value="procedure">流程</option>
        <option value="fact">事实</option>
        <option value="decision">决策</option>
      </select>
      <input v-model="filter.tag" @keyup.enter="load" placeholder="搜索标签..." class="border rounded-lg px-3 py-2 text-sm" />
      <button @click="load" class="bg-gray-100 px-4 py-2 rounded-lg text-sm hover:bg-gray-200">搜索</button>
    </div>

    <!-- 知识列表 -->
    <div class="space-y-3">
      <div v-for="e in entries" :key="e.id" class="bg-white rounded-xl shadow p-5 border border-gray-100">
        <div class="flex items-start justify-between">
          <div class="flex-1">
            <div class="flex items-center gap-2">
              <h3 class="font-semibold text-lg">{{ e.title }}</h3>
              <span :class="statusBadge(e.status)" class="px-2 py-0.5 rounded text-xs">{{ e.status }}</span>
              <span class="px-2 py-0.5 bg-blue-50 text-blue-600 rounded text-xs">{{ e.type }}</span>
            </div>
            <p class="mt-2 text-sm text-gray-600 line-clamp-2">{{ e.summary || e.content?.substring(0, 100) }}</p>
            <div class="mt-2 flex gap-2 flex-wrap">
              <span v-for="tag in (e.tags || '').split(',').filter(Boolean)" :key="tag"
                class="px-2 py-0.5 bg-gray-100 text-gray-600 rounded text-xs">#{{ tag.trim() }}</span>
            </div>
          </div>
          <div class="text-xs text-gray-400 text-right ml-4">
            <div>来源: {{ e.source }}</div>
            <div>置信度: {{ (e.confidence * 100).toFixed(0) }}%</div>
            <div>{{ formatTime(e.updated_at) }}</div>
          </div>
        </div>

        <!-- 审核操作 -->
        <div v-if="e.status === 'pending'" class="mt-4 flex gap-2 pt-3 border-t">
          <button @click="review(e.id, 'approve')" class="px-4 py-1.5 rounded-lg bg-green-50 text-green-600 text-sm hover:bg-green-100">
            ✓ 通过
          </button>
          <button @click="review(e.id, 'reject')" class="px-4 py-1.5 rounded-lg bg-red-50 text-red-600 text-sm hover:bg-red-100">
            ✗ 驳回
          </button>
          <button @click="viewEntry(e)" class="px-4 py-1.5 rounded-lg bg-gray-50 text-gray-600 text-sm hover:bg-gray-100">
            📖 查看
          </button>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="entries.length === 0" class="text-center py-16 text-gray-400">
      <svg class="w-20 h-20 mx-auto mb-4 text-gray-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path></svg>
      <p>暂无知识条目</p>
    </div>

    <!-- 添加知识弹窗 -->
    <div v-if="showAdd" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-lg max-h-[80vh] overflow-y-auto">
        <h2 class="text-xl font-bold mb-4">录入知识</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-1">标题</label>
            <input v-model="form.title" class="w-full border rounded-lg px-3 py-2" />
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-1">类型</label>
              <select v-model="form.type" class="w-full border rounded-lg px-3 py-2">
                <option value="concept">概念</option>
                <option value="procedure">流程</option>
                <option value="fact">事实</option>
                <option value="decision">决策</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">分类</label>
              <input v-model="form.category" class="w-full border rounded-lg px-3 py-2" placeholder="如: AI, 架构" />
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">内容 (Markdown)</label>
            <textarea v-model="form.content" class="w-full border rounded-lg px-3 py-2 font-mono text-sm" rows="6"></textarea>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">标签 (逗号分隔)</label>
            <input v-model="form.tags" class="w-full border rounded-lg px-3 py-2" placeholder="tag1, tag2" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">来源</label>
            <select v-model="form.source" class="w-full border rounded-lg px-3 py-2">
              <option value="manual">手动录入</option>
              <option value="chat">对话提取</option>
              <option value="upload">文件导入</option>
              <option value="agent">Agent 生成</option>
            </select>
          </div>
        </div>
        <div class="flex gap-3 mt-6">
          <button @click="showAdd = false" class="flex-1 py-2 rounded-lg border">取消</button>
          <button @click="saveEntry" class="flex-1 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700">保存</button>
        </div>
      </div>
    </div>

    <!-- 知识详情弹窗 -->
    <div v-if="showDetail" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-2xl max-h-[80vh] overflow-y-auto">
        <div class="flex justify-between items-start">
          <h2 class="text-xl font-bold">{{ currentEntry?.title }}</h2>
          <span :class="statusBadge(currentEntry?.status)" class="px-2 py-0.5 rounded text-xs">{{ currentEntry?.status }}</span>
        </div>
        <div class="mt-2 text-sm text-gray-500">
          <span>类型: {{ currentEntry?.type }}</span> ·
          <span>分类: {{ currentEntry?.category }}</span> ·
          <span>来源: {{ currentEntry?.source }}</span>
        </div>
        <div v-if="currentEntry?.tags" class="mt-2 flex gap-1 flex-wrap">
          <span v-for="tag in currentEntry.tags.split(',').filter(Boolean)" :key="tag"
            class="px-2 py-0.5 bg-gray-100 text-gray-600 rounded text-xs">#{{ tag.trim() }}</span>
        </div>
        <div class="mt-4 p-4 bg-gray-50 rounded-lg whitespace-pre-wrap text-sm font-mono">
          {{ currentEntry?.content }}
        </div>

        <!-- 审核日志 -->
        <div v-if="auditLogs.length > 0" class="mt-4">
          <h3 class="font-medium text-sm text-gray-500 mb-2">审核日志</h3>
          <div v-for="log in auditLogs" :key="log.id" class="text-xs text-gray-500 py-1 border-b">
            {{ log.action }} · {{ formatTime(log.created_at) }}
            <span v-if="log.comment">— {{ log.comment }}</span>
          </div>
        </div>

        <div class="mt-6 flex justify-end gap-2">
          <button @click="showDetail = false" class="px-4 py-2 rounded-lg border">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { getKnowledge, createKnowledge, reviewKnowledge as reviewApi, getKnowledgeStats, getAuditLog } from '../api'

const entries = ref([])
const stats = reactive({ total: 0, pending: 0, approved: 0, rejected: 0 })
const filter = reactive({ status: '', category: '', tag: '' })
const showAdd = ref(false)
const showDetail = ref(false)
const currentEntry = ref(null)
const auditLogs = ref([])
const form = ref({ title: '', content: '', summary: '', type: 'concept', category: '', tags: '', source: 'manual', confidence: 0.8 })

const load = async () => {
  const [entriesRes, statsRes] = await Promise.all([
    getKnowledge(filter),
    getKnowledgeStats()
  ])
  entries.value = entriesRes.data
  Object.assign(stats, statsRes.data)
}

const saveEntry = async () => {
  await createKnowledge(form.value)
  showAdd.value = false
  form.value = { title: '', content: '', summary: '', type: 'concept', category: '', tags: '', source: 'manual', confidence: 0.8 }
  load()
}

const review = async (id, action) => {
  const comment = action === 'reject' ? prompt('驳回原因:') : ''
  await reviewApi(id, action, comment || '')
  load()
}

const viewEntry = async (e) => {
  currentEntry.value = e
  const { data } = await getAuditLog(e.id)
  auditLogs.value = data
  showDetail.value = true
}

const statusBadge = (s) => {
  return {
    pending: 'bg-yellow-100 text-yellow-700',
    approved: 'bg-green-100 text-green-700',
    rejected: 'bg-red-100 text-red-700',
    archived: 'bg-gray-100 text-gray-600'
  }[s] || 'bg-gray-100 text-gray-600'
}

const formatTime = (t) => t ? new Date(t).toLocaleString('zh-CN') : '-'

onMounted(load)
</script>
