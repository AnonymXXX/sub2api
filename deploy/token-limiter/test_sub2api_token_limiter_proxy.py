import contextlib
import http.client
import io
import json
import socketserver
import threading
import unittest
from http.server import ThreadingHTTPServer

import sub2api_token_limiter_proxy as proxy


class RecordingUpstreamHandler(socketserver.BaseRequestHandler):
    def handle(self):
        request = b""
        while b"\r\n\r\n" not in request:
            chunk = self.request.recv(4096)
            if not chunk:
                return
            request += chunk

        raw_headers, body = request.split(b"\r\n\r\n", 1)
        content_length = 0
        for raw_line in raw_headers.split(b"\r\n")[1:]:
            name, _, value = raw_line.partition(b":")
            if name.lower() == b"content-length":
                content_length = int(value.strip())
                break
        while len(body) < content_length:
            body += self.request.recv(content_length - len(body))

        self.server.received_requests.append(raw_headers + b"\r\n\r\n" + body)
        self.request.sendall(self.server.response_bytes)


class RecordingUpstreamServer(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True


class MetricsStateTest(unittest.TestCase):
    def test_render_contains_fixed_zero_value_series(self):
        metrics = proxy.MetricsState()
        metrics.request_started()
        metrics.request_finished()
        metrics.record_event("request_body_too_large")

        rendered = metrics.render_prometheus()

        self.assertIn("sub2api_token_limiter_requests_total 1", rendered)
        self.assertIn("sub2api_token_limiter_inflight_requests 0", rendered)
        self.assertIn(
            'sub2api_token_limiter_events_total{reason="request_body_too_large"} 1',
            rendered,
        )
        self.assertIn(
            'sub2api_token_limiter_events_total{reason="upstream_relay_timeout"} 0',
            rendered,
        )

    def test_alert_threshold_and_cooldown(self):
        metrics = proxy.MetricsState(alert_window_seconds=600, alert_cooldown_seconds=600)

        self.assertIsNone(metrics.record_event("request_body_too_large", "http_413", 3, now=1))
        self.assertIsNone(metrics.record_event("request_body_too_large", "http_413", 3, now=2))
        alert = metrics.record_event("request_body_too_large", "http_413", 3, now=3)
        self.assertEqual("http_413", alert["alert_group"])
        self.assertEqual(3, alert["window_count"])
        self.assertIsNone(metrics.record_event("request_body_too_large", "http_413", 3, now=4))

        self.assertIsNone(metrics.record_event("request_body_too_large", "http_413", 3, now=605))
        self.assertIsNone(metrics.record_event("request_body_too_large", "http_413", 3, now=606))
        second_alert = metrics.record_event("request_body_too_large", "http_413", 3, now=607)
        self.assertEqual("http_413", second_alert["alert_group"])
        self.assertEqual(2, metrics.snapshot()["alerts"]["http_413"])


class SanitizationTest(unittest.TestCase):
    def test_endpoint_category_drops_query_and_unknown_path(self):
        self.assertEqual("responses", proxy.endpoint_category("/v1/responses?token=secret"))
        self.assertEqual("other", proxy.endpoint_category("/users/private-name?token=secret"))

    def test_request_reference_is_one_way_and_truncated(self):
        raw_request_id = "private-request-id"
        reference = proxy.request_reference({"X-Request-ID": raw_request_id})
        self.assertEqual(12, len(reference))
        self.assertNotIn(raw_request_id, reference)

    def test_proxy_failure_reasons_are_fixed(self):
        self.assertEqual("upstream_connect_timeout", proxy.proxy_failure_reason(TimeoutError(), "connect"))
        self.assertEqual("upstream_relay_timeout", proxy.proxy_failure_reason(TimeoutError(), "relay"))
        self.assertEqual(
            "upstream_connect_refused",
            proxy.proxy_failure_reason(ConnectionRefusedError(), "connect"),
        )
        self.assertEqual("upstream_relay_error", proxy.proxy_failure_reason(OSError("secret"), "relay"))


class ProxyHandlerTest(unittest.TestCase):
    def setUp(self):
        proxy.METRICS = proxy.MetricsState(alert_window_seconds=600, alert_cooldown_seconds=600)
        self.original_upstream_port = proxy.UPSTREAM_PORT
        self.original_hard_output_char_limit = proxy.HARD_OUTPUT_CHAR_LIMIT
        self.upstream = None
        self.upstream_thread = None
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), proxy.ProxyHandler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=2)
        if self.upstream is not None:
            self.upstream.shutdown()
            self.upstream.server_close()
            self.upstream_thread.join(timeout=2)
        proxy.UPSTREAM_PORT = self.original_upstream_port
        proxy.HARD_OUTPUT_CHAR_LIMIT = self.original_hard_output_char_limit

    def start_upstream(self, response_bytes):
        self.upstream = RecordingUpstreamServer(("127.0.0.1", 0), RecordingUpstreamHandler)
        self.upstream.received_requests = []
        self.upstream.response_bytes = response_bytes
        proxy.UPSTREAM_PORT = self.upstream.server_address[1]
        self.upstream_thread = threading.Thread(target=self.upstream.serve_forever, daemon=True)
        self.upstream_thread.start()

    def test_metrics_endpoint_uses_dedicated_listener(self):
        metrics_server = ThreadingHTTPServer(("127.0.0.1", 0), proxy.MetricsHandler)
        metrics_thread = threading.Thread(target=metrics_server.serve_forever, daemon=True)
        metrics_thread.start()
        try:
            connection = http.client.HTTPConnection("127.0.0.1", metrics_server.server_port, timeout=2)
            connection.request("GET", "/metrics?secret=query-value")
            response = connection.getresponse()
            body = response.read().decode("ascii")
            connection.close()
        finally:
            metrics_server.shutdown()
            metrics_server.server_close()
            metrics_thread.join(timeout=2)

        self.assertEqual(200, response.status)
        self.assertIn("sub2api_token_limiter_requests_total 0", body)
        self.assertNotIn("query-value", body)

    def test_request_proxy_does_not_intercept_metrics_path(self):
        upstream_body = b"upstream-metrics-response"
        self.start_upstream(
            b"HTTP/1.1 404 Not Found\r\n"
            + f"Content-Length: {len(upstream_body)}\r\n".encode("ascii")
            + b"Content-Type: text/plain\r\nConnection: close\r\n\r\n"
            + upstream_body
        )
        connection = http.client.HTTPConnection("127.0.0.1", self.server.server_port, timeout=2)

        connection.request("GET", "/metrics")
        response = connection.getresponse()
        body = response.read()
        connection.close()

        self.assertEqual(404, response.status)
        self.assertEqual(upstream_body, body)
        self.assertTrue(self.upstream.received_requests[0].startswith(b"GET /metrics HTTP/1.1\r\n"))

    def test_oversized_request_is_counted_and_log_is_sanitized(self):
        output = io.StringIO()
        connection = http.client.HTTPConnection("127.0.0.1", self.server.server_port, timeout=2)
        with contextlib.redirect_stdout(output):
            connection.putrequest("POST", "/v1/responses?token=query-secret")
            connection.putheader("Content-Length", str(proxy.MAX_BODY_BYTES + 1))
            connection.putheader("X-Request-ID", "raw-request-secret")
            connection.endheaders()
            response = connection.getresponse()
            response.read()
        connection.close()

        self.assertEqual(413, response.status)
        snapshot = proxy.METRICS.snapshot()
        self.assertEqual(1, snapshot["events"]["request_body_too_large"])

        log_record = json.loads(output.getvalue().strip())
        self.assertEqual("token_limiter_error", log_record["event"])
        self.assertEqual("request_body_too_large", log_record["reason"])
        self.assertEqual("responses", log_record["endpoint"])
        self.assertNotIn("query-secret", output.getvalue())
        self.assertNotIn("raw-request-secret", output.getvalue())

    def test_unsupported_method_does_not_log_raw_path(self):
        output = io.StringIO()
        connection = http.client.HTTPConnection("127.0.0.1", self.server.server_port, timeout=2)
        with contextlib.redirect_stdout(output):
            connection.request("HEAD", "/private/path?token=query-secret")
            response = connection.getresponse()
            response.read()
        connection.close()

        self.assertEqual(501, response.status)
        self.assertEqual(1, proxy.METRICS.snapshot()["events"]["unsupported_method"])
        log_record = json.loads(output.getvalue().strip())
        self.assertEqual("other", log_record["endpoint"])
        self.assertNotIn("private/path", output.getvalue())
        self.assertNotIn("query-secret", output.getvalue())

    def test_token_rewrite_observation_is_counted_without_logging(self):
        handler = object.__new__(proxy.ProxyHandler)
        handler.command = "POST"
        handler.path = "/v1/chat/completions?token=query-secret"
        handler.headers = {"X-Request-ID": "raw-request-secret"}
        output = io.StringIO()

        with contextlib.redirect_stdout(output):
            handler.record_observation("token_limit_rewrite", emit_log=False)

        self.assertEqual(1, proxy.METRICS.snapshot()["events"]["token_limit_rewrite"])
        self.assertEqual("", output.getvalue())

    def test_chat_completion_is_rewritten_and_forwarded(self):
        upstream_body = b'{"ok":true}'
        self.start_upstream(
            b"HTTP/1.1 200 OK\r\n"
            + f"Content-Length: {len(upstream_body)}\r\n".encode("ascii")
            + b"Content-Type: application/json\r\nConnection: close\r\n\r\n"
            + upstream_body
        )
        request_body = json.dumps({"model": "test", "max_tokens": proxy.MAX_OUTPUT_TOKENS * 2})
        connection = http.client.HTTPConnection("127.0.0.1", self.server.server_port, timeout=2)

        connection.request(
            "POST",
            "/v1/chat/completions",
            body=request_body,
            headers={"Content-Type": "application/json"},
        )
        response = connection.getresponse()
        response_body = response.read()
        connection.close()

        self.assertEqual(200, response.status)
        self.assertEqual(upstream_body, response_body)
        forwarded_body = self.upstream.received_requests[0].split(b"\r\n\r\n", 1)[1]
        forwarded_payload = json.loads(forwarded_body.decode("utf-8"))
        self.assertEqual(proxy.MAX_OUTPUT_TOKENS, forwarded_payload["max_tokens"])
        self.assertEqual(1, proxy.METRICS.snapshot()["events"]["token_limit_rewrite"])

    def test_responses_sse_hard_cap_emits_incomplete_and_observation(self):
        proxy.HARD_OUTPUT_CHAR_LIMIT = 2
        event = {
            "type": "response.output_text.delta",
            "delta": "abc",
            "id": "response-id",
            "model": "test-model",
        }
        event_bytes = b"event: response.output_text.delta\n" + b"data: " + json.dumps(event).encode("utf-8") + b"\n\n"
        self.start_upstream(
            b"HTTP/1.1 200 OK\r\n"
            b"Content-Type: text/event-stream\r\n"
            b"Connection: close\r\n\r\n"
            + event_bytes
        )
        output = io.StringIO()
        connection = http.client.HTTPConnection("127.0.0.1", self.server.server_port, timeout=2)

        with contextlib.redirect_stdout(output):
            connection.request(
                "POST",
                "/v1/responses?token=query-secret",
                body='{"model":"test"}',
                headers={
                    "Content-Type": "application/json",
                    "X-Request-ID": "raw-request-secret",
                },
            )
            response = connection.getresponse()
            response_body = response.fp.readline() + response.fp.readline() + response.fp.readline()
        connection.close()

        self.assertEqual(200, response.status)
        self.assertIn(b"event: response.incomplete", response_body)
        self.assertNotIn(b'"delta": "abc"', response_body)
        self.assertEqual(1, proxy.METRICS.snapshot()["events"]["sse_hard_cap"])
        log_record = json.loads(output.getvalue().strip())
        self.assertEqual("sse_hard_cap", log_record["reason"])
        self.assertNotIn("query-secret", output.getvalue())
        self.assertNotIn("raw-request-secret", output.getvalue())


class ExistingBehaviorTest(unittest.TestCase):
    def test_chat_completion_limit_is_added(self):
        payload, changed = proxy.apply_token_limit({"model": "test"}, "/v1/chat/completions")
        self.assertTrue(changed)
        self.assertEqual(proxy.MAX_OUTPUT_TOKENS, payload["max_completion_tokens"])

    def test_responses_limit_is_not_rewritten(self):
        payload = {"max_output_tokens": proxy.MAX_OUTPUT_TOKENS * 2}
        result, changed = proxy.apply_token_limit(payload.copy(), "/v1/responses")
        self.assertFalse(changed)
        self.assertEqual(payload, result)

    def test_sse_frame_end_accepts_lf_and_crlf(self):
        self.assertEqual(3, proxy.find_sse_frame_end(b"a\n\nrest"))
        self.assertEqual(5, proxy.find_sse_frame_end(b"a\r\n\r\nrest"))


if __name__ == "__main__":
    unittest.main()
