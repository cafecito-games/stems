// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package sbl

import (
	"bytes"
	"cmp"
	"fmt"
	"go/format"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// CompileOptions configures Go code emission for a Snowball program.
type CompileOptions struct {
	PackageName     string
	LanguageDisplay string
	SourceFile      string
	StopWords       []string
	TurkishCase     bool
	NormalizeYo     bool
}

type preparedAmong struct {
	cmd     *AmongCmd
	entries []compiledAmongEntry
}

type compiledAmongEntry struct {
	text           string
	substringIndex int
	result         int
	condition      string
}

type emitter struct {
	program      *Program
	options      CompileOptions
	buf          bytes.Buffer
	amongs       map[int]*preparedAmong
	amongOrder   []int
	routineNames map[string]string
	groupNames   map[string]string
	intNames     map[string]string
	boolNames    map[string]string
	stringNames  map[string]string
	usedNames    map[string]bool
	tempCounter  int
	labelCounter int
}

// Compile generates formatted Go 1.26 source code for program.
func Compile(program *Program, options CompileOptions) ([]byte, error) {
	gen := &emitter{
		program:      program,
		options:      options,
		amongs:       make(map[int]*preparedAmong),
		routineNames: make(map[string]string),
		groupNames:   make(map[string]string),
		intNames:     make(map[string]string),
		boolNames:    make(map[string]string),
		stringNames:  make(map[string]string),
		usedNames:    make(map[string]bool),
	}

	gen.assignNames()
	gen.collectAmongs()
	gen.emitFile()

	formatted, err := format.Source(gen.buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated Go for %s: %w\n%s", options.PackageName, err, gen.buf.String())
	}
	return formatted, nil
}

func (gen *emitter) allocateName(prefix, original string) string {
	var cleaned strings.Builder
	capitalizeNext := true
	for _, character := range original {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if capitalizeNext {
				cleaned.WriteRune(unicode.ToUpper(character))
				capitalizeNext = false
			} else {
				cleaned.WriteRune(character)
			}
		} else {
			capitalizeNext = true
		}
	}
	base := prefix + cleaned.String()
	if base == prefix {
		base = prefix + "Sym"
	}
	candidate := base
	suffix := 2
	for gen.usedNames[candidate] {
		candidate = fmt.Sprintf("%s%d", base, suffix)
		suffix++
	}
	gen.usedNames[candidate] = true
	return candidate
}

func (gen *emitter) assignNames() {
	for _, name := range gen.program.Groupings {
		gen.groupNames[name] = gen.allocateName("group", name)
	}
	for _, name := range gen.program.Integers {
		gen.intNames[name] = gen.allocateName("int", name)
	}
	for _, name := range gen.program.Booleans {
		gen.boolNames[name] = gen.allocateName("bool", name)
	}
	for _, name := range gen.program.Strings {
		gen.stringNames[name] = gen.allocateName("str", name)
	}
	for _, routine := range gen.program.RoutineDefs {
		gen.routineNames[routine.Name] = gen.allocateName("routine", routine.Name)
	}
}

func (gen *emitter) collectAmongs() {
	var walk func(cmd Command)
	walk = func(cmd Command) {
		switch node := cmd.(type) {
		case *SequenceCmd:
			for _, child := range node.Commands {
				walk(child)
			}
		case *BinaryCmd:
			walk(node.Left)
			walk(node.Right)
		case *UnaryCmd:
			walk(node.Body)
		case *LoopCmd:
			walk(node.Body)
		case *SetLimitCmd:
			walk(node.Limit)
			walk(node.Body)
		case *StringScopeCmd:
			walk(node.Body)
		case *AmongCmd:
			prepared := prepareAmong(node)
			gen.amongs[node.ID] = prepared
			gen.amongOrder = append(gen.amongOrder, node.ID)
			if node.Starter != nil {
				walk(node.Starter)
			}
			for _, action := range node.Actions {
				walk(action.Body)
			}
		}
	}

	for _, routine := range gen.program.RoutineDefs {
		walk(routine.Body)
	}
}

func prepareAmong(cmd *AmongCmd) *preparedAmong {
	entries := make([]compiledAmongEntry, len(cmd.Entries))
	for index, entry := range cmd.Entries {
		entries[index] = compiledAmongEntry{
			text:      entry.Text,
			result:    entry.Result,
			condition: entry.Condition,
		}
	}

	if cmd.Backward {
		slices.SortStableFunc(entries, func(left, right compiledAmongEntry) int {
			return compareReversedBytes(left.text, right.text)
		})
	} else {
		slices.SortStableFunc(entries, func(left, right compiledAmongEntry) int {
			return cmp.Compare(left.text, right.text)
		})
	}

	// Compute Snowball SubstringIndex: longest strict prefix (forward) or suffix (backward).
	indexByText := make(map[string]int, len(entries))
	for index, entry := range entries {
		indexByText[entry.text] = index
	}

	for index := range entries {
		entries[index].substringIndex = -1
		text := entries[index].text
		if cmd.Backward {
			for start := 1; start <= len(text); start++ {
				if matchIndex, ok := indexByText[text[start:]]; ok && matchIndex != index {
					entries[index].substringIndex = matchIndex
					break
				}
			}
		} else {
			for end := len(text) - 1; end >= 0; end-- {
				if matchIndex, ok := indexByText[text[:end]]; ok && matchIndex != index {
					entries[index].substringIndex = matchIndex
					break
				}
			}
		}
	}

	return &preparedAmong{
		cmd:     cmd,
		entries: entries,
	}
}

func compareReversedBytes(left, right string) int {
	leftIndex := len(left) - 1
	rightIndex := len(right) - 1
	for leftIndex >= 0 && rightIndex >= 0 {
		if left[leftIndex] != right[rightIndex] {
			if left[leftIndex] < right[rightIndex] {
				return -1
			}
			return 1
		}
		leftIndex--
		rightIndex--
	}
	return cmp.Compare(len(left), len(right))
}

func (gen *emitter) printf(formatText string, arguments ...any) {
	fmt.Fprintf(&gen.buf, formatText, arguments...)
}

func (gen *emitter) nextTemp(prefix string) string {
	gen.tempCounter++
	return fmt.Sprintf("%s%d", prefix, gen.tempCounter)
}

func (gen *emitter) nextLabel(prefix string) string {
	gen.labelCounter++
	return fmt.Sprintf("%s%d", prefix, gen.labelCounter)
}

func (gen *emitter) emitFile() {
	gen.printf("// Copyright (c) 2026 Cafecito Games LLC\n")
	gen.printf("// SPDX-License-Identifier: MIT\n\n")
	gen.printf("// Code generated by sblgen from %s; DO NOT EDIT.\n\n", gen.options.SourceFile)
	gen.printf("// Package %s implements the %s Snowball stemming algorithm.\n", gen.options.PackageName, gen.options.LanguageDisplay)
	gen.printf("package %s\n\n", gen.options.PackageName)

	usesMath := gen.usesMinMaxInt()
	usesUnicode := gen.options.TurkishCase
	usesStrings := gen.options.TurkishCase || gen.options.NormalizeYo
	usesSlices := len(gen.options.StopWords) > 0
	usesUTF8 := gen.usesLenOf()

	gen.printf("import (\n")
	if usesMath {
		gen.printf("\t\"math\"\n")
	}
	if usesSlices {
		gen.printf("\t\"slices\"\n")
	}
	if usesStrings {
		gen.printf("\t\"strings\"\n")
	}
	if usesUnicode {
		gen.printf("\t\"unicode\"\n")
	}
	if usesUTF8 {
		gen.printf("\t\"unicode/utf8\"\n")
	}
	gen.printf("\n\t\"github.com/cafecito-games/stems/internal/snowball\"\n")
	gen.printf(")\n\n")

	gen.emitGroupings()
	gen.emitStaticAmongs()
	gen.emitStopWords()
	gen.emitStemmerType()
	gen.emitPublicAPI()
	gen.emitRoutines()
}

func (gen *emitter) usesMinMaxInt() bool {
	found := false
	var walkArith func(expr ArithExpr)
	walkArith = func(expr ArithExpr) {
		switch node := expr.(type) {
		case *EnvMetricExpr:
			if node.Kind == TokenMaxInt || node.Kind == TokenMinInt {
				found = true
			}
		case *UnaryArithExpr:
			walkArith(node.Expr)
		case *BinaryArithExpr:
			walkArith(node.Left)
			walkArith(node.Right)
		}
	}
	var walkCmd func(cmd Command)
	walkCmd = func(cmd Command) {
		switch node := cmd.(type) {
		case *SequenceCmd:
			for _, child := range node.Commands {
				walkCmd(child)
			}
		case *BinaryCmd:
			walkCmd(node.Left)
			walkCmd(node.Right)
		case *UnaryCmd:
			walkCmd(node.Body)
		case *LoopCmd:
			walkArith(node.Count)
			walkCmd(node.Body)
		case *SetLimitCmd:
			walkCmd(node.Limit)
			walkCmd(node.Body)
		case *HopCmd:
			walkArith(node.Count)
		case *MarkCmd:
			if node.Expr != nil {
				walkArith(node.Expr)
			}
		case *IntAssignCmd:
			walkArith(node.Expr)
		case *IntTestCmd:
			walkArith(node.Left)
			walkArith(node.Right)
		case *StringScopeCmd:
			walkCmd(node.Body)
		case *AmongCmd:
			if node.Starter != nil {
				walkCmd(node.Starter)
			}
			for _, action := range node.Actions {
				walkCmd(action.Body)
			}
		}
	}
	for _, routine := range gen.program.RoutineDefs {
		walkCmd(routine.Body)
	}
	return found
}

func (gen *emitter) usesLenOf() bool {
	found := false
	var walkArith func(expr ArithExpr)
	walkArith = func(expr ArithExpr) {
		switch node := expr.(type) {
		case *StringMetricExpr:
			if node.Kind == TokenLenOf {
				found = true
			}
		case *UnaryArithExpr:
			walkArith(node.Expr)
		case *BinaryArithExpr:
			walkArith(node.Left)
			walkArith(node.Right)
		}
	}
	var walkCmd func(cmd Command)
	walkCmd = func(cmd Command) {
		switch node := cmd.(type) {
		case *SequenceCmd:
			for _, child := range node.Commands {
				walkCmd(child)
			}
		case *BinaryCmd:
			walkCmd(node.Left)
			walkCmd(node.Right)
		case *UnaryCmd:
			walkCmd(node.Body)
		case *LoopCmd:
			walkArith(node.Count)
			walkCmd(node.Body)
		case *SetLimitCmd:
			walkCmd(node.Limit)
			walkCmd(node.Body)
		case *HopCmd:
			walkArith(node.Count)
		case *MarkCmd:
			if node.Expr != nil {
				walkArith(node.Expr)
			}
		case *IntAssignCmd:
			walkArith(node.Expr)
		case *IntTestCmd:
			walkArith(node.Left)
			walkArith(node.Right)
		case *StringScopeCmd:
			walkCmd(node.Body)
		case *AmongCmd:
			if node.Starter != nil {
				walkCmd(node.Starter)
			}
			for _, action := range node.Actions {
				walkCmd(action.Body)
			}
		}
	}
	for _, routine := range gen.program.RoutineDefs {
		walkCmd(routine.Body)
	}
	return found
}

func (gen *emitter) emitGroupings() {
	if len(gen.program.Groupings) == 0 {
		return
	}
	gen.printf("var (\n")
	for _, name := range gen.program.Groupings {
		runes := gen.program.GroupingDefs[name]
		gen.printf("\t%s = snowball.NewGrouping(%s)\n", gen.groupNames[name], strconv.Quote(string(runes)))
	}
	gen.printf(")\n\n")
}

func (gen *emitter) emitStaticAmongs() {
	if len(gen.amongOrder) == 0 {
		return
	}

	gen.printf("var (\n")
	for _, id := range gen.amongOrder {
		prepared := gen.amongs[id]
		gen.printf("\tamongTable%d = []snowball.Among[stemmer]{\n", id)
		for _, entry := range prepared.entries {
			if entry.condition != "" {
				gen.printf("\t\t{Text: %s, SubstringIndex: %d, Result: %d, Condition: (*stemmer).%s},\n",
					strconv.Quote(entry.text), entry.substringIndex, entry.result, gen.routineNames[entry.condition])
			} else {
				gen.printf("\t\t{Text: %s, SubstringIndex: %d, Result: %d},\n",
					strconv.Quote(entry.text), entry.substringIndex, entry.result)
			}
		}
		gen.printf("\t}\n")
	}
	gen.printf(")\n\n")
}

func (gen *emitter) emitStopWords() {
	if len(gen.options.StopWords) == 0 {
		return
	}
	gen.printf("var stopWordList = []string{\n")
	for _, word := range gen.options.StopWords {
		gen.printf("\t%s,\n", strconv.Quote(word))
	}
	gen.printf("}\n\n")

	gen.printf("var stopWordSet = map[string]struct{}{\n")
	for _, word := range gen.options.StopWords {
		gen.printf("\t%s: {},\n", strconv.Quote(word))
	}
	gen.printf("}\n\n")
}

func (gen *emitter) emitStemmerType() {
	gen.printf("type stemmer struct {\n")
	gen.printf("\tenv snowball.Env\n")
	for _, name := range gen.program.Integers {
		gen.printf("\t%s int\n", gen.intNames[name])
	}
	for _, name := range gen.program.Booleans {
		gen.printf("\t%s bool\n", gen.boolNames[name])
	}
	for _, name := range gen.program.Strings {
		gen.printf("\t%s string\n", gen.stringNames[name])
	}
	gen.printf("}\n\n")
}

func (gen *emitter) emitPublicAPI() {
	stemRoutine := "stem"
	if len(gen.program.Externals) > 0 {
		stemRoutine = gen.program.Externals[0]
	}

	gen.printf("// Stem returns the %s Snowball stem of input.\n", gen.options.LanguageDisplay)
	gen.printf("func Stem(input string) string {\n")
	if gen.options.TurkishCase {
		gen.printf("\tnormalized := strings.ToLowerSpecial(unicode.TurkishCase, strings.TrimSpace(input))\n")
	} else {
		gen.printf("\tnormalized := snowball.Normalize(input)\n")
	}
	gen.printf("\treturn stemRaw(normalized)\n")
	gen.printf("}\n\n")

	gen.printf("func stemRaw(input string) string {\n")
	gen.printf("\tvar instance stemmer\n")
	gen.printf("\tvar buffer [snowball.InlineBufferCapacity]byte\n")
	gen.printf("\tinstance.env.InitWithBuffer(input, buffer[:])\n")
	gen.printf("\tinstance.%s()\n", gen.routineNames[stemRoutine])
	gen.printf("\treturn instance.env.Result(input)\n")
	gen.printf("}\n\n")

	gen.printf("// IsStopWord reports whether input is a %s stop word.\n", gen.options.LanguageDisplay)
	gen.printf("func IsStopWord(input string) bool {\n")
	if len(gen.options.StopWords) == 0 {
		gen.printf("\t_ = input\n")
		gen.printf("\treturn false\n")
	} else {
		if gen.options.TurkishCase {
			gen.printf("\tnormalized := strings.ToLowerSpecial(unicode.TurkishCase, strings.TrimSpace(input))\n")
		} else {
			gen.printf("\tnormalized := snowball.Normalize(input)\n")
		}
		if gen.options.NormalizeYo {
			gen.printf("\tnormalized = strings.ReplaceAll(normalized, \"ё\", \"е\")\n")
		}
		gen.printf("\t_, found := stopWordSet[normalized]\n")
		gen.printf("\treturn found\n")
	}
	gen.printf("}\n\n")

	gen.printf("// StopWords returns a sorted copy of the %s stop word list.\n", gen.options.LanguageDisplay)
	gen.printf("func StopWords() []string {\n")
	if len(gen.options.StopWords) == 0 {
		gen.printf("\treturn nil\n")
	} else {
		gen.printf("\treturn slices.Clone(stopWordList)\n")
	}
	gen.printf("}\n\n")
}

func (gen *emitter) emitRoutines() {
	for _, routine := range gen.program.RoutineDefs {
		gen.tempCounter = 0
		gen.labelCounter = 0
		gen.printf("func (instance *stemmer) %s() bool {\n", gen.routineNames[routine.Name])
		if containsSubstringOrAmong(routine.Body) {
			gen.printf("\tvar amongVar int\n")
			gen.printf("\t_ = amongVar\n")
		}
		failStmt := "return false"
		gen.emitCommand(routine.Body, routine.Backward, failStmt, "\t")
		gen.printf("\treturn true\n")
		gen.printf("}\n\n")
	}
}

func containsSubstringOrAmong(cmd Command) bool {
	switch node := cmd.(type) {
	case *SequenceCmd:
		return slices.ContainsFunc(node.Commands, containsSubstringOrAmong)
	case *BinaryCmd:
		return containsSubstringOrAmong(node.Left) || containsSubstringOrAmong(node.Right)
	case *UnaryCmd:
		return containsSubstringOrAmong(node.Body)
	case *LoopCmd:
		return containsSubstringOrAmong(node.Body)
	case *SetLimitCmd:
		return containsSubstringOrAmong(node.Limit) || containsSubstringOrAmong(node.Body)
	case *StringScopeCmd:
		return containsSubstringOrAmong(node.Body)
	case *SubstringCmd, *AmongCmd:
		return true
	}
	return false
}

func (gen *emitter) emitSaveCursor(varName string, backward bool, indent string) {
	if backward {
		gen.printf("%s%s := instance.env.Limit - instance.env.Cursor\n", indent, varName)
	} else {
		gen.printf("%s%s := instance.env.Cursor\n", indent, varName)
	}
}

func (gen *emitter) emitRestoreCursor(varName string, backward bool, indent string) {
	if backward {
		gen.printf("%sinstance.env.Cursor = instance.env.Limit - %s\n", indent, varName)
	} else {
		gen.printf("%sinstance.env.Cursor = %s\n", indent, varName)
	}
}

func (gen *emitter) emitSubcommandDiscard(cmd Command, backward bool, indent string) {
	blockLabel := gen.nextLabel("block")
	gen.printf("%s%s:\n", indent, blockLabel)
	gen.printf("%sfor {\n", indent)
	gen.emitCommand(cmd, backward, fmt.Sprintf("break %s", blockLabel), indent+"\t")
	gen.printf("%s\tbreak %s\n", indent, blockLabel)
	gen.printf("%s}\n", indent)
}

func (gen *emitter) emitSubcommandAsBool(cmd Command, backward bool, indent string) string {
	resultVar := gen.nextTemp("ok")
	blockLabel := gen.nextLabel("block")
	gen.printf("%s%s := true\n", indent, resultVar)
	gen.printf("%s%s:\n", indent, blockLabel)
	gen.printf("%sfor {\n", indent)
	gen.emitCommand(cmd, backward, fmt.Sprintf("%s = false; break %s", resultVar, blockLabel), indent+"\t")
	gen.printf("%s\tbreak %s\n", indent, blockLabel)
	gen.printf("%s}\n", indent)
	return resultVar
}

func (gen *emitter) emitCommand(cmd Command, backward bool, onFail string, indent string) {
	switch node := cmd.(type) {
	case *SequenceCmd:
		for _, child := range node.Commands {
			gen.emitCommand(child, backward, onFail, indent)
		}

	case *LiteralBoolCmd:
		if !node.Value {
			gen.printf("%s%s\n", indent, onFail)
		}

	case *BinaryCmd:
		if node.Op == TokenOr {
			saved := gen.nextTemp("saved")
			gen.emitSaveCursor(saved, backward, indent)
			leftOK := gen.emitSubcommandAsBool(node.Left, backward, indent)
			gen.printf("%sif !%s {\n", indent, leftOK)
			gen.emitRestoreCursor(saved, backward, indent+"\t")
			gen.emitCommand(node.Right, backward, onFail, indent+"\t")
			gen.printf("%s}\n", indent)
			return
		}
		// TokenAnd
		saved := gen.nextTemp("saved")
		gen.emitSaveCursor(saved, backward, indent)
		gen.emitCommand(node.Left, backward, onFail, indent)
		gen.emitRestoreCursor(saved, backward, indent)
		gen.emitCommand(node.Right, backward, onFail, indent)

	case *UnaryCmd:
		switch node.Op {
		case TokenNot:
			saved := gen.nextTemp("saved")
			gen.emitSaveCursor(saved, backward, indent)
			bodyOK := gen.emitSubcommandAsBool(node.Body, backward, indent)
			gen.printf("%sif %s {\n", indent, bodyOK)
			gen.printf("%s\t%s\n", indent, onFail)
			gen.printf("%s}\n", indent)
			gen.emitRestoreCursor(saved, backward, indent)

		case TokenTest:
			saved := gen.nextTemp("saved")
			gen.emitSaveCursor(saved, backward, indent)
			gen.emitCommand(node.Body, backward, onFail, indent)
			gen.emitRestoreCursor(saved, backward, indent)

		case TokenTry:
			saved := gen.nextTemp("saved")
			gen.emitSaveCursor(saved, backward, indent)
			bodyOK := gen.emitSubcommandAsBool(node.Body, backward, indent)
			gen.printf("%sif !%s {\n", indent, bodyOK)
			gen.emitRestoreCursor(saved, backward, indent+"\t")
			gen.printf("%s}\n", indent)

		case TokenDo:
			saved := gen.nextTemp("saved")
			gen.emitSaveCursor(saved, backward, indent)
			gen.emitSubcommandDiscard(node.Body, backward, indent)
			gen.emitRestoreCursor(saved, backward, indent)

		case TokenFail:
			gen.emitSubcommandDiscard(node.Body, backward, indent)
			gen.printf("%s%s\n", indent, onFail)

		case TokenRepeat:
			loopLabel := gen.nextLabel("repeat")
			gen.printf("%s%s:\n", indent, loopLabel)
			gen.printf("%sfor {\n", indent)
			saved := gen.nextTemp("saved")
			gen.emitSaveCursor(saved, backward, indent+"\t")
			bodyOK := gen.emitSubcommandAsBool(node.Body, backward, indent+"\t")
			gen.printf("%s\tif !%s {\n", indent, bodyOK)
			gen.emitRestoreCursor(saved, backward, indent+"\t\t")
			gen.printf("%s\t\tbreak %s\n", indent, loopLabel)
			gen.printf("%s\t}\n", indent)
			gen.printf("%s}\n", indent)

		case TokenGoTo, TokenGoPast:
			loopLabel := gen.nextLabel("goto")
			gen.printf("%s%s:\n", indent, loopLabel)
			gen.printf("%sfor {\n", indent)
			saved := gen.nextTemp("saved")
			gen.emitSaveCursor(saved, backward, indent+"\t")
			bodyOK := gen.emitSubcommandAsBool(node.Body, backward, indent+"\t")
			gen.printf("%s\tif %s {\n", indent, bodyOK)
			if node.Op == TokenGoTo {
				gen.emitRestoreCursor(saved, backward, indent+"\t\t")
			}
			gen.printf("%s\t\tbreak %s\n", indent, loopLabel)
			gen.printf("%s\t}\n", indent)
			gen.emitRestoreCursor(saved, backward, indent+"\t")
			if backward {
				gen.printf("%s\tif !instance.env.HopBackward(1) {\n", indent)
			} else {
				gen.printf("%s\tif !instance.env.Hop(1) {\n", indent)
			}
			gen.printf("%s\t\t%s\n", indent, onFail)
			gen.printf("%s\t}\n", indent)
			gen.printf("%s}\n", indent)

		case TokenBackwards:
			savedLB := gen.nextTemp("savedLimitBackward")
			gen.printf("%s%s := instance.env.LimitBackward\n", indent, savedLB)
			gen.printf("%sinstance.env.LimitBackward = instance.env.Cursor\n", indent)
			gen.printf("%sinstance.env.Cursor = instance.env.Limit\n", indent)
			bodyOK := gen.emitSubcommandAsBool(node.Body, true, indent)
			gen.printf("%sinstance.env.Cursor = instance.env.LimitBackward\n", indent)
			gen.printf("%sinstance.env.LimitBackward = %s\n", indent, savedLB)
			gen.printf("%sif !%s {\n", indent, bodyOK)
			gen.printf("%s\t%s\n", indent, onFail)
			gen.printf("%s}\n", indent)

		case TokenReverse:
			gen.emitCommand(node.Body, !backward, onFail, indent)
		}

	case *LoopCmd:
		countVar := gen.nextTemp("count")
		gen.printf("%s%s := %s\n", indent, countVar, gen.formatArith(node.Count))
		if node.AtLeast {
			loopLabel := gen.nextLabel("atleast")
			gen.printf("%s%s:\n", indent, loopLabel)
			gen.printf("%sfor {\n", indent)
			saved := gen.nextTemp("saved")
			gen.emitSaveCursor(saved, backward, indent+"\t")
			bodyOK := gen.emitSubcommandAsBool(node.Body, backward, indent+"\t")
			gen.printf("%s\tif !%s {\n", indent, bodyOK)
			gen.emitRestoreCursor(saved, backward, indent+"\t\t")
			gen.printf("%s\t\tbreak %s\n", indent, loopLabel)
			gen.printf("%s\t}\n", indent)
			gen.printf("%s\t%s--\n", indent, countVar)
			gen.printf("%s}\n", indent)
			gen.printf("%sif %s > 0 {\n", indent, countVar)
			gen.printf("%s\t%s\n", indent, onFail)
			gen.printf("%s}\n", indent)
			return
		}
		gen.printf("%sfor ; %s > 0; %s-- {\n", indent, countVar, countVar)
		gen.emitCommand(node.Body, backward, onFail, indent+"\t")
		gen.printf("%s}\n", indent)

	case *SetLimitCmd:
		saved := gen.nextTemp("saved")
		gen.emitSaveCursor(saved, backward, indent)
		limitOK := gen.emitSubcommandAsBool(node.Limit, backward, indent)
		gen.printf("%sif !%s {\n", indent, limitOK)
		gen.emitRestoreCursor(saved, backward, indent+"\t")
		gen.printf("%s\t%s\n", indent, onFail)
		gen.printf("%s}\n", indent)
		if backward {
			savedLB := gen.nextTemp("savedLimitBackward")
			gen.printf("%s%s := instance.env.LimitBackward\n", indent, savedLB)
			gen.printf("%sinstance.env.LimitBackward = instance.env.Cursor\n", indent)
			gen.emitRestoreCursor(saved, true, indent)
			bodyOK := gen.emitSubcommandAsBool(node.Body, true, indent)
			gen.printf("%sinstance.env.LimitBackward = %s\n", indent, savedLB)
			gen.printf("%sif !%s {\n", indent, bodyOK)
			gen.printf("%s\t%s\n", indent, onFail)
			gen.printf("%s}\n", indent)
		} else {
			savedDelta := gen.nextTemp("savedLimitDelta")
			gen.printf("%s%s := instance.env.Limit - instance.env.Cursor\n", indent, savedDelta)
			gen.printf("%sinstance.env.Limit = instance.env.Cursor\n", indent)
			gen.emitRestoreCursor(saved, false, indent)
			bodyOK := gen.emitSubcommandAsBool(node.Body, false, indent)
			gen.printf("%sinstance.env.Limit += %s\n", indent, savedDelta)
			gen.printf("%sif !%s {\n", indent, bodyOK)
			gen.printf("%s\t%s\n", indent, onFail)
			gen.printf("%s}\n", indent)
		}

	case *StringMatchCmd:
		operand := gen.formatStringOperand(node.IsVariable, node.Value)
		if backward {
			gen.printf("%sif !instance.env.EqSBackward(%s) {\n", indent, operand)
		} else {
			gen.printf("%sif !instance.env.EqS(%s) {\n", indent, operand)
		}
		gen.printf("%s\t%s\n", indent, onFail)
		gen.printf("%s}\n", indent)

	case *GroupingCmd:
		groupExpr := "&" + gen.groupNames[node.Name]
		method := "InGrouping"
		switch {
		case backward && node.Negated:
			method = "OutGroupingBackward"
		case backward && !node.Negated:
			method = "InGroupingBackward"
		case !backward && node.Negated:
			method = "OutGrouping"
		}
		gen.printf("%sif !instance.env.%s(%s) {\n", indent, method, groupExpr)
		gen.printf("%s\t%s\n", indent, onFail)
		gen.printf("%s}\n", indent)

	case *RoutineCallCmd:
		gen.printf("%sif !instance.%s() {\n", indent, gen.routineNames[node.Name])
		gen.printf("%s\t%s\n", indent, onFail)
		gen.printf("%s}\n", indent)

	case *BoolCmd:
		field := "instance." + gen.boolNames[node.Name]
		switch node.Op {
		case TokenSet:
			gen.printf("%s%s = true\n", indent, field)
		case TokenUnset:
			gen.printf("%s%s = false\n", indent, field)
		default:
			gen.printf("%sif !%s {\n", indent, field)
			gen.printf("%s\t%s\n", indent, onFail)
			gen.printf("%s}\n", indent)
		}

	case *SliceBoundCmd:
		if backward {
			if node.IsClose {
				gen.printf("%sinstance.env.Bra = instance.env.Cursor\n", indent)
			} else {
				gen.printf("%sinstance.env.Ket = instance.env.Cursor\n", indent)
			}
		} else {
			if node.IsClose {
				gen.printf("%sinstance.env.Ket = instance.env.Cursor\n", indent)
			} else {
				gen.printf("%sinstance.env.Bra = instance.env.Cursor\n", indent)
			}
		}

	case *SliceEditCmd:
		switch node.Op {
		case TokenDelete:
			gen.printf("%sinstance.env.SliceDel()\n", indent)
		case TokenSliceFrom:
			gen.printf("%sinstance.env.SliceFrom(%s)\n", indent, gen.formatStringOperand(node.IsVariable, node.Value))
		case TokenSliceTo:
			gen.printf("%sinstance.%s = instance.env.SliceTo()\n", indent, gen.stringNames[node.Value])
		case TokenInsert:
			gen.printf("%sinstance.env.Insert(%s, %t)\n", indent, gen.formatStringOperand(node.IsVariable, node.Value), backward)
		case TokenAttach:
			gen.printf("%sinstance.env.Attach(%s, %t)\n", indent, gen.formatStringOperand(node.IsVariable, node.Value), backward)
		}

	case *HopCmd:
		if backward {
			gen.printf("%sif !instance.env.HopBackward(%s) {\n", indent, gen.formatArith(node.Count))
		} else {
			gen.printf("%sif !instance.env.Hop(%s) {\n", indent, gen.formatArith(node.Count))
		}
		gen.printf("%s\t%s\n", indent, onFail)
		gen.printf("%s}\n", indent)

	case *MarkCmd:
		switch node.Op {
		case TokenSetMark:
			gen.printf("%sinstance.%s = instance.env.Cursor\n", indent, gen.intNames[node.Name])
		case TokenToLimit:
			if backward {
				gen.printf("%sinstance.env.Cursor = instance.env.LimitBackward\n", indent)
			} else {
				gen.printf("%sinstance.env.Cursor = instance.env.Limit\n", indent)
			}
		case TokenAtLimit:
			if backward {
				gen.printf("%sif instance.env.Cursor != instance.env.LimitBackward {\n", indent)
			} else {
				gen.printf("%sif instance.env.Cursor != instance.env.Limit {\n", indent)
			}
			gen.printf("%s\t%s\n", indent, onFail)
			gen.printf("%s}\n", indent)
		case TokenAtMark:
			gen.printf("%sif instance.env.Cursor != %s {\n", indent, gen.formatArith(node.Expr))
			gen.printf("%s\t%s\n", indent, onFail)
			gen.printf("%s}\n", indent)
		case TokenToMark:
			targetVar := gen.nextTemp("mark")
			gen.printf("%s%s := %s\n", indent, targetVar, gen.formatArith(node.Expr))
			if backward {
				gen.printf("%sif instance.env.Cursor < %s || %s < instance.env.LimitBackward || %s > instance.env.Limit {\n", indent, targetVar, targetVar, targetVar)
			} else {
				gen.printf("%sif instance.env.Cursor > %s || %s > instance.env.Limit || %s < instance.env.LimitBackward {\n", indent, targetVar, targetVar, targetVar)
			}
			gen.printf("%s\t%s\n", indent, onFail)
			gen.printf("%s}\n", indent)
			gen.printf("%sinstance.env.Cursor = %s\n", indent, targetVar)
		}

	case *IntAssignCmd:
		field := "instance." + gen.intNames[node.Name]
		op := "="
		switch node.Op {
		case TokenPlusAssign:
			op = "+="
		case TokenSubAssign:
			op = "-="
		case TokenMulAssign:
			op = "*="
		case TokenDivAssign:
			op = "/="
		}
		gen.printf("%s%s %s %s\n", indent, field, op, gen.formatArith(node.Expr))

	case *IntTestCmd:
		op := "=="
		switch node.Op {
		case TokenNotEq:
			op = "!="
		case TokenLess:
			op = "<"
		case TokenLessEq:
			op = "<="
		case TokenGreater:
			op = ">"
		case TokenGreaterEq:
			op = ">="
		}
		gen.printf("%sif !((%s) %s (%s)) {\n", indent, gen.formatArith(node.Left), op, gen.formatArith(node.Right))
		gen.printf("%s\t%s\n", indent, onFail)
		gen.printf("%s}\n", indent)

	case *StringAssignCmd:
		if node.Name == "" {
			gen.printf("%sinstance.env.ReplaceS(instance.env.Cursor, instance.env.Limit, %s)\n",
				indent, gen.formatStringOperand(node.IsVariable, node.Value))
		} else {
			gen.printf("%sinstance.%s = instance.env.String()\n", indent, gen.stringNames[node.Name])
		}

	case *StringScopeCmd:
		savedEnv := gen.nextTemp("savedEnv")
		gen.printf("%s%s := instance.env\n", indent, savedEnv)
		gen.printf("%sinstance.env.Init(instance.%s)\n", indent, gen.stringNames[node.Name])
		bodyOK := gen.emitSubcommandAsBool(node.Body, false, indent)
		gen.printf("%sinstance.%s = instance.env.String()\n", indent, gen.stringNames[node.Name])
		gen.printf("%sinstance.env = %s\n", indent, savedEnv)
		gen.printf("%sif !%s {\n", indent, bodyOK)
		gen.printf("%s\t%s\n", indent, onFail)
		gen.printf("%s}\n", indent)

	case *SubstringCmd:
		gen.emitFindAmong(node.AmongID, backward, onFail, indent)

	case *AmongCmd:
		if !node.HasSubstring {
			gen.emitFindAmong(node.ID, backward, onFail, indent)
		}
		if node.Starter != nil {
			gen.emitCommand(node.Starter, backward, onFail, indent)
		}
		if len(node.Actions) > 0 {
			gen.printf("%sswitch amongVar {\n", indent)
			for _, action := range node.Actions {
				gen.printf("%scase %d:\n", indent, action.Result)
				gen.emitCommand(action.Body, backward, onFail, indent+"\t")
			}
			gen.printf("%s}\n", indent)
		}
	}
}

func (gen *emitter) emitFindAmong(amongID int, backward bool, onFail string, indent string) {
	tableExpr := fmt.Sprintf("amongTable%d", amongID)
	if backward {
		gen.printf("%samongVar = snowball.FindAmongBackward(&instance.env, %s, instance)\n", indent, tableExpr)
	} else {
		gen.printf("%samongVar = snowball.FindAmong(&instance.env, %s, instance)\n", indent, tableExpr)
	}
	gen.printf("%sif amongVar == 0 {\n", indent)
	gen.printf("%s\t%s\n", indent, onFail)
	gen.printf("%s}\n", indent)
}

func (gen *emitter) formatStringOperand(isVariable bool, value string) string {
	if isVariable {
		return "instance." + gen.stringNames[value]
	}
	return strconv.Quote(value)
}

func (gen *emitter) formatArith(expr ArithExpr) string {
	switch node := expr.(type) {
	case *IntLiteralExpr:
		return strconv.Itoa(node.Value)
	case *IntVarExpr:
		return "instance." + gen.intNames[node.Name]
	case *EnvMetricExpr:
		switch node.Kind {
		case TokenCursor:
			return "instance.env.Cursor"
		case TokenLimit:
			return "instance.env.Limit"
		case TokenSize:
			return "instance.env.Limit"
		case TokenLen:
			return "instance.env.RuneCount()"
		case TokenMaxInt:
			return "math.MaxInt"
		case TokenMinInt:
			return "math.MinInt"
		}
	case *StringMetricExpr:
		operand := gen.formatStringOperand(node.IsVariable, node.Value)
		if node.Kind == TokenLenOf {
			return fmt.Sprintf("utf8.RuneCountInString(%s)", operand)
		}
		return fmt.Sprintf("len(%s)", operand)
	case *UnaryArithExpr:
		return fmt.Sprintf("(-(%s))", gen.formatArith(node.Expr))
	case *BinaryArithExpr:
		op := "+"
		switch node.Op {
		case TokenMinus:
			op = "-"
		case TokenStar:
			op = "*"
		case TokenSlash:
			op = "/"
		}
		return fmt.Sprintf("((%s) %s (%s))", gen.formatArith(node.Left), op, gen.formatArith(node.Right))
	}
	return "0"
}
