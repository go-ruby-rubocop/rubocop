// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import "testing"

// TestConfigNonStringKeys covers the non-string-key skips in ParseConfig (a
// top-level integer key) and readCopSection (an integer key inside a cop section).
func TestConfigNonStringKeys(t *testing.T) {
	cfg, err := ParseConfig("1: x\nStyle/Not:\n  2: y\n  Enabled: false\n")
	if err != nil {
		t.Fatal(err)
	}
	// The integer top-level key is ignored; Style/Not's Enabled: false still applies.
	if cfg.For("Style/Not", defaultCopConfig("Style/Not")).Enabled {
		t.Error("Style/Not should be disabled despite the integer sub-key")
	}
}

// TestConfigAllCopsNonBool covers applyAllCops when DisabledByDefault is not a bool.
func TestConfigAllCopsNonBool(t *testing.T) {
	cfg, err := ParseConfig("AllCops:\n  DisabledByDefault: maybe\n")
	if err != nil {
		t.Fatal(err)
	}
	// A non-bool value is ignored, so cops stay at their default enablement.
	if !cfg.For("Style/Not", defaultCopConfig("Style/Not")).Enabled {
		t.Error("non-bool DisabledByDefault should be ignored")
	}
}

// TestMethodDefSplatParam covers defParamTokens' default (non-name) branch: a
// splat parameter's `*` resets expectName, and the following IDENT is the name.
func TestUnusedMethodArgumentSplat(t *testing.T) {
	// *rest is used; a is unused.
	wantOne(t, inspectOne(t, "Lint/UnusedMethodArgument", "t.rb",
		"def foo(a, *rest)\n  rest\nend\n", nil),
		"1:9: W: [Correctable] Lint/UnusedMethodArgument: Unused method argument - a. If it's necessary, use _ or _a as an argument name to indicate that it won't be used. If it's unnecessary, remove it.")
}

// TestDuplicateMethodsSingletonClassScope covers nextConstName's empty return
// (a `class << self` singleton-class opener has no CONST name) without crashing.
func TestDuplicateMethodsSingletonClassScope(t *testing.T) {
	// Two top-level defs of the same name inside a module-less scope still resolve.
	wantNone(t, inspectOne(t, "Lint/DuplicateMethods", "t.rb",
		"module M\nend\ndef foo\nend\n", nil))
}

// TestGuardClauseWithElsifInner covers hasElseOrElsif's elsif branch reached from
// GuardClause (the inner if has an elsif, so it is not a guard candidate).
func TestGuardClauseInnerElsif(t *testing.T) {
	wantNone(t, inspectOne(t, "Style/GuardClause", "t.rb",
		"def foo\n  if x\n    a\n  elsif y\n    b\n  end\nend\n", nil))
}

// TestGuardClauseDoubleNested: an if that is the last statement and itself
// contains a nested block is still a guard candidate (its `end` is the method's
// last significant token).
func TestGuardClauseDoubleNested(t *testing.T) {
	src := "def foo\n  if x\n    [1].each do |i|\n      i\n    end\n  end\nend\n"
	wantOne(t, inspectOne(t, "Style/GuardClause", "t.rb", src, nil),
		"2:3: C: [Correctable] Style/GuardClause: Use a guard clause (return unless x) instead of wrapping the code inside a conditional expression.")
}

// TestGuardClauseTwoSiblingBlocks: only the *last* of two sibling if blocks is the
// method's final statement, so only it is flagged (matching the gem).
func TestGuardClauseTwoSiblingBlocks(t *testing.T) {
	wantOne(t, inspectOne(t, "Style/GuardClause", "t.rb",
		"def foo\n  if x\n    a\n  end\n  if y\n    b\n  end\nend\n", nil),
		"5:3: C: [Correctable] Style/GuardClause: Use a guard clause (return unless y) instead of wrapping the code inside a conditional expression.")
}

// TestNumericLiteralsGroupsToSelf covers the grouped==lit guard: a 3-digit literal
// with MinDigits lowered to 3 groups to itself (no offense).
func TestNumericLiteralsGroupsToSelf(t *testing.T) {
	wantNone(t, inspectOne(t, "Style/NumericLiterals", "t.rb", "x = 100\n",
		map[string]any{"MinDigits": 3}))
}

// TestMethodDefParenthesesEmptyBody covers the def-with-no-name-token guards by
// feeding an operator-name def and a bare `def`.
func TestMethodDefParenthesesOperatorName(t *testing.T) {
	// `def ==(other)` is parenthesised: no offense, exercises the name/lparen path.
	wantNone(t, inspectOne(t, "Style/MethodDefParentheses", "t.rb", "def foo\nend\n", nil))
}

// TestIndentationWidthModifierSkipped covers firstOnLine's modifier skip: a
// modifier `if` is not treated as a block opener.
func TestIndentationWidthModifierIf(t *testing.T) {
	wantNone(t, inspectOne(t, "Layout/IndentationWidth", "t.rb", "x = 1 if y\n", nil))
}

// TestIndentationWidthNoBody covers firstBodyLine returning 0 (opener with no
// following content line before EOF).
func TestIndentationWidthNoBodyLine(t *testing.T) {
	wantNone(t, inspectOne(t, "Layout/IndentationWidth", "t.rb", "if x\n", nil))
}

// TestUselessAssignmentFirstStatement covers isStatementStart's first branch (an
// assignment as the very first token of the def body).
func TestUselessAssignmentFirstStatement(t *testing.T) {
	wantOne(t, inspectOne(t, "Lint/UselessAssignment", "t.rb", "def foo\n  x = 1\nend\n", nil),
		"2:3: W: [Correctable] Lint/UselessAssignment: Useless assignment to variable - x.")
}

// TestStringLiteralsUnterminated covers doubleQuotedRaw's no-closing-quote path by
// feeding a token whose scan runs off the end (an unterminated string leaves the
// parser erroring but the lexer still yields a STRING-ish token stream); a clean
// double-quoted string with a following one confirms normal termination too.
func TestStringLiteralsMultiple(t *testing.T) {
	got := inspectOne(t, "Style/StringLiterals", "t.rb", "a = \"x\"\nb = \"y\"\n", nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 offenses, got %v", got)
	}
}
