// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
	"fmt"
	"strings"
)

// Formatter renders a run's offenses to text. The two built-ins reproduce
// RuboCop's `--format simple` (SimpleTextFormatter) and `--format progress`
// (ProgressFormatter) output byte-for-byte, including the summary line.
type Formatter interface {
	// Format renders the offenses for the files in results (in the given order)
	// plus the trailing summary, returning the complete report text.
	Format(results []FileResult) string
}

// FileResult pairs a file path with the offenses found in it (in report order).
type FileResult struct {
	Path     string
	Offenses []Offense
}

// summaryLine builds RuboCop's "N file(s) inspected, M offense(s) detected[,
// K offense(s) autocorrectable]" tail, matching the gem's pluralisation.
func summaryLine(results []FileResult) string {
	files := len(results)
	offenses, correctable := 0, 0
	for _, r := range results {
		offenses += len(r.Offenses)
		for _, o := range r.Offenses {
			if o.Correctable {
				correctable++
			}
		}
	}
	s := fmt.Sprintf("%d %s inspected, %d %s detected",
		files, pluralize(files, "file"), offenses, pluralize(offenses, "offense"))
	if correctable > 0 {
		s += fmt.Sprintf(", %d %s autocorrectable",
			correctable, pluralize(correctable, "offense"))
	}
	return s
}

// pluralize returns "word" for n==1 and "words" otherwise (English 's' plural).
func pluralize(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// SimpleTextFormatter reproduces `rubocop --format simple`: for each file with
// offenses a "== path ==" header followed by one aligned line per offense, then a
// blank line and the summary.
type SimpleTextFormatter struct{}

// Format renders results in the simple text style.
func (SimpleTextFormatter) Format(results []FileResult) string {
	var b strings.Builder
	for _, r := range results {
		if len(r.Offenses) == 0 {
			continue
		}
		fmt.Fprintf(&b, "== %s ==\n", r.Path)
		for _, o := range r.Offenses {
			mark := ""
			if o.Correctable {
				mark = "[Correctable] "
			}
			fmt.Fprintf(&b, "%s:%3d:%3d: %s%s: %s\n",
				o.Severity.code(), o.Location.Line, o.Location.Column,
				mark, o.CopName, o.Message)
		}
	}
	b.WriteString("\n")
	b.WriteString(summaryLine(results))
	b.WriteString("\n")
	return b.String()
}

// ProgressFormatter reproduces `rubocop --format progress`: a run of per-file
// status dots/letters (`.` clean, or the offenses' severity codes), a blank line,
// an "Offenses:" listing in clang style, a blank line, then the summary.
type ProgressFormatter struct{}

// Format renders results in the progress style.
func (ProgressFormatter) Format(results []FileResult) string {
	var b strings.Builder
	// Status line: one marker per file.
	for _, r := range results {
		if len(r.Offenses) == 0 {
			b.WriteString(".")
			continue
		}
		for _, o := range r.Offenses {
			b.WriteString(o.Severity.code())
		}
	}
	b.WriteString("\n")
	// Offense detail (clang style), only when there are offenses.
	total := 0
	for _, r := range results {
		total += len(r.Offenses)
	}
	if total > 0 {
		b.WriteString("\nOffenses:\n\n")
		for _, r := range results {
			for _, o := range r.Offenses {
				fmt.Fprintf(&b, "%s:%s\n", r.Path, o.String())
			}
		}
	}
	b.WriteString("\n")
	b.WriteString(summaryLine(results))
	b.WriteString("\n")
	return b.String()
}
