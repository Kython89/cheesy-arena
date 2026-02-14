#!/usr/bin/env python3

from fastapi import FastAPI, WebSocket
from fastapi.responses import FileResponse
from fastapi.staticfiles import StaticFiles
import asyncio
import time
import json
from pathlib import Path

app = FastAPI()

# Serve the HTML file at /
@app.get("/")
def serve_audience():
    html_path = Path(__file__).parent / "audience.html"
    return FileResponse(html_path)

# WebSocket for real-time updates
@app.websocket("/ws")
async def websocket_endpoint(ws: WebSocket):
    print("WS connection attempt")
    await ws.accept()
    print("WS connection accepted")

    red_score = 0
    blue_score = 0

    # REBUILT phase definitions
    phases = [
        (20.0, "AUTO", "both"),
        (30.0, "TRANSITION", "both"),
        (55.0, "SHIFT 1", "single"),
        (80.0, "SHIFT 2", "single"),
        (105.0, "SHIFT 3", "single"),
        (130.0, "SHIFT 4", "single"),
        (160.0, "END GAME", "both"),
    ]
    total_game_time = phases[-1][0]

    while True:
        elapsed = time.time() - start_time
        current_phase = phases[-1][1]
        hub_active = "both"
        time_left_in_phase = 0

        # Find current phase
        for i, (phase_end, phase_name, hub) in enumerate(phases):
            if elapsed <= phase_end:
                current_phase = phase_name
                phase_start = 0 if i == 0 else phases[i - 1][0]
                time_left_in_phase = max(0, phase_end - elapsed)
                hub_active = hub
                break

        # Example score increment (replace with real scoring logic)
        red_score += 1
        blue_score += 2

        data = {
            "total_game_time": round(elapsed, 2),
            "current_phase": current_phase,
            "time_left_in_phase": round(time_left_in_phase, 2),
            "hub_active": hub_active,
            "red_score": red_score,
            "blue_score": blue_score
        }

        await ws.send_text(json.dumps(data))
        await asyncio.sleep(0.05)  # ~20 FPS updates
