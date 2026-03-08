import { createHash } from "crypto";
import { Pool } from "pg";
import { query } from "../db.js";

export interface Agent {
  id: string;
  tenant_id: string;
  agent_name: string;
  agent_type: string;
  scopes: string[];
  status: string;
}

export function hashApiKey(apiKey: string): string {
  return createHash("sha256").update(apiKey).digest("hex");
}

export async function verifyAgentApiKey(
  apiKeyHash: string,
  db: Pool
): Promise<Agent | null> {
  try {
    const result = await query<{
      id: string;
      tenant_id: string;
      agent_name: string;
      agent_type: string;
      scopes: string[];
      status: string;
    }>(
      db,
      `SELECT id, tenant_id, agent_name, agent_type, scopes, status
       FROM agents
       WHERE api_key_hash = $1 AND status = 'active'`,
      [apiKeyHash]
    );

    if (result.rows.length === 0) {
      return null;
    }

    const row = result.rows[0];
    return {
      id: row.id,
      tenant_id: row.tenant_id,
      agent_name: row.agent_name,
      agent_type: row.agent_type,
      scopes: Array.isArray(row.scopes) ? row.scopes : [],
      status: row.status,
    };
  } catch (err) {
    console.error(
      JSON.stringify({
        level: "error",
        msg: "verifyAgentApiKey failed",
        error: err instanceof Error ? err.message : String(err),
      })
    );
    return null;
  }
}

export function checkScope(agent: Agent, requiredScope: string): boolean {
  return agent.scopes.includes(requiredScope);
}
