package fixtures

import (
	"bytes"
	bin "encoding/binary"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// if a function has no side effects, (only operates on operands & return vaules)
// then it is a good candidate for fuzzing

// Functions are prefixed with Yes or No to indicate if it is 'trivally fuzzable'

var startString = "start"
var bFunc = func(a string) string { return a }
var errFoo = errors.New("custom err")

func NoPrint(a string) {
	fmt.Fprintln(os.Stdout, a)
}

func NoNet(a string) error {
	_, err := http.Get("http://example.com/")
	return err
}

func NoGlobalWrite(a string) {
	startString = a
}

// No because a different function might have modified the
// value of b between invocations of NoDynamicCall
func NoDynamicCall(a string) string {
	return bFunc(a)
}

type Foo interface {
	Bar()
}

func NoInterface(a Foo) {
	a.Bar()
}

func YesAnonymousDynamicCall() string {
	a := "hi"
	b := func() string { return a }
	c := b()
	return c
}

func NoGlobalRead() string {
	return startString
}

func YesFuncParam() string {
	return yes(Yes)
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

func YesMaybePanic(i byte) {
	if i == 10 {
		panic("bad input")
	}
}

type MyStuff struct {
	MyValue string
}

func (a *MyStuff) YesParse(b string) {
	a.MyValue = b
}

func (a *MyStuff) YesMultArgs(b string, c string) {
	a.MyValue = b + c
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

func YesRead() uint16 {
	var out uint16
	bin.BigEndian.PutUint16([]byte{0x0, 0x01}, out)
	return out
}

func YesLog() {
	log.Println("Foobar")
}

func YesErr() error {
	// this is a global read, but since the type is an error it is ignored
	return errFoo
}

func NoWriteErr() {
	errFoo = errors.New("foo")
}

func YesFmtErr(a int) error {
	if a > 0 {
		return nil
	}
	return fmt.Errorf("bad %s", "asdf")
}

func YesVariadic(nums ...int) int {
	total := 0
	for _, num := range nums {
		total += num
	}
	return total
}

func YesArgs(a string, b []byte, c int, d bool, e float64) {
	if bytes.Equal(b, []byte{0x32, 0xFF}) {
		panic("bad input")
	}
}
