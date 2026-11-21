// Copyright (c) 2026 Cafecito Games LLC
// SPDX-License-Identifier: MIT

package sbl

// Program is the parsed representation of a Snowball (.sbl) source file.
type Program struct {
	Strings   []string
	Integers  []string
	Booleans  []string
	Routines  []string
	Externals []string
	Groupings []string

	GroupingDefs map[string][]rune
	RoutineDefs  []*RoutineDef

	stringSet   map[string]bool
	integerSet  map[string]bool
	booleanSet  map[string]bool
	routineSet  map[string]bool
	groupingSet map[string]bool
}

// RoutineDef represents a single Snowball routine definition (`define name as C`).
type RoutineDef struct {
	Name     string
	Backward bool
	Body     Command
	External bool
}

// Command is the interface implemented by all Snowball AST command nodes.
type Command interface {
	commandNode()
}

// SequenceCmd executes a list of commands in order, failing at the first false signal.
type SequenceCmd struct {
	Commands []Command
}

// BinaryCmd represents `Left or Right` or `Left and Right`.
type BinaryCmd struct {
	Op    TokenKind // TokenOr or TokenAnd
	Left  Command
	Right Command
}

// UnaryCmd represents `not`, `test`, `try`, `do`, `fail`, `goto`, `gopast`, `repeat`, `backwards`.
type UnaryCmd struct {
	Op   TokenKind
	Body Command
}

// LoopCmd represents `loop Count Body` or `atleast Count Body`.
type LoopCmd struct {
	AtLeast bool
	Count   ArithExpr
	Body    Command
}

// SetLimitCmd represents `setlimit Limit for Body`.
type SetLimitCmd struct {
	Limit Command
	Body  Command
}

// StringMatchCmd matches a literal string or string variable at the current cursor.
type StringMatchCmd struct {
	IsVariable bool
	Value      string
}

// GroupingCmd matches a rune inside (or outside when Negated is true) a grouping.
type GroupingCmd struct {
	Name    string
	Negated bool
}

// RoutineCallCmd calls a named routine.
type RoutineCallCmd struct {
	Name string
}

// BoolCmd gets, sets, or unsets a boolean variable.
type BoolCmd struct {
	Op   TokenKind // TokenSet, TokenUnset, or TokenIdent (test)
	Name string
}

// SliceBoundCmd represents `[` (IsClose=false) or `]` (IsClose=true).
type SliceBoundCmd struct {
	IsClose bool
}

// SliceEditCmd represents `<- S`, `delete`, `insert S`, `attach S`, or `-> var`.
type SliceEditCmd struct {
	Op         TokenKind // TokenSliceFrom, TokenDelete, TokenInsert, TokenAttach, TokenSliceTo
	IsVariable bool
	Value      string
}

// HopCmd moves the cursor by Count runes (`hop AE` or `next`).
type HopCmd struct {
	Count ArithExpr
}

// MarkCmd represents `setmark x`, `tomark AE`, `atmark AE`, `tolimit`, `atlimit`.
type MarkCmd struct {
	Op   TokenKind
	Name string
	Expr ArithExpr
}

// IntAssignCmd represents `$x = AE`, `$x += AE`, `$x -= AE`, `$x *= AE`, `$x /= AE`.
type IntAssignCmd struct {
	Name string
	Op   TokenKind
	Expr ArithExpr
}

// IntTestCmd represents `$x <op> AE` or `$(AE <op> AE)`.
type IntTestCmd struct {
	Left  ArithExpr
	Op    TokenKind
	Right ArithExpr
}

// StringAssignCmd represents `$x = S` or `=> x`.
type StringAssignCmd struct {
	Name       string
	IsVariable bool
	Value      string
}

// StringScopeCmd represents `$s C` (executing command C on string variable s).
type StringScopeCmd struct {
	Name string
	Body Command
}

// SubstringCmd represents `substring`, later linked to its following AmongCmd.
type SubstringCmd struct {
	AmongID int
}

// AmongEntry represents one string pattern (with optional condition routine) inside `among(...)`.
type AmongEntry struct {
	Text      string
	Condition string
	Result    int
}

// AmongAction represents one action branch inside `among(...)`.
type AmongAction struct {
	Result int
	Body   Command
}

// AmongCmd represents `among(...)`.
type AmongCmd struct {
	ID           int
	HasSubstring bool
	Backward     bool
	Starter      Command
	Entries      []AmongEntry
	Actions      []AmongAction
}

// LiteralBoolCmd represents `true` or `false`.
type LiteralBoolCmd struct {
	Value bool
}

// ArithExpr is the interface implemented by all Snowball arithmetic expression nodes.
type ArithExpr interface {
	arithNode()
}

// IntLiteralExpr represents an integer constant.
type IntLiteralExpr struct {
	Value int
}

// IntVarExpr represents an integer variable name.
type IntVarExpr struct {
	Name string
}

// EnvMetricExpr represents `cursor`, `limit`, `size`, `len`, `maxint`, `minint`.
type EnvMetricExpr struct {
	Kind TokenKind
}

// StringMetricExpr represents `sizeof S` or `lenof S`.
type StringMetricExpr struct {
	Kind       TokenKind // TokenSizeOf or TokenLenOf
	IsVariable bool
	Value      string
}

// UnaryArithExpr represents `-Expr`.
type UnaryArithExpr struct {
	Expr ArithExpr
}

// BinaryArithExpr represents `Left <op> Right` (`+`, `-`, `*`, `/`).
type BinaryArithExpr struct {
	Op    TokenKind
	Left  ArithExpr
	Right ArithExpr
}

func (*SequenceCmd) commandNode()     {}
func (*BinaryCmd) commandNode()       {}
func (*UnaryCmd) commandNode()        {}
func (*LoopCmd) commandNode()         {}
func (*SetLimitCmd) commandNode()     {}
func (*StringMatchCmd) commandNode()  {}
func (*GroupingCmd) commandNode()     {}
func (*RoutineCallCmd) commandNode()  {}
func (*BoolCmd) commandNode()         {}
func (*SliceBoundCmd) commandNode()   {}
func (*SliceEditCmd) commandNode()    {}
func (*HopCmd) commandNode()          {}
func (*MarkCmd) commandNode()         {}
func (*IntAssignCmd) commandNode()    {}
func (*IntTestCmd) commandNode()      {}
func (*StringAssignCmd) commandNode() {}
func (*StringScopeCmd) commandNode()  {}
func (*SubstringCmd) commandNode()    {}
func (*AmongCmd) commandNode()        {}
func (*LiteralBoolCmd) commandNode()  {}

func (*IntLiteralExpr) arithNode()   {}
func (*IntVarExpr) arithNode()       {}
func (*EnvMetricExpr) arithNode()    {}
func (*StringMetricExpr) arithNode() {}
func (*UnaryArithExpr) arithNode()   {}
func (*BinaryArithExpr) arithNode()  {}
