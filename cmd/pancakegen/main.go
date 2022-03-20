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
	pkgPtr := flag.String("pkg", "", "package path")
	funcPtr := flag.String("func", "", "function")
	flag.Parse()

	conf := loader.Config{Build: &build.Default}
	conf.Import(*pkgPtr)

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

	f, pure := cg.Lookup(*funcPtr)
	if f == nil {
		log.Fatalf("not found: %q", *funcPtr)
	}
	if !pure {
		log.Fatalf("%q is not pure", *funcPtr)
	}
	fmt.Println(cautiouspancake.GenerateFuzz(f))
}
