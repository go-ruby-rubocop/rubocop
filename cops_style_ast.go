// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
	"strings"

	"github.com/go-ruby-parser/parser/token"
)

// block is one keyword…end structure recovered from the token stream: its opening
// keyword token, the index of that token, and the index of the matching `end`.
// The Runner's structural Style cops (RedundantReturn, IfUnlessModifier,
// GuardClause) work off these because go-ruby-parser's AST carries no positions.
type block struct {
	open    token.Token
	openIdx int
	endIdx  int // index of the matching `end`, or len(toks)-1 if unterminated
}

// blockOpeners are the keywords that open an `end`-terminated block.
var blockOpeners = map[token.Type]bool{
	token.DEF: true, token.CLASS: true, token.MODULE: true, token.IF: true,
	token.UNLESS: true, token.WHILE: true, token.UNTIL: true, token.CASE: true,
	token.BEGIN: true, token.FOR: true, token.DO: true,
}

// matchBlocks pairs each block-opening keyword with its `end`, ignoring modifier
// if/unless/while/until (an opener that is not the first significant token on its
// line, which in Ruby is the modifier form and takes no `end`). It is a single
// left-to-right scan with an explicit stack.
func matchBlocks(src *Source) []block {
	toks := src.Tokens
	var stack []int
	blocks := make([]block, 0)
	// pending records, for each stack entry, the block record awaiting its end.
	pending := map[int]int{} // openIdx -> index into blocks
	for i, t := range toks {
		switch {
		case blockOpeners[t.Type]:
			// `do` always opens; if/unless/while/until only in statement position.
			if isModifierKeyword(t.Type) && !tokenFirstOnLine(src, t) {
				continue
			}
			b := block{open: t, openIdx: i, endIdx: len(toks) - 1}
			blocks = append(blocks, b)
			pending[i] = len(blocks) - 1
			stack = append(stack, i)
		case t.Type == token.END:
			if len(stack) > 0 {
				openIdx := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				blocks[pending[openIdx]].endIdx = i
			}
		}
	}
	return blocks
}

// isModifierKeyword reports whether tt can appear as a trailing modifier.
func isModifierKeyword(tt token.Type) bool {
	switch tt {
	case token.IF, token.UNLESS, token.WHILE, token.UNTIL:
		return true
	}
	return false
}

// tokenFirstOnLine reports whether t is the first token on its physical line.
func tokenFirstOnLine(src *Source, t token.Token) bool {
	return indentOf(src.line(t.Line))+1 == t.Col
}

// --- Style/RedundantReturn ----------------------------------------------------

// redundantReturnCop flags a `return` that is the last statement of a method body
// (Ruby's implicit return makes it redundant). It uses the AST to confirm the
// method's final statement is a Return, then locates the `return` keyword token
// inside the def block to report its position.
type redundantReturnCop struct{}

func (redundantReturnCop) Name() string { return "Style/RedundantReturn" }

func (redundantReturnCop) Inspect(src *Source, _ CopConfig) []Offense {
	if src.AST == nil {
		return nil
	}
	blocks := matchBlocks(src)
	toks := src.Tokens
	var offs []Offense
	for _, b := range blocks {
		if b.open.Type != token.DEF {
			continue
		}
		// The last `return` token strictly inside this def block, at the def's own
		// nesting depth, that begins a line, is the trailing return candidate.
		retIdx := lastTrailingReturn(toks, b)
		if retIdx < 0 {
			continue
		}
		rt := toks[retIdx]
		offs = append(offs, Offense{
			CopName:     "Style/RedundantReturn",
			Location:    Location{Line: rt.Line, Column: rt.Col, Length: 6},
			Message:     "Redundant return detected.",
			Severity:    Convention,
			Correctable: true,
		})
	}
	return offs
}

// lastTrailingReturn finds the `return` token that is the final statement of the
// def block b: it is the last statement before b's `end`, at b's depth (not nested
// inside an inner block). Returns the token index, or -1.
func lastTrailingReturn(toks []token.Token, b block) int {
	// The last significant token before the def's `end`. If it belongs to an inner
	// block (its statement is a nested if/while/…), the method's final statement is
	// that block, not a return, so there is nothing to flag.
	i := b.endIdx - 1
	for i > b.openIdx && toks[i].Type == token.NEWLINE {
		i--
	}
	// A trailing `end` means the last statement is a nested block: not a return.
	if toks[i].Type == token.END {
		return -1
	}
	// Scan back to the start of this statement's line; its first significant token
	// must be `return` for the return to be the method's final statement.
	line := toks[i].Line
	for j := i; j > b.openIdx; j-- {
		if toks[j].Line != line {
			break
		}
		if toks[j].Type == token.RETURN {
			// Confirm `return` begins the statement (not e.g. a `x = return`-like
			// mid-line token): the token before it is a NEWLINE or the def opener.
			if toks[j-1].Type == token.NEWLINE || j-1 == b.openIdx {
				return j
			}
		}
	}
	return -1
}

// --- Style/IfUnlessModifier ---------------------------------------------------

// ifUnlessModifierCop flags a multi-line if/unless whose body is a single
// statement (so it could be written as a modifier). It reports at the if/unless
// keyword. It only fires for a plain if/unless with no elsif/else and a one-line
// body, matching the common case the gem flags.
type ifUnlessModifierCop struct{}

func (ifUnlessModifierCop) Name() string { return "Style/IfUnlessModifier" }

func (ifUnlessModifierCop) Inspect(src *Source, _ CopConfig) []Offense {
	blocks := matchBlocks(src)
	toks := src.Tokens
	var offs []Offense
	for _, b := range blocks {
		if b.open.Type != token.IF && b.open.Type != token.UNLESS {
			continue
		}
		if hasElseOrElsif(toks, b) {
			continue
		}
		bodyLines := bodyContentLines(src, toks, b)
		if len(bodyLines) != 1 {
			continue
		}
		kw := "if"
		if b.open.Type == token.UNLESS {
			kw = "unless"
		}
		offs = append(offs, Offense{
			CopName:  "Style/IfUnlessModifier",
			Location: Location{Line: b.open.Line, Column: b.open.Col, Length: len(kw)},
			Message: "Favor modifier " + kw + " usage when having a single-line body. " +
				"Another good alternative is the usage of control flow &&/||.",
			Severity:    Convention,
			Correctable: true,
		})
	}
	return offs
}

// hasElseOrElsif reports whether block b contains an else/elsif at its own depth.
func hasElseOrElsif(toks []token.Token, b block) bool {
	depth := 0
	for i := b.openIdx + 1; i < b.endIdx; i++ {
		switch {
		case blockOpeners[toks[i].Type]:
			depth++
		case toks[i].Type == token.END:
			depth--
		case depth == 0 && (toks[i].Type == token.ELSE || toks[i].Type == token.ELSIF):
			return true
		}
	}
	return false
}

// bodyContentLines returns the distinct source line numbers of the block's body
// (between the opener line and the `end` line, exclusive), that carry content.
func bodyContentLines(src *Source, toks []token.Token, b block) []int {
	openLine := b.open.Line
	endLine := toks[b.endIdx].Line
	seen := map[int]bool{}
	var lines []int
	for i := b.openIdx + 1; i < b.endIdx; i++ {
		l := toks[i].Line
		if toks[i].Type == token.NEWLINE || l == openLine || l == endLine {
			continue
		}
		if !seen[l] {
			seen[l] = true
			lines = append(lines, l)
		}
	}
	return lines
}

// --- Style/GuardClause --------------------------------------------------------

// guardClauseCop flags an if/unless (without an else/elsif) that is the last
// statement of a method body: it could be inverted into an early-return guard
// clause. It reports at the if/unless keyword with the gem's message naming the
// inverted keyword ("return unless" for an if, "return if" for an unless).
type guardClauseCop struct{}

func (guardClauseCop) Name() string { return "Style/GuardClause" }

func (guardClauseCop) Inspect(src *Source, cfg CopConfig) []Offense {
	blocks := matchBlocks(src)
	toks := src.Tokens
	var offs []Offense
	for _, b := range blocks {
		if b.open.Type != token.DEF {
			continue
		}
		ib := lastStatementBlock(blocks, toks, b)
		if ib < 0 {
			continue
		}
		cond := blocks[ib]
		// The last statement must be a plain if/unless (no else/elsif — an if/else
		// is not a guard-clause candidate) whose body meets MinBodyLength.
		if cond.open.Type != token.IF && cond.open.Type != token.UNLESS {
			continue
		}
		if hasElseOrElsif(toks, cond) ||
			len(bodyContentLines(src, toks, cond)) < cfg.Int("MinBodyLength", 1) {
			continue
		}
		inverted := "return unless"
		if cond.open.Type == token.UNLESS {
			inverted = "return if"
		}
		condText := conditionText(src, cond)
		offs = append(offs, Offense{
			CopName:  "Style/GuardClause",
			Location: Location{Line: cond.open.Line, Column: cond.open.Col, Length: len(tokenKeyword(cond.open.Type))},
			Message: "Use a guard clause (" + inverted + " " + condText + ") instead of " +
				"wrapping the code inside a conditional expression.",
			Severity:    Convention,
			Correctable: true,
		})
	}
	return offs
}

// conditionText returns the verbatim source of an if/unless condition: the text
// from just after the keyword to the end of the condition (the keyword's line,
// trimmed, with a trailing `then` removed). It is the placeholder the gem inserts
// into the GuardClause message (e.g. "foo?", "a && b").
func conditionText(src *Source, cond block) string {
	line := src.line(cond.open.Line)
	// Slice after the keyword: cond.open.Col is 1-based start of if/unless.
	start := cond.open.Col - 1 + len(tokenKeyword(cond.open.Type))
	if start > len(line) {
		start = len(line)
	}
	text := strings.TrimSpace(line[start:])
	text = strings.TrimSuffix(text, "then")
	return strings.TrimSpace(text)
}

// lastStatementBlock returns the index into blocks of the block that is the last
// statement of def b's body — i.e. the block whose `end` is the last significant
// token before b's own `end` — or -1 when b's final statement is not a block.
func lastStatementBlock(blocks []block, toks []token.Token, b block) int {
	last := b.endIdx - 1
	for last > b.openIdx && toks[last].Type == token.NEWLINE {
		last--
	}
	if toks[last].Type != token.END {
		return -1
	}
	for i, c := range blocks {
		if c.endIdx == last {
			return i
		}
	}
	return -1
}

// tokenKeyword returns the keyword text for a block-opening token type.
func tokenKeyword(tt token.Type) string {
	if tt == token.UNLESS {
		return "unless"
	}
	return "if"
}
