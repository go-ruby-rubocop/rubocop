// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
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
		retIdx := lastTrailingReturn(toks, blocks, b)
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
func lastTrailingReturn(toks []token.Token, blocks []block, b block) int {
	// The statement immediately preceding `end` at this depth. Walk backwards from
	// end-1 skipping NEWLINE and any fully-nested inner block.
	i := b.endIdx - 1
	for i > b.openIdx {
		t := toks[i]
		if t.Type == token.NEWLINE {
			i--
			continue
		}
		if t.Type == token.END {
			// Jump to the matching opener of this inner block.
			inner := blockEndingAt(blocks, i)
			if inner >= 0 {
				i = blocks[inner].openIdx - 1
				continue
			}
		}
		break
	}
	// Now scan back to the start of this statement's line and check its first
	// significant token is `return`.
	line := toks[i].Line
	for j := i; j > b.openIdx; j-- {
		if toks[j].Line != line {
			break
		}
		if toks[j].Type == token.RETURN {
			return j
		}
	}
	return -1
}

// blockEndingAt returns the index into blocks of the block whose end is endIdx.
func blockEndingAt(blocks []block, endIdx int) int {
	for i, b := range blocks {
		if b.endIdx == endIdx {
			return i
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

// guardClauseCop flags an if/unless that wraps the entire body of a method, which
// could be inverted into an early-return guard clause. It reports at the if/unless
// keyword with the gem's message naming the inverted keyword.
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
		// The def's body must be exactly one if/unless block spanning the whole body.
		inner := soleInnerBlock(blocks, b)
		if inner < 0 {
			continue
		}
		ib := blocks[inner]
		if ib.open.Type != token.IF && ib.open.Type != token.UNLESS {
			continue
		}
		if hasElseOrElsif(toks, ib) {
			continue // an if/else is not a guard-clause candidate here
		}
		if len(bodyContentLines(src, toks, ib)) < cfg.Int("MinBodyLength", 1) {
			continue
		}
		inverted := "return unless"
		if ib.open.Type == token.UNLESS {
			inverted = "return if"
		}
		offs = append(offs, Offense{
			CopName:  "Style/GuardClause",
			Location: Location{Line: ib.open.Line, Column: ib.open.Col, Length: len(tokenKeyword(ib.open.Type))},
			Message: "Use a guard clause (" + inverted + " x) instead of wrapping the " +
				"code inside a conditional expression.",
			Severity: Convention,
		})
	}
	return offs
}

// soleInnerBlock returns the index of the only block directly nested in b (at b's
// body depth), or -1 if there is not exactly one such block or other statements
// share the body.
func soleInnerBlock(blocks []block, b block) int {
	found := -1
	for i, c := range blocks {
		if c.openIdx > b.openIdx && c.endIdx < b.endIdx {
			// Directly nested (no intermediate block contains c but not b).
			direct := true
			for _, d := range blocks {
				if d.openIdx > b.openIdx && d.endIdx < b.endIdx &&
					d.openIdx < c.openIdx && d.endIdx > c.endIdx {
					direct = false
					break
				}
			}
			if direct {
				if found >= 0 {
					return -1
				}
				found = i
			}
		}
	}
	return found
}

// tokenKeyword returns the keyword text for a block-opening token type.
func tokenKeyword(tt token.Type) string {
	if tt == token.UNLESS {
		return "unless"
	}
	return "if"
}
