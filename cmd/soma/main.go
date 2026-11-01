package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/pr0sp3ro/soma/internal/soma"
)

func main() {
	err := soma.Run(os.Args[1:])
	if err == nil || errors.Is(err, flag.ErrHelp) {
		return
	}

	if err == context.Canceled {
		os.Exit(130)
	}

	fmt.Fprintln(os.Stderr, "soma:", err)
	os.Exit(1)
}
