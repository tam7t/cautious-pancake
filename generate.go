package cautiouspancake

import (
	"bytes"
	"go/types"
	"html/template"

	"golang.org/x/tools/go/ssa"
)

// https://pkg.go.dev/testing#F.Fuzz
// TODO: print error on unsupported argument types and other situations
// unsupported by f.Fuzz()
const fuzzTemp = `package {{.Package.Pkg.Name}}

import (
	"testing"
)

func Fuzz{{.Name}}(f *testing.F) {
	f.Fuzz(func(t *testing.T, {{range $i, $v := .Params}}{{if $i}}, {{end}}p{{$i}} {{$v.Type | strippkg}}{{end}}) { {{if .Signature.Recv}}
		p0.{{.Name}}({{range $i, $v := .Params}}{{if $i}}{{if gt $i 1}}, {{end}}p{{$i}}{{end}}{{end}}{{if .Signature.Variadic}}...{{end}}){{else}}
		{{.Name}}({{range $i, $v := .Params}}{{if $i}}, {{end}}p{{$i}}{{end}}{{if .Signature.Variadic}}...{{end}}){{end}}
	})
}
`

func GenerateFuzz(f *ssa.Function) string {
	var out bytes.Buffer
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"strippkg": func(a interface{}) string {
			return types.TypeString(a.(types.Type), func(p *types.Package) string {
				return p.Name()
			})
		},
	}).Parse(fuzzTemp))
	tmpl.Execute(&out, f)
	return out.String()
}
