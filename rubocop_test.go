// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
	"strings"
	"testing"
)

// inspectOne runs a single named cop over source with its default config (plus any
// param overrides) and returns the offenses, mirroring how the gem's --only runs.
func inspectOne(t *testing.T, name, path, source string, params map[string]any) []Offense {
	t.Helper()
	cop, ok := DefaultRegistry().Get(name)
	if !ok {
		t.Fatalf("cop %s not registered", name)
	}
	cfg := defaultCopConfig(name)
	for k, v := range params {
		cfg.Params[k] = v
	}
	return cop.Inspect(NewSource(path, source), cfg)
}

// wantOne asserts exactly one offense with the given canonical String() form.
func wantOne(t *testing.T, offs []Offense, want string) {
	t.Helper()
	if len(offs) != 1 {
		t.Fatalf("got %d offenses, want 1: %v", len(offs), offs)
	}
	if got := offs[0].String(); got != want {
		t.Fatalf("offense = %q\n         want %q", got, want)
	}
}

// wantNone asserts no offenses.
func wantNone(t *testing.T, offs []Offense) {
	t.Helper()
	if len(offs) != 0 {
		t.Fatalf("got %d offenses, want 0: %v", len(offs), offs)
	}
}

// --- Golden vectors: one positive + one negative per cop ----------------------

func TestLayoutTrailingWhitespace(t *testing.T) {
	wantOne(t, inspectOne(t, "Layout/TrailingWhitespace", "t.rb", "x = 1   \n", nil),
		"1:6: C: [Correctable] Layout/TrailingWhitespace: Trailing whitespace detected.")
	wantNone(t, inspectOne(t, "Layout/TrailingWhitespace", "t.rb", "x = 1\n", nil))
}

func TestLayoutTrailingEmptyLines(t *testing.T) {
	wantOne(t, inspectOne(t, "Layout/TrailingEmptyLines", "t.rb", "a = 1\nb = 2\n\n\n", nil),
		"3:1: C: [Correctable] Layout/TrailingEmptyLines: 2 trailing blank lines detected.")
	// Single trailing blank line.
	wantOne(t, inspectOne(t, "Layout/TrailingEmptyLines", "t.rb", "a = 1\n\n", nil),
		"2:1: C: [Correctable] Layout/TrailingEmptyLines: 1 trailing blank lines detected.")
	// Missing final newline.
	wantOne(t, inspectOne(t, "Layout/TrailingEmptyLines", "t.rb", "a = 1", nil),
		"1:6: C: [Correctable] Layout/TrailingEmptyLines: Final newline missing.")
	// Well-formed.
	wantNone(t, inspectOne(t, "Layout/TrailingEmptyLines", "t.rb", "a = 1\n", nil))
	// Empty source: no lines, no offense.
	wantNone(t, inspectOne(t, "Layout/TrailingEmptyLines", "t.rb", "", nil))
}

func TestLayoutSpaceAfterComma(t *testing.T) {
	wantOne(t, inspectOne(t, "Layout/SpaceAfterComma", "t.rb", "foo(a,b)\n", nil),
		"1:6: C: [Correctable] Layout/SpaceAfterComma: Space missing after comma.")
	wantNone(t, inspectOne(t, "Layout/SpaceAfterComma", "t.rb", "foo(a, b)\n", nil))
	// Comma at end of line (trailing) is fine.
	wantNone(t, inspectOne(t, "Layout/SpaceAfterComma", "t.rb", "[a,\nb]\n", nil))
}

func TestLayoutEmptyLines(t *testing.T) {
	wantOne(t, inspectOne(t, "Layout/EmptyLines", "t.rb", "def foo\n\n  x = 1\n\n\n  x\nend\n", nil),
		"5:1: C: [Correctable] Layout/EmptyLines: Extra blank line detected.")
	// A single blank line is allowed.
	wantNone(t, inspectOne(t, "Layout/EmptyLines", "t.rb", "def foo\n\n  1\nend\n", nil))
}

func TestLayoutIndentationWidth(t *testing.T) {
	wantOne(t, inspectOne(t, "Layout/IndentationWidth", "t.rb", "def foo\n    x = 1\n    x\nend\n", nil),
		"2:1: C: [Correctable] Layout/IndentationWidth: Use 2 (not 4) spaces for indentation.")
	wantNone(t, inspectOne(t, "Layout/IndentationWidth", "t.rb", "def foo\n  x = 1\nend\n", nil))
	// Custom width honoured.
	wantNone(t, inspectOne(t, "Layout/IndentationWidth", "t.rb", "def foo\n    x = 1\nend\n",
		map[string]any{"Width": 4}))
}

func TestLayoutLineLength(t *testing.T) {
	wantOne(t, inspectOne(t, "Layout/LineLength", "t.rb", "this_is_a_very_long_line = 123456789\n",
		map[string]any{"Max": 10}),
		"1:11: C: Layout/LineLength: Line is too long. [36/10]")
	wantNone(t, inspectOne(t, "Layout/LineLength", "t.rb", "short\n", nil))
}

func TestStyleStringLiterals(t *testing.T) {
	wantOne(t, inspectOne(t, "Style/StringLiterals", "t.rb", "x = 'a'\ny = \"b\"\n", nil),
		"2:5: C: [Correctable] Style/StringLiterals: Prefer single-quoted strings when you don't need string interpolation or special symbols.")
	// A double-quoted string with an escape is allowed.
	wantNone(t, inspectOne(t, "Style/StringLiterals", "t.rb", "y = \"a\\nb\"\n", nil))
	// double_quotes style: cop switches off here (not re-implemented).
	wantNone(t, inspectOne(t, "Style/StringLiterals", "t.rb", "y = \"b\"\n",
		map[string]any{"EnforcedStyle": "double_quotes"}))
}

func TestStyleFrozenStringLiteralComment(t *testing.T) {
	wantOne(t, inspectOne(t, "Style/FrozenStringLiteralComment", "t.rb", "x = 1\n", nil),
		"1:1: C: [Correctable] Style/FrozenStringLiteralComment: Missing frozen string literal comment.")
	wantNone(t, inspectOne(t, "Style/FrozenStringLiteralComment", "t.rb",
		"# frozen_string_literal: true\n\nx = 1\n", nil))
	// After a shebang.
	wantNone(t, inspectOne(t, "Style/FrozenStringLiteralComment", "t.rb",
		"#!/usr/bin/env ruby\n# frozen_string_literal: true\nx = 1\n", nil))
	// Empty source: nothing to flag.
	wantNone(t, inspectOne(t, "Style/FrozenStringLiteralComment", "t.rb", "", nil))
	// Non-always style: off.
	wantNone(t, inspectOne(t, "Style/FrozenStringLiteralComment", "t.rb", "x = 1\n",
		map[string]any{"EnforcedStyle": "never"}))
}

func TestStyleMethodDefParentheses(t *testing.T) {
	wantOne(t, inspectOne(t, "Style/MethodDefParentheses", "t.rb", "def foo a\n  a\nend\n", nil),
		"1:9: C: [Correctable] Style/MethodDefParentheses: Use def with parentheses when there are parameters.")
	// Parenthesised, or no params: fine.
	wantNone(t, inspectOne(t, "Style/MethodDefParentheses", "t.rb", "def foo(a)\n  a\nend\n", nil))
	wantNone(t, inspectOne(t, "Style/MethodDefParentheses", "t.rb", "def foo\nend\n", nil))
	// Non-default style: off.
	wantNone(t, inspectOne(t, "Style/MethodDefParentheses", "t.rb", "def foo a\n  a\nend\n",
		map[string]any{"EnforcedStyle": "require_no_parentheses"}))
}

func TestStyleRedundantReturn(t *testing.T) {
	wantOne(t, inspectOne(t, "Style/RedundantReturn", "t.rb", "def foo(a, b)\n  return a + b\nend\n", nil),
		"2:3: C: [Correctable] Style/RedundantReturn: Redundant return detected.")
	// A return that is not the last statement is fine.
	wantNone(t, inspectOne(t, "Style/RedundantReturn", "t.rb", "def foo\n  return 1\n  2\nend\n", nil))
	// No AST => no offense.
	wantNone(t, inspectOne(t, "Style/RedundantReturn", "t.rb", "def (\n", nil))
}

func TestStyleNot(t *testing.T) {
	wantOne(t, inspectOne(t, "Style/Not", "t.rb", "x = (not foo)\n", nil),
		"1:6: C: [Correctable] Style/Not: Use ! instead of not.")
	wantNone(t, inspectOne(t, "Style/Not", "t.rb", "x = !foo\n", nil))
}

func TestStyleNumericLiterals(t *testing.T) {
	wantOne(t, inspectOne(t, "Style/NumericLiterals", "t.rb", "x = 10000\n", nil),
		"1:5: C: [Correctable] Style/NumericLiterals: Use underscores(_) as thousands separator and separate every 3 digits with them.")
	// Already grouped, too short, or hex: fine.
	wantNone(t, inspectOne(t, "Style/NumericLiterals", "t.rb", "x = 10_000\n", nil))
	wantNone(t, inspectOne(t, "Style/NumericLiterals", "t.rb", "x = 100\n", nil))
	wantNone(t, inspectOne(t, "Style/NumericLiterals", "t.rb", "x = 0xffff\n", nil))
}

func TestStyleIfUnlessModifier(t *testing.T) {
	wantOne(t, inspectOne(t, "Style/IfUnlessModifier", "t.rb", "x = 1\nif x\n  y = 2\nend\n", nil),
		"2:1: C: [Correctable] Style/IfUnlessModifier: Favor modifier if usage when having a single-line body. Another good alternative is the usage of control flow &&/||.")
	// unless variant.
	wantOne(t, inspectOne(t, "Style/IfUnlessModifier", "t.rb", "x = 1\nunless x\n  y = 2\nend\n", nil),
		"2:1: C: [Correctable] Style/IfUnlessModifier: Favor modifier unless usage when having a single-line body. Another good alternative is the usage of control flow &&/||.")
	// Multi-line body: fine.
	wantNone(t, inspectOne(t, "Style/IfUnlessModifier", "t.rb", "if x\n  a\n  b\nend\n", nil))
	// With else: fine.
	wantNone(t, inspectOne(t, "Style/IfUnlessModifier", "t.rb", "if x\n  a\nelse\n  b\nend\n", nil))
}

func TestStyleGuardClause(t *testing.T) {
	wantOne(t, inspectOne(t, "Style/GuardClause", "t.rb", "def foo\n  if x\n    do_thing\n  end\nend\n", nil),
		"2:3: C: [Correctable] Style/GuardClause: Use a guard clause (return unless x) instead of wrapping the code inside a conditional expression.")
	// unless => "return if".
	wantOne(t, inspectOne(t, "Style/GuardClause", "t.rb", "def foo\n  unless x\n    do_thing\n  end\nend\n", nil),
		"2:3: C: [Correctable] Style/GuardClause: Use a guard clause (return if x) instead of wrapping the code inside a conditional expression.")
	// A trailing if (the method's last statement) is flagged even with a leading
	// statement, matching the gem.
	wantOne(t, inspectOne(t, "Style/GuardClause", "t.rb", "def foo\n  a\n  if x\n    b\n  end\nend\n", nil),
		"3:3: C: [Correctable] Style/GuardClause: Use a guard clause (return unless x) instead of wrapping the code inside a conditional expression.")
	// An if that is NOT the last statement: fine.
	wantNone(t, inspectOne(t, "Style/GuardClause", "t.rb", "def foo\n  if x\n    a\n  end\n  b\nend\n", nil))
	// With else: fine.
	wantNone(t, inspectOne(t, "Style/GuardClause", "t.rb", "def foo\n  if x\n    a\n  else\n    b\n  end\nend\n", nil))
}

func TestLintUselessAssignment(t *testing.T) {
	wantOne(t, inspectOne(t, "Lint/UselessAssignment", "t.rb", "def foo\n  x = 1\n  2\nend\n", nil),
		"2:3: W: [Correctable] Lint/UselessAssignment: Useless assignment to variable - x.")
	// Used afterwards: fine.
	wantNone(t, inspectOne(t, "Lint/UselessAssignment", "t.rb", "def foo\n  x = 1\n  x\nend\n", nil))
	// Reassigned then used: the reassignment (a later statement-start x=) is a read.
	wantNone(t, inspectOne(t, "Lint/UselessAssignment", "t.rb", "def foo\n  x = 1\n  x = 2\n  x\nend\n", nil))
}

func TestLintUnusedMethodArgument(t *testing.T) {
	wantOne(t, inspectOne(t, "Lint/UnusedMethodArgument", "t.rb", "def foo(a, b)\n  a\nend\n", nil),
		"1:12: W: [Correctable] Lint/UnusedMethodArgument: Unused method argument - b. If it's necessary, use _ or _b as an argument name to indicate that it won't be used. If it's unnecessary, remove it.")
	// All used: fine.
	wantNone(t, inspectOne(t, "Lint/UnusedMethodArgument", "t.rb", "def foo(a, b)\n  a + b\nend\n", nil))
	// Underscore-prefixed: exempt.
	wantNone(t, inspectOne(t, "Lint/UnusedMethodArgument", "t.rb", "def foo(_a)\n  1\nend\n", nil))
	// No params: nothing.
	wantNone(t, inspectOne(t, "Lint/UnusedMethodArgument", "t.rb", "def foo\n  1\nend\n", nil))
}

func TestLintDuplicateMethods(t *testing.T) {
	wantOne(t, inspectOne(t, "Lint/DuplicateMethods", "t.rb", "def foo\nend\ndef foo\nend\n", nil),
		"3:1: W: Lint/DuplicateMethods: Method Object#foo is defined at both t.rb:1 and t.rb:3.")
	// Distinct names: fine.
	wantNone(t, inspectOne(t, "Lint/DuplicateMethods", "t.rb", "def foo\nend\ndef bar\nend\n", nil))
	// Same name in different classes: fine.
	wantNone(t, inspectOne(t, "Lint/DuplicateMethods", "t.rb",
		"class A\n  def foo\n  end\nend\nclass B\n  def foo\n  end\nend\n", nil))
}

func TestLintDuplicateMethodsInClass(t *testing.T) {
	offs := inspectOne(t, "Lint/DuplicateMethods", "t.rb",
		"class Foo\n  def bar\n  end\n  def bar\n  end\nend\n", nil)
	wantOne(t, offs,
		"4:3: W: Lint/DuplicateMethods: Method Foo#bar is defined at both t.rb:2 and t.rb:4.")
}

func TestLintAmbiguousOperator(t *testing.T) {
	wantOne(t, inspectOne(t, "Lint/AmbiguousOperator", "t.rb", "def foo(x)\n  bar *x\nend\n", nil),
		"2:7: W: [Correctable] Lint/AmbiguousOperator: Ambiguous splat operator. Parenthesize the method arguments if it's surely a splat operator, or add a whitespace to the right of the * if it should be a multiplication.")
	// Spaced multiplication: fine.
	wantNone(t, inspectOne(t, "Lint/AmbiguousOperator", "t.rb", "def foo(x)\n  bar * x\nend\n", nil))
}

func TestLintShadowingOuterLocalVariable(t *testing.T) {
	wantOne(t, inspectOne(t, "Lint/ShadowingOuterLocalVariable", "t.rb",
		"x = 1\n[1,2].each do |x|\n  puts x\nend\n", nil),
		"2:16: W: Lint/ShadowingOuterLocalVariable: Shadowing outer local variable - x.")
	// Distinct block param: fine.
	wantNone(t, inspectOne(t, "Lint/ShadowingOuterLocalVariable", "t.rb",
		"x = 1\n[1,2].each do |y|\n  puts y\nend\n", nil))
}

func TestMetricsMethodLength(t *testing.T) {
	wantOne(t, inspectOne(t, "Metrics/MethodLength", "t.rb", "def foo\n  a = 1\n  b = 2\n  c = 3\nend\n",
		map[string]any{"Max": 2}),
		"1:1: C: Metrics/MethodLength: Method has too many lines. [3/2]")
	wantNone(t, inspectOne(t, "Metrics/MethodLength", "t.rb", "def foo\n  a = 1\nend\n",
		map[string]any{"Max": 2}))
}

func TestMetricsClassLength(t *testing.T) {
	wantOne(t, inspectOne(t, "Metrics/ClassLength", "t.rb", "class Foo\n  a = 1\n  b = 2\n  c = 3\nend\n",
		map[string]any{"Max": 2}),
		"1:1: C: Metrics/ClassLength: Class has too many lines. [3/2]")
	// Blank + comment lines are not counted.
	wantNone(t, inspectOne(t, "Metrics/ClassLength", "t.rb", "class Foo\n  a = 1\n\n  # c\nend\n",
		map[string]any{"Max": 2}))
}

func TestMetricsLineLength(t *testing.T) {
	wantOne(t, inspectOne(t, "Metrics/LineLength", "t.rb", "abcdefghijk\n",
		map[string]any{"Max": 5}),
		"1:6: C: Metrics/LineLength: Line is too long. [11/5]")
}

// --- Framework, config, runner, formatters ------------------------------------

func TestRunnerReportOrderAndDefaults(t *testing.T) {
	// nil registry + nil config default to the core set and all-defaults.
	run := NewRunner(nil, nil)
	src := "def foo(a, b)\n  return a + b\nend\n"
	offs := run.Inspect("foo.rb", src)
	// Expect RedundantReturn and UnusedMethodArgument, sorted by line/col.
	if len(offs) < 2 {
		t.Fatalf("expected >=2 offenses, got %v", offs)
	}
	// First offense should be on line 1 (the unused arg) before line 2.
	if offs[0].Location.Line > offs[len(offs)-1].Location.Line {
		t.Fatalf("offenses not sorted: %v", offs)
	}
}

func TestSeverityCodesAndNames(t *testing.T) {
	cases := []struct {
		s    Severity
		code string
		name string
	}{
		{Convention, "C", "convention"},
		{Warning, "W", "warning"},
		{Error, "E", "error"},
		{Fatal, "F", "fatal"},
		{Info, "I", "info"},
		{Refactor, "R", "refactor"},
		{Severity(99), "C", "convention"},
	}
	for _, c := range cases {
		if got := c.s.code(); got != c.code {
			t.Errorf("code(%d) = %q want %q", c.s, got, c.code)
		}
		if got := c.s.name(); got != c.name {
			t.Errorf("name(%d) = %q want %q", c.s, got, c.name)
		}
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	r.Register(notCop{})
	if _, ok := r.Get("Style/Not"); !ok {
		t.Fatal("Style/Not not registered")
	}
	if _, ok := r.Get("Nope"); ok {
		t.Fatal("unexpected cop")
	}
	// Idempotent replacement: re-registering does not duplicate the name.
	r.Register(notCop{})
	if len(r.Names()) != 1 {
		t.Fatalf("names = %v", r.Names())
	}
	if len(r.Cops()) != 1 {
		t.Fatalf("cops = %d", len(r.Cops()))
	}
}

func TestConfigParseAndMerge(t *testing.T) {
	yml := "AllCops:\n  DisabledByDefault: true\n" +
		"Style/Not:\n  Enabled: true\n" +
		"Metrics/MethodLength:\n  Max: 3\n"
	cfg, err := ParseConfig(yml)
	if err != nil {
		t.Fatal(err)
	}
	// DisabledByDefault means an un-listed cop is off.
	if cfg.For("Layout/TrailingWhitespace", defaultCopConfig("Layout/TrailingWhitespace")).Enabled {
		t.Error("expected TrailingWhitespace disabled under DisabledByDefault")
	}
	// Explicitly enabled cop is on.
	if !cfg.For("Style/Not", defaultCopConfig("Style/Not")).Enabled {
		t.Error("expected Style/Not enabled")
	}
	// Param override applied.
	mm := cfg.For("Metrics/MethodLength", defaultCopConfig("Metrics/MethodLength"))
	if mm.Int("Max", 10) != 3 {
		t.Errorf("Max = %d want 3", mm.Int("Max", 10))
	}
}

func TestConfigEnableDisableWithoutAllCops(t *testing.T) {
	cfg, _ := ParseConfig("Style/Not:\n  Enabled: false\n")
	if cfg.For("Style/Not", defaultCopConfig("Style/Not")).Enabled {
		t.Error("Style/Not should be disabled")
	}
	// A cop with only a param override keeps its default enablement.
	cfg2, _ := ParseConfig("Metrics/MethodLength:\n  Max: 5\n")
	if !cfg2.For("Metrics/MethodLength", defaultCopConfig("Metrics/MethodLength")).Enabled {
		t.Error("param-only override should keep the cop enabled")
	}
}

func TestConfigEmptyAndNonMapping(t *testing.T) {
	if _, err := ParseConfig(""); err != nil {
		t.Fatal(err)
	}
	// A bare scalar document has no cop config.
	cfg, err := ParseConfig("42\n")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.For("Style/Not", defaultCopConfig("Style/Not")).Enabled != true {
		t.Error("scalar config should leave defaults intact")
	}
}

func TestConfigParseError(t *testing.T) {
	// A tab used for indentation is a YAML syntax error the loader rejects.
	if _, err := ParseConfig("a:\n\t- b\n"); err == nil {
		t.Fatal("expected a parse error")
	}
}

func TestConfigAccessors(t *testing.T) {
	c := CopConfig{Params: map[string]any{
		"i": 7, "i64": int64(8), "f": 3.0, "s": "x", "b": true,
	}}
	if c.Int("i", 0) != 7 || c.Int("i64", 0) != 8 || c.Int("f", 0) != 3 {
		t.Error("Int accessor")
	}
	if c.Int("missing", 42) != 42 || c.Int("s", 42) != 42 {
		t.Error("Int fallback")
	}
	if c.Str("s", "d") != "x" || c.Str("missing", "d") != "d" || c.Str("i", "d") != "d" {
		t.Error("Str accessor")
	}
	if c.Bool("b", false) != true || c.Bool("missing", true) != true || c.Bool("s", true) != true {
		t.Error("Bool accessor")
	}
}

func TestSimpleFormatter(t *testing.T) {
	offs := inspectOne(t, "Layout/TrailingWhitespace", "t.rb", "x = 1   \n", nil)
	got := SimpleTextFormatter{}.Format([]FileResult{{Path: "t.rb", Offenses: offs}})
	want := "== t.rb ==\n" +
		"C:  1:  6: [Correctable] Layout/TrailingWhitespace: Trailing whitespace detected.\n" +
		"\n1 file inspected, 1 offense detected, 1 offense autocorrectable\n"
	if got != want {
		t.Fatalf("simple format:\n%q\nwant:\n%q", got, want)
	}
}

func TestSimpleFormatterClean(t *testing.T) {
	got := SimpleTextFormatter{}.Format([]FileResult{{Path: "a.rb"}, {Path: "b.rb"}})
	want := "\n2 files inspected, no offenses detected\n"
	if got != want {
		t.Fatalf("clean format = %q want %q", got, want)
	}
}

func TestProgressFormatter(t *testing.T) {
	offs := inspectOne(t, "Layout/TrailingWhitespace", "t.rb", "x = 1   \n", nil)
	got := ProgressFormatter{}.Format([]FileResult{
		{Path: "t.rb", Offenses: offs},
		{Path: "clean.rb"},
	})
	if !strings.HasPrefix(got, "C.\n") {
		t.Fatalf("progress status = %q", got)
	}
	if !strings.Contains(got, "\nOffenses:\n\n") {
		t.Fatalf("progress detail missing: %q", got)
	}
	if !strings.HasSuffix(got, "2 files inspected, 1 offense detected, 1 offense autocorrectable\n") {
		t.Fatalf("progress summary = %q", got)
	}
}

func TestProgressFormatterAllClean(t *testing.T) {
	got := ProgressFormatter{}.Format([]FileResult{{Path: "a.rb"}})
	want := ".\n\n1 file inspected, no offenses detected\n"
	if got != want {
		t.Fatalf("progress clean = %q want %q", got, want)
	}
}

func TestOffenseStringNonCorrectable(t *testing.T) {
	o := Offense{CopName: "X/Y", Location: Location{Line: 3, Column: 4}, Message: "m", Severity: Error}
	if got := o.String(); got != "3:4: E: X/Y: m" {
		t.Fatalf("String = %q", got)
	}
}
