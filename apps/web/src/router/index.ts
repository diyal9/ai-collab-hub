import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated, isAdmin } from '../utils/auth'
import { notifyError } from '../utils/notify'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/Login.vue'),
    meta: { public: true },
  },
  {
    path: '/',
    component: () => import('../views/Layout.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', name: 'dashboard', component: () => import('../views/Dashboard.vue') },
      { path: 'tasks', name: 'tasks', component: () => import('../views/Tasks.vue') },
      { path: 'files', name: 'files', component: () => import('../views/Files.vue') },
      { path: 'terminals', name: 'terminals', component: () => import('../views/AgentTerminals.vue') },
      { path: 'orchestrator', name: 'orchestrator', component: () => import('../views/AgentOrchestrator.vue') },
      { path: 'node-templates', name: 'node-templates', component: () => import('../views/NodeTemplateManager.vue') },
      { path: 'knowledge', name: 'knowledge', component: () => import('../views/KnowledgeHub.vue') },
      { path: 'memory', name: 'memory', component: () => import('../views/MemorySystem.vue') },
      { path: 'agents', name: 'agents', component: () => import('../views/AgentManager.vue') },
      {
        path: 'admin',
        name: 'admin',
        component: () => import('../views/Admin.vue'),
        meta: { requiresAdmin: true },
      },
      { path: ':pathMatch(.*)*', name: 'not-found', component: () => import('../views/NotFound.vue') },
    ],
  },
]

const router = createRouter({
  history: createWebHistory('/ai-collab-hub/'),
  routes,
})

router.beforeEach((to) => {
  const publicRoute = to.meta.public === true
  const authed = isAuthenticated()

  if (!publicRoute && !authed) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  if (to.path === '/login' && authed) {
    return { path: '/' }
  }

  if (to.meta.requiresAdmin && !isAdmin()) {
    notifyError('需要管理员权限')
    return { path: '/' }
  }

  return true
})

export default router
