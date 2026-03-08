import { z } from "zod";
import { Pool } from "pg";
import { query } from "../db.js";

const inputSchema = z.object({
  user_id: z.string().uuid(),
  tenant_id: z.string().uuid(),
  api_key: z.string(),
});

export type GetUserInput = z.infer<typeof inputSchema>;

export const toolDefinition = {
  name: "get_user",
  description: "Get user details by user ID",
  inputSchema: {
    type: "object" as const,
    properties: {
      user_id: { type: "string", description: "User ID (UUID)" },
      tenant_id: { type: "string", description: "Tenant ID" },
      api_key: { type: "string", description: "Agent API key for authentication" },
    },
    required: ["user_id", "tenant_id", "api_key"],
  },
};

const REQUIRED_SCOPE = "auth.get_user";

export { REQUIRED_SCOPE };

export async function execute(
  args: unknown,
  db: Pool
): Promise<Record<string, unknown> | { error: string }> {
  const parsed = inputSchema.safeParse(args);
  if (!parsed.success) {
    return { error: parsed.error.message };
  }

  const { user_id, tenant_id } = parsed.data;

  try {
    const result = await query<{
      id: string;
      tenant_id: string;
      email: string;
      email_verified: boolean;
      name: string | null;
      avatar_url: string | null;
      status: string;
      metadata: object;
      created_at: Date;
      updated_at: Date;
    }>(
      db,
      `SELECT id, tenant_id, email, email_verified, name, avatar_url, status, metadata, created_at, updated_at
       FROM users
       WHERE id = $1 AND tenant_id = $2`,
      [user_id, tenant_id]
    );

    if (result.rows.length === 0) {
      return { error: "User not found" };
    }

    const row = result.rows[0];
    return {
      id: row.id,
      tenant_id: row.tenant_id,
      email: row.email,
      email_verified: row.email_verified,
      name: row.name,
      avatar_url: row.avatar_url,
      status: row.status,
      metadata: row.metadata,
      created_at: row.created_at?.toISOString(),
      updated_at: row.updated_at?.toISOString(),
    };
  } catch (err) {
    return {
      error: err instanceof Error ? err.message : String(err),
    };
  }
}
