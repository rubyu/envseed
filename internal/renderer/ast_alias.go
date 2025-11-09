package renderer

// Re-export of AST types for test ergonomics.
// Purpose: Allow external tests to reference core element/token kinds
// through the renderer namespace without importing the ast package directly.
// This avoids leaking internal details in the production API and keeps
// test code concise and readable.
import "envseed/internal/ast"

type Element = ast.Element
type Assignment = ast.Assignment
type ValueToken = ast.ValueToken
type ValueContext = ast.ValueContext
type ValueTokenKind = ast.ValueTokenKind
type AssignmentOperator = ast.AssignmentOperator

const (
	ElementAssignment = ast.ElementAssignment
	ElementComment    = ast.ElementComment
	ElementBlank      = ast.ElementBlank

	ContextBare                = ast.ContextBare
	ContextDoubleQuoted        = ast.ContextDoubleQuoted
	ContextSingleQuoted        = ast.ContextSingleQuoted
	ContextCommandSubstitution = ast.ContextCommandSubstitution
	ContextBacktick            = ast.ContextBacktick

	ValueLiteral     = ast.ValueLiteral
	ValuePlaceholder = ast.ValuePlaceholder
	OperatorAssign   = ast.OperatorAssign
	OperatorAppend   = ast.OperatorAppend
)
