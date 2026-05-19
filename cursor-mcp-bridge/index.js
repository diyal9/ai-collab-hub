/**
 * Cursor MCP Server - AI Collab Hub Bridge
 * 
 * This MCP server runs alongside Cursor and bridges it to the AI Collab Hub.
 * It receives tasks from the Hub via WebSocket and exposes them as MCP tools.
 * 
 * Usage:
 *   1. npm install
 *   2. Configure Cursor to use this as an MCP server
 *   3. Or run standalone: node index.js --hub-url wss://your-hub/ws/agent
 *
 * Cursor MCP config (in cursor mcp config):
 * {
 *   "mcpServers": {
 *     "ai-collab-hub": {
 *       "command": "node",
 *       "args": ["/path/to/cursor-mcp-bridge/index.js"],
 *       "env": {
 *         "HUB_URL": "wss://your-hub/ws/agent",
 *         "AGENT_NAME": "cursor-main",
 *         "AGENT_TOKEN": "agt_cursor_xxx"
 *       }
 *     }
 *   }
 * }
 */

import WebSocket from 'ws';
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { CallToolRequestSchema, ListToolsRequestSchema } from '@modelcontextprotocol/sdk/types.js';
import fs from 'fs';
import path from 'path';
import os from 'os';

// ─── Config ───
const HUB_URL = process.env.HUB_URL || 'ws://localhost:8085/ws/agent';
const AGENT_NAME = process.env.AGENT_NAME || 'cursor-main';
const AGENT_TYPE = process.env.AGENT_TYPE || 'cursor';
const AGENT_TOKEN = process.env.AGENT_TOKEN || '';
const TASK_DIR = process.env.TASK_DIR || path.join(os.tmpdir(), 'cursor-tasks');

fs.mkdirSync(TASK_DIR, { recursive: true });

// ─── WebSocket to Hub ───
let ws = null;
let currentTask = null;
let taskLog = [];

function connectToHub() {
  ws = new WebSocket(HUB_URL);

  ws.on('open', () => {
    console.error('[MCP] Connected to Hub, registering...');
    ws.send(JSON.stringify({
      method: 'agent.register',
      payload: {
        name: AGENT_NAME,
        type: AGENT_TYPE,
        token: AGENT_TOKEN,
        platform: process.platform,
        host: os.hostname(),
        capabilities: ['editor', 'mcp', 'file', 'terminal'],
      }
    }));
  });

  ws.on('message', (data) => {
    const msg = JSON.parse(data.toString());
    console.error(`[MCP] Hub → ${msg.method}:`, JSON.stringify(msg.payload).slice(0, 200));

    switch (msg.method) {
      case 'task.start':
        handleTaskStart(msg.payload);
        break;
      case 'task.cancel':
        currentTask = null;
        break;
      case 'approval.response':
        handleApprovalResponse(msg.payload);
        break;
      case 'system.ack':
        console.error('[MCP] Registration acknowledged');
        break;
    }
  });

  ws.on('close', () => {
    console.error('[MCP] Disconnected from Hub, reconnecting in 5s...');
    setTimeout(connectToHub, 5000);
  });

  ws.on('error', (err) => {
    console.error('[MCP] WebSocket error:', err.message);
  });
}

function sendToHub(method, payload) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ method, payload }));
  }
}

// ─── Task Handling ───
function handleTaskStart(payload) {
  const { task_id, title, prompt } = payload;
  currentTask = { id: task_id, title, prompt, status: 'running' };

  // Write task file for Cursor user to see
  const taskFile = path.join(TASK_DIR, `${task_id}.md`);
  fs.writeFileSync(taskFile, `# Task: ${title}\n\n## Instructions\n${prompt}\n\n## Status: Running\n`);

  sendToHub('task.progress', {
    task_id,
    type: 'system',
    content: `Task "${title}" received and written to ${taskFile}`,
  });

  taskLog.push({ ts: Date.now(), type: 'system', content: `Task started: ${title}` });
}

function handleApprovalResponse(payload) {
  const { task_id, approved, reply } = payload;
  taskLog.push({ ts: Date.now(), type: 'approval', content: `Approved: ${approved}, Reply: ${reply}` });
}

function reportProgress(taskId, type, content) {
  if (taskId) {
    sendToHub('task.progress', {
      task_id: taskId,
      type,
      content: content.slice(0, 5000),
    });
  }
  taskLog.push({ ts: Date.now(), type, content: content.slice(0, 500) });
}

function completeTask(taskId, output) {
  sendToHub('task.complete', {
    task_id: taskId,
    output: output.slice(0, 10000),
  });
  if (currentTask && currentTask.id === taskId) {
    const resultFile = path.join(TASK_DIR, `${taskId}.result`);
    fs.writeFileSync(resultFile, output);
    const taskFile = path.join(TASK_DIR, `${taskId}.md`);
    try { fs.appendFileSync(taskFile, '\n\n## Status: Completed\n'); } catch {}
  }
  currentTask = null;
}

function requestApproval(question, context, suggested) {
  return new Promise((resolve) => {
    sendToHub('approval.request', {
      task_id: currentTask?.id || '',
      question,
      context: context?.slice(0, 2000) || '',
      suggested: suggested || '',
    });
    // Approval response comes via WS message handler
    // For simplicity, we use a callback mechanism
    const checkApproval = () => {
      const lastApproval = taskLog.filter(l => l.type === 'approval').pop();
      if (lastApproval) {
        resolve(JSON.parse(lastApproval.content));
      } else {
        setTimeout(checkApproval, 1000);
      }
    };
    setTimeout(checkApproval, 500);
    // Timeout after 5 minutes
    setTimeout(() => resolve({ approved: true, reply: 'timeout-auto-approve' }), 300000);
  });
}

// ─── MCP Server ───
const server = new Server(
  { name: 'ai-collab-hub', version: '1.0.0' },
  { capabilities: { tools: {} } }
);

// List available tools
server.setRequestHandler(ListToolsRequestSchema, async () => ({
  tools: [
    {
      name: 'get_current_task',
      description: 'Get the current active task from AI Collab Hub',
      inputSchema: {
        type: 'object',
        properties: {},
      },
    },
    {
      name: 'report_progress',
      description: 'Report progress on the current task to AI Collab Hub',
      inputSchema: {
        type: 'object',
        properties: {
          content: { type: 'string', description: 'Progress description' },
          type: { type: 'string', enum: ['stdout', 'stderr', 'system'], default: 'stdout' },
        },
        required: ['content'],
      },
    },
    {
      name: 'complete_task',
      description: 'Mark the current task as completed with final output',
      inputSchema: {
        type: 'object',
        properties: {
          output: { type: 'string', description: 'Final output/summary' },
        },
        required: ['output'],
      },
    },
    {
      name: 'request_approval',
      description: 'Request human approval for a potentially dangerous action',
      inputSchema: {
        type: 'object',
        properties: {
          question: { type: 'string', description: 'What needs approval?' },
          context: { type: 'string', description: 'Context/code snippet' },
          suggested: { type: 'string', description: 'Suggested action' },
        },
        required: ['question'],
      },
    },
    {
      name: 'read_task_file',
      description: 'Read the current task file for detailed instructions',
      inputSchema: {
        type: 'object',
        properties: {},
      },
    },
    {
      name: 'list_task_logs',
      description: 'List recent task logs',
      inputSchema: {
        type: 'object',
        properties: {
          limit: { type: 'number', default: 20 },
        },
      },
    },
    {
      name: 'run_terminal',
      description: 'Run a terminal command and report output to Hub',
      inputSchema: {
        type: 'object',
        properties: {
          command: { type: 'string', description: 'Shell command to run' },
        },
        required: ['command'],
      },
    },
  ],
}));

// Handle tool calls
server.setRequestHandler(CallToolRequestSchema, async ({ params: { name, arguments: args } }) => {
  switch (name) {
    case 'get_current_task':
      return {
        content: [{
          type: 'text',
          text: currentTask
            ? JSON.stringify({ id: currentTask.id, title: currentTask.title, status: currentTask.status })
            : 'No active task',
        }],
      };

    case 'report_progress':
      reportProgress(currentTask?.id, args.type || 'stdout', args.content);
      return { content: [{ type: 'text', text: 'Progress reported' }] };

    case 'complete_task':
      if (!currentTask) return { content: [{ type: 'text', text: 'No active task' }], isError: true };
      completeTask(currentTask.id, args.output);
      return { content: [{ type: 'text', text: 'Task marked as completed' }] };

    case 'request_approval':
      if (!currentTask) return { content: [{ type: 'text', text: 'No active task' }], isError: true };
      const resp = await requestApproval(args.question, args.context, args.suggested);
      return {
        content: [{
          type: 'text',
          text: `Approval: ${resp.approved ? 'APPROVED' : 'REJECTED'}\nReply: ${resp.reply || ''}`,
        }],
      };

    case 'read_task_file':
      if (!currentTask) return { content: [{ type: 'text', text: 'No active task' }], isError: true };
      const taskFile = path.join(TASK_DIR, `${currentTask.id}.md`);
      try {
        const content = fs.readFileSync(taskFile, 'utf-8');
        return { content: [{ type: 'text', text: content }] };
      } catch {
        return { content: [{ type: 'text', text: 'Task file not found' }], isError: true };
      }

    case 'list_task_logs':
      const limit = args.limit || 20;
      const recentLogs = taskLog.slice(-limit);
      return {
        content: [{
          type: 'text',
          text: recentLogs.map(l => `[${new Date(l.ts).toISOString()}] ${l.type}: ${l.content}`).join('\n'),
        }],
      };

    case 'run_terminal': {
      const { execSync } = await import('child_process');
      try {
        const output = execSync(args.command, {
          encoding: 'utf-8',
          timeout: 60000,
          maxBuffer: 50 * 1024 * 1024,
        });
        reportProgress(currentTask?.id, 'stdout', output);
        return { content: [{ type: 'text', text: output.slice(0, 5000) }] };
      } catch (err) {
        const output = err.stdout || err.stderr || err.message;
        reportProgress(currentTask?.id, 'stderr', output);
        return {
          content: [{ type: 'text', text: output.slice(0, 5000) }],
          isError: true,
        };
      }
    }

    default:
      return { content: [{ type: 'text', text: `Unknown tool: ${name}` }], isError: true };
  }
});

// ─── Start ───
async function main() {
  console.error('[MCP] Starting Cursor AI Collab Hub Bridge...');
  console.error(`[MCP] Hub URL: ${HUB_URL}`);
  console.error(`[MCP] Agent: ${AGENT_NAME} (${AGENT_TYPE})`);

  connectToHub();

  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error('[MCP] MCP server running on stdio');
}

main().catch(console.error);
