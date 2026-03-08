import { z } from "zod";

const inputSchema = z.object({
  tenant_id: z.string().uuid(),
  period: z.enum(["24h", "7d", "30d"]).optional(),
  api_key: z.string(),
});

export type GetLoginMetricsInput = z.infer<typeof inputSchema>;

export const toolDefinition = {
  name: "get_login_metrics",
  description: "Get login success/failure metrics for a tenant",
  inputSchema: {
    type: "object" as const,
    properties: {
      tenant_id: { type: "string", description: "Tenant ID" },
      period: {
        type: "string",
        enum: ["24h", "7d", "30d"],
        description: "Time period for metrics (default: 24h)",
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

  const { tenant_id, period = "24h" } = parsed.data;

  try {
    const url = new URL(`${authServiceUrl}/analytics/login-metrics`);
    url.searchParams.set("period", period);

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
