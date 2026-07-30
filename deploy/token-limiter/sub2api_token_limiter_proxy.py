#!/usr/bin/env python3
import hashlib
import json
import os
import select
import socket
import sys
import threading
import time
from collections import Counter, defaultdict, deque
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlsplit


LISTEN_HOST = os.environ.get("LISTEN_HOST", "127.0.0.1")
LISTEN_PORT = int(os.environ.get("LISTEN_PORT", "18080"))
METRICS_HOST = os.environ.get("METRICS_HOST", "127.0.0.1")
METRICS_PORT = int(os.environ.get("METRICS_PORT", "18081"))
UPSTREAM_HOST = os.environ.get("UPSTREAM_HOST", "127.0.0.1")
UPSTREAM_PORT = int(os.environ.get("UPSTREAM_PORT", "8080"))
MAX_OUTPUT_TOKENS = int(os.environ.get("MAX_OUTPUT_TOKENS", "8192"))
ENABLE_SSE_HARD_CAP = os.environ.get("ENABLE_SSE_HARD_CAP", "true").lower() in {"1", "true", "yes", "on"}
HARD_OUTPUT_CHAR_LIMIT = int(os.environ.get("HARD_OUTPUT_CHAR_LIMIT", str(MAX_OUTPUT_TOKENS * 4)))
MAX_BODY_BYTES = int(os.environ.get("MAX_BODY_BYTES", str(16 * 1024 * 1024)))
CONNECT_TIMEOUT_SECONDS = float(os.environ.get("CONNECT_TIMEOUT_SECONDS", "30"))
UPSTREAM_IDLE_TIMEOUT_SECONDS = float(os.environ.get("UPSTREAM_IDLE_TIMEOUT_SECONDS", "900"))
ALERT_WINDOW_SECONDS = int(os.environ.get("ALERT_WINDOW_SECONDS", "600"))
ALERT_COOLDOWN_SECONDS = int(os.environ.get("ALERT_COOLDOWN_SECONDS", "600"))
ALERT_413_THRESHOLD = int(os.environ.get("ALERT_413_THRESHOLD", "5"))
ALERT_502_THRESHOLD = int(os.environ.get("ALERT_502_THRESHOLD", "3"))

HOP_BY_HOP_HEADERS = {
    "connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailers",
    "transfer-encoding",
    "upgrade",
}

ENDPOINT_CATEGORIES = {
    "/v1/responses": "responses",
    "/responses": "responses",
    "/v1/chat/completions": "chat_completions",
    "/chat/completions": "chat_completions",
    "/health": "health",
    "/metrics": "metrics",
}

EVENT_REASONS = (
    "downstream_disconnect",
    "other_local_error",
    "request_body_too_large",
    "sse_hard_cap",
    "token_limit_rewrite",
    "unsupported_method",
    "upstream_connect_error",
    "upstream_connect_refused",
    "upstream_connect_timeout",
    "upstream_relay_error",
    "upstream_relay_timeout",
)

ALERT_GROUPS = ("http_413", "http_502")
LOG_LOCK = threading.Lock()


def clamp_number(value, limit):
    if isinstance(value, bool):
        return limit
    if isinstance(value, int):
        return min(value, limit)
    if isinstance(value, float):
        return min(int(value), limit)
    return limit


def apply_token_limit(payload, path):
    if not isinstance(payload, dict):
        return payload, False

    changed = False
    clean_path = urlsplit(path).path
    if clean_path in {"/v1/responses", "/responses"}:
        # ChatGPT internal Codex upstream rejects public Responses output-limit
        # fields. Keep SSE hard-cap enforcement, but do not inject or rewrite
        # max_output_tokens on Responses requests.
        return payload, False

    default_field = "max_completion_tokens"
    fields = ("max_completion_tokens", "max_tokens")

    present = [field for field in fields if field in payload]
    if present:
        for field in present:
            original = payload[field]
            updated = clamp_number(original, MAX_OUTPUT_TOKENS)
            if updated != original:
                payload[field] = updated
                changed = True
    else:
        payload[default_field] = MAX_OUTPUT_TOKENS
        changed = True

    return payload, changed


def should_rewrite(method, path, content_type):
    if method.upper() not in {"POST", "PUT", "PATCH"}:
        return False
    clean_path = urlsplit(path).path
    if clean_path not in {"/v1/responses", "/responses", "/v1/chat/completions", "/chat/completions"}:
        return False
    return "application/json" in (content_type or "").lower()


def is_responses_path(path):
    return urlsplit(path).path in {"/v1/responses", "/responses"}


def find_sse_frame_end(buffer):
    candidates = []
    for separator in (b"\r\n\r\n", b"\n\n"):
        index = buffer.find(separator)
        if index >= 0:
            candidates.append((index, len(separator)))
    if not candidates:
        return None
    index, separator_length = min(candidates, key=lambda item: item[0])
    return index + separator_length


def endpoint_category(path):
    return ENDPOINT_CATEGORIES.get(urlsplit(path).path, "other")


def request_reference(headers):
    request_id = headers.get("X-Request-ID") or headers.get("X-Client-Request-ID")
    if not request_id:
        return ""
    return hashlib.sha256(request_id.encode("utf-8", errors="ignore")).hexdigest()[:12]


def proxy_failure_reason(error, phase):
    if isinstance(error, TimeoutError):
        return "upstream_connect_timeout" if phase == "connect" else "upstream_relay_timeout"
    if isinstance(error, ConnectionRefusedError):
        return "upstream_connect_refused"
    return "upstream_connect_error" if phase == "connect" else "upstream_relay_error"


def emit_json(level, event, **fields):
    record = {
        "timestamp": datetime.now(timezone.utc).isoformat(timespec="milliseconds").replace("+00:00", "Z"),
        "level": level,
        "event": event,
    }
    record.update({key: value for key, value in fields.items() if value not in (None, "")})
    encoded = json.dumps(record, separators=(",", ":"), sort_keys=True, ensure_ascii=True)
    with LOG_LOCK:
        sys.stdout.write(encoded + "\n")
        sys.stdout.flush()


class MetricsState:
    def __init__(self, alert_window_seconds=ALERT_WINDOW_SECONDS, alert_cooldown_seconds=ALERT_COOLDOWN_SECONDS):
        self.alert_window_seconds = alert_window_seconds
        self.alert_cooldown_seconds = alert_cooldown_seconds
        self.started_monotonic = time.monotonic()
        self.lock = threading.Lock()
        self.requests_total = 0
        self.inflight_requests = 0
        self.events = Counter()
        self.alerts = Counter()
        self.recent_events = defaultdict(deque)
        self.last_alert_at = {}

    def request_started(self):
        with self.lock:
            self.requests_total += 1
            self.inflight_requests += 1

    def request_finished(self):
        with self.lock:
            self.inflight_requests = max(0, self.inflight_requests - 1)

    def record_event(self, reason, alert_group=None, threshold=0, now=None):
        now = time.monotonic() if now is None else now
        with self.lock:
            self.events[reason] += 1
            if not alert_group or threshold <= 0:
                return None

            recent = self.recent_events[alert_group]
            recent.append(now)
            cutoff = now - self.alert_window_seconds
            while recent and recent[0] < cutoff:
                recent.popleft()

            last_alert = self.last_alert_at.get(alert_group)
            if len(recent) < threshold:
                return None
            if last_alert is not None and now - last_alert < self.alert_cooldown_seconds:
                return None

            self.last_alert_at[alert_group] = now
            self.alerts[alert_group] += 1
            return {
                "alert_group": alert_group,
                "window_count": len(recent),
                "threshold": threshold,
                "window_seconds": self.alert_window_seconds,
            }

    def snapshot(self):
        with self.lock:
            return {
                "uptime_seconds": max(0.0, time.monotonic() - self.started_monotonic),
                "requests_total": self.requests_total,
                "inflight_requests": self.inflight_requests,
                "events": Counter(self.events),
                "alerts": Counter(self.alerts),
            }

    def render_prometheus(self):
        snapshot = self.snapshot()
        lines = [
            "# HELP sub2api_token_limiter_uptime_seconds Process uptime in seconds.",
            "# TYPE sub2api_token_limiter_uptime_seconds gauge",
            f"sub2api_token_limiter_uptime_seconds {snapshot['uptime_seconds']:.3f}",
            "# HELP sub2api_token_limiter_requests_total Forwarded requests received by the limiter.",
            "# TYPE sub2api_token_limiter_requests_total counter",
            f"sub2api_token_limiter_requests_total {snapshot['requests_total']}",
            "# HELP sub2api_token_limiter_inflight_requests Requests currently being forwarded.",
            "# TYPE sub2api_token_limiter_inflight_requests gauge",
            f"sub2api_token_limiter_inflight_requests {snapshot['inflight_requests']}",
            "# HELP sub2api_token_limiter_events_total Limiter events by fixed reason.",
            "# TYPE sub2api_token_limiter_events_total counter",
        ]
        for reason in EVENT_REASONS:
            lines.append(f'sub2api_token_limiter_events_total{{reason="{reason}"}} {snapshot["events"][reason]}')
        lines.extend(
            [
                "# HELP sub2api_token_limiter_alerts_total Locally emitted rolling-window alerts.",
                "# TYPE sub2api_token_limiter_alerts_total counter",
            ]
        )
        for group in ALERT_GROUPS:
            lines.append(f'sub2api_token_limiter_alerts_total{{group="{group}"}} {snapshot["alerts"][group]}')
        return "\n".join(lines) + "\n"


METRICS = MetricsState()


class MetricsHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, _format, *_args):
        return

    def do_GET(self):
        self.serve_metrics()

    def do_HEAD(self):
        self.serve_metrics()

    def serve_metrics(self):
        if urlsplit(self.path).path != "/metrics":
            self.send_error(404)
            return
        body = METRICS.render_prometheus().encode("ascii")
        self.send_response(200)
        self.send_header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.send_header("Connection", "close")
        self.end_headers()
        if self.command != "HEAD":
            self.wfile.write(body)


class ProxyHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, _format, *_args):
        return

    def log_error(self, _format, *_args):
        return

    def log_request(self, _code="-", _size="-"):
        return

    def do_GET(self):
        self.forward()

    def do_POST(self):
        self.forward()

    def do_PUT(self):
        self.forward()

    def do_PATCH(self):
        self.forward()

    def do_DELETE(self):
        self.forward()

    def do_OPTIONS(self):
        self.forward()

    def send_tracked_error(self, code, message, reason, started_at=None, content_length=None):
        self._tracked_error = {
            "reason": reason,
            "started_at": started_at,
            "content_length": content_length,
        }
        try:
            self.send_error(code, message)
        finally:
            self._tracked_error = None

    def send_error(self, code, message=None, explain=None):
        tracked = getattr(self, "_tracked_error", None) or {}
        reason = tracked.get("reason")
        if not reason:
            reason = "unsupported_method" if code == 501 else "other_local_error"

        started_at = tracked.get("started_at")
        elapsed_ms = None
        if started_at is not None:
            elapsed_ms = int((time.monotonic() - started_at) * 1000)

        alert_group = None
        threshold = 0
        if code == 413:
            alert_group = "http_413"
            threshold = ALERT_413_THRESHOLD
        elif code == 502:
            alert_group = "http_502"
            threshold = ALERT_502_THRESHOLD

        alert = METRICS.record_event(reason, alert_group=alert_group, threshold=threshold)
        event_fields = {
            "reason": reason,
            "status_code": code,
            "method": self.command,
            "endpoint": endpoint_category(self.path),
            "request_ref": request_reference(self.headers),
            "elapsed_ms": elapsed_ms,
            "content_length": tracked.get("content_length"),
        }
        if code == 413:
            event_fields["body_limit_bytes"] = MAX_BODY_BYTES
        emit_json("warning", "token_limiter_error", **event_fields)
        if alert:
            emit_json("warning", "token_limiter_alert", **alert)

        super().send_error(code, message, explain)

    def record_observation(self, reason, emit_log=True, **fields):
        METRICS.record_event(reason)
        if not emit_log:
            return
        emit_json(
            "info",
            "token_limiter_event",
            reason=reason,
            method=self.command,
            endpoint=endpoint_category(self.path),
            request_ref=request_reference(self.headers),
            **fields,
        )

    def forward(self):
        started_at = time.monotonic()
        METRICS.request_started()
        try:
            content_length = int(self.headers.get("Content-Length", "0") or "0")
            if content_length > MAX_BODY_BYTES:
                self.send_tracked_error(
                    413,
                    "Request body too large",
                    "request_body_too_large",
                    started_at=started_at,
                    content_length=content_length,
                )
                return

            body = self.rfile.read(content_length) if content_length else b""
            headers = {
                key: value
                for key, value in self.headers.items()
                if key.lower() not in HOP_BY_HOP_HEADERS and key.lower() != "host"
            }

            if body and should_rewrite(self.command, self.path, self.headers.get("Content-Type")):
                try:
                    payload = json.loads(body.decode("utf-8"))
                    payload, changed = apply_token_limit(payload, self.path)
                    if changed:
                        body = json.dumps(payload, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
                        headers = {key: value for key, value in headers.items() if key.lower() != "content-length"}
                        headers["Content-Length"] = str(len(body))
                        headers["X-Sub2API-Max-Output-Tokens"] = str(MAX_OUTPUT_TOKENS)
                        self.record_observation("token_limit_rewrite", emit_log=False)
                except (UnicodeDecodeError, json.JSONDecodeError):
                    pass

            headers["Host"] = self.headers.get("Host", f"{UPSTREAM_HOST}:{UPSTREAM_PORT}")
            headers["Connection"] = "close"
            request_bytes = self.build_raw_request(headers, body)
            phase = "connect"
            try:
                with socket.create_connection((UPSTREAM_HOST, UPSTREAM_PORT), timeout=CONNECT_TIMEOUT_SECONDS) as upstream:
                    phase = "relay"
                    upstream.settimeout(UPSTREAM_IDLE_TIMEOUT_SECONDS)
                    upstream.sendall(request_bytes)
                    if ENABLE_SSE_HARD_CAP and is_responses_path(self.path):
                        self.relay_responses_with_hard_cap(upstream)
                    else:
                        self.relay_raw_response(upstream)
            except (BrokenPipeError, ConnectionResetError):
                self.record_observation("downstream_disconnect")
                return
            except OSError as error:
                reason = proxy_failure_reason(error, phase)
                try:
                    self.send_tracked_error(
                        502,
                        "Upstream proxy error",
                        reason,
                        started_at=started_at,
                        content_length=content_length,
                    )
                except (BrokenPipeError, ConnectionResetError):
                    return
        finally:
            METRICS.request_finished()

    def build_raw_request(self, headers, body):
        lines = [f"{self.command} {self.path} HTTP/1.1"]
        if body and "Content-Length" not in headers:
            headers["Content-Length"] = str(len(body))
        if not body:
            headers.pop("Content-Length", None)
        for key, value in headers.items():
            lines.append(f"{key}: {value}")
        return ("\r\n".join(lines) + "\r\n\r\n").encode("iso-8859-1") + body

    def relay_raw_response(self, upstream):
        while True:
            readable, _, _ = select.select([upstream], [], [], UPSTREAM_IDLE_TIMEOUT_SECONDS)
            if not readable:
                raise TimeoutError("upstream response timed out")
            chunk = upstream.recv(16 * 1024)
            if not chunk:
                break
            self.connection.sendall(chunk)

    def relay_responses_with_hard_cap(self, upstream):
        header_buffer = b""
        while b"\r\n\r\n" not in header_buffer and b"\n\n" not in header_buffer:
            chunk = upstream.recv(4096)
            if not chunk:
                if header_buffer:
                    self.connection.sendall(header_buffer)
                return
            header_buffer += chunk
            if len(header_buffer) > 256 * 1024:
                self.connection.sendall(header_buffer)
                self.relay_raw_response(upstream)
                return

        header_end = find_sse_frame_end(header_buffer)
        if header_end is None:
            self.connection.sendall(header_buffer)
            self.relay_raw_response(upstream)
            return

        raw_headers = header_buffer[:header_end]
        body_buffer = header_buffer[header_end:]
        self.connection.sendall(self.filter_response_headers(raw_headers))

        content_type = raw_headers.decode("iso-8859-1", errors="ignore").lower()
        if "text/event-stream" not in content_type:
            if body_buffer:
                self.connection.sendall(body_buffer)
            self.relay_raw_response(upstream)
            return

        sse_buffer = body_buffer
        output_chars = 0
        event_count = 0
        response_id = "resp_sub2api_token_limit"
        model = ""

        while True:
            frame_end = find_sse_frame_end(sse_buffer)
            while frame_end is not None:
                frame = sse_buffer[:frame_end]
                sse_buffer = sse_buffer[frame_end:]
                event_count += 1

                parsed = self.parse_sse_json(frame)
                if parsed:
                    response = parsed.get("response")
                    if isinstance(response, dict):
                        response_id = response.get("id") or response_id
                        model = response.get("model") or model
                    if parsed.get("id"):
                        response_id = parsed.get("id")
                    if parsed.get("model"):
                        model = parsed.get("model")
                    if parsed.get("type") == "response.output_text.delta":
                        delta = parsed.get("delta")
                        if isinstance(delta, str):
                            output_chars += len(delta)

                if output_chars > HARD_OUTPUT_CHAR_LIMIT:
                    self.record_observation(
                        "sse_hard_cap",
                        output_chars=output_chars,
                        output_char_limit=HARD_OUTPUT_CHAR_LIMIT,
                    )
                    self.send_incomplete_sse(response_id, model, event_count)
                    return

                self.connection.sendall(frame)
                frame_end = find_sse_frame_end(sse_buffer)

            readable, _, _ = select.select([upstream], [], [], UPSTREAM_IDLE_TIMEOUT_SECONDS)
            if not readable:
                raise TimeoutError("upstream response timed out")
            chunk = upstream.recv(16 * 1024)
            if not chunk:
                if sse_buffer:
                    self.connection.sendall(sse_buffer)
                return
            sse_buffer += chunk

    def filter_response_headers(self, raw_headers):
        text = raw_headers.decode("iso-8859-1", errors="replace")
        lines = text.replace("\r\n", "\n").split("\n")
        kept = []
        for line in lines:
            if not line:
                continue
            if ":" not in line:
                kept.append(line)
                continue
            key = line.split(":", 1)[0].lower()
            if key in {"content-length"}:
                continue
            kept.append(line)
        kept.append("")
        kept.append("")
        return "\r\n".join(kept).encode("iso-8859-1")

    def parse_sse_json(self, frame):
        data_lines = []
        for raw_line in frame.replace(b"\r\n", b"\n").split(b"\n"):
            if raw_line.startswith(b"data:"):
                data_lines.append(raw_line[5:].strip())
        if not data_lines:
            return None
        data = b"\n".join(data_lines)
        if data == b"[DONE]":
            return None
        try:
            parsed = json.loads(data.decode("utf-8"))
        except (UnicodeDecodeError, json.JSONDecodeError):
            return None
        return parsed if isinstance(parsed, dict) else None

    def send_incomplete_sse(self, response_id, model, sequence_number):
        payload = {
            "type": "response.incomplete",
            "sequence_number": sequence_number,
            "response": {
                "id": response_id,
                "object": "response",
                "model": model,
                "status": "incomplete",
                "output": [],
                "incomplete_details": {"reason": "max_output_tokens"},
            },
        }
        encoded = json.dumps(payload, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
        self.connection.sendall(b"event: response.incomplete\n")
        self.connection.sendall(b"data: " + encoded + b"\n\n")


def main():
    proxy_server = ThreadingHTTPServer((LISTEN_HOST, LISTEN_PORT), ProxyHandler)
    metrics_server = ThreadingHTTPServer((METRICS_HOST, METRICS_PORT), MetricsHandler)
    metrics_thread = threading.Thread(target=metrics_server.serve_forever, daemon=True)
    metrics_thread.start()
    emit_json(
        "info",
        "token_limiter_started",
        listen_port=LISTEN_PORT,
        metrics_port=METRICS_PORT,
        upstream_port=UPSTREAM_PORT,
        max_output_tokens=MAX_OUTPUT_TOKENS,
        max_body_bytes=MAX_BODY_BYTES,
        alert_window_seconds=ALERT_WINDOW_SECONDS,
        alert_413_threshold=ALERT_413_THRESHOLD,
        alert_502_threshold=ALERT_502_THRESHOLD,
    )
    try:
        proxy_server.serve_forever()
    finally:
        metrics_server.shutdown()
        metrics_server.server_close()
        proxy_server.server_close()


if __name__ == "__main__":
    main()
