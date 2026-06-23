# 🎉 Flowman Python - Clickable TUI Quick Start

## Installation

```bash
cd /Users/bill/Documents/source/flowman/python
pip install -e .
```

## Run It!

```bash
flowman tui --config ../examples/flowman.yaml --env uat
```

## 🖱️ Now Click Away!

**What you can click:**
1. ✅ **"▶ Run Request" button** - Executes the selected request
2. ✅ **Environment buttons (UAT/PPD)** - Switches environments
3. ✅ **Request list items** - Selects different requests
4. ✅ **Tabs (Body/Headers/Trace)** - Switches response views

**It's fully interactive!** Click anywhere you see a button, list item, or tab.

## ⌨️ Keyboard Shortcuts (Optional)

- `r` - Run request
- `Ctrl+R` - Replay last
- `Tab` - Navigate panels
- `?` - Help
- `q` - Quit

## 📸 What It Looks Like

```
╭──────────────────────────────────────────────╮
│ Requests                                     │
│ ▸ POST create payment                        │  ← Click to select
│   GET status                                 │
│   POST refund                                │
╰──────────────────────────────────────────────╯

╭──────────────────────────────────────────────╮
│ Environments                                 │
│ [UAT]  [PPD]                                 │  ← Click to switch
╰──────────────────────────────────────────────╯

╭──────────────────────────────────────────────╮
│ Selected Request                             │
│ Name: create payment                         │
│ Method: POST                                 │
│ Endpoint: /v1/payments                       │
│ [▶ Run Request]                              │  ← Click to run!
╰──────────────────────────────────────────────╯

╭──────────────────────────────────────────────╮
│ Response                                     │
│ [Body] [Headers] [Trace]                     │  ← Click tabs!
│                                              │
│ Status: 202                                  │
│ {                                            │
│   "transaction_id": "TX-123",                │
│   "status": "accepted"                       │
│ }                                            │
╰──────────────────────────────────────────────╯
```

## 🆚 Go vs Python

| Feature | Go (Bubble Tea) | Python (Textual) |
|---------|----------------|------------------|
| **Mouse Clickable** | ❌ | ✅ |
| **Speed** | ⚡ Very fast | 🐍 Fast enough |
| **Install** | Single binary | pip install |
| **Dependencies** | None | Python 3.10+ |

## 🎯 Which Version Should I Use?

**Use Python/Textual if:**
- ✅ You want to **click with your mouse**
- ✅ You want a **modern, interactive UI**
- ✅ You're comfortable with Python
- ✅ You want to easily extend/customize

**Use Go/Bubble Tea if:**
- ✅ You want a **single binary** (no dependencies)
- ✅ You prefer **keyboard-only** workflows
- ✅ You need **maximum performance**
- ✅ You're deploying to servers without Python

## 🔥 Try It Now!

```bash
cd /Users/bill/Documents/source/flowman/python
pip install -e .
flowman tui --config ../examples/flowman.yaml --env uat
```

**Then click the "▶ Run Request" button!** 🖱️✨

---

*Built with Textual - the same framework Posting.sh uses!*
