package cautiouspancake

import (
	"go/build"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/tools/go/loader"
)

func loadFixture(t *testing.T) *loader.Program {
	t.Helper()

	pkgs := []string{"github.com/tam7t/cautious-pancake/fixtures"}

	conf := loader.Config{Build: &build.Default}

	// Use the initial packages from the command line.
	_, err := conf.FromArgs(pkgs, false)
	if err != nil {
		t.Fatal(err)
	}

	// Load, parse and type-check the whole program.
	iprog, err := conf.Load()
	if err != nil {
		t.Fatal(err)
	}

	return iprog
}

func TestAnalyze(t *testing.T) {
	want := map[string]bool{
		"YesManipulate":             true,
		"YesPanicArray":             true,
		"YesPanicNil":               true,
		"YesAnonymousDynamicCall":   true,
		"YesFuncParam":              true,
		"YesAnonymousDynamicCall$1": true,
		"YesParser":                 true,
		"YesParse":                  true,
		"YesMultArgs":               true,
		"YesMaybePanic":             true,
		"Yes":                       true,
		"init$1":                    true,
		"YesAppend":                 true,
		"yes":                       true,
		"YesPanic":                  true,
		"YesRead":                   true,
		"YesLog":                    true,
		"YesErr":                    true,
		"YesFmtErr":                 true,
		"YesVariadic":               true,
		"YesArgs":                   true,
		"NoDynamicCall":             false,
		"NoGlobalRead":              false,
		"NoGlobalWrite":             false,
		"NoInterface":               false,
		"NoNet":                     false,
		"NoPrint":                   false,
		"NoWrite":                   false,
		"NoWriteErr":                false,
		"NoWriter":                  false,
		"init":                      false,
	}

	iprog := loadFixture(t)

	cg := NewCallGraph(iprog)
	if err := cg.Analyze(); err != nil {
		t.Errorf("Analyze() returned error: %v", err)
	}

	got := make(map[string]bool)
	for _, v := range cg.Pure() {
		got[v.Name()] = true
	}
	for _, v := range cg.Impure() {
		got[v.Name()] = false
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Analyze() mismatch (-want +got):\n%s", diff)
	}
}
