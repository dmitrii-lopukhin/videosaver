import types

import pytest

import app.pool as poolmod
from app.pool import SessionPool, NoSessionsConfigured, AllSessionsCoolingDown


class FakeLoader:
    """Stands in for instaloader.Instaloader — records load_session_from_file."""

    def __init__(self):
        self.loaded = None
        self.context = types.SimpleNamespace(
            _session=types.SimpleNamespace(proxies={})
        )

    def load_session_from_file(self, username, path):
        self.loaded = (username, path)


def _make_session_files(tmp_path, *usernames):
    for u in usernames:
        (tmp_path / f"{u}.session").write_text("x")


def test_load_globs_and_derives_usernames(tmp_path):
    _make_session_files(tmp_path, "userA", "userB")
    pool = SessionPool(str(tmp_path), 0, loader_factory=FakeLoader)
    assert pool.load() == 2
    assert {s.username for s in pool._sessions} == {"userA", "userB"}


def test_load_empty_dir_returns_zero(tmp_path):
    pool = SessionPool(str(tmp_path), 0, loader_factory=FakeLoader)
    assert pool.load() == 0


def test_acquire_empty_raises(tmp_path):
    pool = SessionPool(str(tmp_path), 0, loader_factory=FakeLoader)
    pool.load()
    with pytest.raises(NoSessionsConfigured):
        pool.acquire()


def test_acquire_round_robin(tmp_path):
    _make_session_files(tmp_path, "a", "b")
    pool = SessionPool(str(tmp_path), 0, loader_factory=FakeLoader, now=lambda: 1000.0)
    pool.load()
    first = pool.acquire().username
    second = pool.acquire().username
    third = pool.acquire().username
    assert first != second
    assert third == first


def test_acquire_skips_cooldown(tmp_path):
    _make_session_files(tmp_path, "a", "b")
    pool = SessionPool(str(tmp_path), 0, loader_factory=FakeLoader, now=lambda: 1000.0)
    pool.load()
    cooled = pool._sessions[0]
    pool.mark_cooldown(cooled, 60)
    got = pool.acquire()
    assert got is pool._sessions[1]


def test_all_cooldown_raises_with_retry_after(tmp_path):
    _make_session_files(tmp_path, "a", "b")
    pool = SessionPool(str(tmp_path), 0, loader_factory=FakeLoader, now=lambda: 1000.0)
    pool.load()
    pool.mark_cooldown(pool._sessions[0], 30)
    pool.mark_cooldown(pool._sessions[1], 90)
    with pytest.raises(AllSessionsCoolingDown) as ei:
        pool.acquire()
    assert ei.value.retry_after == 30


def test_min_interval_sleeps(tmp_path):
    _make_session_files(tmp_path, "a")
    times = iter([1000.0, 1000.0, 1000.5])
    slept = []
    pool = SessionPool(
        str(tmp_path),
        2.0,
        loader_factory=FakeLoader,
        now=lambda: next(times),
        sleep=lambda d: slept.append(d),
    )
    pool.load()
    pool._sessions[0].last_used = 999.0
    pool.acquire()
    assert slept and abs(slept[0] - 1.0) < 1e-9


def test_apply_proxy_sets_session_proxies():
    loader = FakeLoader()
    poolmod.apply_proxy(loader, "http://p:8080")
    assert loader.context._session.proxies == {
        "http": "http://p:8080",
        "https": "http://p:8080",
    }


def test_rate_limit_cooldown_sec_stored(tmp_path):
    pool = SessionPool(str(tmp_path), 0, rate_limit_cooldown_sec=123, loader_factory=FakeLoader)
    assert pool.rate_limit_cooldown_sec == 123
