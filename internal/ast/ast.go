package ast

type ElementType int

const (
	ElementAssignment ElementType = iota
	ElementComment
	ElementBlank
)

type ValueContext int

const (
	ContextBare ValueContext = iota
	ContextDoubleQuoted
	ContextSingleQuoted
	ContextCommandSubstitution
	ContextBacktick
)

type ValueTokenKind int

const (
	ValueLiteral ValueTokenKind = iota
	ValuePlaceholder
)

type AssignmentOperator int

const (
	OperatorAssign AssignmentOperator = iota
	OperatorAppend
)

type ValueToken struct {
	Kind      ValueTokenKind
	Text      string
	Path      string
	Modifiers []string
	Context   ValueContext
	Line      int
	Column    int
}

type Assignment struct {
	Name               string
	Operator           AssignmentOperator
	LeadingWhitespace  string
	Raw                string
	ValueTokens        []ValueToken
	TrailingComment    string
	Line               int
	Column             int
	HasTrailingNewline bool
}

type Element struct {
	Type               ElementType
	Assignment         *Assignment
	Text               string
	Line               int
	ColumnStart        int
	HasTrailingNewline bool
}
