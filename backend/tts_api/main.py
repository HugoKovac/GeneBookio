"""Local TTS API for the book pipeline: Kokoro for English, XTTS-v2 for French.

Replaces the LocalAI container previously used by internal/tts
(see backend/internal/tts and backend/internal/platform/ttsapi on the Go
side). Contract: POST /tts {"input": str, "language": "en"|"fr"} -> WAV bytes.
"""

from __future__ import annotations

import logging
import os
import tempfile
from contextlib import asynccontextmanager
from pathlib import Path
from typing import Literal

import numpy as np
import soundfile as sf
from fastapi import FastAPI, HTTPException
from fastapi.responses import Response
from pydantic import BaseModel

XTTS_SPEAKER_WAV = os.environ.get("XTTS_SPEAKER_WAV", "/app/speaker_ref.wav")
KOKORO_VOICE = os.environ.get("KOKORO_VOICE", "af_heart")

# XTTS's GPT decoder occasionally runs away instead of emitting a stop token
# (a known upstream issue — see coqui-ai/TTS#3407 and similar), overflowing
# its fixed position-embedding table with "IndexError: index out of range in
# self". It's stochastic (do_sample=True), so a retry with fresh sampling
# usually succeeds; XTTS_MAX_ATTEMPTS caps how many times we try.
XTTS_MAX_ATTEMPTS = int(os.environ.get("XTTS_MAX_ATTEMPTS", "3"))

logger = logging.getLogger("tts_api")

engines: dict[str, object] = {}


@asynccontextmanager
async def lifespan(app: FastAPI):
    from kokoro import KPipeline

    engines["kokoro"] = KPipeline(lang_code="a")

    if Path(XTTS_SPEAKER_WAV).is_file():
        import torch
        from TTS.api import TTS

        device = "cuda" if torch.cuda.is_available() else "cpu"
        engines["xtts"] = TTS("tts_models/multilingual/multi-dataset/xtts_v2").to(device)
    else:
        print(f"[tts_api] WARNING: no XTTS speaker reference at {XTTS_SPEAKER_WAV} — "
              f"XTTS is disabled, French requests will fail until it's mounted.")

    yield
    engines.clear()


app = FastAPI(lifespan=lifespan)


class TTSRequest(BaseModel):
    input: str
    language: Literal["en", "fr"]


@app.get("/health")
def health():
    return {"status": "ok", "engines": list(engines)}


def _synthesize_kokoro(text: str, out_path: Path) -> None:
    pipeline = engines["kokoro"]
    chunks = [audio for _, _, audio in pipeline(text, voice=KOKORO_VOICE)]
    audio = np.concatenate(chunks)
    sf.write(out_path, audio, 24000)


def _synthesize_xtts(text: str, out_path: Path) -> None:
    last_error: Exception | None = None
    for attempt in range(1, XTTS_MAX_ATTEMPTS + 1):
        try:
            engines["xtts"].tts_to_file(
                text=text,
                speaker_wav=XTTS_SPEAKER_WAV,
                language="fr",
                file_path=str(out_path),
            )
            return
        except Exception as e:
            last_error = e
            logger.warning("XTTS synthesis attempt %d/%d failed: %s", attempt, XTTS_MAX_ATTEMPTS, e)

    raise HTTPException(
        status_code=502,
        detail=f"XTTS synthesis failed after {XTTS_MAX_ATTEMPTS} attempts: "
               f"{type(last_error).__name__}: {last_error}",
    )


@app.post("/tts")
def synthesize(req: TTSRequest):
    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        out_path = Path(tmp.name)

    try:
        if req.language == "en":
            try:
                _synthesize_kokoro(req.input, out_path)
            except Exception as e:
                raise HTTPException(
                    status_code=502,
                    detail=f"Kokoro synthesis failed: {type(e).__name__}: {e}",
                ) from e
        else:
            if "xtts" not in engines:
                raise HTTPException(
                    status_code=503,
                    detail=f"XTTS not loaded — mount a speaker reference wav at {XTTS_SPEAKER_WAV}",
                )
            _synthesize_xtts(req.input, out_path)

        return Response(content=out_path.read_bytes(), media_type="audio/wav")
    finally:
        out_path.unlink(missing_ok=True)
