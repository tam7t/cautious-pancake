package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"

	cautiouspancake "github.com/tam7t/cautious-pancake"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
)

func main() {
	flag.Parse()

	// environment configuration
	all := os.Getenv("ALL")       // show info on all functions, not just exported ones
	impure := os.Getenv("IMPURE") // show info on impsure functions too

	conf := loader.Config{Build: &build.Default}

	// Use the initial packages from the command line.
	_, err := conf.FromArgs(flag.Args(), false)
	if err != nil {
		log.Fatal(err)
	}

	// Load, parse and type-check the whole program.
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
			if all != "" || isExported(k) {
				if v.Pure() || impure != "" {
					fmt.Printf("%s\n\t%s\n", k, v)
				}
			}
		}
	}
}

func isExported(f *ssa.Function) bool {
	return f.Object() != nil && f.Object().Exported()
}
