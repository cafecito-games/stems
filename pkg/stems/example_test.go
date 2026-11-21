// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package stems_test

import (
	"fmt"
	"slices"

	"github.com/cafecito-games/stems/pkg/stems"
)

func ExampleStem() {
	stemmed, err := stems.Stem("Accumulations", stems.English)
	if err != nil {
		panic(err)
	}
	fmt.Println(stemmed)
	// Output: accumul
}

func ExampleStem_skipStopWords() {
	stemmed, err := stems.Stem("because", stems.English, stems.WithSkipStopWords())
	if err != nil {
		panic(err)
	}
	fmt.Println(stemmed)
	// Output: because
}

func ExampleStemSeq() {
	stemmer, err := stems.New(stems.English)
	if err != nil {
		panic(err)
	}

	words := []string{"running", "jumps", "easily"}
	stemmed := slices.Collect(stems.StemSeq(stemmer, slices.Values(words)))
	fmt.Println(stemmed)
	// Output: [run jump easili]
}
