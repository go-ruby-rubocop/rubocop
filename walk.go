// Copyright (c) the go-ruby-rubocop/rubocop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rubocop

import "github.com/go-ruby-parser/parser/ast"

// walk invokes fn for node and, depth-first, every child node reachable through
// the AST shapes the core cops need (statements, expressions, method/block bodies,
// conditionals). It is a compact visitor over go-ruby-parser's ast package; fn
// returning false prunes the traversal below node. Unhandled node kinds are leaves.
func walk(node ast.Node, fn func(ast.Node) bool) {
	if node == nil || !fn(node) {
		return
	}
	switch n := node.(type) {
	case *ast.Program:
		walkAll(n.Body, fn)
	case *ast.Assign:
		walk(n.Value, fn)
	case *ast.OpAssign:
		walk(n.Value, fn)
	case *ast.BinaryExpr:
		walk(n.Left, fn)
		walk(n.Right, fn)
	case *ast.UnaryExpr:
		walk(n.Operand, fn)
	case *ast.Call:
		walk(n.Recv, fn)
		walkAll(n.Args, fn)
		if n.Block != nil {
			walkAll(n.Block.Body, fn)
			walkAll(n.Block.Defaults, fn)
		}
	case *ast.If:
		walk(n.Cond, fn)
		walkAll(n.Then, fn)
		for _, e := range n.Elsifs {
			walk(e.Cond, fn)
			walkAll(e.Body, fn)
		}
		walkAll(n.Else, fn)
	case *ast.While:
		walk(n.Cond, fn)
		walkAll(n.Body, fn)
	case *ast.MethodDef:
		walkAll(n.Defaults, fn)
		walkAll(n.Body, fn)
	case *ast.ClassDef:
		walkAll(n.Body, fn)
	case *ast.ModuleDef:
		walkAll(n.Body, fn)
	case *ast.Return:
		walk(n.Value, fn)
	case *ast.ArrayLit:
		walkAll(n.Elems, fn)
	case *ast.Begin:
		walkAll(n.Body, fn)
	}
}

// walkAll walks each node in nodes.
func walkAll(nodes []ast.Node, fn func(ast.Node) bool) {
	for _, n := range nodes {
		walk(n, fn)
	}
}
