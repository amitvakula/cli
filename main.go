
package main

import (
	"fmt"
	"flywheel.io/sdk/api"
	_ "github.com/gillesdemey/go-dicom"
)

var Version = "0.2.0"

func main() {
  fmt.Println("Hello World2")
  c := api.NewApiKeyClient("dev.flywheel.io:rWwbTp1kG8GdaL6ESv")
  user, resp, err := c.GetCurrentUser()
  fmt.Println(user, resp, err)
}