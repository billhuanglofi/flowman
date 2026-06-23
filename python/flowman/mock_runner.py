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
        """Generate realistic process_state journey trace"""
        txn_id = f"TXN{self.call_count:010d}"

        # Generate journey rows based on request type
        rows = []

        if "payment" in request.name.lower():
            rows = self._generate_payment_journey(txn_id, request)
        elif "user" in request.name.lower():
            rows = self._generate_user_journey(txn_id, request)
        elif "order" in request.name.lower():
            rows = self._generate_order_journey(txn_id, request)
        else:
            rows = self._generate_generic_journey(txn_id, request)

        return {
            "transaction_id": txn_id,
            "rows": rows,
            "warnings": [],
            "total_steps": len(rows),
            "duration_ms": duration_ms
        }

    def _generate_payment_journey(self, txn_id: str, request: Request) -> list:
        """Generate payment processing journey"""
        if request.method == "POST":
            return [
                {
                    "step": 1,
                    "service": "api-gateway",
                    "state": "RECEIVED",
                    "outcome": "SUCCESS",
                    "message": "Request received and validated",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.000Z"
                },
                {
                    "step": 2,
                    "service": "auth-service",
                    "state": "AUTHENTICATED",
                    "outcome": "SUCCESS",
                    "message": "API key validated",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.015Z"
                },
                {
                    "step": 3,
                    "service": "payment-validator",
                    "state": "VALIDATED",
                    "outcome": "SUCCESS",
                    "message": "Payment details validated",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.030Z"
                },
                {
                    "step": 4,
                    "service": "fraud-check",
                    "state": "APPROVED",
                    "outcome": "PASS",
                    "message": "Fraud check passed - low risk",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.050Z"
                },
                {
                    "step": 5,
                    "service": "payment-processor",
                    "state": "PROCESSING",
                    "outcome": "PENDING",
                    "message": "Sent to payment gateway",
                    "display_label": "active",
                    "timestamp": "2024-06-23T12:00:00.070Z"
                },
                {
                    "step": 6,
                    "service": "payment-processor",
                    "state": "COMPLETED",
                    "outcome": "SUCCESS",
                    "message": "Payment authorized and captured",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.150Z"
                },
                {
                    "step": 7,
                    "service": "notification-service",
                    "state": "SENT",
                    "outcome": "SUCCESS",
                    "message": "Confirmation email queued",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.160Z"
                }
            ]
        elif request.method == "GET":
            return [
                {
                    "step": 1,
                    "service": "api-gateway",
                    "state": "RECEIVED",
                    "outcome": "SUCCESS",
                    "message": "Lookup request received",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.000Z"
                },
                {
                    "step": 2,
                    "service": "auth-service",
                    "state": "AUTHENTICATED",
                    "outcome": "SUCCESS",
                    "message": "Token validated",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.010Z"
                },
                {
                    "step": 3,
                    "service": "payment-db",
                    "state": "RETRIEVED",
                    "outcome": "SUCCESS",
                    "message": "Payment record found",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.025Z"
                }
            ]
        else:  # DELETE
            return [
                {
                    "step": 1,
                    "service": "api-gateway",
                    "state": "RECEIVED",
                    "outcome": "SUCCESS",
                    "message": "Delete request received",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.000Z"
                },
                {
                    "step": 2,
                    "service": "auth-service",
                    "state": "AUTHENTICATED",
                    "outcome": "SUCCESS",
                    "message": "Admin privileges verified",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.010Z"
                },
                {
                    "step": 3,
                    "service": "payment-service",
                    "state": "CANCELLED",
                    "outcome": "SUCCESS",
                    "message": "Payment cancelled and refunded",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.080Z"
                }
            ]

    def _generate_user_journey(self, txn_id: str, request: Request) -> list:
        """Generate user management journey"""
        if request.method == "GET":
            return [
                {
                    "step": 1,
                    "service": "api-gateway",
                    "state": "RECEIVED",
                    "outcome": "SUCCESS",
                    "message": "User lookup request",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.000Z"
                },
                {
                    "step": 2,
                    "service": "user-db",
                    "state": "RETRIEVED",
                    "outcome": "SUCCESS",
                    "message": "User record found",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.015Z"
                }
            ]
        elif request.method == "PUT":
            return [
                {
                    "step": 1,
                    "service": "api-gateway",
                    "state": "RECEIVED",
                    "outcome": "SUCCESS",
                    "message": "User update request",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.000Z"
                },
                {
                    "step": 2,
                    "service": "auth-service",
                    "state": "AUTHORIZED",
                    "outcome": "SUCCESS",
                    "message": "User authorized to modify profile",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.010Z"
                },
                {
                    "step": 3,
                    "service": "validation-service",
                    "state": "VALIDATED",
                    "outcome": "SUCCESS",
                    "message": "Email format validated",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.020Z"
                },
                {
                    "step": 4,
                    "service": "user-db",
                    "state": "UPDATED",
                    "outcome": "SUCCESS",
                    "message": "User profile updated",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.040Z"
                }
            ]
        else:
            return self._generate_generic_journey(txn_id, request)

    def _generate_order_journey(self, txn_id: str, request: Request) -> list:
        """Generate order processing journey"""
        if request.method == "POST":
            return [
                {
                    "step": 1,
                    "service": "api-gateway",
                    "state": "RECEIVED",
                    "outcome": "SUCCESS",
                    "message": "Order creation request",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.000Z"
                },
                {
                    "step": 2,
                    "service": "inventory-service",
                    "state": "CHECKED",
                    "outcome": "SUCCESS",
                    "message": "All items in stock",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.020Z"
                },
                {
                    "step": 3,
                    "service": "pricing-service",
                    "state": "CALCULATED",
                    "outcome": "SUCCESS",
                    "message": "Total: $249.00 (tax included)",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.035Z"
                },
                {
                    "step": 4,
                    "service": "order-db",
                    "state": "CREATED",
                    "outcome": "SUCCESS",
                    "message": "Order record created",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.050Z"
                }
            ]
        else:  # GET
            return [
                {
                    "step": 1,
                    "service": "api-gateway",
                    "state": "RECEIVED",
                    "outcome": "SUCCESS",
                    "message": "Order query request",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.000Z"
                },
                {
                    "step": 2,
                    "service": "order-db",
                    "state": "RETRIEVED",
                    "outcome": "SUCCESS",
                    "message": "Found 3 orders",
                    "display_label": "terminal",
                    "timestamp": "2024-06-23T12:00:00.020Z"
                }
            ]

    def _generate_generic_journey(self, txn_id: str, request: Request) -> list:
        """Generate generic journey"""
        return [
            {
                "step": 1,
                "service": "api-gateway",
                "state": "RECEIVED",
                "outcome": "SUCCESS",
                "message": f"{request.method} request received",
                "display_label": "terminal",
                "timestamp": "2024-06-23T12:00:00.000Z"
            },
            {
                "step": 2,
                "service": "backend-service",
                "state": "PROCESSED",
                "outcome": "SUCCESS",
                "message": "Request processed successfully",
                "display_label": "terminal",
                "timestamp": "2024-06-23T12:00:00.050Z"
            }
        ]

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
