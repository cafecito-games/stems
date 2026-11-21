// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

// Command sblgen compiles Snowball (.sbl) algorithms into Go 1.26 stemmer packages
// and generates the pkg/stems language registry.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cafecito-games/stems/internal/sbl"
)

type languageSpec struct {
	ID          string
	ConstName   string
	PackageName string
	DisplayName string
	Aliases     []string
	TurkishCase bool
	NormalizeYo bool
}

var catalog = []languageSpec{
	{ID: "arabic", ConstName: "Arabic", PackageName: "arabic", DisplayName: "Arabic", Aliases: []string{"arabic", "ar", "ara", "العربية"}},
	{ID: "armenian", ConstName: "Armenian", PackageName: "armenian", DisplayName: "Armenian", Aliases: []string{"armenian", "hy", "hye", "arm", "հայերեն"}},
	{ID: "basque", ConstName: "Basque", PackageName: "basque", DisplayName: "Basque", Aliases: []string{"basque", "eu", "eus", "baq", "euskara"}},
	{ID: "catalan", ConstName: "Catalan", PackageName: "catalan", DisplayName: "Catalan", Aliases: []string{"catalan", "ca", "cat", "català"}},
	{ID: "czech", ConstName: "Czech", PackageName: "czech", DisplayName: "Czech", Aliases: []string{"czech", "cs", "ces", "cze", "čeština", "cestina"}},
	{ID: "danish", ConstName: "Danish", PackageName: "danish", DisplayName: "Danish", Aliases: []string{"danish", "da", "dan", "dansk"}},
	{ID: "dutch", ConstName: "Dutch", PackageName: "dutch", DisplayName: "Dutch", Aliases: []string{"dutch", "nl", "nld", "dut", "nederlands"}},
	{ID: "dutch_porter", ConstName: "DutchPorter", PackageName: "dutchporter", DisplayName: "Dutch (Porter)", Aliases: []string{"dutch_porter", "dutchporter", "dutch-porter"}},
	{ID: "earlymodernenglish", ConstName: "EarlyModernEnglish", PackageName: "earlymodernenglish", DisplayName: "Early Modern English", Aliases: []string{"earlymodernenglish", "early-modern-english", "early_modern_english", "eme"}},
	{ID: "english", ConstName: "English", PackageName: "english", DisplayName: "English", Aliases: []string{"english", "en", "eng"}},
	{ID: "esperanto", ConstName: "Esperanto", PackageName: "esperanto", DisplayName: "Esperanto", Aliases: []string{"esperanto", "eo", "epo"}},
	{ID: "estonian", ConstName: "Estonian", PackageName: "estonian", DisplayName: "Estonian", Aliases: []string{"estonian", "et", "est", "eesti"}},
	{ID: "finnish", ConstName: "Finnish", PackageName: "finnish", DisplayName: "Finnish", Aliases: []string{"finnish", "fi", "fin", "suomi"}},
	{ID: "french", ConstName: "French", PackageName: "french", DisplayName: "French", Aliases: []string{"french", "fr", "fra", "fre", "français", "francais"}},
	{ID: "german", ConstName: "German", PackageName: "german", DisplayName: "German", Aliases: []string{"german", "de", "deu", "ger", "deutsch"}},
	{ID: "greek", ConstName: "Greek", PackageName: "greek", DisplayName: "Greek", Aliases: []string{"greek", "el", "ell", "gre", "ελληνικά"}},
	{ID: "hindi", ConstName: "Hindi", PackageName: "hindi", DisplayName: "Hindi", Aliases: []string{"hindi", "hi", "hin", "हिन्दी"}},
	{ID: "hungarian", ConstName: "Hungarian", PackageName: "hungarian", DisplayName: "Hungarian", Aliases: []string{"hungarian", "hu", "hun", "magyar"}},
	{ID: "indonesian", ConstName: "Indonesian", PackageName: "indonesian", DisplayName: "Indonesian", Aliases: []string{"indonesian", "id", "ind", "bahasa indonesia"}},
	{ID: "irish", ConstName: "Irish", PackageName: "irish", DisplayName: "Irish", Aliases: []string{"irish", "ga", "gle", "gaeilge"}},
	{ID: "italian", ConstName: "Italian", PackageName: "italian", DisplayName: "Italian", Aliases: []string{"italian", "it", "ita", "italiano"}},
	{ID: "lithuanian", ConstName: "Lithuanian", PackageName: "lithuanian", DisplayName: "Lithuanian", Aliases: []string{"lithuanian", "lt", "lit", "lietuvių", "lietuviu"}},
	{ID: "lovins", ConstName: "Lovins", PackageName: "lovins", DisplayName: "Lovins", Aliases: []string{"lovins"}},
	{ID: "nepali", ConstName: "Nepali", PackageName: "nepali", DisplayName: "Nepali", Aliases: []string{"nepali", "ne", "nep", "नेपाली"}},
	{ID: "norwegian", ConstName: "Norwegian", PackageName: "norwegian", DisplayName: "Norwegian", Aliases: []string{"norwegian", "no", "nb", "nor", "nob", "norsk", "bokmål", "bokmal"}},
	{ID: "persian", ConstName: "Persian", PackageName: "persian", DisplayName: "Persian", Aliases: []string{"persian", "fa", "fas", "per", "farsi", "فارسی"}},
	{ID: "polish", ConstName: "Polish", PackageName: "polish", DisplayName: "Polish", Aliases: []string{"polish", "pl", "pol", "polski"}},
	{ID: "porter", ConstName: "Porter", PackageName: "porter", DisplayName: "Porter", Aliases: []string{"porter"}},
	{ID: "portuguese", ConstName: "Portuguese", PackageName: "portuguese", DisplayName: "Portuguese", Aliases: []string{"portuguese", "pt", "por", "português", "portugues"}},
	{ID: "romanian", ConstName: "Romanian", PackageName: "romanian", DisplayName: "Romanian", Aliases: []string{"romanian", "ro", "ron", "rum", "română", "romana"}},
	{ID: "russian", ConstName: "Russian", PackageName: "russian", DisplayName: "Russian", Aliases: []string{"russian", "ru", "rus", "русский"}, NormalizeYo: true},
	{ID: "serbian", ConstName: "Serbian", PackageName: "serbian", DisplayName: "Serbian", Aliases: []string{"serbian", "sr", "srp", "српски", "srpski"}},
	{ID: "sesotho", ConstName: "Sesotho", PackageName: "sesotho", DisplayName: "Sesotho", Aliases: []string{"sesotho", "st", "sot", "sotho"}},
	{ID: "spanish", ConstName: "Spanish", PackageName: "spanish", DisplayName: "Spanish", Aliases: []string{"spanish", "es", "spa", "español", "espanol"}},
	{ID: "swedish", ConstName: "Swedish", PackageName: "swedish", DisplayName: "Swedish", Aliases: []string{"swedish", "sv", "swe", "svenska"}},
	{ID: "tamil", ConstName: "Tamil", PackageName: "tamil", DisplayName: "Tamil", Aliases: []string{"tamil", "ta", "tam", "தமிழ்"}},
	{ID: "turkish", ConstName: "Turkish", PackageName: "turkish", DisplayName: "Turkish", Aliases: []string{"turkish", "tr", "tur", "türkçe", "turkce"}, TurkishCase: true},
	{ID: "ukrainian", ConstName: "Ukrainian", PackageName: "ukrainian", DisplayName: "Ukrainian", Aliases: []string{"ukrainian", "uk", "ukr", "українська"}},
	{ID: "yiddish", ConstName: "Yiddish", PackageName: "yiddish", DisplayName: "Yiddish", Aliases: []string{"yiddish", "yi", "yid", "ייִדיש"}},
}

func main() {
	algorithmsDir := flag.String("algorithms", "algorithms", "Directory containing Snowball .sbl and optional .stop.txt files")
	outputDir := flag.String("out", "pkg/stems", "Output directory for generated stemmer packages and registry")
	flag.Parse()

	if err := generate(*algorithmsDir, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "sblgen: %v\n", err)
		os.Exit(1)
	}
}

func generate(algorithmsDir, outputDir string) error {
	for _, spec := range catalog {
		sblPath := filepath.Join(algorithmsDir, spec.ID+".sbl")
		sourceBytes, err := os.ReadFile(sblPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", sblPath, err)
		}

		program, err := sbl.Parse(string(sourceBytes))
		if err != nil {
			return fmt.Errorf("parse %s: %w", sblPath, err)
		}

		var stopWords []string
		stopPath := filepath.Join(algorithmsDir, spec.ID+".stop.txt")
		stopBytes, stopErr := os.ReadFile(stopPath)
		if stopErr == nil {
			stopWords = sbl.ParseStopWords(string(stopBytes), spec.NormalizeYo)
		} else if !errors.Is(stopErr, os.ErrNotExist) {
			return fmt.Errorf("read %s: %w", stopPath, stopErr)
		}

		code, err := sbl.Compile(program, sbl.CompileOptions{
			PackageName:     spec.PackageName,
			LanguageDisplay: spec.DisplayName,
			SourceFile:      filepath.ToSlash(filepath.Join("algorithms", spec.ID+".sbl")),
			StopWords:       stopWords,
			TurkishCase:     spec.TurkishCase,
			NormalizeYo:     spec.NormalizeYo,
		})
		if err != nil {
			return fmt.Errorf("compile %s: %w", spec.ID, err)
		}

		packageDir := filepath.Join(outputDir, spec.PackageName)
		if err := os.MkdirAll(packageDir, 0o750); err != nil {
			return fmt.Errorf("create directory %s: %w", packageDir, err)
		}

		stemPath := filepath.Join(packageDir, "stem_gen.go")
		if err := os.WriteFile(stemPath, code, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", stemPath, err)
		}

		testCode, err := generatePackageTests(spec, len(stopWords) > 0)
		if err != nil {
			return fmt.Errorf("generate tests for %s: %w", spec.ID, err)
		}
		testPath := filepath.Join(packageDir, "stem_gen_test.go")
		if err := os.WriteFile(testPath, testCode, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", testPath, err)
		}
	}

	registryCode, err := generateRegistry()
	if err != nil {
		return err
	}
	registryPath := filepath.Join(outputDir, "registry_gen.go")
	if err := os.WriteFile(registryPath, registryCode, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", registryPath, err)
	}
	return nil
}

func generatePackageTests(spec languageSpec, hasStopWords bool) ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Copyright (c) 2026 Cafecito Games LLC\n")
	fmt.Fprintf(&buf, "// SPDX-License-Identifier: MIT\n\n")
	fmt.Fprintf(&buf, "// Code generated by sblgen from algorithms/%s.sbl; DO NOT EDIT.\n\n", spec.ID)
	fmt.Fprintf(&buf, "package %s\n\n", spec.PackageName)
	fmt.Fprintf(&buf, "import (\n")
	if hasStopWords {
		fmt.Fprintf(&buf, "\t\"slices\"\n")
	}
	fmt.Fprintf(&buf, "\t\"testing\"\n\n")
	fmt.Fprintf(&buf, "\t\"github.com/stretchr/testify/assert\"\n")
	if hasStopWords {
		fmt.Fprintf(&buf, "\t\"github.com/stretchr/testify/require\"\n")
	}
	fmt.Fprintf(&buf, "\n\t\"github.com/cafecito-games/stems/internal/testutil\"\n")
	fmt.Fprintf(&buf, ")\n\n")

	fmt.Fprintf(&buf, "func TestStemVocabulary(t *testing.T) {\n")
	fmt.Fprintf(&buf, "\tt.Parallel()\n")
	fmt.Fprintf(&buf, "\ttestutil.VerifyVocabulary(t, %s, stemRaw)\n", strconv.Quote(spec.ID))
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "func TestStemEdgeCases(t *testing.T) {\n")
	fmt.Fprintf(&buf, "\tt.Parallel()\n")
	fmt.Fprintf(&buf, "\tassert.Empty(t, Stem(\"\"))\n")
	fmt.Fprintf(&buf, "\tassert.Empty(t, Stem(\"   \"))\n")
	fmt.Fprintf(&buf, "\tassert.Equal(t, Stem(\"abc\"), Stem(\"  ABC  \"))\n")
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "func TestStopWords(t *testing.T) {\n")
	fmt.Fprintf(&buf, "\tt.Parallel()\n")
	if hasStopWords {
		fmt.Fprintf(&buf, "\twords := StopWords()\n")
		fmt.Fprintf(&buf, "\trequire.NotEmpty(t, words)\n")
		fmt.Fprintf(&buf, "\tassert.True(t, slices.IsSorted(words))\n")
		fmt.Fprintf(&buf, "\tassert.True(t, IsStopWord(\"  \"+words[0]+\"  \"))\n")
		fmt.Fprintf(&buf, "\twords[0] = \"__mutated__\"\n")
		fmt.Fprintf(&buf, "\tassert.NotEqual(t, \"__mutated__\", StopWords()[0])\n")
	} else {
		fmt.Fprintf(&buf, "\tassert.Empty(t, StopWords())\n")
	}
	fmt.Fprintf(&buf, "\tassert.False(t, IsStopWord(\"nonexistentstopword12345\"))\n")
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "func BenchmarkStem(b *testing.B) {\n")
	fmt.Fprintf(&buf, "\ttestutil.BenchmarkVocabulary(b, %s, Stem)\n", strconv.Quote(spec.ID))
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "func FuzzStem(f *testing.F) {\n")
	fmt.Fprintf(&buf, "\tfor _, seed := range []string{\"\", \"a\", \"test\", \"café\", \"12345\", \"'quoted'\"} {\n")
	fmt.Fprintf(&buf, "\t\tf.Add(seed)\n")
	fmt.Fprintf(&buf, "\t}\n")
	fmt.Fprintf(&buf, "\tf.Fuzz(func(t *testing.T, input string) {\n")
	fmt.Fprintf(&buf, "\t\tassert.Equal(t, Stem(input), Stem(input))\n")
	fmt.Fprintf(&buf, "\t})\n")
	fmt.Fprintf(&buf, "}\n")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format test file for %s: %w", spec.PackageName, err)
	}
	return formatted, nil
}

func generateRegistry() ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Copyright (c) 2026 Cafecito Games LLC\n")
	fmt.Fprintf(&buf, "// SPDX-License-Identifier: MIT\n\n")
	fmt.Fprintf(&buf, "// Code generated by sblgen; DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package stems\n\n")
	fmt.Fprintf(&buf, "import (\n")
	for _, spec := range catalog {
		fmt.Fprintf(&buf, "\t\"github.com/cafecito-games/stems/pkg/stems/%s\"\n", spec.PackageName)
	}
	fmt.Fprintf(&buf, ")\n\n")

	fmt.Fprintf(&buf, "// Supported stemming languages.\n")
	fmt.Fprintf(&buf, "const (\n")
	for _, spec := range catalog {
		fmt.Fprintf(&buf, "\t%s Language = %s\n", spec.ConstName, strconv.Quote(spec.ID))
	}
	fmt.Fprintf(&buf, ")\n\n")

	fmt.Fprintf(&buf, "var registry = map[Language]languageDefinition{\n")
	for _, spec := range catalog {
		fmt.Fprintf(&buf, "\t%s: {\n", spec.ConstName)
		fmt.Fprintf(&buf, "\t\tlanguage:   %s,\n", spec.ConstName)
		fmt.Fprintf(&buf, "\t\tstem:       %s.Stem,\n", spec.PackageName)
		fmt.Fprintf(&buf, "\t\tisStopWord: %s.IsStopWord,\n", spec.PackageName)
		fmt.Fprintf(&buf, "\t\tstopWords:  %s.StopWords,\n", spec.PackageName)
		fmt.Fprintf(&buf, "\t},\n")
	}
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "var languageAliases = map[string]Language{\n")
	for _, spec := range catalog {
		for _, alias := range spec.Aliases {
			fmt.Fprintf(&buf, "\t%s: %s,\n", strconv.Quote(alias), spec.ConstName)
		}
	}
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "var supportedLanguages = []Language{\n")
	for _, spec := range catalog {
		fmt.Fprintf(&buf, "\t%s,\n", spec.ConstName)
	}
	fmt.Fprintf(&buf, "}\n")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format registry_gen.go: %w", err)
	}
	return formatted, nil
}
