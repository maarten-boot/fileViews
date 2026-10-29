// Command fileviews is a graphical hierarchical file and directory viewer.
//
// Usage: fileviews [directory]
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/maarten-boot/fileViews/internal/ui"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: fileviews [directory]")
	}
	flag.Parse()
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	ui.Run(flag.Arg(0))
}
