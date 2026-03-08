"""Pydantic models for the Risk Engine API."""

from typing import Any

from pydantic import BaseModel, Field


class RiskEvaluationRequest(BaseModel):
    """Request model for risk evaluation."""

    user_id: str
    tenant_id: str
    ip_address: str
    user_agent: str | None = None
    device_fingerprint: str | None = None
    email: str | None = None


class RiskSignals(BaseModel):
    """Individual risk signal scores (0-100 each)."""

    ip_risk: int = 0
    geo_risk: int = 0
    time_risk: int = 0
    device_risk: int = 0
    travel_risk: int = 0


class RiskEvaluationResponse(BaseModel):
    """Response model for risk evaluation."""

    risk_score: int
    risk_level: str  # low | medium | high | critical
    action: str  # allow | mfa_required | block
    signals: RiskSignals
    details: dict[str, Any] = Field(default_factory=dict)


class AttackSimulationRequest(BaseModel):
    """Request model for attack simulation."""

    attack_type: str
    tenant_id: str
    config: dict[str, Any] = Field(
        default_factory=lambda: {
            "num_attempts": 100,
            "target_accounts": 10,
            "source_ips": 5,
            "duration_seconds": 60,
        }
    )


class AttackSimulationResponse(BaseModel):
    """Response model for attack simulation."""

    simulation_id: str
    attack_type: str
    status: str
    results: dict[str, Any]


class SuspiciousIPResponse(BaseModel):
    """Response model for suspicious IP lookup."""

    ip_address: str
    failed_attempts: int
    unique_accounts_targeted: int
    first_seen: str
    last_seen: str
    country: str | None
    risk_score: int
