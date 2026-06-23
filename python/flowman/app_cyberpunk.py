"""
Flowman Textual TUI - Cyberpunk/Neon Theme

A beautiful, polished terminal UI with dark purple/black background,
neon magenta/purple accents, and strong visual hierarchy.
"""

from pathlib import Path
from textual.app import App, ComposeResult
from textual.containers import Container, Horizontal, Vertical, VerticalScroll
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
from textual.widget import Widget
from textual.binding import Binding
from textual.reactive import reactive
from rich.syntax import Syntax
from rich.json import JSON
from rich.table import Table
import json

from flowman.config import load_workspace, Workspace, Request, Environment
from flowman.runner import RequestRunner, Response


# Tokyo Night / Cyberpunk Color Palette
THEME = {
    "background": "#1a1b26",           # Deep navy/purple black
    "panel_bg": "#24283b",             # Slightly lighter panel
    "border": "#565f89",               # Muted purple-gray
    "border_active": "#bb9af7",        # Neon purple
    "text_primary": "#c0caf5",         # Light blue-white
    "text_muted": "#565f89",           # Dimmed gray
    "accent_pink": "#ff007c",          # Hot pink/magenta
    "accent_purple": "#bb9af7",        # Neon purple
    "accent_cyan": "#7dcfff",          # Bright cyan
    "accent_orange": "#ff9e64",        # Neon orange
    "method_get": "#7dcfff",           # Cyan
    "method_post": "#9ece6a",          # Green
    "method_put": "#e0af68",           # Yellow
    "method_delete": "#f7768e",        # Pink-red
    "success": "#9ece6a",              # Green
    "warning": "#e0af68",              # Orange
    "danger": "#f7768e",               # Red
}


class CyberpunkHeader(Horizontal):
    """Polished top header bar with app info and environment"""

    DEFAULT_CSS = f"""
    CyberpunkHeader {{
        height: 1;
        background: {THEME['panel_bg']};
        padding: 0 2;
        dock: top;
    }}

    CyberpunkHeader .app-title {{
        color: {THEME['accent_pink']};
        text-style: bold;
    }}

    CyberpunkHeader .app-version {{
        color: {THEME['text_muted']};
        padding: 0 2;
    }}

    CyberpunkHeader .env-label {{
        color: {THEME['accent_cyan']};
        text-style: bold;
        padding: 0 2;
    }}

    CyberpunkHeader .status {{
        color: {THEME['text_muted']};
        dock: right;
    }}
    """

    def compose(self) -> ComposeResult:
        yield Label("FLOWMAN", classes="app-title")
        yield Label("v0.1.0", classes="app-version")
        yield Label("", id="env-display", classes="env-label")
        yield Label("Ready", classes="status", id="status")


class RequestTree(Container):
    """Left sidebar with tree-style API collection"""

    DEFAULT_CSS = f"""
    RequestTree {{
        width: 40;
        background: {THEME['panel_bg']};
        border-right: solid {THEME['border']};
    }}

    RequestTree .section-header {{
        height: 3;
        background: {THEME['background']};
        padding: 1 2;
        color: {THEME['accent_purple']};
        text-style: bold;
    }}

    RequestTree .env-chips {{
        height: auto;
        padding: 0 2 1 2;
        background: {THEME['panel_bg']};
    }}

    RequestTree Button {{
        height: 1;
        min-width: 8;
        padding: 0 2;
        margin: 0 1 0 0;
        background: {THEME['background']};
        color: {THEME['text_muted']};
        border: none;
    }}

    RequestTree Button:hover {{
        background: {THEME['border']};
        color: {THEME['text_primary']};
    }}

    RequestTree Button.active {{
        background: {THEME['accent_purple']};
        color: {THEME['background']};
        text-style: bold;
    }}

    RequestTree Tree {{
        background: {THEME['panel_bg']};
        padding: 0 1;
    }}

    RequestTree Tree:focus {{
        border: none;
    }}
    """

    def __init__(self, workspace: Workspace, current_env: str, **kwargs):
        super().__init__(**kwargs)
        self.workspace = workspace
        self.current_env = current_env

    def compose(self) -> ComposeResult:
        yield Label("ENVIRONMENTS", classes="section-header")
        with Horizontal(classes="env-chips"):
            for env in self.workspace.environments:
                classes = "env-btn active" if env.name == self.current_env else "env-btn"
                yield Button(env.name.upper(), id=f"env-{env.name}", classes=classes)

        yield Label("REQUESTS", classes="section-header")
        tree = Tree("API Collection", id="request-tree")
        tree.root.expand()

        # Group requests by folder/category
        folders = {}
        for idx, req in enumerate(self.workspace.requests):
            # Extract folder from name or use "General"
            parts = req.name.split("/")
            if len(parts) > 1:
                folder = parts[0]
                name = parts[1]
            else:
                folder = "General"
                name = req.name

            if folder not in folders:
                folders[folder] = []
            folders[folder].append((idx, req.method, name))

        # Build tree
        for folder, requests in sorted(folders.items()):
            folder_node = tree.root.add(f"📁 {folder}/")
            folder_node.expand()
            for idx, method, name in requests:
                # Color-code by method
                if method == "GET":
                    color = THEME['method_get']
                elif method == "POST":
                    color = THEME['method_post']
                elif method == "PUT":
                    color = THEME['method_put']
                elif method == "DELETE":
                    color = THEME['method_delete']
                else:
                    color = THEME['text_primary']

                label = f"[{color}]{method:6}[/] {name}"
                folder_node.add_leaf(label, data=idx)

        yield tree


class RequestPanel(Vertical):
    """Main request editor with tabs"""

    DEFAULT_CSS = f"""
    RequestPanel {{
        background: {THEME['background']};
        border: solid {THEME['border']};
        height: 1fr;
    }}

    RequestPanel .request-info {{
        height: 6;
        background: {THEME['panel_bg']};
        padding: 1 2;
        border-bottom: solid {THEME['border']};
    }}

    RequestPanel .info-line {{
        height: 1;
        color: {THEME['text_primary']};
    }}

    RequestPanel .info-label {{
        color: {THEME['text_muted']};
    }}

    RequestPanel .info-value {{
        color: {THEME['accent_cyan']};
    }}

    RequestPanel Button.send {{
        height: 1;
        min-width: 16;
        background: {THEME['accent_pink']};
        color: {THEME['background']};
        text-style: bold;
        border: none;
        margin: 1 0 0 0;
    }}

    RequestPanel Button.send:hover {{
        background: {THEME['accent_purple']};
    }}

    RequestPanel TabbedContent {{
        height: 1fr;
        background: {THEME['background']};
    }}

    RequestPanel Tabs {{
        background: {THEME['panel_bg']};
    }}

    RequestPanel Tab {{
        color: {THEME['text_muted']};
    }}

    RequestPanel Tab.-active {{
        color: {THEME['accent_pink']};
        text-style: bold;
    }}

    RequestPanel TextArea {{
        background: {THEME['background']};
        height: 1fr;
    }}
    """

    def compose(self) -> ComposeResult:
        with Container(classes="request-info"):
            yield Static("", id="req-name-display", classes="info-line")
            yield Static("", id="req-method-display", classes="info-line")
            yield Static("", id="req-endpoint-display", classes="info-line")
            yield Button("⚡ SEND REQUEST", id="send-btn", classes="send")

        with TabbedContent():
            with TabPane("Body"):
                sample_body = json.dumps({
                    "amount": 100.00,
                    "currency": "USD",
                    "customer_id": "cust_123",
                    "description": "Payment for order #456"
                }, indent=2)
                yield TextArea(sample_body, id="request-body", language="json")
            with TabPane("Headers"):
                yield TextArea("Content-Type: application/json\nAuthorization: Bearer <token>", id="request-headers")
            with TabPane("Query"):
                yield TextArea("", id="request-query")
            with TabPane("Auth"):
                yield TextArea("", id="request-auth")

    def update_request(self, request: Request):
        """Update displayed request details"""
        self.query_one("#req-name-display", Static).update(
            f"[{THEME['text_muted']}]Name:[/] [{THEME['accent_cyan']}]{request.name}[/]"
        )

        # Color-code method
        method_color = {
            "GET": THEME['method_get'],
            "POST": THEME['method_post'],
            "PUT": THEME['method_put'],
            "DELETE": THEME['method_delete'],
        }.get(request.method, THEME['text_primary'])

        self.query_one("#req-method-display", Static).update(
            f"[{THEME['text_muted']}]Method:[/] [{method_color}]{request.method}[/]"
        )

        endpoint = request.endpoint or request.path or request.url
        self.query_one("#req-endpoint-display", Static).update(
            f"[{THEME['text_muted']}]Endpoint:[/] [{THEME['accent_cyan']}]{endpoint}[/]"
        )


class ResponsePanel(Vertical):
    """Response viewer with status and tabs"""

    DEFAULT_CSS = f"""
    ResponsePanel {{
        background: {THEME['background']};
        border: solid {THEME['border']};
        height: 1fr;
    }}

    ResponsePanel .response-status {{
        height: 3;
        background: {THEME['panel_bg']};
        padding: 1 2;
        border-bottom: solid {THEME['border']};
    }}

    ResponsePanel .status-line {{
        height: 1;
    }}

    ResponsePanel .status-code {{
        color: {THEME['success']};
        text-style: bold;
    }}

    ResponsePanel .status-time {{
        color: {THEME['accent_orange']};
    }}

    ResponsePanel .status-size {{
        color: {THEME['text_muted']};
    }}

    ResponsePanel TabbedContent {{
        height: 1fr;
    }}

    ResponsePanel TextArea {{
        background: {THEME['background']};
        height: 1fr;
    }}

    ResponsePanel .empty-state {{
        align: center middle;
        color: {THEME['text_muted']};
        text-style: italic;
    }}
    """

    def compose(self) -> ComposeResult:
        with Container(classes="response-status"):
            yield Static("", id="response-status-line", classes="status-line")
            yield Static("", id="response-meta-line", classes="status-line")

        with TabbedContent():
            with TabPane("Body"):
                yield TextArea("", id="response-body", read_only=True, language="json")
            with TabPane("Headers"):
                yield TextArea("", id="response-headers", read_only=True)
            with TabPane("Cookies"):
                yield TextArea("", id="response-cookies", read_only=True)
            with TabPane("Trace"):
                yield TextArea("", id="response-trace", read_only=True)

    def show_response(self, response: Response):
        """Display response data"""
        # Status line
        status_color = THEME['success'] if 200 <= response.status_code < 300 else THEME['danger']
        self.query_one("#response-status-line", Static).update(
            f"[{status_color}]● {response.status_code}[/] [{THEME['text_primary']}]OK[/]"
        )

        self.query_one("#response-meta-line", Static).update(
            f"[{THEME['accent_orange']}]{response.duration_ms:.0f}ms[/] "
            f"[{THEME['text_muted']}]• {len(response.body)} bytes • JSON[/]"
        )

        # Body
        try:
            data = json.loads(response.body_text)
            body_text = json.dumps(data, indent=2)
        except:
            body_text = response.body_text[:5000]

        self.query_one("#response-body", TextArea).text = body_text

        # Headers
        headers_text = ""
        for name, value in response.headers.items():
            headers_text += f"{name}: {value}\n"
        self.query_one("#response-headers", TextArea).text = headers_text


class CyberpunkFooter(Static):
    """Polished keybinding bar"""

    DEFAULT_CSS = f"""
    CyberpunkFooter {{
        height: 1;
        background: {THEME['panel_bg']};
        color: {THEME['text_muted']};
        dock: bottom;
        padding: 0 2;
    }}

    CyberpunkFooter .key {{
        color: {THEME['accent_pink']};
        text-style: bold;
    }}

    CyberpunkFooter .label {{
        color: {THEME['text_primary']};
    }}
    """

    def render(self) -> str:
        return (
            f"[{THEME['accent_pink']}]^J[/] [{THEME['text_primary']}]Send[/]  "
            f"[{THEME['accent_pink']}]^S[/] [{THEME['text_primary']}]Save[/]  "
            f"[{THEME['accent_pink']}]^N[/] [{THEME['text_primary']}]New[/]  "
            f"[{THEME['accent_pink']}]^O[/] [{THEME['text_primary']}]Jump[/]  "
            f"[{THEME['accent_pink']}]Tab[/] [{THEME['text_primary']}]Switch[/]  "
            f"[{THEME['accent_pink']}]F1[/] [{THEME['text_primary']}]Help[/]  "
            f"[{THEME['accent_pink']}]^Q[/] [{THEME['text_primary']}]Quit[/]"
        )


class FlowmanApp(App):
    """Flowman - Cyberpunk/Neon TUI for API Testing"""

    CSS = f"""
    Screen {{
        background: {THEME['background']};
    }}

    #main-container {{
        layout: horizontal;
        height: 1fr;
    }}

    #editor-area {{
        layout: vertical;
        width: 1fr;
        padding: 1;
    }}
    """

    TITLE = "Flowman - API Testing TUI"

    BINDINGS = [
        Binding("ctrl+q", "quit", "Quit"),
        Binding("ctrl+j", "send_request", "Send"),
        Binding("ctrl+s", "save", "Save"),
        Binding("f1", "help", "Help"),
    ]

    def __init__(self, config_path: str = None, env_name: str = "uat"):
        super().__init__()
        self.config_path = config_path
        self.env_name = env_name
        self.workspace = None
        self.selected_request = None
        self.runner = RequestRunner()

    def compose(self) -> ComposeResult:
        """Create child widgets"""
        # Load workspace or use sample data
        if self.config_path:
            self.workspace = load_workspace(self.config_path)
        else:
            self.workspace = self._create_sample_workspace()

        yield CyberpunkHeader()

        with Container(id="main-container"):
            yield RequestTree(self.workspace, self.env_name)
            with Vertical(id="editor-area"):
                yield RequestPanel()
                yield ResponsePanel()

        yield CyberpunkFooter()

    def _create_sample_workspace(self) -> Workspace:
        """Create sample data for demo"""
        from flowman.config import Workspace, Environment, Request

        class MockWorkspace:
            def __init__(self):
                self.environments = [
                    type('obj', (object,), {'name': 'local', 'base_url': 'http://localhost:3000'})(),
                    type('obj', (object,), {'name': 'uat', 'base_url': 'https://uat.api.example.com'})(),
                    type('obj', (object,), {'name': 'production', 'base_url': 'https://api.example.com'})(),
                ]
                self.requests = [
                    type('obj', (object,), {'name': 'payments/create payment', 'method': 'POST', 'endpoint': '/v1/payments', 'path': '/v1/payments', 'url': '', 'headers': [], 'query': [], 'body': {}})(),
                    type('obj', (object,), {'name': 'payments/get payment', 'method': 'GET', 'endpoint': '/v1/payments/{id}', 'path': '/v1/payments/{id}', 'url': '', 'headers': [], 'query': [], 'body': {}})(),
                    type('obj', (object,), {'name': 'payments/delete payment', 'method': 'DELETE', 'endpoint': '/v1/payments/{id}', 'path': '/v1/payments/{id}', 'url': '', 'headers': [], 'query': [], 'body': {}})(),
                    type('obj', (object,), {'name': 'users/get user', 'method': 'GET', 'endpoint': '/v1/users/{id}', 'path': '/v1/users/{id}', 'url': '', 'headers': [], 'query': [], 'body': {}})(),
                    type('obj', (object,), {'name': 'users/list users', 'method': 'GET', 'endpoint': '/v1/users', 'path': '/v1/users', 'url': '', 'headers': [], 'query': [], 'body': {}})(),
                    type('obj', (object,), {'name': 'users/update user', 'method': 'PUT', 'endpoint': '/v1/users/{id}', 'path': '/v1/users/{id}', 'url': '', 'headers': [], 'query': [], 'body': {}})(),
                    type('obj', (object,), {'name': 'orders/list orders', 'method': 'GET', 'endpoint': '/v1/orders', 'path': '/v1/orders', 'url': '', 'headers': [], 'query': [], 'body': {}})(),
                    type('obj', (object,), {'name': 'orders/create order', 'method': 'POST', 'endpoint': '/v1/orders', 'path': '/v1/orders', 'url': '', 'headers': [], 'query': [], 'body': {}})(),
                ]

            def get_environment(self, name):
                for env in self.environments:
                    if env.name == name:
                        return env
                return None

        return MockWorkspace()

    def on_mount(self) -> None:
        """Called when app starts"""
        env_display = self.query_one("#env-display", Label)
        env_display.update(f"ENV: {self.env_name.upper()}")

        # Select first request
        if self.workspace and self.workspace.requests:
            self.selected_request = self.workspace.requests[0]
            panel = self.query_one(RequestPanel)
            panel.update_request(self.selected_request)

    def on_tree_node_selected(self, event) -> None:
        """Handle tree selection"""
        if event.node.data is not None:
            idx = event.node.data
            self.selected_request = self.workspace.requests[idx]
            panel = self.query_one(RequestPanel)
            panel.update_request(self.selected_request)
            self.notify(f"Selected: {self.selected_request.name}")

    def on_button_pressed(self, event) -> None:
        """Handle button clicks"""
        button_id = event.button.id

        if button_id == "send-btn":
            self.action_send_request()
        elif button_id and button_id.startswith("env-"):
            env_name = button_id.replace("env-", "")
            self.switch_environment(env_name)

    def switch_environment(self, env_name: str):
        """Switch environment"""
        self.env_name = env_name
        env_display = self.query_one("#env-display", Label)
        env_display.update(f"ENV: {env_name.upper()}")

        # Update button styles
        for btn in self.query("Button.env-btn"):
            if btn.id == f"env-{env_name}":
                btn.add_class("active")
            else:
                btn.remove_class("active")

        self.notify(f"Switched to {env_name.upper()}")

    def action_send_request(self) -> None:
        """Send request"""
        if not self.selected_request:
            self.notify("No request selected", severity="warning")
            return

        env = self.workspace.get_environment(self.env_name)
        if not env:
            self.notify(f"Environment {self.env_name} not found", severity="error")
            return

        status = self.query_one("#status", Label)
        status.update("Sending...")

        try:
            response = self.runner.run(self.selected_request, env)
            viewer = self.query_one(ResponsePanel)
            viewer.show_response(response)
            status.update("Ready")
            self.notify(f"✓ {response.status_code} in {response.duration_ms:.0f}ms")
        except Exception as e:
            status.update("Error")
            self.notify(f"Error: {str(e)}", severity="error")

    def action_save(self) -> None:
        self.notify("Save not yet implemented")

    def action_help(self) -> None:
        self.notify("Help: Ctrl+J=Send | Ctrl+S=Save | F1=Help | Ctrl+Q=Quit")


def run(config_path: str = None, env_name: str = "uat"):
    """Run the Flowman TUI app"""
    app = FlowmanApp(config_path=config_path, env_name=env_name)
    app.run()


if __name__ == "__main__":
    run()
