from fastapi import FastAPI

app = FastAPI(title="videosaver insta-resolver", version="0.1.0-skeleton")


@app.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok"}
