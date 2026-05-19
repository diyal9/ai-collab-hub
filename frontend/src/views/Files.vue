<template>
  <div>
    <h1 class="text-2xl font-bold mb-6">📂 文件管理</h1>

    <!-- 上传 -->
    <div v-if="activeTab !== 'shared'" class="mb-6">
      <label class="inline-flex items-center bg-blue-600 text-white px-4 py-2 rounded-lg cursor-pointer hover:bg-blue-700">
        📤 上传文件
        <input type="file" @change="upload" class="hidden" />
      </label>
    </div>

    <!-- Tab 切换 -->
    <div class="flex gap-4 mb-4 border-b">
      <button @click="activeTab = 'db'" :class="activeTab === 'db' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500'"
        class="px-4 py-2 border-b-2 text-sm font-medium">
        数据库文件 ({{ dbFiles.length }})
      </button>
      <button @click="activeTab = 'legacy'" :class="activeTab === 'legacy' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500'"
        class="px-4 py-2 border-b-2 text-sm font-medium">
        原项目文件 ({{ legacyFiles.length }})
      </button>
      <button @click="activeTab = 'uploads'" :class="activeTab === 'uploads' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500'"
        class="px-4 py-2 border-b-2 text-sm font-medium">
        上传目录 ({{ uploadedFiles.length }})
      </button>
      <button @click="activeTab = 'shared'" :class="activeTab === 'shared' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500'"
        class="px-4 py-2 border-b-2 text-sm font-medium">
        📦 共享资源 ({{ sharedFiles.length }})
      </button>
    </div>

    <!-- 数据库文件 -->
    <div v-if="activeTab === 'db'" class="bg-white rounded-xl shadow">
      <table class="w-full text-sm">
        <thead class="bg-gray-50">
          <tr><th class="text-left p-3">文件名</th><th class="text-left p-3">大小</th><th class="text-left p-3">上传者</th><th class="text-left p-3">时间</th></tr>
        </thead>
        <tbody>
          <tr v-for="f in dbFiles" :key="f.id" class="border-t hover:bg-gray-50">
            <td class="p-3">{{ f.file_name }}</td>
            <td class="p-3">{{ formatSize(f.size) }}</td>
            <td class="p-3">{{ f.uploader }}</td>
            <td class="p-3">{{ formatTime(f.created_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 原项目文件 -->
    <div v-if="activeTab === 'legacy'" class="bg-white rounded-xl shadow">
      <div class="p-3 text-xs text-gray-500 bg-yellow-50">来源: /root/aispace/hermesshare</div>
      <table class="w-full text-sm">
        <thead class="bg-gray-50">
          <tr><th class="text-left p-3">文件名</th><th class="text-left p-3">类型</th><th class="text-left p-3">大小</th><th class="text-left p-3">修改时间</th><th class="text-left p-3">操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="f in legacyFiles" :key="f.path" class="border-t hover:bg-gray-50">
            <td class="p-3">{{ f.is_dir ? '📁' : fileIcon(f.ext) }} {{ f.name }}</td>
            <td class="p-3">{{ f.is_dir ? '目录' : f.ext || '-' }}</td>
            <td class="p-3">{{ f.is_dir ? '-' : formatSize(f.size) }}</td>
            <td class="p-3">{{ f.mod_time }}</td>
            <td class="p-3">
              <button v-if="!f.is_dir" @click="download(f.path)" class="text-blue-600 hover:underline text-xs">下载</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 上传目录文件 -->
    <div v-if="activeTab === 'uploads'" class="bg-white rounded-xl shadow">
      <div class="p-3 text-xs text-gray-500 bg-blue-50">来源: 上传存储目录</div>
      <table class="w-full text-sm">
        <thead class="bg-gray-50">
          <tr><th class="text-left p-3">文件名</th><th class="text-left p-3">类型</th><th class="text-left p-3">大小</th><th class="text-left p-3">修改时间</th><th class="text-left p-3">操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="f in uploadedFiles" :key="f.path" class="border-t hover:bg-gray-50">
            <td class="p-3">{{ f.is_dir ? '📁' : fileIcon(f.ext) }} {{ f.name }}</td>
            <td class="p-3">{{ f.is_dir ? '目录' : f.ext || '-' }}</td>
            <td class="p-3">{{ f.is_dir ? '-' : formatSize(f.size) }}</td>
            <td class="p-3">{{ f.mod_time }}</td>
            <td class="p-3">
              <button v-if="!f.is_dir" @click="download(f.path)" class="text-blue-600 hover:underline text-xs">下载</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 共享资源 -->
    <div v-if="activeTab === 'shared'" class="bg-white rounded-xl shadow">
      <div class="p-3 text-xs text-gray-500 bg-green-50">AI Agent 工具、插件、架构文档等共享文件</div>
      
      <!-- 文件夹导航 -->
      <div v-if="sharedCurrentPath !== ''" class="flex items-center space-x-2 text-sm text-gray-500 px-4 py-2 border-b">
        <button @click="sharedCurrentPath = ''" class="hover:text-blue-600">📁 根目录</button>
        <template v-for="(part, i) in sharedPathParts" :key="i">
          <span>/</span>
          <button @click="sharedNavigateTo(i)" class="hover:text-blue-600">{{ part }}</button>
        </template>
      </div>

      <table class="w-full text-sm">
        <thead class="bg-gray-50">
          <tr>
            <th class="text-left p-3">名称</th>
            <th class="text-left p-3">大小</th>
            <th class="text-left p-3">修改时间</th>
            <th class="text-left p-3">操作</th>
          </tr>
        </thead>
        <tbody>
          <!-- 返回上级 -->
          <tr v-if="sharedCurrentPath !== ''" class="hover:bg-gray-50 cursor-pointer" @click="sharedGoUp">
            <td class="p-3">
              <div class="flex items-center">
                <span class="text-xl mr-2">⬆️</span>
                <span class="text-sm font-medium text-gray-900">..</span>
              </div>
            </td>
            <td colspan="3"></td>
          </tr>

          <!-- 文件夹 -->
          <tr v-for="folder in sharedFolders" :key="folder.path" 
              class="hover:bg-gray-50 cursor-pointer" @click="sharedOpenFolder(folder)">
            <td class="p-3">
              <div class="flex items-center">
                <span class="text-xl mr-2">📁</span>
                <span class="text-sm font-medium text-gray-900">{{ folder.name }}</span>
              </div>
            </td>
            <td class="p-3">-</td>
            <td class="p-3">{{ folder.mod_time }}</td>
            <td class="p-3">
              <button @click.stop="sharedOpenFolder(folder)" class="text-blue-600 hover:underline text-xs">打开</button>
            </td>
          </tr>

          <!-- 文件 -->
          <tr v-for="file in sharedFileList" :key="file.path" class="hover:bg-gray-50">
            <td class="p-3">
              <div class="flex items-center">
                <span class="text-xl mr-2">{{ sharedFileIcon(file.ext) }}</span>
                <span class="text-sm font-medium text-gray-900">{{ file.name }}</span>
              </div>
            </td>
            <td class="p-3">{{ sharedFormatSize(file.size) }}</td>
            <td class="p-3">{{ file.mod_time }}</td>
            <td class="p-3">
              <a :href="sharedDownloadUrl(file.path)" 
                 class="text-blue-600 hover:underline text-xs inline-flex items-center"
                 download>
                ⬇️ 下载
              </a>
            </td>
          </tr>

          <!-- 空状态 -->
          <tr v-if="sharedFolders.length === 0 && sharedFileList.length === 0">
            <td colspan="4" class="p-8 text-center text-gray-500">暂无共享资源</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="isEmpty" class="text-center py-16 text-gray-400">
      <div class="text-5xl mb-4">📂</div>
      <p>暂无文件</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getFiles, uploadFile as upFile, downloadFile } from '../api'
import axios from 'axios'

const dbFiles = ref([])
const legacyFiles = ref([])
const uploadedFiles = ref([])
const activeTab = ref('db')

// 共享资源
const sharedFiles = ref([])
const sharedCurrentPath = ref('')

const sharedPathParts = computed(() => {
  if (!sharedCurrentPath.value) return []
  return sharedCurrentPath.value.split('/').filter(Boolean)
})

const sharedFolders = computed(() => {
  return sharedFiles.value.filter(f => {
    const relPath = f.path.replace('/resources/download/', '')
    if (sharedCurrentPath.value === '') {
      return f.is_dir && !relPath.includes('/')
    }
    return f.is_dir && relPath.startsWith(sharedCurrentPath.value + '/') && 
           relPath.replace(sharedCurrentPath.value + '/', '').split('/').length === 1
  })
})

const sharedFileList = computed(() => {
  return sharedFiles.value.filter(f => {
    if (f.is_dir) return false
    const relPath = f.path.replace('/resources/download/', '')
    if (sharedCurrentPath.value === '') {
      return !relPath.includes('/')
    }
    return relPath.startsWith(sharedCurrentPath.value + '/') && 
           relPath.replace(sharedCurrentPath.value + '/', '').split('/').length === 1
  })
})

const isEmpty = computed(() => {
  if (activeTab.value === 'db') return dbFiles.value.length === 0
  if (activeTab.value === 'legacy') return legacyFiles.value.length === 0
  if (activeTab.value === 'uploads') return uploadedFiles.value.length === 0
  return sharedFiles.value.length === 0
})

const load = async () => {
  const { data } = await getFiles()
  dbFiles.value = data.database_files || []
  legacyFiles.value = data.legacy_files || []
  uploadedFiles.value = data.uploaded_files || []
}

const loadSharedFiles = async () => {
  try {
    const { data } = await axios.get('/resources')
    sharedFiles.value = data
  } catch (e) {
    console.error('加载共享资源失败:', e)
  }
}

const upload = async (e) => {
  const file = e.target.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  await upFile(fd)
  load()
}

const download = async (path) => {
  const { data } = await downloadFile(path)
  const url = URL.createObjectURL(data)
  const a = document.createElement('a')
  a.href = url
  a.download = path.split('/').pop()
  a.click()
  URL.revokeObjectURL(url)
}

const sharedOpenFolder = (folder) => {
  const relPath = folder.path.replace('/resources/download/', '')
  sharedCurrentPath.value = relPath
}

const sharedGoUp = () => {
  const parts = sharedCurrentPath.value.split('/').filter(Boolean)
  parts.pop()
  sharedCurrentPath.value = parts.join('/')
}

const sharedNavigateTo = (index) => {
  const parts = sharedPathParts.value.slice(0, index + 1)
  sharedCurrentPath.value = parts.join('/')
}

const sharedDownloadUrl = (path) => path

const sharedFileIcon = (ext) => {
  return {
    '.zip': '📦', '.rar': '📦', '.7z': '📦',
    '.json': '📋', '.yaml': '📋', '.yml': '📋',
    '.js': '📜', '.ts': '📜',
    '.md': '📄', '.txt': '📄',
    '.png': '🖼️', '.jpg': '🖼️', '.gif': '🖼️',
    '.pdf': '📕',
  }[ext] || '📎'
}

const sharedFormatSize = (bytes) => {
  if (!bytes) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(1)} ${units[i]}`
}

const formatSize = (bytes) => {
  if (!bytes) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

const formatTime = (t) => t ? new Date(t).toLocaleString('zh-CN') : '-'

const fileIcon = (ext) => {
  return {
    '.pdf': '📕', '.doc': '📘', '.docx': '📘', '.xls': '📗', '.xlsx': '📗',
    '.png': '🖼️', '.jpg': '🖼️', '.jpeg': '🖼️', '.gif': '🖼️',
    '.zip': '📦', '.tar': '📦', '.gz': '📦',
    '.py': '🐍', '.go': '🔷', '.js': '📜', '.ts': '📜',
    '.md': '📝', '.txt': '📄', '.yaml': '⚙️', '.yml': '⚙️',
    '.json': '📋'
  }[ext] || '📄'
}

onMounted(() => {
  load()
  loadSharedFiles()
})
</script>
