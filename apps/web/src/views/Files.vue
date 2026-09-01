<template>
  <div v-loading="loading">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold flex items-center gap-2">
        <svg class="w-7 h-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
        </svg>
        文件管理
      </h1>

      <div v-if="activeTab !== 'shared'" class="flex items-center gap-3">
        <el-progress
          v-if="uploading"
          :percentage="uploadProgress"
          :stroke-width="8"
          style="width: 160px"
        />
        <label
          class="inline-flex items-center text-blue-600 border border-blue-600 px-4 py-2 rounded-lg cursor-pointer hover:bg-blue-50 transition-colors"
          :class="{ 'opacity-50 pointer-events-none': uploading }"
        >
          <svg class="w-5 h-5 mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
            <polyline points="17 8 12 3 7 8"></polyline>
            <line x1="12" y1="3" x2="12" y2="15"></line>
          </svg>
          {{ uploading ? '上传中...' : '上传文件' }}
          <input type="file" class="hidden" :disabled="uploading" @change="upload" />
        </label>
      </div>
    </div>

    <div class="flex gap-4 mb-4 border-b overflow-x-auto">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        class="px-4 py-2 border-b-2 text-sm font-medium whitespace-nowrap"
        :class="activeTab === tab.key ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500'"
        @click="activeTab = tab.key"
      >
        {{ tab.label }} ({{ tab.count }})
      </button>
    </div>

    <!-- 数据库文件 -->
    <div v-if="activeTab === 'db' && !isCurrentTabEmpty" class="bg-white rounded-xl shadow overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="bg-gray-50">
          <tr>
            <th class="text-left p-3">文件名</th>
            <th class="text-left p-3">大小</th>
            <th class="text-left p-3">上传者</th>
            <th class="text-left p-3">时间</th>
            <th class="text-left p-3">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in dbFiles" :key="f.id" class="border-t hover:bg-gray-50">
            <td class="p-3">{{ f.file_name }}</td>
            <td class="p-3">{{ formatSize(f.size) }}</td>
            <td class="p-3">{{ f.uploader }}</td>
            <td class="p-3">{{ formatTime(f.created_at) }}</td>
            <td class="p-3">
              <button class="text-red-600 hover:underline text-xs" @click="confirmDelete(f)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 原项目文件 -->
    <div v-if="activeTab === 'legacy' && !isCurrentTabEmpty" class="bg-white rounded-xl shadow overflow-x-auto">
      <div class="p-3 text-xs text-gray-500 bg-yellow-50">来源: ai-collab-hub / apps/hub</div>
      <table class="w-full text-sm">
        <thead class="bg-gray-50">
          <tr>
            <th class="text-left p-3">文件名</th>
            <th class="text-left p-3">类型</th>
            <th class="text-left p-3">大小</th>
            <th class="text-left p-3">修改时间</th>
            <th class="text-left p-3">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in legacyFiles" :key="f.path" class="border-t hover:bg-gray-50">
            <td class="p-3">
              <span class="inline-flex items-center gap-2">
                <FileTypeIcon :ext="f.ext" :is-dir="f.is_dir" />
                {{ f.name }}
              </span>
            </td>
            <td class="p-3">{{ f.is_dir ? '目录' : f.ext || '-' }}</td>
            <td class="p-3">{{ f.is_dir ? '-' : formatSize(f.size) }}</td>
            <td class="p-3">{{ f.mod_time }}</td>
            <td class="p-3">
              <button v-if="!f.is_dir" class="text-blue-600 hover:underline text-xs" @click="download(f.path)">
                下载
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 上传目录文件 -->
    <div v-if="activeTab === 'uploads' && !isCurrentTabEmpty" class="bg-white rounded-xl shadow overflow-x-auto">
      <div class="p-3 text-xs text-gray-500 bg-blue-50">来源: 上传存储目录</div>
      <table class="w-full text-sm">
        <thead class="bg-gray-50">
          <tr>
            <th class="text-left p-3">文件名</th>
            <th class="text-left p-3">类型</th>
            <th class="text-left p-3">大小</th>
            <th class="text-left p-3">修改时间</th>
            <th class="text-left p-3">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in uploadedFiles" :key="f.path" class="border-t hover:bg-gray-50">
            <td class="p-3">
              <span class="inline-flex items-center gap-2">
                <FileTypeIcon :ext="f.ext" :is-dir="f.is_dir" />
                {{ f.name }}
              </span>
            </td>
            <td class="p-3">{{ f.is_dir ? '目录' : f.ext || '-' }}</td>
            <td class="p-3">{{ f.is_dir ? '-' : formatSize(f.size) }}</td>
            <td class="p-3">{{ f.mod_time }}</td>
            <td class="p-3">
              <button v-if="!f.is_dir" class="text-blue-600 hover:underline text-xs" @click="download(f.path)">
                下载
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 共享资源 -->
    <div v-if="activeTab === 'shared' && !isCurrentTabEmpty" class="bg-white rounded-xl shadow overflow-x-auto">
      <div class="p-3 text-xs text-gray-500 bg-green-50">AI Agent 工具、插件、架构文档等共享文件</div>

      <div v-if="sharedCurrentPath !== ''" class="flex items-center space-x-2 text-sm text-gray-500 px-4 py-2 border-b">
        <button class="hover:text-blue-600" @click="sharedCurrentPath = ''">根目录</button>
        <template v-for="(part, i) in sharedPathParts" :key="i">
          <span>/</span>
          <button class="hover:text-blue-600" @click="sharedNavigateTo(i)">{{ part }}</button>
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
          <tr v-if="sharedCurrentPath !== ''" class="hover:bg-gray-50 cursor-pointer" @click="sharedGoUp">
            <td class="p-3 font-medium text-gray-900">..</td>
            <td colspan="3"></td>
          </tr>

          <tr
            v-for="folder in sharedFolders"
            :key="folder.path"
            class="hover:bg-gray-50 cursor-pointer"
            @click="sharedOpenFolder(folder)"
          >
            <td class="p-3 font-medium text-gray-900">{{ folder.name }}</td>
            <td class="p-3">-</td>
            <td class="p-3">{{ folder.mod_time }}</td>
            <td class="p-3">
              <button class="text-blue-600 hover:underline text-xs" @click.stop="sharedOpenFolder(folder)">
                打开
              </button>
            </td>
          </tr>

          <tr v-for="file in sharedFileList" :key="file.path" class="hover:bg-gray-50">
            <td class="p-3 font-medium text-gray-900">{{ file.name }}</td>
            <td class="p-3">{{ sharedFormatSize(file.size) }}</td>
            <td class="p-3">{{ file.mod_time }}</td>
            <td class="p-3">
              <a
                :href="sharedDownloadUrl(file.path)"
                class="text-blue-600 hover:underline text-xs"
                download
              >
                下载
              </a>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="!loading && isCurrentTabEmpty" class="text-center py-16 text-gray-400">
      <svg class="w-20 h-20 mx-auto mb-4 text-gray-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
      </svg>
      <p>暂无文件</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessageBox } from 'element-plus'
import axios from 'axios'
import { getFiles, uploadFile as upFile, downloadFile, deleteFile } from '../api'
import { notifySuccess, showApiError } from '../utils/notify'
import FileTypeIcon from '../components/FileTypeIcon.vue'

const loading = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const dbFiles = ref([])
const legacyFiles = ref([])
const uploadedFiles = ref([])
const activeTab = ref('db')
const sharedFiles = ref([])
const sharedCurrentPath = ref('')

const tabs = computed(() => [
  { key: 'db', label: '数据库文件', count: dbFiles.value.length },
  { key: 'legacy', label: '原项目文件', count: legacyFiles.value.length },
  { key: 'uploads', label: '上传目录', count: uploadedFiles.value.length },
  { key: 'shared', label: '共享资源', count: sharedFiles.value.length },
])

const sharedPathParts = computed(() => {
  if (!sharedCurrentPath.value) return []
  return sharedCurrentPath.value.split('/').filter(Boolean)
})

const sharedFolders = computed(() =>
  sharedFiles.value.filter((f) => {
    const relPath = f.path.replace('/resources/download/', '')
    if (sharedCurrentPath.value === '') {
      return f.is_dir && !relPath.includes('/')
    }
    return (
      f.is_dir &&
      relPath.startsWith(`${sharedCurrentPath.value}/`) &&
      relPath.replace(`${sharedCurrentPath.value}/`, '').split('/').length === 1
    )
  }),
)

const sharedFileList = computed(() =>
  sharedFiles.value.filter((f) => {
    if (f.is_dir) return false
    const relPath = f.path.replace('/resources/download/', '')
    if (sharedCurrentPath.value === '') {
      return !relPath.includes('/')
    }
    return (
      relPath.startsWith(`${sharedCurrentPath.value}/`) &&
      relPath.replace(`${sharedCurrentPath.value}/`, '').split('/').length === 1
    )
  }),
)

const sharedVisibleCount = computed(() => sharedFolders.value.length + sharedFileList.value.length)

const isCurrentTabEmpty = computed(() => {
  if (activeTab.value === 'db') return dbFiles.value.length === 0
  if (activeTab.value === 'legacy') return legacyFiles.value.length === 0
  if (activeTab.value === 'uploads') return uploadedFiles.value.length === 0
  return sharedVisibleCount.value === 0
})

const load = async () => {
  loading.value = true
  try {
    const { data } = await getFiles()
    dbFiles.value = data.database_files || []
    legacyFiles.value = data.legacy_files || []
    uploadedFiles.value = data.uploaded_files || []
  } catch (error) {
    showApiError(error, '文件列表加载失败')
  } finally {
    loading.value = false
  }
}

const loadSharedFiles = async () => {
  try {
    const { data } = await axios.get('/resources')
    sharedFiles.value = data
  } catch (error) {
    showApiError(error, '共享资源加载失败')
  }
}

const upload = async (event) => {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return

  uploading.value = true
  uploadProgress.value = 0
  try {
    const fd = new FormData()
    fd.append('file', file)
    await upFile(fd, (percent) => {
      uploadProgress.value = percent
    })
    notifySuccess(`${file.name} 上传成功`)
    await load()
  } catch (error) {
    showApiError(error, '文件上传失败')
  } finally {
    uploading.value = false
    uploadProgress.value = 0
  }
}

const download = async (path) => {
  try {
    const { data } = await downloadFile(path)
    const url = URL.createObjectURL(data)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = path.split('/').pop() || 'download'
    anchor.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    showApiError(error, '文件下载失败')
  }
}

const confirmDelete = async (file) => {
  try {
    await ElMessageBox.confirm(`确定删除「${file.file_name}」吗？此操作不可恢复。`, '删除文件', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
    await deleteFile(file.id)
    notifySuccess('文件已删除')
    await load()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      showApiError(error, '文件删除失败，后端可能尚未支持此接口')
    }
  }
}

const sharedOpenFolder = (folder) => {
  sharedCurrentPath.value = folder.path.replace('/resources/download/', '')
}

const sharedGoUp = () => {
  const parts = sharedCurrentPath.value.split('/').filter(Boolean)
  parts.pop()
  sharedCurrentPath.value = parts.join('/')
}

const sharedNavigateTo = (index) => {
  sharedCurrentPath.value = sharedPathParts.value.slice(0, index + 1).join('/')
}

const sharedDownloadUrl = (path) => path

const sharedFormatSize = (bytes) => formatSize(bytes)

const formatSize = (bytes) => {
  if (!bytes) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

const formatTime = (t) => (t ? new Date(t).toLocaleString('zh-CN') : '-')

onMounted(() => {
  load()
  loadSharedFiles()
})
</script>
