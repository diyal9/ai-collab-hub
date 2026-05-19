#!/usr/bin/env python3
"""
AI Collab Hub MCP Server
Allows Cursor, Claude Code, and other MCP-compatible AI IDEs to interact with the Hub.

Usage:
  1. Install: pip install mcp
  2. Configure in Cursor (.cursor/mcp.json):
     {
       "mcpServers": {
         "ai-collab-hub": {
           "command": "python3",
           "args": ["/path/to/hub_mcp_server.py"],
           "env": {
             "HUB_URL": "http://localhost:8087",
             "HUB_TOKEN": "your_jwt_token"
           }
         }
       }
     }
"""

import os
import sys
import json
import logging
from typing import Optional
import requests

# Configure logging
logging.basicConfig(level=logging.INFO, stream=sys.stderr)
logger = logging.getLogger("hub-mcp")

# MCP imports
try:
    from mcp.server import Server
    from mcp.types import Tool, TextContent
except ImportError:
    logger.error("mcp package not installed. Run: pip install mcp")
    sys.exit(1)

# Configuration
HUB_URL = os.environ.get("HUB_URL", "http://localhost:8087")
HUB_TOKEN = os.environ.get("HUB_TOKEN", "")

# Initialize MCP Server
app = Server("ai-collab-hub")

def api_request(method: str, path: str, json_data: dict = None) -> dict:
    """Make authenticated request to Hub API."""
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {HUB_TOKEN}" if HUB_TOKEN else "",
    }
    url = f"{HUB_URL}{path}"
    
    try:
        if method == "GET":
            resp = requests.get(url, headers=headers, timeout=30)
        elif method == "POST":
            resp = requests.post(url, headers=headers, json=json_data, timeout=30)
        else:
            raise ValueError(f"Unsupported method: {method}")
        
        resp.raise_for_status()
        return resp.json()
    except Exception as e:
        logger.error(f"API request failed: {e}")
        return {"error": str(e)}

# ─── MCP Tools ───

@app.tool()
async def list_tasks(status: Optional[str] = None, limit: int = 20) -> str:
    """List tasks from AI Collab Hub.
    
    Args:
        status: Filter by status (queued, running, completed, failed)
        limit: Maximum number of tasks to return (default: 20)
    
    Returns:
        JSON string of task list
    """
    params = {"limit": limit}
    if status:
        params["status"] = status
    
    result = api_request("GET", "/api/agent-tasks")
    
    if "error" in result:
        return json.dumps({"error": f"Failed to fetch tasks: {result['error']}"}, indent=2)
    
    tasks = result if isinstance(result, list) else result.get("data", [])
    if status:
        tasks = [t for t in tasks if t.get("status") == status]
    
    tasks = tasks[:limit]
    
    output = []
    for t in tasks:
        output.append({
            "id": t.get("id", "?"),
            "title": t.get("title", ""),
            "status": t.get("status", ""),
            "agent_id": t.get("agent_id", ""),
            "parent_task_id": t.get("parent_task_id", ""),
            "required_skills": t.get("required_skills", ""),
            "created_at": t.get("created_at", ""),
        })
    
    return json.dumps({"tasks": output, "count": len(output)}, indent=2, ensure_ascii=False)

@app.tool()
async def get_task(task_id: str) -> str:
    """Get details of a specific task.
    
    Args:
        task_id: The task ID to look up
    
    Returns:
        JSON string of task details including logs
    """
    result = api_request("GET", f"/api/agent-tasks/{task_id}")
    if "error" in result:
        return json.dumps({"error": f"Task not found: {result['error']}"}, indent=2)
    
    logs = api_request("GET", f"/api/agent-tasks/{task_id}/logs")
    
    return json.dumps({
        "task": result,
        "logs": logs.get("data", []) if isinstance(logs, dict) else logs[:50],
    }, indent=2, ensure_ascii=False)

@app.tool()
async def create_task(title: str, prompt: str, agent_id: Optional[int] = None, 
                     required_skills: Optional[str] = None, parent_task_id: Optional[str] = None,
                     timeout_minutes: int = 30) -> str:
    """Create a new task in AI Collab Hub.
    
    Args:
        title: Task title
        prompt: Task instruction/prompt for the agent
        agent_id: Optional specific agent ID to assign
        required_skills: Optional JSON array of required skills (e.g., '["terminal", "file"]')
        parent_task_id: Optional parent task ID for context chain
        timeout_minutes: Task timeout in minutes (default: 30)
    
    Returns:
        JSON string of created task
    """
    payload = {
        "title": title,
        "prompt": prompt,
        "priority": 1,
        "timeout_minutes": timeout_minutes,
    }
    
    if agent_id:
        payload["agent_id"] = agent_id
    if required_skills:
        payload["required_skills"] = required_skills
    if parent_task_id:
        payload["parent_task_id"] = parent_task_id
    
    result = api_request("POST", "/api/agent-tasks", payload)
    
    if "error" in result:
        return json.dumps({"error": f"Failed to create task: {result['error']}"}, indent=2)
    
    return json.dumps({
        "message": "Task created successfully",
        "task": result,
    }, indent=2, ensure_ascii=False)

@app.tool()
async def complete_task(task_id: str, output: str) -> str:
    """Mark a task as completed with output.
    
    Use this when you've finished working on a task assigned to you.
    
    Args:
        task_id: The task ID to complete
        output: The output/result of the task
    
    Returns:
        JSON string of completion status
    """
    payload = {
        "task_id": task_id,
        "output": output,
    }
    
    result = api_request("POST", "/api/agent/callback", {
        "task_id": task_id,
        "status": "completed",
        "output": output,
    })
    
    if "error" in result:
        return json.dumps({"error": f"Failed to complete task: {result['error']}"}, indent=2)
    
    return json.dumps({
        "message": f"Task {task_id} marked as completed",
        "result": result,
    }, indent=2, ensure_ascii=False)

@app.tool()
async def list_flows() -> str:
    """List all workflow flows in AI Collab Hub.
    
    Returns:
        JSON string of flow list
    """
    result = api_request("GET", "/api/flows")
    
    if "error" in result:
        return json.dumps({"error": f"Failed to fetch flows: {result['error']}"}, indent=2)
    
    flows = result if isinstance(result, list) else result.get("data", [])
    
    output = []
    for f in flows:
        output.append({
            "id": f.get("id"),
            "name": f.get("name", ""),
            "description": f.get("description", ""),
            "type": f.get("type", ""),
            "status": f.get("status", ""),
        })
    
    return json.dumps({"flows": output, "count": len(output)}, indent=2, ensure_ascii=False)

@app.tool()
async def execute_flow(flow_id: int) -> str:
    """Execute a workflow flow.
    
    Args:
        flow_id: The flow ID to execute
    
    Returns:
        JSON string of execution status
    """
    result = api_request("POST", f"/api/flows/{flow_id}/execute")
    
    if "error" in result:
        return json.dumps({"error": f"Failed to execute flow: {result['error']}"}, indent=2)
    
    return json.dumps({
        "message": f"Flow {flow_id} execution started",
        "result": result,
    }, indent=2, ensure_ascii=False)

@app.tool()
async def get_agent_instances() -> str:
    """List all registered agent instances.
    
    Returns:
        JSON string of agent instances
    """
    result = api_request("GET", "/api/agent-instances")
    
    if "error" in result:
        return json.dumps({"error": f"Failed to fetch agents: {result['error']}"}, indent=2)
    
    agents = result if isinstance(result, list) else result.get("data", [])
    
    output = []
    for a in agents:
        output.append({
            "id": a.get("id"),
            "name": a.get("name", ""),
            "type": a.get("type", ""),
            "status": a.get("status", ""),
            "capabilities": a.get("capabilities", ""),
            "platform": a.get("platform", ""),
        })
    
    return json.dumps({"agents": output, "count": len(output)}, indent=2, ensure_ascii=False)

# ─── MCP Server Entry Point ───

async def main():
    """Run the MCP server."""
    from mcp.server.stdio import stdio_server
    
    async with stdio_server() as (read_stream, write_stream):
        await app.run(read_stream, write_stream)

if __name__ == "__main__":
    import asyncio
    asyncio.run(main())
