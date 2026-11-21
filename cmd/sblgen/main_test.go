// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAllLanguages(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	algorithmsDir := filepath.Join("..", "..", "algorithms")

	require.NoError(t, generate(algorithmsDir, outputDir))
	require.FileExists(t, filepath.Join(outputDir, "registry_gen.go"))

	for _, spec := range catalog {
		assert.FileExists(t, filepath.Join(outputDir, spec.PackageName, "stem_gen.go"))
		assert.FileExists(t, filepath.Join(outputDir, spec.PackageName, "stem_gen_test.go"))
	}
}
