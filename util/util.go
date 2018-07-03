package util

import (
	"encoding/json"
	. "fmt"
	"os"
	"path/filepath"
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

// Remove the contents of directory
func RemoveTree(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return os.Remove(dir)
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
