package cautiouspancake

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateFuzz(t *testing.T) {
	cg := NewCallGraph(loadFixture(t))
	if err := cg.Analyze(); err != nil {
		t.Fatalf("Analyze() returned error: %v", err)
	}

	tests := []struct {
		in   string
		want string
	}{
		{
			in: "YesMaybePanic",
			want: `package fixtures

import (
	"testing"
)

func FuzzYesMaybePanic(f *testing.F) {
	f.Fuzz(func(t *testing.T, p0 byte) { 
		YesMaybePanic(p0)
	})
}
`,
		},
		{
			in: "YesArgs",
			want: `package fixtures

import (
	"testing"
)

func FuzzYesArgs(f *testing.F) {
	f.Fuzz(func(t *testing.T, p0 string, p1 []byte, p2 int, p3 bool, p4 float64) { 
		YesArgs(p0, p1, p2, p3, p4)
	})
}
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			f, _ := cg.Lookup(tc.in)
			if f == nil {
				t.Errorf("could not find function %q", tc.in)
				return
			}

			got := GenerateFuzz(f)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Log(got)
				t.Errorf("GenerateFuzz() returned diff (-want +got):\n%s", diff)
			}
		})
	}
}
