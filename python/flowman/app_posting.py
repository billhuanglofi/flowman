"""
Flowman Textual TUI - Posting-based Architecture

Clean, professional TUI using Posting's proven component structure.
"""

from pathlib import Path
from typing import Generator, TypeVar
from textual.app import App, ComposeResult
from textual.containers import Container, Horizontal, Vertical
from textual.widgets import (
    Header,
    Footer,
    Button,
    Label,
    Static,
    TabbedContent,
    TabPane,
    Tree,
    TextArea,
)
from textual.widgets.tree import TreeNode
from textual.binding import Binding
from textual.reactive import reactive
import json

from flowman.config import load_workspace, Workspace, Request, Environment
from flowman.runner import RequestRunner, Response

T = TypeVar("T")


class FlowmanTree(Tree[T]):
    """Custom tree with vim-like navigation (from Posting)"""

    DEFAULT_CSS = """
    FlowmanTree {
        scrollbar-size-horizontal: 0;
        & .tree--cursor {
            background: $accent-muted;
            color: $text;
            text-style: bold;
        }
    }
    """

    BINDINGS = [
        Binding("k", "cursor_up", "Up", show=False),
        Binding("j", "cursor_down", "Down", show=False),
        Binding("g", "scroll_home", "Top", show=False),
        Binding("G", "scroll_end", "Bottom", show=False),
        Binding("enter", "select_cursor", "Select", show=False),
    ]


class AppHeader(Horizontal):
    """Top header bar"""

    def compose(self) -> ComposeResult:
        yield Label("FLOWMAN", classes="app-title")
        yield Label("v0.1.0", classes="version")
        yield Label("", id="env-display", classes="env-indicator")
        yield Label("Ready", id="status-display", classes="status")


class CollectionBrowser(Container):
    """Left sidebar with environments and request tree"""

    def __init__(self, workspace: Workspace, current_env: str, **kwargs):
        super().__init__(**kwargs)
        self.workspace = workspace
        self.current_env = current_env

    def compose(self) -> ComposeResult:
        yield Label("ENVIRONMENTS", classes="section-header")

        with Horizontal(classes="env-buttons"):
            for env in self.workspace.environments:
                classes = "active" if env.name == self.current_env else ""
                yield Button(
                    env.name.upper(),
                    id=f"env-{env.name}",
                    classes=classes
                )

        yield Label("REQUESTS", classes="section-header")
        yield CollectionTree(self.workspace)


class CollectionTree(FlowmanTree):
    """Request tree with folder organization"""

    def __init__(self, workspace: Workspace, **kwargs):
        super().__init__("API Collection", id="request-tree", **kwargs)
        self.workspace = workspace
        self.root.expand()
        self._build_tree()

    def _build_tree(self):
        """Build tree structure from requests"""
        folders = {}

        for idx, req in enumerate(self.workspace.requests):
            # Parse folder structure
            parts = req.name.split("/")
            if len(parts) > 1:
                folder = parts[0]
                name = parts[1]
            else:
                folder = "General"
                name = req.name

            if folder not in folders:
                folders[folder] = []
            folders[folder].append((idx, req))

        # Build tree nodes
        for folder_name, requests in sorted(folders.items()):
            folder_node = self.root.add(f"📁 {folder_name}/", expand=True)

            for idx, req in requests:
                # Color-code by method
                method_colors = {
                    "GET": "cyan",
                    "POST": "green",
                    "PUT": "yellow",
                    "DELETE": "red",
                }
                color = method_colors.get(req.method, "white")
                label = f"[{color}]{req.method:6}[/] {req.name.split('/')[-1]}"
                folder_node.add_leaf(label, data=idx)


class RequestEditor(Vertical):
    """Main request editor with tabs"""

    def compose(self) -> ComposeResult:
        with Container(classes="request-info"):
            yield Static("", id="req-name", classes="info-line")
            yield Static("", id="req-method", classes="info-line")
            yield Static("", id="req-endpoint", classes="info-line")
            yield Button("⚡ SEND REQUEST", id="send-btn", classes="send")

        with TabbedContent():
            with TabPane("Body"):
                sample = json.dumps({
                    "amount": 100.00,
                    "currency": "USD",
                    "description": "Test payment"
                }, indent=2)
                yield TextArea(sample, language="json", id="request-body")

            with TabPane("Headers"):
                yield TextArea(
                    "Content-Type: application/json\nAuthorization: Bearer <token>",
                    id="request-headers"
                )

            with TabPane("Query"):
                yield TextArea("", id="request-query")

    def update_request(self, request: Request):
        """Update displayed request"""
        self.query_one("#req-name", Static).update(
            f"[dim]Name:[/] [cyan]{request.name}[/]"
        )

        method_colors = {
            "GET": "cyan",
            "POST": "green",
            "PUT": "yellow",
            "DELETE": "red",
        }
        color = method_colors.get(request.method, "white")

        self.query_one("#req-method", Static).update(
            f"[dim]Method:[/] [{color}]{request.method}[/]"
        )

        endpoint = request.endpoint or request.path or request.url
        self.query_one("#req-endpoint", Static).update(
            f"[dim]Endpoint:[/] [cyan]{endpoint}[/]"
        )


class ResponseArea(Vertical):
    """Response viewer with status and tabs"""

    def compose(self) -> ComposeResult:
        with Container(classes="response-status"):
            yield Static("", id="response-status", classes="status-line")
            yield Static("", id="response-meta", classes="meta-line")

        with TabbedContent():
            with TabPane("Body"):
                yield TextArea("", language="json", read_only=True, id="response-body")

            with TabPane("Headers"):
                yield TextArea("", read_only=True, id="response-headers")

            with TabPane("Trace"):
                yield TextArea("", read_only=True, id="response-trace")

    def show_response(self, response: Response):
        """Display response"""
        # Status
        status_class = "status-success" if 200 <= response.status_code < 300 else "status-error"
        self.query_one("#response-status", Static).update(
            f"[{status_class}]● {response.status_code}[/] [dim]OK[/]"
        )

        self.query_one("#response-meta", Static).update(
            f"[status-time]{response.duration_ms:.0f}ms[/] "
            f"[status-size]• {len(response.body)} bytes • JSON[/]"
        )

        # Body
        try:
            data = json.loads(response.body_text)
            body_text = json.dumps(data, indent=2)
        except:
            body_text = response.body_text[:5000]

        self.query_one("#response-body", TextArea).text = body_text

        # Headers
        headers_text = "\n".join(f"{k}: {v}" for k, v in response.headers.items())
        self.query_one("#response-headers", TextArea).text = headers_text


class FlowmanApp(App):
    """Flowman - Posting-based TUI"""

    CSS_PATH = Path(__file__).parent / "flowman.scss"

    TITLE = "Flowman"

    BINDINGS = [
        Binding("ctrl+q", "quit", "Quit"),
        Binding("ctrl+j", "send_request", "Send"),
        Binding("f1", "help", "Help"),
    ]

    def __init__(self, config_path: str = None, env_name: str = "uat", mock_mode: bool = False):
        super().__init__()
        self.config_path = config_path
        self.env_name = env_name
        self.mock_mode = mock_mode
        self.workspace = None
        self.selected_request = None

        # Use mock or real runner
        if mock_mode:
            from flowman.mock_runner import MockRunner
            self.runner = MockRunner()
        else:
            self.runner = RequestRunner()

    def compose(self) -> ComposeResult:
        """Create child widgets"""
        # Load workspace or create sample
        if self.config_path:
            self.workspace = load_workspace(self.config_path)
        else:
            self.workspace = self._create_sample_workspace()

        yield AppHeader()

        with Container(id="main-container"):
            yield CollectionBrowser(self.workspace, self.env_name)

            with Vertical(id="content-area"):
                yield RequestEditor(id="request-panel")
                yield ResponseArea(id="response-panel")

        yield Footer()

    def _create_sample_workspace(self):
        """Create sample data"""
        class MockEnv:
            def __init__(self, name, url):
                self.name = name
                self.base_url = url
                self.headers = []
                self.trace = {}

        class MockReq:
            def __init__(self, name, method, endpoint):
                self.name = name
                self.method = method
                self.endpoint = endpoint
                self.path = endpoint
                self.url = ""
                self.headers = []
                self.query = []
                self.body = {}

        class MockWorkspace:
            def __init__(self):
                self.environments = [
                    MockEnv("local", "http://localhost:3000"),
                    MockEnv("uat", "https://uat.api.example.com"),
                    MockEnv("production", "https://api.example.com"),
                ]
                self.requests = [
                    MockReq("payments/create payment", "POST", "/v1/payments"),
                    MockReq("payments/get payment", "GET", "/v1/payments/{id}"),
                    MockReq("payments/delete payment", "DELETE", "/v1/payments/{id}"),
                    MockReq("users/get user", "GET", "/v1/users/{id}"),
                    MockReq("users/list users", "GET", "/v1/users"),
                    MockReq("users/update user", "PUT", "/v1/users/{id}"),
                    MockReq("orders/list orders", "GET", "/v1/orders"),
                    MockReq("orders/create order", "POST", "/v1/orders"),
                ]

            def get_environment(self, name):
                for env in self.environments:
                    if env.name == name:
                        return env
                return None

        return MockWorkspace()

    def on_mount(self):
        """Initialize UI"""
        env_label = f"ENV: {self.env_name.upper()}"
        if self.mock_mode:
            env_label += " 🎭"
        self.query_one("#env-display", Label).update(env_label)

        if self.workspace and self.workspace.requests:
            self.selected_request = self.workspace.requests[0]
            self.query_one(RequestEditor).update_request(self.selected_request)

    def on_tree_node_selected(self, event: Tree.NodeSelected):
        """Handle tree selection"""
        if event.node.data is not None:
            idx = event.node.data
            self.selected_request = self.workspace.requests[idx]
            self.query_one(RequestEditor).update_request(self.selected_request)
            self.notify(f"Selected: {self.selected_request.name}")

    def on_button_pressed(self, event: Button.Pressed):
        """Handle button clicks"""
        if event.button.id == "send-btn":
            self.action_send_request()
        elif event.button.id and event.button.id.startswith("env-"):
            env_name = event.button.id.replace("env-", "")
            self.switch_environment(env_name)

    def switch_environment(self, env_name: str):
        """Switch environment"""
        self.env_name = env_name
        self.query_one("#env-display", Label).update(f"ENV: {env_name.upper()}")

        for btn in self.query("Button"):
            if btn.id == f"env-{env_name}":
                btn.add_class("active")
            else:
                btn.remove_class("active")

        self.notify(f"Switched to {env_name.upper()}")

    def action_send_request(self):
        """Send request"""
        if not self.selected_request:
            self.notify("No request selected", severity="warning")
            return

        env = self.workspace.get_environment(self.env_name)
        if not env:
            self.notify(f"Environment not found", severity="error")
            return

        self.query_one("#status-display", Label).update("Sending...")

        try:
            response = self.runner.run(self.selected_request, env)
            self.query_one(ResponseArea).show_response(response)
            self.query_one("#status-display", Label).update("Ready")
            self.notify(f"✓ {response.status_code} in {response.duration_ms:.0f}ms")
        except Exception as e:
            self.query_one("#status-display", Label).update("Error")
            self.notify(f"Error: {str(e)}", severity="error")

    def action_help(self):
        """Show help"""
        self.notify("Ctrl+J=Send | Ctrl+Q=Quit | j/k=Navigate")


def run(config_path: str = None, env_name: str = "uat", mock_mode: bool = False):
    """Run Flowman TUI"""
    app = FlowmanApp(config_path=config_path, env_name=env_name, mock_mode=mock_mode)
    app.run()


if __name__ == "__main__":
    run()
