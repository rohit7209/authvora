import type { AuthvoraConfig, User } from "./types.js";

function decodeJWTPayload(token: string): Record<string, unknown> | null {
  const parts = token.split(".");
  if (parts.length !== 3) return null;
  try {
    const payload = Buffer.from(parts[1], "base64url").toString("utf8");
    return JSON.parse(payload) as Record<string, unknown>;
  } catch {
    return null;
  }
}

export function authvoraMiddleware(_config: AuthvoraConfig) {
  return async (req: { headers?: { authorization?: string }; authvora?: unknown }, res: { status: (code: number) => { json: (body: unknown) => void } }, next: () => void) => {
    const authHeader = req.headers?.authorization;
    if (!authHeader?.startsWith("Bearer ")) {
      return res.status(401).json({
        error: { code: "TOKEN_MISSING", message: "Authorization header required", status: 401 },
      });
    }

    const token = authHeader.substring(7);
    try {
      const payload = decodeJWTPayload(token);

      if (!payload || !payload.sub || !payload.tid) {
        throw new Error("Invalid token claims");
      }

      if (typeof payload.exp === "number" && payload.exp < Math.floor(Date.now() / 1000)) {
        return res.status(401).json({
          error: { code: "TOKEN_EXPIRED", message: "Access token expired", status: 401 },
        });
      }

      const user: User = {
        id: String(payload.sub),
        email: (payload.email as string) ?? "",
        name: (payload.name as string) ?? null,
        email_verified: Boolean(payload.email_verified),
        avatar_url: (payload.avatar_url as string) ?? null,
        created_at: (payload.created_at as string) ?? "",
        updated_at: (payload.updated_at as string) ?? "",
      };

      req.authvora = {
        user,
        token,
        tenantId: String(payload.tid),
      };
      next();
    } catch {
      return res.status(401).json({
        error: { code: "TOKEN_INVALID", message: "Invalid access token", status: 401 },
      });
    }
  };
}
