// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

// Package testutil provides shared helpers for Snowball vocabulary verification and benchmarks.
package testutil

import (
	"bufio"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxBenchmarkPairs   = 50000
	shortModeSampleStep = 100
	shortModeThreshold  = 200000
)

// VocabPair holds an input word and its expected Snowball stem.
type VocabPair struct {
	Input    string
	Expected string
}

func vocabularyPath(tb testing.TB, language string) string {
	tb.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(tb, ok, "resolve testutil source path")
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "testdata", language+".tsv.gz")
}

// LoadVocabularySample loads up to maxPairs entries from the compressed vocabulary file for language.
func LoadVocabularySample(tb testing.TB, language string, maxPairs int) []VocabPair {
	tb.Helper()

	path := vocabularyPath(tb, language)
	file, err := os.Open(path)
	require.NoError(tb, err, "open vocabulary %s", path)
	defer func() {
		assert.NoError(tb, file.Close(), "close vocabulary %s", path)
	}()

	reader, err := gzip.NewReader(file)
	require.NoError(tb, err, "create gzip reader for %s", path)
	defer func() {
		assert.NoError(tb, reader.Close(), "close gzip reader for %s", path)
	}()

	pairs := make([]VocabPair, 0, min(maxPairs, 32768))
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() && len(pairs) < maxPairs {
		input, expected, found := strings.Cut(scanner.Text(), "\t")
		require.True(tb, found, "%s: malformed vocabulary line", path)
		pairs = append(pairs, VocabPair{Input: input, Expected: expected})
	}
	require.NoError(tb, scanner.Err(), "scan vocabulary %s", path)
	require.NotEmpty(tb, pairs, "vocabulary %s is empty", path)
	return pairs
}

// VerifyVocabulary streams the compressed official Snowball vocabulary for language
// and verifies stemFunc against every entry without loading large corpora into RAM.
// Under `go test -short`, corpora larger than 200,000 words are sampled deterministically
// after verifying the first 200,000 entries.
func VerifyVocabulary(t *testing.T, language string, stemFunc func(string) string) {
	t.Helper()

	path := vocabularyPath(t, language)
	file, err := os.Open(path)
	require.NoError(t, err, "open vocabulary %s", path)
	defer func() {
		assert.NoError(t, file.Close(), "close vocabulary %s", path)
	}()

	reader, err := gzip.NewReader(file)
	require.NoError(t, err, "create gzip reader for %s", path)
	defer func() {
		assert.NoError(t, reader.Close(), "close gzip reader for %s", path)
	}()

	const maxLoggedMismatches = 25
	scanner := bufio.NewScanner(reader)
	lineNumber := 0
	checked := 0
	mismatches := 0
	shortMode := testing.Short()

	for scanner.Scan() {
		lineNumber++
		if shortMode && lineNumber > shortModeThreshold && lineNumber%shortModeSampleStep != 0 {
			continue
		}
		input, expected, found := strings.Cut(scanner.Text(), "\t")
		require.True(t, found, "%s:%d: malformed vocabulary line", path, lineNumber)

		checked++
		got := stemFunc(input)
		if got != expected {
			mismatches++
			if mismatches <= maxLoggedMismatches {
				assert.Equal(t, expected, got, "%s:%d: Stem(%q)", language, lineNumber, input)
			}
		}
	}
	require.NoError(t, scanner.Err(), "scan vocabulary %s", path)
	require.Positive(t, checked, "vocabulary %s is empty", path)
	if mismatches > maxLoggedMismatches {
		assert.Zero(t, mismatches, "%s: %d total mismatches out of %d checked words (showing first %d)",
			language, mismatches, checked, maxLoggedMismatches)
	}
}

// BenchmarkVocabulary benchmarks stemFunc across up to 50,000 words of the vocabulary for language,
// reporting bytes/sec throughput, allocations, and ns/word.
func BenchmarkVocabulary(b *testing.B, language string, stemFunc func(string) string) {
	b.Helper()

	pairs := LoadVocabularySample(b, language, maxBenchmarkPairs)
	var totalBytes int64
	for _, pair := range pairs {
		totalBytes += int64(len(pair.Input))
	}

	b.SetBytes(totalBytes)
	b.ReportAllocs()

	for b.Loop() {
		for _, pair := range pairs {
			_ = stemFunc(pair.Input)
		}
	}

	if totalWords := float64(b.N) * float64(len(pairs)); totalWords > 0 {
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/totalWords, "ns/word")
	}
}
