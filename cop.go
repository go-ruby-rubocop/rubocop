// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import "sort"

// Cop is a single lint rule. Inspect examines a Source and returns the offenses it
// finds, in source order. It receives its resolved per-cop configuration (the
// merged params from .rubocop.yml plus the cop's own defaults) so parameterised
// cops (Metrics/*, Layout/LineLength, …) read their Max and friends from it.
//
// This mirrors RuboCop's cop protocol: the gem's cops implement on_<node>
// callbacks the commissioner dispatches while walking the AST, but the contract
// that matters to a consumer is the same — given a source, yield offenses. A cop
// here walks whichever view (lines, tokens, AST) it needs itself; the Runner is
// the commissioner that invokes each enabled cop once per source.
type Cop interface {
	// Name is the department-qualified cop name, e.g. "Style/StringLiterals".
	Name() string
	// Inspect returns the offenses this cop finds in src, in source order.
	Inspect(src *Source, cfg CopConfig) []Offense
}

// Registry is an ordered, name-indexed set of cops. The zero Registry is not
// usable; construct one with NewRegistry (empty) or DefaultRegistry (the built-in
// core cop set). Registration is idempotent-by-replacement: registering a cop
// whose name already exists replaces it, so a host can override a built-in.
type Registry struct {
	byName map[string]Cop
	order  []string
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byName: map[string]Cop{}}
}

// Register adds (or replaces) a cop. It returns the registry for chaining.
func (r *Registry) Register(cops ...Cop) *Registry {
	for _, c := range cops {
		if _, ok := r.byName[c.Name()]; !ok {
			r.order = append(r.order, c.Name())
		}
		r.byName[c.Name()] = c
	}
	return r
}

// Get returns the cop registered under name and whether it was present.
func (r *Registry) Get(name string) (Cop, bool) {
	c, ok := r.byName[name]
	return c, ok
}

// Names returns the registered cop names sorted the way RuboCop orders them for
// output: by department then cop name (i.e. plain lexicographic on the qualified
// name), which is the order the formatters group offenses within a line.
func (r *Registry) Names() []string {
	out := append([]string(nil), r.order...)
	sort.Strings(out)
	return out
}

// Cops returns the registered cops in Names() order.
func (r *Registry) Cops() []Cop {
	names := r.Names()
	out := make([]Cop, 0, len(names))
	for _, n := range names {
		out = append(out, r.byName[n])
	}
	return out
}
