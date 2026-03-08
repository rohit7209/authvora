import { z } from "zod";

const inputSchema = z.object({
  tenant_id: z.string().uuid(),
  limit: z.number().min(1).max(100).optional(),
  api_key: z.string(),
});

export type ListSuspiciousIpsInput = z.infer<typeof inputSchema>;

export const toolDefinition = {
  name: "list_suspicious_ips",
  description: "List suspicious IP addresses flagged by the system",
  inputSchema: {
    type: "object" as const,
    properties: {
      tenant_id: { type: "string", description: "Tenant ID" },
      limit: {
        type: "number",
        description: "Max number of IPs to return (default: 10)",
      },
      api_key: { type: "string", description: "Agent API key for authentication" },
    },
    required: ["tenant_id", "api_key"],
  },
};

const REQUIRED_SCOPE = "analytics.query_events";

export { REQUIRED_SCOPE };

export async function execute(
  args: unknown,
  authServiceUrl: string
): Promise<Record<string, unknown> | { error: string }> {
  const parsed = inputSchema.safeParse(args);
  if (!parsed.success) {
    return { error: parsed.error.message };
  }

  const { tenant_id, limit = 10 } = parsed.data;

  try {
    const url = new URL(`${authServiceUrl}/analytics/suspicious-ips`);
    url.searchParams.set("limit", String(limit));

    const res = await fetch(url.toString(), {
      method: "GET",
      headers: {
        "X-Tenant-ID": tenant_id,
        "Content-Type": "application/json",
      },
    });

    const body = await res.json();

    if (res.ok) {
      return body;
    }

    const errorMsg = body.error ?? body.message ?? `HTTP ${res.status}`;
    return { error: errorMsg };
  } catch (err) {
    return {
      error: err instanceof Error ? err.message : String(err),
    };
  }
}
