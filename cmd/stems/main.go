// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

// Command stems stems newline-delimited words from a file or standard input.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cafecito-games/stems/pkg/stems"
)

const maxScannerTokenSize = 1024 * 1024

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "stems: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, input io.Reader, output, errorOutput io.Writer) error {
	flags := flag.NewFlagSet("stems", flag.ContinueOnError)
	flags.SetOutput(errorOutput)

	languageFlag := flags.String("language", string(stems.English), "Language name or ISO 639 code (e.g., english, en, spanish, es)")
	inputPath := flags.String("input", "", "Path to input file of newline-delimited words (defaults to stdin)")
	skipStopWords := flags.Bool("skip-stop-words", false, "Leave stop words unchanged instead of stemming them")
	listLanguages := flags.Bool("list-languages", false, "Print supported languages and exit")
	showVersion := flags.Bool("version", false, "Print version information and exit")

	if err := flags.Parse(arguments); err != nil {
		return err
	}

	if *showVersion {
		_, err := fmt.Fprintf(output, "stems %s (commit %s, built %s)\n", version, commit, date)
		return err
	}

	if *listLanguages {
		for _, language := range stems.Languages() {
			if _, err := fmt.Fprintln(output, language); err != nil {
				return err
			}
		}
		return nil
	}

	language, err := stems.ParseLanguage(*languageFlag)
	if err != nil {
		return err
	}

	var options []stems.Option
	if *skipStopWords {
		options = append(options, stems.WithSkipStopWords())
	}

	stemmer, err := stems.New(language, options...)
	if err != nil {
		return err
	}

	source := input
	if *inputPath != "" && *inputPath != "-" {
		file, openErr := os.Open(*inputPath)
		if openErr != nil {
			return openErr
		}
		defer func() {
			_ = file.Close()
		}()
		source = file
	}

	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 0, 64*1024), maxScannerTokenSize)
	writer := bufio.NewWriter(output)

	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if _, writeErr := fmt.Fprintln(writer, stemmer.Stem(word)); writeErr != nil {
			return writeErr
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return scanErr
	}
	return writer.Flush()
}
