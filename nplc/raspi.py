import asyncio
import RPi.GPIO as GPIO
from fastapi import FastAPI
import uvicorn

class BallCounter:
    def __init__(self, input_pins: list[int], poll_interval: float = 0.1):
        self.input_pins = input_pins
        self.poll_interval = poll_interval
        self.count = 0
        self.active = False

    def setup_gpio(self) -> None:
        GPIO.setmode(GPIO.BCM)
        for pin in self.input_pins:
            GPIO.setup(pin, GPIO.IN)

    async def _read_sensor(self, pin: int) -> None:
        prev_state = True
        while True:
            current_state = GPIO.input(pin)
            if self.active and current_state and not prev_state:
                self.count += 1
                print("Balls Counted:", self.count)

            prev_state = current_state
            await asyncio.sleep(self.poll_interval)

    def start_tasks(self) -> None:
        for pin in self.input_pins:
            asyncio.create_task(self._read_sensor(pin))

    def reset(self) -> None:
        self.count = 0
        self.active = False

    def set_active(self, active: bool) -> None:
        self.active = active

# ---------------- FastAPI setup ---------------- #

app = FastAPI()
counter = BallCounter(input_pins=[17, 22])

@app.on_event("startup")
async def startup() -> None:
    counter.setup_gpio()
    counter.start_tasks()

@app.get("/status")
def status():
    return {"state": "running" if counter.active else "stopped", "count": counter.count}

@app.post("/reset")
def reset():
    counter.reset()
    return {"ok": True}

@app.post("/start")
def start():
    counter.set_active(True)
    return {"ok": True}

@app.post("/stop")
def stop():
    counter.set_active(False)
    return {"ok": True}

if __name__ == "__main__":
    uvicorn.run(app, host="10.0.0.10", port=8000)
