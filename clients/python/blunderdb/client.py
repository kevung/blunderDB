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
from typing import Any, Iterable, Iterator, Optional, Union


class APIError(RuntimeError):
    """An error the daemon returned in its own envelope.

    The daemon answers a failure with ``{"error": {"code", "message", "details"}}``
    and an HTTP status derived from the code. Both are kept: the code is what a
    program branches on (``not_found``, ``unknown_route``, ``conflict``,
    ``invalid``, ``rate_limited``), the message is what a person reads.
    ``unknown_route`` is a method this daemon does not serve, never missing data.
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
        # The Direction-Version of the last answer that carried one — a read, a gesture, or the
        # 409 that refused a gesture: the If-Match of the next gesture.
        self.last_version: Optional[str] = None
        # The tenant is a positive decimal integer. A name is refused by the
        # daemon with 400 invalid rather than mapped to a tenant, so refusing
        # it here too turns a server round-trip into a local error.
        self.tenant = int(tenant)
        if self.tenant <= 0:
            raise ValueError("tenant must be a positive integer")
        self.timeout = timeout

    # -- the two verbs the generated methods use ---------------------------

    def _call(
        self,
        path: str,
        payload: Optional[dict] = None,
        *,
        idempotency_key: Optional[str] = None,
        if_match: Optional[Union[int, str]] = None,
    ) -> Optional[Any]:
        """One JSON call. Returns the decoded body, or None for no content."""
        body = self._request(path, payload, idempotency_key, if_match)
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

    # -- events ------------------------------------------------------------

    def events(
        self,
        *,
        tournament: Optional[Iterable[int]] = None,
        rencontre: Optional[Iterable[int]] = None,
        transcription: Optional[Iterable[int]] = None,
        last_event_id: Optional[str] = None,
        timeout: Optional[float] = None,
    ) -> Iterator[dict]:
        """Follow ``GET /v1/events``: one dict per committed gesture of the tenant.

        Each event names what moved and its new ``version`` (or ``revision``),
        never the state: read it again. A ``{"kind": "resync"}`` event means
        events may have been missed and everything shown must be read again:
        every stream opens with one (``reason`` "reconnected" when
        ``last_event_id`` is given, "subscribed" otherwise), and a subscriber
        dropped for falling behind gets one before the stream ends. The
        iterator ends when the daemon closes the stream; the caller reconnects
        with the last ``_id`` it saw. ``timeout`` (default: none) bounds the
        silence between two frames; the daemon sends a heartbeat every 25 s.
        """
        query = []
        for name, ids in (("tournament", tournament), ("rencontre", rencontre), ("transcription", transcription)):
            if ids:
                query.append("%s=%s" % (name, ",".join(str(int(i)) for i in ids)))
        url = self.base_url + "/v1/events" + ("?" + "&".join(query) if query else "")
        headers = {"Accept": "text/event-stream", "X-Tenant-ID": str(self.tenant)}
        if last_event_id:
            headers["Last-Event-ID"] = last_event_id
        request = urllib.request.Request(url, headers=headers, method="GET")
        try:
            with urllib.request.urlopen(request, timeout=timeout) as response:
                event_id, data = None, []
                for raw in response:
                    line = raw.decode("utf-8").rstrip("\r\n")
                    if line == "":
                        if data:
                            event = json.loads("\n".join(data))
                            if event_id:
                                event["_id"] = event_id
                            yield event
                        event_id, data = None, []
                    elif line.startswith("id:"):
                        event_id = line[3:].strip()
                    elif line.startswith("data:"):
                        data.append(line[5:].strip())
        except urllib.error.HTTPError as err:
            raise self._error(err) from None

    # -- plumbing ----------------------------------------------------------

    def _build(
        self, path: str, payload: Optional[dict], idempotency_key: Optional[str], if_match: Optional[Union[int, str]] = None
    ) -> urllib.request.Request:
        data = json.dumps(payload if payload is not None else {}).encode("utf-8")
        headers = {
            "Content-Type": "application/json",
            "X-Tenant-ID": str(self.tenant),
        }
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        if if_match is not None:
            # A transcription's revision (an integer) or a Direction-Version (a token).
            headers["If-Match"] = '"%s"' % if_match
        return urllib.request.Request(self.base_url + path, data=data, headers=headers, method="POST")

    def _request(
        self, path: str, payload: Optional[dict], idempotency_key: Optional[str], if_match: Optional[Union[int, str]] = None
    ) -> str:
        request = self._build(path, payload, idempotency_key, if_match)
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                self._note_version(response.headers)
                return response.read().decode("utf-8")
        except urllib.error.HTTPError as err:
            self._note_version(err.headers)
            raise self._error(err) from None

    def _note_version(self, headers: Any) -> None:
        version = headers.get("Direction-Version") if headers is not None else None
        if version:
            self.last_version = version.strip('"')

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
