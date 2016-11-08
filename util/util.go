package util

import (
	. "fmt"
	"os"
)

// :/
func Check(err error) {
	if err != nil {
		Println(err)
		os.Exit(1)
	}
}
