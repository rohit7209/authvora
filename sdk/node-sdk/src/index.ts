import type { AuthvoraConfig } from "./types.js";
import { AuthvoraClient } from "./client.js";
import { AuthvoraAuth } from "./auth.js";
import { authvoraMiddleware } from "./middleware.js";

export class Authvora {
  public auth: AuthvoraAuth;
  private client: AuthvoraClient;

  constructor(config: AuthvoraConfig) {
    this.client = new AuthvoraClient(config);
    this.auth = new AuthvoraAuth(this.client);
  }

  middleware() {
    return authvoraMiddleware({
      baseUrl: this.client.baseUrl,
      tenantId: this.client.tenantId,
    });
  }
}

export { AuthvoraAuth } from "./auth.js";
export { AuthvoraClient } from "./client.js";
export { authvoraMiddleware } from "./middleware.js";
export * from "./types.js";
