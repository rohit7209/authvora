import { z } from "zod";

const inputSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
  name: z.string().optional(),
  tenant_id: z.string().uuid(),
  api_key: z.string(),
});

export type CreateUserInput = z.infer<typeof inputSchema>;

export const toolDefinition = {
  name: "create_user",
  description: "Create a new user account with email and password",
  inputSchema: {
    type: "object" as const,
    properties: {
      email: { type: "string", description: "User email address" },
      password: { type: "string", description: "User password (min 8 chars)" },
      name: { type: "string", description: "Optional display name" },
      tenant_id: { type: "string", description: "Tenant ID" },
      api_key: { type: "string", description: "Agent API key for authentication" },
    },
    required: ["email", "password", "tenant_id", "api_key"],
  },
};

const REQUIRED_SCOPE = "auth.create_user";

export { REQUIRED_SCOPE };

export async function execute(
  args: unknown,
  authServiceUrl: string
): Promise<{ user_id: string; email: string; created: boolean } | { error: string }> {
  const parsed = inputSchema.safeParse(args);
  if (!parsed.success) {
    return { error: parsed.error.message };
  }

  const { email, password, name, tenant_id } = parsed.data;

  try {
    const url = `${authServiceUrl}/auth/register`;
    const res = await fetch(url, {
      method: "POST",
      headers: {
        "X-Tenant-ID": tenant_id,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ email, password, name }),
    });

    const body = await res.json();

    if (res.ok) {
      const userId = body.user?.id ?? body.id;
      return {
        user_id: userId,
        email,
        created: true,
      };
    }

    const errorMsg = body.error ?? body.message ?? `HTTP ${res.status}`;
    return { error: errorMsg };
  } catch (err) {
    return {
      error: err instanceof Error ? err.message : String(err),
    };
  }
}
