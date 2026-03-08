import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  ListToolsRequestSchema,
  CallToolRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
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
): Promise<{ content: Array<{ type: "text"; text: string }> }> {
  const apiKey = args?.api_key as string | undefined;
  if (!apiKey) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({ error: "Missing api_key in tool arguments" }),
        },
      ],
    };
  }

  if (!dbPool) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            error: "Database not configured (DATABASE_URL required for agent auth)",
          }),
        },
      ],
    };
  }

  const apiKeyHash = hashApiKey(apiKey);
  const agent = await verifyAgentApiKey(apiKeyHash, dbPool);
  if (!agent) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({ error: "Invalid or inactive agent API key" }),
        },
      ],
    };
  }

  const executor = TOOL_EXECUTORS[name as keyof typeof TOOL_EXECUTORS];
  if (!executor) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({ error: `Unknown tool: ${name}` }),
        },
      ],
    };
  }

  const requiredScope = (executor as { REQUIRED_SCOPE?: string }).REQUIRED_SCOPE;
  if (requiredScope && !checkScope(agent, requiredScope)) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            error: `Agent lacks required scope: ${requiredScope}`,
          }),
        },
      ],
    };
  }

  let output: unknown;
  let status = "success";

  try {
    const exec = executor.execute as (a: unknown, b: Pool | string) => Promise<unknown>;
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

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(output),
      },
    ],
  };
}

async function main() {
  const server = new Server(
    {
      name: "authvora-mcp",
      version: "1.0.0",
    },
    {
      capabilities: {
        tools: {},
      },
    }
  );

  server.setRequestHandler(ListToolsRequestSchema, async () => {
    return { tools: buildToolList() };
  });

  server.setRequestHandler(CallToolRequestSchema, async (request) => {
    const { name, arguments: args } = request.params;
    const result = await handleToolCall(name, args ?? {}, db);
    return result;
  });

  const transport = new StdioServerTransport();
  await server.connect(transport);

  console.error(
    JSON.stringify({
      level: "info",
      msg: "Authvora MCP server running (stdio)",
    })
  );
}

main().catch((err) => {
  console.error(
    JSON.stringify({
      level: "error",
      msg: "MCP server failed",
      error: err instanceof Error ? err.message : String(err),
    })
  );
  process.exit(1);
});
