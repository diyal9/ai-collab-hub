<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold flex items-center gap-2"><svg class="w-7 h-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="18" cy="5" r="3"></circle><circle cx="6" cy="12" r="3"></circle><circle cx="18" cy="19" r="3"></circle><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line></svg> Agent 编排</h1>
      <button @click="newFlow" class="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700">
        + 新建流程
      </button>
    </div>

    <!-- 流程选择 -->
    <div v-if="!currentFlow" class="grid gap-4 md:grid-cols-3">
      <div v-for="f in flows" :key="f.id" @click="selectFlow(f)"
        class="bg-white rounded-xl shadow p-5 border border-gray-100 cursor-pointer hover:border-blue-300">
        <div class="flex justify-between items-start">
          <div>
            <h3 class="font-semibold">{{ f.name }}</h3>
            <p class="text-sm text-gray-500 mt-1">{{ f.description || '无描述' }}</p>
          </div>
          <span :class="flowStatusClass(f.status)" class="px-2 py-1 rounded text-xs">{{ f.status }}</span>
        </div>
        <div class="mt-3 text-xs text-gray-400">类型: {{ f.type }} · 更新: {{ formatTime(f.updated_at) }}</div>
      </div>
    </div>

    <!-- 画布 (Vue Flow) -->
    <div v-if="currentFlow" class="bg-white rounded-xl shadow border border-gray-100">
      <div class="flex items-center justify-between p-4 border-b">
        <div class="flex items-center gap-3">
          <button @click="currentFlow = null" class="text-gray-500 hover:text-gray-700">← 返回</button>
          <h2 class="font-semibold">{{ currentFlow.name }}</h2>
          <span :class="flowStatusClass(currentFlow.status)" class="px-2 py-1 rounded text-xs">{{ currentFlow.status }}</span>
        </div>
        <div class="flex gap-2">
          <button @click="$router.push('/node-templates')" class="bg-gray-100 text-gray-700 px-3 py-1 rounded text-sm hover:bg-gray-200 flex items-center gap-1.5">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"></path></svg>
            节点工厂
          </button>
          <button @click="validateFlow" class="bg-gray-100 text-gray-700 px-3 py-1 rounded text-sm hover:bg-gray-200 flex items-center gap-1.5">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
            验证
          </button>
          <el-popconfirm title="确定执行此流程？" @confirm="executeCurrentFlow" width="200">
            <template #reference>
              <button @click.stop :disabled="executing" class="bg-green-600 text-white px-3 py-1 rounded text-sm hover:bg-green-700 disabled:opacity-50 flex items-center gap-1.5">
                <svg v-if="!executing" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>
                <svg v-else class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 1 1-6.219-8.56"></path></svg>
                {{ executing ? '执行中...' : '执行流程' }}
              </button>
            </template>
          </el-popconfirm>
          <select v-model="currentFlow.status" @change="saveFlowMeta" class="border rounded px-2 py-1 text-sm">
            <option value="draft">草稿</option>
            <option value="active">运行中</option>
            <option value="archived">已归档</option>
          </select>
          <button @click="saveGraph" class="bg-blue-600 text-white px-3 py-1 rounded text-sm hover:bg-blue-700 flex items-center gap-1.5">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path><polyline points="17 21 17 13 7 13 7 21"></polyline><polyline points="7 3 7 8 15 8"></polyline></svg>
            保存
          </button>
          <el-popconfirm title="确定删除此流程？" @confirm="() => deleteFlow(currentFlow.id)" width="200">
            <template #reference>
              <button class="bg-red-50 text-red-600 px-3 py-1 rounded text-sm hover:bg-red-100 flex items-center gap-1.5">
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                删除
              </button>
            </template>
          </el-popconfirm>
        </div>
      </div>

      <div class="flex flex-col h-[calc(100vh-130px)]" :class="{ 'pb-14': currentFlow && executions.length > 0 }">
  <!-- Main Content (Canvas + Sidebar) -->
  <div class="flex flex-1 min-h-0">
        <!-- 节点面板 -->
        <div class="w-56 border-r bg-gray-50 flex flex-col overflow-hidden">
          <!-- 搜索 -->
          <div class="p-2 border-b">
            <input v-model="nodeSearch" placeholder="搜索节点..." class="w-full border rounded px-2 py-1 text-xs bg-white" />
          </div>
          
          <div class="flex-1 overflow-y-auto p-2">
            <!-- 可折叠分类 -->
            <div v-for="cat in filteredCategories" :key="cat.name" class="mb-2">
              <div @click="toggleCategory(cat.name)"
                class="flex items-center gap-2 px-2 py-2 rounded cursor-pointer hover:bg-gray-200/50 transition select-none">
                <svg class="w-3.5 h-3.5 text-gray-400 transition-transform flex-shrink-0"
                  :class="{ 'rotate-90': expandedCats.has(cat.name) }"
                  viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="9 18 15 12 9 6"></polyline>
                </svg>
                <svg class="w-4 h-4 text-gray-600 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="3" width="7" height="7"></rect>
                  <rect x="14" y="3" width="7" height="7"></rect>
                  <rect x="14" y="14" width="7" height="7"></rect>
                  <rect x="3" y="14" width="7" height="7"></rect>
                </svg>
                <span class="text-xs font-semibold text-gray-500 flex-1">{{ cat.name }}</span>
                <span class="text-[10px] text-gray-400">{{ cat.templates.length }}</span>
              </div>
              <div v-show="expandedCats.has(cat.name)" class="space-y-1 pl-2 mt-1">
                <div v-for="tpl in cat.templates" :key="tpl.type + tpl.id"
                  @click="addNodeFromTemplate(tpl)"
                  class="group flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer hover:bg-white hover:shadow-sm transition border border-transparent hover:border-gray-200">
                  <span class="w-4 h-4 flex-shrink-0" v-html="tplNodeIcon(tpl.type)"></span>
                  <div class="min-w-0 flex-1">
                    <div class="text-xs font-medium truncate">{{ tpl.name }}</div>
                    <div class="text-[10px] text-gray-400 truncate">{{ tpl.desc }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          <div class="p-2 border-t text-[10px] text-gray-400 text-center">
            点击模板添加到画布 · 拖拽连线
          </div>
        </div>

        <!-- Vue Flow 画布 -->
        <div class="flex-1 relative" @contextmenu.prevent="showContextMenu">
          <VueFlow 
            :nodes="flowNodes" 
            :edges="flowEdges" 
            @node-click="onNodeClick"
            @node-context-menu="onNodeContextMenu"
            @pane-click="closeNodeContextMenu"
            @connect="onConnect"
            @nodes-change="onNodesChange"
            class="bg-gray-50"
          >
            <Background />
          </VueFlow>
        </div>

        <!-- 节点属性面板 -->
        <div v-if="selectedNodeData" class="w-72 border-l p-4 space-y-3 bg-white overflow-y-auto">
          <div class="font-medium text-base border-b pb-2 mb-2">
            {{ nodeTypeIcon(selectedNodeData.type) }} 节点属性
          </div>
          
          <div>
            <label class="block text-xs text-gray-500 mb-1">名称</label>
            <input v-model="selectedNodeData.label" class="w-full border rounded px-2 py-1 text-sm" />
          </div>
          <div>
            <label class="block text-xs text-gray-500 mb-1">类型</label>
            <select v-model="selectedNodeData.type" class="w-full border rounded px-2 py-1 text-sm">
              <option value="agent">Agent</option>
              <option value="condition">Condition</option>
              <option value="merge">Merge</option>
              <option value="trigger">Trigger</option>
              <option value="webhook">Webhook</option>
              <option value="code">Code</option>
            </select>
          </div>
          <div v-if="selectedNodeData.type === 'agent'">
            <label class="block text-xs text-gray-500 mb-1">关联 Agent</label>
            <select v-model="selectedNodeData._selectedAgentId" @change="onAgentChange" class="w-full border rounded px-2 py-1 text-sm">
              <option value="">自动匹配（任意空闲 Agent）</option>
              <option v-for="agent in agents" :key="agent.id" :value="String(agent.id)">
                {{ agent.name }} ({{ agent.status }})
              </option>
            </select>
          </div>
          <div>
            <label class="block text-xs text-gray-500 mb-1">配置 / Prompt</label>
            <textarea v-model="selectedNodeData._rawConfig" class="w-full border rounded px-2 py-1 text-xs font-mono" rows="6" placeholder="JSON 配置或 Prompt..."></textarea>
          </div>

          <!-- 动态模板的 Schema 渲染 -->
          <div v-if="currentTemplateForNode">
            <label class="block text-xs text-gray-500 mb-1">节点参数 (动态表单)</label>
            <div v-for="field in currentTemplateForNode.schema" :key="field.key" class="mb-2">
              <span class="text-xs text-gray-400">{{ field.label }} ({{ field.key }})</span>
              <div v-if="field.type === 'select'">
                <select v-model="selectedNodeData['_' + field.key]" class="w-full border rounded px-2 py-1 text-sm">
                  <option v-for="opt in field.options" :value="opt">{{ opt }}</option>
                </select>
              </div>
              <div v-else-if="field.type === 'boolean'">
                <input type="checkbox" v-model="selectedNodeData['_' + field.key]" class="w-4 h-4" />
              </div>
              <div v-else>
                <input v-model="selectedNodeData['_' + field.key]" class="w-full border rounded px-2 py-1 text-sm" />
              </div>
            </div>
          </div>
          
          <div>
            <label class="block text-xs text-gray-500 mb-1">超时 (分钟)</label>
            <input v-model.number="selectedNodeData._timeout" type="number" class="w-full border rounded px-2 py-1 text-sm" min="1" max="120" />
          </div>
          
          <div class="flex gap-2 mt-4">
            <el-popconfirm title="确定删除此节点？" @confirm="() => deleteNode(selectedNodeData.id)" width="200">
              <template #reference>
                <button class="flex-1 text-sm py-1.5 rounded bg-red-50 text-red-600 hover:bg-red-100 border border-red-200">
                  删除节点
                </button>
              </template>
            </el-popconfirm>
          </div>
        </div>
      </div>

    </div>

    <!-- 底部执行状态栏 (固定在视口底部) -->
    <div v-if="currentFlow && executions.length > 0" 
      class="fixed bottom-0 left-16 md:left-52 right-72 z-40 bg-white border-t border-gray-200 shadow-xl transition-all duration-300"
      :class="execPanelExpanded ? 'max-h-[220px]' : 'max-h-[52px]'"
      style="box-shadow: 0 -4px 20px rgba(0,0,0,0.08);">
      
      <!-- 顶部单行 (点击展开/收起) -->
      <div @click="execPanelExpanded = !execPanelExpanded"
        class="flex items-center justify-between px-4 py-2.5 cursor-pointer hover:bg-gray-50 transition select-none border-b border-gray-100">
        <div class="flex items-center gap-3">
          <svg class="w-4 h-4 text-gray-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"></circle>
            <polyline points="12 6 12 12 16 14"></polyline>
          </svg>
          <span class="text-sm font-medium text-gray-700">
            最近执行: #{{ executions[0].id }} 
            <span :class="execStatusBadge(executions[0].status)" class="px-1.5 py-0.5 rounded ml-1">{{ executions[0].status }}</span>
          </span>
          <span class="text-xs text-gray-400">{{ formatTime(executions[0].started_at) }}</span>
        </div>
        <svg class="w-4 h-4 text-gray-400 transition-transform duration-300"
          :class="{ 'rotate-180': execPanelExpanded }"
          viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9"></polyline>
        </svg>
      </div>
      
      <!-- 展开后显示最近3次执行记录 -->
      <div v-show="execPanelExpanded" class="overflow-y-auto max-h-[168px] p-2 space-y-1.5">
        <div v-for="exec in executions.slice(0, 3)" :key="exec.id"
          @click="showExecDetail(exec)"
          class="flex items-center justify-between px-3 py-2 rounded-lg cursor-pointer hover:bg-blue-50 transition border border-gray-100 bg-white">
          <div class="flex items-center gap-2">
            <span class="text-xs font-medium text-gray-700">#{{ exec.id }}</span>
            <span :class="execStatusBadge(exec.status)" class="px-1.5 py-0.5 rounded text-xs">{{ exec.status }}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-xs text-gray-400">{{ formatTime(exec.started_at) }}</span>
            <svg class="w-3.5 h-3.5 text-gray-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M5 12h14"></path><path d="M12 5l7 7-7 7"></path>
            </svg>
          </div>
        </div>
        <div v-if="executions.length > 3" class="text-center text-xs text-gray-400 py-1">
          共 {{ executions.length }} 次执行 · 点击查看详情
        </div>
      </div>
    </div>

    <!-- 执行详情弹窗 -->
      <div v-if="showDetail" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div class="bg-white rounded-lg p-6 w-full max-w-2xl max-h-[80vh] flex flex-col">
          <div class="flex justify-between items-center mb-4 border-b pb-2">
            <h3 class="text-lg font-semibold">执行 #{{ currentExec?.id }} 详情</h3>
            <div class="flex gap-2">
              <button v-if="currentExec?.status === 'running'" @click="cancelExec(currentExec.id)"
                class="bg-red-500 text-white px-3 py-1 rounded text-sm hover:bg-red-600">取消</button>
              <button @click="showDetail = false" class="text-gray-500 hover:text-gray-700 text-2xl">&times;</button>
            </div>
          </div>
          <div class="text-sm text-gray-500 mb-4">
            状态: <span :class="execStatusBadge(currentExec?.status)" class="px-2 py-0.5 rounded">{{ currentExec?.status }}</span>
            · 开始: {{ formatTime(currentExec?.started_at) }}
            · 结束: {{ formatTime(currentExec?.ended_at) }}
          </div>
          <div class="space-y-2 overflow-y-auto flex-1 pr-2">
            <div v-for="step in execSteps" :key="step.id"
              class="flex items-center gap-3 p-3 rounded border bg-white"
              :class="stepStatusColor(step.status)">
              <span class="text-xl" v-html="stepTypeIcon(step.node_type)"></span>
              <div class="flex-1">
                <div class="font-medium">{{ step.node_name }} <span class="text-xs text-gray-400">({{ step.node_type }})</span></div>
                <div class="text-[10px] text-gray-400 mt-0.5">开始: {{ formatTime(step.started_at) }}</div>
                <div v-if="step.output" class="text-xs text-gray-600 mt-1 font-mono bg-gray-50 p-1 rounded max-h-20 overflow-auto">{{ step.output }}</div>
                <div v-if="step.error" class="text-xs text-red-500 mt-1">{{ step.error }}</div>
              </div>
              <span class="text-xs px-2 py-1 rounded font-medium" :class="stepStatusBadge(step.status)">{{ step.status }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- 右键菜单 -->
  <div v-if="contextMenu.show" 
       class="fixed bg-white rounded-lg shadow-xl border border-gray-200 z-50 py-1 min-w-[180px]"
       :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
       @click.stop>
    <button @click="arrangeNodes(); hideContextMenu()" class="w-full text-left px-4 py-2.5 text-sm hover:bg-blue-50 text-gray-700 flex items-center gap-2 transition-colors">
      ✨ 整理节点 (自动排版)
    </button>
    <div class="border-t border-gray-100 my-1"></div>
    <button @click="deleteSelectedNodes(); hideContextMenu()" class="w-full text-left px-4 py-2.5 text-sm hover:bg-red-50 text-red-600 flex items-center gap-2 transition-colors">
      🗑️ 删除选中节点
    </button>
  </div>
</template>
<script setup>
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import {
  executeFlow, getFlowExecutions, getExecutionSteps, cancelExecution, validateFlow as apiValidateFlow,
  getFlows, createFlow, updateFlow, deleteFlow as deleteFlowApi,
  getFlowNodes, createFlowNode, updateFlowNode, deleteFlowNode,
  getFlowEdges, createFlowEdge, deleteFlowEdge, saveFlowGraph,
  getTerminals,
} from '../api'
import { ElMessage, ElMessageBox, ElPopconfirm } from 'element-plus'

// State
const flows = ref([])
const currentFlow = ref(null)
const nodes = ref([]) // 原始数据 (用于后端同步)
const edges = ref([]) // 原始数据
const executions = ref([])
const executing = ref(false)
const showDetail = ref(false)
const currentExec = ref(null)
const execSteps = ref([])
const execPanelExpanded = ref(false)

// Agent 终端列表
const agents = ref([])

// 节点搜索
const nodeSearch = ref('')
// 节点右键菜单状态
const nodeContextMenu = ref({ visible: false, x: 0, y: 0, node: null });
const contextMenu = ref({ show: false, x: 0, y: 0 })

// 左侧分类折叠状态
const expandedCats = ref(new Set(['基础'])) // 默认展开「基础」
const toggleCategory = (name) => {
  if (expandedCats.value.has(name)) expandedCats.value.delete(name)
  else expandedCats.value.add(name)
  expandedCats.value = new Set(expandedCats.value) // trigger reactivity
}

// 左侧面板节点图标 (SVG for v-html)
const tplNodeIcon = (type) => {
  const icons = {
    agent: '<svg class="w-4 h-4 text-gray-500 group-hover:text-blue-500 transition-colors" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="10" rx="2"/><circle cx="12" cy="5" r="2"/><path d="M12 7v4"/></svg>',
    condition: '<svg class="w-4 h-4 text-gray-500 group-hover:text-blue-500 transition-colors" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>',
    merge: '<svg class="w-4 h-4 text-gray-500 group-hover:text-blue-500 transition-colors" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>',
    trigger: '<svg class="w-4 h-4 text-gray-500 group-hover:text-blue-500 transition-colors" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>',
    webhook: '<svg class="w-4 h-4 text-gray-500 group-hover:text-blue-500 transition-colors" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>',
    code: '<svg class="w-4 h-4 text-gray-500 group-hover:text-blue-500 transition-colors" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',
  }
  return icons[type] || '<svg class="w-4 h-4 text-gray-500 group-hover:text-blue-500 transition-colors" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/></svg>'
}

// 节点模板定义 (静态)
const staticNodeTemplates = [
  // 基础
  { id: 'agent', type: 'agent', name: 'Agent 节点', icon: '🤖', desc: '通用 AI Agent 任务', category: '基础',
    config: { prompt: '', timeout_minutes: 30 }},
  { id: 'condition', type: 'condition', name: '条件分支', icon: '🔀', desc: 'if/else 逻辑判断', category: '基础',
    config: { expression: '', timeout_minutes: 5 }},
  { id: 'merge', type: 'merge', name: '合并节点', icon: '🔗', desc: '等待多个上游完成', category: '基础',
    config: {}},
  { id: 'trigger', type: 'trigger', name: '触发器', icon: '⚡', desc: '流程入口 / 定时触发', category: '基础',
    config: {}},
  { id: 'webhook', type: 'webhook', name: 'Webhook', icon: '🔌', desc: 'HTTP 请求 / API 调用', category: '基础',
    config: { method: 'GET', url: '', timeout_minutes: 10 }},
  { id: 'code', type: 'code', name: '代码执行', icon: '💻', desc: '运行脚本 / 命令', category: '基础',
    config: { language: 'python', code: '', timeout_minutes: 10 }},
  // 代码质量
  { id: 'review', type: 'agent', name: '代码审查', icon: '👁️', desc: '检查规范/安全/性能', category: '代码质量',
    config: { prompt: '检查代码规范、安全漏洞和性能问题', timeout_minutes: 15 }},
  { id: 'unittest', type: 'agent', name: '单元测试', icon: '🧪', desc: '生成并运行单元测试', category: '代码质量',
    config: { prompt: '为核心逻辑编写单元测试并执行', timeout_minutes: 15 }},
  { id: 'lint', type: 'code', name: 'Lint 检查', icon: '🔍', desc: '代码风格静态检查', category: '代码质量',
    config: { language: 'shell', code: 'golangci-lint run ./...', timeout_minutes: 10 }},
  // Cocos / 游戏
  { id: 'datagen', type: 'agent', name: '打表生成', icon: '📊', desc: 'Excel → JSON/TS 配置表', category: 'Cocos / 游戏',
    config: { prompt: '将策划配置表转换为项目格式的 JSON 和 TypeScript 定义', timeout_minutes: 15 }},
  { id: 'build', type: 'code', name: '构建打包', icon: '📦', desc: 'Cocos 命令行构建', category: 'Cocos / 游戏',
    config: { language: 'shell', code: 'cocos compile -p web-mobile', timeout_minutes: 30 }},
  { id: 'prefab', type: 'agent', name: 'Prefab 生成', icon: '🏗️', desc: 'DSL → Prefab 节点树', category: 'Cocos / 游戏',
    config: { prompt: '根据 UI 描述生成 Cocos Prefab 节点树结构', timeout_minutes: 20 }},
  { id: 'asset', type: 'agent', name: '资源压缩', icon: '🖼️', desc: '纹理压缩 / Atlas 打包', category: 'Cocos / 游戏',
    config: { prompt: '检查并压缩项目中的图片资源，生成 Sprite Atlas', timeout_minutes: 15 }},
  // 部署
  { id: 'deploy', type: 'code', name: '部署发布', icon: '🚀', desc: 'CI/CD 构建与部署', category: '部署',
    config: { language: 'shell', code: '', timeout_minutes: 30 }},
  { id: 'notify', type: 'webhook', name: '通知', icon: '📢', desc: '钉钉/飞书/Slack 消息', category: '部署',
    config: { method: 'POST', url: '', timeout_minutes: 5 }},
  { id: 'git', type: 'webhook', name: 'Git 操作', icon: '🔀', desc: 'Push / PR / Merge', category: '部署',
    config: { method: 'POST', url: 'https://api.github.com/repos/{owner}/{repo}/git/refs', timeout_minutes: 10 }},
  // AI 编码
  { id: 'claude-code', type: 'code', name: 'Claude Code', icon: '🤖', desc: 'Claude 原生 CLI 编码', category: 'AI 编码',
    config: { language: 'shell', code: 'claude "你的任务描述"', timeout_minutes: 30 }},
  { id: 'cursor-cli', type: 'code', name: 'Cursor CLI', icon: '🔷', desc: 'Cursor Headless (通过 CloudCLI)', category: 'AI 编码',
    config: { language: 'shell', code: 'npx @cloudcli-ai/cloudcli --run "你的任务描述"', timeout_minutes: 30 }},
  { id: 'codex-cli', type: 'code', name: 'Codex CLI', icon: '🧠', desc: 'OpenAI Codex CLI', category: 'AI 编码',
    config: { language: 'shell', code: 'codex "你的任务描述"', timeout_minutes: 30 }},
  { id: 'cloudcli', type: 'code', name: 'CloudCLI', icon: '🌐', desc: '多 CLI Web UI (Claude/Cursor/Codex)', category: 'AI 编码',
    config: { language: 'shell', code: 'npx @cloudcli-ai/cloudcli', timeout_minutes: 60 }},
]

const nodeCategories = [
  { name: '基础', icon: '📋', templates: [] },
  { name: '代码质量', icon: '✅', templates: [] },
  { name: 'Cocos / 游戏', icon: '🎮', templates: [] },
  { name: 'AI 编码', icon: '🤖', templates: [] },
  { name: '部署', icon: '🚀', templates: [] },
]

// 加载动态模板并合并
const loadDynamicTemplates = () => {
  // 1. Load static templates first
  staticNodeTemplates.forEach(t => {
    const cat = nodeCategories.find(c => c.name === t.category)
    if (cat) cat.templates.push(t)
  })

  // 2. Load dynamic templates
  try {
    const saved = localStorage.getItem('node_templates')
    if (saved) {
      const dynamic = JSON.parse(saved)
      dynamic.forEach(t => {
        const tpl = {
          id: t.id,
          type: t.type || 'custom',
          name: t.name,
          icon: t.icon || '🔌',
          desc: t.description || t.desc || '',
          category: t.category || '自定义',
          config: {},
          isDynamic: true,
          schema: t.inputs || [],
          command_template: t.command_template || ''
        }
        
        let cat = nodeCategories.find(c => c.name === tpl.category)
        if (!cat) {
          cat = { name: tpl.category, icon: '📁', templates: [] }
          nodeCategories.push(cat)
        }
        cat.templates.push(tpl)
      })
    }
  } catch (e) { console.error('Failed to load dynamic templates:', e) }
}
loadDynamicTemplates()

// 监听本地存储变化 (来自 NodeTemplateManager)
window.addEventListener('node-templates-updated', () => {
  // 清空旧动态分类
  const dynamicCats = nodeCategories.filter(c => c.name !== '基础' && c.name !== '代码质量' && c.name !== 'Cocos / 游戏' && c.name !== 'AI 编码' && c.name !== '部署')
  dynamicCats.forEach(c => {
    const idx = nodeCategories.indexOf(c)
    if (idx > -1) nodeCategories.splice(idx, 1)
  })
  // 重新加载
  loadDynamicTemplates()
})

// 过滤后的分类（支持搜索）
const filteredCategories = computed(() => {
  if (!nodeSearch.value) return nodeCategories
  const q = nodeSearch.value.toLowerCase()
  return nodeCategories.map(cat => ({
    ...cat,
    templates: cat.templates.filter(t =>
      t.name.toLowerCase().includes(q) || t.desc.toLowerCase().includes(q)
    )
  })).filter(cat => cat.templates.length > 0)
})

// 选中状态
const selectedNodeId = ref(null)

// 执行时节点状态追踪
const nodeStatuses = ref({})
const pollingTimer = ref(null)

// Vue Flow 适配器
const { project, addEdges, findNode } = useVueFlow()

// 转换 Nodes -> VueFlow Node (带执行状态)
const flowNodes = computed(() => {
  return nodes.value.map(n => {
    let config = {}
    try { config = JSON.parse(n.config || '{}') } catch {}
    
    const status = nodeStatuses.value[n.node_id] || ''
    const { bg, text, border, glow } = nodeStyleByStatus(n.type, status)
    
    return {
      id: n.node_id,
      type: 'default',
      label: `${nodeTypeIcon(n.type)} ${sanitizeNodeName(n.name)}${status ? ' [' + statusLabel(status) + ']' : ''}`,
      position: { x: n.position_x || 0, y: n.position_y || 0 },
      style: { 
        background: bg, 
        color: text,
        border: `2px solid ${selectedNodeId.value === n.node_id ? '#3b82f6' : border}`,
        borderRadius: '8px',
        fontWeight: status ? 'bold' : '600',
        fontSize: '12px',
        width: '180px',
         boxShadow: selectedNodeId.value === n.node_id 
          ? '0 0 0 4px rgba(59,130,246,0.2), ' + (glow || '0 2px 4px rgba(0,0,0,0.05)') 
          : (glow || '0 2px 4px rgba(0,0,0,0.05)'),
        transition: 'all 0.3s ease'
      },
      selected: n.selected || false,
      data: { ...n, ...config } // 原始数据挂载在 data 中
    }
  })
})

// 转换 Edges -> VueFlow Edge (执行时连线动画)
const flowEdges = computed(() => {
  const hasRunning = Object.values(nodeStatuses.value).some(s => s === 'running')
  return edges.value.map(e => ({
    id: `e-${e.source_id}-${e.target_id}`,
    source: e.source_id,
    target: e.target_id,
    label: e.condition || '',
    style: { stroke: hasRunning ? '#3b82f6' : '#94a3b8', strokeWidth: hasRunning ? 3 : 2 },
    animated: hasRunning
  }))
})

// 当前选中的节点数据 (双向绑定用)
const selectedNodeData = computed({
  get() {
    if (!selectedNodeId.value) return null
    const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
    if (!n) return null
    let config = {}
    try { config = JSON.parse(n.config || '{}') } catch {}
    return {
      ...n,
      ...config,
      label: n.name,
      id: n.node_id, // VueFlow uses 'id'
      _rawConfig: n.config,
      _timeout: config.timeout_minutes || 30,
      _selectedAgentId: config.agent_id || n.agent_id || ''
    }
  },
  set(val) {
    if (!val || !val.id) return
    const n = nodes.value.find(x => x.node_id === val.id)
    if (!n) return
    
    // 同步回 nodes 数组
    n.name = val.label
    n.type = val.type
    n.agent_id = val.agent_id || ''
    n._timeout = val._timeout
    
    // 重新组装 config JSON
    const cfg = {
      agent_id: val.agent_id || '',
      prompt: val.prompt || '',
      required_skills: val.required_skills || [],
      expression: val.expression || '',
      timeout_minutes: val._timeout || 30,
    }

    // 合并动态 Schema 字段
    if (currentTemplateForNode.value) {
      currentTemplateForNode.value.schema.forEach(field => {
        const k = '_' + field.key
        if (val[k] !== undefined) {
          cfg[field.key] = val[k]
        }
      })
    }
    
    // 保留用户可能手动修改的其他字段
    n.config = JSON.stringify({ ...JSON.parse(val._rawConfig || '{}'), ...cfg })
    
    // 触发后端更新
    updateFlowNode(currentFlow.value.id, n.node_id, n)
  }
})

// 查找当前选中节点对应的模板 (用于渲染动态表单)
const currentTemplateForNode = computed(() => {
  if (!selectedNodeId.value) return null
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return null
  
  // 遍历所有分类查找模板
  for (const cat of nodeCategories) {
    const tpl = cat.templates.find(t => t.id === n.template_id || (t.type === n.type && t.isDynamic))
    if (tpl && tpl.schema && tpl.schema.length > 0) return tpl
  }
  return null
})

// ─── 事件处理 ───

const onNodeClick = (e) => {
  selectedNodeId.value = e.node.id
}

const onNodesChange = (changes) => {
  // 同步位置变化
  changes.forEach(change => {
    if (change.type === 'position' && change.position) {
      const n = nodes.value.find(x => x.node_id === change.id)
      if (n) {
        n.position_x = Math.round(change.position.x)
        n.position_y = Math.round(change.position.y)
      }
    }
  })
}

const onConnect = (params) => {
  // params: { source, target, sourceHandle, targetHandle }
  const edge = {
    source_id: params.source,
    target_id: params.target,
    condition: '',
    flow_id: currentFlow.value.id
  }
  edges.value.push({ ...edge, id: Date.now() })
  createFlowEdge(currentFlow.value.id, edge)
}

const addNode = (type) => {
  addNodeFromTemplate({ type, name: `${type}_${nodes.value.filter(n => n.type === type).length + 1}`, config: {} })
}

const addNodeFromTemplate = (tpl) => {
  const id = `node_${Date.now()}`
  const count = nodes.value.filter(n => n.type === tpl.type).length
  
  // 动态生成 config
  let config = {}
  if (tpl.isDynamic) {
    if (tpl.schema) {
      tpl.schema.forEach(field => {
        config[field.key] = field.default !== undefined ? field.default : ''
      })
    }
    if (tpl.command_template) {
      config.command_template = tpl.command_template
    }
  } else {
    config = tpl.config || {}
  }

  const node = {
    node_id: id,
    name: tpl.name || `${tpl.type}_${count + 1}`,
    type: tpl.type,
    agent_id: '',
    config: JSON.stringify(config),
    template_id: tpl.id, // 记录模板 ID 以便查找
    position_x: 200 + count * 50,
    position_y: 100 + count * 50,
    flow_id: currentFlow.value.id
  }
  createFlowNode(currentFlow.value.id, node)
  nodes.value.push(node)
  selectedNodeId.value = id
}

const deleteNode = async (nid) => {
  // Confirm is handled in template via ElPopconfirm
  await deleteFlowNode(currentFlow.value.id, nid)
  nodes.value = nodes.value.filter(n => n.node_id !== nid)
  edges.value = edges.value.filter(e => e.source_id !== nid && e.target_id !== nid)
  selectedNodeId.value = null
}


// ─── 节点右键菜单 ───

const onNodeContextMenu = (e) => {
  e.event.preventDefault(); // 阻止默认浏览器菜单
  nodeContextMenu.value = {
    visible: true,
    x: e.event.clientX,
    y: e.event.clientY,
    node: e.node
  };
  // 自动选中该节点
  selectedNodeId.value = e.node.id;
};

const closeNodeContextMenu = () => {
  nodeContextMenu.value.visible = false;
};

const handleNodeMenuAction = (action) => {
  if (!nodeContextMenu.value.node) return;
  const node = nodeContextMenu.value.node;
  
  if (action === 'delete') {
    deleteNode(node.id);
  } else if (action === 'edit') {
    selectedNodeId.value = node.id;
  } else if (action === 'copy') {
    const originalNode = nodes.value.find(n => n.node_id === node.id);
    if (originalNode) {
      const id = `node_${Date.now()}`;
      const newNode = {
        node_id: id,
        name: originalNode.name + '_副本',
        type: originalNode.type,
        agent_id: '',
        config: originalNode.config,
        template_id: originalNode.template_id,
        position_x: (originalNode.position_x || 0) + 50,
        position_y: (originalNode.position_y || 0) + 50,
        flow_id: currentFlow.value.id
      };
      createFlowNode(currentFlow.value.id, newNode);
      nodes.value.push(newNode);
      selectedNodeId.value = id;
      ElMessage.success('节点已复制');
    }
  }
  closeNodeContextMenu();
};

// ─── 数据加载 ───

const loadFlows = async () => {
  const { data } = await getFlows()
  flows.value = data
}

const loadAgents = async () => {
  try {
    const { data } = await getTerminals()
    agents.value = data
  } catch (e) { agents.value = [] }
}

// Agent 选择变更
const onAgentChange = () => {
  if (!selectedNodeId.value) return
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return
  const agentId = selectedNodeData.value?._selectedAgentId || ''
  // 更新 config 中的 agent_id
  try {
    const cfg = JSON.parse(n.config || '{}')
    cfg.agent_id = agentId
    n.config = JSON.stringify(cfg)
    n.agent_id = agentId
    updateFlowNode(currentFlow.value.id, n.node_id, n)
  } catch {}
}

const selectFlow = async (f) => {
  currentFlow.value = f
  selectedNodeId.value = null
  nodeStatuses.value = {} // 清空状态
  stopStatusPolling()
  const [nodesRes, edgesRes] = await Promise.all([
    getFlowNodes(f.id),
    getFlowEdges(f.id)
  ])
  nodes.value = nodesRes.data
  edges.value = edgesRes.data
  loadExecutions()
}

const newFlow = async () => {
  try {
    const { value } = await ElMessageBox.prompt('请输入流程名称', '新建流程', {
      confirmButtonText: '创建',
      cancelButtonText: '取消',
      inputPlaceholder: '例如：商业化活动管线',
    })
    if (!value) return
    const { data } = await createFlow({ name: value, description: '', type: 'dag', status: 'draft', graph: '{}' })
    currentFlow.value = data
    nodes.value = []
    edges.value = []
    loadFlows()
    ElMessage.success('流程创建成功')
  } catch {
    // cancelled
  }
}

const deleteFlow = async (id) => {
  // Confirm is handled in template via ElPopconfirm
  await deleteFlowApi(id)
  currentFlow.value = null
  loadFlows()
}

const saveFlowMeta = async () => {
  await updateFlow(currentFlow.value.id, currentFlow.value)
}

const saveGraph = async () => {
  await saveFlowGraph(currentFlow.value.id, {
    graph: JSON.stringify({ nodes: nodes.value, edges: edges.value }),
    nodes: nodes.value,
    edges: edges.value
  })
  ElMessage.success('保存成功')
}

// ─── 执行与验证 ───

const validateFlow = async () => {
  try {
    const { data } = await apiValidateFlow(currentFlow.value.id)
    if (data.valid) {
      ElMessage.success('✅ 流程结构验证通过')
    } else {
      ElMessage.error('❌ ' + data.error)
    }
  } catch (e) {
    ElMessage.error('❌ 验证失败: ' + (e.response?.data?.error || e.message))
  }
}

const executeCurrentFlow = async () => {
  // Confirm is handled in template via ElPopconfirm
  executing.value = true
  nodeStatuses.value = {} // 清空旧状态
  try {
    await executeFlow(currentFlow.value.id)
    // 获取最新执行记录，开始轮询
    await loadExecutions()
    if (executions.value.length > 0) {
      startStatusPolling(executions.value[0].id)
    }
  } catch (e) {
    ElMessage.error('执行失败: ' + (e.response?.data?.error || e.message))
  } finally {
    executing.value = false
  }
}

const loadExecutions = async () => {
  if (!currentFlow.value) return
  const { data } = await getFlowExecutions(currentFlow.value.id)
  executions.value = data
}

const showExecDetail = async (exec) => {
  currentExec.value = exec
  const { data } = await getExecutionSteps(exec.id)
  execSteps.value = data
  showDetail.value = true
}

const cancelExec = async (id) => {
  await cancelExecution(id)
  showDetail.value = false
  await loadExecutions()
}

// ─── UI 辅助 ───

// 节点执行状态样式
const statusLabel = (s) => ({ running: '⏳运行中', completed: '✅完成', failed: '❌失败', skipped: '⏭跳过', pending: '⏸等待' }[s] || '')
const nodeStyleByStatus = (type, status) => {
  const base = { agent: { bg: '#dbeafe', text: '#1e40af' }, condition: { bg: '#fef9c3', text: '#854d0e' }, merge: { bg: '#dcfce7', text: '#166534' }, trigger: { bg: '#f3e8ff', text: '#6b21a8' }, webhook: { bg: '#ffedd5', text: '#9a3412' }, code: { bg: '#f3f4f6', text: '#374151' } }[type] || { bg: '#f1f5f9', text: '#475569' }
  switch (status) {
    case 'running':  return { ...base, border: '#3b82f6', glow: '0 0 12px rgba(59,130,246,0.6), 0 2px 4px rgba(0,0,0,0.1)' }
    case 'completed': return { ...base, border: '#22c55e', glow: '0 0 8px rgba(34,197,94,0.4)' }
    case 'failed':   return { bg: '#fee2e2', text: '#dc2626', border: '#ef4444', glow: '0 0 8px rgba(239,68,68,0.4)' }
    case 'skipped':  return { bg: '#f1f5f9', text: '#94a3b8', border: '#cbd5e1', glow: '' }
    default:         return { ...base, border: '#cbd5e1', glow: '' }
  }
}

// 轮询执行状态
const startStatusPolling = (execId) => {
  stopStatusPolling()
  pollingTimer.value = setInterval(async () => {
    try {
      const { data } = await getExecutionSteps(execId)
      const newStatuses = {}
      for (const step of data) {
        newStatuses[step.node_id] = step.status
      }
      // 只在状态变化时更新
      const changed = Object.keys(newStatuses).some(k => nodeStatuses.value[k] !== newStatuses[k])
      if (changed) nodeStatuses.value = newStatuses
      
      // 检查是否全部完成
      const allDone = data.length > 0 && data.every(s => ['completed', 'failed', 'skipped'].includes(s.status))
      if (allDone) {
        stopStatusPolling()
        await loadExecutions()
      }
    } catch (e) { /* ignore */ }
  }, 1000)
}

const stopStatusPolling = () => {
  if (pollingTimer.value) { clearInterval(pollingTimer.value); pollingTimer.value = null }
}

const nodeColor = (type) => {
  return { agent: '#dbeafe', condition: '#fef9c3', merge: '#dcfce7', trigger: '#f3e8ff', webhook: '#ffedd5', code: '#f3f4f6' }[type] || '#f1f5f9'
}
const nodeTextColor = (type) => {
  return { agent: '#1e40af', condition: '#854d0e', merge: '#166534', trigger: '#6b21a8', webhook: '#9a3412', code: '#374151' }[type] || '#475569'
}
const nodeTypeIcon = (type) => {
  return { agent: '🤖', condition: '🔀', merge: '🔗', trigger: '⚡', webhook: '🔌', code: '💻' }[type] || '📦'
}

// 清理可能混入 SVG 代码的节点名称（历史 Bug 修复）
const sanitizeNodeName = (name) => {
  if (!name) return '未命名节点'
  if (typeof name !== 'string') return String(name)
  // 如果名称包含 <svg 标签，说明数据损坏，重置
  if (name.includes('<svg') || name.includes('viewBox')) {
    return '已修复节点'
  }
  return name
}
const flowStatusClass = (s) => {
  return { draft: 'bg-gray-100 text-gray-600', active: 'bg-green-100 text-green-700', archived: 'bg-yellow-100 text-yellow-700' }[s] || ''
}
const formatTime = (t) => t ? new Date(t).toLocaleString('zh-CN') : '-'
const execStatusBadge = (s) => {
  return { running: 'bg-blue-100 text-blue-700', completed: 'bg-green-100 text-green-700', failed: 'bg-red-100 text-red-700', cancelled: 'bg-gray-200 text-gray-600', waiting: 'bg-yellow-100 text-yellow-700' }[s] || 'bg-gray-100'
}
const stepStatusBadge = (s) => {
  return { running: 'bg-blue-100 text-blue-700', completed: 'bg-green-100 text-green-700', failed: 'bg-red-100 text-red-700', skipped: 'bg-gray-200 text-gray-600' }[s] || 'bg-gray-100'
}
const stepStatusColor = (s) => {
  return { running: 'border-blue-300 bg-blue-50', completed: 'border-green-300 bg-green-50', failed: 'border-red-300 bg-red-50', skipped: 'border-gray-200 bg-gray-50' }[s] || ''
}
const stepTypeIcon = (t) => {
  const icons = {
    agent: '<svg class="w-5 h-5 inline" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="10" rx="2"></rect><circle cx="12" cy="5" r="2"></circle><path d="M12 7v4"></path></svg>',
    condition: '<svg class="w-5 h-5 inline" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>',
    merge: '<svg class="w-5 h-5 inline" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="5" r="3"></circle><circle cx="6" cy="12" r="3"></circle><circle cx="18" cy="19" r="3"></circle><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line></svg>',
    trigger: '<svg class="w-5 h-5 inline" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>',
  }
  return icons[t] || '📦'
}


// ─── 交互增强 ───

const showContextMenu = (e) => {
  // Only show on canvas background (vue-flow__pane)
  if (e.target.closest('.vue-flow__pane') && !e.target.closest('.vue-flow__node')) {
    contextMenu.value = { show: true, x: e.clientX, y: e.clientY }
  }
}

const hideContextMenu = () => {
  contextMenu.value.show = false
}

// Close menu on click outside
const handleGlobalClick = () => hideContextMenu()

const arrangeNodes = () => {
  if (nodes.value.length === 0) return
  
  // Simple Topological Sort Layout
  const inDegree = {}
  nodes.value.forEach(n => { inDegree[n.node_id] = 0 })
  edges.value.forEach(e => {
    if (inDegree[e.target_id] !== undefined) inDegree[e.target_id]++
  })
  
  const queue = []
  nodes.value.forEach(n => {
    if (inDegree[n.node_id] === 0) {
      queue.push(n.node_id)
    }
  })
  
  const levels = {}
  queue.forEach(id => levels[id] = 0)
  
  let idx = 0
  while(idx < queue.length){
    const u = queue[idx++]
    edges.value.forEach(e => {
      if (e.source_id === u) {
        inDegree[e.target_id]--
        if (inDegree[e.target_id] === 0) {
          levels[e.target_id] = Math.max(levels[e.target_id] || 0, levels[u] + 1)
          queue.push(e.target_id)
        }
      }
    })
  }
  
  // Assign levels for unconnected/cycle nodes
  nodes.value.forEach(n => {
    if (levels[n.node_id] === undefined) levels[n.node_id] = 0
  })
  
  // Group by level
  const grouped = {}
  nodes.value.forEach(n => {
    const l = levels[n.node_id]
    if (!grouped[l]) grouped[l] = []
    grouped[l].push(n)
  })
  
  // Coordinates
  const startX = 50
  const startY = 50
  const gapX = 280
  const gapY = 120
  
  const sortedLevels = Object.keys(grouped).map(Number).sort((a,b) => a-b)
  
  sortedLevels.forEach(level => {
    const col = grouped[level]
    col.forEach((n, i) => {
      n.position_x = startX + level * gapX
      n.position_y = startY + i * gapY
      // Save to backend
      updateFlowNode(currentFlow.value.id, n.node_id, n)
    })
  })
  
  nodes.value = [...nodes.value] // Trigger reactivity
  ElMessage.success('节点已重新排列')
}

const deleteSelectedNodes = async () => {
  const selected = nodes.value.filter(n => n.selected)
  if (selected.length === 0) {
    ElMessage.info('请先选中要删除的节点')
    return
  }
  
  for (const n of selected) {
    await deleteFlowNode(currentFlow.value.id, n.node_id)
  }
  
  nodes.value = nodes.value.filter(n => !n.selected)
  edges.value = edges.value.filter(e => !selected.some(n => n.node_id === e.source_id || n.node_id === e.target_id))
  selectedNodeId.value = null
  ElMessage.success(`已删除 ${selected.length} 个节点`)
}

const handleKeyDown = (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
    if (['INPUT', 'TEXTAREA'].includes(document.activeElement.tagName)) return
    e.preventDefault()
    nodes.value.forEach(n => n.selected = true)
    nodes.value = [...nodes.value] // Trigger reactivity
  }
  if (e.key === 'Delete' || e.key === 'Backspace') {
    // Allow deleting selected nodes with Delete key
    if (['INPUT', 'TEXTAREA'].includes(document.activeElement.tagName)) return
    const selected = nodes.value.filter(n => n.selected)
    if (selected.length > 0) {
      e.preventDefault()
      deleteSelectedNodes()
    }
  }
}


onMounted(() => { 
  loadFlows(); loadAgents()
  window.addEventListener('keydown', handleKeyDown)
  window.addEventListener('click', handleGlobalClick)
})

onUnmounted(() => {
  stopStatusPolling()
  window.removeEventListener('keydown', handleKeyDown)
  window.removeEventListener('click', handleGlobalClick)
})
</script>

<style>
.vue-flow__edge-path { stroke: #cbd5e1; stroke-width: 2px; }

/* 运行中节点脉冲动画 */
@keyframes pulse-running {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.8; }
}
.vue-flow__node { transition: all 0.3s ease; }

/* 简单的淡入动画 */
@keyframes fade-in {
  from { opacity: 0; transform: scale(0.95); }
  to { opacity: 1; transform: scale(1); }
}
.animate-fade-in {
  animation: fade-in 0.1s ease-out;
}

</style>
