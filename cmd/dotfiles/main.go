package main

import (
	"fmt"
	"os"

	"github.com/dsaleh/dotfiles/internal/app"
)

var version = "dev"

func main() {
	if err := app.Run(os.Args[1:], version); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
