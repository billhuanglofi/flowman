"""
Mock HTTP runner for demo/testing purposes

Returns realistic fake responses without making real HTTP calls.
"""

import time
import json
from typing import Dict
from flowman.config import Request, Environment


class MockResponse:
    """Mock response object"""

    def __init__(self, status_code: int, headers: Dict, body: bytes, duration_ms: float, trace: Dict = None):
        self.status_code = status_code
        self.headers = headers
        self.body = body
        self.duration_ms = duration_ms
        self.trace = trace or {}

    @property
    def body_text(self) -> str:
        return self.body.decode('utf-8', errors='replace')

    def __repr__(self):
        return f"<MockResponse {self.status_code} {len(self.body)} bytes in {self.duration_ms:.0f}ms>"


class MockRunner:
    """Mock HTTP request runner for demos"""

    def __init__(self):
        self.call_count = 0

    def run(self, request: Request, environment: Environment) -> MockResponse:
        """Return a fake response based on request method"""
        self.call_count += 1

        # Simulate network delay
        time.sleep(0.1 + (0.05 * (self.call_count % 3)))
        duration_ms = 100 + (50 * (self.call_count % 5))

        # Generate trace data
        trace = self._generate_trace(request, environment, duration_ms)

        # Generate mock response based on method
        if request.method == "GET":
            response = self._mock_get_response(request, duration_ms)
        elif request.method == "POST":
            response = self._mock_post_response(request, duration_ms)
        elif request.method == "PUT":
            response = self._mock_put_response(request, duration_ms)
        elif request.method == "DELETE":
            response = self._mock_delete_response(request, duration_ms)
        else:
            response = self._mock_generic_response(request, duration_ms)

        # Add trace to response
        response.trace = trace
        return response

    def _generate_trace(self, request: Request, environment: Environment, duration_ms: float) -> Dict:
        """Generate realistic HTTP trace data"""
        dns_time = 5 + (self.call_count % 10)
        tcp_time = 15 + (self.call_count % 20)
        tls_time = 25 + (self.call_count % 30)
        server_time = duration_ms - dns_time - tcp_time - tls_time - 10
        transfer_time = 5 + (self.call_count % 8)

        return {
            "request_id": f"req_{self.call_count:06d}",
            "timestamps": {
                "dns_start": 0,
                "dns_end": dns_time,
                "tcp_start": dns_time,
                "tcp_end": dns_time + tcp_time,
                "tls_start": dns_time + tcp_time,
                "tls_end": dns_time + tcp_time + tls_time,
                "request_start": dns_time + tcp_time + tls_time,
                "request_end": dns_time + tcp_time + tls_time + server_time,
                "response_start": dns_time + tcp_time + tls_time + server_time,
                "response_end": duration_ms,
            },
            "timings": {
                "dns_lookup": f"{dns_time:.1f}ms",
                "tcp_connection": f"{tcp_time:.1f}ms",
                "tls_handshake": f"{tls_time:.1f}ms",
                "server_processing": f"{server_time:.1f}ms",
                "content_transfer": f"{transfer_time:.1f}ms",
                "total": f"{duration_ms:.1f}ms"
            },
            "network": {
                "local_address": "192.168.1.100:54321",
                "remote_address": f"{environment.base_url.replace('https://', '').replace('http://', '')}:443",
                "protocol": "HTTP/2.0",
                "tls_version": "TLSv1.3",
                "cipher_suite": "TLS_AES_256_GCM_SHA384"
            },
            "request": {
                "method": request.method,
                "url": f"{environment.base_url}{request.path or request.endpoint}",
                "headers_sent": len(request.headers or []) + 4,  # + standard headers
                "body_size": "0 bytes" if request.method == "GET" else "256 bytes"
            },
            "response": {
                "status_code": 200,
                "headers_received": 8,
                "body_size": f"{len(self._get_mock_body(request))} bytes",
                "compressed": False
            },
            "events": [
                {"time": 0, "event": "DNS lookup started", "detail": f"Resolving {environment.base_url}"},
                {"time": dns_time, "event": "DNS lookup complete", "detail": "IP: 203.0.113.42"},
                {"time": dns_time, "event": "TCP connection initiated", "detail": "Connecting to port 443"},
                {"time": dns_time + tcp_time, "event": "TCP connection established", "detail": "3-way handshake complete"},
                {"time": dns_time + tcp_time, "event": "TLS handshake started", "detail": "ClientHello sent"},
                {"time": dns_time + tcp_time + tls_time, "event": "TLS handshake complete", "detail": "Using TLSv1.3"},
                {"time": dns_time + tcp_time + tls_time, "event": "HTTP request sent", "detail": f"{request.method} {request.path or request.endpoint}"},
                {"time": dns_time + tcp_time + tls_time + server_time, "event": "Response headers received", "detail": "Status: 200 OK"},
                {"time": duration_ms, "event": "Response body received", "detail": "Transfer complete"},
            ]
        }

    def _get_mock_body(self, request: Request) -> str:
        """Get sample body for size calculation"""
        return json.dumps({"status": "success", "data": {}}, indent=2)

    def _mock_get_response(self, request: Request, duration_ms: float) -> MockResponse:
        """Mock GET response"""
        if "user" in request.name.lower():
            body = {
                "id": "usr_abc123",
                "name": "John Doe",
                "email": "john.doe@example.com",
                "created_at": "2024-01-15T10:30:00Z",
                "status": "active"
            }
        elif "order" in request.name.lower():
            body = {
                "orders": [
                    {"id": "ord_001", "amount": 129.99, "status": "completed"},
                    {"id": "ord_002", "amount": 89.50, "status": "pending"},
                    {"id": "ord_003", "amount": 249.00, "status": "shipped"}
                ],
                "total": 3,
                "page": 1
            }
        elif "payment" in request.name.lower():
            body = {
                "id": "pay_xyz789",
                "amount": 100.00,
                "currency": "USD",
                "status": "succeeded",
                "customer_id": "cust_123"
            }
        else:
            body = {
                "status": "success",
                "data": {"message": "GET request successful"},
                "timestamp": "2024-06-23T12:00:00Z"
            }

        return MockResponse(
            status_code=200,
            headers={
                "content-type": "application/json",
                "x-request-id": f"req_{self.call_count:06d}",
                "x-response-time": f"{duration_ms:.0f}ms"
            },
            body=json.dumps(body, indent=2).encode('utf-8'),
            duration_ms=duration_ms
        )

    def _mock_post_response(self, request: Request, duration_ms: float) -> MockResponse:
        """Mock POST response"""
        if "payment" in request.name.lower():
            body = {
                "id": f"pay_{self.call_count:06d}",
                "status": "processing",
                "amount": 100.00,
                "currency": "USD",
                "created_at": "2024-06-23T12:00:00Z",
                "transaction_id": f"txn_{self.call_count:010d}"
            }
            status_code = 202
        elif "order" in request.name.lower():
            body = {
                "id": f"ord_{self.call_count:06d}",
                "status": "created",
                "items": [],
                "total": 0.00,
                "created_at": "2024-06-23T12:00:00Z"
            }
            status_code = 201
        elif "user" in request.name.lower():
            body = {
                "id": f"usr_{self.call_count:06d}",
                "status": "created",
                "email": "newuser@example.com",
                "created_at": "2024-06-23T12:00:00Z"
            }
            status_code = 201
        else:
            body = {
                "status": "created",
                "id": f"res_{self.call_count:06d}",
                "message": "Resource created successfully"
            }
            status_code = 201

        return MockResponse(
            status_code=status_code,
            headers={
                "content-type": "application/json",
                "location": f"/v1/resources/{self.call_count}",
                "x-request-id": f"req_{self.call_count:06d}"
            },
            body=json.dumps(body, indent=2).encode('utf-8'),
            duration_ms=duration_ms
        )

    def _mock_put_response(self, request: Request, duration_ms: float) -> MockResponse:
        """Mock PUT response"""
        body = {
            "id": f"res_{self.call_count:06d}",
            "status": "updated",
            "updated_at": "2024-06-23T12:00:00Z",
            "message": "Resource updated successfully"
        }

        return MockResponse(
            status_code=200,
            headers={
                "content-type": "application/json",
                "x-request-id": f"req_{self.call_count:06d}"
            },
            body=json.dumps(body, indent=2).encode('utf-8'),
            duration_ms=duration_ms
        )

    def _mock_delete_response(self, request: Request, duration_ms: float) -> MockResponse:
        """Mock DELETE response"""
        body = {
            "status": "deleted",
            "id": f"res_{self.call_count:06d}",
            "message": "Resource deleted successfully",
            "deleted_at": "2024-06-23T12:00:00Z"
        }

        return MockResponse(
            status_code=200,
            headers={
                "content-type": "application/json",
                "x-request-id": f"req_{self.call_count:06d}"
            },
            body=json.dumps(body, indent=2).encode('utf-8'),
            duration_ms=duration_ms
        )

    def _mock_generic_response(self, request: Request, duration_ms: float) -> MockResponse:
        """Generic mock response"""
        body = {
            "status": "success",
            "message": f"{request.method} request completed",
            "request_id": f"req_{self.call_count:06d}"
        }

        return MockResponse(
            status_code=200,
            headers={
                "content-type": "application/json",
                "x-request-id": f"req_{self.call_count:06d}"
            },
            body=json.dumps(body, indent=2).encode('utf-8'),
            duration_ms=duration_ms
        )
