import app.config as config


def test_load_defaults(monkeypatch):
    for k in [
        "SESSIONS_DIR",
        "MIN_REQUEST_INTERVAL_SEC",
        "RATE_LIMIT_COOLDOWN_SEC",
        "INSTA_PROXY_URL",
        "LOG_LEVEL",
    ]:
        monkeypatch.delenv(k, raising=False)

    cfg = config.load()
    assert cfg.sessions_dir == "/app/sessions"
    assert cfg.min_request_interval_sec == 2.0
    assert cfg.rate_limit_cooldown_sec == 300
    assert cfg.insta_proxy_url == ""
    assert cfg.log_level == "INFO"


def test_load_from_env(monkeypatch):
    monkeypatch.setenv("SESSIONS_DIR", "/tmp/s")
    monkeypatch.setenv("MIN_REQUEST_INTERVAL_SEC", "0.5")
    monkeypatch.setenv("RATE_LIMIT_COOLDOWN_SEC", "120")
    monkeypatch.setenv("INSTA_PROXY_URL", "http://proxy:8080")
    monkeypatch.setenv("LOG_LEVEL", "DEBUG")

    cfg = config.load()
    assert cfg.sessions_dir == "/tmp/s"
    assert cfg.min_request_interval_sec == 0.5
    assert cfg.rate_limit_cooldown_sec == 120
    assert cfg.insta_proxy_url == "http://proxy:8080"
    assert cfg.log_level == "DEBUG"
