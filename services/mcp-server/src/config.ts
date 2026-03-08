export interface Config {
  port: number;
  authServiceUrl: string;
  riskEngineUrl: string;
  apiGatewayUrl: string;
  databaseUrl: string;
  logLevel: string;
}

function getEnv(key: string, defaultValue: string): string {
  const value = process.env[key];
  return value !== undefined && value !== "" ? value : defaultValue;
}

export function loadConfig(): Config {
  return {
    port: parseInt(getEnv("PORT", "8084"), 10),
    authServiceUrl: getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
    riskEngineUrl: getEnv("RISK_ENGINE_URL", "http://localhost:8082"),
    apiGatewayUrl: getEnv("API_GATEWAY_URL", "http://localhost:8080"),
    databaseUrl: getEnv("DATABASE_URL", ""),
    logLevel: getEnv("LOG_LEVEL", "info"),
  };
}
