# Oracle usage

Flowman keeps Oracle support optional. Default CI and local test runs do not require Oracle libraries, Oracle credentials, or an Oracle server.

## Default path

These commands work without Oracle tags:

```bash
go test ./...
go run ./cmd/flowman config validate --config testdata/flowman/flowman.yaml --env uat
```

Tracing or Safestore import with the default build returns an actionable message:

```text
oracle support not enabled; rebuild with -tags oracle or run integration with -tags oracle_integration
```

## Build with Oracle support

Build or run with the `oracle` tag when you want the real adapter compiled in:

```bash
go test -tags=oracle ./internal/oracle/...
go run -tags=oracle ./cmd/flowman trace --tx TX-123 --env uat --config testdata/flowman/flowman.yaml
go run -tags=oracle ./cmd/flowman import safestore --tx TX-123 --env uat --endpoint payments --path /v1/payments/replay --out requests/replay-TX-123.request.yaml --non-interactive --config testdata/flowman/flowman.yaml
```

## Integration test path

The integration tag is for real database verification and clean skipping behavior.

Required env vars:

- `FLOWMAN_ORACLE_DSN`
- `FLOWMAN_ORACLE_USER`
- `FLOWMAN_ORACLE_PASSWORD`

Run the integration test package:

```bash
FLOWMAN_ORACLE_DSN=example \
FLOWMAN_ORACLE_USER=example \
FLOWMAN_ORACLE_PASSWORD=example \
go test -tags=oracle_integration ./internal/oracle/...
```

Without those env vars, the tagged integration tests skip instead of failing.

## Environment file wiring

Flowman reads Oracle env var names from the selected environment file:

```yaml
oracle:
  dsn_env: FLOWMAN_UAT_ORACLE_DSN
  user_env: FLOWMAN_UAT_ORACLE_USER
  password_env: FLOWMAN_UAT_ORACLE_PASSWORD
```

That keeps canonical YAML safe for Git while letting each runtime supply secrets through the environment.

## Query behavior

Stage 1 Oracle support uses bound transaction ID parameters.

PROCESS_STATE expectations:

- query by `transaction_id = :transaction_id`
- order by `timestamp, step`
- map rows into a typed journey view

Safestore replay expectations:

- query by `transaction_id = :transaction_id`
- constrain `direction = 'Request'`
- parse header JSON and request body for replay reconstruction

Flowman does not interpolate transaction IDs into SQL strings.

## Secret handling

- Do not put Oracle usernames or passwords in YAML
- Do not paste DSNs, tokens, or cookies into docs, evidence, or examples
- Use env vars locally and in CI systems that run tagged Oracle jobs outside the default workflow
