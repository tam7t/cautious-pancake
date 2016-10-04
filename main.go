package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"go/types"
	"html/template"
	"log"

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
		_, ok := conf.ImportPkgs[k.Package().Pkg.Path()]
		if ok && !v {
			filtered[k] = v

			fmt.Printf("-- (%s)\n", k.Name())
			fmt.Println(PrintFuzz(k))
			fmt.Println("--")
		}
	}

	return filtered, nil
}

const fuzzTemp = `package main

import (
	"fmt"

	"github.com/google/gofuzz"
	"{{.Package.Pkg.Path}}"
)

func main() { {{ $length := len .Params }}{{if gt $length 0}}
	f := fuzz.New(){{end}}
	{{range $i, $v := .Params}}var p{{$i}} {{$v.Type.String}}{{end}}
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("found panic", r){{range $i, $v := .Params}}
			fmt.Printf("p{{$i}}: %v\n", p{{$i}}){{end}}
		}
	}()
	for { {{range $i, $v := .Params}}
		f.Fuzz(&p{{$i}}){{end}}
		{{.Package.Pkg.Name}}.{{.Name}}({{range $i, $v := .Params}}p{{$i}}{{if $i}}, {{end}}{{end}})
	}
}`

func PrintFuzz(f *ssa.Function) string {
	var out bytes.Buffer
	tmpl := template.Must(template.New("").Parse(fuzzTemp))
	tmpl.Execute(&out, f)
	return out.String()
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
