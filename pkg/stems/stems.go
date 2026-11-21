// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

// Package stems implements all 39 official Snowball stemming algorithms in pure Go
// with zero external dependencies.
//
// Each supported language is also available as a standalone subpackage (for example,
// github.com/cafecito-games/stems/pkg/stems/english) when direct, zero-dispatch access
// is preferred.
package stems

import (
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"

	"github.com/cafecito-games/stems/internal/snowball"
)

// Language identifies a supported Snowball stemming language.
type Language string

// ErrUnsupportedLanguage is returned when a requested language is not supported.
var ErrUnsupportedLanguage = errors.New("stems: unsupported language")

// Stemmer stems words and identifies stop words for a specific language.
// Implementations returned by New are safe for concurrent use by multiple goroutines.
type Stemmer interface {
	Language() Language
	Stem(input string) string
	IsStopWord(input string) bool
	StopWords() []string
}

type config struct {
	skipStopWords bool
}

// Option configures the behavior of New and Stem.
type Option func(*config)

// WithSkipStopWords configures stemming to leave stop words unchanged (after
// trimming whitespace and lowercasing) rather than reducing them.
func WithSkipStopWords() Option {
	return func(settings *config) {
		settings.skipStopWords = true
	}
}

type languageDefinition struct {
	language   Language
	stem       func(string) string
	isStopWord func(string) bool
	stopWords  func() []string
}

// ParseLanguage resolves a language name, native name, or ISO 639-1/639-2 code
// into a canonical Language constant.
func ParseLanguage(name string) (Language, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if language, found := languageAliases[normalized]; found {
		return language, nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnsupportedLanguage, name)
}

// Languages returns a sorted slice of all supported languages.
func Languages() []Language {
	return slices.Clone(supportedLanguages)
}

type stemmer struct {
	definition    languageDefinition
	skipStopWords bool
}

func (instance *stemmer) Language() Language {
	return instance.definition.language
}

func (instance *stemmer) Stem(input string) string {
	if instance.skipStopWords {
		normalized := snowball.Normalize(input)
		if instance.definition.isStopWord(normalized) {
			return normalized
		}
		return instance.definition.stem(normalized)
	}
	return instance.definition.stem(input)
}

func (instance *stemmer) IsStopWord(input string) bool {
	return instance.definition.isStopWord(input)
}

func (instance *stemmer) StopWords() []string {
	return instance.definition.stopWords()
}

func resolveDefinition(language Language) (languageDefinition, error) {
	if definition, found := registry[language]; found {
		return definition, nil
	}
	canonical, err := ParseLanguage(string(language))
	if err != nil {
		return languageDefinition{}, err
	}
	return registry[canonical], nil
}

// New constructs a reusable, goroutine-safe Stemmer for language.
// Both canonical Language constants and aliases accepted by ParseLanguage are supported.
func New(language Language, options ...Option) (Stemmer, error) {
	definition, err := resolveDefinition(language)
	if err != nil {
		return nil, err
	}
	var settings config
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	return &stemmer{
		definition:    definition,
		skipStopWords: settings.skipStopWords,
	}, nil
}

// Stem stems a single word in the specified language.
func Stem(input string, language Language, options ...Option) (string, error) {
	definition, err := resolveDefinition(language)
	if err != nil {
		return "", err
	}
	var settings config
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	if settings.skipStopWords {
		normalized := snowball.Normalize(input)
		if definition.isStopWord(normalized) {
			return normalized, nil
		}
		return definition.stem(normalized), nil
	}
	return definition.stem(input), nil
}

// StemSeq returns an iterator that lazily yields the stemmed form of each word in words.
func StemSeq(instance Stemmer, words iter.Seq[string]) iter.Seq[string] {
	return func(yield func(string) bool) {
		for currentWord := range words {
			if !yield(instance.Stem(currentWord)) {
				return
			}
		}
	}
}
