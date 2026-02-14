from fastapi import FastAPI, WebSocket
from fastapi.staticfiles import StaticFiles
from fastapi.responses import FileResponse
import asyncio
import time

app = FastAPI()

# Serve static frontend
app.mount("/static", StaticFiles(directory="static"), name="static")


@app.get("/")
def index():
    return FileResponse("static/audience.html")


# -------------------------
# REBUILT PHASE DEFINITION
# -------------------------

PHASES = [
    (20.0, "AUTO", "both"),
    (30.0, "TRANSITION", "both"),
    (55.0, "SHIFT 1", "single"),
    (80.0, "SHIFT 2", "single"),
    (105.0, "SHIFT 3", "single"),
    (130.0, "SHIFT 4", "single"),
    (160.0, "END GAME", "both"),
]

TOTAL_MATCH_TIME = 160.0


def get_phase_info(overall_time):
    previous_end = 0.0

    for end_time, name, mode in PHASES:
        if overall_time <= end_time:
            phase_time_left = end_time - overall_time
            return name, mode, phase_time_left
        previous_end = end_time

    return "MATCH OVER", "none", 0.0


# -------------------------
# WEBSOCKET
# -------------------------

@app.websocket("/ws/audience")
async def websocket_endpoint(websocket: WebSocket):
    await websocket.accept()

    start_time = time.time()

    red_score = 0
    blue_score = 0

    while True:
        overall = time.time() - start_time

        if overall > TOTAL_MATCH_TIME:
            overall = TOTAL_MATCH_TIME

        phase_name, hub_mode, phase_time_left = get_phase_info(overall)

        # Hub Logic
        if hub_mode == "both":
            active_hub = "BOTH"
        elif hub_mode == "single":
            # Shifts begin after 30 seconds
            shift_index = int((overall - 30) // 25)
            active_hub = "RED" if shift_index % 2 == 0 else "BLUE"
        else:
            active_hub = "NONE"

        await websocket.send_json({
            "overall_time": overall,
            "phase_time_left": phase_time_left,
            "phase_name": phase_name,
            "active_hub": active_hub,
            "red_score": red_score,
            "blue_score": blue_score
        })

        await asyncio.sleep(0.1)
