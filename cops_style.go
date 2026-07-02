// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
	"strings"

	"github.com/go-ruby-parser/parser/token"
)

// --- Style/StringLiterals -----------------------------------------------------

// stringLiteralsCop enforces single-quoted strings (the default EnforcedStyle)
// when no interpolation or escapes require double quotes. It flags a plain
// double-quoted STRING token whose content has no backslash escape (a `\n`, `\t`,
// … that only a double-quoted string can express). Interpolated strings are lexed
// as STRBEG/STRMID/STREND and are never flagged.
type stringLiteralsCop struct{}

func (stringLiteralsCop) Name() string { return "Style/StringLiterals" }

func (stringLiteralsCop) Inspect(src *Source, cfg CopConfig) []Offense {
	// Only the default single_quotes style is implemented; double_quotes is a
	// host-configurable inversion left for a later pass.
	if cfg.Str("EnforcedStyle", "single_quotes") != "single_quotes" {
		return nil
	}
	var offs []Offense
	for _, t := range src.Tokens {
		if t.Type != token.STRING {
			continue
		}
		if !isDoubleQuotedInSource(src, t) {
			continue
		}
		// A backslash escape in the literal content means double quotes are needed.
		if strings.ContainsRune(t.Lit, '\\') {
			continue
		}
		offs = append(offs, Offense{
			CopName:     "Style/StringLiterals",
			Location:    Location{Line: t.Line, Column: t.Col, Length: len(t.Lit) + 2},
			Message:     "Prefer single-quoted strings when you don't need string interpolation or special symbols.",
			Severity:    Convention,
			Correctable: true,
		})
	}
	return offs
}

// isDoubleQuotedInSource reports whether the STRING token t was written with
// double quotes, by inspecting the opening delimiter byte at its position (the
// token carries the decoded content, not the delimiter).
func isDoubleQuotedInSource(src *Source, t token.Token) bool {
	off := src.offsetOf(t.Line, t.Col)
	if off < len(src.Raw) {
		return src.Raw[off] == '"'
	}
	return false
}

// --- Style/FrozenStringLiteralComment -----------------------------------------

// frozenStringLiteralCommentCop (default style "always") flags source that lacks a
// `# frozen_string_literal: true` magic comment. The comment must appear on the
// first line, or the second when the first is a shebang. Reports at 1:1.
type frozenStringLiteralCommentCop struct{}

func (frozenStringLiteralCommentCop) Name() string { return "Style/FrozenStringLiteralComment" }

func (frozenStringLiteralCommentCop) Inspect(src *Source, cfg CopConfig) []Offense {
	if cfg.Str("EnforcedStyle", "always") != "always" {
		return nil
	}
	if len(src.Lines) == 0 {
		return nil
	}
	// Locate the candidate magic-comment line (line 1, or 2 after a shebang).
	idx := 0
	if strings.HasPrefix(src.Lines[0], "#!") && len(src.Lines) > 1 {
		idx = 1
	}
	for i := idx; i < len(src.Lines); i++ {
		line := strings.TrimSpace(src.Lines[i])
		if line == "" {
			continue
		}
		if isFrozenMagicComment(line) {
			return nil // already present
		}
		// The first non-blank line is not a magic comment: comment missing.
		break
	}
	return []Offense{{
		CopName:     "Style/FrozenStringLiteralComment",
		Location:    Location{Line: 1, Column: 1},
		Message:     "Missing frozen string literal comment.",
		Severity:    Convention,
		Correctable: true,
		Correction:  &Correction{Begin: 0, End: 0, Replacement: "# frozen_string_literal: true\n"},
	}}
}

// isFrozenMagicComment reports whether line is a frozen_string_literal magic
// comment (any boolean value satisfies "present").
func isFrozenMagicComment(line string) bool {
	if !strings.HasPrefix(line, "#") {
		return false
	}
	body := strings.TrimSpace(strings.TrimPrefix(line, "#"))
	return strings.HasPrefix(body, "frozen_string_literal:")
}

// --- Style/MethodDefParentheses -----------------------------------------------

// methodDefParenthesesCop (default require_parentheses) flags a `def` with
// parameters but no parentheses (`def foo a` / `def foo a, b`). It reports at the
// first parameter. Implemented on the token stream so the exact `(`-absence and
// parameter column are available.
type methodDefParenthesesCop struct{}

func (methodDefParenthesesCop) Name() string { return "Style/MethodDefParentheses" }

func (methodDefParenthesesCop) Inspect(src *Source, cfg CopConfig) []Offense {
	if cfg.Str("EnforcedStyle", "require_parentheses") != "require_parentheses" {
		return nil
	}
	var offs []Offense
	toks := src.Tokens
	for i := 0; i < len(toks); i++ {
		if toks[i].Type != token.DEF {
			continue
		}
		// Walk the method-name portion: IDENT/CONST/operator name, optional recv.
		j := i + 1
		// Skip a `self.` / `Recv.` receiver.
		if j+1 < len(toks) && (toks[j].Type == token.SELF || toks[j].Type == token.CONST || toks[j].Type == token.IDENT) && toks[j+1].Type == token.DOT {
			j += 2
		}
		if j >= len(toks) {
			continue
		}
		// The method name token.
		nameTok := toks[j]
		if nameTok.Type != token.IDENT && nameTok.Type != token.CONST {
			continue
		}
		after := toks[j+1]
		// Parenthesised def: next is '('. No offense.
		if after.Type == token.LPAREN {
			continue
		}
		// End of signature (newline / `;` / `=` for endless def) with no params.
		if after.Type == token.NEWLINE || after.Type == token.EOF || after.Type == token.ASSIGN {
			continue
		}
		// A bare parameter follows the name on the same line: offense at it.
		if after.Line == nameTok.Line {
			offs = append(offs, Offense{
				CopName:     "Style/MethodDefParentheses",
				Location:    Location{Line: after.Line, Column: after.Col, Length: len(after.Lit)},
				Message:     "Use def with parentheses when there are parameters.",
				Severity:    Convention,
				Correctable: true,
			})
		}
	}
	return offs
}

// --- Style/Not ----------------------------------------------------------------

// notCop flags the low-precedence `not` keyword, recommending `!`. Reports at the
// `not` token.
type notCop struct{}

func (notCop) Name() string { return "Style/Not" }

func (notCop) Inspect(src *Source, _ CopConfig) []Offense {
	var offs []Offense
	for _, t := range src.Tokens {
		if t.Type != token.NOT {
			continue
		}
		offs = append(offs, Offense{
			CopName:     "Style/Not",
			Location:    Location{Line: t.Line, Column: t.Col, Length: 3},
			Message:     "Use ! instead of not.",
			Severity:    Convention,
			Correctable: true,
		})
	}
	return offs
}

// --- Style/NumericLiterals ----------------------------------------------------

// numericLiteralsCop flags a decimal integer literal of at least MinDigits digits
// (default 5) that is not grouped with underscores every three digits. It reports
// at the literal and offers the underscored form as a correction.
type numericLiteralsCop struct{}

func (numericLiteralsCop) Name() string { return "Style/NumericLiterals" }

func (numericLiteralsCop) Inspect(src *Source, cfg CopConfig) []Offense {
	minDigits := cfg.Int("MinDigits", 5)
	var offs []Offense
	for _, t := range src.Tokens {
		if t.Type != token.INT {
			continue
		}
		lit := t.Lit
		// Only plain decimal literals are grouped (skip 0x/0b/0o and any that
		// already contain an underscore or a non-decimal digit).
		if strings.HasPrefix(lit, "0x") || strings.HasPrefix(lit, "0b") ||
			strings.HasPrefix(lit, "0o") || strings.HasPrefix(lit, "0X") ||
			strings.HasPrefix(lit, "0B") || strings.HasPrefix(lit, "0O") {
			continue
		}
		if !allDecimalDigits(lit) {
			continue
		}
		if len(lit) < minDigits {
			continue
		}
		grouped := groupThousands(lit)
		if grouped == lit {
			continue
		}
		begin := src.offsetOf(t.Line, t.Col)
		offs = append(offs, Offense{
			CopName:     "Style/NumericLiterals",
			Location:    Location{Line: t.Line, Column: t.Col, Length: len(lit)},
			Message:     "Use underscores(_) as thousands separator and separate every 3 digits with them.",
			Severity:    Convention,
			Correctable: true,
			Correction:  &Correction{Begin: begin, End: begin + len(lit), Replacement: grouped},
		})
	}
	return offs
}

// allDecimalDigits reports whether s is entirely ASCII decimal digits.
func allDecimalDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// groupThousands inserts underscores every three digits from the right.
func groupThousands(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	rem := n % 3
	if rem > 0 {
		b.WriteString(s[:rem])
	}
	for i := rem; i < n; i += 3 {
		if b.Len() > 0 {
			b.WriteByte('_')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
