# Flowman Terminal Design System

## 1. Atmosphere

Flowman's preview TUI should feel like a warm operations desk: calm, readable, and precise. The terminal palette translates parchment, ivory, warm charcoal, and terracotta into terminal-safe colors without bright tech blues or decorative noise.

## 2. Color Tokens

| Token | Terminal color | Role |
| --- | --- | --- |
| `surface.parchment` | `230` | Primary screen text and quiet highlights |
| `surface.ivory` | `231` | High-emphasis labels on dark panels |
| `surface.sand` | `223` | Secondary headings and separators |
| `text.charcoal` | `236` | Warm dark panel backgrounds |
| `text.muted` | `244` | Secondary metadata |
| `accent.terracotta` | `167` | Selected request, important state, primary accents |
| `accent.olive` | `108` | Healthy journey rows and steady state |
| `semantic.warning` | `179` | Placeholder and pending states |
| `semantic.error` | `131` | Actionable error states |
| `border.warm` | `240` | Panel borders and dividers |

## 3. Typography Tokens

Terminal typography uses weight and case rather than font changes.

| Token | Treatment | Role |
| --- | --- | --- |
| `type.title` | Bold, `surface.ivory` on `accent.terracotta` | Product title |
| `type.panel` | Bold, `surface.sand` | Panel headings |
| `type.body` | Normal, `surface.parchment` | Main values |
| `type.meta` | Normal, `text.muted` | Paths, timings, hints |
| `type.code` | Normal, `surface.sand` | Methods, transaction ids, raw-ish values |

## 4. Spacing Tokens

Use cell-based spacing only.

| Token | Cells | Role |
| --- | --- | --- |
| `space.inline` | 1 | Separators inside a line |
| `space.panel_x` | 1 | Horizontal panel padding |
| `space.panel_y` | 0 | Vertical panel padding for compact previews |
| `space.gap` | 1 blank line | Between major rows |

## 5. Component Rules

- Screen: title row, context row, two-column body, response strip, trace journey panel.
- Panels: rounded borders where supported, `border.warm`, single-cell horizontal padding.
- Request list: selected row uses `accent.terracotta`; unselected rows use `type.body`; methods use `type.code`.
- Response/status area: always present, even before a run; pending text uses `semantic.warning`.
- Trace journey: table-like rows with service, step, state, outcome, and message; failed rows use `semantic.error`, stuck rows use `accent.terracotta`, terminal rows use `accent.olive`.

## 6. Interaction States

- Initial preview state is read-only: no text inputs, so Bubble Tea v2 cursor/IME text input handling is not applicable yet.
- Keyboard hints must be plain text, no emojis.
- Error states must explain the next action, such as checking `--config` or `--env`.

## 7. Guardrails

- No emojis, secret values, cool blue-gray accents, or arbitrary one-off color numbers outside the token table.
- No business logic in the TUI: loading, running, extraction, and tracing stay in existing packages; TUI adapts their results for display.
- YAML strings are untrusted display text. Render them as inert text and never interpret escape sequences or markup from configuration content.
