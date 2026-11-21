// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

// Package sbl implements a Snowball (.sbl) lexer, parser, and Go 1.26 code generator.
package sbl

// TokenKind identifies the lexical category of a Snowball token.
type TokenKind int

// Snowball token kinds.
const (
	TokenEOF TokenKind = iota
	TokenIdent
	TokenString
	TokenNumber

	// Punctuation and operators
	TokenLParen     // (
	TokenRParen     // )
	TokenLBracket   // [
	TokenRBracket   // ]
	TokenDollar     // $
	TokenPlus       // +
	TokenMinus      // -
	TokenStar       // *
	TokenSlash      // /
	TokenAssign     // =
	TokenPlusAssign // +=
	TokenSubAssign  // -=
	TokenMulAssign  // *=
	TokenDivAssign  // /=
	TokenEq         // ==
	TokenNotEq      // !=
	TokenLess       // <
	TokenLessEq     // <=
	TokenGreater    // >
	TokenGreaterEq  // >=
	TokenSliceFrom  // <-
	TokenSliceTo    // ->
	TokenInsertOp   // <+
	TokenCopyOp     // =>
	TokenQuestion   // ?

	// Keywords
	TokenAmong
	TokenAnd
	TokenAs
	TokenAtLeast
	TokenAtLimit
	TokenAtMark
	TokenAttach
	TokenBackwardMode
	TokenBackwards
	TokenBooleans
	TokenCursor
	TokenDefine
	TokenDelete
	TokenDo
	TokenExternals
	TokenFail
	TokenFalse
	TokenFor
	TokenGoPast
	TokenGoTo
	TokenGroupings
	TokenHop
	TokenInsert
	TokenIntegers
	TokenLen
	TokenLenOf
	TokenLimit
	TokenLoop
	TokenMaxInt
	TokenMinInt
	TokenNext
	TokenNon
	TokenNot
	TokenOr
	TokenRepeat
	TokenReverse
	TokenRoutines
	TokenSet
	TokenSetLimit
	TokenSetMark
	TokenSize
	TokenSizeOf
	TokenStrings
	TokenSubstring
	TokenTest
	TokenToLimit
	TokenToMark
	TokenTrue
	TokenTry
	TokenUnset
)

// Token represents a single lexed Snowball token with source location.
type Token struct {
	Kind   TokenKind
	Value  string
	Line   int
	Column int
}

var keywords = map[string]TokenKind{
	"among":        TokenAmong,
	"and":          TokenAnd,
	"as":           TokenAs,
	"atleast":      TokenAtLeast,
	"atlimit":      TokenAtLimit,
	"atmark":       TokenAtMark,
	"attach":       TokenAttach,
	"backwardmode": TokenBackwardMode,
	"backwards":    TokenBackwards,
	"booleans":     TokenBooleans,
	"cursor":       TokenCursor,
	"define":       TokenDefine,
	"delete":       TokenDelete,
	"do":           TokenDo,
	"externals":    TokenExternals,
	"fail":         TokenFail,
	"false":        TokenFalse,
	"for":          TokenFor,
	"gopast":       TokenGoPast,
	"goto":         TokenGoTo,
	"groupings":    TokenGroupings,
	"hop":          TokenHop,
	"insert":       TokenInsert,
	"integers":     TokenIntegers,
	"len":          TokenLen,
	"lenof":        TokenLenOf,
	"limit":        TokenLimit,
	"loop":         TokenLoop,
	"maxint":       TokenMaxInt,
	"minint":       TokenMinInt,
	"next":         TokenNext,
	"non":          TokenNon,
	"not":          TokenNot,
	"or":           TokenOr,
	"repeat":       TokenRepeat,
	"reverse":      TokenReverse,
	"routines":     TokenRoutines,
	"set":          TokenSet,
	"setlimit":     TokenSetLimit,
	"setmark":      TokenSetMark,
	"size":         TokenSize,
	"sizeof":       TokenSizeOf,
	"strings":      TokenStrings,
	"substring":    TokenSubstring,
	"test":         TokenTest,
	"tolimit":      TokenToLimit,
	"tomark":       TokenToMark,
	"true":         TokenTrue,
	"try":          TokenTry,
	"unset":        TokenUnset,
}
