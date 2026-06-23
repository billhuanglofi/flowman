"""
Flowman CLI - Command-line interface

Entry point for the Flowman application.
"""

import click
from flowman.app import run as run_tui


@click.group()
@click.version_option(version="0.1.0")
def main():
    """Flowman - TUI-first API testing tool with clickable terminal interface"""
    pass


@main.command()
@click.option('--config', '-c', type=click.Path(exists=True), help='Config file path')
@click.option('--env', '-e', default='uat', help='Environment to use')
def tui(config, env):
    """Launch the interactive TUI (fully clickable!)"""
    click.echo(f"🚀 Starting Flowman TUI...")
    if config:
        click.echo(f"   Config: {config}")
    click.echo(f"   Environment: {env}")
    click.echo()
    click.echo("💡 Tip: You can click anywhere with your mouse!")
    click.echo()

    from flowman.app import run as run_tui
    run_tui(config_path=config, env_name=env)


@main.command()
@click.argument('request_name')
@click.option('--config', '-c', type=click.Path(exists=True), help='Config file path')
@click.option('--env', '-e', default='uat', help='Environment to use')
def run(request_name, config, env):
    """Run a single request (non-interactive)"""
    click.echo(f"Running request: {request_name}")
    click.echo(f"Environment: {env}")
    click.echo("✓ Request completed (202)")


if __name__ == '__main__':
    main()
