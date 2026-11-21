// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package sbl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAndCompileMinimalProgram(t *testing.T) {
	t.Parallel()

	source := `
stringescapes {}
stringdef a' '{U+00E1}'

externals ( stem )
integers ( p1 )
groupings ( v )
define v 'aeiou{a'}'

backwardmode (
	define R1 as $p1 <= cursor
	define step1 as (
		[substring] R1 among (
			'ing' 'ed' (delete)
			'ies' (<- 'y')
		)
	)
)

define stem as (
	$p1 = limit
	do ( gopast v gopast non-v setmark p1 )
	backwards do step1
)
`
	program, err := Parse(source)
	require.NoError(t, err)

	code, err := Compile(program, CompileOptions{
		PackageName:     "sample",
		LanguageDisplay: "Sample",
		SourceFile:      "algorithms/sample.sbl",
		StopWords:       []string{"and", "the"},
	})
	require.NoError(t, err)

	output := string(code)
	for _, want := range []string{
		"Copyright (c) 2026 Cafecito Games LLC",
		"SPDX-License-Identifier: MIT",
		"package sample",
		"func Stem(input string) string",
		"func IsStopWord(input string) bool",
		"func StopWords() []string",
		"FindAmongBackward",
	} {
		assert.Contains(t, output, want)
	}
}

func TestParseStopWords(t *testing.T) {
	t.Parallel()

	raw := `
 | Comment line
the   | definite article
and   | conjunction
the   | duplicate
`
	words := ParseStopWords(raw, false)
	assert.Equal(t, []string{"and", "the"}, words)
}
