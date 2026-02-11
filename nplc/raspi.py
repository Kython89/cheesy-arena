import asyncio
import RPi.GPIO as GPIO
from fastapi import FastAPI
import uvicorn

class BallCounter:
    def __init__(self, led_pin: int, input_pins: list[int], poll_interval: float = 0.1):
        self.input_pins = input_pins
        self.poll_interval = poll_interval
        self.count = 0
        self.active = False
        self.led_pin = led_pin

    def setup_gpio(self) -> None:
        GPIO.setmode(GPIO.BCM)
        for pin in self.input_pins:
            GPIO.setup(pin, GPIO.IN)
        GPIO.setup(self.led_pin, GPIO.OUT)

    async def _read_sensor(self, pin: int) -> None:
        prev_state = False
        while True:
            current_state = GPIO.input(pin)
            if self.active and not current_state and prev_state:
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

    def set_LED_status(self, active: bool) -> None:
        GPIO.output(self.led_pin, GPIO.HIGH if active else GPIO.LOW)

# ---------------- FastAPI setup ---------------- #

app = FastAPI()
counter = BallCounter(led_pin=-1, input_pins=[17, 22]) # TODO get the led_pin correct

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

@app.post("/lights_on")
def lights_on():
    counter.set_LED_status(True)
    return {"ok": True}

@app.post("/lights_off")
def lights_off():
    counter.set_LED_status(False)
    return {"ok": True}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
