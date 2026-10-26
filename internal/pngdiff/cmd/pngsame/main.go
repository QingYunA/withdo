// Command pngsame compares two PNGs with pngdiff's tolerance.
//
//	pngsame a.png b.png
//
// It exits 0 when they are the same, 1 when they differ and 2 on an error.
package main

import (
	"fmt"
	"os"

	"github.com/QingYunA/adhd-todo/internal/pngdiff"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: pngsame a.png b.png")
		os.Exit(2)
	}
	same, err := pngdiff.SameFiles(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "pngsame:", err)
		os.Exit(2)
	}
	if !same {
		os.Exit(1)
	}
}
