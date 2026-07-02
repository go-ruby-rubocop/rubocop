// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
	"strings"

	"github.com/go-ruby-parser/parser"
	"github.com/go-ruby-parser/parser/ast"
	"github.com/go-ruby-parser/parser/lexer"
	"github.com/go-ruby-parser/parser/token"
)

// Source is the processed view of one Ruby source string handed to every cop: the
// raw bytes, the lines split for the line-oriented Layout cops, the lexed token
// stream (each token carrying its 1-based Line/Col and MRI's SpaceBefore flag),
// and the parsed AST (nil if the source does not parse). RuboCop cops likewise
// mix line, token and AST views; go-ruby-parser's AST nodes carry no positions, so
// token positions are how the AST cops locate their offenses.
type Source struct {
	// Path is the host-supplied filename used in messages that name the file
	// (e.g. Lint/DuplicateMethods); "" when inspecting an anonymous string.
	Path string
	// Raw is the original source, verbatim.
	Raw string
	// Lines are Raw split on "\n" with the newlines removed; Lines[0] is line 1.
	// A trailing newline does not produce a final empty element, matching how
	// RuboCop counts physical lines.
	Lines []string
	// hasFinalNewline records whether Raw ended in "\n" (so line-count cops that
	// care about the trailing blank can tell "a\n" from "a").
	hasFinalNewline bool
	// Tokens is the full lexer output including the terminating EOF token.
	Tokens []token.Token
	// AST is the parsed program, or nil when the source failed to parse.
	AST *ast.Program
	// ParseErr is the parse error, or nil.
	ParseErr error
}

// NewSource lexes and parses src, building the shared Source view. It never
// fails: a parse error is recorded in ParseErr / leaves AST nil so line- and
// token-oriented cops still run on unparseable input, exactly as RuboCop keeps
// its Layout cops working when the AST is unavailable.
func NewSource(path, src string) *Source {
	s := &Source{Path: path, Raw: src}
	s.hasFinalNewline = strings.HasSuffix(src, "\n")
	body := src
	if s.hasFinalNewline {
		body = src[:len(src)-1]
	}
	if body == "" && !s.hasFinalNewline {
		s.Lines = nil
	} else {
		s.Lines = strings.Split(body, "\n")
	}
	s.Tokens = lexer.New(src).Tokenize()
	prog, err := parser.Parse(src)
	if err != nil {
		s.ParseErr = err
	} else {
		s.AST = prog
	}
	return s
}

// lineCount is the number of physical lines RuboCop attributes to the source.
func (s *Source) lineCount() int { return len(s.Lines) }

// line returns the 1-based line n (or "" when out of range).
func (s *Source) line(n int) string {
	if n < 1 || n > len(s.Lines) {
		return ""
	}
	return s.Lines[n-1]
}

// offsetOf converts a 1-based line/column into a byte offset into Raw. Columns
// past the end of a line clamp to the line's newline position; this is only used
// to build Correction spans, so exactness past EOL is immaterial.
func (s *Source) offsetOf(line, col int) int {
	off := 0
	for i := 1; i < line && i <= len(s.Lines); i++ {
		off += len(s.Lines[i-1]) + 1 // +1 for the '\n'
	}
	off += col - 1
	if off < 0 {
		off = 0
	}
	if off > len(s.Raw) {
		off = len(s.Raw)
	}
	return off
}
