package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/diffpal/diffpal/internal/cmd"
)

func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "diffpal: %v\n", err)
		var coder interface{ ExitCode() int }
		if errors.As(err, &coder) {
			os.Exit(coder.ExitCode())
		}
		os.Exit(1)
	}
}
