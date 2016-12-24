package cautiouspancake

import (
	"fmt"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// CallGraph keeps track of 'pure' and function
type CallGraph struct {
	Prog      *loader.Program
	CallGraph *callgraph.Graph
	Mapping   map[*ssa.Function]*Node
}

// Node contains details on why a function is marked as not pure
type Node struct {
	CallGraph     *CallGraph
	RuleBasic     *token.Pos
	RuleInterface *token.Pos
	RuleCallee    *ssa.Function
}

// NewCallGraph creates a CallGraph by performing ssa analysis
func NewCallGraph(iprog *loader.Program) *CallGraph {
	// Create and build SSA-form program representation
	prog := ssautil.CreateProgram(iprog, 0)
	prog.Build()

	// cha to find dynamic calls
	cgo := cha.CallGraph(prog)
	cgo.DeleteSyntheticNodes()

	return &CallGraph{
		Prog:      iprog,
		CallGraph: cgo,
		Mapping:   make(map[*ssa.Function]*Node),
	}
}

func (n *Node) Pure() bool {
	return (n.RuleBasic == nil && n.RuleInterface == nil && n.RuleCallee == nil)
}

func (n *Node) String() string {
	if n.RuleBasic != nil {
		return fmt.Sprintf("Impure - basic - %s", n.CallGraph.Prog.Fset.Position(*n.RuleBasic))
	}
	if n.RuleInterface != nil {
		return fmt.Sprintf("Impure - iface - %s", n.CallGraph.Prog.Fset.Position(*n.RuleInterface))
	}
	if n.RuleCallee != nil {
		return fmt.Sprintf("Impure - calls - %s", n.RuleCallee)
	}
	return "Pure"
}

// Analyze examines the callgraph looking for functions that modify global
// state or call functions that modify global state
func (cg *CallGraph) Analyze() error {
	// basic analysis
	for fn, _ := range cg.CallGraph.Nodes {
		n := &Node{
			CallGraph: cg,
		}
		if pos := accessGlobal(fn); pos != nil {
			n.RuleBasic = pos
		}
		if pos := usesInterface(fn); pos != nil {
			n.RuleInterface = pos
		}
		cg.Mapping[fn] = n
	}

	// sub function analysis, if a func calls an func that is not pure then it
	// needs to be marked as such
	if err := callgraph.GraphVisitEdges(cg.CallGraph, func(edge *callgraph.Edge) error {
		// analyze callee (ssa.Function)
		callee := edge.Callee.Func
		caller := edge.Caller.Func

		// callee modifies state, mark static callers bad as well
		if !cg.Mapping[callee].Pure() {
			if dynamic(edge) {
				if !includes(edge.Site.Common().Value, caller.Params) {
					cg.Mapping[edge.Caller.Func].RuleCallee = callee
				}
			} else {
				cg.Mapping[edge.Caller.Func].RuleCallee = callee
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// Pure returns a slice of the ssa representations of the pure functions in the
// analyzed package
func (cg *CallGraph) Pure() []*ssa.Function {
	// filter output to only include functions imported by the program
	var filtered []*ssa.Function

	for k, v := range cg.Mapping {
		if k == nil {
			continue
		}
		_, ok := cg.Prog.Imported[k.Package().Pkg.Path()]
		if ok && v.Pure() {
			filtered = append(filtered, k)
		}
	}

	return filtered
}

// Impure returns a slice of the ssa representations of the impure functions
// in the analyzed package
func (cg *CallGraph) Impure() []*ssa.Function {
	// filter output to only include functions imported by the program
	var filtered []*ssa.Function

	for k, v := range cg.Mapping {
		if k == nil {
			continue
		}
		_, ok := cg.Prog.Imported[k.Package().Pkg.Path()]
		if ok && !v.Pure() {
			filtered = append(filtered, k)
		}
	}

	return filtered
}

// accessGlobal does a simple analysis of the ssa representation of a function
// to detect whether a global variable is read or modified
func accessGlobal(f *ssa.Function) *token.Pos {
	if f == nil {
		// nil functions probably do not modify state?
		return nil
	}

	for _, b := range f.Blocks {
		for _, i := range b.Instrs {
			switch v := i.(type) {
			case *ssa.Store:
				if _, ok := v.Addr.(*ssa.Global); ok {
					return tokenPtr(v.Pos())
				}
			case *ssa.UnOp:
				if _, ok := v.X.(*ssa.Global); ok {
					return tokenPtr(v.Pos())
				}
			}
		}
	}
	return nil
}

// usesInterface eliminates functions that accept an interface as a parameter
// since it is difficult to resolve whether the interface's implementation will
// modify global state
func usesInterface(f *ssa.Function) *token.Pos {
	if f == nil {
		return nil
	}
	for _, v := range f.Params {
		// TODO: check structs for iface parameters
		if types.IsInterface(v.Type()) {
			return tokenPtr(v.Pos())
		}
	}
	return nil
}

// dynamic is a helper that checks whether the callgraph edge is a dynamic call
func dynamic(edge *callgraph.Edge) bool {
	return edge.Site != nil && edge.Site.Common().StaticCallee() == nil
}

// includes checks whether the value (needle) is a function's parameters (hay)
func includes(needle ssa.Value, hay []*ssa.Parameter) bool {
	n, ok := needle.(*ssa.Parameter)
	if !ok {
		return false
	}

	for _, val := range hay {
		if n == val {
			return true
		}
	}
	return false
}

// tokenPtr helps convert a function return value into a pointer
func tokenPtr(in token.Pos) *token.Pos {
	return &in
}