package cautiouspancake

import (
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
	Mapping   map[*ssa.Function]bool
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
		Mapping:   make(map[*ssa.Function]bool),
	}
}

// Analyze examines the callgraph looking for functions that modify global
// state or call functions that modify global state
func (cg *CallGraph) Analyze() error {
	// basic analysis
	for fn, _ := range cg.CallGraph.Nodes {
		cg.Mapping[fn] = accessGlobal(fn) || usesInterface(fn)
	}

	// sub function analysis, if a func calls an func that is not pure then it
	// needs to be marked as such
	if err := callgraph.GraphVisitEdges(cg.CallGraph, func(edge *callgraph.Edge) error {
		// analyze callee (ssa.Function)
		callee := edge.Callee.Func
		caller := edge.Caller.Func

		// callee modifies state, mark static callers bad as well
		if cg.Mapping[callee] {
			if dynamic(edge) {
				if !includes(edge.Site.Common().Value, caller.Params) {
					cg.Mapping[edge.Caller.Func] = true
				}
			} else {
				cg.Mapping[edge.Caller.Func] = true
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
		if ok && !v {
			filtered = append(filtered, k)
		}
	}

	return filtered
}

// accessGlobal does a simple analysis of the ssa representation of a function
// to detect whether a global variable is read or modified
func accessGlobal(f *ssa.Function) bool {
	if f == nil {
		// nil functions probably do not modify state?
		return false
	}

	for _, b := range f.Blocks {
		for _, i := range b.Instrs {
			switch v := i.(type) {
			case *ssa.Store:
				if _, ok := v.Addr.(*ssa.Global); ok {
					return true
				}
			case *ssa.UnOp:
				if _, ok := v.X.(*ssa.Global); ok {
					return true
				}
			}
		}
	}
	return false
}

// usesInterface eliminates functions that accept an interface as a parameter
// since it is difficult to resolve whether the interface's implementation will
// modify global state
func usesInterface(f *ssa.Function) bool {
	if f == nil {
		return false
	}
	for _, v := range f.Params {
		// TODO: check structs for iface parameters
		if types.IsInterface(v.Type()) {
			return true
		}
	}
	return false
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
