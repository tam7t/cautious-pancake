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
YesParse
YesManipulate
YesAppend
YesAnonymousDynamicCall
...
```
