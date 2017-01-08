# cautious-pancake [![Build Status](https://travis-ci.org/tam7t/cautious-pancake.svg?branch=master)](https://travis-ci.org/tam7t/cautious-pancake) [![GoDoc](https://godoc.org/github.com/tam7t/cautious-pancake?status.svg)](https://godoc.org/github.com/tam7t/cautious-pancake) [![Go Report Card](https://goreportcard.com/badge/github.com/tam7t/cautious-pancake)](https://goreportcard.com/report/github.com/tam7t/cautious-pancake)
github generated the repo name for me

## fuzzing
`cautious-pancake` aims to make fuzzing golang packages easier by identifying
[pure functions](https://en.wikipedia.org/wiki/Pure_function). These functions
can be easily fuzzed since they only operate on their direct inputs and do not
modify global state.

## example

### `pancakeinfo`
Given a package, `pancakeinfo` will tell you which functions are pure:

```
$ go install github.com/tam7t/cautious-pancake/cmd/pancakeinfo
$ pancakeinfo -pkg=github.com/mdlayher/arp
NewPacket Pure
(Client).HardwareAddr Pure
(*Packet).MarshalBinary Pure
(*Packet).UnmarshalBinary Pure
```

The `-filter=impure` flag will return all functions deemed impure, including
the reason for the determination and the `-all` flag will display information
on private functions as well.

### `pancakegen`
Given a package and a function, `pancakegen` will generate code to fuzz that
function:

```
$ go install github.com/tam7t/cautious-pancake/cmd/pancakegen
$ pancakegen -pkg=github.com/tam7t/cautious-pancake/fixtures -func=YesMaybePanic
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
```

If you run the generated code you will quickly get:
```
found panic bad input
p0: 10
```
indicating that `fixtures.YesMaybePanic(0xA)` will result in a panic.
