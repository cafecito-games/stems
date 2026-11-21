// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

// Package snowball provides a UTF-8 runtime environment for generated Snowball stemmers.
package snowball

import (
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	// InlineBufferCapacity is the recommended stack buffer size in bytes for InitWithBuffer.
	InlineBufferCapacity = 128

	maxDenseBitsetSpan = 8192
	maxPrefixSlack     = 64
)

// Among represents a single entry in a Snowball among(...) lookup table.
// Tables for forward matching must be sorted lexicographically by Text.
// Tables for backward matching must be sorted lexicographically by the reversed bytes of Text.
type Among[T any] struct {
	Text           string
	SubstringIndex int
	Result         int
	Condition      func(*T) bool
}

// Grouping represents an immutable set of Unicode runes for fast membership checks.
type Grouping struct {
	minRune  rune
	maxRune  rune
	bits     []uint64
	fallback []rune
}

// NewGrouping builds a Grouping containing every distinct Unicode rune in characters.
func NewGrouping(characters string) Grouping {
	if characters == "" {
		return Grouping{minRune: 1, maxRune: 0}
	}

	unique := make([]rune, 0, utf8.RuneCountInString(characters))
	for _, character := range characters {
		unique = append(unique, character)
	}
	slices.Sort(unique)
	unique = slices.Compact(unique)

	minRune := unique[0]
	maxRune := unique[len(unique)-1]
	span := int(maxRune - minRune)

	if span <= maxDenseBitsetSpan {
		wordCount := span/64 + 1
		bits := make([]uint64, wordCount)
		for _, character := range unique {
			offset := int(character - minRune)
			bits[offset/64] |= 1 << (offset % 64)
		}
		return Grouping{
			minRune: minRune,
			maxRune: maxRune,
			bits:    bits,
		}
	}

	return Grouping{
		minRune:  minRune,
		maxRune:  maxRune,
		fallback: unique,
	}
}

// Contains reports whether character belongs to the grouping.
func (grouping *Grouping) Contains(character rune) bool {
	if character < grouping.minRune || character > grouping.maxRune {
		return false
	}
	if len(grouping.bits) > 0 {
		offset := int(character - grouping.minRune)
		return (grouping.bits[offset/64] & (1 << (offset % 64))) != 0
	}
	_, found := slices.BinarySearch(grouping.fallback, character)
	return found
}

// Env holds the mutable UTF-8 byte buffer, cursor, region limits, and slice bounds
// used during execution of a Snowball stemming program.
type Env struct {
	Current       []byte
	Cursor        int
	Limit         int
	LimitBackward int
	Bra           int
	Ket           int
}

// Normalize trims surrounding whitespace and converts input to lowercase.
func Normalize(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}

// Init resets env to hold input using a freshly allocated buffer.
func (env *Env) Init(input string) {
	env.Current = make([]byte, len(input))
	copy(env.Current, input)
	env.Cursor = 0
	env.Limit = len(input)
	env.LimitBackward = 0
	env.Bra = 0
	env.Ket = len(input)
}

// InitWithBuffer resets env to hold input, reusing backing when len(input) <= cap(backing)
// to avoid heap allocation for typical word lengths.
func (env *Env) InitWithBuffer(input string, backing []byte) {
	if len(input) <= cap(backing) {
		env.Current = backing[:len(input)]
	} else {
		env.Current = make([]byte, len(input))
	}
	copy(env.Current, input)
	env.Cursor = 0
	env.Limit = len(input)
	env.LimitBackward = 0
	env.Bra = 0
	env.Ket = len(input)
}

// String returns the current word up to Limit as a newly allocated UTF-8 string.
func (env *Env) String() string {
	return string(env.Current[:env.Limit])
}

// Result returns the current word up to Limit, avoiding allocation when the result
// is an unchanged prefix of original within maxPrefixSlack bytes.
func (env *Env) Result(original string) string {
	if env.Limit == 0 {
		return ""
	}
	if env.Limit <= len(original) && len(original)-env.Limit <= maxPrefixSlack &&
		string(env.Current[:env.Limit]) == original[:env.Limit] {
		return original[:env.Limit]
	}
	return string(env.Current[:env.Limit])
}

// RuneCount returns the number of UTF-8 runes in the current word up to Limit.
func (env *Env) RuneCount() int {
	return utf8.RuneCount(env.Current[:env.Limit])
}

// Hop advances Cursor forward by count Unicode runes, returning false if count is negative
// or fewer than count runes remain before Limit.
func (env *Env) Hop(count int) bool {
	if count < 0 {
		return false
	}
	position := env.Cursor
	for range count {
		if position >= env.Limit {
			return false
		}
		_, width := utf8.DecodeRune(env.Current[position:env.Limit])
		position += width
	}
	env.Cursor = position
	return true
}

// HopBackward moves Cursor backward by count Unicode runes toward LimitBackward,
// returning false if count is negative or fewer than count runes remain.
func (env *Env) HopBackward(count int) bool {
	if count < 0 {
		return false
	}
	position := env.Cursor
	for range count {
		if position <= env.LimitBackward {
			return false
		}
		_, width := utf8.DecodeLastRune(env.Current[env.LimitBackward:position])
		position -= width
	}
	env.Cursor = position
	return true
}

// InGrouping matches a single rune in grouping at Cursor and advances Cursor forward.
func (env *Env) InGrouping(grouping *Grouping) bool {
	if env.Cursor >= env.Limit {
		return false
	}
	character, width := utf8.DecodeRune(env.Current[env.Cursor:env.Limit])
	if !grouping.Contains(character) {
		return false
	}
	env.Cursor += width
	return true
}

// OutGrouping matches a single rune not in grouping at Cursor and advances Cursor forward.
func (env *Env) OutGrouping(grouping *Grouping) bool {
	if env.Cursor >= env.Limit {
		return false
	}
	character, width := utf8.DecodeRune(env.Current[env.Cursor:env.Limit])
	if grouping.Contains(character) {
		return false
	}
	env.Cursor += width
	return true
}

// InGroupingBackward matches a single rune in grouping immediately before Cursor
// and moves Cursor backward.
func (env *Env) InGroupingBackward(grouping *Grouping) bool {
	if env.Cursor <= env.LimitBackward {
		return false
	}
	character, width := utf8.DecodeLastRune(env.Current[env.LimitBackward:env.Cursor])
	if !grouping.Contains(character) {
		return false
	}
	env.Cursor -= width
	return true
}

// OutGroupingBackward matches a single rune not in grouping immediately before Cursor
// and moves Cursor backward.
func (env *Env) OutGroupingBackward(grouping *Grouping) bool {
	if env.Cursor <= env.LimitBackward {
		return false
	}
	character, width := utf8.DecodeLastRune(env.Current[env.LimitBackward:env.Cursor])
	if grouping.Contains(character) {
		return false
	}
	env.Cursor -= width
	return true
}

// EqS matches target at Cursor and advances Cursor forward by len(target).
func (env *Env) EqS(target string) bool {
	if env.Limit-env.Cursor < len(target) {
		return false
	}
	if string(env.Current[env.Cursor:env.Cursor+len(target)]) != target {
		return false
	}
	env.Cursor += len(target)
	return true
}

// EqSBackward matches target immediately before Cursor and moves Cursor backward by len(target).
func (env *Env) EqSBackward(target string) bool {
	if env.Cursor-env.LimitBackward < len(target) {
		return false
	}
	start := env.Cursor - len(target)
	if string(env.Current[start:env.Cursor]) != target {
		return false
	}
	env.Cursor = start
	return true
}

// ReplaceS replaces env.Current[start:end] with replacement, updates Limit,
// adjusts Cursor if it lies at or past end, and returns the net byte-length change.
func (env *Env) ReplaceS(start, end int, replacement string) int {
	adjustment := len(replacement) - (end - start)
	newLength := len(env.Current) + adjustment

	if adjustment > 0 {
		if newLength > cap(env.Current) {
			grown := make([]byte, newLength, max(newLength, cap(env.Current)*2))
			copy(grown, env.Current[:start])
			copy(grown[start:], replacement)
			copy(grown[start+len(replacement):], env.Current[end:])
			env.Current = grown
		} else {
			env.Current = env.Current[:newLength]
			copy(env.Current[start+len(replacement):], env.Current[end:len(env.Current)-adjustment])
			copy(env.Current[start:], replacement)
		}
	} else {
		copy(env.Current[start:], replacement)
		if adjustment < 0 {
			copy(env.Current[start+len(replacement):], env.Current[end:])
			env.Current = env.Current[:newLength]
		}
	}

	env.Limit += adjustment
	if env.Cursor >= end {
		env.Cursor += adjustment
	} else if env.Cursor > start {
		env.Cursor = start
	}
	return adjustment
}

// SliceFrom replaces the current slice [Bra:Ket] with replacement.
func (env *Env) SliceFrom(replacement string) {
	env.ReplaceS(env.Bra, env.Ket, replacement)
}

// SliceDel deletes the current slice [Bra:Ket].
func (env *Env) SliceDel() {
	env.ReplaceS(env.Bra, env.Ket, "")
}

// SliceTo returns the current slice [Bra:Ket] as a string.
func (env *Env) SliceTo() string {
	return string(env.Current[env.Bra:env.Ket])
}

// Insert inserts text at Cursor according to Snowball insert semantics:
// in forward mode Cursor advances past text; in backward mode Cursor remains before text.
func (env *Env) Insert(text string, backward bool) {
	position := env.Cursor
	adjustment := env.ReplaceS(position, position, text)
	if position <= env.Bra {
		env.Bra += adjustment
	}
	if position <= env.Ket {
		env.Ket += adjustment
	}
	if backward {
		env.Cursor = position
	} else {
		env.Cursor = position + len(text)
	}
}

// Attach inserts text at Cursor according to Snowball attach semantics:
// in forward mode Cursor remains before text; in backward mode Cursor advances past text.
func (env *Env) Attach(text string, backward bool) {
	position := env.Cursor
	adjustment := env.ReplaceS(position, position, text)
	if position <= env.Bra {
		env.Bra += adjustment
	}
	if position <= env.Ket {
		env.Ket += adjustment
	}
	if backward {
		env.Cursor = position + len(text)
	} else {
		env.Cursor = position
	}
}

// FindAmong searches a sorted forward Among table for the longest entry matching at Cursor
// whose optional Condition succeeds. On match, Cursor advances past the matched text and
// the entry's Result is returned. Otherwise Cursor is unchanged and 0 is returned.
func FindAmong[T any](env *Env, entries []Among[T], state *T) int {
	low := 0
	high := len(entries)
	startCursor := env.Cursor
	limit := env.Limit

	commonLow := 0
	commonHigh := 0
	firstKeyInspected := false

	for {
		mid := low + ((high - low) >> 1)
		diff := 0
		common := min(commonLow, commonHigh)
		entry := &entries[mid]

		for index := common; index < len(entry.Text); index++ {
			if startCursor+common == limit {
				diff = -1
				break
			}
			diff = int(env.Current[startCursor+common]) - int(entry.Text[index])
			if diff != 0 {
				break
			}
			common++
		}

		if diff < 0 {
			high = mid
			commonHigh = common
		} else {
			low = mid
			commonLow = common
		}

		if high-low <= 1 {
			if low > 0 || high == low || firstKeyInspected {
				break
			}
			firstKeyInspected = true
		}
	}

	for {
		entry := &entries[low]
		if commonLow >= len(entry.Text) {
			env.Cursor = startCursor + len(entry.Text)
			if entry.Condition == nil {
				return entry.Result
			}
			if entry.Condition(state) {
				env.Cursor = startCursor + len(entry.Text)
				return entry.Result
			}
			env.Cursor = startCursor
		}
		low = entry.SubstringIndex
		if low < 0 {
			return 0
		}
	}
}

// FindAmongBackward searches a sorted backward Among table for the longest entry matching
// immediately before Cursor whose optional Condition succeeds. On match, Cursor moves
// backward past the matched text and the entry's Result is returned.
func FindAmongBackward[T any](env *Env, entries []Among[T], state *T) int {
	low := 0
	high := len(entries)
	startCursor := env.Cursor
	limitBackward := env.LimitBackward

	commonLow := 0
	commonHigh := 0
	firstKeyInspected := false

	for {
		mid := low + ((high - low) >> 1)
		diff := 0
		common := min(commonLow, commonHigh)
		entry := &entries[mid]

		for index := len(entry.Text) - 1 - common; index >= 0; index-- {
			if startCursor-common == limitBackward {
				diff = -1
				break
			}
			diff = int(env.Current[startCursor-1-common]) - int(entry.Text[index])
			if diff != 0 {
				break
			}
			common++
		}

		if diff < 0 {
			high = mid
			commonHigh = common
		} else {
			low = mid
			commonLow = common
		}

		if high-low <= 1 {
			if low > 0 || high == low || firstKeyInspected {
				break
			}
			firstKeyInspected = true
		}
	}

	for {
		entry := &entries[low]
		if commonLow >= len(entry.Text) {
			env.Cursor = startCursor - len(entry.Text)
			if entry.Condition == nil {
				return entry.Result
			}
			if entry.Condition(state) {
				env.Cursor = startCursor - len(entry.Text)
				return entry.Result
			}
			env.Cursor = startCursor
		}
		low = entry.SubstringIndex
		if low < 0 {
			return 0
		}
	}
}
