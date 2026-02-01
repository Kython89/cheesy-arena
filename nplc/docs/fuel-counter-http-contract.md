# Fuel Counter Pi HTTP Contract (v1)

This document defines a minimal HTTP API for two Raspberry Pis, one per hub.
Each Pi runs the same API and is addressed directly over Ethernet from the FMS
computer.

## Network assumptions
- Two Pis: one for the Red hub and one for the Blue hub.
- FMS computer connects to both Pis over Ethernet.
- Each Pi has a fixed IP (static IP or DHCP reservation).
- The display runs on the FMS computer and polls the Pis directly.

## Base URL (examples)
- Red hub Pi: `http://10.0.0.10:8000`
- Blue hub Pi: `http://10.0.0.11:8000`

## Endpoints

### POST /start
Start counting fuel.

Response:
```json
{ "ok": true }
```

### POST /stop
Stop counting fuel (no reset).

Response:
```json
{ "ok": true }
```

### POST /reset
Reset the count to 0 and stop counting.

Response:
```json
{ "ok": true }
```

### GET /status
Return the current state and count.

Response:
```json
{ "state": "running", "count": 123 }

```

## Notes
- All responses are JSON and use HTTP status 200 on success.
- If an error occurs, return a non-200 with `{ "ok": false, "error": "..." }`.
- The FMS-side display should poll `/status` for each Pi (e.g., 5–10 Hz).
