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

// Compare two files by looking at the following fields from os.Stat:
// File Mode
// Modified Time
// Size
func ShallowFileCmp(path1, path2 string) (bool, error) {
	stat1, err := os.Stat(path1)
	if err != nil {
		Printf("Unable to read \"%s\": %v\n", path1, err)
		return false, err
	}
	stat2, err := os.Stat(path2)
	if err != nil {
		Printf("Unable to read \"%s\": %v\n", path2, err)
		return false, err
	}
	return (stat1.Mode() == stat2.Mode() &&
		stat1.ModTime() == stat2.ModTime() &&
		stat1.Size() == stat2.Size()), nil
}
