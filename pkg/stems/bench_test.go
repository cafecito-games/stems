// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package stems_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/stems/internal/testutil"
	"github.com/cafecito-games/stems/pkg/stems"
)

const comparativeSampleSize = 10000

// BenchmarkStemByLanguage measures corpus throughput, allocations, and ns/word
// across every supported Snowball language.
func BenchmarkStemByLanguage(b *testing.B) {
	for _, language := range stems.Languages() {
		instance, err := stems.New(language)
		require.NoError(b, err)

		pairs := testutil.LoadVocabularySample(b, string(language), comparativeSampleSize)
		var totalBytes int64
		for _, pair := range pairs {
			totalBytes += int64(len(pair.Input))
		}

		b.Run(string(language), func(b *testing.B) {
			b.SetBytes(totalBytes)
			b.ReportAllocs()

			for b.Loop() {
				for _, pair := range pairs {
					_ = instance.Stem(pair.Input)
				}
			}

			if totalWords := float64(b.N) * float64(len(pairs)); totalWords > 0 {
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/totalWords, "ns/word")
			}
		})
	}
}

// BenchmarkStemAPI compares top-level stems.Stem dispatch, reusable stems.Stemmer,
// stop-word skipping, and iter.Seq streaming on English words.
func BenchmarkStemAPI(b *testing.B) {
	pairs := testutil.LoadVocabularySample(b, string(stems.English), comparativeSampleSize)
	words := make([]string, len(pairs))
	var totalBytes int64
	for index, pair := range pairs {
		words[index] = pair.Input
		totalBytes += int64(len(pair.Input))
	}

	b.Run("TopLevelDispatch", func(b *testing.B) {
		b.SetBytes(totalBytes)
		b.ReportAllocs()
		for b.Loop() {
			for _, input := range words {
				_, _ = stems.Stem(input, stems.English)
			}
		}
		if totalWords := float64(b.N) * float64(len(words)); totalWords > 0 {
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/totalWords, "ns/word")
		}
	})

	b.Run("ReusableStemmer", func(b *testing.B) {
		instance, err := stems.New(stems.English)
		require.NoError(b, err)

		b.SetBytes(totalBytes)
		b.ReportAllocs()
		for b.Loop() {
			for _, input := range words {
				_ = instance.Stem(input)
			}
		}
		if totalWords := float64(b.N) * float64(len(words)); totalWords > 0 {
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/totalWords, "ns/word")
		}
	})

	b.Run("ReusableStemmerSkipStopWords", func(b *testing.B) {
		instance, err := stems.New(stems.English, stems.WithSkipStopWords())
		require.NoError(b, err)

		b.SetBytes(totalBytes)
		b.ReportAllocs()
		for b.Loop() {
			for _, input := range words {
				_ = instance.Stem(input)
			}
		}
		if totalWords := float64(b.N) * float64(len(words)); totalWords > 0 {
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/totalWords, "ns/word")
		}
	})

	b.Run("StemSeq", func(b *testing.B) {
		instance, err := stems.New(stems.English)
		require.NoError(b, err)

		b.SetBytes(totalBytes)
		b.ReportAllocs()
		for b.Loop() {
			for stemmed := range stems.StemSeq(instance, slices.Values(words)) {
				_ = stemmed
			}
		}
		if totalWords := float64(b.N) * float64(len(words)); totalWords > 0 {
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/totalWords, "ns/word")
		}
	})
}

// BenchmarkStemParallel measures concurrent throughput of a shared Stemmer across goroutines.
func BenchmarkStemParallel(b *testing.B) {
	instance, err := stems.New(stems.English)
	require.NoError(b, err)

	pairs := testutil.LoadVocabularySample(b, string(stems.English), comparativeSampleSize)
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		index := 0
		for pb.Next() {
			_ = instance.Stem(pairs[index%len(pairs)].Input)
			index++
		}
	})
}
