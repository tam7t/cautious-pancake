package cautiouspancake

import (
	"fmt"
	"go/build"
	"log"
	"reflect"
	"testing"

	"golang.org/x/tools/go/loader"
)

func TestAnalyze(t *testing.T) {
	pkgs := []string{"github.com/tam7t/cautious-pancake/fixtures"}

	exp := map[string]bool{
		"YesManipulate":             false,
		"YesPanicArray":             false,
		"YesPanicNil":               false,
		"YesAnonymousDynamicCall":   false,
		"YesFuncParam":              false,
		"YesAnonymousDynamicCall$1": false,
		"YesParser":                 false,
		"YesParse":                  false,
		"YesMultArgs":               false,
		"YesMaybePanic":             false,
		"Yes":                       false,
		"init$1":                    false,
		"YesAppend":                 false,
		"yes":                       false,
		"YesPanic":                  false,
		"YesRead":                   false,
		"YesLog":                    false,
		"YesErr":                    false,
		"YesFmtErr":                 false,
		"YesVariadic":               false,
	}

	conf := loader.Config{Build: &build.Default}

	// Use the initial packages from the command line.
	_, err := conf.FromArgs(pkgs, false)
	if err != nil {
		log.Fatal(err)
	}

	// Load, parse and type-check the whole program.
	iprog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}

	cg := NewCallGraph(iprog)
	err = cg.Analyze()
	if err != nil {
		t.Error("err is not nil")
	}

	actual := make(map[string]bool)

	for _, v := range cg.Pure() {
		fmt.Println(v.Name())
		actual[v.Name()] = false
	}

	if !reflect.DeepEqual(exp, actual) {
		t.Logf("want: %s\n", exp)
		t.Logf("got: %s\n", actual)
		t.Error("wrong packages")
	}
}
