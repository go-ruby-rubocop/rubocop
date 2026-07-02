// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
	"strings"
	"testing"
)

// TestAutocorrectApplies exercises the reference corrector: a source with several
// autocorrectable offenses is rewritten to the fixed form.
func TestAutocorrectApplies(t *testing.T) {
	run := NewRunner(
		NewRegistry().Register(trailingWhitespaceCop{}, spaceAfterCommaCop{}),
		NewConfig(),
	)
	got := run.Autocorrect("t.rb", "foo(a,b)   \n")
	if got != "foo(a, b)\n" {
		t.Fatalf("autocorrect = %q", got)
	}
}

// TestAutocorrectNoCorrections returns the source unchanged when nothing corrects.
func TestAutocorrectNoCorrections(t *testing.T) {
	run := NewRunner(NewRegistry().Register(lineLengthCop{name: "Layout/LineLength"}), NewConfig())
	src := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"
	if got := run.Autocorrect("t.rb", src); got != src {
		t.Fatalf("autocorrect changed non-correctable source: %q", got)
	}
}

// TestAutocorrectOverlapSkipped keeps the earliest of two overlapping corrections.
func TestAutocorrectOverlapSkipped(t *testing.T) {
	// Two cops proposing overlapping edits on the same bytes: only the first
	// (lowest offset) is applied.
	a := fakeCorrectCop{name: "A/One", corr: &Correction{Begin: 0, End: 3, Replacement: "XXX"}}
	b := fakeCorrectCop{name: "A/Two", corr: &Correction{Begin: 1, End: 4, Replacement: "Y"}}
	run := NewRunner(NewRegistry().Register(a, b), NewConfig())
	got := run.Autocorrect("t.rb", "abcdef")
	if got != "XXXdef" {
		t.Fatalf("overlap autocorrect = %q", got)
	}
}

// TestAutocorrectOutOfRangeSkipped ignores a correction whose span is invalid.
func TestAutocorrectOutOfRangeSkipped(t *testing.T) {
	c := fakeCorrectCop{name: "A/Bad", corr: &Correction{Begin: 0, End: 999, Replacement: "z"}}
	run := NewRunner(NewRegistry().Register(c), NewConfig())
	if got := run.Autocorrect("t.rb", "abc"); got != "abc" {
		t.Fatalf("out-of-range correction applied: %q", got)
	}
}

// fakeCorrectCop is a test cop that always reports one correctable offense with a
// fixed correction, for driving the corrector's branches.
type fakeCorrectCop struct {
	name string
	corr *Correction
}

func (f fakeCorrectCop) Name() string { return f.name }
func (f fakeCorrectCop) Inspect(_ *Source, _ CopConfig) []Offense {
	return []Offense{{
		CopName:     f.name,
		Location:    Location{Line: 1, Column: 1},
		Message:     "x",
		Correctable: true,
		Correction:  f.corr,
	}}
}

// TestSortOffensesTieBreaks drives the line/column/name ordering in sortOffenses.
func TestSortOffensesTieBreaks(t *testing.T) {
	offs := []Offense{
		{CopName: "B", Location: Location{Line: 1, Column: 1}},
		{CopName: "A", Location: Location{Line: 1, Column: 1}},
		{CopName: "C", Location: Location{Line: 1, Column: 2}},
		{CopName: "D", Location: Location{Line: 2, Column: 1}},
	}
	sortOffenses(offs)
	order := []string{offs[0].CopName, offs[1].CopName, offs[2].CopName, offs[3].CopName}
	want := []string{"A", "B", "C", "D"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("sort order = %v want %v", order, want)
		}
	}
}

// TestRunnerSkipsDisabledCop confirms a disabled cop is not run.
func TestRunnerSkipsDisabledCop(t *testing.T) {
	cfg, _ := ParseConfig("Layout/TrailingWhitespace:\n  Enabled: false\n")
	run := NewRunner(DefaultRegistry(), cfg)
	for _, o := range run.Inspect("t.rb", "x = 1   \n") {
		if o.CopName == "Layout/TrailingWhitespace" {
			t.Fatal("disabled cop still ran")
		}
	}
}

// TestSourceLineAndOffsetBounds drives the out-of-range guards.
func TestSourceLineAndOffsetBounds(t *testing.T) {
	s := NewSource("t.rb", "ab\ncd\n")
	if s.line(0) != "" || s.line(99) != "" {
		t.Error("line bounds")
	}
	if s.line(1) != "ab" {
		t.Error("line 1")
	}
	// offsetOf past EOF clamps to len(Raw).
	if s.offsetOf(99, 1) != len(s.Raw) {
		t.Error("offset past EOF should clamp to len(Raw)")
	}
}

// TestNewSourceEmptyAndNoNewline covers the line-splitting branches.
func TestNewSourceEmptyAndNoNewline(t *testing.T) {
	if s := NewSource("", ""); len(s.Lines) != 0 {
		t.Errorf("empty source lines = %v", s.Lines)
	}
	if s := NewSource("", "abc"); len(s.Lines) != 1 || s.Lines[0] != "abc" {
		t.Errorf("no-newline source lines = %v", s.Lines)
	}
	if s := NewSource("", "a\nb\n"); len(s.Lines) != 2 {
		t.Errorf("trailing-newline lines = %v", s.Lines)
	}
}

// TestNewSourceParseError records a parse failure but still lexes.
func TestNewSourceParseError(t *testing.T) {
	s := NewSource("t.rb", "def (\n")
	if s.ParseErr == nil {
		t.Fatal("expected a parse error")
	}
	if s.AST != nil {
		t.Fatal("AST should be nil on parse error")
	}
	if len(s.Tokens) == 0 {
		t.Fatal("tokens should still be produced")
	}
}

// TestConfigNormalizeNestedMap covers normalizeParam's map recursion and the
// non-string-key skip.
func TestConfigNormalizeNestedMap(t *testing.T) {
	yml := "Style/Foo:\n  Details:\n    Nested: 5\n"
	cfg, err := ParseConfig(yml)
	if err != nil {
		t.Fatal(err)
	}
	cc := cfg.For("Style/Foo", CopConfig{Enabled: true, Params: map[string]any{}})
	details, ok := cc.Params["Details"].(map[string]any)
	if !ok {
		t.Fatalf("Details param = %#v", cc.Params["Details"])
	}
	nested := CopConfig{Params: details}.Int("Nested", 0)
	if nested != 5 {
		t.Errorf("nested = %v", details["Nested"])
	}
}

// TestConfigIgnoresNonMappingSections covers the branch where a top-level cop key
// maps to a scalar (not a mapping) and is skipped.
func TestConfigIgnoresNonMappingSections(t *testing.T) {
	cfg, err := ParseConfig("Style/Foo: bar\n")
	if err != nil {
		t.Fatal(err)
	}
	// The scalar section is ignored; the cop keeps its default enablement.
	if !cfg.For("Style/Foo", CopConfig{Enabled: true}).Enabled {
		t.Error("scalar section should not disable the cop")
	}
}

// TestDefaultCopConfigUnknown covers the fallback for a host-registered cop.
func TestDefaultCopConfigUnknown(t *testing.T) {
	c := defaultCopConfig("Custom/Whatever")
	if !c.Enabled || c.Params == nil {
		t.Fatalf("unknown default = %#v", c)
	}
}

// TestRedundantReturnNestedBlock exercises the inner-block skip in
// lastTrailingReturn (a nested if before the trailing return).
func TestRedundantReturnNestedBlock(t *testing.T) {
	src := "def foo\n  if x\n    a\n  end\n  return 1\nend\n"
	wantOne(t, inspectOne(t, "Style/RedundantReturn", "t.rb", src, nil),
		"5:3: C: [Correctable] Style/RedundantReturn: Redundant return detected.")
}

// TestRedundantReturnNotLast: a trailing statement after the return means the
// return is not the method's last statement.
func TestRedundantReturnNotLast(t *testing.T) {
	wantNone(t, inspectOne(t, "Style/RedundantReturn", "t.rb",
		"def foo\n  return 1 if x\n  2\nend\n", nil))
}

// TestGuardClauseTrailingStatement: an if followed by another statement is not
// the method's last statement, so it is not a guard candidate.
func TestGuardClauseTrailingStatement(t *testing.T) {
	wantNone(t, inspectOne(t, "Style/GuardClause", "t.rb",
		"def foo\n  if x\n    a\n  end\n  b\nend\n", nil))
}

// TestGuardClauseLeadingStatement: an if that is the method's last statement is
// flagged even with a leading statement before it.
func TestGuardClauseLeadingStatement(t *testing.T) {
	wantOne(t, inspectOne(t, "Style/GuardClause", "t.rb",
		"def foo\n  b\n  if x\n    a\n  end\nend\n", nil),
		"3:3: C: [Correctable] Style/GuardClause: Use a guard clause (return unless x) instead of wrapping the code inside a conditional expression.")
}

// TestSingletonMethodDef covers the self. receiver path in defNameToken /
// MethodDefParentheses.
func TestSingletonMethodDef(t *testing.T) {
	wantNone(t, inspectOne(t, "Lint/DuplicateMethods", "t.rb",
		"def self.foo\nend\ndef self.bar\nend\n", nil))
	// A singleton def with a bare param is still flagged by MethodDefParentheses.
	wantOne(t, inspectOne(t, "Style/MethodDefParentheses", "t.rb", "def self.foo a\n  a\nend\n", nil),
		"1:14: C: [Correctable] Style/MethodDefParentheses: Use def with parentheses when there are parameters.")
}

// TestIfUnlessModifierNoInnerBlockCases hits the elsif branch and the non-if path.
func TestIfUnlessModifierElsif(t *testing.T) {
	wantNone(t, inspectOne(t, "Style/IfUnlessModifier", "t.rb",
		"if x\n  a\nelsif y\n  b\nend\n", nil))
}

// TestAmbiguousDoubleSplatAndBlock covers the ** and & operator branches.
func TestAmbiguousDoubleSplatAndBlock(t *testing.T) {
	got := inspectOne(t, "Lint/AmbiguousOperator", "t.rb", "def foo(x)\n  bar **x\nend\n", nil)
	if len(got) != 1 || !strings.Contains(got[0].Message, "double splat") {
		t.Fatalf("** operator = %v", got)
	}
	got = inspectOne(t, "Lint/AmbiguousOperator", "t.rb", "def foo(x)\n  bar &x\nend\n", nil)
	if len(got) != 1 || !strings.Contains(got[0].Message, "block operator") {
		t.Fatalf("& operator = %v", got)
	}
}

// TestUselessAssignmentNonStatementStart: an assignment inside an expression is
// not a statement-start local assignment.
func TestUselessAssignmentReadInSameExpr(t *testing.T) {
	// x is read by the later `x + 1`, so no offense.
	wantNone(t, inspectOne(t, "Lint/UselessAssignment", "t.rb",
		"def foo\n  x = 1\n  x + 1\nend\n", nil))
}

// TestLineLengthUnicode counts runes, not bytes.
func TestLineLengthUnicode(t *testing.T) {
	// Five multibyte runes, Max 4 => too long, length reported in columns.
	got := inspectOne(t, "Layout/LineLength", "t.rb", "ééééé\n",
		map[string]any{"Max": 4})
	wantOne(t, got, "1:5: C: Layout/LineLength: Line is too long. [5/4]")
}

// TestFrozenCommentPresentNotFirst covers the "first non-blank line is not the
// magic comment" break.
func TestFrozenCommentOtherCommentFirst(t *testing.T) {
	wantOne(t, inspectOne(t, "Style/FrozenStringLiteralComment", "t.rb",
		"# some other comment\nx = 1\n", nil),
		"1:1: C: [Correctable] Style/FrozenStringLiteralComment: Missing frozen string literal comment.")
}

// TestFrozenCommentBlankLinesLeading covers the leading-blank-line skip.
func TestFrozenCommentLeadingBlank(t *testing.T) {
	wantNone(t, inspectOne(t, "Style/FrozenStringLiteralComment", "t.rb",
		"\n# frozen_string_literal: true\nx = 1\n", nil))
}

// TestGroupThousandsShort covers the <=3 digit early return path in groupThousands
// via a literal exactly at MinDigits with a remainder-0 grouping.
func TestNumericLiteralsExactGroups(t *testing.T) {
	// 123456 -> 123_456 (rem 0 path).
	wantOne(t, inspectOne(t, "Style/NumericLiterals", "t.rb", "x = 123456\n", nil),
		"1:5: C: [Correctable] Style/NumericLiterals: Use underscores(_) as thousands separator and separate every 3 digits with them.")
}

// TestMethodLengthCommentsBlanksIgnored confirms blank/comment lines don't count.
func TestMethodLengthIgnoresBlanksComments(t *testing.T) {
	wantNone(t, inspectOne(t, "Metrics/MethodLength", "t.rb",
		"def foo\n  a = 1\n\n  # c\n  b = 2\nend\n", map[string]any{"Max": 2}))
}

// TestEmptyLinesTrailingExcluded: trailing blank lines are TrailingEmptyLines'
// job, so EmptyLines ignores them.
func TestEmptyLinesTrailingExcluded(t *testing.T) {
	wantNone(t, inspectOne(t, "Layout/EmptyLines", "t.rb", "a = 1\n\n\n", nil))
}

// TestIndentationWidthEndLineSkipped covers the body-line-is-`end` guard.
func TestIndentationWidthEmptyBody(t *testing.T) {
	wantNone(t, inspectOne(t, "Layout/IndentationWidth", "t.rb", "def foo\nend\n", nil))
}

// TestBlockEndingAtMiss covers the not-found return of blockEndingAt via a source
// where a trailing return follows a `do…end` block (an end not opening a matched
// def/if but still handled).
func TestRedundantReturnAfterDoBlock(t *testing.T) {
	src := "def foo\n  [1].each do |i|\n    i\n  end\n  return 1\nend\n"
	wantOne(t, inspectOne(t, "Style/RedundantReturn", "t.rb", src, nil),
		"5:3: C: [Correctable] Style/RedundantReturn: Redundant return detected.")
}
