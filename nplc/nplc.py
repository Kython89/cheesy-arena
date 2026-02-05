import json
from urllib import request, error


class NPLC:
    """Minimal abstraction over sensor microcontrollers, in place of the PLC"""

    def __init__(self, red_hub_base_url, blue_hub_base_url, timeout_sec=2.0):
        self._hub_base_urls = {
            "red": red_hub_base_url.rstrip("/"),
            "blue": blue_hub_base_url.rstrip("/"),
        }
        self._timeout_sec = timeout_sec

    def set_hub_base_url(self, hub, base_url):
        self._hub_base_urls[self._hub_key(hub)] = base_url.rstrip("/")

    def start_hub_counting(self, hub):
        return self._post_hub(hub, "/start")

    def stop_hub_counting(self, hub):
        return self._post_hub(hub, "/stop")

    def reset_hub_count(self, hub):
        return self._post_hub(hub, "/reset")

    def get_hub_status(self, hub):
        return self._get_hub(hub, "/status")

    def _get_hub(self, hub, path):
        return self._request_json(self._hub_url(hub, path), "GET")

    def _post_hub(self, hub, path):
        return self._request_json(self._hub_url(hub, path), "POST")

    def _hub_url(self, hub, path):
        base_url = self._hub_base_urls[self._hub_key(hub)]
        return f"{base_url}{path}"

    def _hub_key(self, hub):
        if not hub:
            raise ValueError("hub is required")
        hub = hub.lower()
        if hub not in ("red", "blue"):
            raise ValueError("hub must be 'red' or 'blue'")
        return hub

    def _request_json(self, url, method):
        req = request.Request(url, method=method)
        try:
            with request.urlopen(req, timeout=self._timeout_sec) as response:
                payload = response.read().decode("utf-8")
                if not payload:
                    return {}
                return json.loads(payload)
        except error.HTTPError as exc:
            message = exc.read().decode("utf-8").strip()
            raise RuntimeError(f"nplc http error {exc.code} from {url}: {message}") from exc
        except error.URLError as exc:
            raise RuntimeError(f"nplc connection error to {url}: {exc.reason}") from exc