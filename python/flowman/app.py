"""
Flowman Textual TUI - Main Application

A fully clickable terminal UI for API testing.
Mouse support, tabs, buttons, and interactive forms.
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
    RichLog,
)
from textual.binding import Binding
from textual.reactive import reactive
from rich.syntax import Syntax
from rich.json import JSON
from rich.table import Table
import json

from flowman.config import load_workspace, Workspace, Request, Environment
from flowman.runner import RequestRunner, Response


class RequestList(Container):
    """Sidebar showing all requests"""

    def __init__(self, workspace: Workspace, **kwargs):
        super().__init__(**kwargs)
        self.workspace = workspace

    def compose(self) -> ComposeResult:
        yield Label("Requests", classes="panel-title")
        items = []
        for req in self.workspace.requests:
            items.append(ListItem(Label(f"{req.method} {req.name}"), id=f"req-{req.name}"))
        yield ListView(*items, id="request-list")


class EnvironmentSelector(Container):
    """Environment selection panel"""

    def __init__(self, workspace: Workspace, current_env: str, **kwargs):
        super().__init__(**kwargs)
        self.workspace = workspace
        self.current_env = current_env

    def compose(self) -> ComposeResult:
        yield Label("Environments", classes="panel-title")
        for env in self.workspace.environments:
            variant = "primary" if env.name == self.current_env else "default"
            yield Button(env.name.upper(), variant=variant, id=f"env-{env.name}")


class RequestDetails(Container):
    """Request details and run button"""

    selected_request = reactive(None)

    def __init__(self, **kwargs):
        super().__init__(**kwargs)

    def compose(self) -> ComposeResult:
        yield Label("Selected Request", classes="panel-title")
        yield Static("Select a request from the list", id="req-name")
        yield Static("", id="req-method")
        yield Static("", id="req-endpoint")
        yield Button("▶ Run Request", variant="success", id="run-btn")

    def update_request(self, request: Request):
        """Update displayed request details"""
        self.query_one("#req-name", Static).update(f"Name: {request.name}")
        self.query_one("#req-method", Static).update(f"Method: {request.method}")
        endpoint = request.endpoint or request.path or request.url
        self.query_one("#req-endpoint", Static).update(f"Endpoint: {endpoint}")


class ResponseViewer(Container):
    """Tabbed response viewer with Body/Headers/Trace"""

    def compose(self) -> ComposeResult:
        yield Label("Response", classes="panel-title")
        with TabbedContent():
            with TabPane("Body", id="tab-body"):
                yield RichLog(highlight=True, markup=True, id="response-body")
            with TabPane("Headers", id="tab-headers"):
                yield RichLog(id="response-headers")
            with TabPane("Trace", id="tab-trace"):
                yield RichLog(id="response-trace")

    def show_response(self, response: Response):
        """Display response data"""
        # Body tab
        body_log = self.query_one("#response-body", RichLog)
        body_log.clear()
        body_log.write(f"Status: {response.status_code}")
        body_log.write(f"Duration: {response.duration_ms:.0f}ms")
        body_log.write(f"Size: {len(response.body)} bytes")
        body_log.write("")

        # Try to parse as JSON
        try:
            data = json.loads(response.body_text)
            body_log.write(JSON(json.dumps(data, indent=2)))
        except:
            body_log.write(response.body_text[:1000])

        # Headers tab
        headers_log = self.query_one("#response-headers", RichLog)
        headers_log.clear()
        table = Table(title="Response Headers")
        table.add_column("Header", style="cyan")
        table.add_column("Value", style="green")
        for name, value in response.headers.items():
            table.add_row(name, value)
        headers_log.write(table)


class HistoryPanel(Container):
    """Recent request history"""

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.history = []

    def compose(self) -> ComposeResult:
        yield Label("History (last 5)", classes="panel-title")
        yield ListView(id="history-list")

    def add_entry(self, request_name: str, status_code: int, duration_ms: float):
        """Add entry to history"""
        status_icon = "✓" if 200 <= status_code < 300 else "✗"
        entry = f"{status_icon} {request_name} [{status_code}] {duration_ms:.0f}ms"
        self.history.insert(0, entry)

        # Keep last 5
        if len(self.history) > 5:
            self.history = self.history[:5]

        # Update list
        list_view = self.query_one("#history-list", ListView)
        list_view.clear()
        for h in self.history:
            list_view.append(ListItem(Label(h)))


class FlowmanApp(App):
    """Flowman - Clickable TUI for API Testing"""

    CSS = """
    Screen {
        layout: grid;
        grid-size: 3 4;
        grid-gutter: 1;
    }

    .panel-title {
        text-style: bold;
        color: $accent;
        margin: 0 0 1 0;
    }

    RequestList {
        column-span: 1;
        row-span: 2;
        border: solid $primary;
        padding: 1;
    }

    EnvironmentSelector {
        column-span: 1;
        row-span: 1;
        border: solid $primary;
        padding: 1;
    }

    RequestDetails {
        column-span: 1;
        row-span: 1;
        border: solid $primary;
        padding: 1;
    }

    ResponseViewer {
        column-span: 2;
        row-span: 3;
        border: solid $accent;
        padding: 1;
    }

    HistoryPanel {
        column-span: 2;
        row-span: 1;
        border: solid $primary;
        padding: 1;
    }

    Button {
        width: 100%;
        margin: 1 0;
    }

    ListView {
        height: auto;
    }

    RichLog {
        height: 100%;
    }
    """

    TITLE = "Flowman - API Testing TUI"

    BINDINGS = [
        Binding("q", "quit", "Quit"),
        Binding("r", "run_request", "Run"),
        Binding("e", "switch_env", "Switch Env"),
        Binding("ctrl+r", "replay", "Replay"),
        ("?", "help", "Help"),
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

        yield Header()
        yield RequestList(self.workspace)
        yield EnvironmentSelector(self.workspace, self.env_name)
        yield RequestDetails()
        yield ResponseViewer()
        yield HistoryPanel()
        yield Footer()

    def on_mount(self) -> None:
        """Called when app starts."""
        self.sub_title = f"Environment: {self.env_name} | Click anywhere or use shortcuts!"

        # Select first request
        if self.workspace and self.workspace.requests:
            self.selected_request = self.workspace.requests[0]
            details = self.query_one(RequestDetails)
            details.update_request(self.selected_request)

    def on_list_view_selected(self, event: ListView.Selected) -> None:
        """Handle request selection from list"""
        # Find request by index
        if self.workspace:
            try:
                index = event.list_view.index
                self.selected_request = self.workspace.requests[index]
                details = self.query_one(RequestDetails)
                details.update_request(self.selected_request)
                self.notify(f"Selected: {self.selected_request.name}", severity="information")
            except:
                pass

    def on_button_pressed(self, event: Button.Pressed) -> None:
        """Handle button clicks."""
        button_id = event.button.id

        if button_id == "run-btn":
            self.action_run_request()
        elif button_id and button_id.startswith("env-"):
            env_name = button_id.replace("env-", "")
            self.env_name = env_name
            self.sub_title = f"Environment: {env_name} | Click anywhere!"
            self.notify(f"Switched to {env_name.upper()} environment", severity="information")

    def action_run_request(self) -> None:
        """Run the selected request."""
        if not self.selected_request:
            self.notify("No request selected", severity="warning")
            return

        env = self.workspace.get_environment(self.env_name)
        if not env:
            self.notify(f"Environment {self.env_name} not found", severity="error")
            return

        self.notify(f"Running {self.selected_request.name}...", severity="information")

        try:
            # Run request
            response = self.runner.run(self.selected_request, env)

            # Show response
            viewer = self.query_one(ResponseViewer)
            viewer.show_response(response)

            # Add to history
            history = self.query_one(HistoryPanel)
            history.add_entry(self.selected_request.name, response.status_code, response.duration_ms)

            self.notify(f"✓ Completed [{response.status_code}] in {response.duration_ms:.0f}ms", severity="success")

        except Exception as e:
            self.notify(f"Error: {str(e)}", severity="error")

    def action_switch_env(self) -> None:
        """Switch environment."""
        self.notify("Click an environment button to switch", severity="information")

    def action_replay(self) -> None:
        """Replay last request."""
        if self.selected_request:
            self.notify("Replaying last request...", severity="information")
            self.action_run_request()
        else:
            self.notify("No request to replay", severity="warning")

    def action_help(self) -> None:
        """Show help."""
        self.notify(
            "Keybindings: r=run | Ctrl+R=replay | e=env | q=quit | Click any button!",
            severity="information",
            timeout=5
        )


def run(config_path: str = None, env_name: str = "uat"):
    """Run the Flowman TUI app."""
    app = FlowmanApp(config_path=config_path, env_name=env_name)
    app.run()


if __name__ == "__main__":
    run()
