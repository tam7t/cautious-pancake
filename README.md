# cautious-pancake
github generated the repo name for me

## fuzzing
This is a work in progress project to automate fuzzing of golang packages.

[go-fuzz](https://github.com/dvyukov/go-fuzz) is a great tool for finding bugs in golang programs, but it
requires that you identify and instrument the functions to fuzz. Thats where cautious-pancake comes in.

cautious-pancake identifies [pure function](https://en.wikipedia.org/wiki/Pure_function) in golang packages.
These are functions can be fuzzed easily since they only operate on their direct inputs and not global
state.


## example
```
$ go build .
$ ./cautious-pancake github.com/tam7t/cautious-pancake/fixtures
-- (YesMaybePanic)
package main

import (
	"fmt"

	"github.com/google/gofuzz"
	"github.com/tam7t/cautious-pancake/fixtures"
)

func main() {
	f := fuzz.New()
	for {
		var p0 byte
		f.Fuzz(&p0)

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("found panic", r)
				fmt.Printf("p0: %v\n", p0)
			}
		}()
		fixtures.YesMaybePanic(p0)
	}
}
--
-- (YesManipulate)
...
```
If you run the generated code for `YesMaybePanic` you will quickly get the following output:
```
found panic bad input
p0: 10
```
indicating that `fixtures.YesMaybePanic(0xA)` will result in a panic.
