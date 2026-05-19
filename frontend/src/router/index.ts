import { createRouter, createWebHistory } from 'vue-router'
const routes = [
  { path: '/login', component: () => import('../views/Login.vue') },
  { path: '/', component: () => import('../views/Layout.vue'), children: [
    { path: '', name: 'dashboard', component: () => import('../views/Dashboard.vue') },
    { path: 'tasks', name: 'tasks', component: () => import('../views/Tasks.vue') },
    { path: 'files', name: 'files', component: () => import('../views/Files.vue') },
    { path: 'terminals', name: 'terminals', component: () => import('../views/AgentTerminals.vue') },
    { path: 'skills', name: 'skills', component: () => import('../views/SkillsManager.vue') },
    { path: 'orchestrator', name: 'orchestrator', component: () => import('../views/AgentOrchestrator.vue') },
    { path: 'node-templates', name: 'node-templates', component: () => import('../views/NodeTemplateManager.vue') },
    { path: 'knowledge', name: 'knowledge', component: () => import('../views/KnowledgeHub.vue') },
    { path: 'agents', name: 'agents', component: () => import('../views/AgentManager.vue') },
    { path: 'admin', name: 'admin', component: () => import('../views/Admin.vue'), meta: { requiresAdmin: true } }
  ]}
]
export default createRouter({ history: createWebHistory('/hermesshare/'), routes })
