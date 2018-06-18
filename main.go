package main

import (
	"os"

	"github.com/dollarshaveclub/psst/cmd"
)

func main() {
	cmd.Execute()
	os.Exit(0)
}
