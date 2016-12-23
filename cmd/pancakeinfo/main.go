package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"

	cautiouspancake "github.com/tam7t/cautious-pancake"

	"golang.org/x/tools/go/loader"
)

func main() {
	flag.Parse()

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
			fmt.Printf("%s\n\t%s\n", k, v)
		}
	}
}
