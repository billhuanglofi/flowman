# Import and export subset guide

Stage 1 supports a practical subset of Postman and Insomnia. It does not claim lossless compatibility with every collection feature.

## Supported import subset

### Postman

Supported:

- Folder structure
- Request name
- Method
- URL
- Query params
- Headers
- Raw body
- JSON body
- Form URL encoded body
- Basic, bearer, and API key auth represented as env references

Unsupported, reported as warnings:

- Pre request scripts
- Test scripts
- Collection level JavaScript variables
- Cookie jar behavior
- Client certificates
- Multipart file uploads
- GraphQL special schema handling
- Any unmapped field outside the supported request subset

Command shape:

```bash
go run ./cmd/flowman import postman path/to/collection.json \
  --out .flowman/imports/postman \
  --report .flowman/generated/postman-import-report.json
```

Strict mode:

```bash
go run ./cmd/flowman import postman path/to/collection.json \
  --out .flowman/imports/postman \
  --report .flowman/generated/postman-import-report.json \
  --strict
```

### Insomnia

Supported:

- Request groups
- Request name
- Method
- URL
- Query params
- Headers
- Raw body
- JSON body
- Form URL encoded body
- Env backed sensitive headers

Unsupported, reported as warnings:

- Cookie jars
- Client certificates
- Multipart file uploads
- Scripting behavior outside canonical request data
- Fields that do not map to canonical Flowman YAML

Command shape:

```bash
go run ./cmd/flowman import insomnia path/to/workspace.json \
  --out .flowman/imports/insomnia \
  --report .flowman/generated/insomnia-import-report.json
```

## Supported export subset

### Export to Postman

```bash
go run ./cmd/flowman export postman testdata/flowman \
  --out .flowman/generated/collection.postman_collection.json
```

What exports cleanly:

- Request folders derived from request paths
- Request method, URL or path, query params, headers, and body
- Bearer auth when Flowman stores an env backed `Authorization` header

Warning cases:

- Multi step flows that Postman cannot represent as Flowman flow sequencing
- Flow steps that point at missing request paths

### Export to Insomnia

```bash
go run ./cmd/flowman export insomnia testdata/flowman \
  --out .flowman/generated/collection.insomnia.json
```

What exports cleanly:

- Request groups derived from request paths
- Request method, URL or path, query params, headers, and body
- Bearer auth derived from env backed `Authorization` headers

Warning cases:

- Multi step flow sequencing that Insomnia cannot represent losslessly
- Flow step request references that are missing from the workspace request list

## Warning model

Import and export warnings are written with machine readable paths. Commands also support `--strict`.

- Default mode writes outputs and a warning report
- `--strict` writes the report, returns non zero, and skips canonical writes when warnings are present

That model protects the source of truth while still letting teams inspect partial conversions.

## Redaction and safety

- Supported auth imports become env references, not embedded secret values
- Warning reports should describe unsupported content without copying raw bearer tokens, cookies, or API keys
- Generated outputs belong in `.flowman/generated/`, not in the canonical YAML tree
