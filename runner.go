// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import "sort"

// Runner drives a registry of cops over a source with a configuration — the
// counterpart of RuboCop's Commissioner + team. Construct one with NewRunner;
// Inspect and InspectSource return the offenses in RuboCop's report order.
type Runner struct {
	reg *Registry
	cfg *Config
}

// NewRunner returns a Runner over reg and cfg. A nil registry defaults to the
// built-in core cop set (DefaultRegistry); a nil config to the all-defaults
// Config (NewConfig).
func NewRunner(reg *Registry, cfg *Config) *Runner {
	if reg == nil {
		reg = DefaultRegistry()
	}
	if cfg == nil {
		cfg = NewConfig()
	}
	return &Runner{reg: reg, cfg: cfg}
}

// Inspect lexes/parses source (named path for file-naming cops) and runs every
// enabled cop, returning the offenses sorted the way RuboCop reports them.
func (r *Runner) Inspect(path, source string) []Offense {
	return r.InspectSource(NewSource(path, source))
}

// InspectSource runs every enabled cop over an already-built Source. Splitting
// this out lets a host lex/parse once and reuse the Source.
func (r *Runner) InspectSource(src *Source) []Offense {
	var out []Offense
	for _, cop := range r.reg.Cops() {
		cc := r.cfg.For(cop.Name(), defaultCopConfig(cop.Name()))
		if !cc.Enabled {
			continue
		}
		out = append(out, cop.Inspect(src, cc)...)
	}
	sortOffenses(out)
	return out
}

// sortOffenses orders offenses as RuboCop's formatters do: by line, then column,
// then cop name (a stable, deterministic order for golden comparison).
func sortOffenses(offs []Offense) {
	sort.SliceStable(offs, func(i, j int) bool {
		a, b := offs[i], offs[j]
		if a.Location.Line != b.Location.Line {
			return a.Location.Line < b.Location.Line
		}
		if a.Location.Column != b.Location.Column {
			return a.Location.Column < b.Location.Column
		}
		return a.CopName < b.CopName
	})
}

// Autocorrect applies, to source, the corrections of every offense that carries
// one, returning the rewritten source. Non-overlapping corrections are applied
// right-to-left so earlier byte offsets stay valid; overlapping corrections keep
// the first (lowest-offset) and skip the rest, mirroring how RuboCop's corrector
// refuses to clobber an already-edited range in a single pass. It is a reference
// applier for hosts that want the whole edit in one call.
func (r *Runner) Autocorrect(path, source string) string {
	offs := r.Inspect(path, source)
	var corr []Correction
	for _, o := range offs {
		if o.Correction != nil {
			corr = append(corr, *o.Correction)
		}
	}
	sort.SliceStable(corr, func(i, j int) bool { return corr[i].Begin < corr[j].Begin })
	// Drop overlaps (keep the earliest).
	var kept []Correction
	last := -1
	for _, c := range corr {
		if c.Begin < last {
			continue
		}
		kept = append(kept, c)
		last = c.End
	}
	// Apply right-to-left.
	out := source
	for i := len(kept) - 1; i >= 0; i-- {
		c := kept[i]
		if c.Begin < 0 || c.End > len(out) || c.Begin > c.End {
			continue
		}
		out = out[:c.Begin] + c.Replacement + out[c.End:]
	}
	return out
}
