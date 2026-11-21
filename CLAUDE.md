# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`stems` (module `github.com/cafecito-games/stems`, Go 1.26) is a pure-Go implementation of all 39 official Snowball stemming algorithms. The only non-test dependency is `testify`. Every language package is **generated** — the repository ships its own Snowball-to-Go compiler rather than hand-written stemmers.

## Commands

Workflows run through [Task](https://taskfile.dev/) (`Taskfile.yml`); `task` with no arguments lists them.

```sh
task check           # generate, fmt:check, header:check, mod:check, vet, lint, test, build — run before any PR
task generate        # go generate ./... — recompile all .sbl algorithms into Go
task test            # go test -race -short ./...  (samples corpora over 200k words)
task test:full       # go test -count=1 ./... — all 11,707,009 official vocabulary pairs
task lint            # golangci-lint v2.x
task fmt             # golangci-lint fmt (gofmt + goimports)
task header:fix      # apply/repair the MIT headers required on every .go file
task bench           # comparative benchmarks across languages and API modes (pkg/stems)
task bench:all       # per-package corpus benchmarks (pkg/stems/...)
task build           # bin/stems and bin/sblgen
```

Running a narrower target:

```sh
go test -run TestStemVocabulary ./pkg/stems/english
go test -race -short ./internal/sbl
go test -run='^$' -bench=BenchmarkStem -benchmem ./pkg/stems/spanish
go test -fuzz=FuzzStem -fuzztime=10s ./pkg/stems/english
```

`testdata/*.tsv.gz` is tracked with Git LFS; tests fail on pointer files, so run `git lfs pull` if a fresh clone did not fetch them.

## Architecture

The data flows in one direction: `algorithms/*.sbl` → `cmd/sblgen` → `internal/sbl` → generated code in `pkg/stems/<language>/stem_gen.go` + `pkg/stems/registry_gen.go`, all running against the `internal/snowball` runtime.

- **`algorithms/`** — upstream Snowball `.sbl` specs (BSD 3-Clause) and official `*.stop.txt` lists. This is the single source of truth; never patch generated Go to fix stemming behavior.
- **`internal/sbl/`** — the compiler. `lexer.go`/`token.go` tokenize Snowball, `parser.go`/`ast.go` build the AST (`linkSubstrings` resolves `substring`/`among` pairings), and `compiler.go` is the emitter: it allocates Go identifiers, hoists `among(...)` tables into package-level `[]snowball.Among[stemmer]` vars (forward tables sorted by bytes, backward tables by *reversed* bytes), emits groupings, the stemmer struct, the public API, and each Snowball routine as a Go method with labeled-block control flow. `stopwords.go` parses stop word lists.
- **`internal/snowball/`** — the runtime `Env`: a mutable UTF-8 byte buffer plus cursor, `Bra`/`Ket` slice bounds, and region limits, with Snowball primitives (`Hop`, `EqS`, `InGrouping`, `SliceFrom`, `Insert`, `FindAmong`/`FindAmongBackward`). `Grouping` compiles a character class into a dense bitset when the rune span allows, else a sorted slice with binary search. `InitWithBuffer` lets generated code pass a stack array (`InlineBufferCapacity`) so short words stem without heap allocation, and `Result(original)` returns the original string unchanged when nothing was modified.
- **`pkg/stems/`** — the public API (`stems.go`): `Language` constants, `ParseLanguage` over native names and ISO 639-1/2 aliases, `New`/`Stem`, `WithSkipStopWords`, and `StemSeq` for `iter.Seq`. `registry_gen.go` maps languages to the generated subpackage functions. Stemmers hold no mutable state, so they are goroutine-safe.
- **`pkg/stems/<language>/`** — generated per-language packages exposing `Stem`, `IsStopWord`, `StopWords`, plus a generated test file with vocabulary verification, edge cases, a benchmark, and a fuzz target. `Stem` normalizes (trim + lowercase, Turkish-aware or ёе-normalizing where the catalog says so) and then calls the unexported `stemRaw`, which is what vocabulary tests exercise directly.
- **`internal/testutil/`** — `VerifyVocabulary` streams a gzipped TSV of official input/expected pairs so multi-million-word corpora never land in RAM; under `-short` it checks the first 200k entries then samples every 100th line. `BenchmarkVocabulary` reports bytes/sec, allocations, and ns/word.

## Working in this repository

- Files named `*_gen.go` carry `DO NOT EDIT` and are overwritten by `task generate`. CI runs `task generate` followed by `git diff --exit-code`, so regenerate and commit generated output alongside any compiler change.
- A compiler or runtime change affects all 39 languages at once: validate with `task test:full`, not just one package.
- Adding or updating a language means touching `algorithms/<id>.sbl` (+ optional `.stop.txt`), `testdata/<id>.tsv.gz`, the `catalog` slice in `cmd/sblgen/main.go` (ID, constant, package name, aliases, `TurkishCase`/`NormalizeYo`), then `task generate`. Language-count assertions live in `pkg/stems/stems_test.go` and `cmd/stems/main_test.go`, and the README table needs the new row. See `CONTRIBUTING.md`.
- Every `.go` file must start with the MIT header from `.go-header.yml` (`goheader` lints it; `task header:fix` writes it).
- `golangci-lint` runs `revive`, `gosec`, `gocritic`, `staticcheck`, and `modernize`; `modernize` plus Go 1.26 means current idioms are expected (`b.Loop`, `slices`/`maps`, `min`/`max`, `for range n`).
- `internal/sbl/sbl_test.go` compiles a small inline Snowball program and asserts on the emitted source — the fastest way to check an emitter change before regenerating everything.
