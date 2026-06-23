# Stage 1 user guide

Stage 1 turns Flowman into a Git friendly CLI and TUI for request execution, transaction ID extraction, Oracle PROCESS_STATE tracing, Safestore replay import, and supported subset import and export work.

## What stage 1 covers

- Canonical YAML under `flowman.yaml`, `envs/`, `requests/`, and `flows/`
- Request execution from canonical request files
- Transaction ID extraction from `--tx`, response headers, JSON paths, or regex rules
- Oracle PROCESS_STATE tracing behind build tags
- Safestore replay import that reconstructs canonical request YAML
- Deterministic generated cURL and `.http` projections
- Supported subset Postman and Insomnia import and export
- Terminal workflow support through `flowman tui`

## Stage 1 non goals

- No cloud sync
- No automatic commit or push
- No secret vault or keychain storage
- No SQLite or local history database
- No full Newman compatibility
- No Postman JavaScript runtime
- No lossless Postman or Insomnia round trip claims outside the documented subset
- No default Oracle dependency in CI or plain `go test ./...`

## Quickstart with fixtures

Run everything from the repo root:

```bash
go test ./...
go run ./cmd/flowman --help
go run ./cmd/flowman config validate --config testdata/flowman/flowman.yaml --env uat
```

Run a fixture request and print a redacted JSON report:

```bash
go run ./cmd/flowman request run testdata/flowman/requests/payment/create.request.yaml --env uat --config testdata/flowman/flowman.yaml
```

Render deterministic generated projections:

```bash
go run ./cmd/flowman generate curl testdata/flowman/requests/payment/create.request.yaml --env uat --out .flowman/generated/create-payment.sh
go run ./cmd/flowman generate http testdata/flowman/requests/payment/create.request.yaml --env uat --out .flowman/generated/create-payment.http
```

Preview or launch the terminal workflow:

```bash
go run ./cmd/flowman tui --config testdata/flowman/flowman.yaml --env uat --preview
go run ./cmd/flowman tui --config testdata/flowman/flowman.yaml --env uat
```

## Environment profiles

Environment files keep runtime URLs and env var names, not secret values.

`testdata/flowman/envs/uat.yaml`:

```yaml
version: v1
name: uat
base_url: https://uat.api.example.test
endpoints:
  - name: payments
    path: /payments
oracle:
  dsn_env: FLOWMAN_UAT_ORACLE_DSN
  user_env: FLOWMAN_UAT_ORACLE_USER
  password_env: FLOWMAN_UAT_ORACLE_PASSWORD
trace:
  transaction_id:
    header: X-Transaction-ID
    json_paths:
      - $.transaction_id
      - $.data.transactionId
  poll_interval: 2s
  timeout: 30s
```

`testdata/flowman/envs/ppd.yaml` follows the same pattern with `FLOWMAN_PPD_*` env var names and a different base URL.

Validate an environment profile:

```bash
go run ./cmd/flowman config validate --config testdata/flowman/flowman.yaml --env ppd
```

Check that referenced secret env vars exist, without printing values:

```bash
FLOWMAN_UAT_ORACLE_DSN=example \
FLOWMAN_UAT_ORACLE_USER=example \
FLOWMAN_UAT_ORACLE_PASSWORD=example \
go run ./cmd/flowman config validate --config testdata/flowman/flowman.yaml --env uat --check-secrets
```

## Safestore replay flow

Safestore replay import reads request rows with a bound `transaction_id`, parses header JSON, defaults the method to `POST`, and writes canonical YAML. URL resolution works in this order:

1. `--url`
2. `--endpoint` plus optional `--path`
3. environment `base_url` plus `--path`

In non interactive mode, missing URL information fails fast.

Import one replay request:

```bash
go run ./cmd/flowman import safestore \
  --tx TX-123 \
  --env uat \
  --endpoint payments \
  --path /v1/payments/replay \
  --out requests/replay-TX-123.request.yaml \
  --non-interactive \
  --config testdata/flowman/flowman.yaml
```

Choose a row when Safestore returns multiple request rows:

```bash
go run ./cmd/flowman import safestore \
  --tx TX-123 \
  --env uat \
  --url https://uat.api.example.test/payments/v1/payments/replay \
  --select latest \
  --out requests/replay-TX-123.request.yaml \
  --non-interactive \
  --config testdata/flowman/flowman.yaml
```

Important limits:

- Flowman does not write raw sensitive headers by default
- Existing output paths fail unless `--overwrite` is set
- Interactive URL prompting is not implemented in the core importer

## Oracle tracing flow

The default binary compiles without Oracle support. That keeps CI and plain `go test ./...` free from Oracle client requirements.

Default behavior:

```bash
go test ./...
go run ./cmd/flowman trace --tx TX-123 --env uat --config testdata/flowman/flowman.yaml
```

Without Oracle tags, the trace command returns an actionable error that tells you to rebuild with Oracle support.

Oracle enabled usage is documented in [`docs/oracle.md`](oracle.md).

## Import and export subset rules

Flowman supports a documented subset for Postman and Insomnia. Supported fields include folders or request groups, method, URL, query params, headers, raw body, JSON body, form URL encoded body, and common auth shapes represented as env backed references.

Unsupported fields produce warnings with JSON pointer style paths. `--strict` turns those warnings into a non zero command result.

Examples:

```bash
go run ./cmd/flowman import postman testdata/e2e/postman_supported_collection.json --out .flowman/imports/postman --report .flowman/generated/postman-import-report.json
go run ./cmd/flowman import insomnia testdata/e2e/insomnia_supported_workspace.json --out .flowman/imports/insomnia --report .flowman/generated/insomnia-import-report.json
go run ./cmd/flowman export postman testdata/flowman --out .flowman/generated/collection.postman_collection.json
go run ./cmd/flowman export insomnia testdata/flowman --out .flowman/generated/collection.insomnia.json
```

Use strict mode when you want unsupported fields to stop the workflow:

```bash
go run ./cmd/flowman import postman testdata/e2e/postman_unsupported_collection.json --out .flowman/imports/postman --report .flowman/generated/postman-import-report.json --strict
```

See [`docs/import-export.md`](import-export.md) for the supported subset table.

## Full plan and technical details summary

This is the compact technical summary the user asked to keep with the docs.

### Product shape

- Flowman is a Go CLI and TUI rooted at `cmd/flowman`
- Canonical state lives in YAML, not in generated cURL or `.http` files
- Oracle support is isolated behind build tags and interface seams
- Replay and import or export paths write warnings when data falls outside the supported subset

### Execution waves

1. Foundation and schema
2. Trace, replay, preview UI, and projections
3. Import and export subset support, full TUI flow, end to end fixtures, CI, and docs

### Technical decisions

- YAML is the source of truth because it is reviewable and merge friendly
- Generated cURL and `.http` files are deterministic projections with a do not edit marker
- Safestore replay defaults to `POST`, with explicit override support
- Oracle and Safestore SQL uses bound `:transaction_id` parameters
- CI only exercises the default non Oracle path
- Secret values stay in env vars, while YAML stores env var names or redacted references

### Verification expectations

- `go test ./...` passes without Oracle, credentials, or client libraries
- `flowman config validate` proves environment setup uses env var references, not embedded secrets
- Request run, trace, replay import, and projection generation can all be exercised with fixture only workflows
- Postman and Insomnia support is explicitly subset based, warning driven, and strict mode aware
