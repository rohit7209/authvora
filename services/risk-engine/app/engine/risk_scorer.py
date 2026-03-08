"""Core risk evaluation engine."""

import logging
from datetime import datetime
from typing import Any

import asyncpg
from redis.asyncio import Redis

from app.engine.geo import calculate_travel_speed, haversine, is_impossible_travel
from app.models.schemas import (
    RiskEvaluationRequest,
    RiskEvaluationResponse,
    RiskSignals,
)

logger = logging.getLogger(__name__)

# Weights for weighted average (must sum to 1.0)
IP_WEIGHT = 0.25
GEO_WEIGHT = 0.25
TIME_WEIGHT = 0.15
DEVICE_WEIGHT = 0.15
TRAVEL_WEIGHT = 0.20

# Off-hours in UTC: 00:00-05:59 and 22:00-23:59
OFF_HOUR_START = 0
OFF_HOUR_END = 6
LATE_OFF_HOUR_START = 22


class RiskScorer:
    """Evaluates login risk based on multiple signals."""

    def __init__(self) -> None:
        """Initialize the risk scorer."""
        pass

    async def evaluate(
        self,
        request: RiskEvaluationRequest,
        db_pool: asyncpg.Pool,
        redis_client: Redis[Any] | None = None,
    ) -> RiskEvaluationResponse:
        """
        Evaluate risk for a login attempt.

        Computes individual signal scores and weighted average.
        Handles missing data gracefully with moderate defaults.
        """
        details: dict[str, Any] = {}
        signals = RiskSignals()

        try:
            # IP Risk (weight 0.25)
            ip_risk = await self._compute_ip_risk(
                request.tenant_id, request.ip_address, request.user_id, db_pool
            )
            signals.ip_risk = ip_risk
            details["ip_risk_reason"] = self._ip_risk_reason(ip_risk)

            # Geo Risk (weight 0.25)
            geo_risk = await self._compute_geo_risk(
                request.user_id, request.tenant_id, request.ip_address, db_pool
            )
            signals.geo_risk = geo_risk
            details["geo_risk_reason"] = self._geo_risk_reason(geo_risk)

            # Time Risk (weight 0.15)
            time_risk = await self._compute_time_risk(
                request.user_id, request.tenant_id, db_pool
            )
            signals.time_risk = time_risk
            details["time_risk_reason"] = self._time_risk_reason(time_risk)

            # Device Risk (weight 0.15)
            device_risk = await self._compute_device_risk(
                request.user_id,
                request.tenant_id,
                request.device_fingerprint,
                db_pool,
            )
            signals.device_risk = device_risk
            details["device_risk_reason"] = self._device_risk_reason(device_risk)

            # Travel Risk (weight 0.20)
            travel_risk = await self._compute_travel_risk(
                request.user_id,
                request.tenant_id,
                request.ip_address,
                db_pool,
            )
            signals.travel_risk = travel_risk
            details["travel_risk_reason"] = self._travel_risk_reason(travel_risk)

        except asyncpg.PostgresError as e:
            logger.error("Database error during risk evaluation: %s", e)
            # Use moderate defaults on DB error
            signals = RiskSignals(ip_risk=40, geo_risk=30, time_risk=20, device_risk=30, travel_risk=20)
            details["error"] = "Database error, using default scores"

        # Compute weighted average
        risk_score = int(
            signals.ip_risk * IP_WEIGHT
            + signals.geo_risk * GEO_WEIGHT
            + signals.time_risk * TIME_WEIGHT
            + signals.device_risk * DEVICE_WEIGHT
            + signals.travel_risk * TRAVEL_WEIGHT
        )
        risk_score = min(100, max(0, risk_score))

        risk_level, action = self._score_to_level_and_action(risk_score)
        details["risk_score_breakdown"] = {
            "weighted_components": {
                "ip": round(signals.ip_risk * IP_WEIGHT, 1),
                "geo": round(signals.geo_risk * GEO_WEIGHT, 1),
                "time": round(signals.time_risk * TIME_WEIGHT, 1),
                "device": round(signals.device_risk * DEVICE_WEIGHT, 1),
                "travel": round(signals.travel_risk * TRAVEL_WEIGHT, 1),
            }
        }

        logger.info(
            "Risk evaluation: user=%s tenant=%s score=%d level=%s action=%s",
            request.user_id,
            request.tenant_id,
            risk_score,
            risk_level,
            action,
        )

        return RiskEvaluationResponse(
            risk_score=risk_score,
            risk_level=risk_level,
            action=action,
            signals=signals,
            details=details,
        )

    async def _compute_ip_risk(
        self, tenant_id: str, ip_address: str, user_id: str, pool: asyncpg.Pool
    ) -> int:
        """Compute IP risk: suspicious IP=80, new for user=40, else 10."""
        async with pool.acquire() as conn:
            # Check if IP has >10 failed logins (suspicious)
            suspicious = await conn.fetchval(
                """
                SELECT COUNT(*) > 10
                FROM login_events
                WHERE tenant_id = $1 AND ip_address = $2 AND success = false
                """,
                tenant_id,
                ip_address,
            )
            if suspicious:
                return 80

            # Check if IP is new for this user
            seen = await conn.fetchval(
                """
                SELECT EXISTS(
                    SELECT 1 FROM login_events
                    WHERE tenant_id = $1 AND user_id = $2 AND ip_address = $3
                )
                """,
                tenant_id,
                user_id,
                ip_address,
            )
            if not seen:
                return 40

        return 10

    def _ip_risk_reason(self, score: int) -> str:
        if score >= 80:
            return "IP has many failed login attempts"
        if score >= 40:
            return "IP is new for this user"
        return "IP is known to user"

    async def _compute_geo_risk(
        self, user_id: str, tenant_id: str, ip_address: str, pool: asyncpg.Pool
    ) -> int:
        """Compute geo risk: different country=70, no history=20, same=5."""
        async with pool.acquire() as conn:
            # Get user's most common country from ip_history or login_events
            user_country = await conn.fetchval(
                """
                SELECT country FROM (
                    SELECT country, COUNT(*) as cnt
                    FROM ip_history
                    WHERE tenant_id = $1 AND user_id = $2 AND country IS NOT NULL
                    GROUP BY country
                    ORDER BY cnt DESC
                    LIMIT 1
                ) sub
                """,
                tenant_id,
                user_id,
            )

            # Fallback: derive from login_events if ip_history empty
            if user_country is None:
                user_country = await conn.fetchval(
                    """
                    SELECT country FROM (
                        SELECT country, COUNT(*) as cnt
                        FROM login_events
                        WHERE tenant_id = $1 AND user_id = $2 AND country IS NOT NULL
                        GROUP BY country
                        ORDER BY cnt DESC
                        LIMIT 1
                    ) sub
                    """,
                    tenant_id,
                    user_id,
                )

            # Get current IP's country from ip_history or login_events
            current_country = await conn.fetchval(
                """
                SELECT country FROM ip_history
                WHERE tenant_id = $1 AND ip_address = $2 AND country IS NOT NULL
                ORDER BY created_at DESC
                LIMIT 1
                """,
                tenant_id,
                ip_address,
            )
            if current_country is None:
                current_country = await conn.fetchval(
                    """
                    SELECT country FROM login_events
                    WHERE tenant_id = $1 AND ip_address = $2 AND country IS NOT NULL
                    ORDER BY created_at DESC
                    LIMIT 1
                    """,
                    tenant_id,
                    ip_address,
                )

        if user_country is None:
            return 20  # No history
        if current_country is None:
            return 30  # Can't determine current location, moderate risk
        if current_country != user_country:
            return 70  # Different country
        return 5  # Same country

    def _geo_risk_reason(self, score: int) -> str:
        if score >= 70:
            return "Login from different country than usual"
        if score >= 20:
            return "Unable to verify location or no prior history"
        return "Login from usual country"

    async def _compute_time_risk(
        self, user_id: str, tenant_id: str, pool: asyncpg.Pool
    ) -> int:
        """Compute time risk: outside typical hours=50, else 5."""
        current_hour = datetime.utcnow().hour
        is_off_hours = (
            OFF_HOUR_START <= current_hour < OFF_HOUR_END
            or current_hour >= LATE_OFF_HOUR_START
        )

        if is_off_hours:
            # Optionally verify against user's typical hours
            async with pool.acquire() as conn:
                typical_row = await conn.fetchrow(
                    """
                    SELECT EXTRACT(HOUR FROM created_at)::int as typical_hour
                    FROM login_events
                    WHERE tenant_id = $1 AND user_id = $2 AND success = true
                    GROUP BY EXTRACT(HOUR FROM created_at)::int
                    ORDER BY COUNT(*) DESC
                    LIMIT 1
                    """,
                    tenant_id,
                    user_id,
                )
                typical = typical_row["typical_hour"] if typical_row else None
            # If we have typical hour and current is far from it, score 50
            if typical is not None:
                diff = abs(current_hour - typical)
                if diff > 6 or (diff > 12 and diff < 18):  # Off-hours
                    return 50
            else:
                return 50  # No history, off-hours is risky

        return 5

    def _time_risk_reason(self, score: int) -> str:
        if score >= 50:
            return "Login outside typical hours"
        return "Login during typical hours"

    async def _compute_device_risk(
        self,
        user_id: str,
        tenant_id: str,
        device_fingerprint: str | None,
        pool: asyncpg.Pool,
    ) -> int:
        """Compute device risk: unknown device=60, no fingerprint=30, known=5."""
        if device_fingerprint is None or device_fingerprint.strip() == "":
            return 30

        async with pool.acquire() as conn:
            known = await conn.fetchval(
                """
                SELECT EXISTS(
                    SELECT 1 FROM devices
                    WHERE tenant_id = $1 AND user_id = $2 AND device_fingerprint = $3
                )
                """,
                tenant_id,
                user_id,
                device_fingerprint,
            )
        if not known:
            return 60
        return 5

    def _device_risk_reason(self, score: int) -> str:
        if score >= 60:
            return "Unknown device"
        if score >= 30:
            return "No device fingerprint provided"
        return "Known device"

    async def _compute_travel_risk(
        self, user_id: str, tenant_id: str, ip_address: str, pool: asyncpg.Pool
    ) -> int:
        """Compute travel risk: impossible travel=95, else 0."""
        async with pool.acquire() as conn:
            last_login = await conn.fetchrow(
                """
                SELECT latitude, longitude, created_at
                FROM login_events
                WHERE tenant_id = $1 AND user_id = $2
                  AND latitude IS NOT NULL AND longitude IS NOT NULL
                ORDER BY created_at DESC
                LIMIT 1
                """,
                tenant_id,
                user_id,
            )

            current_loc = await conn.fetchrow(
                """
                SELECT latitude, longitude
                FROM ip_history
                WHERE tenant_id = $1 AND ip_address = $2
                  AND latitude IS NOT NULL AND longitude IS NOT NULL
                ORDER BY created_at DESC
                LIMIT 1
                """,
                tenant_id,
                ip_address,
            )
            if current_loc is None:
                current_loc = await conn.fetchrow(
                    """
                    SELECT latitude, longitude
                    FROM login_events
                    WHERE tenant_id = $1 AND ip_address = $2
                      AND latitude IS NOT NULL AND longitude IS NOT NULL
                    ORDER BY created_at DESC
                    LIMIT 1
                    """,
                    tenant_id,
                    ip_address,
                )

        if last_login is None or current_loc is None:
            return 0

        lat1 = float(last_login["latitude"])
        lon1 = float(last_login["longitude"])
        time1 = last_login["created_at"]
        lat2 = float(current_loc["latitude"])
        lon2 = float(current_loc["longitude"])
        time2 = datetime.utcnow()

        speed = calculate_travel_speed(lat1, lon1, time1, lat2, lon2, time2)
        if is_impossible_travel(speed):
            return 95
        return 0

    def _travel_risk_reason(self, score: int) -> str:
        if score >= 95:
            return "Impossible travel detected"
        return "No impossible travel"

    def _score_to_level_and_action(self, score: int) -> tuple[str, str]:
        """Map risk score to level and action."""
        if score <= 30:
            return "low", "allow"
        if score <= 50:
            return "medium", "allow"
        if score <= 70:
            return "high", "mfa_required"
        return "critical", "block"
