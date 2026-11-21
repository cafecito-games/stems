// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package sbl

import (
	"fmt"
	"slices"
	"strconv"
)

type parser struct {
	tokens  []Token
	index   int
	program *Program
	nextID  int
}

// Parse lexes and parses a Snowball (.sbl) program into an AST.
func Parse(source string) (*Program, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}

	instance := &parser{
		tokens: tokens,
		program: &Program{
			GroupingDefs: make(map[string][]rune),
			stringSet:    make(map[string]bool),
			integerSet:   make(map[string]bool),
			booleanSet:   make(map[string]bool),
			routineSet:   make(map[string]bool),
			groupingSet:  make(map[string]bool),
		},
	}

	if err := instance.parseProgramItems(false); err != nil {
		return nil, err
	}
	if instance.peek().Kind != TokenEOF {
		token := instance.peek()
		return nil, fmt.Errorf("line %d, column %d: unexpected token %q", token.Line, token.Column, token.Value)
	}

	for _, routine := range instance.program.RoutineDefs {
		if err := linkSubstrings(routine.Body, routine.Backward); err != nil {
			return nil, fmt.Errorf("routine %q: %w", routine.Name, err)
		}
	}

	return instance.program, nil
}

func (instance *parser) peek() Token {
	if instance.index >= len(instance.tokens) {
		return Token{Kind: TokenEOF}
	}
	return instance.tokens[instance.index]
}

func (instance *parser) advance() Token {
	token := instance.peek()
	if instance.index < len(instance.tokens) {
		instance.index++
	}
	return token
}

func (instance *parser) expect(kind TokenKind) error {
	token := instance.peek()
	if token.Kind != kind {
		return fmt.Errorf("line %d, column %d: unexpected token %q", token.Line, token.Column, token.Value)
	}
	instance.advance()
	return nil
}

// acceptName accepts an identifier or a demotable keyword (len, lenof) as a symbol name.
func (instance *parser) acceptName() (Token, bool) {
	token := instance.peek()
	if token.Kind == TokenIdent || token.Kind == TokenLen || token.Kind == TokenLenOf {
		instance.advance()
		return token, true
	}
	return Token{}, false
}

func (instance *parser) isDeclaredName(token Token) bool {
	if token.Kind == TokenIdent {
		return true
	}
	if token.Kind == TokenLen || token.Kind == TokenLenOf {
		name := token.Value
		return instance.program.stringSet[name] ||
			instance.program.integerSet[name] ||
			instance.program.booleanSet[name] ||
			instance.program.routineSet[name] ||
			instance.program.groupingSet[name]
	}
	return false
}

func (instance *parser) parseProgramItems(backward bool) error {
	for {
		token := instance.peek()
		switch token.Kind {
		case TokenStrings, TokenIntegers, TokenBooleans, TokenRoutines, TokenExternals, TokenGroupings:
			if err := instance.parseDeclaration(); err != nil {
				return err
			}
		case TokenDefine:
			if err := instance.parseDefine(backward); err != nil {
				return err
			}
		case TokenBackwardMode:
			instance.advance()
			if err := instance.expect(TokenLParen); err != nil {
				return err
			}
			if err := instance.parseProgramItems(true); err != nil {
				return err
			}
			if err := instance.expect(TokenRParen); err != nil {
				return err
			}
		case TokenEOF, TokenRParen:
			return nil
		default:
			return fmt.Errorf("line %d, column %d: unexpected top-level token %q", token.Line, token.Column, token.Value)
		}
	}
}

func (instance *parser) parseDeclaration() error {
	kind := instance.advance().Kind
	if err := instance.expect(TokenLParen); err != nil {
		return err
	}
	for instance.peek().Kind != TokenRParen {
		nameToken, ok := instance.acceptName()
		if !ok {
			token := instance.peek()
			return fmt.Errorf("line %d, column %d: expected identifier in declaration, got %q", token.Line, token.Column, token.Value)
		}
		name := nameToken.Value
		switch kind {
		case TokenStrings:
			if !instance.program.stringSet[name] {
				instance.program.stringSet[name] = true
				instance.program.Strings = append(instance.program.Strings, name)
			}
		case TokenIntegers:
			if !instance.program.integerSet[name] {
				instance.program.integerSet[name] = true
				instance.program.Integers = append(instance.program.Integers, name)
			}
		case TokenBooleans:
			if !instance.program.booleanSet[name] {
				instance.program.booleanSet[name] = true
				instance.program.Booleans = append(instance.program.Booleans, name)
			}
		case TokenRoutines:
			if !instance.program.routineSet[name] {
				instance.program.routineSet[name] = true
				instance.program.Routines = append(instance.program.Routines, name)
			}
		case TokenExternals:
			if !instance.program.routineSet[name] {
				instance.program.routineSet[name] = true
			}
			if !slices.Contains(instance.program.Externals, name) {
				instance.program.Externals = append(instance.program.Externals, name)
			}
		case TokenGroupings:
			if !instance.program.groupingSet[name] {
				instance.program.groupingSet[name] = true
				instance.program.Groupings = append(instance.program.Groupings, name)
			}
		}
	}
	instance.advance() // consume ')'
	return nil
}

func (instance *parser) parseDefine(backward bool) error {
	instance.advance() // consume 'define'
	nameToken, ok := instance.acceptName()
	if !ok {
		token := instance.peek()
		return fmt.Errorf("line %d, column %d: expected name after define, got %q", token.Line, token.Column, token.Value)
	}
	name := nameToken.Value

	if instance.peek().Kind == TokenAs {
		instance.advance()
		body, err := instance.parseCommand(backward)
		if err != nil {
			return err
		}
		if !instance.program.routineSet[name] {
			instance.program.routineSet[name] = true
			instance.program.Routines = append(instance.program.Routines, name)
		}
		instance.program.RoutineDefs = append(instance.program.RoutineDefs, &RoutineDef{
			Name:     name,
			Backward: backward,
			Body:     body,
			External: slices.Contains(instance.program.Externals, name),
		})
		return nil
	}

	// Grouping definition: define g G (+|- G)*
	runes, err := instance.parseGroupingExpr()
	if err != nil {
		return err
	}
	if !instance.program.groupingSet[name] {
		instance.program.groupingSet[name] = true
		instance.program.Groupings = append(instance.program.Groupings, name)
	}
	instance.program.GroupingDefs[name] = runes
	return nil
}

func (instance *parser) parseGroupingExpr() ([]rune, error) {
	current, err := instance.parseGroupingAtom()
	if err != nil {
		return nil, err
	}
	runeSet := make(map[rune]bool, len(current))
	for _, character := range current {
		runeSet[character] = true
	}

	for instance.peek().Kind == TokenPlus || instance.peek().Kind == TokenMinus {
		operator := instance.advance().Kind
		next, err := instance.parseGroupingAtom()
		if err != nil {
			return nil, err
		}
		for _, character := range next {
			if operator == TokenPlus {
				runeSet[character] = true
			} else {
				delete(runeSet, character)
			}
		}
	}

	result := make([]rune, 0, len(runeSet))
	for character := range runeSet {
		result = append(result, character)
	}
	slices.Sort(result)
	return result, nil
}

func (instance *parser) parseGroupingAtom() ([]rune, error) {
	token := instance.peek()
	if token.Kind == TokenString {
		instance.advance()
		return []rune(token.Value), nil
	}
	if instance.isDeclaredName(token) {
		instance.advance()
		existing, ok := instance.program.GroupingDefs[token.Value]
		if !ok {
			return nil, fmt.Errorf("line %d, column %d: grouping %q used before definition", token.Line, token.Column, token.Value)
		}
		return existing, nil
	}
	return nil, fmt.Errorf("line %d, column %d: expected string or grouping name, got %q", token.Line, token.Column, token.Value)
}

func (instance *parser) parseCommand(backward bool) (Command, error) {
	return instance.parseOrAnd(backward)
}

func (instance *parser) parseOrAnd(backward bool) (Command, error) {
	left, err := instance.parsePrefix(backward)
	if err != nil {
		return nil, err
	}
	for instance.peek().Kind == TokenOr || instance.peek().Kind == TokenAnd {
		operator := instance.advance().Kind
		right, err := instance.parsePrefix(backward)
		if err != nil {
			return nil, err
		}
		left = &BinaryCmd{Op: operator, Left: left, Right: right}
	}
	return left, nil
}

func (instance *parser) parsePrefix(backward bool) (Command, error) {
	token := instance.peek()
	switch token.Kind {
	case TokenNot, TokenTest, TokenTry, TokenDo, TokenFail, TokenGoTo, TokenGoPast, TokenRepeat:
		instance.advance()
		body, err := instance.parsePrefix(backward)
		if err != nil {
			return nil, err
		}
		return &UnaryCmd{Op: token.Kind, Body: body}, nil
	case TokenBackwards:
		instance.advance()
		body, err := instance.parsePrefix(true)
		if err != nil {
			return nil, err
		}
		return &UnaryCmd{Op: TokenBackwards, Body: body}, nil
	case TokenReverse:
		instance.advance()
		body, err := instance.parsePrefix(!backward)
		if err != nil {
			return nil, err
		}
		return &UnaryCmd{Op: TokenReverse, Body: body}, nil
	case TokenLoop, TokenAtLeast:
		instance.advance()
		countExpr, err := instance.parseArithExpr()
		if err != nil {
			return nil, err
		}
		body, err := instance.parsePrefix(backward)
		if err != nil {
			return nil, err
		}
		return &LoopCmd{AtLeast: token.Kind == TokenAtLeast, Count: countExpr, Body: body}, nil
	case TokenSetLimit:
		instance.advance()
		limitCmd, err := instance.parseCommand(backward)
		if err != nil {
			return nil, err
		}
		if err := instance.expect(TokenFor); err != nil {
			return nil, err
		}
		bodyCmd, err := instance.parsePrefix(backward)
		if err != nil {
			return nil, err
		}
		return &SetLimitCmd{Limit: limitCmd, Body: bodyCmd}, nil
	default:
		return instance.parsePrimary(backward)
	}
}

func (instance *parser) parsePrimary(backward bool) (Command, error) {
	token := instance.peek()
	switch token.Kind {
	case TokenLParen:
		instance.advance()
		var commands []Command
		for instance.peek().Kind != TokenRParen && instance.peek().Kind != TokenEOF {
			command, err := instance.parseCommand(backward)
			if err != nil {
				return nil, err
			}
			commands = append(commands, command)
		}
		if err := instance.expect(TokenRParen); err != nil {
			return nil, err
		}
		if len(commands) == 0 {
			return &LiteralBoolCmd{Value: true}, nil
		}
		if len(commands) == 1 {
			return commands[0], nil
		}
		return &SequenceCmd{Commands: commands}, nil

	case TokenDollar:
		return instance.parseDollarCommand(backward)

	case TokenString:
		instance.advance()
		return &StringMatchCmd{Value: token.Value}, nil

	case TokenLBracket:
		instance.advance()
		return &SliceBoundCmd{IsClose: false}, nil

	case TokenRBracket:
		instance.advance()
		return &SliceBoundCmd{IsClose: true}, nil

	case TokenSliceFrom:
		instance.advance()
		isVar, value, err := instance.parseStringOperand()
		if err != nil {
			return nil, err
		}
		return &SliceEditCmd{Op: TokenSliceFrom, IsVariable: isVar, Value: value}, nil

	case TokenDelete:
		instance.advance()
		return &SliceEditCmd{Op: TokenDelete}, nil

	case TokenInsert, TokenInsertOp:
		instance.advance()
		isVar, value, err := instance.parseStringOperand()
		if err != nil {
			return nil, err
		}
		return &SliceEditCmd{Op: TokenInsert, IsVariable: isVar, Value: value}, nil

	case TokenAttach:
		instance.advance()
		isVar, value, err := instance.parseStringOperand()
		if err != nil {
			return nil, err
		}
		return &SliceEditCmd{Op: TokenAttach, IsVariable: isVar, Value: value}, nil

	case TokenSliceTo:
		instance.advance()
		nameToken, ok := instance.acceptName()
		if !ok {
			next := instance.peek()
			return nil, fmt.Errorf("line %d, column %d: expected string variable after ->, got %q", next.Line, next.Column, next.Value)
		}
		return &SliceEditCmd{Op: TokenSliceTo, IsVariable: true, Value: nameToken.Value}, nil

	case TokenCopyOp:
		instance.advance()
		nameToken, ok := instance.acceptName()
		if !ok {
			next := instance.peek()
			return nil, fmt.Errorf("line %d, column %d: expected string variable after =>, got %q", next.Line, next.Column, next.Value)
		}
		return &StringAssignCmd{Name: nameToken.Value, IsVariable: false, Value: ""}, nil

	case TokenAssign:
		instance.advance()
		isVar, value, err := instance.parseStringOperand()
		if err != nil {
			return nil, err
		}
		return &StringAssignCmd{Name: "", IsVariable: isVar, Value: value}, nil

	case TokenHop:
		instance.advance()
		countExpr, err := instance.parseArithExpr()
		if err != nil {
			return nil, err
		}
		return &HopCmd{Count: countExpr}, nil

	case TokenNext:
		instance.advance()
		return &HopCmd{Count: &IntLiteralExpr{Value: 1}}, nil

	case TokenSetMark:
		instance.advance()
		nameToken, ok := instance.acceptName()
		if !ok {
			next := instance.peek()
			return nil, fmt.Errorf("line %d, column %d: expected integer name after setmark, got %q", next.Line, next.Column, next.Value)
		}
		return &MarkCmd{Op: TokenSetMark, Name: nameToken.Value}, nil

	case TokenToMark, TokenAtMark:
		instance.advance()
		expr, err := instance.parseArithExpr()
		if err != nil {
			return nil, err
		}
		return &MarkCmd{Op: token.Kind, Expr: expr}, nil

	case TokenToLimit, TokenAtLimit:
		instance.advance()
		return &MarkCmd{Op: token.Kind}, nil

	case TokenSubstring:
		instance.advance()
		return &SubstringCmd{AmongID: -1}, nil

	case TokenAmong:
		return instance.parseAmong(backward)

	case TokenSet, TokenUnset:
		instance.advance()
		nameToken, ok := instance.acceptName()
		if !ok {
			next := instance.peek()
			return nil, fmt.Errorf("line %d, column %d: expected boolean name after %s, got %q", next.Line, next.Column, token.Value, next.Value)
		}
		return &BoolCmd{Op: token.Kind, Name: nameToken.Value}, nil

	case TokenNon:
		instance.advance()
		if instance.peek().Kind == TokenMinus {
			instance.advance()
		}
		nameToken, ok := instance.acceptName()
		if !ok {
			next := instance.peek()
			return nil, fmt.Errorf("line %d, column %d: expected grouping name after non, got %q", next.Line, next.Column, next.Value)
		}
		return &GroupingCmd{Name: nameToken.Value, Negated: true}, nil

	case TokenTrue:
		instance.advance()
		return &LiteralBoolCmd{Value: true}, nil

	case TokenFalse:
		instance.advance()
		return &LiteralBoolCmd{Value: false}, nil

	case TokenQuestion:
		instance.advance()
		return &LiteralBoolCmd{Value: true}, nil

	default:
		if instance.isDeclaredName(token) {
			instance.advance()
			name := token.Value
			switch {
			case instance.program.routineSet[name]:
				return &RoutineCallCmd{Name: name}, nil
			case instance.program.groupingSet[name]:
				return &GroupingCmd{Name: name, Negated: false}, nil
			case instance.program.booleanSet[name]:
				return &BoolCmd{Op: TokenIdent, Name: name}, nil
			case instance.program.stringSet[name]:
				return &StringMatchCmd{IsVariable: true, Value: name}, nil
			default:
				return nil, fmt.Errorf("line %d, column %d: undeclared identifier %q", token.Line, token.Column, name)
			}
		}
		return nil, fmt.Errorf("line %d, column %d: unexpected token %q in command", token.Line, token.Column, token.Value)
	}
}

func (instance *parser) parseDollarCommand(backward bool) (Command, error) {
	instance.advance() // consume '$'
	if instance.peek().Kind == TokenLParen {
		instance.advance()
		left, err := instance.parseArithExpr()
		if err != nil {
			return nil, err
		}
		operator, right, err := instance.parseComparisonRHS()
		if err != nil {
			return nil, err
		}
		if err := instance.expect(TokenRParen); err != nil {
			return nil, err
		}
		return &IntTestCmd{Left: left, Op: operator, Right: right}, nil
	}

	nameToken, ok := instance.acceptName()
	if !ok {
		token := instance.peek()
		return nil, fmt.Errorf("line %d, column %d: expected variable name or '(' after $, got %q", token.Line, token.Column, token.Value)
	}
	name := nameToken.Value

	if instance.program.integerSet[name] {
		next := instance.peek()
		switch next.Kind {
		case TokenAssign, TokenPlusAssign, TokenSubAssign, TokenMulAssign, TokenDivAssign:
			instance.advance()
			expr, err := instance.parseArithExpr()
			if err != nil {
				return nil, err
			}
			return &IntAssignCmd{Name: name, Op: next.Kind, Expr: expr}, nil
		case TokenEq, TokenNotEq, TokenLess, TokenLessEq, TokenGreater, TokenGreaterEq, TokenSliceFrom:
			operator, right, err := instance.parseComparisonRHS()
			if err != nil {
				return nil, err
			}
			return &IntTestCmd{Left: &IntVarExpr{Name: name}, Op: operator, Right: right}, nil
		default:
			return nil, fmt.Errorf("line %d, column %d: expected integer operator after $%s, got %q", next.Line, next.Column, name, next.Value)
		}
	}

	if instance.program.stringSet[name] {
		body, err := instance.parseCommand(backward)
		if err != nil {
			return nil, err
		}
		return &StringScopeCmd{Name: name, Body: body}, nil
	}

	return nil, fmt.Errorf("line %d, column %d: undeclared variable $%s", nameToken.Line, nameToken.Column, name)
}

func (instance *parser) parseComparisonRHS() (TokenKind, ArithExpr, error) {
	token := instance.peek()
	switch token.Kind {
	case TokenEq, TokenNotEq, TokenLess, TokenLessEq, TokenGreater, TokenGreaterEq:
		instance.advance()
		right, err := instance.parseArithExpr()
		if err != nil {
			return TokenEOF, nil, err
		}
		return token.Kind, right, nil
	case TokenSliceFrom:
		// Snowball 3.2+ treats `<-` in an arithmetic test as `< -`.
		instance.advance()
		right, err := instance.parseArithUnary()
		if err != nil {
			return TokenEOF, nil, err
		}
		return TokenLess, &UnaryArithExpr{Expr: right}, nil
	default:
		return TokenEOF, nil, fmt.Errorf("line %d, column %d: expected comparison operator, got %q", token.Line, token.Column, token.Value)
	}
}

func (instance *parser) parseStringOperand() (bool, string, error) {
	token := instance.peek()
	if token.Kind == TokenString {
		instance.advance()
		return false, token.Value, nil
	}
	if instance.isDeclaredName(token) && instance.program.stringSet[token.Value] {
		instance.advance()
		return true, token.Value, nil
	}
	return false, "", fmt.Errorf("line %d, column %d: expected string literal or string variable, got %q", token.Line, token.Column, token.Value)
}

func (instance *parser) parseAmong(backward bool) (*AmongCmd, error) {
	instance.advance() // consume 'among'
	if err := instance.expect(TokenLParen); err != nil {
		return nil, err
	}

	instance.nextID++
	cmd := &AmongCmd{
		ID:       instance.nextID,
		Backward: backward,
	}

	// Optional starter command `(C)` before the first literal string.
	if instance.peek().Kind == TokenLParen {
		starter, err := instance.parsePrimary(backward)
		if err != nil {
			return nil, err
		}
		cmd.Starter = starter
	}

	var pendingEntries []AmongEntry
	nextResult := 1

	for instance.peek().Kind != TokenRParen && instance.peek().Kind != TokenEOF {
		token := instance.peek()
		if token.Kind == TokenString {
			instance.advance()
			entry := AmongEntry{Text: token.Value}
			if instance.isDeclaredName(instance.peek()) && instance.program.routineSet[instance.peek().Value] {
				entry.Condition = instance.advance().Value
			}
			pendingEntries = append(pendingEntries, entry)
			continue
		}
		if token.Kind == TokenLParen {
			body, err := instance.parsePrimary(backward)
			if err != nil {
				return nil, err
			}
			resultID := nextResult
			nextResult++
			for index := range pendingEntries {
				pendingEntries[index].Result = resultID
			}
			cmd.Entries = append(cmd.Entries, pendingEntries...)
			pendingEntries = pendingEntries[:0]
			cmd.Actions = append(cmd.Actions, AmongAction{Result: resultID, Body: body})
			continue
		}
		return nil, fmt.Errorf("line %d, column %d: unexpected token %q in among(...)", token.Line, token.Column, token.Value)
	}

	if len(pendingEntries) > 0 {
		resultID := nextResult
		for index := range pendingEntries {
			pendingEntries[index].Result = resultID
		}
		cmd.Entries = append(cmd.Entries, pendingEntries...)
	}

	if err := instance.expect(TokenRParen); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (instance *parser) parseArithExpr() (ArithExpr, error) {
	left, err := instance.parseArithTerm()
	if err != nil {
		return nil, err
	}
	for instance.peek().Kind == TokenPlus || instance.peek().Kind == TokenMinus {
		operator := instance.advance().Kind
		right, err := instance.parseArithTerm()
		if err != nil {
			return nil, err
		}
		left = &BinaryArithExpr{Op: operator, Left: left, Right: right}
	}
	return left, nil
}

func (instance *parser) parseArithTerm() (ArithExpr, error) {
	left, err := instance.parseArithUnary()
	if err != nil {
		return nil, err
	}
	for instance.peek().Kind == TokenStar || instance.peek().Kind == TokenSlash {
		operator := instance.advance().Kind
		right, err := instance.parseArithUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryArithExpr{Op: operator, Left: left, Right: right}
	}
	return left, nil
}

func (instance *parser) parseArithUnary() (ArithExpr, error) {
	if instance.peek().Kind == TokenMinus {
		instance.advance()
		expr, err := instance.parseArithUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryArithExpr{Expr: expr}, nil
	}
	return instance.parseArithPrimary()
}

func (instance *parser) parseArithPrimary() (ArithExpr, error) {
	token := instance.peek()
	switch token.Kind {
	case TokenLParen:
		instance.advance()
		expr, err := instance.parseArithExpr()
		if err != nil {
			return nil, err
		}
		if err := instance.expect(TokenRParen); err != nil {
			return nil, err
		}
		return expr, nil
	case TokenNumber:
		instance.advance()
		value, err := strconv.Atoi(token.Value)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid number %q: %w", token.Line, token.Value, err)
		}
		return &IntLiteralExpr{Value: value}, nil
	case TokenCursor, TokenLimit, TokenSize, TokenMaxInt, TokenMinInt:
		instance.advance()
		return &EnvMetricExpr{Kind: token.Kind}, nil
	case TokenLen:
		if instance.program.integerSet[token.Value] {
			instance.advance()
			return &IntVarExpr{Name: token.Value}, nil
		}
		instance.advance()
		return &EnvMetricExpr{Kind: TokenLen}, nil
	case TokenSizeOf, TokenLenOf:
		if instance.program.integerSet[token.Value] {
			instance.advance()
			return &IntVarExpr{Name: token.Value}, nil
		}
		instance.advance()
		isVar, value, err := instance.parseStringOperand()
		if err != nil {
			return nil, err
		}
		return &StringMetricExpr{Kind: token.Kind, IsVariable: isVar, Value: value}, nil
	default:
		if instance.isDeclaredName(token) && instance.program.integerSet[token.Value] {
			instance.advance()
			return &IntVarExpr{Name: token.Value}, nil
		}
		return nil, fmt.Errorf("line %d, column %d: unexpected token %q in arithmetic expression", token.Line, token.Column, token.Value)
	}
}

func linkSubstrings(root Command, backward bool) error {
	var pending *SubstringCmd
	var walk func(cmd Command, isBackward bool) error

	walk = func(cmd Command, isBackward bool) error {
		switch node := cmd.(type) {
		case *SequenceCmd:
			for _, child := range node.Commands {
				if err := walk(child, isBackward); err != nil {
					return err
				}
			}
		case *BinaryCmd:
			if err := walk(node.Left, isBackward); err != nil {
				return err
			}
			if err := walk(node.Right, isBackward); err != nil {
				return err
			}
		case *UnaryCmd:
			childBackward := isBackward
			switch node.Op {
			case TokenBackwards:
				childBackward = true
			case TokenReverse:
				childBackward = !isBackward
			}
			if err := walk(node.Body, childBackward); err != nil {
				return err
			}
		case *LoopCmd:
			if err := walk(node.Body, isBackward); err != nil {
				return err
			}
		case *SetLimitCmd:
			if err := walk(node.Limit, isBackward); err != nil {
				return err
			}
			if err := walk(node.Body, isBackward); err != nil {
				return err
			}
		case *StringScopeCmd:
			if err := walk(node.Body, false); err != nil {
				return err
			}
		case *SubstringCmd:
			if pending != nil {
				return fmt.Errorf("unmatched substring before another substring")
			}
			pending = node
		case *AmongCmd:
			node.Backward = isBackward
			if pending != nil {
				pending.AmongID = node.ID
				node.HasSubstring = true
				pending = nil
			}
			if node.Starter != nil {
				if err := walk(node.Starter, isBackward); err != nil {
					return err
				}
			}
			for _, action := range node.Actions {
				if err := walk(action.Body, isBackward); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := walk(root, backward); err != nil {
		return err
	}
	if pending != nil {
		return fmt.Errorf("substring without matching among(...)")
	}
	return nil
}
