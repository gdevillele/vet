package main

import (
	"os"

	"github.com/gdevillele/vet/implementations/go/internal/cli"
)

func main() {
	os.Exit(cli.Run(cli.Invocation{
		Args:   os.Args[1:],
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}))
}
