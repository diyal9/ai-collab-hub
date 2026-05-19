import axios from 'axios'

export const api = axios.create({ baseURL: '/hermesshare/api' })
api.interceptors.request.use(c => { c.headers.Authorization = `Bearer ${localStorage.getItem('token')}`; return c })
api.interceptors.response.use(r => r, e => {
  if (e.response?.status === 401) {
    localStorage.removeItem('token')
    localStorage.removeItem('role')
    if (window.location.pathname !== '/hermesshare/login') window.location.href = '/hermesshare/login'
  }
  // 403 不再自动踢出登录 —— 可能是权限不足，而非 token 无效
  return Promise.reject(e)
})

// Auth
export const login = (u: string, p: string) => api.post('/login', { user: u, pass: p })

// Tasks
export const getTasks = () => api.get('/tasks')
export const createTask = (data: any) => api.post('/tasks', data)
export const getTaskSteps = (id: string) => api.get(`/tasks/${id}/steps`)
export const replyTask = (id: string, sid: string, reply: string) => api.post(`/tasks/${id}/steps/${sid}/reply`, { reply })

// Files
export const uploadFile = (fd: FormData) => api.post('/upload', fd, { headers: { 'Content-Type': 'multipart/form-data' } })
export const getFiles = () => api.get('/files')
export const downloadFile = (path: string) => api.get('/files/download', { params: { path }, responseType: 'blob' })

// Users
export const getUsers = () => api.get('/users')
export const createUser = (u: string, p: string) => api.post('/users', { username: u, password: p })

// Agents
export const getAgents = () => api.get('/agents')

// Agent Terminals
export const getTerminals = () => api.get('/terminals')
export const createTerminal = (data: any) => api.post('/terminals', data)
export const updateTerminal = (id: string, data: any) => api.put(`/terminals/${id}`, data)
export const deleteTerminal = (id: string) => api.delete(`/terminals/${id}`)
export const connectTerminal = (id: string) => api.post(`/terminals/${id}/connect`)

// Agent Flows (Orchestration)
export const getFlows = () => api.get('/flows')
export const createFlow = (data: any) => api.post('/flows', data)
export const updateFlow = (id: string, data: any) => api.put(`/flows/${id}`, data)
export const deleteFlow = (id: string) => api.delete(`/flows/${id}`)
export const getFlowNodes = (id: string) => api.get(`/flows/${id}/nodes`)
export const createFlowNode = (id: string, data: any) => api.post(`/flows/${id}/nodes`, data)
export const updateFlowNode = (id: string, nid: string, data: any) => api.put(`/flows/${id}/nodes/${nid}`, data)
export const deleteFlowNode = (id: string, nid: string) => api.delete(`/flows/${id}/nodes/${nid}`)
export const getFlowEdges = (id: string) => api.get(`/flows/${id}/edges`)
export const createFlowEdge = (id: string, data: any) => api.post(`/flows/${id}/edges`, data)
export const deleteFlowEdge = (id: string, eid: string) => api.delete(`/flows/${id}/edges/${eid}`)
export const saveFlowGraph = (id: string, data: any) => api.post(`/flows/${id}/save-graph`, data)

// Knowledge Hub
export const getKnowledge = (params?: any) => api.get('/knowledge', { params })
export const createKnowledge = (data: any) => api.post('/knowledge', data)
export const updateKnowledge = (id: string, data: any) => api.put(`/knowledge/${id}`, data)
export const deleteKnowledge = (id: string) => api.delete(`/knowledge/${id}`)
export const reviewKnowledge = (id: string, action: string, comment: string) => api.post(`/knowledge/${id}/review`, { action, comment })
export const getAuditLog = (id: string) => api.get(`/knowledge/${id}/audit-log`)
export const getKnowledgeStats = () => api.get('/knowledge/stats')

// Memory Sessions
export const getMemorySessions = () => api.get('/memory-sessions')
export const createMemorySession = (data: any) => api.post('/memory-sessions', data)

// Flow Execution
export const executeFlow = (id: string) => api.post(`/flows/${id}/execute`)
export const getFlowExecutions = (id: string) => api.get(`/flows/${id}/executions`)
export const getExecutionSteps = (id: string) => api.get(`/flow-executions/${id}/steps`)
export const cancelExecution = (id: string) => api.post(`/flow-executions/${id}/cancel`)
export const validateFlow = (id: string) => api.post(`/flows/${id}/validate`)
