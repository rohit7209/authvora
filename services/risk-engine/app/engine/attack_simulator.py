"""Attack simulation for testing risk engine detection capability."""

import logging
import random
import string
import time
import uuid
from typing import Any

import asyncpg

from app.engine.risk_scorer import RiskScorer
from app.models.schemas import (
    AttackSimulationRequest,
    AttackSimulationResponse,
    RiskEvaluationRequest,
)

logger = logging.getLogger(__name__)


def _random_ip() -> str:
    """Generate a random IP address for simulation."""
    return ".".join(str(random.randint(1, 254)) for _ in range(4))


def _random_id(prefix: str = "", length: int = 8) -> str:
    """Generate a random ID string."""
    chars = string.ascii_lowercase + string.digits
    return prefix + "".join(random.choices(chars, k=length))


class AttackSimulator:
    """Simulates attack types and measures detection capability."""

    def __init__(self) -> None:
        """Initialize the attack simulator."""
        self.risk_scorer = RiskScorer()

    async def simulate(
        self,
        request: AttackSimulationRequest,
        db_pool: asyncpg.Pool,
    ) -> AttackSimulationResponse:
        """
        Run attack simulation and return detection metrics.

        Does not persist synthetic events - only measures risk scoring.
        """
        simulation_id = str(uuid.uuid4())
        config = request.config or {}
        num_attempts = config.get("num_attempts", 100)
        target_accounts = config.get("target_accounts", 10)
        source_ips = config.get("source_ips", 5)
        tenant_id = request.tenant_id

        logger.info(
            "Starting simulation: id=%s type=%s tenant=%s attempts=%d",
            simulation_id,
            request.attack_type,
            tenant_id,
            num_attempts,
        )

        try:
            if request.attack_type == "credential_stuffing":
                results = await self._simulate_credential_stuffing(
                    db_pool, tenant_id, num_attempts, target_accounts, source_ips
                )
            elif request.attack_type == "brute_force":
                results = await self._simulate_brute_force(
                    db_pool, tenant_id, num_attempts
                )
            elif request.attack_type == "bot_signup":
                results = await self._simulate_bot_signup(
                    db_pool, tenant_id, num_attempts
                )
            elif request.attack_type == "token_replay":
                results = await self._simulate_token_replay(
                    db_pool, tenant_id, num_attempts
                )
            else:
                results = {
                    "error": f"Unknown attack type: {request.attack_type}",
                    "total_attempts": 0,
                    "blocked": 0,
                    "rate_limited": 0,
                    "flagged_suspicious": 0,
                    "detection_rate": 0.0,
                    "mean_detection_time_ms": 0.0,
                }
        except asyncpg.PostgresError as e:
            logger.error("Simulation database error: %s", e)
            results = {
                "error": str(e),
                "total_attempts": num_attempts,
                "blocked": 0,
                "rate_limited": 0,
                "flagged_suspicious": 0,
                "detection_rate": 0.0,
                "mean_detection_time_ms": 0.0,
            }

        return AttackSimulationResponse(
            simulation_id=simulation_id,
            attack_type=request.attack_type,
            status="completed",
            results=results,
        )

    async def _simulate_credential_stuffing(
        self,
        pool: asyncpg.Pool,
        tenant_id: str,
        num_attempts: int,
        target_accounts: int,
        source_ips: int,
    ) -> dict[str, Any]:
        """Simulate credential stuffing: many IPs, many accounts."""
        ips = [_random_ip() for _ in range(source_ips)]
        user_ids = [f"sim_user_{_random_id()}" for _ in range(target_accounts)]

        blocked = 0
        flagged = 0
        times: list[float] = []

        for i in range(num_attempts):
            user_id = random.choice(user_ids)
            ip = random.choice(ips)
            start = time.perf_counter()
            try:
                response = await self.risk_scorer.evaluate(
                    RiskEvaluationRequest(
                        user_id=user_id,
                        tenant_id=tenant_id,
                        ip_address=ip,
                    ),
                    pool,
                    None,
                )
            except Exception:
                response = None
            elapsed_ms = (time.perf_counter() - start) * 1000
            times.append(elapsed_ms)

            if response:
                if response.action == "block":
                    blocked += 1
                if response.risk_level in ("high", "critical"):
                    flagged += 1

        total = num_attempts
        detection_rate = (blocked + flagged) / total if total > 0 else 0.0
        mean_time = sum(times) / len(times) if times else 0.0

        return {
            "total_attempts": total,
            "blocked": blocked,
            "rate_limited": 0,  # Not implemented in scorer
            "flagged_suspicious": flagged,
            "detection_rate": round(detection_rate, 4),
            "mean_detection_time_ms": round(mean_time, 2),
        }

    async def _simulate_brute_force(
        self, pool: asyncpg.Pool, tenant_id: str, num_attempts: int
    ) -> dict[str, Any]:
        """Simulate brute force: single IP, single account."""
        ip = _random_ip()
        user_id = f"sim_user_{_random_id()}"

        blocked = 0
        flagged = 0
        times: list[float] = []

        for _ in range(num_attempts):
            start = time.perf_counter()
            try:
                response = await self.risk_scorer.evaluate(
                    RiskEvaluationRequest(
                        user_id=user_id,
                        tenant_id=tenant_id,
                        ip_address=ip,
                    ),
                    pool,
                    None,
                )
            except Exception:
                response = None
            elapsed_ms = (time.perf_counter() - start) * 1000
            times.append(elapsed_ms)

            if response:
                if response.action == "block":
                    blocked += 1
                if response.risk_level in ("high", "critical"):
                    flagged += 1

        total = num_attempts
        detection_rate = (blocked + flagged) / total if total > 0 else 0.0
        mean_time = sum(times) / len(times) if times else 0.0

        return {
            "total_attempts": total,
            "blocked": blocked,
            "rate_limited": 0,
            "flagged_suspicious": flagged,
            "detection_rate": round(detection_rate, 4),
            "mean_detection_time_ms": round(mean_time, 2),
        }

    async def _simulate_bot_signup(
        self, pool: asyncpg.Pool, tenant_id: str, num_attempts: int
    ) -> dict[str, Any]:
        """Simulate bot signup: sequential emails, same IP."""
        ip = _random_ip()
        base_email = f"bot_{_random_id()}"

        blocked = 0
        flagged = 0
        times: list[float] = []

        for i in range(num_attempts):
            user_id = f"sim_user_{base_email}_{i}"
            start = time.perf_counter()
            try:
                response = await self.risk_scorer.evaluate(
                    RiskEvaluationRequest(
                        user_id=user_id,
                        tenant_id=tenant_id,
                        ip_address=ip,
                        email=f"{base_email}{i}@example.com",
                    ),
                    pool,
                    None,
                )
            except Exception:
                response = None
            elapsed_ms = (time.perf_counter() - start) * 1000
            times.append(elapsed_ms)

            if response:
                if response.action == "block":
                    blocked += 1
                if response.risk_level in ("high", "critical"):
                    flagged += 1

        total = num_attempts
        detection_rate = (blocked + flagged) / total if total > 0 else 0.0
        mean_time = sum(times) / len(times) if times else 0.0

        return {
            "total_attempts": total,
            "blocked": blocked,
            "rate_limited": 0,
            "flagged_suspicious": flagged,
            "detection_rate": round(detection_rate, 4),
            "mean_detection_time_ms": round(mean_time, 2),
        }

    async def _simulate_token_replay(
        self, pool: asyncpg.Pool, tenant_id: str, num_attempts: int
    ) -> dict[str, Any]:
        """Simulate token reuse: same device fingerprint, different IPs."""
        device_fp = f"fp_{uuid.uuid4().hex[:16]}"
        user_id = f"sim_user_{_random_id()}"

        blocked = 0
        flagged = 0
        times: list[float] = []

        for _ in range(num_attempts):
            ip = _random_ip()
            start = time.perf_counter()
            try:
                response = await self.risk_scorer.evaluate(
                    RiskEvaluationRequest(
                        user_id=user_id,
                        tenant_id=tenant_id,
                        ip_address=ip,
                        device_fingerprint=device_fp,
                    ),
                    pool,
                    None,
                )
            except Exception:
                response = None
            elapsed_ms = (time.perf_counter() - start) * 1000
            times.append(elapsed_ms)

            if response:
                if response.action == "block":
                    blocked += 1
                if response.risk_level in ("high", "critical"):
                    flagged += 1

        total = num_attempts
        detection_rate = (blocked + flagged) / total if total > 0 else 0.0
        mean_time = sum(times) / len(times) if times else 0.0

        return {
            "total_attempts": total,
            "blocked": blocked,
            "rate_limited": 0,
            "flagged_suspicious": flagged,
            "detection_rate": round(detection_rate, 4),
            "mean_detection_time_ms": round(mean_time, 2),
        }
