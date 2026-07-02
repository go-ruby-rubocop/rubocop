// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import (
	"fmt"

	"github.com/go-ruby-yaml/yaml"
)

// CopConfig is the resolved configuration for a single cop: whether it is enabled
// and its parameter map (Max, EnforcedStyle, …). Params values are the Go shapes
// go-ruby-yaml decodes YAML scalars to (int, float64, bool, string, []any).
type CopConfig struct {
	Enabled bool
	Params  map[string]any
}

// Int returns the integer parameter key, or def when it is absent or not an
// integer. RuboCop's numeric params (Max, …) are plain integers.
func (c CopConfig) Int(key string, def int) int {
	if v, ok := c.Params[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return def
}

// Str returns the string parameter key, or def when absent / not a string.
func (c CopConfig) Str(key, def string) string {
	if v, ok := c.Params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

// Bool returns the boolean parameter key, or def when absent / not a bool.
func (c CopConfig) Bool(key string, def bool) bool {
	if v, ok := c.Params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

// Config is a whole-run configuration: the AllCops defaults plus per-cop
// overrides parsed from a .rubocop.yml document. A cop's effective CopConfig is
// its built-in defaults merged with any overrides here (overrides win).
type Config struct {
	// disabledByDefault mirrors AllCops.DisabledByDefault: when true a cop is
	// enabled only if its own entry sets Enabled: true.
	disabledByDefault bool
	// cops holds the raw per-cop entries keyed by qualified cop name.
	cops map[string]rawCop
}

// rawCop is one cop's entry as read from YAML before merging with defaults.
type rawCop struct {
	enabledSet bool
	enabled    bool
	params     map[string]any
}

// NewConfig returns an empty Config: every cop takes its built-in default
// (enabled, default params). This is the "no .rubocop.yml present" case.
func NewConfig() *Config {
	return &Config{cops: map[string]rawCop{}}
}

// ParseConfig parses a .rubocop.yml document into a Config. It understands the
// subset RuboCop configs use in practice: an AllCops section (DisabledByDefault)
// and per-cop sections carrying Enabled plus arbitrary scalar/param keys. Unknown
// keys are preserved as params so parameterised cops can read them. A YAML syntax
// error is returned; an empty document yields the empty (all-default) Config.
func ParseConfig(src string) (*Config, error) {
	cfg := NewConfig()
	if src == "" {
		return cfg, nil
	}
	v, err := yaml.Load(src)
	if err != nil {
		return nil, fmt.Errorf("rubocop: parsing config: %w", err)
	}
	root, ok := v.(*yaml.Map)
	if !ok {
		// A non-mapping document (a bare scalar / null) carries no cop config.
		return cfg, nil
	}
	for _, pair := range root.Pairs() {
		key, ok := pair.Key.(string)
		if !ok {
			continue
		}
		section, ok := pair.Val.(*yaml.Map)
		if !ok {
			continue
		}
		if key == "AllCops" {
			cfg.applyAllCops(section)
			continue
		}
		cfg.cops[key] = readCopSection(section)
	}
	return cfg, nil
}

// applyAllCops reads the AllCops section's fields Config understands.
func (c *Config) applyAllCops(section *yaml.Map) {
	if v, ok := section.Get("DisabledByDefault"); ok {
		if b, ok := v.(bool); ok {
			c.disabledByDefault = b
		}
	}
}

// readCopSection turns a per-cop YAML mapping into a rawCop: Enabled is pulled out
// (recording whether it was present), every other key becomes a param.
func readCopSection(section *yaml.Map) rawCop {
	rc := rawCop{params: map[string]any{}}
	for _, p := range section.Pairs() {
		k, ok := p.Key.(string)
		if !ok {
			continue
		}
		if k == "Enabled" {
			if b, ok := p.Val.(bool); ok {
				rc.enabledSet = true
				rc.enabled = b
			}
			continue
		}
		rc.params[k] = normalizeParam(p.Val)
	}
	return rc
}

// normalizeParam collapses go-ruby-yaml's sequence value into a plain []any and
// otherwise passes the value through; params are consumed via CopConfig accessors.
func normalizeParam(v any) any {
	if m, ok := v.(*yaml.Map); ok {
		out := map[string]any{}
		for _, p := range m.Pairs() {
			if k, ok := p.Key.(string); ok {
				out[k] = normalizeParam(p.Val)
			}
		}
		return out
	}
	return v
}

// For resolves the effective CopConfig for cop name, merging def (the cop's
// built-in defaults) with any override from the parsed config. Enabled follows
// RuboCop's precedence: an explicit per-cop Enabled always wins; otherwise the cop
// is on unless AllCops.DisabledByDefault turned everything off. Params from the
// override are layered over the defaults key-by-key.
func (c *Config) For(name string, def CopConfig) CopConfig {
	out := CopConfig{Enabled: def.Enabled, Params: map[string]any{}}
	for k, v := range def.Params {
		out.Params[k] = v
	}
	if c.disabledByDefault {
		out.Enabled = false
	}
	if rc, ok := c.cops[name]; ok {
		if rc.enabledSet {
			out.Enabled = rc.enabled
		} else if !c.disabledByDefault {
			out.Enabled = def.Enabled
		}
		for k, v := range rc.params {
			out.Params[k] = v
		}
	}
	return out
}
