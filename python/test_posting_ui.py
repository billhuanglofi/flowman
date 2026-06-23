#!/usr/bin/env python3
"""
Quick test script for Flowman Posting-based TUI
Run this to verify everything works before launching the full app
"""

import sys
from pathlib import Path

print("🧪 Testing Flowman Posting-based TUI...\n")

# Test 1: Import modules
print("1. Testing imports...")
try:
    from flowman.app_posting import FlowmanApp, run
    from flowman.config import load_workspace
    from flowman.runner import RequestRunner
    print("   ✓ All imports successful")
except Exception as e:
    print(f"   ✗ Import failed: {e}")
    sys.exit(1)

# Test 2: Check SCSS file
print("2. Checking SCSS file...")
try:
    app = FlowmanApp()
    css_path = Path(app.CSS_PATH)
    if not css_path.exists():
        print(f"   ✗ SCSS file not found at: {css_path}")
        sys.exit(1)
    print(f"   ✓ SCSS file found: {css_path}")
except Exception as e:
    print(f"   ✗ CSS check failed: {e}")
    sys.exit(1)

# Test 3: Create sample workspace
print("3. Testing sample workspace creation...")
try:
    app = FlowmanApp()
    workspace = app._create_sample_workspace()
    print(f"   ✓ Created workspace with {len(workspace.environments)} environments")
    print(f"   ✓ Created workspace with {len(workspace.requests)} requests")
except Exception as e:
    print(f"   ✗ Workspace creation failed: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)

# Test 4: Test component initialization
print("4. Testing component initialization...")
try:
    from flowman.app_posting import (
        AppHeader,
        CollectionBrowser,
        CollectionTree,
        RequestEditor,
        ResponseArea,
        FlowmanTree
    )
    print("   ✓ All components available")
except Exception as e:
    print(f"   ✗ Component initialization failed: {e}")
    sys.exit(1)

print("\n✅ All tests passed! Ready to run:")
print("\n   flowman tui --theme posting\n")
print("Keyboard shortcuts:")
print("   j/k       - Navigate tree")
print("   Enter     - Select request")
print("   Ctrl+J    - Send request")
print("   Ctrl+Q    - Quit")
print()
