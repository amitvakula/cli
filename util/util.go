package util

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime/debug"

	"github.com/fatih/color"
)

// Automatically print to stderr
func Println(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
}

// Print json blob to stderr
func PrintFormat(x interface{}) {
	y, err := json.MarshalIndent(x, "", "\t")
	if err != nil {
		panic(err)
	}
	Println(string(y))
}

// Unchecked errors
func Check(err error) {
	if err != nil {
		Println(err)
		Fatal(1)
	}
}

// JSON encoding that must succeed
func FormatBytes(x interface{}) []byte {
	y, err := json.MarshalIndent(x, "", "\t")
	if err != nil {
		panic(err)
	}
	return append(y, '\n')
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
		Println()
		Println(VersionString)
	}
}

func Fatal(exitCode int) {
	Println()
	Println(color.HiBlackString(VersionString))
	Println()
	os.Exit(exitCode)
}

// Exit with message
func FatalWithMessage(a ...interface{}) {
	Println(a...)
	Fatal(1)
}

// Populated by command.BuildCommand
var VersionString = ""
