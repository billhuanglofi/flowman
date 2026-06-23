"""
Config loader for Flowman YAML files

Loads workspace, environments, and requests from YAML.
"""

from pathlib import Path
from typing import Dict, List, Optional
import yaml


class Request:
    """Request model"""

    def __init__(self, data: Dict):
        self.version = data.get('version', 'v1')
        self.name = data.get('name', 'Unnamed')
        self.method = data.get('method', 'GET')
        self.url = data.get('url', '')
        self.endpoint = data.get('endpoint', '')
        self.path = data.get('path', '')
        self.headers = data.get('headers', [])
        self.query = data.get('query', [])
        self.body = data.get('body', )
        self.trace = data.get('trace', {})

    def __repr__(self):
        return f"<Request {self.method} {self.name}>"


class Environment:
    """Environment model"""

    def __init__(self, data: Dict):
        self.version = data.get('version', 'v1')
        self.name = data.get('name', 'default')
        self.base_url = data.get('base_url', '')
        self.headers = data.get('headers', [])
        self.oracle = data.get('oracle', {})
        self.trace = data.get('trace', {})

    def __repr__(self):
        return f"<Environment {self.name} @ {self.base_url}>"


class Workspace:
    """Workspace model"""

    def __init__(self, root_path: Path):
        self.root = root_path
        self.project = self._load_project()
        self.environments = self._load_environments()
        self.requests = self._load_requests()

    def _load_project(self) -> Dict:
        """Load flowman.yaml"""
        project_file = self.root / 'flowman.yaml'
        if not project_file.exists():
            return {}

        with open(project_file) as f:
            return yaml.safe_load(f)

    def _load_environments(self) -> List[Environment]:
        """Load all environment files"""
        envs = []
        env_paths = self.project.get('environments', [])

        for path in env_paths:
            env_file = self.root / path
            if env_file.exists():
                with open(env_file) as f:
                    data = yaml.safe_load(f)
                    envs.append(Environment(data))

        return envs

    def _load_requests(self) -> List[Request]:
        """Load all request files"""
        requests = []
        request_paths = self.project.get('requests', [])

        for path in request_paths:
            req_file = self.root / path
            if req_file.exists():
                with open(req_file) as f:
                    data = yaml.safe_load(f)
                    requests.append(Request(data))

        return requests

    def get_environment(self, name: str) -> Optional[Environment]:
        """Get environment by name"""
        for env in self.environments:
            if env.name == name:
                return env
        return None

    def __repr__(self):
        return f"<Workspace {len(self.environments)} envs, {len(self.requests)} requests>"


def load_workspace(path: str) -> Workspace:
    """Load workspace from path"""
    root = Path(path).resolve()
    if root.is_file():
        root = root.parent
    return Workspace(root)
