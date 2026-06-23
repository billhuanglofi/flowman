# Flowman Examples

This tree is a safe local demo workspace for Flowman. It uses only `localhost` and `example.test` hosts, and it keeps Oracle credentials as environment variable names only.

## Quickstart

Run these from the repo root:

```bash
go run ./cmd/flowman config validate --config examples/flowman.yaml --env uat
go run ./cmd/flowman generate curl examples/requests/payments/create-payment.request.yaml --env uat --out /tmp/flowman-example-curl.sh
go run ./cmd/flowman generate http examples/requests/payments/create-payment.request.yaml --env uat --out /tmp/flowman-example-request.http
go run ./cmd/flowman import postman examples/postman/payment-demo.postman_collection.json --out /tmp/flowman-example-postman --report /tmp/flowman-example-postman-report.json
go run ./cmd/flowman import insomnia examples/insomnia/payment-demo.insomnia.json --out /tmp/flowman-example-insomnia --report /tmp/flowman-example-insomnia-report.json
go run ./cmd/flowman export postman examples --out /tmp/flowman-example-postman.json
go run ./cmd/flowman export insomnia examples --out /tmp/flowman-example-insomnia.json
go run ./cmd/flowman tui --config examples/flowman.yaml --env uat --preview
```

## Safestore replay note

The Safestore replay command reads from Oracle, so it does not use a file fixture as its primary input.

For a no-Oracle demo path, use the existing fake-store E2E flow instead of trying to import a real database row:

```bash
go test ./internal/e2e -run TestStage1HappyPath
```

The sample row shape in `examples/safestore/replay-row.example.json` shows the fields the importer expects when a real Oracle-backed store is available.
