package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func main() {
	flag.Parse()
	_, err := Analyze(flag.Args())
	if err != nil {
		log.Fatal(err)
	}
}

func Analyze(pkgs []string) (map[*ssa.Function]bool, error) {
	conf := loader.Config{Build: &build.Default}

	// Use the initial packages from the command line.
	_, err := conf.FromArgs(pkgs, false)
	if err != nil {
		return nil, err
	}

	// Load, parse and type-check the whole program.
	iprog, err := conf.Load()
	if err != nil {
		return nil, err
	}

	// Create and build SSA-form program representation.
	prog := ssautil.CreateProgram(iprog, 0)
	prog.Build()

	// cha to find dynamic calls
	cg := cha.CallGraph(prog)
	cg.DeleteSyntheticNodes()

	results, err := MarkImpure(cg)
	filtered := make(map[*ssa.Function]bool)

	// filter output
	for k, v := range results {
		if k == nil {
			continue
		}
		_, ok := conf.ImportPkgs[strings.TrimPrefix(k.Package().String(), "package ")]
		if ok && !v {
			filtered[k] = v
			fmt.Printf("%s\n", k.Name())
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
		results[fn] = modifiesState(fn)
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

// modifiesState does a simple analysis of the ssa representation of a function to detect
// whether a global variable is modified as part of the function
func modifiesState(f *ssa.Function) bool {
	if f == nil {
		// nil functions probably dont modify state?
		return false
	}

	// check each instruction
	for _, b := range f.Blocks {
		for _, i := range b.Instrs {
			if _, ok := i.(*ssa.Store); ok {
				store := i.(*ssa.Store)
				// are we writing to a global?
				if _, ok := store.Addr.(*ssa.Global); ok {
					return true
				}
			}
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
