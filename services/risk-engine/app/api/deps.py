"""Dependency injection for FastAPI."""

from collections.abc import AsyncGenerator
from typing import Annotated

import asyncpg
from fastapi import Depends, Request
from redis.asyncio import Redis


async def get_db(
    request: Request,
) -> AsyncGenerator[asyncpg.Pool, None]:
    """Yield the asyncpg connection pool from app state."""
    pool: asyncpg.Pool = request.app.state.db_pool
    yield pool


async def get_redis(
    request: Request,
) -> AsyncGenerator[Redis, None]:
    """Yield the Redis client from app state."""
    redis: Redis = request.app.state.redis_client
    yield redis


# Type aliases for cleaner dependency injection
DbPoolDep = Annotated[asyncpg.Pool, Depends(get_db)]
RedisDep = Annotated[Redis, Depends(get_redis)]
