# NPLC (Network PLC) Fuel Counter

This folder contains a lightweight, networked fuel counter system for two hubs (red and blue). The FMS-side tools connect to two Raspberry Pis over HTTP and start/stop counting based on match timing rules.

**Key files**
- `nplc/nplc.py` HTTP client for hub control and status.
- `nplc/dashboard.py` Match-timed dashboard that starts/stops counting based on the game rules.
- `nplc/display.py` Manual control display for starting/stopping/resetting hubs.
- `nplc/dummy_pi.py` Local dummy server to simulate a hub.
- `nplc/raspi.py` Real Pi server (FastAPI + GPIO).
- `nplc/docs/fuel-counter-http-contract.md` HTTP API contract.
- `nplc/docs/game-counting-requirements.md` Timing rules for counting.

**Quick start with dummy hubs (local dev)**
1. Start two dummy Pi servers in separate terminals:
```bash
python nplc/dummy_pi.py --port 8000 --rate 3
python nplc/dummy_pi.py --port 8001 --rate 2
```
2. Start the match dashboard in a third terminal:
```bash
python nplc/dashboard.py --red http://127.0.0.1:8000 --blue http://127.0.0.1:8001
```
3. Press `g` to start the game, `x` to stop, `r` to reset, `q` to quit.

**Manual control display (optional)**
Use this if you want manual start/stop/reset controls instead of match timing:
```bash
python nplc/display.py --red http://127.0.0.1:8000 --blue http://127.0.0.1:8001
```

**Running on real Raspberry Pis**
1. Install dependencies on each Pi:
```bash
python -m pip install fastapi uvicorn
```
2. Ensure `RPi.GPIO` is installed (usually preinstalled on Raspberry Pi OS).
3. Update pins and host in `nplc/raspi.py` if needed.
4. Run on each Pi (red and blue):
```bash
python nplc/raspi.py
```
5. Point the dashboard or display at each Pi's IP address.

**Dashboard behavior (ball counting only)**
- AUTO and TRANSITION: both hubs active.
- SHIFT 1–4: only one hub active, alternating each shift.
- END GAME: both hubs active.
- Deactivation grace: 3 seconds after a hub becomes inactive.
- Match end grace: 3 seconds after match end.

**Notes**
- The dashboard reads AUTO counts to decide SHIFT 1 active hub. Ties are broken randomly.
- The HTTP contract is defined in `nplc/docs/fuel-counter-http-contract.md`.
