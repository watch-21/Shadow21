package main

import (
	"fmt"
	"os"

	"gitlab.com/parrotsec/packages/anonsurf/internal/safekill"
)

func main() {
	if err := (safekill.Runner{}).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "safekill:", err)
		os.Exit(1)
	}
}
