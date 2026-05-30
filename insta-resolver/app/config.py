import os
from dataclasses import dataclass


@dataclass
class Config:
    sessions_dir: str
    min_request_interval_sec: float
    rate_limit_cooldown_sec: int
    insta_proxy_url: str
    log_level: str


def _get(key: str, default: str) -> str:
    v = os.getenv(key)
    return v if v not in (None, "") else default


def load() -> Config:
    return Config(
        sessions_dir=_get("SESSIONS_DIR", "/app/sessions"),
        min_request_interval_sec=float(_get("MIN_REQUEST_INTERVAL_SEC", "2")),
        rate_limit_cooldown_sec=int(_get("RATE_LIMIT_COOLDOWN_SEC", "300")),
        insta_proxy_url=_get("INSTA_PROXY_URL", ""),
        log_level=_get("LOG_LEVEL", "INFO"),
    )
