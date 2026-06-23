"""
Flowman Textual TUI - Main Application (Posting-inspired)

A fully clickable terminal UI for API testing.
Clean layout with proper spacing and organization.
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
    ListView,
    ListItem,
    TextArea,
)
from textual.binding import Binding
from textual.reactive import reactive
from rich.syntax import Syntax
from rich.json import JSON
from rich.table import Table
from rich.panel import Panel
import json

from flowman.config import load_workspace, Workspace, Request, Environment
from flowman.runner import RequestRunner, Response


class AppHeader(Horizontal):
    """Top application header"""

    DEFAULT_CSS = """
    AppHeader {
        height: 1;
        background: $panel;
        padding: 0 2;
        dock: top;
    }

    AppHeader Label {
        padding: 0 2 0 0;
        text-style: bold;
    }
    """

    def compose(self) -> ComposeResult:
        yield Label("[b]Flowman[/] [dim]v0.1.0[/]", id="app-title")
        yield Label("", id="app-env")


class CollectionBrowser(Container):
    """Left sidebar with request list and environment selector"""

    DEFAULT_CSS = """
    CollectionBrowser {
        width: 35;
        border-right: solid $primary;
        background: $panel;
    }

    CollectionBrowser .section-title {
        padding: 1 2;
        background: $boost;
        text-style: bold;
    }

    CollectionBrowser ListView {
        height: 1fr;
        padding: 0 1;
    }

    CollectionBrowser .env-buttons {
        padding: 1;
        height: auto;
        layout: horizontal;
    }

    CollectionBrowser Button {
        min-width: 8;
        height: 1;
        padding: 0 2;
        margin: 0 1 0 0;
    }
    """

    def __init__(self, workspace: Workspace, current_env: str, **kwargs):
        super().__init__(**kwargs)
        self.workspace = workspace
        self.current_env = current_env

    def compose(self) -> ComposeResult:
        yield Label("Environments", classes="section-title")
        with Horizontal(classes="env-buttons"):
            for env in self.workspace.environments:
                variant = "primary" if env.name == self.current_env else "default"
                yield Button(env.name.upper(), variant=variant, id=f"env-{env.name}", classes="env-btn")

        yield Label("Requests", classes="section-title")
        items = []
        for idx, req in enumerate(self.workspace.requests):
            label = f"{req.method} {req.name}"
            items.append(ListItem(Label(label), id=f"req-{idx}"))
        yield ListView(*items, id="request-list")


class RequestDetailsPanel(Vertical):
    """Shows selected request details"""

    DEFAULT_CSS = """
    RequestDetailsPanel {
        height: 8;
        background: $panel;
        padding: 1 2;
        border-bottom: solid $primary;
    }

    RequestDetailsPanel Label {
        padding: 0 0 0 0;
    }

    RequestDetailsPanel Static {
        padding: 0;
        height: 1;
    }

    RequestDetailsPanel Button {
        margin: 1 0 0 0;
        height: 1;
        min-width: 16;
    }
    """

    def compose(self) -> ComposeResult:
        yield Label("[b]Selected Request[/]", id="section-title")
        yield Static("", id="req-name")
        yield Static("", id="req-method")
        yield Static("", id="req-endpoint")
        yield Button("▶ Send Request", variant="success", id="run-btn")

    def update_request(self, request: Request):
        """Update displayed request details"""
        self.query_one("#req-name", Static).update(f"Name: [cyan]{request.name}[/]")
        self.query_one("#req-method", Static).update(f"Method: [yellow]{request.method}[/]")
        endpoint = request.endpoint or request.path or request.url
        self.query_one("#req-endpoint", Static).update(f"Endpoint: [green]{endpoint}[/]")


class ResponseViewer(Vertical):
    """Right side response viewer with tabs"""

    DEFAULT_CSS = """
    ResponseViewer {
        background: $surface;
        border-left: solid $primary;
    }

    ResponseViewer TabbedContent {
        height: 1fr;
    }

    ResponseViewer TextArea {
        height: 1fr;
    }
    """

    def compose(self) -> ComposeResult:
        with TabbedContent():
            with TabPane("Body", id="tab-body"):
                yield TextArea("", id="response-body", read_only=True)
            with TabPane("Headers", id="tab-headers"):
                yield TextArea("", id="response-headers", read_only=True)
            with TabPane("Trace", id="tab-trace"):
                yield TextArea("", id="response-trace", read_only=True)

    def show_response(self, response: Response):
        """Display response data"""
        # Body tab
        body_area = self.query_one("#response-body", TextArea)
        header_text = f"Status: {response.status_code}\n"
        header_text += f"Duration: {response.duration_ms:.0f}ms\n"
        header_text += f"Size: {len(response.body)} bytes\n"
        header_text += "\n"

        # Try to format JSON
        try:
            data = json.loads(response.body_text)
            body_text = json.dumps(data, indent=2)
        except:
            body_text = response.body_text[:5000]  # Truncate if too large

        body_area.text = header_text + body_text

        # Headers tab
        headers_area = self.query_one("#response-headers", TextArea)
        headers_text = f"Status: {response.status_code}\n\n"
        for name, value in response.headers.items():
            headers_text += f"{name}: {value}\n"
        headers_area.text = headers_text


class AppBody(Horizontal):
    """Main body container"""

    DEFAULT_CSS = """
    AppBody {
        height: 1fr;
    }
    """


class FlowmanApp(App):
    """Flowman - Clickable TUI for API Testing"""

    CSS = """
    Screen {
        background: $background;
    }

    .section-title {
        color: $accent;
        text-style: bold;
    }
    """

    TITLE = "Flowman - API Testing TUI"

    BINDINGS = [
        Binding("q", "quit", "Quit"),
        Binding("ctrl+c", "quit", "Quit"),
        Binding("ctrl+r", "send_request", "Send"),
        Binding("e", "switch_env", "Switch Env"),
        Binding("?", "help", "Help"),
    ]

    def __init__(self, config_path: str = None, env_name: str = "uat"):
        super().__init__()
        self.config_path = config_path
        self.env_name = env_name
        self.workspace = None
        self.selected_request = None
        self.runner = RequestRunner()

    def compose(self) -> ComposeResult:
        """Create child widgets for the app."""
        # Load workspace
        if self.config_path:
            self.workspace = load_workspace(self.config_path)
        else:
            # Use example config
            example_path = Path(__file__).parent.parent.parent / "examples"
            if example_path.exists():
                self.workspace = load_workspace(str(example_path))

        if not self.workspace:
            self.notify("No workspace loaded", severity="error")
            return

        yield AppHeader()
        with AppBody():
            yield CollectionBrowser(self.workspace, self.env_name)
            with Vertical():
                yield RequestDetailsPanel()
                yield ResponseViewer()
        yield Footer()

    def on_mount(self) -> None:
        """Called when app starts."""
        # Update header
        env_label = self.query_one("#app-env", Label)
        env_label.update(f"Environment: [cyan]{self.env_name}[/]")

        # Select first request
        if self.workspace and self.workspace.requests:
            self.selected_request = self.workspace.requests[0]
            details = self.query_one(RequestDetailsPanel)
            details.update_request(self.selected_request)

    def on_list_view_selected(self, event: ListView.Selected) -> None:
        """Handle request selection from list"""
        if self.workspace:
            try:
                index = event.list_view.index
                self.selected_request = self.workspace.requests[index]
                details = self.query_one(RequestDetailsPanel)
                details.update_request(self.selected_request)
                self.notify(f"Selected: {self.selected_request.name}", severity="information")
            except:
                pass

    def on_button_pressed(self, event: Button.Pressed) -> None:
        """Handle button clicks."""
        button_id = event.button.id

        if button_id == "run-btn":
            self.action_send_request()
        elif button_id and button_id.startswith("env-"):
            env_name = button_id.replace("env-", "")
            self.switch_environment(env_name)

    def switch_environment(self, env_name: str):
        """Switch to a different environment"""
        self.env_name = env_name
        env_label = self.query_one("#app-env", Label)
        env_label.update(f"Environment: [cyan]{env_name}[/]")

        # Update button styles
        for btn in self.query("Button.env-btn"):
            if btn.id == f"env-{env_name}":
                btn.variant = "primary"
            else:
                btn.variant = "default"

        self.notify(f"Switched to {env_name.upper()}", severity="information")

    def action_send_request(self) -> None:
        """Send the selected request."""
        if not self.selected_request:
            self.notify("No request selected", severity="warning")
            return

        env = self.workspace.get_environment(self.env_name)
        if not env:
            self.notify(f"Environment {self.env_name} not found", severity="error")
            return

        self.notify(f"Sending {self.selected_request.name}...", severity="information")

        try:
            # Run request
            response = self.runner.run(self.selected_request, env)

            # Show response
            viewer = self.query_one(ResponseViewer)
            viewer.show_response(response)

            self.notify(
                f"✓ Completed [{response.status_code}] in {response.duration_ms:.0f}ms",
                severity="success"
            )

        except Exception as e:
            self.notify(f"Error: {str(e)}", severity="error")

    def action_switch_env(self) -> None:
        """Switch environment hint"""
        self.notify("Click an environment button to switch", severity="information")

    def action_help(self) -> None:
        """Show help."""
        self.notify(
            "Keybindings: Ctrl+R=send | e=env | q=quit | Click any button!",
            severity="information",
            timeout=5
        )


def run(config_path: str = None, env_name: str = "uat"):
    """Run the Flowman TUI app."""
    app = FlowmanApp(config_path=config_path, env_name=env_name)
    app.run()


if __name__ == "__main__":
    run()
