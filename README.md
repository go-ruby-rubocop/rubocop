<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-rubocop/brand/main/social/go-ruby-rubocop-rubocop.png" alt="go-ruby-rubocop/rubocop" width="720"></p>

# rubocop — go-ruby-rubocop

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-rubocop.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the core of Ruby's
[RuboCop](https://rubocop.org) linter** — the offense/cop framework, a faithful
core cop set, the `.rubocop.yml` configuration model and the default formatters —
built on the [go-ruby-parser](https://github.com/go-ruby-parser/parser) Ruby AST
and lexer. It inspects a Ruby **source string** (its lines, its token stream and
its AST) and emits offenses whose cop name, location and message **byte-match**
those the `rubocop` gem produces, **without any Ruby runtime**.

It is the RuboCop backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a **standalone,
reusable** module with no dependency on the Ruby runtime — a sibling of
[go-ruby-parser](https://github.com/go-ruby-parser/parser),
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp),
[go-ruby-erb](https://github.com/go-ruby-erb/erb) and
[go-ruby-yaml](https://github.com/go-ruby-yaml/yaml) (whose YAML loader parses the
`.rubocop.yml` config here).

> **What it is — and isn't.** Detecting an offense from a source string — its cop
> name, its line/column, its message text — is fully deterministic and needs **no
> interpreter**, so it lives here as pure Go. The gem's file-walking and its
> ~500-cop breadth are out of scope: this is a **core** cop set (22 cops across
> Layout / Style / Lint / Metrics) implemented faithfully rather than a fake of
> the whole. Applying autocorrections is left to the host; each cop computes the
> edit (a source-range `Correction`) but does not rewrite your file.

## Features

- **Cop framework** — a `Cop` interface (`Inspect(*Source, CopConfig) []Offense`),
  an `Offense{CopName, Location, Message, Severity, Correctable, Correction}`
  model, a `Registry`, and a `Runner` (the commissioner) that runs every enabled
  cop over a lexed+parsed `Source` and returns offenses in RuboCop's report order.
- **Configuration** — `ParseConfig` reads a `.rubocop.yml` (via go-ruby-yaml) into
  a `Config`: `AllCops.DisabledByDefault`, per-cop `Enabled`, and per-cop params
  (`Max`, `EnforcedStyle`, …). A cop's effective config is its built-in default
  merged with the file's overrides.
- **Formatters** — `SimpleTextFormatter` (`--format simple`) and
  `ProgressFormatter` (`--format progress`), including the byte-faithful summary
  line (`N files inspected, M offenses detected[, K offenses autocorrectable]`).
- **A faithful core cop set** — detection and message text validated
  differentially against the `rubocop` gem (see [Cops](#cops)).

CGO-free, **100% test coverage**, `gofmt` + `go vet` clean, and green across the
six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le, s390x) and three
operating systems.

## Install

```sh
go get github.com/go-ruby-rubocop/rubocop
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-rubocop/rubocop"
)

func main() {
	src := "def foo(a, b)\n  return a + b\nend\n"

	// Parse an optional .rubocop.yml (empty => every cop at its default).
	cfg, _ := rubocop.ParseConfig("")

	// Run the built-in core cop set.
	run := rubocop.NewRunner(rubocop.DefaultRegistry(), cfg)
	offenses := run.Inspect("foo.rb", src)

	// Render like `rubocop --format simple`.
	report := rubocop.SimpleTextFormatter{}.Format([]rubocop.FileResult{
		{Path: "foo.rb", Offenses: offenses},
	})
	fmt.Print(report)
	// == foo.rb ==
	// C:  2:  3: [Correctable] Style/RedundantReturn: Redundant return detected.
	// W:  1: 12: [Correctable] Lint/UnusedMethodArgument: Unused method argument - b. ...
	//
	// 1 file inspected, 2 offenses detected, 2 offenses autocorrectable
}
```

## Cops

Implemented faithfully (detection + message + location validated against the gem):

| Department | Cops |
| ---------- | ---- |
| **Layout** | `TrailingWhitespace`, `TrailingEmptyLines`, `SpaceAfterComma`, `EmptyLines`, `IndentationWidth`, `LineLength` |
| **Style**  | `StringLiterals`, `FrozenStringLiteralComment`, `MethodDefParentheses`, `RedundantReturn`, `Not`, `IfUnlessModifier`, `GuardClause`, `NumericLiterals` |
| **Lint**   | `UselessAssignment`, `UnusedMethodArgument`, `DuplicateMethods`, `AmbiguousOperator`, `ShadowingOuterLocalVariable` |
| **Metrics**| `MethodLength`, `LineLength`, `ClassLength` (each with a configurable `Max`) |

**Not yet ported.** RuboCop bundles ~500 cops; the ~480 beyond the set above are
deliberately absent rather than stubbed. Parameterised cops here honour the
subset of their gem params they document (`Max`, `Width`, `MinDigits`,
`EnforcedStyle`, `MinBodyLength`); the `double_quotes` inversion of
`Style/StringLiterals` and non-`always` styles of a few cops are recognised (they
switch the cop off) but not yet re-implemented.

## Oracle

The test suite is a **differential oracle** against the `rubocop` gem
(version-gated on `RUBY_VERSION >= "4.0"`): each implemented cop runs on a corpus
of Ruby snippets and its offenses (cop name + line/col + message) plus the
formatter output are compared **byte-for-byte** with the gem's. The oracle skips
itself where the gem (or a target-arch Ruby) is absent — the deterministic golden
vectors alone hold coverage at 100%, so every no-Ruby lane still passes the gate.

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright (c) the go-ruby-rubocop/rubocop
authors.

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```
