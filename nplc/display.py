import argparse
import select
import sys
import termios
import time
import tty

from nplc import NPLC


def render(red_status, blue_status, message):
    red_count = red_status.get("count", "?")
    red_state = red_status.get("state", "?")
    blue_count = blue_status.get("count", "?")
    blue_state = blue_status.get("state", "?")
    return (
        "NPLC Fuel Counter Display\n"
        "==========================\n"
        f"Red  Hub: {red_count}  [{red_state}]\n"
        f"Blue Hub: {blue_count}  [{blue_state}]\n"
        "\n"
        "Controls:\n"
        "  1 start red   2 stop red   3 reset red\n"
        "  4 start blue  5 stop blue  6 reset blue\n"
        "  a start both  s stop both  d reset both\n"
        "  q quit\n"
        f"\nLast action: {message}\n"
    )


def main():
    parser = argparse.ArgumentParser(description="Simple NPLC fuel counter display.")
    parser.add_argument("--red", required=True, help="Red hub base URL, e.g. http://10.0.0.10:8000")
    parser.add_argument("--blue", required=True, help="Blue hub base URL, e.g. http://10.0.0.11:8000")
    parser.add_argument("--rate", type=float, default=5.0, help="Polling rate in Hz (default: 5)")
    parser.add_argument("--timeout", type=float, default=2.0, help="HTTP timeout in seconds (default: 2)")
    args = parser.parse_args()

    nplc = NPLC(args.red, args.blue, timeout_sec=args.timeout)
    sleep_s = 1.0 / max(args.rate, 0.1)
    last_message = "none"

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
                    if key == "1":
                        nplc.start_hub_counting("red")
                        last_message = "start red"
                    elif key == "2":
                        nplc.stop_hub_counting("red")
                        last_message = "stop red"
                    elif key == "3":
                        nplc.reset_hub_count("red")
                        last_message = "reset red"
                    elif key == "4":
                        nplc.start_hub_counting("blue")
                        last_message = "start blue"
                    elif key == "5":
                        nplc.stop_hub_counting("blue")
                        last_message = "stop blue"
                    elif key == "6":
                        nplc.reset_hub_count("blue")
                        last_message = "reset blue"
                    elif key == "a":
                        nplc.start_hub_counting("red")
                        nplc.start_hub_counting("blue")
                        last_message = "start both"
                    elif key == "s":
                        nplc.stop_hub_counting("red")
                        nplc.stop_hub_counting("blue")
                        last_message = "stop both"
                    elif key == "d":
                        nplc.reset_hub_count("red")
                        nplc.reset_hub_count("blue")
                        last_message = "reset both"

                try:
                    red_status = nplc.get_hub_status("red")
                except RuntimeError as exc:
                    red_status = {"count": "ERR", "state": str(exc)}
                try:
                    blue_status = nplc.get_hub_status("blue")
                except RuntimeError as exc:
                    blue_status = {"count": "ERR", "state": str(exc)}

                sys.stdout.write("\x1b[2J\x1b[H")
                sys.stdout.write(render(red_status, blue_status, last_message))
                sys.stdout.flush()
                time.sleep(sleep_s)
        finally:
            termios.tcsetattr(stdin_fd, termios.TCSADRAIN, original_settings)
    except KeyboardInterrupt:
        sys.stdout.write("\n")


if __name__ == "__main__":
    main()