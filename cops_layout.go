// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
	"fmt"
	"strings"

	"github.com/go-ruby-parser/parser/token"
)

// --- Layout/TrailingWhitespace ------------------------------------------------

// trailingWhitespaceCop flags any line ending in a space or tab. Offense column
// is the first trailing-whitespace character (1-based); the correction strips it.
type trailingWhitespaceCop struct{}

func (trailingWhitespaceCop) Name() string { return "Layout/TrailingWhitespace" }

func (trailingWhitespaceCop) Inspect(src *Source, _ CopConfig) []Offense {
	var offs []Offense
	for i, line := range src.Lines {
		trimmed := strings.TrimRight(line, " \t")
		if len(trimmed) == len(line) {
			continue
		}
		col := len(trimmed) + 1
		begin := src.offsetOf(i+1, col)
		offs = append(offs, Offense{
			CopName:     "Layout/TrailingWhitespace",
			Location:    Location{Line: i + 1, Column: col, Length: len(line) - len(trimmed)},
			Message:     "Trailing whitespace detected.",
			Severity:    Convention,
			Correctable: true,
			Correction:  &Correction{Begin: begin, End: begin + (len(line) - len(trimmed)), Replacement: ""},
		})
	}
	return offs
}

// --- Layout/TrailingEmptyLines ------------------------------------------------

// trailingEmptyLinesCop flags a missing final newline or trailing blank lines at
// end of file. RuboCop reports at (lastContentLine+1, 1) with the count.
type trailingEmptyLinesCop struct{}

func (trailingEmptyLinesCop) Name() string { return "Layout/TrailingEmptyLines" }

func (trailingEmptyLinesCop) Inspect(src *Source, _ CopConfig) []Offense {
	if len(src.Lines) == 0 {
		return nil
	}
	// Count trailing blank lines (from the end).
	blank := 0
	for i := len(src.Lines) - 1; i >= 0 && strings.TrimSpace(src.Lines[i]) == ""; i-- {
		blank++
	}
	// A single blank last line with a final newline is the well-formed "file ends
	// in exactly one \n" shape RuboCop wants; no offense.
	if blank == 0 {
		if !src.hasFinalNewline {
			// Final line has content but no newline: "final newline missing".
			last := len(src.Lines)
			col := len(src.Lines[last-1]) + 1
			return []Offense{{
				CopName:     "Layout/TrailingEmptyLines",
				Location:    Location{Line: last, Column: col},
				Message:     "Final newline missing.",
				Severity:    Convention,
				Correctable: true,
				Correction:  &Correction{Begin: len(src.Raw), End: len(src.Raw), Replacement: "\n"},
			}}
		}
		return nil
	}
	// blank >= 1: "N trailing blank lines detected." reported after content. The
	// gem uses the plural "lines" even for a single trailing blank line.
	line := len(src.Lines) - blank + 1
	msg := fmt.Sprintf("%d trailing blank lines detected.", blank)
	// Correction: cut the trailing blank lines' bytes down to a single newline.
	contentEnd := src.offsetOf(len(src.Lines)-blank+1, 1)
	if contentEnd > 0 {
		contentEnd-- // drop back onto the newline that ends the last content line
	}
	return []Offense{{
		CopName:     "Layout/TrailingEmptyLines",
		Location:    Location{Line: line, Column: 1},
		Message:     msg,
		Severity:    Convention,
		Correctable: true,
		Correction:  &Correction{Begin: contentEnd, End: len(src.Raw), Replacement: "\n"},
	}}
}

// --- Layout/SpaceAfterComma ---------------------------------------------------

// spaceAfterCommaCop flags a comma not followed by whitespace. It reports at the
// comma column; the correction inserts a space.
type spaceAfterCommaCop struct{}

func (spaceAfterCommaCop) Name() string { return "Layout/SpaceAfterComma" }

func (spaceAfterCommaCop) Inspect(src *Source, _ CopConfig) []Offense {
	var offs []Offense
	toks := src.Tokens
	for i, t := range toks {
		if t.Type != token.COMMA {
			continue
		}
		next := toks[i+1] // EOF terminates the stream, so i+1 is always valid
		// A space or a line break after the comma satisfies the cop.
		if next.SpaceBefore || next.Line != t.Line || next.Type == token.NEWLINE || next.Type == token.EOF {
			continue
		}
		begin := src.offsetOf(t.Line, t.Col)
		offs = append(offs, Offense{
			CopName:     "Layout/SpaceAfterComma",
			Location:    Location{Line: t.Line, Column: t.Col, Length: 1},
			Message:     "Space missing after comma.",
			Severity:    Convention,
			Correctable: true,
			Correction:  &Correction{Begin: begin + 1, End: begin + 1, Replacement: " "},
		})
	}
	return offs
}

// --- Layout/EmptyLines --------------------------------------------------------

// emptyLinesCop flags two or more consecutive blank lines, reporting the second
// (extra) blank line. It ignores trailing blank lines at EOF (that is
// TrailingEmptyLines' job), matching the gem.
type emptyLinesCop struct{}

func (emptyLinesCop) Name() string { return "Layout/EmptyLines" }

func (emptyLinesCop) Inspect(src *Source, _ CopConfig) []Offense {
	var offs []Offense
	// Find the last content line so trailing blanks are excluded.
	lastContent := 0
	for i := len(src.Lines); i >= 1; i-- {
		if strings.TrimSpace(src.line(i)) != "" {
			lastContent = i
			break
		}
	}
	run := 0
	for i := 1; i <= lastContent; i++ {
		if strings.TrimSpace(src.line(i)) == "" {
			run++
			if run >= 2 {
				begin := src.offsetOf(i, 1)
				offs = append(offs, Offense{
					CopName:     "Layout/EmptyLines",
					Location:    Location{Line: i, Column: 1},
					Message:     "Extra blank line detected.",
					Severity:    Convention,
					Correctable: true,
					Correction:  &Correction{Begin: begin, End: begin + len(src.line(i)) + 1, Replacement: ""},
				})
			}
		} else {
			run = 0
		}
	}
	return offs
}

// --- Layout/IndentationWidth --------------------------------------------------

// indentationWidthCop is a faithful-enough port for the common case: it checks
// that the body of a def/class/module/if/while block is indented exactly Width
// (default 2) columns past the opening keyword. It reports the first body line
// whose indentation is off, with the gem's "Use N (not M) spaces for indentation."
// message.
type indentationWidthCop struct{}

func (indentationWidthCop) Name() string { return "Layout/IndentationWidth" }

func (indentationWidthCop) Inspect(src *Source, cfg CopConfig) []Offense {
	width := cfg.Int("Width", 2)
	var offs []Offense
	toks := src.Tokens
	// Openers whose first body line's indentation we check against the opener's.
	openers := map[token.Type]bool{
		token.DEF: true, token.CLASS: true, token.MODULE: true,
		token.IF: true, token.UNLESS: true, token.WHILE: true, token.UNTIL: true,
		token.CASE: true, token.BEGIN: true,
	}
	for i, t := range toks {
		if !openers[t.Type] {
			continue
		}
		// Skip modifier-if/unless/while (opener not first significant on its line).
		if !firstOnLine(src, t) {
			continue
		}
		openerIndent := indentOf(src.line(t.Line))
		// Find the first body line: the next line after the opener that has content
		// and is not itself the matching `end`.
		bodyLine := firstBodyLine(src, t.Line)
		if bodyLine == 0 {
			continue
		}
		got := indentOf(src.line(bodyLine))
		want := openerIndent + width
		if got != want && strings.TrimSpace(src.line(bodyLine)) != "end" {
			offs = append(offs, Offense{
				CopName:     "Layout/IndentationWidth",
				Location:    Location{Line: bodyLine, Column: 1, Length: got},
				Message:     fmt.Sprintf("Use %d (not %d) spaces for indentation.", width, got-openerIndent),
				Severity:    Convention,
				Correctable: true,
			})
		}
		_ = i
	}
	return offs
}

// firstOnLine reports whether tok is the first non-space token on its line.
func firstOnLine(src *Source, tok token.Token) bool {
	line := src.line(tok.Line)
	return indentOf(line)+1 == tok.Col
}

// firstBodyLine returns the first content line strictly after openerLine, or 0.
func firstBodyLine(src *Source, openerLine int) int {
	for i := openerLine + 1; i <= len(src.Lines); i++ {
		if strings.TrimSpace(src.line(i)) != "" {
			return i
		}
	}
	return 0
}

// indentOf returns the number of leading spaces of line.
func indentOf(line string) int {
	n := 0
	for n < len(line) && line[n] == ' ' {
		n++
	}
	return n
}

// --- Layout/LineLength & Metrics/LineLength -----------------------------------

// lineLengthCop flags a physical line longer than Max columns. RuboCop ships the
// same check under both Layout/LineLength (modern) and, historically,
// Metrics/LineLength; the name is parameterised so one implementation serves both.
type lineLengthCop struct{ name string }

func (c lineLengthCop) Name() string { return c.name }

func (c lineLengthCop) Inspect(src *Source, cfg CopConfig) []Offense {
	max := cfg.Int("Max", 120)
	var offs []Offense
	for i, line := range src.Lines {
		n := len([]rune(line))
		if n <= max {
			continue
		}
		offs = append(offs, Offense{
			CopName:  c.name,
			Location: Location{Line: i + 1, Column: max + 1, Length: n - max},
			Message:  fmt.Sprintf("Line is too long. [%d/%d]", n, max),
			Severity: Convention,
		})
	}
	return offs
}
