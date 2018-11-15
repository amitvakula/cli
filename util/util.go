package util

import (
	"encoding/json"
	. "fmt"
	"os"
	"regexp"
	"runtime/debug"
)

// :/
func Check(err error) {
	if err != nil {
		Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func CheckM(err error, msg string) {
	if err != nil {
		Fprintln(os.Stderr, "")
		Fprintln(os.Stderr, err)
		Fprintln(os.Stderr, "")
		Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
}

func Format(x interface{}) string {
	y, err := json.MarshalIndent(x, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(y)
}

func PrintFormat(x interface{}) {
	y, err := json.MarshalIndent(x, "", "\t")
	if err != nil {
		panic(err)
	}
	Fprintln(os.Stderr, string(y))
}

// Aims to match the $GOPATH/src this binary was compiled with, during stack trace output.
// Newline, tab, bunch of non-whitespace, "/src/", useful path, colon, line number.
var matchGoSrc = regexp.MustCompile(`(\n\t)/\S+/src/(\S+:[0-9]+)`)

// GracefulRecover will recover any panic, printing a stack trace then anything passed afterwards. Useful for reporting crashes to end-users.
func GracefulRecover(postamble ...interface{}) {
	if r := recover(); r != nil {
		stack := string(debug.Stack())
		stack = matchGoSrc.ReplaceAllString(stack, "$1$2")

		Println()
		Println(stack)
		Println("Crash report:", r)
		Println(postamble...)
	}
}
