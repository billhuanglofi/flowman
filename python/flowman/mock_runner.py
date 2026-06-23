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

    def __init__(self, status_code: int, headers: Dict, body: bytes, duration_ms: float):
        self.status_code = status_code
        self.headers = headers
        self.body = body
        self.duration_ms = duration_ms

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

        # Generate mock response based on method
        if request.method == "GET":
            return self._mock_get_response(request, duration_ms)
        elif request.method == "POST":
            return self._mock_post_response(request, duration_ms)
        elif request.method == "PUT":
            return self._mock_put_response(request, duration_ms)
        elif request.method == "DELETE":
            return self._mock_delete_response(request, duration_ms)
        else:
            return self._mock_generic_response(request, duration_ms)

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
