import { Pool } from "pg";
import { query } from "../db.js";

export interface LogAgentActionParams {
  agent_id: string;
  tenant_id: string;
  tool_name: string;
  input: unknown;
  output: unknown;
  status: string;
}

export async function logAgentAction(
  db: Pool,
  params: LogAgentActionParams
): Promise<void> {
  try {
    await query(
      db,
      `INSERT INTO login_events (
        tenant_id,
        user_id,
        event_type,
        ip_address,
        metadata,
        created_at
      ) VALUES ($1, NULL, 'agent_action', NULL, $2, NOW())`,
      [
        params.tenant_id,
        JSON.stringify({
          agent_id: params.agent_id,
          tool_name: params.tool_name,
          input: params.input,
          output: params.output,
          status: params.status,
        }),
      ]
    );
  } catch (err) {
    console.error(
      JSON.stringify({
        level: "error",
        msg: "logAgentAction failed",
        error: err instanceof Error ? err.message : String(err),
      })
    );
  }
}
