import argparse
import random
import select
import sys
import termios
import time
import tty

from nplc import NPLC

DEACTIVATION_GRACE_SEC = 3.0


def fmt_time(seconds):
    seconds = max(0, int(round(seconds)))
    return f"{seconds // 60}:{seconds % 60:02d}"


def render(status, message):
    return (
        "NPLC Match Dashboard\n"
        "====================\n"
        f"Phase: {status['phase']}\n"
        f"Match: {status['match_time']}\n"
        f"Phase Time: {status['phase_time']}\n"
        f"Shift Active: {status['shift_active']}\n"
        "\n"
        f"Red  Hub: {status['red_count']}  [{status['red_state']}]\n"
        f"Blue Hub: {status['blue_count']}  [{status['blue_state']}]\n"
        "\n"
        "Controls:\n"
        "  g start game   x stop game   r reset counts   q quit\n"
        f"\nLast action: {message}\n"
    )


class Dashboard:
    def __init__(self, red_url, blue_url, rate_hz, timeout_sec):
        self.nplc = NPLC(red_url, blue_url, timeout_sec=timeout_sec)
        self.sleep_s = 1.0 / max(rate_hz, 0.1)
        self.timeline = self._build_timeline()
        self.match_end = self.timeline[-1][0]
        self.running = False
        self.start_time = None
        self.shift1_active = None
        self.last_message = "none"
        self.hub_active = {"red": False, "blue": False}
        self.stop_at = {"red": None, "blue": None}

    def _build_timeline(self):
        return [
            (20.0, "AUTO", "both"),
            (30.0, "TRANSITION", "both"),
            (55.0, "SHIFT 1", "single"),
            (80.0, "SHIFT 2", "single"),
            (105.0, "SHIFT 3", "single"),
            (130.0, "SHIFT 4", "single"),
            (160.0, "END GAME", "both"),
        ]

    def _start_hub(self, hub):
        try:
            self.nplc.start_hub_counting(hub)
            self.hub_active[hub] = True
        except RuntimeError as exc:
            self.last_message = f"error starting {hub}: {exc}"

    def _stop_hub(self, hub):
        try:
            self.nplc.stop_hub_counting(hub)
            self.hub_active[hub] = False
        except RuntimeError as exc:
            self.last_message = f"error stopping {hub}: {exc}"

    def _reset_hubs(self):
        try:
            self.nplc.reset_hub_count("red")
            self.nplc.reset_hub_count("blue")
            self.hub_active = {"red": False, "blue": False}
            self.stop_at = {"red": None, "blue": None}
        except RuntimeError as exc:
            self.last_message = f"error resetting hubs: {exc}"

    def _schedule_stop(self, hub, now):
        if self.hub_active[hub] and self.stop_at[hub] is None:
            self.stop_at[hub] = now + DEACTIVATION_GRACE_SEC

    def _ensure_active(self, hub):
        if self.stop_at[hub] is not None:
            self.stop_at[hub] = None
        if not self.hub_active[hub]:
            self._start_hub(hub)

    def _update_stops(self, now):
        for hub in ("red", "blue"):
            if self.stop_at[hub] is not None and now >= self.stop_at[hub]:
                self.stop_at[hub] = None
                self._stop_hub(hub)

    def _choose_shift1_active(self):
        try:
            red_status = self.nplc.get_hub_status("red")
            blue_status = self.nplc.get_hub_status("blue")
            red_auto = int(red_status.get("count", 0))
            blue_auto = int(blue_status.get("count", 0))
        except RuntimeError as exc:
            self.last_message = f"error reading auto counts: {exc}"
            red_auto = 0
            blue_auto = 0
        if red_auto == blue_auto:
            winner = random.choice(["red", "blue"])
            self.last_message = f"auto tie, random winner {winner}"
        else:
            winner = "red" if red_auto > blue_auto else "blue"
            self.last_message = f"auto winner {winner}"
        return "blue" if winner == "red" else "red"

    def _active_for_shift(self, index):
        if index % 2 == 1:
            return self.shift1_active
        return "blue" if self.shift1_active == "red" else "red"

    def _desired_state(self, elapsed):
        for idx, (end, phase, mode) in enumerate(self.timeline):
            if elapsed < end:
                if mode == "both":
                    return {"red": True, "blue": True}, phase, end - elapsed, "both"
                if self.shift1_active is None:
                    self.shift1_active = self._choose_shift1_active()
                active = self._active_for_shift(idx - 1)
                return (
                    {"red": active == "red", "blue": active == "blue"},
                    phase,
                    end - elapsed,
                    active,
                )
        return {"red": False, "blue": False}, "POST", 0, "none"

    def _poll_status(self, hub):
        try:
            return self.nplc.get_hub_status(hub)
        except RuntimeError as exc:
            return {"count": "ERR", "state": str(exc)}

    def _handle_key(self, key):
        if key == "g":
            self._reset_hubs()
            self._start_hub("red")
            self._start_hub("blue")
            self.running = True
            self.start_time = time.time()
            self.shift1_active = None
            self.last_message = "game started"
        elif key == "x":
            self.running = False
            now = time.time()
            self._schedule_stop("red", now)
            self._schedule_stop("blue", now)
            self.last_message = "game stopped"
        elif key == "r":
            self._reset_hubs()
            self.running = False
            self.last_message = "counts reset"

    def _tick(self):
        now = time.time()
        if self.running:
            elapsed = now - self.start_time
            desired, phase, phase_left, shift_active = self._desired_state(elapsed)
            for hub in ("red", "blue"):
                if desired[hub]:
                    self._ensure_active(hub)
                else:
                    self._schedule_stop(hub, now)
            self._update_stops(now)
            if elapsed >= self.match_end + DEACTIVATION_GRACE_SEC:
                self.running = False
                self.last_message = "match complete"
        else:
            self._update_stops(now)
            phase = "IDLE"
            phase_left = 0
            shift_active = "none"
            elapsed = 0

        red_status = self._poll_status("red")
        blue_status = self._poll_status("blue")
        match_left = max(0, self.match_end - elapsed)
        status = {
            "phase": phase,
            "match_time": fmt_time(match_left),
            "phase_time": fmt_time(phase_left),
            "shift_active": shift_active,
            "red_count": red_status.get("count", "?"),
            "red_state": red_status.get("state", "?"),
            "blue_count": blue_status.get("count", "?"),
            "blue_state": blue_status.get("state", "?"),
        }
        sys.stdout.write("\x1b[2J\x1b[H")
        sys.stdout.write(render(status, self.last_message))
        sys.stdout.flush()
        time.sleep(self.sleep_s)

    def run(self):
        try:
            stdin_fd = sys.stdin.fileno()
            original_settings = termios.tcgetattr(stdin_fd)
            tty.setcbreak(stdin_fd)
            try:
                while True:
                    if select.select([sys.stdin], [], [], 0)[0]:
                        key = sys.stdin.read(1)
                        if key == "q":
                            break
                        self._handle_key(key)
                    self._tick()
            finally:
                termios.tcsetattr(stdin_fd, termios.TCSADRAIN, original_settings)
        except KeyboardInterrupt:
            sys.stdout.write("\n")


def main():
    parser = argparse.ArgumentParser(description="Simple match dashboard for NPLC fuel counters.")
    parser.add_argument("--red", required=True, help="Red hub base URL, e.g. http://10.0.0.10:8000")
    parser.add_argument("--blue", required=True, help="Blue hub base URL, e.g. http://10.0.0.11:8000")
    parser.add_argument("--rate", type=float, default=5.0, help="Polling rate in Hz (default: 5)")
    parser.add_argument("--timeout", type=float, default=2.0, help="HTTP timeout in seconds (default: 2)")
    args = parser.parse_args()

    Dashboard(args.red, args.blue, args.rate, args.timeout).run()


if __name__ == "__main__":
    main()
