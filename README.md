# flowman

Flowman is a Go CLI and TUI for running API requests, extracting transaction IDs, tracing Oracle PROCESS_STATE journeys, replaying Safestore requests, and syncing stage-1 workflow files through canonical YAML.

## Stage 1 status

Stage 1 is focused on a local, Git friendly workflow.

- Canonical YAML is the source of truth
- Generated cURL and `.http` files are deterministic projections
- Oracle support is optional and isolated behind build tags
- Postman and Insomnia support is a documented subset with warnings and strict mode
- Secret values stay in environment variables or redacted outputs

Stage 1 does not include cloud sync, secret vault storage, automatic git push or commit, SQLite history, a full Newman runtime, or lossless Postman and Insomnia compatibility claims.

## Install and build

Prerequisites:

- Go `1.25.0` or newer

Common commands:

```bash
go test ./...
go run ./cmd/flowman --help
go run ./cmd/flowman version
go build ./cmd/flowman
```

The default build and test path does not require Oracle client libraries.

## Quickstart

Use the fixture workspace in `testdata/flowman`.

```bash
go run ./cmd/flowman config validate --config testdata/flowman/flowman.yaml --env uat
go run ./cmd/flowman request run testdata/flowman/requests/payment/create.request.yaml --env uat --config testdata/flowman/flowman.yaml
go run ./cmd/flowman generate curl testdata/flowman/requests/payment/create.request.yaml --env uat --out .flowman/generated/create-payment.sh
go run ./cmd/flowman generate http testdata/flowman/requests/payment/create.request.yaml --env uat --out .flowman/generated/create-payment.http
go run ./cmd/flowman tui --config testdata/flowman/flowman.yaml --env uat --preview
```

## Canonical YAML layout

Flowman keeps these files under version control:

```text
flowman.yaml
envs/
requests/
flows/
```

Use environment files for `uat`, `ppd`, and other profiles. Store env var names such as `FLOWMAN_UAT_ORACLE_PASSWORD`, not the secret values.

Generated cURL and `.http` outputs should live under `.flowman/generated/` and carry the built in do not edit marker.

## Oracle usage

Default CI and local tests run without Oracle support:

```bash
go test ./...
```

Oracle enabled commands use build tags:

```bash
go run -tags=oracle ./cmd/flowman trace --tx TX-123 --env uat --config testdata/flowman/flowman.yaml
go run -tags=oracle ./cmd/flowman import safestore --tx TX-123 --env uat --endpoint payments --path /v1/payments/replay --out requests/replay-TX-123.request.yaml --non-interactive --config testdata/flowman/flowman.yaml
```

For real Oracle integration tests:

```bash
FLOWMAN_ORACLE_DSN=example \
FLOWMAN_ORACLE_USER=example \
FLOWMAN_ORACLE_PASSWORD=example \
go test -tags=oracle_integration ./internal/oracle/...
```

More detail lives in [`docs/oracle.md`](docs/oracle.md).

## Safestore replay and subset import or export

Safestore replay reconstructs canonical request YAML from request rows, with method defaulting to `POST` and URL resolution from `--url`, `--endpoint`, or environment `base_url` plus `--path`.

Subset import and export commands:

```bash
go run ./cmd/flowman import postman path/to/collection.json --out .flowman/imports/postman --report .flowman/generated/postman-import-report.json
go run ./cmd/flowman import insomnia path/to/workspace.json --out .flowman/imports/insomnia --report .flowman/generated/insomnia-import-report.json
go run ./cmd/flowman export postman testdata/flowman --out .flowman/generated/collection.postman_collection.json
go run ./cmd/flowman export insomnia testdata/flowman --out .flowman/generated/collection.insomnia.json
```

Use `--strict` when unsupported fields should fail the command instead of producing warnings.

## Docs

- [`docs/stage1.md`](docs/stage1.md), stage-1 user guide and plan summary
- [`docs/file-format.md`](docs/file-format.md), canonical YAML and generated file rules
- [`docs/import-export.md`](docs/import-export.md), supported subset and strict mode behavior
- [`docs/oracle.md`](docs/oracle.md), build tags, integration tests, and Oracle safety rules
