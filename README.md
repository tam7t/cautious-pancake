# cautious-pancake

[![GoDoc](https://godoc.org/github.com/tam7t/cautious-pancake?status.svg)](https://pkg.go.dev/github.com/tam7t/cautious-pancake) [![Go Report Card](https://goreportcard.com/badge/github.com/tam7t/cautious-pancake)](https://goreportcard.com/report/github.com/tam7t/cautious-pancake)

github generated the repo name for me

## fuzzing

`cautious-pancake` aims to make fuzzing golang packages easier by identifying
[pure functions](https://en.wikipedia.org/wiki/Pure_function). These functions
can be easily fuzzed since they only operate on their direct inputs and do not
modify global state.

## example

### `pancakeinfo`

Given a package, `pancakeinfo` will tell you which functions are pure:

```shell
$ go get -u github.com/tam7t/cautious-pancake/cmd/pancakeinfo
$ pancakeinfo -pkg=github.com/mdlayher/arp
(Operation).String
NewPacket
(Client).HardwareAddr
(*Packet).UnmarshalBinary
(*Packet).MarshalBinary
```

The `-pure=false` flag will return all functions deemed impure, including
the reason for the determination and the `-private` flag will display
information on private functions as well.

### `pancakegen`

Given a package and a function, `pancakegen` will generate code to fuzz that
function:

```text
$ go get -u github.com/tam7t/cautious-pancake/cmd/pancakegen
$ pancakegen -pkg=github.com/tam7t/cautious-pancake/fixtures -func=YesMaybePanic
package fixtures

import (
	"testing"
)

func FuzzYesMaybePanic(f *testing.F) {
	f.Fuzz(func(t *testing.T, p0 byte) { 
		YesMaybePanic(p0)
	})
}
```

If you run the generated code you will quickly get:

```shell
$ go test ./fixtures/... --fuzz=Fuzz
fuzz: elapsed: 0s, gathering baseline coverage: 0/1 completed
fuzz: elapsed: 0s, gathering baseline coverage: 1/1 completed, now fuzzing with 16 workers
fuzz: elapsed: 0s, execs: 20 (701/sec), new interesting: 0 (total: 1)
--- FAIL: FuzzYesMaybePanic (0.03s)
    --- FAIL: FuzzYesMaybePanic (0.00s)
        testing.go:1349: panic: bad input
            goroutine 35 [running]:
            runtime/debug.Stack()
                /usr/local/go/src/runtime/debug/stack.go:24 +0x90
            testing.tRunner.func1()
                /usr/local/go/src/testing/testing.go:1349 +0x1f2
            panic({0x65dd40, 0x6ee3e0})
                /usr/local/go/src/runtime/panic.go:838 +0x207
            github.com/tam7t/cautious-pancake/fixtures.YesMaybePanic(...)
                /home/tam7t/code/github.com/tam7t/cautious-pancake/fixtures/fuzzable.go:97
            github.com/tam7t/cautious-pancake/fixtures.FuzzYesMaybePanic.func1(0x0?, 0xa)
                /home/tam7t/code/github.com/tam7t/cautious-pancake/fixtures/fuzzable_test.go:9 +0x77
            reflect.Value.call({0x660400?, 0x6b5f98?, 0x13?}, {0x69f589, 0x4}, {0xc00009bc80, 0x2, 0x2?})
                /usr/local/go/src/reflect/value.go:556 +0x845
            reflect.Value.Call({0x660400?, 0x6b5f98?, 0x514?}, {0xc00009bc80, 0x2, 0x2})
                /usr/local/go/src/reflect/value.go:339 +0xbf
            testing.(*F).Fuzz.func1.1(0x0?)
                /usr/local/go/src/testing/fuzz.go:337 +0x231
            testing.tRunner(0xc0001a16c0, 0xc00018aea0)
                /usr/local/go/src/testing/testing.go:1439 +0x102
            created by testing.(*F).Fuzz.func1
                /usr/local/go/src/testing/fuzz.go:324 +0x5b8
            
    
    Failing input written to testdata/fuzz/FuzzYesMaybePanic/358fa4d16da00de4d29482b2ea74da673eb27bfc5614d2b123ac0aa89e0e1ea5
    To re-run:
    go test -run=FuzzYesMaybePanic/358fa4d16da00de4d29482b2ea74da673eb27bfc5614d2b123ac0aa89e0e1ea5
FAIL
exit status 1
FAIL    github.com/tam7t/cautious-pancake/fixtures      0.032s
```

indicating that `fixtures.YesMaybePanic(0xA)` will result in a panic.
