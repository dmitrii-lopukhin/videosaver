from __future__ import annotations

import re
from dataclasses import dataclass

import instaloader
from instaloader import exceptions as ie

# Matches /p/, /reel/, /reels/, /tv/ optionally preceded by a username segment.
_SHORTCODE_RE = re.compile(
    r"instagram\.com/(?:[^/]+/)?(?:p|reel|reels|tv)/([A-Za-z0-9_-]+)"
)


class UnsupportedURL(Exception):
    pass


class SessionExpired(Exception):
    pass


class NotFoundError(Exception):
    pass


class RateLimited(Exception):
    def __init__(self, retry_after: int) -> None:
        super().__init__(f"rate limited, retry after {retry_after}s")
        self.retry_after = retry_after


class ResolverError(Exception):
    pass


@dataclass
class ResolveResult:
    direct_url: str
    title: str
    thumbnail_url: str
    duration_sec: int
    is_audio: bool


def extract_shortcode(url: str) -> str | None:
    m = _SHORTCODE_RE.search(url)
    return m.group(1) if m else None


def resolve(pool, url: str, audio: bool, quality: str) -> ResolveResult:
    """Resolve an Instagram post/reel URL to a direct video URL.

    `audio` and `quality` are accepted for API compatibility but ignored on v0.1:
    instaloader returns a single video stream and has no audio-only mode. The Go
    bot performs any audio extraction post-download (roadmap v0.2).
    """
    shortcode = extract_shortcode(url)
    if not shortcode:
        raise UnsupportedURL(url)

    # May raise NoSessionsConfigured / AllSessionsCoolingDown — propagated to caller.
    session = pool.acquire()

    try:
        post = instaloader.Post.from_shortcode(session.loader.context, shortcode)
    except ie.TooManyRequestsException as e:
        pool.mark_cooldown(session, pool.rate_limit_cooldown_sec)
        raise RateLimited(retry_after=int(pool.rate_limit_cooldown_sec)) from e
    except (
        ie.LoginRequiredException,
        ie.LoginException,
        ie.BadCredentialsException,
    ) as e:
        raise SessionExpired(str(e)) from e
    except (
        ie.QueryReturnedNotFoundException,
        ie.PrivateProfileNotFollowedException,
        ie.ProfileNotExistsException,
    ) as e:
        raise NotFoundError(str(e)) from e
    except ie.InstaloaderException as e:
        raise ResolverError(str(e)) from e

    if not post.is_video:
        raise NotFoundError("post has no video")

    caption = (getattr(post, "title", None) or post.caption or "").strip()
    return ResolveResult(
        direct_url=post.video_url,
        title=caption[:200],
        thumbnail_url=post.url,
        duration_sec=int(post.video_duration or 0),
        is_audio=False,
    )
