// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
	"testing"

	"github.com/go-ruby-parser/parser/token"
)

// TestAllDecimalDigits drives the helper's empty and non-digit branches directly
// (the NumericLiterals cop only ever feeds it all-digit INT literals).
func TestAllDecimalDigits(t *testing.T) {
	if allDecimalDigits("") {
		t.Error("empty string is not all-digits")
	}
	if allDecimalDigits("12a3") {
		t.Error("12a3 is not all-digits")
	}
	if !allDecimalDigits("123") {
		t.Error("123 is all-digits")
	}
}

// TestGroupThousandsShort covers the <=3 digit early return.
func TestGroupThousandsShort(t *testing.T) {
	if groupThousands("12") != "12" {
		t.Error("short input should pass through")
	}
	if groupThousands("1234567") != "1_234_567" {
		t.Errorf("1234567 grouped = %q", groupThousands("1234567"))
	}
}

// TestDoubleQuotedRawUnterminated covers the no-closing-quote return of
// doubleQuotedRaw with a hand-built token past a lone opening quote.
func TestDoubleQuotedRawUnterminated(t *testing.T) {
	src := NewSource("t.rb", `"abc`)
	tok := token.Token{Type: token.STRING, Line: 1, Col: 1}
	if _, ok := doubleQuotedRaw(src, tok); ok {
		t.Error("unterminated double-quote should not report a raw body")
	}
	// A token whose position is past EOF also returns false.
	if _, ok := doubleQuotedRaw(src, token.Token{Line: 9, Col: 9}); ok {
		t.Error("out-of-range token should return false")
	}
}

// TestConditionTextClamp covers conditionText's start-past-EOL clamp with a
// crafted block whose keyword column exceeds the line length.
func TestConditionTextClamp(t *testing.T) {
	src := NewSource("t.rb", "if\n")
	b := block{open: token.Token{Type: token.IF, Line: 1, Col: 10}}
	if got := conditionText(src, b); got != "" {
		t.Errorf("clamped condition = %q", got)
	}
}

// TestLexerHelpersOnClass ensures a class/module opener is matched (feeds the
// non-DEF continue branches in the Lint cops that only act on defs).
func TestLintCopsIgnoreClassOnlySource(t *testing.T) {
	src := "class Foo\n  X = 1\nend\n"
	for _, name := range []string{
		"Lint/UselessAssignment", "Lint/UnusedMethodArgument",
		"Lint/ShadowingOuterLocalVariable", "Metrics/MethodLength",
	} {
		if offs := inspectOne(t, name, "t.rb", src, nil); len(offs) != 0 {
			t.Errorf("%s on class-only source = %v", name, offs)
		}
	}
}

// TestDefNameTokenAndConstScope drives defNameToken's false return (a def whose
// name token is absent) via a class with no CONST name is not applicable; instead
// use a nested def inside a module to reach the scope-name path and a class body.
func TestDuplicateMethodsNestedModuleScope(t *testing.T) {
	// A module wrapping two same-named defs: scope name is the module const.
	offs := inspectOne(t, "Lint/DuplicateMethods", "t.rb",
		"module M\n  def foo\n  end\n  def foo\n  end\nend\n", nil)
	wantOne(t, offs,
		"4:3: W: Lint/DuplicateMethods: Method M#foo is defined at both t.rb:2 and t.rb:4.")
}

// TestAmbiguousOperatorNonCommand covers the prev-not-IDENT skip (a `*` after a
// non-identifier is not the command-arg ambiguity).
func TestAmbiguousOperatorNonCommand(t *testing.T) {
	// `2 *x` — prev is an INT, not an IDENT command name.
	wantNone(t, inspectOne(t, "Lint/AmbiguousOperator", "t.rb", "def foo(x)\n  2 *x\nend\n", nil))
}

// TestUnusedMethodArgumentNoParen covers defParamTokens returning nil when the def
// has no parenthesised signature (a paren-less def with a used bareword param).
func TestUnusedMethodArgumentNoParen(t *testing.T) {
	// `def foo a` (no parens) — defParamTokens finds no '(' and returns nil, so no
	// unused-arg analysis runs.
	wantNone(t, inspectOne(t, "Lint/UnusedMethodArgument", "t.rb", "def foo a\n  1\nend\n", nil))
}

// TestMethodDefParenthesesDegenerate covers the j>=len and non-name guards with
// degenerate inputs (a bare `def` at EOF and an operator-named def).
func TestMethodDefParenthesesDegenerate(t *testing.T) {
	// A lone `def` with no following name: the scan runs off the token stream.
	wantNone(t, inspectOne(t, "Style/MethodDefParentheses", "t.rb", "def", nil))
	// An operator method name (`def +`) is not an IDENT/CONST name token.
	wantNone(t, inspectOne(t, "Style/MethodDefParentheses", "t.rb", "def +(o)\n  o\nend\n", nil))
}

// TestGuardClauseWhileLast covers the guard's non-if/unless branch: a def whose
// last statement is a while loop is not a guard-clause candidate.
func TestGuardClauseWhileLast(t *testing.T) {
	wantNone(t, inspectOne(t, "Style/GuardClause", "t.rb",
		"def foo\n  while x\n    a\n  end\nend\n", nil))
}

// TestGuardClauseIfElseLast covers the guard's hasElseOrElsif skip on a trailing
// if that carries an else.
func TestGuardClauseIfElseLast(t *testing.T) {
	wantNone(t, inspectOne(t, "Style/GuardClause", "t.rb",
		"def foo\n  if x\n    a\n  else\n    b\n  end\nend\n", nil))
}

// TestRedundantReturnBlockLast covers lastTrailingReturn's trailing-`end` branch:
// a def whose last statement is a block (not a return).
func TestRedundantReturnBlockLast(t *testing.T) {
	wantNone(t, inspectOne(t, "Style/RedundantReturn", "t.rb",
		"def foo\n  if x\n    a\n  end\nend\n", nil))
}

// TestDuplicateMethodsSingletonClassOpener covers nextConstName's empty return: a
// `class << self` opener has no CONST name.
func TestDuplicateMethodsSingletonClassOpener(t *testing.T) {
	wantNone(t, inspectOne(t, "Lint/DuplicateMethods", "t.rb",
		"class << self\n  def foo\n  end\nend\n", nil))
}

// TestDuplicateMethodsOperatorName covers defNameToken's false return (a def whose
// name is an operator token, not an IDENT/CONST) — no crash, no offense.
func TestDuplicateMethodsOperatorName(t *testing.T) {
	wantNone(t, inspectOne(t, "Lint/DuplicateMethods", "t.rb", "def +(o)\nend\n", nil))
}

// TestHelperEdgeCases drives the last few private-helper guards directly, over
// real Sources plus the single crafted case the cop API cannot produce.
func TestHelperEdgeCases(t *testing.T) {
	// defParamTokens with an opening paren but no closing one: returns what it
	// collected so far (the fall-through return). Build a def block by hand.
	src := NewSource("t.rb", "def foo(a, b\n")
	b := block{open: src.Tokens[0], openIdx: 0, endIdx: len(src.Tokens) - 1}
	if got := defParamTokens(src.Tokens, b); len(got) != 2 {
		t.Errorf("defParamTokens (no rparen) = %d params, want 2", len(got))
	}

	// lastStatementBlock when the def's last statement is not a block returns -1.
	src3 := NewSource("t.rb", "def foo\n  x\nend\n")
	blocks3 := matchBlocks(src3)
	if lastStatementBlock(blocks3, src3.Tokens, blocks3[0]) != -1 {
		t.Error("lastStatementBlock should be -1 when last statement is not a block")
	}

	// isStatementStart at the opener boundary (i == openIdx+1) and beyond.
	if !isStatementStart(src3.Tokens, blocks3[0].openIdx+1, blocks3[0]) {
		t.Error("first body token should be a statement start")
	}

	// lastStatementBlock's no-matching-block return: point a synthetic def block's
	// end onto a stray `end` token that matchBlocks did not record as any block's
	// endIdx (the second `end` in a source with one too many).
	srcStray := NewSource("t.rb", "def foo\nend\nend\n")
	strayBlocks := matchBlocks(srcStray)
	var strayEnd int
	for i, tk := range srcStray.Tokens {
		if tk.Type == token.END {
			strayEnd = i // the last END is the unmatched, stray one
		}
	}
	synthetic := block{open: srcStray.Tokens[0], openIdx: 0, endIdx: strayEnd + 1}
	if lastStatementBlock(strayBlocks, srcStray.Tokens, synthetic) != -1 {
		t.Error("stray end should not resolve to a block")
	}
}

// TestMethodDefParenthesesTruncatedReceiver covers the j>=len guard: a `def self.`
// with nothing after the dot runs the name scan off the token stream.
func TestMethodDefParenthesesTruncatedReceiver(t *testing.T) {
	wantNone(t, inspectOne(t, "Style/MethodDefParentheses", "t.rb", "def self.", nil))
}

// TestUselessAssignmentMidStatement covers isStatementStart's non-start branch: an
// assignment IDENT that is not at a statement boundary (a keyword-argument-like
// `k: v` is lexed as a LABEL, but `foo(x = 1)` has x= not at statement start).
func TestUselessAssignmentInsideCall(t *testing.T) {
	// `bar(x = 1)` — x= is not at a statement start, so it is not analysed here.
	wantNone(t, inspectOne(t, "Lint/UselessAssignment", "t.rb",
		"def foo\n  bar(x = 1)\nend\n", nil))
}
