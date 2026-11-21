// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package sbl

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type lexer struct {
	source        string
	offset        int
	line          int
	column        int
	hasEscapes    bool
	escapeOpen    byte
	escapeClose   byte
	stringDefines map[string]string
}

func lex(source string) ([]Token, error) {
	scanner := &lexer{
		source:        source,
		line:          1,
		column:        1,
		stringDefines: make(map[string]string),
	}

	var tokens []Token
	for {
		if err := scanner.skipWhitespaceAndDirectives(); err != nil {
			return nil, err
		}
		if scanner.eof() {
			tokens = append(tokens, Token{Kind: TokenEOF, Line: scanner.line, Column: scanner.column})
			return tokens, nil
		}

		token, err := scanner.nextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
}

func (scanner *lexer) eof() bool {
	return scanner.offset >= len(scanner.source)
}

func (scanner *lexer) peek() byte {
	if scanner.eof() {
		return 0
	}
	return scanner.source[scanner.offset]
}

func (scanner *lexer) peekAt(delta int) byte {
	index := scanner.offset + delta
	if index < 0 || index >= len(scanner.source) {
		return 0
	}
	return scanner.source[index]
}

func (scanner *lexer) advance() byte {
	character := scanner.source[scanner.offset]
	scanner.offset++
	if character == '\n' {
		scanner.line++
		scanner.column = 1
	} else {
		scanner.column++
	}
	return character
}

func (scanner *lexer) skipWhitespaceAndComments() error {
	for !scanner.eof() {
		character := scanner.peek()
		if character == ' ' || character == '\t' || character == '\r' || character == '\n' {
			scanner.advance()
			continue
		}
		if character == '/' && scanner.peekAt(1) == '/' {
			for !scanner.eof() && scanner.peek() != '\n' {
				scanner.advance()
			}
			continue
		}
		if character == '/' && scanner.peekAt(1) == '*' {
			startLine := scanner.line
			scanner.advance()
			scanner.advance()
			closed := false
			for !scanner.eof() {
				if scanner.peek() == '*' && scanner.peekAt(1) == '/' {
					scanner.advance()
					scanner.advance()
					closed = true
					break
				}
				scanner.advance()
			}
			if !closed {
				return fmt.Errorf("line %d: unterminated block comment", startLine)
			}
			continue
		}
		break
	}
	return nil
}

func (scanner *lexer) skipWhitespaceAndDirectives() error {
	for {
		if err := scanner.skipWhitespaceAndComments(); err != nil {
			return err
		}
		if scanner.eof() {
			return nil
		}
		if scanner.matchKeywordDirective("stringescapes") {
			if err := scanner.parseStringEscapes(); err != nil {
				return err
			}
			continue
		}
		if scanner.matchKeywordDirective("stringdef") {
			if err := scanner.parseStringDef(); err != nil {
				return err
			}
			continue
		}
		return nil
	}
}

func (scanner *lexer) matchKeywordDirective(keyword string) bool {
	if !strings.HasPrefix(scanner.source[scanner.offset:], keyword) {
		return false
	}
	after := scanner.peekAt(len(keyword))
	if isIdentChar(after) {
		return false
	}
	for range len(keyword) {
		scanner.advance()
	}
	return true
}

func (scanner *lexer) parseStringEscapes() error {
	if err := scanner.skipWhitespaceAndComments(); err != nil {
		return err
	}
	if scanner.eof() {
		return fmt.Errorf("line %d: expected escape characters after stringescapes", scanner.line)
	}
	openCharacter := scanner.advance()
	if err := scanner.skipWhitespaceAndComments(); err != nil {
		return err
	}
	if scanner.eof() {
		return fmt.Errorf("line %d: expected closing escape character after stringescapes", scanner.line)
	}
	closeCharacter := scanner.advance()
	scanner.hasEscapes = true
	scanner.escapeOpen = openCharacter
	scanner.escapeClose = closeCharacter
	return nil
}

func (scanner *lexer) parseStringDef() error {
	if err := scanner.skipWhitespaceAndComments(); err != nil {
		return err
	}
	startLine := scanner.line
	var nameBuilder strings.Builder
	for !scanner.eof() {
		character := scanner.peek()
		if character == ' ' || character == '\t' || character == '\r' || character == '\n' {
			break
		}
		nameBuilder.WriteByte(scanner.advance())
	}
	name := nameBuilder.String()
	if name == "" {
		return fmt.Errorf("line %d: expected macro name after stringdef", startLine)
	}
	if err := scanner.skipWhitespaceAndComments(); err != nil {
		return err
	}

	mode := ""
	if strings.HasPrefix(scanner.source[scanner.offset:], "hex") && !isIdentChar(scanner.peekAt(3)) {
		for range 3 {
			scanner.advance()
		}
		mode = "hex"
		if err := scanner.skipWhitespaceAndComments(); err != nil {
			return err
		}
	} else if strings.HasPrefix(scanner.source[scanner.offset:], "decimal") && !isIdentChar(scanner.peekAt(7)) {
		for range 7 {
			scanner.advance()
		}
		mode = "decimal"
		if err := scanner.skipWhitespaceAndComments(); err != nil {
			return err
		}
	}

	if scanner.peek() != '\'' {
		return fmt.Errorf("line %d: expected string literal in stringdef %q", scanner.line, name)
	}
	literal, err := scanner.readStringLiteral()
	if err != nil {
		return err
	}

	if mode != "" {
		literal, err = decodeNumericStringDef(literal, mode)
		if err != nil {
			return fmt.Errorf("line %d: stringdef %q: %w", startLine, name, err)
		}
	}
	scanner.stringDefines[name] = literal
	return nil
}

func decodeNumericStringDef(raw, mode string) (string, error) {
	base := 16
	if mode == "decimal" {
		base = 10
	}
	var builder strings.Builder
	for field := range strings.FieldsSeq(raw) {
		codePoint, err := strconv.ParseInt(field, base, 32)
		if err != nil {
			return "", err
		}
		builder.WriteRune(rune(codePoint))
	}
	return builder.String(), nil
}

func (scanner *lexer) readStringLiteral() (string, error) {
	startLine := scanner.line
	if scanner.advance() != '\'' {
		return "", fmt.Errorf("line %d: expected opening quote", startLine)
	}

	var builder strings.Builder
	for !scanner.eof() {
		character := scanner.advance()
		if character == '\'' {
			return builder.String(), nil
		}
		if scanner.hasEscapes && character == scanner.escapeOpen {
			var macroBuilder strings.Builder
			closed := false
			for !scanner.eof() {
				next := scanner.advance()
				if next == scanner.escapeClose {
					closed = true
					break
				}
				macroBuilder.WriteByte(next)
			}
			if !closed {
				return "", fmt.Errorf("line %d: unterminated string escape", startLine)
			}
			macro := macroBuilder.String()
			if err := scanner.expandMacro(&builder, macro, startLine); err != nil {
				return "", err
			}
			continue
		}
		builder.WriteByte(character)
	}
	return "", fmt.Errorf("line %d: unterminated string literal", startLine)
}

func (scanner *lexer) expandMacro(builder *strings.Builder, macro string, line int) error {
	if isWhitespaceContinuation(macro) {
		return nil
	}
	if defined, ok := scanner.stringDefines[macro]; ok {
		builder.WriteString(defined)
		return nil
	}
	if macro == "'" {
		builder.WriteByte('\'')
		return nil
	}
	if len(macro) == 1 && macro[0] == scanner.escapeOpen {
		builder.WriteByte(scanner.escapeOpen)
		return nil
	}
	if strings.HasPrefix(macro, "U+") || strings.HasPrefix(macro, "u+") {
		upperKey := "U+" + strings.ToUpper(macro[2:])
		if defined, ok := scanner.stringDefines[upperKey]; ok {
			builder.WriteString(defined)
			return nil
		}
		codePoint, err := strconv.ParseInt(macro[2:], 16, 32)
		if err != nil {
			return fmt.Errorf("line %d: invalid Unicode escape %q: %w", line, macro, err)
		}
		builder.WriteRune(rune(codePoint))
		return nil
	}
	return fmt.Errorf("line %d: undefined string macro %q", line, macro)
}

func isWhitespaceContinuation(text string) bool {
	if text == "" {
		return false
	}
	hasNewline := false
	for _, character := range text {
		if character == '\n' {
			hasNewline = true
		} else if !unicode.IsSpace(character) {
			return false
		}
	}
	return hasNewline
}

func (scanner *lexer) nextToken() (Token, error) {
	line := scanner.line
	column := scanner.column
	character := scanner.peek()

	if character == '\'' {
		value, err := scanner.readStringLiteral()
		if err != nil {
			return Token{}, err
		}
		return Token{Kind: TokenString, Value: value, Line: line, Column: column}, nil
	}

	if character >= '0' && character <= '9' {
		start := scanner.offset
		for !scanner.eof() && scanner.peek() >= '0' && scanner.peek() <= '9' {
			scanner.advance()
		}
		return Token{Kind: TokenNumber, Value: scanner.source[start:scanner.offset], Line: line, Column: column}, nil
	}

	if isIdentStart(character) {
		start := scanner.offset
		for !scanner.eof() && isIdentChar(scanner.peek()) {
			scanner.advance()
		}
		word := scanner.source[start:scanner.offset]
		if kind, ok := keywords[word]; ok {
			return Token{Kind: kind, Value: word, Line: line, Column: column}, nil
		}
		return Token{Kind: TokenIdent, Value: word, Line: line, Column: column}, nil
	}

	// Two-character operators first.
	next := scanner.peekAt(1)
	switch {
	case character == '+' && next == '=':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenPlusAssign, Value: "+=", Line: line, Column: column}, nil
	case character == '-' && next == '=':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenSubAssign, Value: "-=", Line: line, Column: column}, nil
	case character == '*' && next == '=':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenMulAssign, Value: "*=", Line: line, Column: column}, nil
	case character == '/' && next == '=':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenDivAssign, Value: "/=", Line: line, Column: column}, nil
	case character == '=' && next == '=':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenEq, Value: "==", Line: line, Column: column}, nil
	case character == '!' && next == '=':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenNotEq, Value: "!=", Line: line, Column: column}, nil
	case character == '<' && next == '=':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenLessEq, Value: "<=", Line: line, Column: column}, nil
	case character == '>' && next == '=':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenGreaterEq, Value: ">=", Line: line, Column: column}, nil
	case character == '<' && next == '-':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenSliceFrom, Value: "<-", Line: line, Column: column}, nil
	case character == '-' && next == '>':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenSliceTo, Value: "->", Line: line, Column: column}, nil
	case character == '<' && next == '+':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenInsertOp, Value: "<+", Line: line, Column: column}, nil
	case character == '=' && next == '>':
		scanner.advance()
		scanner.advance()
		return Token{Kind: TokenCopyOp, Value: "=>", Line: line, Column: column}, nil
	}

	scanner.advance()
	switch character {
	case '(':
		return Token{Kind: TokenLParen, Value: "(", Line: line, Column: column}, nil
	case ')':
		return Token{Kind: TokenRParen, Value: ")", Line: line, Column: column}, nil
	case '[':
		return Token{Kind: TokenLBracket, Value: "[", Line: line, Column: column}, nil
	case ']':
		return Token{Kind: TokenRBracket, Value: "]", Line: line, Column: column}, nil
	case '$':
		return Token{Kind: TokenDollar, Value: "$", Line: line, Column: column}, nil
	case '+':
		return Token{Kind: TokenPlus, Value: "+", Line: line, Column: column}, nil
	case '-':
		return Token{Kind: TokenMinus, Value: "-", Line: line, Column: column}, nil
	case '*':
		return Token{Kind: TokenStar, Value: "*", Line: line, Column: column}, nil
	case '/':
		return Token{Kind: TokenSlash, Value: "/", Line: line, Column: column}, nil
	case '=':
		return Token{Kind: TokenAssign, Value: "=", Line: line, Column: column}, nil
	case '<':
		return Token{Kind: TokenLess, Value: "<", Line: line, Column: column}, nil
	case '>':
		return Token{Kind: TokenGreater, Value: ">", Line: line, Column: column}, nil
	case '?':
		return Token{Kind: TokenQuestion, Value: "?", Line: line, Column: column}, nil
	default:
		return Token{}, fmt.Errorf("line %d, column %d: unexpected character %q", line, column, character)
	}
}

func isIdentStart(character byte) bool {
	return (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')
}

func isIdentChar(character byte) bool {
	return isIdentStart(character) || (character >= '0' && character <= '9') || character == '_'
}
