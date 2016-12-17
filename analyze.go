package cautiouspancake

import (
	"go/types"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func Analyze(iprog *loader.Program) (map[*ssa.Function]bool, error) {
	// Create and build SSA-form program representation.
	prog := ssautil.CreateProgram(iprog, 0)
	prog.Build()

	// cha to find dynamic calls
	cg := cha.CallGraph(prog)
	cg.DeleteSyntheticNodes()

	// mark functions in the callgraph that are not 'pure'
	results, err := MarkImpure(cg)
	if err != nil {
		return nil, err
	}

	// filter output to only include functions imported by the program
	filtered := make(map[*ssa.Function]bool)
	for k, v := range results {
		if k == nil {
			continue
		}
		_, ok := iprog.Imported[k.Package().Pkg.Path()]
		if ok && !v {
			filtered[k] = v
		}
	}

	return filtered, nil
}

// MarkImpure takes a callgraph and analyses the nodes looking for functions that modify
// global state or functions that call functions that modify global state
func MarkImpure(cg *callgraph.Graph) (map[*ssa.Function]bool, error) {
	results := make(map[*ssa.Function]bool)

	// basic analysis
	for fn, _ := range cg.Nodes {
		results[fn] = accessGlobal(fn) || usesInterface(fn)
	}

	// sub function analysis, if a func calls an func that is not pure then it
	// needs to be marked as such
	if err := callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		// analyze callee (ssa.Function)
		callee := edge.Callee.Func
		caller := edge.Caller.Func

		// callee modifies state, mark static callers bad as well
		if results[callee] {
			if dynamic(edge) {
				if !includes(edge.Site.Common().Value, caller.Params) {
					results[edge.Caller.Func] = true
				}
			} else {
				results[edge.Caller.Func] = true
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return results, nil
}

// accessGlobal does a simple analysis of the ssa representation of a function to detect
// whether a global variable is read or modified as part of the function
func accessGlobal(f *ssa.Function) bool {
	if f == nil {
		// nil functions probably dont modify state?
		return false
	}

	// check each instruction
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

// eliminate functions that have interface inputs since we dont know how they will actually be implemented (and may depend on global state.
// todo: check structs for iface parameters
func usesInterface(f *ssa.Function) bool {
	if f == nil {
		return false
	}
	for _, v := range f.Params {
		if types.IsInterface(v.Type()) {
			return true
		}
	}
	return false
}

// dynamic is a helper that checks whether the callgraph edge is a dynamic funciton call
func dynamic(edge *callgraph.Edge) bool {
	return edge.Site != nil && edge.Site.Common().StaticCallee() == nil
}

// includes checks whether the value (needle) is one of the parameters (hay) of a function
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
