from fastapi.testclient import TestClient

import app.main as main
from app import pool as poolmod
from app import resolver as resolver_mod
from app.resolver import ResolveResult


def _client():
    return TestClient(main.app)


def test_health():
    with _client() as c:
        r = c.get("/health")
        assert r.status_code == 200
        assert r.json() == {"status": "ok"}


def test_resolve_success(monkeypatch):
    def fake(pool, url, audio, quality):
        return ResolveResult("https://cdn/v.mp4", "title", "https://cdn/t.jpg", 12, False)

    monkeypatch.setattr(main.resolver_mod, "resolve", fake)
    with _client() as c:
        r = c.post("/resolve", json={"url": "https://instagram.com/p/ABC/"})
        assert r.status_code == 200
        body = r.json()
        assert body["direct_url"] == "https://cdn/v.mp4"
        assert body["title"] == "title"
        assert body["thumbnail_url"] == "https://cdn/t.jpg"
        assert body["duration_sec"] == 12
        assert body["is_audio"] is False


def test_resolve_unsupported_url_400(monkeypatch):
    def fake(pool, url, audio, quality):
        raise resolver_mod.UnsupportedURL(url)

    monkeypatch.setattr(main.resolver_mod, "resolve", fake)
    with _client() as c:
        r = c.post("/resolve", json={"url": "https://x/y"})
        assert r.status_code == 400


def test_resolve_no_sessions_503(monkeypatch):
    def fake(pool, url, audio, quality):
        raise poolmod.NoSessionsConfigured()

    monkeypatch.setattr(main.resolver_mod, "resolve", fake)
    with _client() as c:
        r = c.post("/resolve", json={"url": "https://instagram.com/p/ABC/"})
        assert r.status_code == 503


def test_resolve_all_cooldown_429_retry_after(monkeypatch):
    def fake(pool, url, audio, quality):
        raise poolmod.AllSessionsCoolingDown(retry_after=42)

    monkeypatch.setattr(main.resolver_mod, "resolve", fake)
    with _client() as c:
        r = c.post("/resolve", json={"url": "https://instagram.com/p/ABC/"})
        assert r.status_code == 429
        assert r.headers["Retry-After"] == "42"


def test_resolve_rate_limited_429(monkeypatch):
    def fake(pool, url, audio, quality):
        raise resolver_mod.RateLimited(retry_after=300)

    monkeypatch.setattr(main.resolver_mod, "resolve", fake)
    with _client() as c:
        r = c.post("/resolve", json={"url": "https://instagram.com/p/ABC/"})
        assert r.status_code == 429
        assert r.headers["Retry-After"] == "300"


def test_resolve_session_expired_401(monkeypatch):
    def fake(pool, url, audio, quality):
        raise resolver_mod.SessionExpired()

    monkeypatch.setattr(main.resolver_mod, "resolve", fake)
    with _client() as c:
        r = c.post("/resolve", json={"url": "https://instagram.com/p/ABC/"})
        assert r.status_code == 401


def test_resolve_not_found_404(monkeypatch):
    def fake(pool, url, audio, quality):
        raise resolver_mod.NotFoundError()

    monkeypatch.setattr(main.resolver_mod, "resolve", fake)
    with _client() as c:
        r = c.post("/resolve", json={"url": "https://instagram.com/p/ABC/"})
        assert r.status_code == 404


def test_resolve_resolver_error_500(monkeypatch):
    def fake(pool, url, audio, quality):
        raise resolver_mod.ResolverError()

    monkeypatch.setattr(main.resolver_mod, "resolve", fake)
    with _client() as c:
        r = c.post("/resolve", json={"url": "https://instagram.com/p/ABC/"})
        assert r.status_code == 500
