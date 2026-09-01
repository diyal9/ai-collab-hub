// Cursor MCP Bridge v2 — 双向桥接 Hub ↔ Cursor IDE
// 作为 MCP Server 被 Cursor 调用 + 作为 Hub Agent Client 连接 Hub
//
// 用法:
//   HUB_URL=ws://localhost:8087/ws/agent \
//   AGENT_NAME=cursor-main \
//   AGENT_TOKEN=agt_cursor_xxx \
//   PROJECT_PATH=/path/to/project \
//   HUB_API_KEY=your-api-key \
//   node index.js
//
// Cursor MCP 配置:
// { "mcpServers": { "ai-collab-hub": { "command": "node", "args": ["/path/to/index.js"],
//   "env": { "HUB_URL": "...", "AGENT_NAME": "...", "AGENT_TOKEN": "...", "HUB_API_KEY": "..." }}}}

const { McpServer } = require("@modelcontextprotocol/sdk/server/mcp.js");
const { StdioServerTransport } = require("@modelcontextprotocol/sdk/server/stdio.js");
const { z } = require("zod");
const WebSocket = require("ws");
const fs = require("fs");
const path = require("path");
const { exec } = require("child_process");
const { promisify } = require("util");
const execAsync = promisify(exec);

// ─── 配置 ───
const CONFIG = {
  hubUrl: process.env.HUB_URL || "ws://localhost:8087/ws/agent",
  agentName: process.env.AGENT_NAME || "cursor-main",
  agentToken: process.env.AGENT_TOKEN || "",
  projectPath: process.env.PROJECT_PATH || process.cwd(),
  hubApiKey: process.env.HUB_API_KEY || "",
  taskDir: "/tmp/cursor-tasks",
  logLevel: process.env.LOG_LEVEL || "info",
};

// ─── 日志 ───
const log = (level, ...args) => {
  const levels = { error: 0, warn: 1, info: 2, debug: 3 };
  if (levels[level] <= levels[CONFIG.logLevel]) {
    const ts = new Date().toISOString();
    process.stderr.write(`[${ts}] [${level.toUpperCase()}] ${args.join(" ")}\n`);
  }
};

// ─── Hub Client ───
class HubClient {
  constructor(config) {
    this.config = config;
    this.ws = null;
    this.connected = false;
    this.currentTaskId = null;
    this.pendingApprovals = new Map();
  }

  connect() {
    return new Promise((resolve, reject) => {
      const url = this.config.hubUrl.replace(/^http/, "ws");
      log("info", `Connecting to Hub: ${url}`);

      this.ws = new WebSocket(url);

      this.ws.on("open", () => {
        log("info", "WebSocket connected");
        this.connected = true;
        this.register();
        resolve();
      });

      this.ws.on("message", (data) => {
        try {
          const msg = JSON.parse(data.toString());
          this.handleMessage(msg);
        } catch (e) {
          log("error", "Failed to parse message:", e.message);
        }
      });

      this.ws.on("close", () => {
        log("warn", "WebSocket disconnected");
        this.connected = false;
        this.currentTaskId = null;
        setTimeout(() => this.connect(), 5000);
      });

      this.ws.on("error", (err) => {
        log("error", "WebSocket error:", err.message);
        reject(err);
      });
    });
  }

  register() {
    this.send({
      method: "agent.register",
      payload: {
        name: this.config.agentName,
        type: "cursor",
        token: this.config.agentToken,
        platform: process.platform,
        host: require("os").hostname(),
        capabilities: ["editor", "mcp", "file", "terminal"],
      },
    });
  }

  send(msg) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg));
    }
  }

  handleMessage(msg) {
    switch (msg.method) {
      case "system.ack":
        log("info", "Registered with Hub:", JSON.stringify(msg.payload));
        break;
      case "system.ping":
        this.send({ method: "agent.heartbeat" });
        break;
      case "task.start":
        this.handleTaskStart(msg.payload);
        break;
      case "task.cancel":
        this.handleTaskCancel(msg.payload);
        break;
      case "approval.response":
        this.handleApprovalResponse(msg.payload);
        break;
      default:
        log("debug", "Unknown message:", msg.method);
    }
  }

  async handleTaskStart(payload) {
    const { task_id, title, prompt, executor, config: taskConfig, steps } = payload;
    this.currentTaskId = task_id;

    log("info", `[Task ${task_id}] Starting: ${title} (executor: ${executor || "default"})`);
    this.sendProgress(task_id, "system", `Task started: ${title}`);

    try {
      if (executor === "llm") {
        await this.executeLLM(task_id, prompt, taskConfig);
      } else if (executor === "shell") {
        await this.executeShell(task_id, prompt, steps, taskConfig);
      } else {
        // executor === "cursor" 或默认
        await this.executeCursor(task_id, prompt, steps, taskConfig);
      }

      this.sendComplete(task_id, "Task completed successfully");
    } catch (err) {
      this.sendError(task_id, err.message);
    }

    this.currentTaskId = null;
  }

  async executeCursor(taskId, prompt, steps, config) {
    const taskFile = path.join(CONFIG.taskDir, `${taskId}.md`);
    const doneFile = path.join(CONFIG.taskDir, `${taskId}.done`);
    const resultFile = path.join(CONFIG.taskDir, `${taskId}.result`);

    fs.mkdirSync(CONFIG.taskDir, { recursive: true });

    // 写任务描述
    const taskContent = `# Task: ${prompt}\n\n## Project: ${config?.project_path || CONFIG.projectPath}\n${config?.target_file ? `## Target File: ${config.target_file}\n` : ""}\n## Instructions\n${prompt}`;
    fs.writeFileSync(taskFile, taskContent);

    log("info", `[Task ${taskId}] Written to ${taskFile}, waiting for Cursor...`);
    this.sendProgress(taskId, "system", `Waiting for Cursor IDE to process task...`);

    // 轮询完成信号
    const timeout = (config?.timeout || 300) * 1000;
    const start = Date.now();
    while (Date.now() - start < timeout) {
      if (fs.existsSync(doneFile)) {
        break;
      }
      await sleep(2000);
    }

    if (!fs.existsSync(doneFile)) {
      throw new Error(`Timeout waiting for Cursor to complete task`);
    }

    // 读取结果
    if (fs.existsSync(resultFile)) {
      const result = fs.readFileSync(resultFile, "utf-8");
      this.sendProgress(taskId, "stdout", result.slice(0, 2000));
    }
  }

  async executeLLM(taskId, prompt, config) {
    const model = config?.model || "gpt-4";
    log("info", `[Task ${taskId}] LLM call with model: ${model}`);
    this.sendProgress(taskId, "system", `Calling LLM: ${model}`);

    try {
      const response = await this.callLLM(prompt, model);
      this.sendProgress(taskId, "stdout", response.content?.slice(0, 2000) || "(empty response)");
    } catch (err) {
      throw new Error(`LLM call failed: ${err.message}`);
    }
  }

  async callLLM(prompt, model) {
    // 通过 Hub API 调用 LLM (如果 Hub API Key 可用)
    if (this.config.hubApiKey) {
      const res = await fetch(`${this.config.hubUrl.replace("/ws/agent", "/api").replace(/^ws/, "http")}/llm/chat`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${this.config.hubApiKey}`,
        },
        body: JSON.stringify({ model, messages: [{ role: "user", content: prompt }] }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return res.json();
    }

    // 兜底: 使用环境变量
    if (model.startsWith("gpt") && process.env.OPENAI_API_KEY) {
      const res = await fetch("https://api.openai.com/v1/chat/completions", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${process.env.OPENAI_API_KEY}`,
        },
        body: JSON.stringify({ model, messages: [{ role: "user", content: prompt }] }),
      });
      if (!res.ok) throw new Error(`OpenAI HTTP ${res.status}`);
      const data = await res.json();
      return { content: data.choices[0]?.message?.content || "", tokens_used: data.usage?.total_tokens };
    }

    throw new Error(`No API key configured for model: ${model}`);
  }

  async executeShell(taskId, prompt, steps, config) {
    const workDir = config?.work_dir || CONFIG.projectPath;
    const timeout = (config?.timeout || 300) * 1000;

    if (steps && Array.isArray(steps)) {
      for (let i = 0; i < steps.length; i++) {
        const step = steps[i];
        this.sendProgress(taskId, "system", `Step ${i + 1}/${steps.length}: ${step.name || step.command}`);

        try {
          const { stdout, stderr } = await execAsync(step.command, {
            cwd: workDir,
            timeout,
            env: { ...process.env, ...(config?.env || {}) },
          });
          if (stdout) this.sendProgress(taskId, "stdout", stdout.slice(0, 1000));
          if (stderr) this.sendProgress(taskId, "stderr", stderr.slice(0, 1000));
        } catch (err) {
          this.sendProgress(taskId, "stderr", `Step failed: ${err.message}`);
        }
      }
    } else {
      const { stdout, stderr } = await execAsync(prompt, {
        cwd: workDir,
        timeout,
        maxBuffer: 10 * 1024 * 1024,
      });
      if (stdout) this.sendProgress(taskId, "stdout", stdout.slice(0, 2000));
      if (stderr) this.sendProgress(taskId, "stderr", stderr.slice(0, 2000));
    }
  }

  handleTaskCancel(payload) {
    const { task_id } = payload;
    log("warn", `[Task ${task_id}] Cancelled by Hub`);
    this.currentTaskId = null;
  }

  handleApprovalResponse(payload) {
    const { task_id, approved, reply } = payload;
    const resolver = this.pendingApprovals.get(task_id);
    if (resolver) {
      resolver({ approved, reply });
      this.pendingApprovals.delete(task_id);
    }
  }

  sendProgress(taskId, type, content) {
    this.send({
      method: "task.progress",
      payload: { task_id, type, content },
    });
  }

  sendComplete(taskId, output) {
    this.send({
      method: "task.complete",
      payload: { task_id, output },
    });
    log("info", `[Task ${taskId}] Completed`);
  }

  sendError(taskId, error) {
    this.send({
      method: "task.error",
      payload: { task_id, error },
    });
    log("error", `[Task ${taskId}] Error: ${error}`);
  }

  // ─── Hub REST API 封装 (供 MCP Tools 调用) ───
  async getTasks() {
    const res = await this.hubFetch("/agent-tasks");
    return res.json();
  }

  async getTaskStatus(taskId) {
    const res = await this.hubFetch(`/agent-tasks/${taskId}`);
    return res.json();
  }

  async getTaskLogs(taskId) {
    const res = await this.hubFetch(`/agent-tasks/${taskId}/logs`);
    return res.json();
  }

  async getKnowledge(query) {
    const res = await this.hubFetch(`/knowledge?q=${encodeURIComponent(query)}`);
    return res.json();
  }

  async reportTaskProgress(taskId, progress) {
    const res = await this.hubFetch(`/agent-tasks/${taskId}/progress`, {
      method: "POST",
      body: JSON.stringify({ progress }),
    });
    return res.json();
  }

  async hubFetch(apiPath, options = {}) {
    const baseUrl = this.config.hubUrl.replace("/ws/agent", "/api").replace(/^ws/, "http");
    const url = `${baseUrl}${apiPath}`;
    const headers = { "Content-Type": "application/json" };
    if (this.config.hubApiKey) {
      headers["Authorization"] = `Bearer ${this.config.hubApiKey}`;
    }
    return fetch(url, { ...options, headers: { ...headers, ...(options.headers || {}) } });
  }
}

// ─── MCP Server ───
async function createMCPServer(hubClient) {
  const server = new McpServer({
    name: "ai-collab-hub-bridge",
    version: "2.0.0",
  });

  // Tools
  server.tool("get_hub_tasks", "获取 Hub 上的 Agent 任务列表", {}, async () => {
    try {
      const tasks = await hubClient.getTasks();
      return { content: [{ type: "text", text: JSON.stringify(tasks, null, 2) }] };
    } catch (e) {
      return { content: [{ type: "text", text: `Error: ${e.message}` }], isError: true };
    }
  });

  server.tool("get_task_status", "查询指定任务的状态和进度", { task_id: z.string().describe("任务 ID") }, async ({ task_id }) => {
    try {
      const task = await hubClient.getTaskStatus(task_id);
      return { content: [{ type: "text", text: JSON.stringify(task, null, 2) }] };
    } catch (e) {
      return { content: [{ type: "text", text: `Error: ${e.message}` }], isError: true };
    }
  });

  server.tool("get_task_logs", "获取任务的执行日志", { task_id: z.string().describe("任务 ID") }, async ({ task_id }) => {
    try {
      const logs = await hubClient.getTaskLogs(task_id);
      return { content: [{ type: "text", text: JSON.stringify(logs, null, 2) }] };
    } catch (e) {
      return { content: [{ type: "text", text: `Error: ${e.message}` }], isError: true };
    }
  });

  server.tool("search_knowledge", "从 Hub 知识库搜索信息", { query: z.string().describe("搜索关键词") }, async ({ query }) => {
    try {
      const results = await hubClient.getKnowledge(query);
      return { content: [{ type: "text", text: JSON.stringify(results, null, 2) }] };
    } catch (e) {
      return { content: [{ type: "text", text: `Error: ${e.message}` }], isError: true };
    }
  });

  server.tool("report_progress", "上报任务进度到 Hub", {
    task_id: z.string().describe("任务 ID"),
    progress: z.number().describe("进度百分比 0-100"),
    message: z.string().describe("进度描述"),
  }, async ({ task_id, progress, message }) => {
    try {
      hubClient.sendProgress(task_id, "system", `[${progress}%] ${message}`);
      return { content: [{ type: "text", text: `Progress reported: ${progress}%` }] };
    } catch (e) {
      return { content: [{ type: "text", text: `Error: ${e.message}` }], isError: true };
    }
  });

  server.tool("read_file", "读取项目文件内容", {
    file_path: z.string().describe("文件路径"),
  }, async ({ file_path }) => {
    try {
      const fullPath = path.isAbsolute(file_path) ? file_path : path.join(CONFIG.projectPath, file_path);
      const content = fs.readFileSync(fullPath, "utf-8");
      return { content: [{ type: "text", text: content.slice(0, 5000) }] };
    } catch (e) {
      return { content: [{ type: "text", text: `Error: ${e.message}` }], isError: true };
    }
  });

  server.tool("write_file", "写入文件内容", {
    file_path: z.string().describe("文件路径"),
    content: z.string().describe("文件内容"),
  }, async ({ file_path, content }) => {
    try {
      const fullPath = path.isAbsolute(file_path) ? file_path : path.join(CONFIG.projectPath, file_path);
      fs.mkdirSync(path.dirname(fullPath), { recursive: true });
      fs.writeFileSync(fullPath, content, "utf-8");
      return { content: [{ type: "text", text: `File written: ${fullPath}` }] };
    } catch (e) {
      return { content: [{ type: "text", text: `Error: ${e.message}` }], isError: true };
    }
  });

  server.tool("run_command", "在项目中执行命令", {
    command: z.string().describe("Shell 命令"),
    work_dir: z.string().describe("工作目录 (可选)").optional(),
  }, async ({ command, work_dir }) => {
    try {
      const { stdout, stderr } = await execAsync(command, {
        cwd: work_dir || CONFIG.projectPath,
        timeout: 60000,
        maxBuffer: 10 * 1024 * 1024,
      });
      let result = "";
      if (stdout) result += `STDOUT:\n${stdout.slice(0, 3000)}\n`;
      if (stderr) result += `STDERR:\n${stderr.slice(0, 3000)}\n`;
      return { content: [{ type: "text", text: result || "(no output)" }] };
    } catch (e) {
      return { content: [{ type: "text", text: `Error: ${e.message}\nOutput: ${e.stdout || ""}\nStderr: ${e.stderr || ""}` }], isError: true };
    }
  });

  // Resources
  server.resource("active_tasks", "hub://tasks/active", async (uri) => {
    try {
      const tasks = await hubClient.getTasks();
      const active = tasks.filter(t => ["running", "queued"].includes(t.status));
      return { contents: [{ uri: uri.href, text: JSON.stringify(active, null, 2) }] };
    } catch (e) {
      return { contents: [{ uri: uri.href, text: `Error: ${e.message}` }] };
    }
  });

  server.resource("project_info", "hub://project/info", async (uri) => {
    try {
      const stat = fs.statSync(CONFIG.projectPath);
      const files = fs.readdirSync(CONFIG.projectPath).slice(0, 50);
      return { contents: [{ uri: uri.href, text: JSON.stringify({ path: CONFIG.projectPath, files, isDirectory: stat.isDirectory() }, null, 2) }] };
    } catch (e) {
      return { contents: [{ uri: uri.href, text: `Error: ${e.message}` }] };
    }
  });

  return server;
}

// ─── 工具函数 ───
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

// ─── 主入口 ───
async function main() {
  log("info", "=== Cursor MCP Bridge v2 ===");
  log("info", `Hub URL: ${CONFIG.hubUrl}`);
  log("info", `Agent Name: ${CONFIG.agentName}`);
  log("info", `Project Path: ${CONFIG.projectPath}`);

  // 创建 Hub Client
  const hubClient = new HubClient(CONFIG);

  // 连接 Hub
  try {
    await hubClient.connect();
  } catch (err) {
    log("error", "Failed to connect to Hub, continuing with MCP only:", err.message);
  }

  // 启动 MCP Server
  const mcpServer = await createMCPServer(hubClient);
  const transport = new StdioServerTransport();

  log("info", "Starting MCP Server (stdio)...");
  await mcpServer.connect(transport);
  log("info", "MCP Server ready. Waiting for Cursor IDE connection...");

  // 保持进程运行
  process.on("SIGINT", () => {
    log("info", "Shutting down...");
    process.exit(0);
  });
}

main().catch(err => {
  log("error", "Fatal error:", err.message);
  process.exit(1);
});
