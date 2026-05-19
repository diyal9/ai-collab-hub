<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold">技能管理 (Skills)</h1>
          <div class="flex gap-3">
            <button @click="showInstall = true" class="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700">
              📦 npx 安装
            </button>
            <label class="bg-purple-600 text-white px-4 py-2 rounded-lg hover:bg-purple-700 cursor-pointer">
              📤 上传 Skill
              <input type="file" @change="uploadSkill" accept=".zip,.md,.yaml,.yml,.json" class="hidden" />
            </label>
            <button @click="showAdd = true" class="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700">
              + 手动创建
            </button>
          </div>
    </div>

    <!-- Skill 列表 -->
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <div v-for="s in skills" :key="s.id" class="bg-white rounded-xl shadow p-5 border border-gray-100"
        :class="{ 'opacity-60': !s.enabled }">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <span class="text-3xl">{{ s.icon || '🧩' }}</span>
            <div>
              <h3 class="font-semibold text-lg">{{ s.name }}</h3>
              <p class="text-sm text-gray-500">v{{ s.version }} · {{ s.category }}</p>
            </div>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" :checked="s.enabled" @change="toggleSkill(s.id)" class="sr-only peer" />
            <div class="w-9 h-5 bg-gray-200 peer-checked:bg-blue-600 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all"></div>
          </label>
        </div>

        <p class="mt-3 text-sm text-gray-600 line-clamp-2">{{ s.description }}</p>

        <div class="mt-3 text-xs text-gray-400">
          <span>触发: {{ s.trigger || '-' }}</span> · <span>作者: {{ s.author }}</span>
        </div>

        <div v-if="s.manifest" class="mt-3">
          <button @click="viewManifest(s)" class="text-xs text-blue-600 hover:underline">查看 SKILL.md</button>
        </div>

        <div class="mt-4 flex gap-2">
          <button @click="editSkill(s)" class="flex-1 text-sm py-2 rounded-lg bg-gray-50 hover:bg-gray-100">
            ✏️ 编辑
          </button>
          <button @click="deleteSkill(s.id)" class="px-3 py-2 text-sm rounded-lg bg-red-50 text-red-600 hover:bg-red-100">
            🗑️
          </button>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="skills.length === 0" class="text-center py-16 text-gray-400">
      <div class="text-5xl mb-4">🧩</div>
      <p>暂无 Skills，上传或手动创建</p>
    </div>

    <!-- 添加/编辑弹窗 -->
    <div v-if="showAdd" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-lg max-h-[80vh] overflow-y-auto">
        <h2 class="text-xl font-bold mb-4">{{ editing ? '编辑 Skill' : '创建 Skill' }}</h2>
        <div class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-1">名称</label>
              <input v-model="form.name" class="w-full border rounded-lg px-3 py-2" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">版本</label>
              <input v-model="form.version" class="w-full border rounded-lg px-3 py-2" />
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">描述</label>
            <textarea v-model="form.description" class="w-full border rounded-lg px-3 py-2" rows="2"></textarea>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-1">分类</label>
              <select v-model="form.category" class="w-full border rounded-lg px-3 py-2">
                <option value="devops">DevOps</option>
                <option value="research">Research</option>
                <option value="creative">Creative</option>
                <option value="data-science">Data Science</option>
                <option value="mlops">MLOps</option>
                <option value="productivity">Productivity</option>
                <option value="other">Other</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">Icon (Emoji)</label>
              <input v-model="form.icon" class="w-full border rounded-lg px-3 py-2" placeholder="🧩" />
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">触发关键词</label>
            <input v-model="form.trigger" class="w-full border rounded-lg px-3 py-2" placeholder="如: deploy, build" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">SKILL.md 内容</label>
            <textarea v-model="form.manifest" class="w-full border rounded-lg px-3 py-2 font-mono text-xs" rows="6"></textarea>
          </div>
        </div>
        <div class="flex gap-3 mt-6">
          <button @click="showAdd = false" class="flex-1 py-2 rounded-lg border">取消</button>
          <button @click="saveSkill" class="flex-1 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700">保存</button>
        </div>
      </div>
    </div>

    <!-- npx 安装弹窗 -->
    <div v-if="showInstall" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-lg">
        <h2 class="text-xl font-bold mb-4">📦 npx 安装技能包</h2>
        
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-1">技能包名称</label>
            <input v-model="installForm.name" class="w-full border rounded-lg px-3 py-2" placeholder="如: cocos-ui-workflow" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">技能源路径 (可选)</label>
            <input v-model="installForm.source" class="w-full border rounded-lg px-3 py-2" placeholder="默认: ~/aispace/hermesshare/skillhub-registry" />
          </div>
          
          <div class="bg-gray-50 p-3 rounded-lg text-sm">
            <div class="font-medium mb-1">命令行等价命令:</div>
            <code class="text-xs bg-gray-100 p-1 rounded block">
              cd ~/aispace/hermesshare/skillhub-cli && \<br/>
              node bin/skillhub.js install {{ installForm.name || '名称' }} {{ installForm.source ? '-s ' + installForm.source : '' }}
            </code>
          </div>
        </div>
        
        <div class="flex gap-3 mt-6">
          <button @click="showInstall = false" class="flex-1 py-2 rounded-lg border">取消</button>
          <button @click="installViaNpx" class="flex-1 py-2 rounded-lg bg-green-600 text-white hover:bg-green-700">安装</button>
        </div>
      </div>
    </div>

    <!-- Manifest 查看弹窗 -->
    <div v-if="showManifest" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-2xl max-h-[80vh] overflow-y-auto">
        <h2 class="text-xl font-bold mb-4">SKILL.md - {{ currentSkill?.name }}</h2>
        <pre class="bg-gray-50 p-4 rounded-lg text-sm whitespace-pre-wrap">{{ manifestContent }}</pre>
        <div class="mt-4 flex justify-end">
          <button @click="showManifest = false" class="px-4 py-2 rounded-lg border">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getSkills, createSkill, updateSkill, deleteSkill as delSkill, uploadSkill as upSkill, toggleSkill as togSkill, getSkillManifest } from '../api'

const skills = ref([])
const showAdd = ref(false)
const showInstall = ref(false)
const installForm = ref({ name: '', source: '' })
const showManifest = ref(false)
const editing = ref(null)
const currentSkill = ref(null)
const manifestContent = ref('')
const form = ref({ name: '', version: '1.0.0', description: '', category: 'other', icon: '🧩', trigger: '', manifest: '' })

const load = async () => {
  const { data } = await getSkills()
  skills.value = data
}

const saveSkill = async () => {
  if (editing.value) {
    await updateSkill(editing.value.id, form.value)
  } else {
    await createSkill(form.value)
  }
  showAdd.value = false
  form.value = { name: '', version: '1.0.0', description: '', category: 'other', icon: '🧩', trigger: '', manifest: '' }
  editing.value = null
  load()
}

const editSkill = (s) => {
  editing.value = s
  form.value = { name: s.name, version: s.version, description: s.description, category: s.category, icon: s.icon, trigger: s.trigger, manifest: s.manifest }
  showAdd.value = true
}

const deleteSkill = async (id) => {
  if (!confirm('确定删除此 Skill？')) return
  await delSkill(id)
  load()
}

const uploadSkill = async (e) => {
  const file = e.target.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  await upSkill(fd)
  load()
}

const toggleSkill = async (id) => {
  await togSkill(id)
  load()
}

const installViaNpx = async () => {
  if (!installForm.value.name) {
    alert('请输入技能包名称')
    return
  }
  
  // 调用后端 API 执行安装命令
  try {
    const cmd = `cd ~/aispace/hermesshare/skillhub-cli && node bin/skillhub.js install ${installForm.value.name}${installForm.value.source ? ' -s ' + installForm.value.source : ''}`
    // 这里应该调用后端执行命令的 API
    // 暂时使用 alert 提示用户手动执行
    alert(`请在终端执行以下命令安装:\n\n${cmd}`)
    showInstall.value = false
    installForm.value = { name: '', source: '' }
  } catch (err) {
    alert('安装失败: ' + err.message)
  }
}

const viewManifest = async (s) => {
  currentSkill.value = s
  const { data } = await getSkillManifest(s.id)
  manifestContent.value = data.manifest || '无内容'
  showManifest.value = true
}

onMounted(load)
</script>
