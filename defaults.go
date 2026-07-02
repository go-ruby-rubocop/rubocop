// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

// copDefaults holds each built-in cop's default configuration: whether it is on
// out of the box (all the core cops are) and its default parameters. These mirror
// the corresponding entries in RuboCop's bundled config/default.yml.
var copDefaults = map[string]CopConfig{
	// Layout
	"Layout/TrailingWhitespace": {Enabled: true},
	"Layout/TrailingEmptyLines": {Enabled: true},
	"Layout/SpaceAfterComma":    {Enabled: true},
	"Layout/EmptyLines":         {Enabled: true},
	"Layout/IndentationWidth":   {Enabled: true, Params: map[string]any{"Width": 2}},
	"Layout/LineLength":         {Enabled: true, Params: map[string]any{"Max": 120}},

	// Style
	"Style/StringLiterals":             {Enabled: true, Params: map[string]any{"EnforcedStyle": "single_quotes"}},
	"Style/FrozenStringLiteralComment": {Enabled: true, Params: map[string]any{"EnforcedStyle": "always"}},
	"Style/MethodDefParentheses":       {Enabled: true, Params: map[string]any{"EnforcedStyle": "require_parentheses"}},
	"Style/RedundantReturn":            {Enabled: true},
	"Style/Not":                        {Enabled: true},
	"Style/IfUnlessModifier":           {Enabled: true},
	"Style/GuardClause":                {Enabled: true, Params: map[string]any{"MinBodyLength": 1}},
	"Style/NumericLiterals":            {Enabled: true, Params: map[string]any{"MinDigits": 5}},

	// Lint
	"Lint/UselessAssignment":         {Enabled: true},
	"Lint/UnusedMethodArgument":      {Enabled: true},
	"Lint/DuplicateMethods":          {Enabled: true},
	"Lint/AmbiguousOperator":         {Enabled: true},
	"Lint/ShadowingOuterLocalVariable": {Enabled: true},

	// Metrics
	"Metrics/MethodLength": {Enabled: true, Params: map[string]any{"Max": 10}},
	"Metrics/LineLength":   {Enabled: true, Params: map[string]any{"Max": 120}},
	"Metrics/ClassLength":  {Enabled: true, Params: map[string]any{"Max": 100}},
}

// defaultCopConfig returns the built-in default CopConfig for cop name, or an
// enabled, param-less default for a cop not in the table (a host-registered cop).
func defaultCopConfig(name string) CopConfig {
	if c, ok := copDefaults[name]; ok {
		// Copy the params map so callers cannot mutate the shared default.
		out := CopConfig{Enabled: c.Enabled, Params: map[string]any{}}
		for k, v := range c.Params {
			out.Params[k] = v
		}
		return out
	}
	return CopConfig{Enabled: true, Params: map[string]any{}}
}

// coreCops is the built-in core cop set, in construction order.
func coreCops() []Cop {
	return []Cop{
		// Layout
		trailingWhitespaceCop{}, trailingEmptyLinesCop{}, spaceAfterCommaCop{},
		emptyLinesCop{}, indentationWidthCop{}, lineLengthCop{name: "Layout/LineLength"},
		// Style
		stringLiteralsCop{}, frozenStringLiteralCommentCop{}, methodDefParenthesesCop{},
		redundantReturnCop{}, notCop{}, ifUnlessModifierCop{}, guardClauseCop{},
		numericLiteralsCop{},
		// Lint
		uselessAssignmentCop{}, unusedMethodArgumentCop{}, duplicateMethodsCop{},
		ambiguousOperatorCop{}, shadowingOuterLocalVariableCop{},
		// Metrics
		methodLengthCop{}, lineLengthCop{name: "Metrics/LineLength"}, classLengthCop{},
	}
}

// DefaultRegistry returns a Registry populated with the built-in core cop set.
func DefaultRegistry() *Registry {
	return NewRegistry().Register(coreCops()...)
}
