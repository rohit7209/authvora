import { z } from "zod";

const inputSchema = z.object({
  token: z.string(),
  tenant_id: z.string().uuid(),
  api_key: z.string(),
});

export type VerifyTokenInput = z.infer<typeof inputSchema>;

export const toolDefinition = {
  name: "verify_token",
  description: "Verify a JWT access token and return its claims",
  inputSchema: {
    type: "object" as const,
    properties: {
      token: { type: "string", description: "JWT access token to verify" },
      tenant_id: { type: "string", description: "Tenant ID" },
      api_key: { type: "string", description: "Agent API key for authentication" },
    },
    required: ["token", "tenant_id", "api_key"],
  },
};

const REQUIRED_SCOPE = "auth.verify_token";

export { REQUIRED_SCOPE };

export async function execute(
  args: unknown,
  authServiceUrl: string
): Promise<{ valid: boolean; claims?: object; error?: string }> {
  const parsed = inputSchema.safeParse(args);
  if (!parsed.success) {
    return { valid: false, error: parsed.error.message };
  }

  const { token, tenant_id } = parsed.data;

  try {
    const url = `${authServiceUrl}/users/me`;
    const res = await fetch(url, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
        "X-Tenant-ID": tenant_id,
        "Content-Type": "application/json",
      },
    });

    if (res.ok) {
      const user = await res.json();
      return { valid: true, claims: user };
    }

    const body = await res.text();
    let errorMsg = `HTTP ${res.status}`;
    try {
      const errJson = JSON.parse(body);
      if (errJson.error) errorMsg = errJson.error;
      if (errJson.message) errorMsg = errJson.message;
    } catch {
      if (body) errorMsg = body.slice(0, 200);
    }
    return { valid: false, error: errorMsg };
  } catch (err) {
    return {
      valid: false,
      error: err instanceof Error ? err.message : String(err),
    };
  }
}
