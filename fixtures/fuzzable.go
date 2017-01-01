package fixtures

import (
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
var b func(a string) string = func(a string) string { return a }
var e = errors.New("custom err")

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
	return b(a)
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

func YesRead() uint16 {
	var out uint16
	bin.BigEndian.PutUint16([]byte{0x0, 0x01}, out)
	return out
}

func YesLog() {
	log.Println("Foobar")
}

func YesErr() error {
	return e
}

func NoWriteErr() {
	e = errors.New("foo")
}

// TODO: do i care if it mutates the input?
