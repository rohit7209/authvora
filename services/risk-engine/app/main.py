"""Authvora Risk Engine - FastAPI application."""

import logging
import sys

import asyncpg
from fastapi import FastAPI
from redis.asyncio import Redis

from app.api.risk import router as risk_router
from app.models.config import Settings

# Load settings from environment
settings = Settings()

# Configure logging
logging.basicConfig(
    level=getattr(logging, settings.log_level.upper(), logging.INFO),
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    stream=sys.stdout,
)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="Authvora Risk Engine",
    description="Risk evaluation and attack simulation for the Authvora authentication platform",
    version="1.0.0",
)

app.include_router(risk_router)


@app.get("/health")
async def health_check() -> dict[str, str]:
    """Health check endpoint."""
    return {"status": "ok", "service": "risk-engine"}


@app.on_event("startup")
async def startup() -> None:
    """Connect to PostgreSQL and Redis on startup."""
    logger.info("Starting Risk Engine...")
    try:
        app.state.db_pool = await asyncpg.create_pool(
            settings.database_url,
            min_size=2,
            max_size=10,
            command_timeout=60,
        )
        logger.info("Connected to PostgreSQL")
    except Exception as e:
        logger.error("Failed to connect to PostgreSQL: %s", e)
        raise

    try:
        app.state.redis_client = Redis.from_url(
            settings.redis_url,
            encoding="utf-8",
            decode_responses=True,
        )
        await app.state.redis_client.ping()
        logger.info("Connected to Redis")
    except Exception as e:
        logger.error("Failed to connect to Redis: %s", e)
        raise


@app.on_event("shutdown")
async def shutdown() -> None:
    """Close database and Redis connections on shutdown."""
    logger.info("Shutting down Risk Engine...")
    if hasattr(app.state, "db_pool") and app.state.db_pool:
        await app.state.db_pool.close()
        logger.info("Closed PostgreSQL connection pool")
    if hasattr(app.state, "redis_client") and app.state.redis_client:
        await app.state.redis_client.close()
        logger.info("Closed Redis connection")
