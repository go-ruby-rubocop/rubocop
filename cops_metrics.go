// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
	"fmt"
	"strings"

	"github.com/go-ruby-parser/parser/token"
)

// countBodyLines counts the "code lines" of a def/class body the way RuboCop's
// Metrics cops do: lines between the opener and its `end` (both exclusive) that
// are neither blank nor comment-only. This is the [count/Max] numerator.
func countBodyLines(src *Source, b block) int {
	endLine := src.Tokens[b.endIdx].Line
	n := 0
	for l := b.open.Line + 1; l < endLine; l++ {
		trimmed := strings.TrimSpace(src.line(l))
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		n++
	}
	return n
}

// --- Metrics/MethodLength -----------------------------------------------------

// methodLengthCop flags a method whose body exceeds Max code lines (default 10).
// It reports at the `def` keyword with the "[count/Max]" tally, matching the gem.
type methodLengthCop struct{}

func (methodLengthCop) Name() string { return "Metrics/MethodLength" }

func (methodLengthCop) Inspect(src *Source, cfg CopConfig) []Offense {
	max := cfg.Int("Max", 10)
	var offs []Offense
	for _, b := range matchBlocks(src) {
		if b.open.Type != token.DEF {
			continue
		}
		n := countBodyLines(src, b)
		if n <= max {
			continue
		}
		offs = append(offs, Offense{
			CopName:  "Metrics/MethodLength",
			Location: Location{Line: b.open.Line, Column: b.open.Col, Length: 3},
			Message:  fmt.Sprintf("Method has too many lines. [%d/%d]", n, max),
			Severity: Convention,
		})
	}
	return offs
}

// --- Metrics/ClassLength ------------------------------------------------------

// classLengthCop flags a class whose body exceeds Max code lines (default 100),
// reporting at the `class` keyword with the "[count/Max]" tally.
type classLengthCop struct{}

func (classLengthCop) Name() string { return "Metrics/ClassLength" }

func (classLengthCop) Inspect(src *Source, cfg CopConfig) []Offense {
	max := cfg.Int("Max", 100)
	var offs []Offense
	for _, b := range matchBlocks(src) {
		if b.open.Type != token.CLASS {
			continue
		}
		n := countBodyLines(src, b)
		if n <= max {
			continue
		}
		offs = append(offs, Offense{
			CopName:  "Metrics/ClassLength",
			Location: Location{Line: b.open.Line, Column: b.open.Col, Length: 5},
			Message:  fmt.Sprintf("Class has too many lines. [%d/%d]", n, max),
			Severity: Convention,
		})
	}
	return offs
}
