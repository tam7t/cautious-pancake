package main

import (
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"text/tabwriter"

	cautiouspancake "github.com/tam7t/cautious-pancake"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
)

func main() {
	pkgPtr := flag.String("pkg", "", "path to analyze")
	purePtr := flag.Bool("pure", true, "display pure functions")
	privatePtr := flag.Bool("private", false, "display private functions")
	noArgs := flag.Bool("noargs", true, "display functions with zero arguments")
	debugPtr := flag.Bool("debug", false, "display info from all packages")
	tracePtr := flag.Bool("trace", false, "print call graphs for why something is impure")

	flag.Parse()

	// Load, parse and type-check the whole program.
	conf := loader.Config{Build: &build.Default}
	conf.Import(*pkgPtr)
	iprog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}

	cg := cautiouspancake.NewCallGraph(iprog)
	err = cg.Analyze()
	if err != nil {
		log.Fatal(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.DiscardEmptyColumns|tabwriter.TabIndent)

	for k, v := range cg.Mapping {
		if k == nil {
			continue
		}

		if _, ok := cg.Prog.Imported[k.Package().Pkg.Path()]; !ok && !*debugPtr {
			continue
		}

		if len(k.Params) == 0 && !*noArgs {
			continue
		}

		if !*privatePtr && !isExported(k) {
			continue
		}

		if *purePtr && !v.Pure() {
			continue
		}

		if !*purePtr && v.Pure() {
			continue
		}

		why, where := v.Reason()
		fmt.Fprintf(w, "%s\t%s\t%s\n", k.RelString(k.Package().Pkg), why, where)
		if v.RuleCallee != nil && *tracePtr {
			printTrace(w, cg, v.RuleCallee)
			w.Flush()
		}
	}
	w.Flush()
}

func isExported(f *ssa.Function) bool {
	return f.Object() != nil && f.Object().Exported()
}

func printTrace(w io.Writer, cg *cautiouspancake.CallGraph, k *ssa.Function) {
	v := cg.Mapping[k]
	why, where := v.Reason()
	fmt.Fprintf(w, "*\t*\t%s\t%s\t%s\n", k.RelString(k.Package().Pkg), why, where)

	if v.RuleCallee == nil {
		return
	}

	printTrace(w, cg, v.RuleCallee)
}
