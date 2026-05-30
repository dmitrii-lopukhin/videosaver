import types

import instaloader
import pytest
from instaloader import exceptions as ie

import app.resolver as resolver
from app.resolver import (
    ResolveResult,
    NotFoundError,
    RateLimited,
    ResolverError,
    SessionExpired,
    UnsupportedURL,
    extract_shortcode,
    resolve,
)


class FakePost:
    def __init__(
        self,
        is_video=True,
        video_url="https://cdn/v.mp4",
        title="t",
        caption="c",
        url="https://cdn/thumb.jpg",
        video_duration=12.7,
    ):
        self.is_video = is_video
        self.video_url = video_url
        self.title = title
        self.caption = caption
        self.url = url
        self.video_duration = video_duration


class FakeSession:
    def __init__(self):
        self.loader = types.SimpleNamespace(context=object())
        self.cooldown_until = 0.0


class FakePool:
    def __init__(self):
        self.rate_limit_cooldown_sec = 300
        self.cooled = None
        self._s = FakeSession()

    def acquire(self):
        return self._s

    def mark_cooldown(self, s, seconds):
        self.cooled = (s, seconds)


@pytest.mark.parametrize(
    "url,expected",
    [
        ("https://www.instagram.com/p/ABC123/", "ABC123"),
        ("https://instagram.com/reel/Xy_-9/", "Xy_-9"),
        ("https://www.instagram.com/reels/ZZZ/", "ZZZ"),
        ("https://www.instagram.com/tv/TTT/", "TTT"),
        ("https://www.instagram.com/someuser/reel/RRR/", "RRR"),
        ("https://instagram.com/p/ABC123/?igshid=xyz", "ABC123"),
    ],
)
def test_extract_shortcode_ok(url, expected):
    assert extract_shortcode(url) == expected


def test_extract_shortcode_none():
    assert extract_shortcode("https://example.com/p/abc") is None


def test_resolve_unsupported_url():
    with pytest.raises(UnsupportedURL):
        resolve(FakePool(), "https://example.com/x", False, "best")


def test_resolve_success(monkeypatch):
    monkeypatch.setattr(
        instaloader.Post, "from_shortcode", staticmethod(lambda ctx, sc: FakePost())
    )
    r = resolve(FakePool(), "https://instagram.com/p/ABC/", False, "best")
    assert isinstance(r, ResolveResult)
    assert r.direct_url == "https://cdn/v.mp4"
    assert r.thumbnail_url == "https://cdn/thumb.jpg"
    assert r.duration_sec == 12
    assert r.is_audio is False
    assert r.title == "t"


def test_resolve_not_video_is_404(monkeypatch):
    monkeypatch.setattr(
        instaloader.Post,
        "from_shortcode",
        staticmethod(lambda ctx, sc: FakePost(is_video=False)),
    )
    with pytest.raises(NotFoundError):
        resolve(FakePool(), "https://instagram.com/p/ABC/", False, "best")


def test_resolve_too_many_requests_marks_cooldown(monkeypatch):
    def boom(ctx, sc):
        raise ie.TooManyRequestsException("429")

    monkeypatch.setattr(instaloader.Post, "from_shortcode", staticmethod(boom))
    pool = FakePool()
    with pytest.raises(RateLimited) as ei:
        resolve(pool, "https://instagram.com/p/ABC/", False, "best")
    assert ei.value.retry_after == 300
    assert pool.cooled is not None
    assert pool.cooled[1] == 300


def test_resolve_login_required_is_session_expired(monkeypatch):
    def boom(ctx, sc):
        raise ie.LoginRequiredException("login required")

    monkeypatch.setattr(instaloader.Post, "from_shortcode", staticmethod(boom))
    with pytest.raises(SessionExpired):
        resolve(FakePool(), "https://instagram.com/p/ABC/", False, "best")


def test_resolve_not_found(monkeypatch):
    def boom(ctx, sc):
        raise ie.QueryReturnedNotFoundException("404")

    monkeypatch.setattr(instaloader.Post, "from_shortcode", staticmethod(boom))
    with pytest.raises(NotFoundError):
        resolve(FakePool(), "https://instagram.com/p/ABC/", False, "best")


def test_resolve_generic_is_resolver_error(monkeypatch):
    def boom(ctx, sc):
        raise ie.ConnectionException("conn")

    monkeypatch.setattr(instaloader.Post, "from_shortcode", staticmethod(boom))
    with pytest.raises(ResolverError):
        resolve(FakePool(), "https://instagram.com/p/ABC/", False, "best")
