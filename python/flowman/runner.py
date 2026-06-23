"""
HTTP Request Runner

Executes HTTP requests and captures responses.
"""

import httpx
import time
from typing import Dict, List, Optional, Tuple
from flowman.config import Request, Environment


class Response:
    """Response model"""

    def __init__(self, status_code: int, headers: Dict, body: bytes, duration_ms: float):
        self.status_code = status_code
        self.headers = headers
        self.body = body
        self.duration_ms = duration_ms

    @property
    def body_text(self) -> str:
        """Decode body as text"""
        return self.body.decode('utf-8', errors='replace')

    def __repr__(self):
        return f"<Response {self.status_code} {len(self.body)} bytes in {self.duration_ms:.0f}ms>"


class RequestRunner:
    """Runs HTTP requests"""

    def __init__(self, timeout: float = 30.0):
        self.timeout = timeout

    def run(self, request: Request, environment: Environment) -> Response:
        """Execute a request against an environment"""
        # Build URL
        url = self._build_url(request, environment)

        # Build headers
        headers = self._build_headers(request, environment)

        # Build query params
        params = self._build_params(request)

        # Build body
        body = self._build_body(request)

        # Execute request
        start = time.time()

        with httpx.Client(timeout=self.timeout) as client:
            response = client.request(
                method=request.method,
                url=url,
                headers=headers,
                params=params,
                content=body,
            )

        duration_ms = (time.time() - start) * 1000

        return Response(
            status_code=response.status_code,
            headers=dict(response.headers),
            body=response.content,
            duration_ms=duration_ms,
        )

    def _build_url(self, request: Request, environment: Environment) -> str:
        """Build full URL from request and environment"""
        if request.url:
            return request.url

        base = environment.base_url.rstrip('/')
        path = request.path or f"/{request.endpoint}"
        return f"{base}{path}"

    def _build_headers(self, request: Request, environment: Environment) -> Dict[str, str]:
        """Merge headers from environment and request"""
        headers = {}

        # Environment headers
        for h in environment.headers:
            name = h.get('name', '')
            value = h.get('value', '')
            if name and value:
                headers[name] = value

        # Request headers
        for h in request.headers:
            name = h.get('name', '')
            value = h.get('value', '')
            if name and value:
                headers[name] = value

        return headers

    def _build_params(self, request: Request) -> Dict[str, str]:
        """Build query parameters"""
        params = {}
        for q in request.query:
            name = q.get('name', '')
            value = q.get('value', '')
            if name:
                params[name] = value
        return params

    def _build_body(self, request: Request) -> Optional[bytes]:
        """Build request body"""
        if not request.body:
            return None

        body = request.body
        mode = body.get('mode', 'raw')

        if mode == 'raw' or mode == 'json':
            raw = body.get('raw', '')
            return raw.encode('utf-8')

        return None
