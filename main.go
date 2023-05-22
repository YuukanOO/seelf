package main

import (
	"os"

	"github.com/YuukanOO/seelf/cmd"
)

func main() {
	cmd := cmd.Root()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
