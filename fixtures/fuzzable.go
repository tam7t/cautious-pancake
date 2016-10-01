package cases

import (
	"fmt"
	"net/http"
	"strings"
)

// if a function has no side effects, (only operates on operands & return vaules)
// then it is a good candidate for fuzzing

// Functions are prefixed with Yes or No to indicate if it is 'trivally fuzzable'

var startString = "start"
var b func(a string) string = func(a string) string { return a }

func NoPrint(a string) {
	fmt.Println(a)
}

func NoNet(a string) error {
	_, err := http.Get("http://example.com/")
	return err
}

func NoGlobal(a string) {
	startString = a
}

// No because a different function might have modified the
// value of b between invocations of NoDynamicCall
func NoDynamicCall(a string) string {
	return b(a)
}

func YesAnonymousDynamicCall() string {
	a := "hi"
	b := func() string { return a }
	c := b()
	return c
}

// not sure if this is actually a good one
func YesGlobal() string {
	return startString
}

func YesFuncParam() string {
	return yes(YesGlobal)
}

func yes(a func() string) string {
	return a()
}

func Yes() string {
	return "hello"
}

func YesManipulate(a string) string {
	return strings.ToLower(a)
}

func YesAppend(a string) string {
	return a + "!"
}

func YesPanicArray() bool {
	a := []bool{true}
	return a[1]
}

func YesPanicNil() string {
	var a *MyStuff
	return a.MyValue
}

func YesPanic() {
	panic("bad")
}

type MyStuff struct {
	MyValue string
}

// does not work yet
func (a *MyStuff) YesParse(b string) {
	a.MyValue = b
}

func (a *MyStuff) NoWrite() {
	startString = a.MyValue
}

func YesParser(a string) *MyStuff {
	x := &MyStuff{}
	x.YesParse(a)
	return x
}

func NoWriter() {
	x := &MyStuff{
		MyValue: "myval",
	}
	x.NoWrite()

}

// TODO: do i care if it mutates the input?
