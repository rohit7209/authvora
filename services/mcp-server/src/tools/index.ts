import * as verifyToken from "./verify-token.js";
import * as createUser from "./create-user.js";
import * as getUser from "./get-user.js";
import * as getRiskScore from "./get-risk-score.js";
import * as simulateAttack from "./simulate-attack.js";
import * as getLoginMetrics from "./get-login-metrics.js";
import * as listSuspiciousIps from "./list-suspicious-ips.js";

export const TOOLS = [
  verifyToken.toolDefinition,
  createUser.toolDefinition,
  getUser.toolDefinition,
  getRiskScore.toolDefinition,
  simulateAttack.toolDefinition,
  getLoginMetrics.toolDefinition,
  listSuspiciousIps.toolDefinition,
] as const;

export const TOOL_EXECUTORS = {
  verify_token: verifyToken,
  create_user: createUser,
  get_user: getUser,
  get_risk_score: getRiskScore,
  simulate_attack: simulateAttack,
  get_login_metrics: getLoginMetrics,
  list_suspicious_ips: listSuspiciousIps,
} as const;
