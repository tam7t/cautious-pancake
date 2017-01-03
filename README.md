# cautious-pancake [![Build Status](https://travis-ci.org/tam7t/cautious-pancake.svg?branch=master)](https://travis-ci.org/tam7t/cautious-pancake) [![GoDoc](https://godoc.org/github.com/tam7t/cautious-pancake?status.svg)](https://godoc.org/github.com/tam7t/cautious-pancake) [![Go Report Card](https://goreportcard.com/badge/github.com/tam7t/cautious-pancake)](https://goreportcard.com/report/github.com/tam7t/cautious-pancake)
github generated the repo name for me

## fuzzing
This is a work in progress project to automate fuzzing of golang packages.

[go-fuzz](https://github.com/dvyukov/go-fuzz) is a great tool for finding bugs in golang programs, but it
requires that you identify and instrument the functions to fuzz. Thats where `cautious-pancake` comes in.

`cautious-pancake` identifies [pure function](https://en.wikipedia.org/wiki/Pure_function) in golang packages.
These are functions can be fuzzed easily since they only operate on their direct inputs and not global
state.


## example

### `pancakeinfo`
Given a package, `pancakeinfo` will tell you which functions are pure:

```
$ go install github.com/tam7t/cautious-pancake/cmd/pancakeinfo
$ pancakeinfo github.com/mdlayher/arp
github.com/mdlayher/arp.NewPacket
	Pure
(github.com/mdlayher/arp.Client).HardwareAddr
	Pure
(*github.com/mdlayher/arp.Packet).UnmarshalBinary
	Pure
(*github.com/mdlayher/arp.Packet).MarshalBinary
	Pure
```

You can set `IMPURE` environment variable to show information about why
functions were deemed impure and `ALL` to include info on unexported functions.

### `pancakeid`
You can provide `cautious-pancake` with a package to analyze and it will print out all of the 'pure' functions
and attempt to generate code that can be run to fuzz those functions:

```
$ go install github.com/tam7t/cautious-pancake/cmd/pancakeid
$ pancakeid github.com/tam7t/cautious-pancake/fixtures
-- (YesMaybePanic)
package main

import (
	"fmt"

	"github.com/google/gofuzz"
	"github.com/tam7t/cautious-pancake/fixtures"
)

func main() {
	f := fuzz.New()
	var p0 byte
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("found panic", r)
			fmt.Printf("p0: %v\n", p0)
		}
	}()
	for {
		f.Fuzz(&p0)
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
