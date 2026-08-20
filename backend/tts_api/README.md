# tts_api

Local TTS API used by `internal/tts` (via `internal/platform/ttsapi`) instead of
the old LocalAI container. Kokoro for English, XTTS-v2 (voice cloning) for French.

`POST /tts {"input": "...", "language": "en"|"fr"}` -> WAV bytes. `GET /health`.

## Required before French (XTTS) works

Drop a ~10s clean voice recording (no background noise, one speaker) at
`speaker_ref.wav` in this directory. `docker-compose.yaml` mounts it read-only
into the container at `/app/speaker_ref.wav`. Without it, `/health` still
reports OK but XTTS is disabled and `POST /tts` with `"language":"fr"` returns
503 — `docker compose up` will still start, but check the `tts_api` container
logs for the "no XTTS speaker reference" warning if French requests fail.

## Run

```bash
docker compose up -d --build tts_api
```

Model weights are cached on the host so they only download once: Kokoro under
`../models` (mounted at `/root/.cache`, HuggingFace hub's cache convention),
and XTTS-v2 (~1.8GB) under `../models-tts` (mounted at `/root/.local/share/tts`,
where Coqui TTS's own `ModelManager` downloads to — it does not honor
`/root/.cache`).
