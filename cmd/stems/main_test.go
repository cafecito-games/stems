// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunFromStdin(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("Accumulations\nbecause\nrunning\n")
	var output bytes.Buffer
	var errorOutput bytes.Buffer

	require.NoError(t, run([]string{"-language", "en", "-skip-stop-words"}, input, &output, &errorOutput))
	assert.Equal(t, "accumul\nbecause\nrun\n", output.String())
}

func TestRunFromFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "words.txt")
	require.NoError(t, os.WriteFile(path, []byte("preocuparse\noposición\n"), 0o600))

	var output bytes.Buffer
	var errorOutput bytes.Buffer
	require.NoError(t, run([]string{"-language", "spanish", "-input", path}, strings.NewReader(""), &output, &errorOutput))
	assert.Equal(t, "preocup\noposicion\n", output.String())
}

func TestRunListLanguages(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	var errorOutput bytes.Buffer
	require.NoError(t, run([]string{"-list-languages"}, strings.NewReader(""), &output, &errorOutput))

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	assert.Len(t, lines, 39)
}

func TestRunVersion(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	var errorOutput bytes.Buffer
	require.NoError(t, run([]string{"-version"}, strings.NewReader(""), &output, &errorOutput))
	assert.Contains(t, output.String(), "stems "+version)
}

func TestRunUnsupportedLanguage(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	var errorOutput bytes.Buffer
	assert.Error(t, run([]string{"-language", "unknown"}, strings.NewReader("word\n"), &output, &errorOutput))
}
