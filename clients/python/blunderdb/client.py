# The hand-written half of the client (#289, fiche I.33): the transport.
#
# What changes with the API is generated (_generated.py, one method per route,
# rewritten by `go run ./cmd/openapi-gen`). What changes with judgement is
# here: the session, the tenant header, the error envelope, the NDJSON decode.
# Neither half has to know much about the other.
"""Transport for the blunderDB engine API."""

from __future__ import annotations

import json
import urllib.error
import urllib.request
from typing import Any, Iterator, Optional


class APIError(RuntimeError):
    """An error the daemon returned in its own envelope.

    The daemon answers a failure with ``{"error": {"code", "message", "details"}}``
    and an HTTP status derived from the code. Both are kept: the code is what a
    program branches on (``not_found``, ``conflict``, ``invalid``,
    ``rate_limited``), the message is what a person reads.
    """

    def __init__(self, code: str, message: str, status: int, details: Optional[dict] = None):
        super().__init__(f"{code}: {message}")
        self.code = code
        self.message = message
        self.status = status
        self.details = details or {}


class BaseClient:
    """Talks to one daemon as one tenant.

    SECURITY. The daemon performs **no authentication of its own**: it trusts
    ``X-Tenant-ID`` verbatim and must run behind an authenticating reverse
    proxy (ADR-0005). This client therefore sends the tenant you give it and
    nothing else — it is not a credential, and treating it as one is the
    mistake ADR-0005 exists to prevent.
    """

    def __init__(self, base_url: str = "http://127.0.0.1:8080", tenant: int = 1, timeout: float = 30.0):
        self.base_url = base_url.rstrip("/")
        # The tenant is a positive decimal integer. A name is refused by the
        # daemon with 400 invalid rather than mapped to a tenant, so refusing
        # it here too turns a server round-trip into a local error.
        self.tenant = int(tenant)
        if self.tenant <= 0:
            raise ValueError("tenant must be a positive integer")
        self.timeout = timeout

    # -- the two verbs the generated methods use ---------------------------

    def _call(self, path: str, payload: Optional[dict] = None, *, idempotency_key: Optional[str] = None) -> Optional[Any]:
        """One JSON call. Returns the decoded body, or None for no content."""
        body = self._request(path, payload, idempotency_key)
        if not body.strip():
            return None
        return json.loads(body)

    def _stream(self, path: str, payload: Optional[dict] = None, *, idempotency_key: Optional[str] = None) -> Iterator[Any]:
        """One NDJSON call, decoded line by line.

        Lazily: a list endpoint on a large tenant streams, and materialising it
        would defeat the reason it streams.
        """
        request = self._build(path, payload, idempotency_key)
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                for line in response:
                    line = line.strip()
                    if line:
                        yield json.loads(line)
        except urllib.error.HTTPError as err:
            raise self._error(err) from None

    # -- plumbing ----------------------------------------------------------

    def _build(self, path: str, payload: Optional[dict], idempotency_key: Optional[str]) -> urllib.request.Request:
        data = json.dumps(payload if payload is not None else {}).encode("utf-8")
        headers = {
            "Content-Type": "application/json",
            "X-Tenant-ID": str(self.tenant),
        }
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        return urllib.request.Request(self.base_url + path, data=data, headers=headers, method="POST")

    def _request(self, path: str, payload: Optional[dict], idempotency_key: Optional[str]) -> str:
        request = self._build(path, payload, idempotency_key)
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                return response.read().decode("utf-8")
        except urllib.error.HTTPError as err:
            raise self._error(err) from None

    @staticmethod
    def _error(err: urllib.error.HTTPError) -> APIError:
        try:
            envelope = json.loads(err.read().decode("utf-8")).get("error", {})
        except Exception:
            envelope = {}
        return APIError(
            envelope.get("code", "unknown"),
            envelope.get("message", err.reason or "request failed"),
            err.code,
            envelope.get("details"),
        )

    # -- health ------------------------------------------------------------

    def healthy(self) -> bool:
        """Is the daemon up? ``/healthz`` needs no tenant and no body."""
        try:
            with urllib.request.urlopen(self.base_url + "/healthz", timeout=self.timeout) as response:
                return response.status == 200
        except Exception:
            return False
