from __future__ import annotations

import glob
import math
import os
import time
from dataclasses import dataclass
from typing import Callable, List

import instaloader


class NoSessionsConfigured(Exception):
    """The pool has zero loaded sessions."""


class AllSessionsCoolingDown(Exception):
    """Every session is currently on cooldown."""

    def __init__(self, retry_after: int) -> None:
        super().__init__(f"all sessions cooling down, retry after {retry_after}s")
        self.retry_after = retry_after


@dataclass
class Session:
    username: str
    loader: "instaloader.Instaloader"
    cooldown_until: float = 0.0
    last_used: float = 0.0


def _default_loader_factory() -> "instaloader.Instaloader":
    return instaloader.Instaloader(
        quiet=True,
        download_pictures=False,
        download_videos=False,
        download_video_thumbnails=False,
        download_geotags=False,
        download_comments=False,
        save_metadata=False,
        compress_json=False,
    )


def apply_proxy(loader, proxy_url: str) -> None:
    """Route the loader's underlying requests session through an HTTP proxy."""
    if not proxy_url:
        return
    loader.context._session.proxies.update(
        {"http": proxy_url, "https": proxy_url}
    )


class SessionPool:
    def __init__(
        self,
        sessions_dir: str,
        min_interval_sec: float,
        rate_limit_cooldown_sec: int = 300,
        proxy_url: str = "",
        loader_factory: Callable[[], "instaloader.Instaloader"] = _default_loader_factory,
        now: Callable[[], float] = time.time,
        sleep: Callable[[float], None] = time.sleep,
    ) -> None:
        self.sessions_dir = sessions_dir
        self.min_interval_sec = min_interval_sec
        self.rate_limit_cooldown_sec = rate_limit_cooldown_sec
        self.proxy_url = proxy_url
        self._loader_factory = loader_factory
        self._now = now
        self._sleep = sleep
        self._sessions: List[Session] = []
        self._rr = 0

    def load(self) -> int:
        self._sessions = []
        pattern = os.path.join(self.sessions_dir, "*.session")
        for path in sorted(glob.glob(pattern)):
            username = os.path.splitext(os.path.basename(path))[0]
            loader = self._loader_factory()
            loader.load_session_from_file(username, path)
            apply_proxy(loader, self.proxy_url)
            self._sessions.append(Session(username=username, loader=loader))
        return len(self._sessions)

    def count(self) -> int:
        return len(self._sessions)

    def acquire(self) -> Session:
        n = len(self._sessions)
        if n == 0:
            raise NoSessionsConfigured()

        now = self._now()
        for i in range(n):
            idx = (self._rr + i) % n
            s = self._sessions[idx]
            if s.cooldown_until <= now:
                self._rr = (idx + 1) % n
                self._enforce_interval(s)
                return s

        soonest = min(s.cooldown_until for s in self._sessions)
        retry_after = max(1, math.ceil(soonest - now))
        raise AllSessionsCoolingDown(retry_after)

    def _enforce_interval(self, s: Session) -> None:
        wait = self.min_interval_sec - (self._now() - s.last_used)
        if wait > 0:
            self._sleep(wait)
        s.last_used = self._now()

    def mark_cooldown(self, s: Session, seconds: float) -> None:
        s.cooldown_until = self._now() + seconds
