import { createServer, IncomingMessage, ServerResponse } from "http";
import { loadConfig } from "./config.js";
import { createPool } from "./db.js";
import { verifyAgentApiKey, hashApiKey, checkScope } from "./auth/agent-auth.js";
import { logAgentAction } from "./auth/audit.js";
import { TOOLS, TOOL_EXECUTORS } from "./tools/index.js";
import type { Pool } from "pg";

const config = loadConfig();
let db: Pool | null = null;

if (config.databaseUrl) {
  db = createPool(config.databaseUrl);
}

function buildToolList() {
  return TOOLS.map((t) => ({
    name: t.name,
    description: t.description,
    inputSchema: {
      type: "object" as const,
      properties: t.inputSchema.properties,
      required: (t.inputSchema as { required?: string[] }).required ?? [],
    },
  }));
}

async function handleToolCall(
  name: string,
  args: Record<string, unknown>,
  dbPool: Pool | null
): Promise<unknown> {
  const apiKey = args?.api_key as string | undefined;
  if (!apiKey) {
    return { error: "Missing api_key in tool arguments" };
  }

  if (!dbPool) {
    return {
      error:
        "Database not configured (DATABASE_URL required for agent auth)",
    };
  }

  const apiKeyHash = hashApiKey(apiKey);
  const agent = await verifyAgentApiKey(apiKeyHash, dbPool);
  if (!agent) {
    return { error: "Invalid or inactive agent API key" };
  }

  const executor = TOOL_EXECUTORS[name as keyof typeof TOOL_EXECUTORS];
  if (!executor) {
    return { error: `Unknown tool: ${name}` };
  }

  const requiredScope = (executor as { REQUIRED_SCOPE?: string }).REQUIRED_SCOPE;
  if (requiredScope && !checkScope(agent, requiredScope)) {
    return { error: `Agent lacks required scope: ${requiredScope}` };
  }

  let output: unknown;
  let status = "success";

  try {
    const exec = executor.execute as (
      a: unknown,
      b: Pool | string
    ) => Promise<unknown>;
    if (name === "get_user") {
      output = await exec(args, dbPool);
    } else if (
      name === "verify_token" ||
      name === "create_user" ||
      name === "get_login_metrics" ||
      name === "list_suspicious_ips"
    ) {
      output = await exec(args, config.authServiceUrl);
    } else if (name === "get_risk_score" || name === "simulate_attack") {
      output = await exec(args, config.riskEngineUrl);
    } else {
      output = await exec(args, config.authServiceUrl);
    }
  } catch (err) {
    status = "error";
    output = { error: err instanceof Error ? err.message : String(err) };
  }

  const tenantId = (args?.tenant_id as string) ?? agent.tenant_id;
  logAgentAction(dbPool, {
    agent_id: agent.id,
    tenant_id: tenantId,
    tool_name: name,
    input: args,
    output,
    status,
  }).catch(() => {});

  return output;
}

async function parseJsonBody(req: IncomingMessage): Promise<unknown> {
  return new Promise((resolve, reject) => {
    let body = "";
    req.on("data", (chunk) => {
      body += chunk.toString();
    });
    req.on("end", () => {
      try {
        resolve(body ? JSON.parse(body) : {});
      } catch {
        reject(new Error("Invalid JSON"));
      }
    });
    req.on("error", reject);
  });
}

function sendJson(res: ServerResponse, statusCode: number, data: unknown) {
  res.writeHead(statusCode, {
    "Content-Type": "application/json",
  });
  res.end(JSON.stringify(data));
}

const server = createServer(async (req: IncomingMessage, res: ServerResponse) => {
  const url = req.url ?? "/";
  const method = req.method ?? "GET";

  if (method === "GET" && url === "/health") {
    sendJson(res, 200, { status: "ok", service: "mcp-server" });
    return;
  }

  if (method === "POST" && url === "/mcp/tools/list") {
    sendJson(res, 200, { tools: buildToolList() });
    return;
  }

  if (method === "POST" && url === "/mcp/tools/call") {
    try {
      const body = (await parseJsonBody(req)) as {
        tool_name?: string;
        arguments?: Record<string, unknown>;
        api_key?: string;
      };

      const toolName = body?.tool_name;
      const args = body?.arguments ?? {};
      const apiKey = body?.api_key;

      if (!toolName) {
        sendJson(res, 400, { error: "Missing tool_name" });
        return;
      }

      if (apiKey) {
        args.api_key = apiKey;
      }

      const result = await handleToolCall(toolName, args, db);
      sendJson(res, 200, result);
    } catch (err) {
      sendJson(res, 500, {
        error: err instanceof Error ? err.message : String(err),
      });
    }
    return;
  }

  sendJson(res, 404, { error: "Not found" });
});

server.listen(config.port, () => {
  console.log(
    JSON.stringify({
      level: "info",
      msg: `Authvora MCP HTTP server listening on port ${config.port}`,
    })
  );
});
