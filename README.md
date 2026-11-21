# stems

A modern, pure-Go implementation of **all 39 official [Snowball](https://snowballstem.org/) stemming algorithms** with zero runtime dependencies.

Built for **Go 1.26+**, `stems` includes its own Snowball-to-Go compiler (`cmd/sblgen`), a fast UTF-8 Snowball runtime (`internal/snowball`), goroutine-safe stemmers, typed language selection, `iter.Seq` streaming support, official stop word lists, and a CLI tool — verified against **11,707,009 official Snowball vocabulary word pairs** with zero mismatches.

## Installation

```sh
go get github.com/cafecito-games/stems
```

## Usage

### Stem a Single Word

```go
package main

import (
	"fmt"
	"log"

	"github.com/cafecito-games/stems/pkg/stems"
)

func main() {
	stemmed, err := stems.Stem("Accumulations", stems.English)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(stemmed) // Output: accumul
}
```

### Reusable Stemmer with Stop-Word Skipping and `iter.Seq`

```go
package main

import (
	"fmt"
	"log"
	"slices"

	"github.com/cafecito-games/stems/pkg/stems"
)

func main() {
	language, err := stems.ParseLanguage("es")
	if err != nil {
		log.Fatal(err)
	}

	stemmer, err := stems.New(language, stems.WithSkipStopWords())
	if err != nil {
		log.Fatal(err)
	}

	words := []string{"preocuparse", "porque", "oposición"}
	stemmed := slices.Collect(stems.StemSeq(stemmer, slices.Values(words)))
	fmt.Println(stemmed)
}
```

### Direct Language Subpackages

If you only need a single language, import its generated package directly:

```go
import "github.com/cafecito-games/stems/pkg/stems/english"

stem := english.Stem("running")         // "run"
isStop := english.IsStopWord("because") // true
```

## Supported Languages (39)

| Language | Constant | Package | Stop Words | Vocabulary Verified |
| --- | --- | --- | :---: | ---: |
| Arabic | `stems.Arabic` | `pkg/stems/arabic` | — | 9,196,214 |
| Armenian | `stems.Armenian` | `pkg/stems/armenian` | — | 101,780 |
| Basque | `stems.Basque` | `pkg/stems/basque` | — | 21,386 |
| Catalan | `stems.Catalan` | `pkg/stems/catalan` | — | 48,897 |
| Czech | `stems.Czech` | `pkg/stems/czech` | — | 58,137 |
| Danish | `stems.Danish` | `pkg/stems/danish` | Yes | 23,868 |
| Dutch | `stems.Dutch` | `pkg/stems/dutch` | Yes | 45,670 |
| Dutch (Porter) | `stems.DutchPorter` | `pkg/stems/dutchporter` | — | 45,670 |
| Early Modern English | `stems.EarlyModernEnglish` | `pkg/stems/earlymodernenglish` | — | 36,534 |
| English (Porter2) | `stems.English` | `pkg/stems/english` | Yes | 42,649 |
| Esperanto | `stems.Esperanto` | `pkg/stems/esperanto` | — | 84,456 |
| Estonian | `stems.Estonian` | `pkg/stems/estonian` | — | 57,537 |
| Finnish | `stems.Finnish` | `pkg/stems/finnish` | Yes | 84,399 |
| French | `stems.French` | `pkg/stems/french` | Yes | 21,653 |
| German | `stems.German` | `pkg/stems/german` | Yes | 35,053 |
| Greek | `stems.Greek` | `pkg/stems/greek` | — | 90,727 |
| Hindi | `stems.Hindi` | `pkg/stems/hindi` | — | 65,118 |
| Hungarian | `stems.Hungarian` | `pkg/stems/hungarian` | Yes | 29,881 |
| Indonesian | `stems.Indonesian` | `pkg/stems/indonesian` | Yes | 64,586 |
| Irish | `stems.Irish` | `pkg/stems/irish` | Yes | 186,280 |
| Italian | `stems.Italian` | `pkg/stems/italian` | Yes | 37,426 |
| Lithuanian | `stems.Lithuanian` | `pkg/stems/lithuanian` | — | 86,109 |
| Lovins | `stems.Lovins` | `pkg/stems/lovins` | — | 42,603 |
| Nepali | `stems.Nepali` | `pkg/stems/nepali` | — | 4,000 |
| Norwegian (Bokmål) | `stems.Norwegian` | `pkg/stems/norwegian` | Yes | 20,913 |
| Persian | `stems.Persian` | `pkg/stems/persian` | — | 104,268 |
| Polish | `stems.Polish` | `pkg/stems/polish` | — | 62,576 |
| Porter | `stems.Porter` | `pkg/stems/porter` | — | 42,603 |
| Portuguese | `stems.Portuguese` | `pkg/stems/portuguese` | Yes | 32,016 |
| Romanian | `stems.Romanian` | `pkg/stems/romanian` | — | 87,642 |
| Russian | `stems.Russian` | `pkg/stems/russian` | Yes | 49,785 |
| Serbian | `stems.Serbian` | `pkg/stems/serbian` | — | 29,996 |
| Sesotho | `stems.Sesotho` | `pkg/stems/sesotho` | Yes | 9,848 |
| Spanish | `stems.Spanish` | `pkg/stems/spanish` | Yes | 28,378 |
| Swedish | `stems.Swedish` | `pkg/stems/swedish` | Yes | 30,738 |
| Tamil | `stems.Tamil` | `pkg/stems/tamil` | — | 443,271 |
| Turkish | `stems.Turkish` | `pkg/stems/turkish` | — | 96,325 |
| Ukrainian | `stems.Ukrainian` | `pkg/stems/ukrainian` | Yes | 58,021 |
| Yiddish | `stems.Yiddish` | `pkg/stems/yiddish` | — | 99,996 |

## Project Layout

```text
algorithms/         Upstream Snowball .sbl specs and official stop word lists
cmd/
├── sblgen/         Snowball-to-Go code generator invoked via `go generate ./...`
└── stems/          Command-line stemming utility
internal/
├── sbl/            Snowball lexer, parser, AST, and Go 1.26 code emitter
├── snowball/       UTF-8 Snowball runtime environment (cursor, groupings, among matcher)
└── testutil/       Streaming vocabulary verification and benchmark helpers
pkg/
└── stems/          Public Go API, generated registry, and 39 language packages
testdata/           Compressed official Snowball vocabularies (tracked with Git LFS)
```

## CLI Tool

Install or run the included command-line tool to stem newline-delimited words from a file or `stdin`:

```sh
go install github.com/cafecito-games/stems/cmd/stems@latest

# Stem from stdin
printf "Accumulations\nrunning\nbecause\n" | stems -language en -skip-stop-words

# Stem from a file
stems -language spanish -input words.txt

# List all 39 supported languages
stems -list-languages
```

## Code Generation, Linting & Testing

Project workflows are orchestrated via [Taskfile.yml](Taskfile.yml). Compressed test vocabularies in `testdata/*.tsv.gz` are tracked via Git LFS (`.gitattributes`); run `git lfs pull` after cloning if LFS objects were not fetched automatically.

```sh
# List all available tasks
task

# Run full local verification (generate, fmt:check, header:check, vet, lint, test, build)
task check

# Regenerate all 39 language packages and registry from algorithms/*.sbl
task generate

# Format Go files and auto-fix MIT copyright headers (.go-header.yml)
task header:fix
task fmt

# Run full verification across all 11,707,009 official vocabulary pairs
task test:full

# Run comparative benchmarks across all 39 languages and API modes
task bench

# Run per-package corpus benchmarks across all 39 language packages
task bench:all

# Run fuzz tests
go test -fuzz=FuzzStem -fuzztime=10s ./pkg/stems/english
```

## License

`stems` is released under the [MIT License](LICENSE). Upstream Snowball algorithm sources in `algorithms/` are BSD 3-Clause, and test vocabularies in `testdata/` carry their respective attributions in [testdata/LICENSE](testdata/LICENSE).
