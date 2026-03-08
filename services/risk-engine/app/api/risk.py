"""Risk evaluation and attack simulation API endpoints."""

from fastapi import APIRouter, Depends, Query

from app.api.deps import DbPoolDep, RedisDep
from app.engine.attack_simulator import AttackSimulator
from app.engine.risk_scorer import RiskScorer
from app.models.schemas import (
    AttackSimulationRequest,
    AttackSimulationResponse,
    RiskEvaluationRequest,
    RiskEvaluationResponse,
)

router = APIRouter(prefix="/api/v1", tags=["risk"])


@router.get("/risk/evaluate", response_model=RiskEvaluationResponse)
async def evaluate_risk(
    user_id: str = Query(..., description="User ID"),
    tenant_id: str = Query(..., description="Tenant ID"),
    ip_address: str = Query(..., description="IP address of the login attempt"),
    user_agent: str | None = Query(None, description="User agent string"),
    device_fingerprint: str | None = Query(None, description="Device fingerprint"),
    email: str | None = Query(None, description="User email"),
    db: DbPoolDep,
    redis: RedisDep,
) -> RiskEvaluationResponse:
    """
    Evaluate risk for a login attempt.

    Returns risk score, level, recommended action, and signal breakdown.
    """
    request = RiskEvaluationRequest(
        user_id=user_id,
        tenant_id=tenant_id,
        ip_address=ip_address,
        user_agent=user_agent,
        device_fingerprint=device_fingerprint,
        email=email,
    )
    scorer = RiskScorer()
    return await scorer.evaluate(request, db, redis)


@router.post("/simulate/attack", response_model=AttackSimulationResponse)
async def simulate_attack(
    body: AttackSimulationRequest,
    db: DbPoolDep,
) -> AttackSimulationResponse:
    """
    Simulate an attack type and measure detection capability.

    Does not persist any data - generates synthetic events and measures
    how the risk engine would respond.
    """
    simulator = AttackSimulator()
    return await simulator.simulate(body, db)


@router.get("/risk/health")
async def risk_health() -> dict[str, str]:
    """Health check for the risk API."""
    return {"status": "ok"}
