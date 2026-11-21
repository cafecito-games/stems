# Contributing to stems

Contributions are welcome! Whether you want to add or update a Snowball algorithm, optimize the compiler/runtime, or improve documentation, please follow these guidelines.

## Requirements

- Go 1.26+
- [Task](https://taskfile.dev/) (`task`) for running project workflows
- Git LFS (`git lfs pull`) to fetch the compressed official vocabularies in `testdata/*.tsv.gz`
- `golangci-lint` v2.x (used by `task lint` and `task fmt`)

## Updating or Adding a Snowball Language

All 39 language stemmers are generated from Snowball `.sbl` specifications in `algorithms/` by `cmd/sblgen`.

1. Add or update `algorithms/<language>.sbl` (and `algorithms/<language>.stop.txt` if upstream publishes a stop word list).
2. Add or update the official vocabulary fixture at `testdata/<language>.tsv.gz` (tracked via Git LFS in `.gitattributes`) and update `testdata/LICENSE` if additional attribution applies.
3. For a new language, add its `languageSpec` entry to `catalog` in `cmd/sblgen/main.go`.
4. Run code generation:
   ```sh
   task generate
   ```
5. Update language count assertions in `pkg/stems/stems_test.go` and `cmd/stems/main_test.go`, and update the table in `README.md`.

## License Headers & Formatting

All Go source files must carry the MIT copyright header defined in `.go-header.yml`:

```go
// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT
```

Use the Taskfile helpers to format and apply headers automatically:

```sh
task header:fix
task fmt
```

## Verification Before Opening a Pull Request

```sh
task check
task test:full
```
