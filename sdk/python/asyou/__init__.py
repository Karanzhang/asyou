"""asyou — Python SDK for the asyou management API."""

import json
import urllib.request
import urllib.error
from dataclasses import dataclass
from typing import Optional


@dataclass
class Proxy:
    id: int
    name: str
    type: str
    local_ip: str
    local_port: int
    status: str
    node_id: Optional[int] = None


@dataclass
class Node:
    id: int
    name: str
    host: str
    bind_port: int


class AsyouError(Exception):
    pass


class Client:
    """asyou API client."""

    def __init__(self, server_url: str = "http://localhost:8080"):
        self.base_url = server_url.rstrip("/")
        self.token: Optional[str] = None

    def _request(self, method: str, path: str, body: dict = None):
        url = f"{self.base_url}{path}"
        data = json.dumps(body).encode() if body else None
        req = urllib.request.Request(url, data=data, method=method)
        req.add_header("Content-Type", "application/json")
        if self.token:
            req.add_header("Authorization", f"Bearer {self.token}")
        try:
            with urllib.request.urlopen(req) as resp:
                raw = resp.read()
                if raw:
                    return json.loads(raw)
                return None
        except urllib.error.HTTPError as e:
            msg = f"HTTP {e.code}"
            try:
                err = json.loads(e.read())
                msg = err.get("error", msg)
            except Exception:
                pass
            raise AsyouError(msg)

    def login(self, email: str, password: str):
        """Authenticate and store the JWT token."""
        res = self._request("POST", "/api/v1/auth/login", {
            "email": email, "password": password
        })
        self.token = res["access_token"]

    def register(self, email: str, password: str, display_name: str = ""):
        """Create an account and log in."""
        self._request("POST", "/api/v1/auth/register", {
            "email": email, "password": password, "display_name": display_name
        })
        self.login(email, password)

    def list_proxies(self):
        """List all tunnels."""
        data = self._request("GET", "/api/v1/proxies") or []
        return [Proxy(**p) for p in data]

    def create_proxy(self, name: str, proxy_type: str = "tcp",
                     local_port: int = 8080, node_id: int = None) -> Proxy:
        """Create a new tunnel."""
        body = {"name": name, "type": proxy_type, "local_port": local_port}
        if node_id:
            body["node_id"] = node_id
        data = self._request("POST", "/api/v1/proxies", body)
        return Proxy(**data)

    def delete_proxy(self, proxy_id: int):
        """Delete a tunnel."""
        self._request("DELETE", f"/api/v1/proxies/{proxy_id}")

    def proxy_action(self, proxy_id: int, action: str):
        """Send lifecycle action: start, stop, reload."""
        self._request("POST", f"/api/v1/proxies/{proxy_id}/action",
                      {"action": action})

    def list_nodes(self):
        """List all frps nodes."""
        data = self._request("GET", "/api/v1/nodes") or []
        return [Node(**n) for n in data]

    # --- Convenience ---

    def expose(self, local_port: int, name: str = None,
               node_id: int = None) -> Proxy:
        """Create and start a tunnel in one step."""
        if not name:
            name = f"py-tunnel-{local_port}"
        proxy = self.create_proxy(name, "tcp", local_port, node_id)
        self.proxy_action(proxy.id, "start")
        return proxy
