// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package stems_test

import (
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/stems/pkg/stems"
)

func TestParseLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  stems.Language
	}{
		{input: "english", want: stems.English},
		{input: "EN", want: stems.English},
		{input: " eng ", want: stems.English},
		{input: "french", want: stems.French},
		{input: "Français", want: stems.French},
		{input: "fr", want: stems.French},
		{input: "hungarian", want: stems.Hungarian},
		{input: "magyar", want: stems.Hungarian},
		{input: "hu", want: stems.Hungarian},
		{input: "norwegian", want: stems.Norwegian},
		{input: "nb", want: stems.Norwegian},
		{input: "norsk", want: stems.Norwegian},
		{input: "russian", want: stems.Russian},
		{input: "русский", want: stems.Russian},
		{input: "ru", want: stems.Russian},
		{input: "spanish", want: stems.Spanish},
		{input: "Español", want: stems.Spanish},
		{input: "es", want: stems.Spanish},
		{input: "swedish", want: stems.Swedish},
		{input: "svenska", want: stems.Swedish},
		{input: "sv", want: stems.Swedish},
		{input: "german", want: stems.German},
		{input: "de", want: stems.German},
		{input: "italian", want: stems.Italian},
		{input: "it", want: stems.Italian},
		{input: "portuguese", want: stems.Portuguese},
		{input: "pt", want: stems.Portuguese},
		{input: "dutch", want: stems.Dutch},
		{input: "dutch-porter", want: stems.DutchPorter},
		{input: "finnish", want: stems.Finnish},
		{input: "fi", want: stems.Finnish},
		{input: "turkish", want: stems.Turkish},
		{input: "tr", want: stems.Turkish},
		{input: "ukrainian", want: stems.Ukrainian},
		{input: "uk", want: stems.Ukrainian},
		{input: "arabic", want: stems.Arabic},
		{input: "ar", want: stems.Arabic},
	}

	for _, testCase := range tests {
		got, err := stems.ParseLanguage(testCase.input)
		require.NoError(t, err, "ParseLanguage(%q)", testCase.input)
		assert.Equal(t, testCase.want, got, "ParseLanguage(%q)", testCase.input)
	}

	_, err := stems.ParseLanguage("klingon")
	assert.ErrorIs(t, err, stems.ErrUnsupportedLanguage)
}

func TestLanguagesReturnsSortedCopy(t *testing.T) {
	t.Parallel()

	languages := stems.Languages()
	require.Len(t, languages, 39)
	assert.True(t, slices.IsSorted(languages))

	languages[0] = "mutated"
	assert.NotEqual(t, stems.Language("mutated"), stems.Languages()[0])
}

func TestStemAcrossLanguages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input         string
		language      stems.Language
		skipStopWords bool
		want          string
	}{
		{input: "Accumulations", language: stems.English, want: "accumul"},
		{input: "running", language: stems.Language("en"), want: "run"},
		{input: "because", language: stems.English, skipStopWords: true, want: "because"},
		{input: "above", language: stems.English, skipStopWords: false, want: "abov"},
		{input: "preocuparse", language: stems.Spanish, want: "preocup"},
		{input: "oposición", language: stems.Spanish, want: "oposicion"},
		{input: "continuation", language: stems.French, want: "continu"},
		{input: "большими", language: stems.Russian, want: "больш"},
		{input: "jaktkarlarne", language: stems.Swedish, want: "jaktkarl"},
		{input: "havets", language: stems.Norwegian, want: "hav"},
		{input: "ablakokban", language: stems.Hungarian, want: "ablak"},
		{input: "aaltoon", language: stems.Finnish, want: "aalto"},
		{input: "häusern", language: stems.German, want: "haus"},
		{input: "abbandonata", language: stems.Italian, want: "abbandon"},
		{input: "abastecimento", language: stems.Portuguese, want: "abastec"},
	}

	for _, testCase := range tests {
		var options []stems.Option
		if testCase.skipStopWords {
			options = append(options, stems.WithSkipStopWords())
		}
		got, err := stems.Stem(testCase.input, testCase.language, options...)
		require.NoError(t, err, "Stem(%q, %q)", testCase.input, testCase.language)
		assert.Equal(t, testCase.want, got, "Stem(%q, %q)", testCase.input, testCase.language)
	}

	_, err := stems.Stem("word", stems.Language("unknown"))
	assert.ErrorIs(t, err, stems.ErrUnsupportedLanguage)
}

func TestNewStemmerAndStopWords(t *testing.T) {
	t.Parallel()

	languagesWithStopWords := 0
	for _, language := range stems.Languages() {
		instance, err := stems.New(language, stems.WithSkipStopWords())
		require.NoError(t, err, "New(%q)", language)
		assert.Equal(t, language, instance.Language())

		stopWords := instance.StopWords()
		if len(stopWords) == 0 {
			continue
		}
		languagesWithStopWords++
		sampleStopWord := stopWords[0]
		assert.True(t, instance.IsStopWord(sampleStopWord), "IsStopWord(%q) for %q", sampleStopWord, language)
		assert.Equal(t, sampleStopWord, instance.Stem("  "+sampleStopWord+"  "), "Stem stop word %q for %q", sampleStopWord, language)
	}
	assert.Equal(t, 17, languagesWithStopWords)
}

func TestStemSeq(t *testing.T) {
	t.Parallel()

	instance, err := stems.New(stems.English, stems.WithSkipStopWords())
	require.NoError(t, err)

	inputs := []string{"Running", "because", "accumulations", "happily"}
	got := slices.Collect(stems.StemSeq(instance, slices.Values(inputs)))
	assert.Equal(t, []string{"run", "because", "accumul", "happili"}, got)
}

func TestStemmerConcurrency(t *testing.T) {
	t.Parallel()

	instance, err := stems.New(stems.English)
	require.NoError(t, err)

	var waitGroup sync.WaitGroup
	for range 16 {
		waitGroup.Go(func() {
			for range 200 {
				assert.Equal(t, "accumul", instance.Stem("Accumulations"))
			}
		})
	}
	waitGroup.Wait()
}
