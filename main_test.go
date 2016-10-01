package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestAnalyze(t *testing.T) {
	pkg := "github.com/tam7t/cautious-pancake/fixtures"

	exp := map[string]bool{
		"YesManipulate":             false,
		"YesPanicArray":             false,
		"YesPanicNil":               false,
		"YesAnonymousDynamicCall":   false,
		"YesFuncParam":              false,
		"YesAnonymousDynamicCall$1": false,
		"YesParser":                 false,
		"YesParse":                  false,
		"Yes":                       false,
		"init$1":                    false,
		"YesAppend":                 false,
		"YesGlobal":                 false,
		"yes":                       false,
		"YesPanic":                  false,
	}

	answer, err := Analyze([]string{pkg})

	if err != nil {
		t.Error("err is not nil")
	}

	actual := make(map[string]bool)

	for k, v := range answer {
		if !v {
			fmt.Println(k.Name())
			actual[k.Name()] = v
		}
	}

	if !reflect.DeepEqual(exp, actual) {
		t.Logf("want: %s\n", exp)
		t.Logf("got: %s\n", actual)
		t.Error("wrong packages")
	}
}
