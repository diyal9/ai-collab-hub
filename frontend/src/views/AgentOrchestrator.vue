<!-- DEBUG: TEST BUILD -->
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
            <label class="block text-xs text-gray-500 mb-1">节点 ID</label>
            <div class="w-full border rounded px-2 py-1 text-xs font-mono bg-gray-100 text-gray-600 select-all" title="用于变量引用：${node_id}">{{ selectedNodeData.node_id }}</div>
          </div>
          <div>
            <label class="block text-xs text-gray-500 mb-1">名称</label>
            <input :value="selectedNodeData.label" @input="saveNodeName($event.target.value)" class="w-full border rounded px-2 py-1 text-sm" />
          </div>
          <div>
            <label class="block text-xs text-gray-500 mb-1">类型</label>
            <select v-model="selectedNodeData.type" class="w-full border rounded px-2 py-1 text-sm">
              <option value="agent">Agent</option>
              <option value="sandbox">Sandbox</option>
              <option value="condition">Condition</option>
              <option value="merge">Merge</option>
              <option value="trigger">Trigger</option>
              <option value="webhook">Webhook</option>
              <option value="approval">Approval (人在回路)</option>
              <option value="code">Code</option>
            </select>
          </div>
          <!-- ─── Agent 节点增强配置 ─── -->
          <div v-if="selectedNodeData.type === 'agent'" class="space-y-3">
            <div>
              <label class="block text-xs text-gray-500 mb-1">角色 / System</label>
              <div class="text-xs text-gray-600 bg-gray-50 p-2 rounded border truncate h-8 flex items-center" :title="agentSummary.role">{{ agentSummary.role || '未配置' }}</div>
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-1">任务 / Goal</label>
              <div class="text-xs text-gray-600 bg-gray-50 p-2 rounded border h-12 overflow-auto whitespace-pre-wrap mb-1">{{ agentSummary.instruction || '未配置' }}</div>
              <button @click="openAgentEditor" class="w-full bg-purple-50 text-purple-600 hover:bg-purple-100 border border-purple-200 py-2 rounded text-sm font-medium flex items-center justify-center gap-1.5 transition-all shadow-sm">
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"></path></svg>
                编辑结构化 Prompt
              </button>
            </div>
            <div v-if="agentSummary.constraints" class="mt-1">
               <label class="block text-[10px] text-gray-400 mb-0.5">约束 / Constraints</label>
               <div class="text-[10px] text-gray-500 bg-gray-50 p-1.5 rounded border h-8 overflow-auto whitespace-pre-wrap">{{ agentSummary.constraints }}</div>
            </div>
            <div v-if="agentSummary.format" class="mt-1">
               <label class="block text-[10px] text-gray-400 mb-0.5">输出格式 / Output</label>
               <div class="text-[10px] text-gray-500 bg-gray-50 p-1.5 rounded border h-8 overflow-auto whitespace-pre-wrap">{{ agentSummary.format }}</div>
            </div>
          </div>
          <div>
            <label class="block text-xs text-gray-500 mb-1">超时 (分钟)</label>
            <input v-model.number="selectedNodeData._timeout" type="number" class="w-full border rounded px-2 py-1 text-sm" min="1" max="120" />
          </div>
          <!-- ─── Sandbox 节点增强配置 ─── -->
          <div v-if="selectedNodeData.type === 'sandbox'" class="space-y-3">
            <div>
              <label class="block text-xs text-gray-500 mb-1">Sandbox ID</label>
              <select :value="selectedNodeData.sandbox_id" @change="updateNodeField('sandbox_id', $event.target.value)" class="w-full border rounded px-2 py-1 text-sm">
                <option value="">-- 选择 Sandbox --</option>
                <option v-for="sb in onlineSandboxes" :key="sb.id" :value="sb.name">{{ sb.name }} ({{ sb.status }})</option>
                <option v-if="selectedNodeData.sandbox_id && !onlineSandboxes.find(s=>s.name===selectedNodeData.sandbox_id)" :value="selectedNodeData.sandbox_id">{{ selectedNodeData.sandbox_id }} (离线)</option>
              </select>
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-1">执行器</label>
              <select :value="selectedNodeData.executor" @change="updateNodeField('executor', $event.target.value)" class="w-full border rounded px-2 py-1 text-sm">
                <option value="shell">Shell</option>
                <option value="codex">Codex</option>
                <option value="cursor">Cursor</option>
                <option value="aider">Aider</option>
                <option value="claude">Claude Code</option>
              </select>
            </div>
            <div class="grid grid-cols-2 gap-2">
               <div>
                 <label class="block text-xs text-gray-500 mb-1">Git 分支</label>
                 <input :value="selectedNodeData.git_branch" @input="updateNodeField('git_branch', $event.target.value)" class="w-full border rounded px-2 py-1 text-sm" placeholder="develop" />
               </div>
               <div>
                 <label class="block text-xs text-gray-500 mb-1">工作目录</label>
                 <input :value="selectedNodeData.work_dir" @input="updateNodeField('work_dir', $event.target.value)" class="w-full border rounded px-2 py-1 text-sm" placeholder="/tmp/agent-workspace" />
               </div>
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-1">Git 仓库地址</label>
              <input :value="selectedNodeData.git_url" @input="updateNodeField('git_url', $event.target.value)" class="w-full border rounded px-2 py-1 text-sm" placeholder="git@gitlab.com:team/repo.git" />
            </div>
            
            <!-- 命令 / Prompt 概览 -->
            <label class="block text-xs text-gray-500 mb-1">命令 / Prompt</label>
            <div class="bg-gray-50 p-2 rounded border text-xs font-mono h-16 overflow-auto whitespace-pre-wrap mb-1">{{ selectedNodeData.command || '未配置' }}</div>
            <div class="flex gap-2">
              <button @click="openCommandEditor" class="flex-1 bg-blue-50 text-blue-600 hover:bg-blue-100 border border-blue-200 py-1.5 rounded text-xs font-medium flex items-center justify-center gap-1 transition-all">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"></path></svg>
                编辑命令
              </button>
            </div>
            <p class="text-xs text-gray-400 mt-1">💡 支持变量引用、模板加载</p>
          </div>
          
          <!-- Approval 节点配置 -->
          <!-- ─── Webhook 节点增强配置 (概览 + 弹窗编辑) ─── -->
          <div v-if="selectedNodeData.type === 'webhook'" class="space-y-3">
            <div>
              <label class="block text-xs text-gray-500 mb-1">Webhook URL</label>
              <div class="text-xs text-gray-600 bg-gray-50 p-2 rounded border truncate" :title="selectedNodeData.url">
                {{ selectedNodeData.url || '未配置' }}
              </div>
            </div>
            
            <div class="flex gap-2">
              <div class="flex-1 bg-gray-50 p-2 rounded border text-center cursor-pointer hover:bg-blue-50 transition-colors" @click="openWebhookEditor('headers')">
                <span class="text-[10px] text-gray-400 block">Headers</span>
                <span class="text-xs font-medium text-blue-600">{{ webhookSummary.headers }}</span>
              </div>
              <div class="flex-1 bg-gray-50 p-2 rounded border text-center cursor-pointer hover:bg-blue-50 transition-colors" @click="openWebhookEditor('body')">
                <span class="text-[10px] text-gray-400 block">Body</span>
                <span class="text-xs font-medium text-blue-600">{{ webhookSummary.body }}</span>
              </div>
            </div>

            <button @click="openWebhookEditor()" class="w-full bg-blue-50 text-blue-600 hover:bg-blue-100 border border-blue-200 py-2 rounded text-sm font-medium flex items-center justify-center gap-1.5 transition-all shadow-sm">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"></path></svg>
              编辑完整配置
            </button>
          </div>
          
          <!-- Approval 节点配置 -->
          <div v-if="selectedNodeData.type === 'approval'" class="space-y-3">
            <div>
              <label class="block text-xs text-gray-500 mb-1">飞书 Webhook URL</label>
              <div class="text-xs text-gray-600 bg-gray-50 p-2 rounded border truncate" :title="selectedNodeData.webhook_url">{{ selectedNodeData.webhook_url || '未配置' }}</div>
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-1">卡片标题</label>
              <div class="text-xs text-gray-600 bg-gray-50 p-2 rounded border">{{ selectedNodeData.card_title || '未配置' }}</div>
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-1">卡片内容</label>
              <div class="bg-gray-50 p-2 rounded border text-xs h-20 overflow-auto whitespace-pre-wrap">{{ selectedNodeData.card_body || '未配置' }}</div>
              <button @click="openApprovalEditor" class="w-full mt-2 bg-blue-50 text-blue-600 hover:bg-blue-100 border border-blue-200 py-1.5 rounded text-xs font-medium flex items-center justify-center gap-1 transition-all">
                <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"></path></svg>
                编辑内容与预览
              </button>
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-1">审批超时 (分钟)</label>
              <input :value="selectedNodeData.timeout_minutes || 1440" @input="updateNodeField('timeout_minutes', Number($event.target.value))" class="w-full border rounded px-2 py-1 text-sm" type="number" min="1" />
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
      class="fixed bottom-0 left-16 md:left-52 right-0 z-40 bg-white border-t border-gray-200 shadow-xl transition-all duration-300"
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

  <!-- ─── Webhook 配置弹窗 ─── -->
  <el-dialog v-model="webhookDialogVisible" title="Webhook 配置" width="850px" :close-on-click-modal="false" top="5vh">
    <div class="space-y-4">
      <!-- URL -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">请求 URL</label>
        <input v-model="tempWebhookUrl" class="w-full border rounded px-3 py-2 text-sm font-mono bg-gray-50 focus:bg-white focus:ring-2 focus:ring-blue-500 transition-all" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..." />
      </div>
      
      <el-tabs v-model="webhookActiveTab">
        <!-- Headers Tab -->
        <el-tab-pane label="Headers" name="headers">
          <div class="flex justify-between items-center mb-2">
             <span class="text-xs text-gray-400">支持 ${node_xxx} 变量引用</span>
             <div class="flex gap-2">
               <button @click="formatJson('headers')" class="text-xs text-blue-600 hover:text-blue-800 font-medium flex items-center gap-1">
                 <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 6h16M4 12h16M4 18h16"></path></svg> 格式化
               </button>
             </div>
          </div>
          <div class="relative">
            <textarea ref="headersTextarea" v-model="tempHeadersJson" @input="validateJson('headers')" class="w-full border rounded px-3 py-2 text-xs font-mono bg-gray-50 h-40 focus:bg-white focus:ring-2 focus:ring-blue-500 transition-all pr-12"></textarea>
            <div class="absolute right-2 top-2">
              <span v-if="jsonErrors.headers" class="text-red-500 text-xs bg-white px-1 rounded shadow">❌</span>
              <span v-else-if="tempHeadersJson.trim()" class="text-green-500 text-xs bg-white px-1 rounded shadow">✅</span>
            </div>
          </div>
          <div v-if="jsonErrors.headers" class="text-xs text-red-500 mt-1">{{ jsonErrors.headers }}</div>
          
          <div class="mt-3">
             <span class="text-xs text-gray-500">快捷插入变量:</span>
             <div class="flex flex-wrap gap-1.5 mt-1.5">
               <button v-for="v in availableVariables" :key="v" @click="insertVar('headers', v)" class="text-[10px] bg-gray-100 hover:bg-blue-100 px-2 py-0.5 rounded border text-blue-600 transition-colors">{{v}}</button>
             </div>
          </div>
        </el-tab-pane>

        <!-- Body Tab -->
        <el-tab-pane label="Body" name="body">
          <div class="flex justify-between items-center mb-2">
             <span class="text-xs text-gray-400">支持 ${node_xxx} 变量引用</span>
             <button @click="formatJson('body')" class="text-xs text-blue-600 hover:text-blue-800 font-medium flex items-center gap-1">
               <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 6h16M4 12h16M4 18h16"></path></svg> 格式化
             </button>
          </div>
          <div class="relative">
            <textarea ref="bodyTextarea" v-model="tempBodyJson" @input="validateJson('body')" class="w-full border rounded px-3 py-2 text-xs font-mono bg-gray-50 h-60 focus:bg-white focus:ring-2 focus:ring-blue-500 transition-all pr-12"></textarea>
            <div class="absolute right-2 top-2">
              <span v-if="jsonErrors.body" class="text-red-500 text-xs bg-white px-1 rounded shadow">❌</span>
              <span v-else-if="tempBodyJson.trim()" class="text-green-500 text-xs bg-white px-1 rounded shadow">✅</span>
            </div>
          </div>
          <div v-if="jsonErrors.body" class="text-xs text-red-500 mt-1">{{ jsonErrors.body }}</div>

          <div class="mt-3">
             <span class="text-xs text-gray-500">快捷插入变量:</span>
             <div class="flex flex-wrap gap-1.5 mt-1.5">
               <button v-for="v in availableVariables" :key="v" @click="insertVar('body', v)" class="text-[10px] bg-gray-100 hover:bg-blue-100 px-2 py-0.5 rounded border text-blue-600 transition-colors">{{v}}</button>
             </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="webhookDialogVisible = false">取消</el-button>
        <el-button type="primary" :disabled="!!jsonErrors.headers || !!jsonErrors.body" @click="saveWebhookConfig">保存配置</el-button>
      </span>
    </template>
  </el-dialog>

  <!-- ─── Sandbox 命令弹窗 ─── -->
  <el-dialog v-model="commandDialogVisible" title="Sandbox 命令 / Prompt 编辑" width="900px" :close-on-click-modal="false" top="5vh">
    <div class="flex flex-col h-[60vh]">
      <div class="flex-1 flex gap-4">
        <!-- 编辑区 -->
        <div class="flex-1 flex flex-col">
          <div class="flex justify-between items-center mb-2">
             <span class="text-sm font-medium text-gray-700">命令内容</span>
             <div class="flex gap-2">
               <select @change="onTemplateSelectFromDialog($event)" class="border rounded px-2 py-1 text-xs bg-white">
                 <option value="">📄 加载模板...</option>
                 <option v-for="t in promptTemplates" :key="t.id" :value="t.name">{{ t.name }} (v{{t.version}})</option>
               </select>
             </div>
          </div>
          <textarea ref="commandTextarea" v-model="tempCommand" @input="checkCommandVariables" class="flex-1 w-full border rounded px-3 py-2 text-xs font-mono bg-gray-50 focus:bg-white focus:ring-2 focus:ring-blue-500 transition-all resize-none"></textarea>
          <div class="mt-2 flex gap-1.5 flex-wrap">
            <span class="text-xs text-gray-500 py-1">插入变量:</span>
            <button v-for="v in availableVariables" :key="v" @click="insertVar('command', v)" class="text-[10px] bg-gray-100 hover:bg-blue-100 px-2 py-0.5 rounded border text-blue-600 transition-colors">{{v}}</button>
          </div>
        </div>

        <!-- 预览区 -->
        <div class="w-80 border-l pl-4 flex flex-col">
          <span class="text-sm font-medium text-gray-700 mb-2">变量替换预览</span>
          <div class="flex-1 bg-gray-50 rounded p-3 text-xs font-mono whitespace-pre-wrap overflow-auto border">
            <div v-if="commandPreview" class="text-gray-700">{{ commandPreview }}</div>
            <div v-else class="text-gray-400 italic">输入命令后显示预览...</div>
          </div>
        </div>
      </div>
    </div>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="commandDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveCommandConfig">保存</el-button>
      </span>
    </template>
  </el-dialog>

  <!-- ─── Approval Markdown 预览弹窗 ─── -->
  <el-dialog v-model="approvalDialogVisible" title="审批卡片内容 (Markdown)" width="1000px" :close-on-click-modal="false" top="5vh">
    <div class="flex gap-4 h-[60vh]">
      <div class="flex-1 flex flex-col">
        <span class="text-sm font-medium text-gray-700 mb-2">编辑</span>
        <textarea ref="approvalTextarea" v-model="tempApprovalBody" class="flex-1 w-full border rounded px-3 py-2 text-xs font-mono bg-gray-50 focus:bg-white focus:ring-2 focus:ring-blue-500 transition-all resize-none"></textarea>
        <div class="mt-2 flex gap-1.5 flex-wrap">
          <span class="text-xs text-gray-500 py-1">插入变量:</span>
          <button v-for="v in availableVariables" :key="v" @click="insertVar('approval', v)" class="text-[10px] bg-gray-100 hover:bg-blue-100 px-2 py-0.5 rounded border text-blue-600 transition-colors">{{v}}</button>
        </div>
      </div>
      <div class="flex-1 border-l pl-4 flex flex-col">
        <span class="text-sm font-medium text-gray-700 mb-2">实时预览</span>
        <div class="flex-1 bg-white rounded p-4 overflow-auto border prose prose-sm max-w-none" v-html="renderedMarkdown"></div>
      </div>
    </div>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="approvalDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveApprovalConfig">保存</el-button>
      </span>
    </template>
  </el-dialog>

  <!-- ─── Agent 结构化 Prompt 弹窗 ─── -->
  <el-dialog v-model="agentDialogVisible" title="Agent 结构化 Prompt 配置" width="1000px" :close-on-click-modal="false" top="5vh">
    <div class="flex gap-4 h-[65vh]">
      <!-- 左侧编辑区 -->
      <div class="flex-1 flex flex-col space-y-4 overflow-y-auto pr-2">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">
            <span class="text-purple-600 mr-1">👤</span> 角色设定 (System)
          </label>
          <p class="text-xs text-gray-400 mb-1">定义 AI 的身份、背景知识和语气风格。</p>
          <textarea v-model="tempAgentRole" class="w-full border rounded px-3 py-2 text-xs font-mono bg-gray-50 h-20 focus:bg-white focus:ring-2 focus:ring-blue-500 transition-all resize-none"></textarea>
        </div>
        
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">
            <span class="text-blue-600 mr-1">🎯</span> 任务目标 (Instruction)
          </label>
          <p class="text-xs text-gray-400 mb-1">具体的执行步骤和核心目标。</p>
          <textarea ref="agentTextarea" v-model="tempAgentInstruction" @input="updateAgentPreview" class="w-full border rounded px-3 py-2 text-xs font-mono bg-gray-50 h-32 focus:bg-white focus:ring-2 focus:ring-blue-500 transition-all resize-none"></textarea>
          <div class="mt-1 flex gap-1.5 flex-wrap">
            <span class="text-xs text-gray-500 py-0.5">插入变量:</span>
            <button v-for="v in availableVariables" :key="v" @click="insertVar('agent_instruction', v)" class="text-[10px] bg-gray-100 hover:bg-blue-100 px-1.5 py-0.5 rounded border text-blue-600 transition-colors">{{v}}</button>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3">
           <div>
             <label class="block text-xs font-medium text-gray-700 mb-1">
               <span class="text-red-600 mr-1">⛔</span> 约束 / Constraints
             </label>
             <p class="text-[10px] text-gray-400 mb-1">禁止的行为或限制条件。</p>
             <textarea v-model="tempAgentConstraints" class="w-full border rounded px-2 py-1 text-xs font-mono bg-gray-50 h-20 focus:bg-white focus:ring-2 focus:ring-blue-500 transition-all resize-none"></textarea>
           </div>
           <div>
             <label class="block text-xs font-medium text-gray-700 mb-1">
               <span class="text-green-600 mr-1">📦</span> 输出格式 / Output
             </label>
             <p class="text-[10px] text-gray-400 mb-1">期望的返回格式 (JSON, Markdown 等)。</p>
             <textarea v-model="tempAgentFormat" class="w-full border rounded px-2 py-1 text-xs font-mono bg-gray-50 h-20 focus:bg-white focus:ring-2 focus:ring-blue-500 transition-all resize-none"></textarea>
           </div>
        </div>
      </div>

      <!-- 右侧预览区 -->
      <div class="w-96 border-l pl-4 flex flex-col">
        <span class="text-sm font-medium text-gray-700 mb-2 flex justify-between">
          完整 Prompt 预览
          <button @click="formatAgentPreview" class="text-xs text-blue-600 hover:underline">Markdown 格式化</button>
        </span>
        <div class="flex-1 bg-gray-900 text-green-400 rounded p-3 text-[11px] font-mono whitespace-pre-wrap overflow-auto shadow-inner">
          {{ agentPreviewText }}
        </div>
        <div class="mt-2 text-[10px] text-gray-500 flex justify-between">
           <span>最终长度: {{ agentPreviewText.length }} 字符</span>
           <span v-if="agentTemplateMatch" class="text-blue-500">已加载模板: {{ agentTemplateMatch }}</span>
        </div>
      </div>
    </div>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="agentDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveAgentConfig">保存配置</el-button>
      </span>
    </template>
  </el-dialog>
</template>
<script setup>
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import { marked } from 'marked'
import {
  executeFlow, getFlowExecutions, getExecutionSteps, cancelExecution, validateFlow as apiValidateFlow,
  getFlows, createFlow, updateFlow, deleteFlow as deleteFlowApi,
  getFlowNodes, createFlowNode, updateFlowNode, deleteFlowNode,
  getFlowEdges, createFlowEdge, deleteFlowEdge, saveFlowGraph,
  getTerminals,
  getPromptTemplates, previewPrompt,
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
const onlineSandboxes = computed(() => agents.value.filter(a => a.status === 'connected'))

// Prompt 模板
const promptTemplates = ref([])
const loadPromptTemplates = async () => {
  try {
    const { data } = await getPromptTemplates()
    promptTemplates.value = data
  } catch {}
}

// 命令预览与试运行
const showCommandPreview = ref(false)
const previewResolvedText = ref('')
const trialRunning = ref(false)
const trialOutput = ref('')

// Webhook JSON 编辑器状态
const webhookMode = ref({ headers: 'json', body: 'json' })
const webhookForm = ref({ headers: [], body: [] })
const webhookJson = ref({ headers: '', body: '' })
const webhookJsonError = ref({ headers: '', body: '' })

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
    sandbox: '<svg class="w-4 h-4 text-gray-500 group-hover:text-blue-500 transition-colors" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2"/><path d="M4 12h16"/><path d="M12 4v16"/></svg>',
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
  { id: 'sandbox', type: 'sandbox', name: '沙箱执行 (Sandbox)', icon: '🔒', desc: '通过 Bridge 调度远程 Agent', category: '基础',
    config: { sandbox_id: 'sandbox-node-1', executor: 'shell', command: '', work_dir: '/tmp', timeout_minutes: 10 }},
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

    // Sandbox 字段
    if (val.type === 'sandbox') {
      cfg.sandbox_id = val.sandbox_id || ''
      cfg.executor = val.executor || 'shell'
      cfg.command = val.command || ''
      cfg.work_dir = val.work_dir || ''
    }
    // Code 字段
    if (val.type === 'code') {
      cfg.language = val.language || 'python'
      cfg.code = val.code || ''
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
    
    // Webhook 节点：直接使用 _rawConfig，不覆盖
    if (val.type === 'webhook') {
      n.config = val._rawConfig || '{}'
    } else {
      // 保留用户可能手动修改的其他字段
      n.config = JSON.stringify({ ...JSON.parse(val._rawConfig || '{}'), ...cfg })
    }
    
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

// 显式更新节点字段（避免 computed setter v-model 不触发的问题）
const updateNodeField = (field, value) => {
  if (!selectedNodeId.value || !currentFlow.value) return
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return

  // 更新本地 config
  let cfg = {}
  try { cfg = JSON.parse(n.config || '{}') } catch {}
  cfg[field] = value
  n.config = JSON.stringify(cfg)

  // 同步到后端
  updateFlowNode(currentFlow.value.id, n.node_id, n)
}

// 保存节点名称（name 字段不在 config 内，需要单独处理）
const saveNodeName = (value) => {
  if (!selectedNodeId.value || !currentFlow.value) return
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return

  n.name = value
  updateFlowNode(currentFlow.value.id, n.node_id, n)
}

// 更新 Webhook 节点的完整 JSON 配置
const updateRawConfig = (e) => {
  if (!selectedNodeId.value || !currentFlow.value) return
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return
  
  n.config = e.target.value
  updateFlowNode(currentFlow.value.id, n.node_id, n)
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
      let msg = '✅ 流程验证通过'
      if (data.message) msg += `\n${data.message}`
      if (data.warnings && data.warnings.length > 0) {
        msg += `\n\n⚠️ 警告:\n${data.warnings.join('\n')}`
      }
      ElMessage.success({ message: msg, duration: 5000 })
    } else {
      let errMsg = '❌ 验证失败'
      if (data.errors) errMsg += `:\n${data.errors.join('\n')}`
      else if (data.error) errMsg += `: ${data.error}`
      if (data.warnings && data.warnings.length > 0) {
        errMsg += `\n\n⚠️ 警告:\n${data.warnings.join('\n')}`
      }
      ElMessage.error({ message: errMsg, duration: 8000 })
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
const statusLabel = (s) => ({ running: '⏳运行中', completed: '✅完成', failed: '❌失败', skipped: '⏭跳过', pending: '⏸等待', waiting_approval: '⏳等待审批' }[s] || '')
const nodeStyleByStatus = (type, status) => {
  const base = { agent: { bg: '#dbeafe', text: '#1e40af' }, condition: { bg: '#fef9c3', text: '#854d0e' }, merge: { bg: '#dcfce7', text: '#166534' }, trigger: { bg: '#f3e8ff', text: '#6b21a8' }, webhook: { bg: '#ffedd5', text: '#9a3412' }, code: { bg: '#f3f4f6', text: '#374151' }, sandbox: { bg: '#e0f2fe', text: '#0369a1' }, approval: { bg: '#fee2e2', text: '#dc2626' } }[type] || { bg: '#f1f5f9', text: '#475569' }
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
  return { agent: '#dbeafe', condition: '#fef9c3', merge: '#dcfce7', trigger: '#f3e8ff', webhook: '#ffedd5', code: '#f3f4f6', sandbox: '#e0f2fe', approval: '#fee2e2' }[type] || '#f1f5f9'
}
const nodeTextColor = (type) => {
  return { agent: '#1e40af', condition: '#854d0e', merge: '#166534', trigger: '#6b21a8', webhook: '#9a3412', code: '#374151', sandbox: '#0369a1', approval: '#dc2626' }[type] || '#475569'
}
const nodeTypeIcon = (type) => {
  return { agent: '🤖', condition: '🔀', merge: '🔗', trigger: '⚡', webhook: '🔌', code: '💻', sandbox: '🔒', approval: '🛡️' }[type] || '📦'
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
  return { running: 'bg-blue-100 text-blue-700', completed: 'bg-green-100 text-green-700', failed: 'bg-red-100 text-red-700', cancelled: 'bg-gray-200 text-gray-600', waiting: 'bg-yellow-100 text-yellow-700', waiting_approval: 'bg-orange-100 text-orange-700' }[s] || 'bg-gray-100'
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

// ─── Prompt 编辑器辅助 ───

// 检测变量
const hasVariables = (text) => /\$\{[^}]+\}|\{\{[^}]+\}\}/.test(text)
const extractVariables = (text) => {
  const matches = text.match(/\$\{([^}]+)\}|\{\{([^}]+)\}\}/g)
  if (!matches) return []
  return [...new Set(matches.map(m => m.replace(/[\$\{\}]/g, '')))]
}

// 模板选择
const onTemplateSelect = async (e) => {
  const name = e.target.value
  if (!name) return
  const tpl = promptTemplates.value.find(t => t.name === name)
  if (tpl) {
    updateNodeField('command', tpl.content)
    ElMessage.success(`已加载模板: ${name}`)
  }
  e.target.value = '' // Reset selector
}

// 预览替换结果
const toggleCommandPreview = async () => {
  if (!selectedNodeId.value) return
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return
  let cfg = {}
  try { cfg = JSON.parse(n.config || '{}') } catch {}
  const text = cfg.command || ''
  if (!text) { ElMessage.warning('命令为空'); return }

  // 构建变量映射
  const vars = {}
  // 添加前置节点输出
  const steps = Object.keys(nodeStatuses.value)
  for (const sid of steps) {
    vars[sid] = `(节点 ${sid} 的输出)`
  }
  // 添加 Git 变量
  if (cfg.git_url) vars.repo_url = cfg.git_url
  if (cfg.git_branch) vars.branch = cfg.git_branch

  try {
    const { data } = await previewPrompt({ text, variables: vars })
    previewResolvedText.value = data.resolved
    showCommandPreview.value = true
  } catch (e) {
    // Fallback: 本地替换
    let resolved = text
    for (const [k, v] of Object.entries(vars)) {
      resolved = resolved.replaceAll(`{{${k}}}`, v).replaceAll(`\${${k}}`, v)
    }
    previewResolvedText.value = resolved
    showCommandPreview.value = true
  }
}

// 试运行命令
const trialRunCommand = async () => {
  if (!selectedNodeId.value) return
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return
  let cfg = {}
  try { cfg = JSON.parse(n.config || '{}') } catch {}
  const cmd = cfg.command || ''
  const sid = cfg.sandbox_id || ''
  if (!sid) { ElMessage.warning('请先选择 Sandbox'); return }
  if (!cmd) { ElMessage.warning('命令为空'); return }

  trialRunning.value = true
  trialOutput.value = '> 正在连接 Sandbox...'
  try {
    // 通过 Bridge API 直接执行
    const sb = agents.value.find(a => a.name === sid)
    if (!sb) { trialOutput.value = '❌ Sandbox 未找到'; return }
    
    const res = await api.post(`/terminals/${sb.id}/execute`, { command: cmd, timeout: 30 })
    trialOutput.value = res.data.output || '(无输出)'
    ElMessage.success('试运行完成')
  } catch (e) {
    trialOutput.value = `❌ 执行失败: ${e.response?.data?.error || e.message}`
  } finally {
    trialRunning.value = false
  }
}

// ─── Dialog 状态 ───
const webhookDialogVisible = ref(false)
const webhookActiveTab = ref('headers')
const tempWebhookUrl = ref('')
const tempHeadersJson = ref('')
const tempBodyJson = ref('')
const jsonErrors = ref({ headers: '', body: '' })
const headersTextarea = ref(null)
const bodyTextarea = ref(null)

const commandDialogVisible = ref(false)
const tempCommand = ref('')
const commandPreview = ref('')
const commandTextarea = ref(null)

const approvalDialogVisible = ref(false)
const tempApprovalBody = ref('')
const approvalTextarea = ref(null)

// Agent 状态
const agentDialogVisible = ref(false)
const tempAgentRole = ref('')
const tempAgentInstruction = ref('')
const tempAgentConstraints = ref('')
const tempAgentFormat = ref('')
const agentPreviewText = ref('')
const agentTemplateMatch = ref('')
const agentTextarea = ref(null)

// 可用变量列表
const availableVariables = computed(() => {
  if (!selectedNodeId.value) return []
  // 获取前置节点
  const predecessors = []
  edges.value.forEach(e => {
    if (e.target_id === selectedNodeId.value) predecessors.push(e.source_id)
  })
  
  const vars = predecessors.map(id => `\${${id}}`)
  // 添加通用变量
  vars.push('{{repo_url}}', '{{branch}}', '{{node_output}}')
  return vars
})

// ─── Webhook 编辑器逻辑 ───
const webhookSummary = computed(() => {
  if (!selectedNodeId.value) return { headers: '-', body: '-' }
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return { headers: '-', body: '-' }
  let cfg = {}
  try { cfg = JSON.parse(n.config || '{}') } catch {}
  
  let hCount = 0, bType = '空'
  try {
    const h = typeof cfg.headers === 'string' ? JSON.parse(cfg.headers) : cfg.headers
    hCount = Object.keys(h || {}).length
  } catch {}
  try {
    const b = typeof cfg.body === 'string' ? JSON.parse(cfg.body) : cfg.body
    bType = b ? 'JSON' : '空'
  } catch { bType = '格式错误' }
  
  return { headers: `${hCount} 字段`, body: bType }
})

// ─── Agent 节点逻辑 ───
const agentSummary = computed(() => {
  if (!selectedNodeId.value) return { role: '', instruction: '', constraints: '', format: '' }
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return { role: '', instruction: '', constraints: '', format: '' }
  let cfg = {}
  try { cfg = JSON.parse(n.config || '{}') } catch {}
  
  const prompt = cfg.prompt || ''
  // 简单解析 Markdown Header 格式
  const roleMatch = prompt.match(/# Role\n([\s\S]*?)(?=\n# |$)/i)
  const instrMatch = prompt.match(/# (?:Instruction|Task)\n([\s\S]*?)(?=\n# |$)/i)
  const constrMatch = prompt.match(/# (?:Constraints|Rules)\n([\s\S]*?)(?=\n# |$)/i)
  const formatMatch = prompt.match(/# (?:Output|Format)\n([\s\S]*?)(?=\n# |$)/i)
  
  return {
    role: roleMatch ? roleMatch[1].trim() : '',
    instruction: instrMatch ? instrMatch[1].trim() : (roleMatch ? '' : prompt), // 如果没有 headers，全放在 instruction
    constraints: constrMatch ? constrMatch[1].trim() : '',
    format: formatMatch ? formatMatch[1].trim() : ''
  }
})

const openWebhookEditor = (tab) => {
  if (!selectedNodeId.value) return
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return
  let cfg = {}
  try { cfg = JSON.parse(n.config || '{}') } catch {}
  
  tempWebhookUrl.value = cfg.url || ''
  try {
    const h = typeof cfg.headers === 'string' ? JSON.parse(cfg.headers) : cfg.headers
    tempHeadersJson.value = JSON.stringify(h || {}, null, 2)
  } catch { tempHeadersJson.value = cfg.headers || '{}' }
  
  try {
    const b = typeof cfg.body === 'string' ? JSON.parse(cfg.body) : cfg.body
    tempBodyJson.value = JSON.stringify(b || {}, null, 2)
  } catch { tempBodyJson.value = cfg.body || '{}' }
  
  jsonErrors.value = { headers: '', body: '' }
  webhookActiveTab.value = tab || 'headers'
  webhookDialogVisible.value = true
}

const validateJson = (field) => {
  const val = field === 'headers' ? tempHeadersJson.value : tempBodyJson.value
  try {
    JSON.parse(val)
    jsonErrors.value[field] = ''
  } catch (e) {
    jsonErrors.value[field] = e.message
  }
}

const formatJson = (field) => {
  try {
    const val = field === 'headers' ? tempHeadersJson.value : tempBodyJson.value
    const parsed = JSON.parse(val)
    const formatted = JSON.stringify(parsed, null, 2)
    if (field === 'headers') tempHeadersJson.value = formatted
    else tempBodyJson.value = formatted
    jsonErrors.value[field] = ''
  } catch (e) {
    jsonErrors.value[field] = e.message
  }
}

const insertVar = (target, variable) => {
  const textarea = target === 'headers' ? headersTextarea.value 
               : target === 'body' ? bodyTextarea.value
               : target === 'command' ? commandTextarea.value
               : target === 'agent_instruction' ? agentTextarea.value
               : approvalTextarea.value
  
  if (textarea) {
    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const text = target === 'headers' ? tempHeadersJson.value 
               : target === 'body' ? tempBodyJson.value
               : target === 'command' ? tempCommand.value
               : target === 'agent_instruction' ? tempAgentInstruction.value
               : tempApprovalBody.value
    
    const before = text.substring(0, start)
    const after = text.substring(end)
    const newText = before + variable + after
    
    if (target === 'headers') tempHeadersJson.value = newText
    else if (target === 'body') tempBodyJson.value = newText
    else if (target === 'command') tempCommand.value = newText
    else if (target === 'agent_instruction') tempAgentInstruction.value = newText
    else tempApprovalBody.value = newText
    
    setTimeout(() => {
      textarea.focus()
      textarea.setSelectionRange(start + variable.length, start + variable.length)
    }, 10)
  }
}

const saveWebhookConfig = () => {
  if (jsonErrors.value.headers || jsonErrors.value.body) {
    ElMessage.warning('请修复 JSON 格式错误后再保存')
    return
  }
  
  updateNodeField('url', tempWebhookUrl.value)
  
  // 保存 Headers
  try {
    const h = JSON.parse(tempHeadersJson.value)
    updateNodeField('headers', JSON.stringify(h))
  } catch {}
  
  // 保存 Body
  try {
    const b = JSON.parse(tempBodyJson.value)
    updateNodeField('body', JSON.stringify(b))
  } catch {}
  
  webhookDialogVisible.value = false
  ElMessage.success('Webhook 配置已保存')
}

// ─── Sandbox 命令编辑器逻辑 ───
const openCommandEditor = () => {
  if (!selectedNodeId.value) return
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return
  let cfg = {}
  try { cfg = JSON.parse(n.config || '{}') } catch {}
  
  tempCommand.value = cfg.command || ''
  checkCommandVariables()
  commandDialogVisible.value = true
}

const onTemplateSelectFromDialog = (e) => {
  const name = e.target.value
  if (!name) return
  const tpl = promptTemplates.value.find(t => t.name === name)
  if (tpl) {
    tempCommand.value = tpl.content
    checkCommandVariables()
    ElMessage.success(`已加载模板: ${name}`)
  }
  e.target.value = ''
}

const checkCommandVariables = () => {
  const vars = {}
  // 模拟替换
  availableVariables.value.forEach(v => {
    const key = v.replace(/[\$\{\}]/g, '')
    vars[key] = `(变量 ${key})`
  })
  vars['repo_url'] = 'git@github.com:test/repo'
  vars['branch'] = 'main'
  
  let resolved = tempCommand.value
  for (const [k, v] of Object.entries(vars)) {
    resolved = resolved.replaceAll(`{{${k}}}`, v).replaceAll(`\${${k}}`, v)
  }
  commandPreview.value = resolved
}

const saveCommandConfig = () => {
  updateNodeField('command', tempCommand.value)
  commandDialogVisible.value = false
  ElMessage.success('命令已保存')
}

// ─── Approval 编辑器逻辑 ───
const openApprovalEditor = () => {
  if (!selectedNodeId.value) return
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return
  let cfg = {}
  try { cfg = JSON.parse(n.config || '{}') } catch {}
  
  tempApprovalBody.value = cfg.card_body || ''
  approvalDialogVisible.value = true
}

const saveApprovalConfig = () => {
  updateNodeField('card_body', tempApprovalBody.value)
  approvalDialogVisible.value = false
  ElMessage.success('卡片内容已保存')
}

// ─── Agent 编辑器逻辑 ───
const openAgentEditor = () => {
  if (!selectedNodeId.value) return
  const n = nodes.value.find(x => x.node_id === selectedNodeId.value)
  if (!n) return
  let cfg = {}
  try { cfg = JSON.parse(n.config || '{}') } catch {}
  
  const prompt = cfg.prompt || ''
  // 解析逻辑同 agentSummary
  const roleMatch = prompt.match(/# Role\n([\s\S]*?)(?=\n# |$)/i)
  const instrMatch = prompt.match(/# (?:Instruction|Task)\n([\s\S]*?)(?=\n# |$)/i)
  const constrMatch = prompt.match(/# (?:Constraints|Rules)\n([\s\S]*?)(?=\n# |$)/i)
  const formatMatch = prompt.match(/# (?:Output|Format)\n([\s\S]*?)(?=\n# |$)/i)
  
  if (roleMatch) {
     tempAgentRole.value = roleMatch[1].trim()
     tempAgentInstruction.value = instrMatch ? instrMatch[1].trim() : ''
     tempAgentConstraints.value = constrMatch ? constrMatch[1].trim() : ''
     tempAgentFormat.value = formatMatch ? formatMatch[1].trim() : ''
  } else {
     // 旧格式，没有 headers，全部放进 Instruction
     tempAgentInstruction.value = prompt
     tempAgentRole.value = ''
     tempAgentConstraints.value = ''
     tempAgentFormat.value = ''
  }
  
  updateAgentPreview()
  agentDialogVisible.value = true
}

const updateAgentPreview = () => {
  // 自动拼接为结构化 Markdown
  let md = ''
  if (tempAgentRole.value.trim()) md += `# Role\n${tempAgentRole.value.trim()}\n\n`
  if (tempAgentInstruction.value.trim()) md += `# Instruction\n${tempAgentInstruction.value.trim()}\n\n`
  if (tempAgentConstraints.value.trim()) md += `# Constraints\n${tempAgentConstraints.value.trim()}\n\n`
  if (tempAgentFormat.value.trim()) md += `# Output Format\n${tempAgentFormat.value.trim()}\n\n`
  
  agentPreviewText.value = md
}

const formatAgentPreview = () => {
  // 简单的格式化，比如去掉多余空行
  agentPreviewText.value = agentPreviewText.value.replace(/\n{3,}/g, '\n\n').trim()
}

const saveAgentConfig = () => {
  updateNodeField('prompt', agentPreviewText.value.trim())
  agentDialogVisible.value = false
  ElMessage.success('Prompt 已保存')
}

// 更新 insertVar 逻辑

const renderedMarkdown = computed(() => {
  if (!tempApprovalBody.value) return '<div class="text-gray-400 italic">暂无内容...</div>'
  return marked.parse(tempApprovalBody.value)
})


onMounted(() => { 
  loadFlows(); loadAgents(); loadPromptTemplates()
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
