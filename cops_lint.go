// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
	"fmt"

	"github.com/go-ruby-parser/parser/token"
)

// --- Lint/UselessAssignment ---------------------------------------------------

// uselessAssignmentCop flags a local-variable assignment inside a method whose
// value is never subsequently read in that method. It reports at the assigned
// variable (a Warning), with the gem's "Useless assignment to variable - x."
// message. Detection walks each def block's tokens: an `IDENT = …` at statement
// start whose name never appears again (as a read) before the def's `end`.
type uselessAssignmentCop struct{}

func (uselessAssignmentCop) Name() string { return "Lint/UselessAssignment" }

func (uselessAssignmentCop) Inspect(src *Source, _ CopConfig) []Offense {
	toks := src.Tokens
	var offs []Offense
	for _, b := range matchBlocks(src) {
		if b.open.Type != token.DEF {
			continue
		}
		offs = append(offs, uselessInDef(toks, b)...)
	}
	return offs
}

// uselessInDef reports useless assignments within a single def block b.
func uselessInDef(toks []token.Token, b block) []Offense {
	var offs []Offense
	for i := b.openIdx + 1; i < b.endIdx; i++ {
		if toks[i].Type != token.IDENT {
			continue
		}
		// Must be a plain assignment `IDENT =` at statement start.
		if toks[i+1].Type != token.ASSIGN {
			continue
		}
		if !isStatementStart(toks, i, b) {
			continue
		}
		name := toks[i].Lit
		// Read after this assignment within the def?
		read := false
		for j := i + 2; j < b.endIdx; j++ {
			if toks[j].Type == token.IDENT && toks[j].Lit == name {
				// A later `name =` reassignment (statement start) is not a read.
				if toks[j+1].Type == token.ASSIGN && isStatementStart(toks, j, b) {
					continue
				}
				read = true
				break
			}
		}
		if read {
			continue
		}
		offs = append(offs, Offense{
			CopName:     "Lint/UselessAssignment",
			Location:    Location{Line: toks[i].Line, Column: toks[i].Col, Length: len(name)},
			Message:     fmt.Sprintf("Useless assignment to variable - %s.", name),
			Severity:    Warning,
			Correctable: true,
		})
	}
	return offs
}

// isStatementStart reports whether toks[i] is the first significant token of its
// statement (line start, or just after a NEWLINE), within block b.
func isStatementStart(toks []token.Token, i int, b block) bool {
	if i <= b.openIdx+1 {
		return toks[i-1].Type == token.NEWLINE || i == b.openIdx+1
	}
	return toks[i-1].Type == token.NEWLINE
}

// --- Lint/UnusedMethodArgument ------------------------------------------------

// unusedMethodArgumentCop flags a method parameter never referenced in the body.
// It reports at the parameter token (a Warning). Parameters named with a leading
// underscore are exempt, matching the gem.
type unusedMethodArgumentCop struct{}

func (unusedMethodArgumentCop) Name() string { return "Lint/UnusedMethodArgument" }

func (unusedMethodArgumentCop) Inspect(src *Source, _ CopConfig) []Offense {
	toks := src.Tokens
	var offs []Offense
	for _, b := range matchBlocks(src) {
		if b.open.Type != token.DEF {
			continue
		}
		params := defParamTokens(toks, b)
		for _, pt := range params {
			name := pt.Lit
			if name == "" || name[0] == '_' {
				continue
			}
			if paramUsedInBody(toks, b, params, name) {
				continue
			}
			offs = append(offs, Offense{
				CopName:  "Lint/UnusedMethodArgument",
				Location: Location{Line: pt.Line, Column: pt.Col, Length: len(name)},
				Message: fmt.Sprintf("Unused method argument - %s. "+
					"If it's necessary, use _ or _%s as an argument name to indicate that "+
					"it won't be used. If it's unnecessary, remove it.", name, name),
				Severity:    Warning,
				Correctable: true,
			})
		}
	}
	return offs
}

// defParamTokens returns the IDENT tokens naming the parameters of def block b,
// scanning the parenthesised signature on the def line.
func defParamTokens(toks []token.Token, b block) []token.Token {
	// Find the '(' after the def name; params run to the matching ')'.
	i := b.openIdx + 1
	for i < b.endIdx && toks[i].Type != token.LPAREN && toks[i].Line == b.open.Line {
		i++
	}
	if i >= b.endIdx || toks[i].Type != token.LPAREN {
		return nil
	}
	var params []token.Token
	depth := 0
	expectName := true
	for ; i < b.endIdx; i++ {
		switch toks[i].Type {
		case token.LPAREN:
			depth++
		case token.RPAREN:
			depth--
			if depth == 0 {
				return params
			}
		case token.COMMA:
			if depth == 1 {
				expectName = true
			}
		case token.IDENT:
			if depth == 1 && expectName {
				params = append(params, toks[i])
				expectName = false
			}
		default:
			if depth == 1 {
				expectName = false
			}
		}
	}
	return params
}

// paramUsedInBody reports whether name appears as an IDENT in b's body after the
// signature (i.e. after the params' last position), excluding the param tokens
// themselves.
func paramUsedInBody(toks []token.Token, b block, params []token.Token, name string) bool {
	// Body starts after the closing ')' of the signature (or the def name line).
	start := b.openIdx + 1
	if len(params) > 0 {
		// Advance start past the signature line.
		for start < b.endIdx && toks[start].Line == b.open.Line {
			start++
		}
	}
	for j := start; j < b.endIdx; j++ {
		if toks[j].Type == token.IDENT && toks[j].Lit == name {
			return true
		}
	}
	return false
}

// --- Lint/DuplicateMethods ----------------------------------------------------

// duplicateMethodsCop flags a method defined more than once in the same scope. It
// reports at the second (and later) definition with the gem's message naming both
// definition lines and the enclosing scope (Object for top-level defs). The file
// path in the message is src.Path.
type duplicateMethodsCop struct{}

func (duplicateMethodsCop) Name() string { return "Lint/DuplicateMethods" }

func (duplicateMethodsCop) Inspect(src *Source, _ CopConfig) []Offense {
	toks := src.Tokens
	blocks := matchBlocks(src)
	var offs []Offense
	// Map scope+name -> first def token.
	type key struct {
		scope int // openIdx of enclosing class/module block, or -1 for top level
		name  string
	}
	first := map[key]token.Token{}
	scopeName := map[int]string{-1: "Object"}
	for _, b := range blocks {
		if b.open.Type != token.CLASS && b.open.Type != token.MODULE {
			continue
		}
		if nt := nextConstName(toks, b); nt != "" {
			scopeName[b.openIdx] = nt
		}
	}
	for _, b := range blocks {
		if b.open.Type != token.DEF {
			continue
		}
		nameTok, ok := defNameToken(toks, b)
		if !ok {
			continue
		}
		scope := enclosingScope(blocks, b)
		k := key{scope: scope, name: nameTok.Lit}
		if prev, seen := first[k]; seen {
			sn := scopeName[scope]
			path := src.Path
			offs = append(offs, Offense{
				CopName:  "Lint/DuplicateMethods",
				Location: Location{Line: b.open.Line, Column: b.open.Col, Length: 3},
				Message: fmt.Sprintf("Method %s#%s is defined at both %s:%d and %s:%d.",
					sn, nameTok.Lit, path, prev.Line, path, b.open.Line),
				Severity: Warning,
			})
		} else {
			first[k] = b.open
		}
	}
	return offs
}

// defNameToken returns the method-name IDENT/CONST token of def block b.
func defNameToken(toks []token.Token, b block) (token.Token, bool) {
	j := b.openIdx + 1
	// Skip a self./Recv. receiver.
	if j+1 < len(toks) && (toks[j].Type == token.SELF || toks[j].Type == token.CONST || toks[j].Type == token.IDENT) && toks[j+1].Type == token.DOT {
		j += 2
	}
	if j < len(toks) && (toks[j].Type == token.IDENT || toks[j].Type == token.CONST) {
		return toks[j], true
	}
	return token.Token{}, false
}

// nextConstName returns the CONST name following a class/module keyword.
func nextConstName(toks []token.Token, b block) string {
	if b.openIdx+1 < len(toks) && toks[b.openIdx+1].Type == token.CONST {
		return toks[b.openIdx+1].Lit
	}
	return ""
}

// enclosingScope returns the openIdx of the innermost class/module block that
// contains b, or -1 for top level.
func enclosingScope(blocks []block, b block) int {
	best := -1
	bestSpan := 1 << 30
	for _, c := range blocks {
		if c.open.Type != token.CLASS && c.open.Type != token.MODULE {
			continue
		}
		if c.openIdx < b.openIdx && c.endIdx > b.endIdx {
			if span := c.endIdx - c.openIdx; span < bestSpan {
				bestSpan = span
				best = c.openIdx
			}
		}
	}
	return best
}

// --- Lint/AmbiguousOperator ---------------------------------------------------

// ambiguousOperatorCop flags a `*`/`**`/`&` that follows a bareword command call
// with a space before but none after — ambiguous between a splat/block-pass
// argument and a binary operator (`foo *x`). It reports at the operator (a
// Warning) with the gem's splat/block-pass wording.
type ambiguousOperatorCop struct{}

func (ambiguousOperatorCop) Name() string { return "Lint/AmbiguousOperator" }

func (ambiguousOperatorCop) Inspect(src *Source, _ CopConfig) []Offense {
	toks := src.Tokens
	var offs []Offense
	for i := 1; i < len(toks)-1; i++ {
		t := toks[i]
		msg := ""
		switch t.Type {
		case token.STAR:
			msg = "Ambiguous splat operator. Parenthesize the method arguments if it's " +
				"surely a splat operator, or add a whitespace to the right of the * if it " +
				"should be a multiplication."
		case token.POW:
			msg = "Ambiguous double splat operator. Parenthesize the method arguments if " +
				"it's surely a double splat operator, or add a whitespace to the right of " +
				"the ** if it should be a exponent."
		case token.AMPER:
			msg = "Ambiguous block operator. Parenthesize the method arguments if it's " +
				"surely a block operator, or add a whitespace to the right of the & if it " +
				"should be a binary AND."
		default:
			continue
		}
		prev := toks[i-1]
		next := toks[i+1]
		// bareword command: prev is an IDENT method name, operator has a space
		// before and the operand hugs it (no space after).
		if prev.Type != token.IDENT {
			continue
		}
		if !t.SpaceBefore || next.SpaceBefore || next.Line != t.Line {
			continue
		}
		offs = append(offs, Offense{
			CopName:     "Lint/AmbiguousOperator",
			Location:    Location{Line: t.Line, Column: t.Col, Length: len(t.Type.String())},
			Message:     msg,
			Severity:    Warning,
			Correctable: true,
		})
	}
	return offs
}

// --- Lint/ShadowingOuterLocalVariable -----------------------------------------

// shadowingOuterLocalVariableCop flags a block parameter whose name is an
// already-defined outer local variable. It reports at the block parameter (a
// Warning). Outer locals are the top-level `name =` assignments preceding the
// block; block params are the identifiers between the `|…|` of a do/brace block.
type shadowingOuterLocalVariableCop struct{}

func (shadowingOuterLocalVariableCop) Name() string { return "Lint/ShadowingOuterLocalVariable" }

func (shadowingOuterLocalVariableCop) Inspect(src *Source, _ CopConfig) []Offense {
	toks := src.Tokens
	var offs []Offense
	outer := map[string]bool{}
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		// Track outer local assignments at statement start.
		if t.Type == token.IDENT && i+1 < len(toks) && toks[i+1].Type == token.ASSIGN &&
			(i == 0 || toks[i-1].Type == token.NEWLINE) {
			outer[t.Lit] = true
			continue
		}
		// A block-param list opens with a PIPE that follows `do` or `{`.
		if t.Type == token.PIPE && i > 0 && (toks[i-1].Type == token.DO || toks[i-1].Type == token.LBRACE) {
			for j := i + 1; j < len(toks) && toks[j].Type != token.PIPE; j++ {
				if toks[j].Type == token.IDENT && outer[toks[j].Lit] {
					offs = append(offs, Offense{
						CopName:  "Lint/ShadowingOuterLocalVariable",
						Location: Location{Line: toks[j].Line, Column: toks[j].Col, Length: len(toks[j].Lit)},
						Message:  fmt.Sprintf("Shadowing outer local variable - %s.", toks[j].Lit),
						Severity: Warning,
					})
				}
			}
		}
	}
	return offs
}
