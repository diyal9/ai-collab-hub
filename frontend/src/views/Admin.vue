<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold flex items-center gap-2">
        <svg class="w-7 h-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="3"></circle>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
        </svg>
        系统设置
      </h1>
      <button @click="logout" class="flex items-center gap-2 px-4 py-2 rounded-lg bg-red-50 text-red-600 hover:bg-red-100 transition">
        <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
          <polyline points="16 17 21 12 16 7"></polyline>
          <line x1="21" y1="12" x2="9" y2="12"></line>
        </svg>
        退出登录
      </button>
    </div>

    <!-- Tab 切换 -->
    <div class="bg-white rounded-xl shadow border border-gray-100">
      <div class="border-b border-gray-100 px-6">
        <nav class="flex gap-0 -mb-px">
          <button v-for="tab in tabs" :key="tab.key"
            @click="activeTab = tab.key"
            class="px-5 py-3.5 text-sm font-medium border-b-2 transition-colors flex items-center gap-2"
            :class="activeTab === tab.key
              ? 'border-blue-500 text-blue-600'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'">
            <span v-html="tab.icon" class="w-5 h-5"></span>
            {{ tab.label }}
          </button>
        </nav>
      </div>

      <div class="p-6">
        <!-- ═══════ 用户管理 ═══════ -->
        <div v-if="activeTab === 'users'">
          <div class="flex justify-between items-center mb-5">
            <div>
              <h2 class="text-lg font-semibold">用户列表</h2>
              <p class="text-sm text-gray-400 mt-0.5">管理 Web 登录用户及角色权限</p>
            </div>
            <button @click="showUserDialog = true" class="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 flex items-center gap-2">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
              添加用户
            </button>
          </div>

          <el-table :data="users" class="w-full" style="width: 100%" stripe>
            <el-table-column prop="username" label="用户名" min-width="120">
              <template #default="{ row }">
                <div class="flex items-center gap-2">
                  <div class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium"
                    :class="row.role === 'admin' ? 'bg-blue-100 text-blue-600' : 'bg-gray-100 text-gray-600'">
                    {{ (row.username || '?')[0]?.toUpperCase() }}
                  </div>
                  <span class="font-medium">{{ row.username }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="role" label="角色" width="120">
              <template #default="{ row }">
                <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" size="small" effect="plain">
                  {{ row.role === 'admin' ? '管理员' : '普通用户' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="180">
              <template #default="{ row }">
                <span class="text-gray-500 text-sm">{{ formatTime(row.created_at) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="200">
              <template #default="{ row }">
                <div class="flex gap-2">
                  <button @click="editUser(row)" class="text-blue-600 hover:text-blue-800 text-sm flex items-center gap-1">
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
                    编辑
                  </button>
                  <el-popconfirm title="确定删除此用户？" @confirm="deleteUser(row.id)" width="200">
                    <template #reference>
                      <button class="text-red-500 hover:text-red-700 text-sm flex items-center gap-1">
                        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                        删除
                      </button>
                    </template>
                  </el-popconfirm>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- ═══════ API Key 管理 ═══════ -->
        <div v-if="activeTab === 'apikeys'">
          <div class="flex justify-between items-center mb-5">
            <div>
              <h2 class="text-lg font-semibold">API Key 管理</h2>
              <p class="text-sm text-gray-400 mt-0.5">用于 Agent 终端 / 第三方服务认证的 Token</p>
            </div>
            <button @click="generateApiKey" class="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 flex items-center gap-2">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
              生成新 Key
            </button>
          </div>

          <el-table :data="apiKeys" class="w-full" style="width: 100%" stripe>
            <el-table-column label="Key 名称" min-width="150">
              <template #default="{ row }">
                <div class="flex items-center gap-2">
                  <svg class="w-5 h-5 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"></path></svg>
                  <span class="font-medium">{{ row.name || '未命名' }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="Token" min-width="250">
              <template #default="{ row }">
                <div class="flex items-center gap-2">
                  <code class="bg-gray-100 px-2 py-1 rounded text-sm font-mono">{{ row.token_hidden || row.token }}</code>
                  <button @click="copyToken(row.token)" class="text-gray-400 hover:text-blue-500" title="复制">
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                  </button>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="owner" label="关联用户" width="120">
              <template #default="{ row }">
                <span class="text-gray-500 text-sm">{{ row.owner || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small" effect="plain">
                  {{ row.status === 'active' ? '启用' : '已禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="180">
              <template #default="{ row }">
                <span class="text-gray-500 text-sm">{{ formatTime(row.created_at) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="150">
              <template #default="{ row }">
                <div class="flex gap-2">
                  <button @click="toggleApiKey(row)" class="text-gray-500 hover:text-blue-500 text-sm">
                    {{ row.status === 'active' ? '禁用' : '启用' }}
                  </button>
                  <el-popconfirm title="确定吊销此 Key？" @confirm="revokeApiKey(row.id)" width="200">
                    <template #reference>
                      <button class="text-red-500 hover:text-red-700 text-sm">吊销</button>
                    </template>
                  </el-popconfirm>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- ═══════ 飞书接入 (预留) ═══════ -->
        <div v-if="activeTab === 'feishu'">
          <div class="max-w-2xl mx-auto">
            <!-- 状态卡片 -->
            <div class="bg-gradient-to-br from-cyan-50 to-blue-50 rounded-xl border border-cyan-100 p-6 mb-6">
              <div class="flex items-center gap-3 mb-3">
                <div class="w-10 h-10 rounded-lg bg-cyan-100 flex items-center justify-center">
                  <svg class="w-6 h-6 text-cyan-600" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
                  </svg>
                </div>
                <div>
                  <h3 class="font-semibold text-gray-800">飞书 (Feishu/Lark) 接入</h3>
                  <p class="text-sm text-gray-500">将 AI Collab Hub 接入飞书开放平台</p>
                </div>
              </div>
              <el-tag type="info" size="small" effect="plain">🚧 开发中</el-tag>
            </div>

            <!-- 配置表单 (灰色禁用态) -->
            <div class="space-y-4 opacity-60 pointer-events-none select-none">
              <div class="bg-white rounded-xl border border-gray-200 p-6">
                <h4 class="font-medium mb-4 flex items-center gap-2">
                  <svg class="w-5 h-5 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
                  应用凭证
                </h4>
                <div class="grid grid-cols-2 gap-4">
                  <div>
                    <label class="block text-sm text-gray-500 mb-1">App ID</label>
                    <input disabled class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm bg-gray-50" placeholder="cli_xxxxxxxxxx" />
                  </div>
                  <div>
                    <label class="block text-sm text-gray-500 mb-1">App Secret</label>
                    <input disabled class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm bg-gray-50" placeholder="••••••••••••" />
                  </div>
                </div>
              </div>

              <div class="bg-white rounded-xl border border-gray-200 p-6">
                <h4 class="font-medium mb-4 flex items-center gap-2">
                  <svg class="w-5 h-5 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"></path><circle cx="12" cy="10" r="3"></circle></svg>
                  事件回调
                </h4>
                <div class="grid grid-cols-2 gap-4">
                  <div>
                    <label class="block text-sm text-gray-500 mb-1">Request URL</label>
                    <input disabled class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm bg-gray-50 font-mono text-xs" placeholder="https://your-domain.com/api/feishu/callback" />
                  </div>
                  <div>
                    <label class="block text-sm text-gray-500 mb-1">Encrypt Key</label>
                    <input disabled class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm bg-gray-50" placeholder="自定义加密密钥" />
                  </div>
                </div>
              </div>

              <div class="bg-white rounded-xl border border-gray-200 p-6">
                <h4 class="font-medium mb-4 flex items-center gap-2">
                  <svg class="w-5 h-5 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect><line x1="8" y1="21" x2="16" y2="21"></line><line x1="12" y1="17" x2="12" y2="21"></line></svg>
                  功能开关
                </h4>
                <div class="space-y-3">
                  <div class="flex items-center justify-between py-2 border-b border-gray-100">
                    <span class="text-sm text-gray-600">飞书机器人消息</span>
                    <el-switch disabled />
                  </div>
                  <div class="flex items-center justify-between py-2 border-b border-gray-100">
                    <span class="text-sm text-gray-600">任务通知推送</span>
                    <el-switch disabled />
                  </div>
                  <div class="flex items-center justify-between py-2">
                    <span class="text-sm text-gray-600">审批流同步</span>
                    <el-switch disabled />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加/编辑用户弹窗 -->
    <el-dialog v-model="showUserDialog" :title="editingUser ? '编辑用户' : '添加用户'" width="450px" :close-on-click-modal="false">
      <el-form label-position="top" class="space-y-4">
        <el-form-item label="用户名">
          <el-input v-model="userForm.username" placeholder="输入用户名" :disabled="!!editingUser" />
        </el-form-item>
        <el-form-item v-if="!editingUser" label="初始密码">
          <el-input v-model="userForm.password" type="password" placeholder="设置初始密码" show-password />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="userForm.role" class="w-full">
            <el-option label="普通用户" value="user" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="closeUserDialog">取消</el-button>
          <el-button type="primary" @click="saveUser">{{ editingUser ? '保存' : '创建' }}</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getUsers, createUser } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()

// Tab 定义
const tabs = [
  { key: 'users', label: '用户管理', icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg>' },
  { key: 'apikeys', label: 'API Key', icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"></path></svg>' },
  { key: 'feishu', label: '飞书接入', icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"></path><circle cx="12" cy="10" r="3"></circle></svg>' },
]
const activeTab = ref('users')

// 用户管理
const users = ref([])
const showUserDialog = ref(false)
const editingUser = ref(null)
const userForm = ref({ username: '', password: '', role: 'user' })

const loadUsers = async () => {
  try {
    const { data } = await getUsers()
    users.value = data || []
  } catch { users.value = [] }
}

const editUser = (u) => {
  editingUser.value = u
  userForm.value = { username: u.username, password: '', role: u.role }
  showUserDialog.value = true
}

const closeUserDialog = () => {
  showUserDialog.value = false
  editingUser.value = null
  userForm.value = { username: '', password: '', role: 'user' }
}

const saveUser = async () => {
  if (!userForm.value.username) { ElMessage.warning('请输入用户名'); return }
  if (!editingUser.value && !userForm.value.password) { ElMessage.warning('请输入初始密码'); return }

  if (editingUser.value) {
    // 更新角色
    ElMessage.info('角色更新功能开发中')
  } else {
    await createUser(userForm.value.username, userForm.value.password)
    ElMessage.success('用户创建成功')
  }
  closeUserDialog()
  loadUsers()
}

const deleteUser = async (id) => {
  try {
    // TODO: call delete user API
    ElMessage.info('删除功能开发中')
  } catch { ElMessage.error('删除失败') }
}

// API Key 管理
const apiKeys = ref([])

const generateApiKey = async () => {
  const token = 'tk_' + Date.now() + '_' + Math.random().toString(36).substr(2, 8)
  const newKey = {
    id: Date.now(),
    name: `Key-${new Date().toLocaleDateString('zh-CN')}`,
    token,
    token_hidden: token.slice(0, 6) + '••••••••' + token.slice(-4),
    owner: '-',
    status: 'active',
    created_at: new Date().toISOString()
  }
  apiKeys.value.unshift(newKey)
  ElMessage.success('API Key 已生成: ' + token)
}

const copyToken = async (token) => {
  try {
    await navigator.clipboard.writeText(token)
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

const toggleApiKey = (row) => {
  row.status = row.status === 'active' ? 'revoked' : 'active'
  ElMessage.success(row.status === 'active' ? 'Key 已启用' : 'Key 已禁用')
}

const revokeApiKey = async (id) => {
  const key = apiKeys.value.find(k => k.id === id)
  if (key) {
    key.status = 'revoked'
    ElMessage.success('Key 已吊销')
  }
}

const logout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('role')
  router.push('/login')
}

const formatTime = (t) => t ? new Date(t).toLocaleString('zh-CN') : '-'

onMounted(() => {
  loadUsers()
})
</script>
