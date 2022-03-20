package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"go/types"
	"html/template"
	"log"

	cautiouspancake "github.com/tam7t/cautious-pancake"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
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

	for _, v := range cg.Pure() {
		if v.RelString(v.Package().Pkg) == *funcPtr {
			fmt.Println(PrintFuzz(v))
		}
	}
}

const fuzzTemp = `package {{.Package.Pkg.Name}}

import (
	"testing"
)

func Fuzz{{.Name}}(f *testing.F) {
	f.Fuzz(func(t *testing.T, {{range $i, $v := .Params}}p{{$i}} {{$v.Type | strippkg}}{{end}}) { {{if .Signature.Recv}}
		p0.{{.Name}}({{range $i, $v := .Params}}{{if $i}}{{if gt $i 1}}, {{end}}p{{$i}}{{end}}{{end}}{{if .Signature.Variadic}}...{{end}}){{else}}
		{{.Name}}({{range $i, $v := .Params}}{{if $i}}, {{end}}p{{$i}}{{end}}{{if .Signature.Variadic}}...{{end}}){{end}}
	})
}
`

func PrintFuzz(f *ssa.Function) string {
	var out bytes.Buffer
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"strippkg": func(a interface{}) string {
			return types.TypeString(a.(types.Type), func(p *types.Package) string {
				return p.Name()
			})
		},
		"fuzznil": func(a interface{}) bool {
			if f.Signature.Recv() != nil {
				_, ok := f.Params[0].Type().(*types.Pointer)
				if ok {
					return true
				}
			}
			return false
		},
	}).Parse(fuzzTemp))
	tmpl.Execute(&out, f)
	return out.String()
}
