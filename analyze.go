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
	Prog       *loader.Program
	CallGraph  *callgraph.Graph
	Mapping    map[*ssa.Function]*Node
	IgnoreCall map[string][]string
	IgnoreRead map[string][]string
}

// Node contains details on why a function is marked as not pure
type Node struct {
	CallGraph     *CallGraph
	RuleBasic     *token.Pos
	RuleInterface *token.Pos
	RuleCallee    *ssa.Function
}

var (
	// DefaultIgnoreCall lists functions that should be ignored when
	// determining if a function is pure. These calls may technically access
	// or modify global state, but the side effects are rarely important.
	DefaultIgnoreCall = map[string][]string{
		"log": {"Print", "Printf", "Println"},
		"fmt": {"Print", "Printf", "Println", "Errorf"},
	}

	// DefaultIgnoreRead lists global variables that should be ignored
	// when determining if a function is pure. These variables may
	// change values between function calls, but in practice are most
	// often used as constants.
	DefaultIgnoreRead = map[string][]string{
		"encoding/binary": {"BigEndian", "LittleEndian"},
	}
)

// NewCallGraph creates a CallGraph by performing ssa analysis
func NewCallGraph(iprog *loader.Program) *CallGraph {
	// Create and build SSA-form program representation
	prog := ssautil.CreateProgram(iprog, 0)
	prog.Build()

	// cha to find dynamic calls
	cgo := cha.CallGraph(prog)
	cgo.DeleteSyntheticNodes()

	return &CallGraph{
		Prog:       iprog,
		CallGraph:  cgo,
		Mapping:    make(map[*ssa.Function]*Node),
		IgnoreCall: DefaultIgnoreCall,
		IgnoreRead: DefaultIgnoreRead,
	}
}

// Pure returns whether an analyzed node is pure.
func (n *Node) Pure() bool {
	return (n.RuleBasic == nil && n.RuleInterface == nil && n.RuleCallee == nil)
}

// Reason returns descriptive strings for why a node is not pure.
func (n *Node) Reason() (string, string) {
	if n.RuleBasic != nil {
		return "basic", fmt.Sprintf("%s", n.CallGraph.Prog.Fset.Position(*n.RuleBasic))
	}
	if n.RuleInterface != nil {
		return "iface", fmt.Sprintf("%s", n.CallGraph.Prog.Fset.Position(*n.RuleInterface))
	}
	if n.RuleCallee != nil {
		return "calls", fmt.Sprintf("%s", n.RuleCallee)
	}
	return "", ""

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
// state or call functions that modify global state.
func (cg *CallGraph) Analyze() error {
	// basic analysis
	for fn := range cg.CallGraph.Nodes {
		n := &Node{
			CallGraph: cg,
		}
		if pos := accessGlobal(fn, cg.IgnoreRead); pos != nil {
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

		if isWhitelisted(callee, cg.IgnoreCall) {
			return nil
		}

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

// isWhitelisted returns whether the ssa.Member (function or variable
// reference) is in the whitelist. The whitlelist is a map of package paths to
// a slice of names. Additionally, all error types are considered whitelisted
// since they are commonly used as return values from parsers and I would
// rather not enumerate all errors in the whitelist.
func isWhitelisted(n ssa.Member, whitelist map[string][]string) bool {
	if v, ok := n.Type().(*types.Pointer); ok {
		if v.String() == "*error" {
			return true
		}
	}

	pkg := n.Package().Pkg.Path()
	f := n.Name()

	if list, ok := whitelist[pkg]; ok {
		for _, v := range list {
			if f == v {
				return true
			}
		}
	}

	return false
}

// accessGlobal does a simple analysis of the ssa representation of a function
// to detect whether a global variable is read or modified
func accessGlobal(f *ssa.Function, whitelist map[string][]string) *token.Pos {
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
				if j, ok := v.X.(*ssa.Global); ok {
					if !isWhitelisted(j, whitelist) {
						return tokenPtr(v.Pos())
					}
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

// dynamic checks whether the callgraph edge is a dynamic call
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
