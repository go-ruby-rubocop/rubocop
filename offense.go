// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package rubocop is a pure-Go (no cgo) reimplementation of the core of Ruby's
// RuboCop linter — the offense/cop framework, a faithful core cop set, the
// .rubocop.yml configuration model and the default formatters — built on the
// go-ruby-parser Ruby AST and lexer.
//
// It is interpreter-independent: cops inspect a Ruby source string (its lines,
// its token stream and its abstract syntax tree) and emit Offenses whose cop
// name, location and message byte-match those the `rubocop` gem produces. The
// file-walking that the gem performs is a thin host seam; this library runs on
// source strings, which is the shape [go-embedded-ruby] binds into `rbgo`.
//
// It is a sibling of the other go-ruby-* front-end libraries
// (go-ruby-parser, go-ruby-regexp, go-ruby-erb, go-ruby-yaml) and, like them,
// hands the host an explicit value model rather than driving a live interpreter.
package rubocop

import "fmt"

// Severity is an offense severity level, matching RuboCop's five-level scale.
// Its single-letter code is what the formatters print (I/R/C/W/E/F).
type Severity int

// The severity levels, in ascending order of seriousness. The zero value is
// Convention, RuboCop's default for most Style/Layout/Metrics cops.
const (
	Convention Severity = iota // C — a style/convention offense (the common case)
	Warning                    // W — a Lint warning (a probable mistake)
	Error                      // E — a definite problem
	Fatal                      // F — an internal/parse failure
	Info                       // I — informational only
	Refactor                   // R — a refactoring suggestion
)

// code is the single-letter severity marker RuboCop prints in its formatters.
func (s Severity) code() string {
	switch s {
	case Warning:
		return "W"
	case Error:
		return "E"
	case Fatal:
		return "F"
	case Info:
		return "I"
	case Refactor:
		return "R"
	default:
		return "C"
	}
}

// name is the lowercase severity word (used in the JSON-ish / long forms).
func (s Severity) name() string {
	switch s {
	case Warning:
		return "warning"
	case Error:
		return "error"
	case Fatal:
		return "fatal"
	case Info:
		return "info"
	case Refactor:
		return "refactor"
	default:
		return "convention"
	}
}

// Location is a 1-based line/column span within the inspected source, exactly as
// RuboCop reports it: Line and Column point at the first offending character and
// Length is the number of columns the caret underline spans in the clang-style
// formatter (0 when the cop does not define a precise span).
type Location struct {
	Line   int
	Column int
	Length int
}

// Offense is a single reported violation. CopName is the department-qualified cop
// name (e.g. "Layout/TrailingWhitespace"); Message is the human-readable text the
// cop emits (byte-identical to the gem's); Severity is its level; Correctable
// records whether the cop offers an autocorrection (the "[Correctable]" marker).
type Offense struct {
	CopName     string
	Location    Location
	Message     string
	Severity    Severity
	Correctable bool
	// Correction, when non-nil, is the autocorrection the cop would apply. It is
	// advisory: this library computes the edit but leaves applying it to the host.
	Correction *Correction
}

// String renders an offense in RuboCop's canonical one-line form used by the
// clang / progress formatters: "line:col: S: [Correctable] Cop: Message".
func (o Offense) String() string {
	mark := ""
	if o.Correctable {
		mark = "[Correctable] "
	}
	return fmt.Sprintf("%d:%d: %s: %s%s: %s",
		o.Location.Line, o.Location.Column, o.Severity.code(), mark, o.CopName, o.Message)
}

// Correction is a single source edit an autocorrecting cop proposes: replace the
// half-open byte range [Begin, End) of the original source with Replacement.
// Applying corrections is the host's job (see Runner.Autocorrect for a reference
// applier); the cop only computes them.
type Correction struct {
	Begin       int
	End         int
	Replacement string
}
