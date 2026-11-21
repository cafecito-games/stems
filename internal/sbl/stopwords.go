// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package sbl

import (
	"slices"
	"strings"
)

// ParseStopWords extracts unique, sorted stop words from a Snowball stop.txt file.
// Comments begin with '|' and extend to the end of each line.
func ParseStopWords(content string, normalizeYo bool) []string {
	seen := make(map[string]struct{})
	for line := range strings.SplitSeq(content, "\n") {
		beforeComment, _, _ := strings.Cut(line, "|")
		for field := range strings.FieldsSeq(beforeComment) {
			word := strings.ToLower(strings.TrimSpace(field))
			if normalizeYo {
				word = strings.ReplaceAll(word, "ё", "е")
			}
			if word != "" {
				seen[word] = struct{}{}
			}
		}
	}

	words := make([]string, 0, len(seen))
	for word := range seen {
		words = append(words, word)
	}
	slices.Sort(words)
	return words
}
