// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package snowball

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupingDenseAndSparse(t *testing.T) {
	t.Parallel()

	dense := NewGrouping("aeiouáéíóú")
	for _, character := range "aeiouáéíóú" {
		assert.True(t, dense.Contains(character), "dense grouping missing %q", character)
	}
	assert.False(t, dense.Contains('z'))
	assert.False(t, dense.Contains('б'))

	sparse := NewGrouping("a\u2603\u4e2d")
	for _, character := range "a\u2603\u4e2d" {
		assert.True(t, sparse.Contains(character), "sparse grouping missing %q", character)
	}
	assert.False(t, sparse.Contains('b'))
	assert.False(t, sparse.Contains('\u2604'))

	empty := NewGrouping("")
	assert.False(t, empty.Contains('a'))
}

func TestEnvNavigationAndGrouping(t *testing.T) {
	t.Parallel()

	var env Env
	var buffer [InlineBufferCapacity]byte
	env.InitWithBuffer("café", buffer[:])
	require.Equal(t, 4, env.RuneCount())

	vowels := NewGrouping("aeioué")
	require.True(t, env.OutGrouping(&vowels))
	require.True(t, env.InGrouping(&vowels))
	require.True(t, env.Hop(2))
	assert.Equal(t, env.Limit, env.Cursor)
	assert.False(t, env.Hop(1))

	require.True(t, env.InGroupingBackward(&vowels))
	require.True(t, env.HopBackward(3))
	assert.Equal(t, 0, env.Cursor)
	assert.False(t, env.HopBackward(1))
}

func TestEnvEditsInsertAndAttach(t *testing.T) {
	t.Parallel()

	var env Env
	var buffer [InlineBufferCapacity]byte
	env.InitWithBuffer("skating", buffer[:])
	env.Bra = 4
	env.Ket = 7
	env.Cursor = 4
	env.SliceDel()
	assert.Equal(t, "skat", env.Result("skating"))
	assert.Equal(t, 4, env.Cursor)

	env.Insert("e", true)
	assert.Equal(t, "skate", env.Result("skating"))
	assert.Equal(t, 4, env.Cursor)

	env.InitWithBuffer("sp", buffer[:])
	env.Cursor = 2
	env.Attach("l", true)
	assert.Equal(t, "spl", env.Result("sp"))
	assert.Equal(t, 3, env.Cursor)

	longInput := strings.Repeat("a", 200)
	env.InitWithBuffer(longInput, buffer[:])
	env.Bra = 0
	env.Ket = 10
	env.Cursor = 10
	env.SliceFrom(strings.Repeat("b", 50))
	assert.Equal(t, 240, env.Limit)
	assert.Equal(t, 50, env.Cursor)
}

func TestFindAmongForwardAndBackward(t *testing.T) {
	t.Parallel()

	type state struct {
		allowLong bool
	}

	forwardEntries := []Among[state]{
		{Text: "a", SubstringIndex: -1, Result: 1},
		{Text: "ab", SubstringIndex: 0, Result: 2},
		{Text: "abc", SubstringIndex: 1, Result: 3, Condition: func(s *state) bool { return s.allowLong }},
	}

	var env Env
	var buffer [InlineBufferCapacity]byte
	currentState := state{allowLong: false}
	env.InitWithBuffer("abcdef", buffer[:])
	assert.Equal(t, 2, FindAmong(&env, forwardEntries, &currentState))
	assert.Equal(t, 2, env.Cursor)

	currentState.allowLong = true
	env.Cursor = 0
	assert.Equal(t, 3, FindAmong(&env, forwardEntries, &currentState))
	assert.Equal(t, 3, env.Cursor)

	backwardEntries := []Among[state]{
		{Text: "e", SubstringIndex: -1, Result: 1},
		{Text: "de", SubstringIndex: 0, Result: 2},
		{Text: "cde", SubstringIndex: 1, Result: 3, Condition: func(*state) bool { return false }},
	}
	env.InitWithBuffer("abcde", buffer[:])
	env.Cursor = env.Limit
	assert.Equal(t, 2, FindAmongBackward(&env, backwardEntries, &currentState))
	assert.Equal(t, 3, env.Cursor)
}
