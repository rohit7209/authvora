import { z } from "zod";

const inputSchema = z.object({
  attack_type: z.string(),
  tenant_id: z.string().uuid(),
  config: z
    .object({
      num_attempts: z.number().optional(),
      target_accounts: z.number().optional(),
      source_ips: z.number().optional(),
      duration_seconds: z.number().optional(),
    })
    .optional(),
  api_key: z.string(),
});

export type SimulateAttackInput = z.infer<typeof inputSchema>;

export const toolDefinition = {
  name: "simulate_attack",
  description: "Run a simulated attack against the authentication system",
  inputSchema: {
    type: "object" as const,
    properties: {
      attack_type: { type: "string", description: "Type of attack to simulate" },
      tenant_id: { type: "string", description: "Tenant ID" },
      config: {
        type: "object",
        description: "Simulation configuration",
        properties: {
          num_attempts: { type: "number", description: "Number of attempts" },
          target_accounts: { type: "number", description: "Target accounts count" },
          source_ips: { type: "number", description: "Number of source IPs" },
          duration_seconds: { type: "number", description: "Duration in seconds" },
        },
      },
      api_key: { type: "string", description: "Agent API key for authentication" },
    },
    required: ["attack_type", "tenant_id", "api_key"],
  },
};

const REQUIRED_SCOPE = "security.simulate_attack";

export { REQUIRED_SCOPE };

export async function execute(
  args: unknown,
  riskEngineUrl: string
): Promise<Record<string, unknown> | { error: string }> {
  const parsed = inputSchema.safeParse(args);
  if (!parsed.success) {
    return { error: parsed.error.message };
  }

  const { attack_type, tenant_id, config } = parsed.data;

  try {
    const url = `${riskEngineUrl}/api/v1/simulate/attack`;
    const res = await fetch(url, {
      method: "POST",
      headers: {
        "X-Tenant-ID": tenant_id,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ attack_type, config: config ?? {} }),
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
