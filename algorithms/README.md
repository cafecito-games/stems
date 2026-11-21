# Snowball Algorithms

This directory contains the upstream [Snowball](https://github.com/snowballstem/snowball) algorithm definitions (`*.sbl`) and official stop word lists (`*.stop.txt`) used as the single source of truth for code generation.

## Regenerating Go Code

Run the built-in Snowball-to-Go compiler (`cmd/sblgen`) via `go generate`:

```sh
go generate ./...
```

This compiles all `.sbl` algorithms into `pkg/stems/<language>/stem_gen.go`, generates matching verification test suites (`stem_gen_test.go`), and updates `pkg/stems/registry_gen.go`.

## Upstream License

The Snowball algorithm sources are licensed under the BSD 3-Clause License by the Snowball project (`https://github.com/snowballstem/snowball`).
