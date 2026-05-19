<template>
  <div class="node-template-manager">
    <!-- Header -->
    <div class="header">
      <h2>🔧 节点工厂 (Node Template Editor)</h2>
      <p>定义、编辑和管理可视化编排中的节点模板</p>
    </div>

    <div class="main-content">
      <!-- 左侧：模板列表 -->
      <div class="sidebar">
        <div class="sidebar-header">
          <button class="add-btn" @click="createNew">
            + 新建模板
          </button>
        </div>
        <div class="template-list">
          <div 
            v-for="tpl in templates" 
            :key="tpl.id"
            :class="['template-item', { active: currentTemplate?.id === tpl.id }]"
            @click="selectTemplate(tpl)"
          >
            <span class="icon">{{ tpl.icon }}</span>
            <div class="info">
              <span class="name">{{ tpl.name }}</span>
              <span class="type">{{ tpl.type }}</span>
            </div>
            <button class="del-btn" @click.stop="deleteTemplate(tpl.id)">×</button>
          </div>
          <div v-if="templates.length === 0" class="empty-state">
            暂无模板，请点击"新建模板"
          </div>
        </div>
      </div>

      <!-- 右侧：编辑器 -->
      <div class="editor" v-if="currentTemplate">
        <div class="editor-tabs">
          <button :class="{ active: activeTab === 'basic' }" @click="activeTab = 'basic'">基础设置</button>
          <button :class="{ active: activeTab === 'params' }" @click="activeTab = 'params'">参数定义</button>
          <button :class="{ active: activeTab === 'script' }" @click="activeTab = 'script'">执行逻辑</button>
        </div>

        <div class="editor-body">
          <!-- Tab 1: 基础设置 -->
          <div v-show="activeTab === 'basic'" class="tab-content">
            <el-form label-position="top">
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="节点名称">
                    <el-input v-model="currentTemplate.name" placeholder="如：Cocos 构建" />
                  </el-form-item>
                </el-col>
                <el-col :span="6">
                  <el-form-item label="图标 (Emoji)">
                    <el-input v-model="currentTemplate.icon" placeholder="🏗️" maxlength="2" />
                  </el-form-item>
                </el-col>
                <el-col :span="6">
                  <el-form-item label="分类标签">
                    <el-select v-model="currentTemplate.category" filterable allow-create>
                      <el-option label="AI 编码" value="AI 编码" />
                      <el-option label="构建工具" value="构建工具" />
                      <el-option label="部署" value="部署" />
                      <el-option label="通知" value="通知" />
                    </el-select>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item label="描述">
                <el-input v-model="currentTemplate.description" type="textarea" rows="2" placeholder="用于左侧面板的副标题展示" />
              </el-form-item>
            </el-form>
          </div>

          <!-- Tab 2: 参数定义 -->
          <div v-show="activeTab === 'params'" class="tab-content">
            <div class="param-header">
              <h4>输入参数 (Inputs)</h4>
              <el-button size="small" type="primary" @click="addParam">+ 添加参数</el-button>
            </div>
            <el-table :data="currentTemplate.inputs" border class="param-table">
              <el-table-column label="变量名 (Key)" width="150">
                <template #default="{ row }">
                  <el-input v-model="row.key" placeholder="key_name" size="small" />
                </template>
              </el-table-column>
              <el-table-column label="显示名 (Label)" width="150">
                <template #default="{ row }">
                  <el-input v-model="row.label" placeholder="显示名称" size="small" />
                </template>
              </el-table-column>
              <el-table-column label="类型" width="120">
                <template #default="{ row }">
                  <el-select v-model="row.type" size="small">
                    <el-option label="String" value="string" />
                    <el-option label="Number" value="number" />
                    <el-option label="Select" value="select" />
                    <el-option label="Boolean" value="boolean" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column label="默认值">
                <template #default="{ row }">
                  <el-input v-if="row.type !== 'boolean'" v-model="row.default" size="small" />
                  <el-switch v-else v-model="row.default" active-value="true" inactive-value="false" size="small" />
                </template>
              </el-table-column>
              <el-table-column label="必填" width="60" align="center">
                <template #default="{ row }">
                  <el-checkbox v-model="row.required" />
                </template>
              </el-table-column>
              <el-table-column label="操作" width="60" align="center">
                <template #default="{ $index }">
                  <el-button type="danger" link size="small" @click="removeParam($index)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
            
            <div v-if="currentTemplate.inputs.some(i => i.type === 'select')" class="options-config">
              <h4>下拉选项配置 (Select)</h4>
              <el-alert title="配置类型为 'Select' 的参数的选项，每行一个" type="info" :closable="false" class="mb-2" />
              <el-table :data="selectInputs" border class="param-table">
                <el-table-column label="参数 Key">
                  <template #default="{ row }">{{ row.key }}</template>
                </el-table-column>
                <el-table-column label="选项列表 (用逗号分隔)">
                  <template #default="{ row }">
                    <el-input v-model="row.optionsStr" size="small" placeholder="opt1, opt2, opt3" />
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </div>

          <!-- Tab 3: 执行逻辑 -->
          <div v-show="activeTab === 'script'" class="tab-content">
            <el-form label-position="top">
              <el-form-item label="执行引擎">
                <el-radio-group v-model="currentTemplate.type">
                  <el-radio-button label="shell">Shell</el-radio-button>
                  <el-radio-button label="python">Python</el-radio-button>
                  <el-radio-button label="http">HTTP Request</el-radio-button>
                </el-radio-group>
              </el-form-item>

              <el-form-item :label="currentTemplate.type === 'http' ? 'URL 模板' : '命令/脚本模板'">
                <el-input 
                  v-model="currentTemplate.command_template" 
                  type="textarea" 
                  rows="6" 
                  placeholder="例如：echo hello {{name}}"
                  class="code-editor"
                />
                <div class="tip">
                  💡 提示：使用 <code v-pre>{{ key }}</code> 引用上方定义的参数变量名
                </div>
              </el-form-item>
            </el-form>
          </div>

          <!-- Footer Actions -->
          <div class="editor-footer">
            <div class="json-preview">
              <div class="json-header">JSON 预览</div>
              <pre>{{ JSON.stringify(currentTemplate, null, 2) }}</pre>
            </div>
            <el-button type="success" @click="saveTemplate">💾 保存模板</el-button>
          </div>
        </div>
      </div>
      
      <!-- 空状态 -->
      <div v-else class="editor-placeholder">
        选择或新建一个模板开始编辑
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';

// 模拟初始数据
const initialTemplates = [
  {
    id: 'tpl_shell_demo',
    name: 'Shell 演示',
    icon: '🐚',
    category: '基础',
    description: '执行简单的 Shell 命令',
    type: 'shell',
    command_template: 'echo "Hello {{name}}, today is {{date}}"',
    inputs: [
      { key: 'name', label: '名字', type: 'string', default: 'World', required: true },
      { key: 'date', label: '日期', type: 'string', default: '2026-05-19', required: false }
    ]
  },
  {
    id: 'tpl_cocos_build',
    name: 'Cocos 构建',
    icon: '🏗️',
    category: '构建工具',
    description: '调用 cocos 命令行构建',
    type: 'shell',
    command_template: 'cocos compile -p {{platform}} -m release',
    inputs: [
      { key: 'platform', label: '平台', type: 'select', default: 'web-mobile', required: true, options: ['web-mobile', 'ios', 'android'] }
    ]
  }
];

const templates = reactive([]);
const currentTemplate = ref(null);
const activeTab = ref('basic');

onMounted(() => {
  const saved = localStorage.getItem('node_templates');
  if (saved) {
    try {
      templates.push(...JSON.parse(saved));
    } catch (e) {
      templates.push(...initialTemplates);
    }
  } else {
    templates.push(...initialTemplates);
  }
  
  if (templates.length > 0) {
    selectTemplate(templates[0]);
  }
});

// 辅助计算属性：提取出类型为 select 的 inputs
const selectInputs = computed(() => {
  if (!currentTemplate.value) return [];
  return currentTemplate.value.inputs
    .filter(i => i.type === 'select')
    .map(i => ({
      ...i,
      optionsStr: i.options ? i.options.join(', ') : ''
    }));
});

// 监听 selectInputs 的 optionsStr 变化，同步回 original
watch(selectInputs, (newVals) => {
  if (!currentTemplate.value) return;
  newVals.forEach(sv => {
    const target = currentTemplate.value.inputs.find(i => i.key === sv.key);
    if (target) {
      target.options = sv.optionsStr.split(',').map(s => s.trim()).filter(s => s);
    }
  });
}, { deep: true });

const createNew = () => {
  const id = 'tpl_' + Date.now();
  const newTpl = {
    id,
    name: '新节点',
    icon: '🔌',
    category: '自定义',
    description: '描述...',
    type: 'shell',
    command_template: '',
    inputs: []
  };
  templates.push(newTpl);
  selectTemplate(newTpl);
  ElMessage.success('已创建新模板');
};

const selectTemplate = (tpl) => {
  currentTemplate.value = tpl;
  activeTab.value = 'basic';
};

const deleteTemplate = (id) => {
  ElMessageBox.confirm('确定删除此模板吗？', '警告', { type: 'warning' })
    .then(() => {
      const idx = templates.findIndex(t => t.id === id);
      if (idx > -1) {
        templates.splice(idx, 1);
        if (currentTemplate.value?.id === id) {
          currentTemplate.value = templates.length > 0 ? templates[0] : null;
        }
        saveToLocal();
        ElMessage.success('已删除');
      }
    })
    .catch(() => {});
};

const addParam = () => {
  if (!currentTemplate.value) return;
  currentTemplate.value.inputs.push({
    key: '',
    label: '',
    type: 'string',
    default: '',
    required: false
  });
};

const removeParam = (index) => {
  if (!currentTemplate.value) return;
  currentTemplate.value.inputs.splice(index, 1);
};

const saveTemplate = () => {
  if (!currentTemplate.value) return;
  // 校验
  if (!currentTemplate.value.name) {
    ElMessage.error('请输入节点名称');
    return;
  }
  if (!currentTemplate.value.command_template && currentTemplate.value.type !== 'http') {
    ElMessage.error('请输入命令/脚本模板');
    return;
  }

  saveToLocal();
  ElMessage.success('模板保存成功！画布左侧面板已更新');
};

const saveToLocal = () => {
  localStorage.setItem('node_templates', JSON.stringify(templates));
  // 触发一个自定义事件，通知左侧面板刷新 (如果面板是同一个 Vue App)
  window.dispatchEvent(new Event('node-templates-updated'));
};
</script>

<style scoped>
.node-template-manager {
  padding: 20px;
  height: 100vh;
  background: #f5f7fa;
  color: #1f2937;
  display: flex;
  flex-direction: column;
}

.header {
  margin-bottom: 20px;
}
.header h2 { margin: 0 0 5px; font-size: 20px; }
.header p { margin: 0; color: #6b7280; font-size: 14px; }

.main-content {
  display: flex;
  flex: 1;
  gap: 20px;
  overflow: hidden;
}

.sidebar {
  width: 280px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 15px;
  border-bottom: 1px solid #eee;
}

.add-btn {
  width: 100%;
  padding: 8px;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: bold;
}

.template-list {
  flex: 1;
  overflow-y: auto;
}

.template-item {
  display: flex;
  align-items: center;
  padding: 10px 15px;
  border-bottom: 1px solid #f3f4f6;
  cursor: pointer;
  transition: background 0.2s;
}
.template-item:hover { background: #f9fafb; }
.template-item.active { background: #eff6ff; border-left: 3px solid #3b82f6; }

.icon { font-size: 20px; margin-right: 10px; }
.info { flex: 1; display: flex; flex-direction: column; }
.name { font-weight: 500; font-size: 14px; }
.type { font-size: 12px; color: #9ca3af; }

.del-btn {
  background: none;
  border: none;
  color: #ef4444;
  font-size: 16px;
  cursor: pointer;
}

.editor {
  flex: 1;
  background: white;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  display: flex;
  flex-direction: column;
}

.editor-placeholder {
  flex: 1;
  background: white;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #9ca3af;
}

.editor-tabs {
  display: flex;
  border-bottom: 1px solid #eee;
  background: #fafafa;
}
.editor-tabs button {
  flex: 1;
  padding: 12px;
  background: none;
  border: none;
  cursor: pointer;
  font-weight: 500;
  color: #6b7280;
}
.editor-tabs button.active {
  background: white;
  color: #3b82f6;
  border-bottom: 2px solid #3b82f6;
}

.editor-body {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
}

.tab-content { max-width: 800px; }

.param-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.param-header h4 { margin: 0; }

.param-table { margin-bottom: 20px; }
.options-config { margin-top: 20px; }
.mb-2 { margin-bottom: 10px; }

.code-editor { font-family: monospace; }
.tip { margin-top: 5px; font-size: 12px; color: #6b7280; }
.tip code { background: #f3f4f6; padding: 2px 4px; border-radius: 3px; }

.editor-footer {
  margin-top: 30px;
  display: flex;
  align-items: flex-end;
  gap: 20px;
  border-top: 1px solid #eee;
  padding-top: 20px;
}

.json-preview {
  flex: 1;
  background: #1f2937;
  color: #a5b4fc;
  padding: 15px;
  border-radius: 6px;
  font-family: monospace;
  font-size: 12px;
  height: 200px;
  overflow: auto;
}
.json-header { color: white; font-weight: bold; margin-bottom: 5px; }
</style>
