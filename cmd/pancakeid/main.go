package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"html/template"
	"log"

	cautiouspancake "github.com/tam7t/cautious-pancake"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
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

	a, err := cautiouspancake.Analyze(iprog)
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range a {
		if !v {
			fmt.Println(PrintFuzz(k))
		}
	}
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
