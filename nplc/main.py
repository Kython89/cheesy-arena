from fastapi import FastAPI, WebSocket
from fastapi.responses import FileResponse
from fastapi.staticfiles import StaticFiles
import asyncio
import time
import json

app = FastAPI()

# Serve HTML
@app.get("/")
def serve_audience():
    return FileResponse("audience.html")

# WebSocket for real-time updates
@app.websocket("/ws")
async def websocket_endpoint(ws: WebSocket):
    await ws.accept()
    start_time = time.time()
    # Example scores
    red_score = 0
    blue_score = 0

    while True:
        elapsed = time.time() - start_time

        # Define your phases based on REBUILT manual
        phases = [
            (20.0, "AUTO", "both"),
            (30.0, "TRANSITION", "both"),
            (55.0, "SHIFT 1", "single"),
            (80.0, "SHIFT 2", "single"),
            (105.0, "SHIFT 3", "single"),
            (130.0, "SHIFT 4", "single"),
            (160.0, "END GAME", "both"),
        ]

        # Determine current phase and time left
        total_game_time = phases[-1][0]
        time_left = max(0, total_game_time - elapsed)
        current_phase = phases[-1][1]
        hub_active = "both"

        for i, (phase_end, phase_name, hub) in enumerate(phases):
            if elapsed <= phase_end:
                current_phase = phase_name
                if i == 0:
                    phase_start = 0
                else:
                    phase_start = phases[i - 1][0]
                time_left_in_phase = max(0, phase_end - elapsed)
                hub_active = hub
                break

        # Example: increment scores for demo
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

        # Update every 50 milliseconds (~20 FPS)
        await asyncio.sleep(0.05)
