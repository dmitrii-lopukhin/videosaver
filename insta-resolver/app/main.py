from __future__ import annotations

import asyncio
import logging
import sys
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from app import config as config_mod
from app import pool as poolmod
from app import resolver as resolver_mod

log = logging.getLogger("insta-resolver")


def _setup_logging(level: str) -> None:
    logging.basicConfig(
        stream=sys.stdout,
        level=getattr(logging, level.upper(), logging.INFO),
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )


@asynccontextmanager
async def lifespan(app: FastAPI):
    cfg = config_mod.load()
    _setup_logging(cfg.log_level)

    pool = poolmod.SessionPool(
        sessions_dir=cfg.sessions_dir,
        min_interval_sec=cfg.min_request_interval_sec,
        rate_limit_cooldown_sec=cfg.rate_limit_cooldown_sec,
        proxy_url=cfg.insta_proxy_url,
    )
    try:
        count = pool.load()
        log.info("loaded %d session(s) from %s", count, cfg.sessions_dir)
    except Exception as e:  # startup must not crash on a bad/missing session file
        count = 0
        log.error("session load failed: %s", e)
    if count == 0:
        log.warning("no sessions loaded — /resolve returns 503 until sessions are added")

    app.state.pool = pool
    app.state.lock = asyncio.Lock()
    yield


app = FastAPI(title="videosaver insta-resolver", version="0.1.0", lifespan=lifespan)


@app.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok"}


class ResolveRequest(BaseModel):
    url: str
    audio: bool = False
    quality: str = "best"


class ResolveResponse(BaseModel):
    direct_url: str
    title: str
    thumbnail_url: str
    duration_sec: int
    is_audio: bool


@app.post("/resolve", response_model=ResolveResponse)
async def resolve_endpoint(req: ResolveRequest, request: Request):
    pool = request.app.state.pool
    lock = request.app.state.lock

    # instaloader sessions are not thread-safe: serialize all resolve work.
    async with lock:
        try:
            result = await asyncio.to_thread(
                resolver_mod.resolve, pool, req.url, req.audio, req.quality
            )
        except resolver_mod.UnsupportedURL:
            return JSONResponse(status_code=400, content={"detail": "unsupported url"})
        except poolmod.NoSessionsConfigured:
            return JSONResponse(status_code=503, content={"detail": "no sessions loaded"})
        except poolmod.AllSessionsCoolingDown as e:
            return JSONResponse(
                status_code=429,
                content={"detail": "all sessions cooling down"},
                headers={"Retry-After": str(e.retry_after)},
            )
        except resolver_mod.RateLimited as e:
            return JSONResponse(
                status_code=429,
                content={"detail": "rate limited"},
                headers={"Retry-After": str(e.retry_after)},
            )
        except resolver_mod.SessionExpired:
            return JSONResponse(status_code=401, content={"detail": "session expired"})
        except resolver_mod.NotFoundError:
            return JSONResponse(status_code=404, content={"detail": "not found"})
        except resolver_mod.ResolverError:
            return JSONResponse(status_code=500, content={"detail": "resolver error"})

    return ResolveResponse(
        direct_url=result.direct_url,
        title=result.title,
        thumbnail_url=result.thumbnail_url,
        duration_sec=result.duration_sec,
        is_audio=result.is_audio,
    )
