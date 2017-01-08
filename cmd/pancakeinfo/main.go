package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"

	cautiouspancake "github.com/tam7t/cautious-pancake"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
)

func main() {
	pkgPtr := flag.String("pkg", "", "path to analyze")
	filterPtr := flag.String("filter", "pure", "show pure | impure functions")
	allPtr := flag.Bool("all", false, "include private functions")
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

	for k, v := range cg.Mapping {
		if k == nil {
			continue
		}
		if _, ok := cg.Prog.Imported[k.Package().Pkg.Path()]; ok {
			if *allPtr || isExported(k) {
				if (*filterPtr == "pure" && v.Pure()) || (*filterPtr == "impure" && !v.Pure()) {
					fmt.Println(k.RelString(k.Package().Pkg), v)
				}
			}
		}
	}
}

func isExported(f *ssa.Function) bool {
	return f.Object() != nil && f.Object().Exported()
}
