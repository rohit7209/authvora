import { z } from "zod";

const inputSchema = z.object({
  user_id: z.string().uuid(),
  tenant_id: z.string().uuid(),
  ip_address: z.string(),
  api_key: z.string(),
});

export type GetRiskScoreInput = z.infer<typeof inputSchema>;

export const toolDefinition = {
  name: "get_risk_score",
  description: "Evaluate the risk score for a user login context",
  inputSchema: {
    type: "object" as const,
    properties: {
      user_id: { type: "string", description: "User ID (UUID)" },
      tenant_id: { type: "string", description: "Tenant ID" },
      ip_address: { type: "string", description: "IP address for risk evaluation" },
      api_key: { type: "string", description: "Agent API key for authentication" },
    },
    required: ["user_id", "tenant_id", "ip_address", "api_key"],
  },
};

const REQUIRED_SCOPE = "risk.evaluate_login";

export { REQUIRED_SCOPE };

export async function execute(
  args: unknown,
  riskEngineUrl: string
): Promise<Record<string, unknown> | { error: string }> {
  const parsed = inputSchema.safeParse(args);
  if (!parsed.success) {
    return { error: parsed.error.message };
  }

  const { user_id, tenant_id, ip_address } = parsed.data;

  try {
    const url = new URL(`${riskEngineUrl}/api/v1/risk/evaluate`);
    url.searchParams.set("user_id", user_id);
    url.searchParams.set("ip_address", ip_address);

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
